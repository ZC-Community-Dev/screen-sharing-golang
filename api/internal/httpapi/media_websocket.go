package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"api/internal/links"
	"api/internal/media"
	"api/internal/room"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	mediaSubprotocol              = "screen-share-media-v1"
	mediaReadTimeout              = 15 * time.Second
	mediaWriteTimeout             = 5 * time.Second
	mediaPingInterval             = 5 * time.Second
	maxPublisherMessagesPerSecond = 100
)

type webSocketTicketRequest struct {
	SessionID      string           `json:"sessionId"`
	Role           media.TicketRole `json:"role"`
	PresenterToken string           `json:"presenterToken,omitempty"`
}

type webSocketTicketResponse struct {
	Ticket        string    `json:"ticket"`
	ExpiresAt     time.Time `json:"expiresAt"`
	WebSocketPath string    `json:"websocketPath"`
}

type mediaOpen struct {
	Type            string `json:"type"`
	ProtocolVersion int    `json:"protocolVersion"`
	MIMEType        string `json:"mimeType,omitempty"`
	TimesliceMS     int    `json:"timesliceMs,omitempty"`
}

type ticketReservation struct {
	cancel    chan struct{}
	linkID    string
	sessionID string
	cleanup   func()
}

func (s *Server) createMediaWebSocketTicket(c *gin.Context) {
	linkID, ok := s.parseID(c)
	if !ok {
		return
	}
	if _, err := s.Links.GetByID(linkID); err != nil {
		if errors.Is(err, links.ErrNotFound) {
			writeError(c, http.StatusNotFound, CodeLinkNotFound, "link not found")
		} else {
			writeError(c, http.StatusInternalServerError, CodeInternalError, "could not load link")
		}
		return
	}
	if !containsTransport(s.Config.MediaAllowedTransports, media.TransportWebSocket) {
		writeError(c, http.StatusForbidden, CodeTransportNotAllowed, "websocket media transport is not enabled")
		return
	}
	var request webSocketTicketRequest
	if !decodeMediaJSON(c, &request) ||
		(request.Role != media.RolePublisher && request.Role != media.RoleViewer) {
		writeError(c, http.StatusBadRequest, CodeTransportInvalid, "invalid media ticket request")
		return
	}
	sessionRole, exists := s.Hub.SessionRole(linkID, request.SessionID)
	if !exists {
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "room session is invalid")
		return
	}

	claims := media.TicketClaims{
		LinkID: linkID, SessionID: request.SessionID, Role: request.Role,
		Transport: media.TransportWebSocket,
	}
	var cleanup func()
	switch request.Role {
	case media.RolePublisher:
		if sessionRole != room.RolePresenter || s.Links.VerifyToken(linkID, request.PresenterToken) != nil {
			writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "presenter authorization failed")
			return
		}
		publicationID, generation, err := s.Media.ReservePublication(linkID, request.SessionID, media.TransportWebSocket)
		if err != nil {
			s.writeMediaError(c, err)
			return
		}
		claims.PublicationID = publicationID
		claims.MediaSessionID = publicationID
		claims.Generation = generation
		cleanup = func() { s.Media.ClosePublisher(linkID, publicationID) }
	case media.RoleViewer:
		if sessionRole != room.RoleViewer {
			writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "viewer session is invalid")
			return
		}
		publication, exists := s.Media.Publication(linkID)
		if !exists {
			writeError(c, http.StatusConflict, CodeMediaNotReady, "publisher media is not ready")
			return
		}
		if publication.Transport != media.TransportWebSocket {
			writeError(c, http.StatusConflict, CodeTransportMismatch, "publication uses another transport")
			return
		}
		mediaID, current, err := s.Media.ReserveWebSocketSubscriber(linkID, request.SessionID, publication.Generation)
		if err != nil {
			s.writeMediaError(c, err)
			return
		}
		claims.PublicationID = current.ID
		claims.MediaSessionID = mediaID
		claims.Generation = current.Generation
		cleanup = func() { _ = s.Media.CloseSubscriber(linkID, mediaID, request.SessionID) }
	}

	raw, ticket, err := s.Tickets.Issue(claims)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		writeError(c, http.StatusInternalServerError, CodeInternalError, "could not issue media ticket")
		return
	}
	s.trackTicketReservation(raw, ticket, cleanup)
	c.JSON(http.StatusCreated, webSocketTicketResponse{
		Ticket: raw, ExpiresAt: ticket.ExpiresAt,
		WebSocketPath: "/api/v2/links/" + linkID + "/media/websocket",
	})
}

func (s *Server) mediaWebSocket(c *gin.Context) {
	linkID, ok := s.parseID(c)
	if !ok {
		return
	}
	if !hasSubprotocol(c.Request, mediaSubprotocol) {
		writeError(c, http.StatusUpgradeRequired, CodeWebSocketUpgrade, "media websocket subprotocol is required")
		return
	}
	if !s.validMediaOrigin(c.Request) {
		writeError(c, http.StatusForbidden, CodePresenterUnauthorized, "media websocket origin is not allowed")
		return
	}
	raw := c.Query("ticket")
	ticket, err := s.Tickets.ConsumeForLink(raw, linkID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "media ticket is invalid")
		return
	}
	s.cancelTicketReservation(raw)
	if role, exists := s.Hub.SessionRole(linkID, ticket.SessionID); !exists ||
		(ticket.Role == media.RolePublisher && role != room.RolePresenter) ||
		(ticket.Role == media.RoleViewer && role != room.RoleViewer) {
		s.cleanupTicketClaim(ticket)
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "room session is invalid")
		return
	}
	upgrader := websocket.Upgrader{
		Subprotocols:      []string{mediaSubprotocol},
		EnableCompression: false,
		CheckOrigin:       func(*http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.cleanupTicketClaim(ticket)
		return
	}
	conn.EnableWriteCompression(false)
	conn.SetReadLimit(int64(s.Config.MediaWSMaxChunkBytes))
	if ticket.Role == media.RolePublisher {
		s.runWebSocketPublisher(conn, ticket)
	} else {
		s.runWebSocketViewer(conn, ticket)
	}
}

func (s *Server) runWebSocketPublisher(conn *websocket.Conn, ticket media.Ticket) {
	defer conn.Close()
	defer s.Media.ClosePublisher(ticket.LinkID, ticket.PublicationID)
	endedCleanly := false
	defer func() {
		if !endedCleanly {
			s.Media.FailPublication(ticket.LinkID, ticket.PublicationID)
		}
	}()
	if err := setPublisherReadDeadline(conn); err != nil {
		return
	}
	var open mediaOpen
	if err := readMediaOpen(conn, &open); err != nil || open.Type != "publisher.open" ||
		open.ProtocolVersion != 1 || open.MIMEType != webSocketMIMEType ||
		open.TimesliceMS < 100 || open.TimesliceMS > 2000 {
		writeMediaClose(conn, 4400, CodeMediaProtocolError)
		return
	}
	ring := media.NewClusterRing(2*time.Second, s.Config.MediaWSMaxBufferBytes, 16)
	if err := s.Media.AttachWebSocketPublisher(ticket.LinkID, ticket.PublicationID, ring, func() { _ = conn.Close() }); err != nil {
		writeMediaClose(conn, 4409, CodeTransportMismatch)
		return
	}
	if err := writeJSONDeadline(conn, map[string]any{
		"type": "media.opened", "publicationId": ticket.PublicationID,
		"mediaSessionId": ticket.MediaSessionID, "transport": media.TransportWebSocket,
		"startupBufferMs": webSocketStartupBufferMS, "maxChunkBytes": s.Config.MediaWSMaxChunkBytes,
	}); err != nil {
		return
	}
	parser := media.NewWebMParser(s.Config.MediaWSMaxChunkBytes * 2)
	defer parser.Close()
	generation := ticket.Generation
	initSet := false
	live := false
	windowStart := time.Now()
	messageCount := 0
	pingDone := make(chan struct{})
	defer close(pingDone)
	go mediaPingPump(conn, pingDone)

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if err := conn.SetReadDeadline(time.Now().Add(mediaReadTimeout)); err != nil {
			return
		}
		now := time.Now()
		if now.Sub(windowStart) >= time.Second {
			windowStart, messageCount = now, 0
		}
		messageCount++
		if messageCount > maxPublisherMessagesPerSecond {
			writeMediaClose(conn, 4400, CodeMediaProtocolError)
			return
		}
		switch messageType {
		case websocket.BinaryMessage:
			records, err := parser.Push(payload)
			if err != nil {
				writeMediaClose(conn, 4400, protocolCode(err))
				return
			}
			if len(records) > 0 && !initSet {
				if !ring.SetInit(parser.Init(), generation) {
					writeMediaClose(conn, websocket.CloseMessageTooBig, CodeMediaProtocolError)
					return
				}
				initSet = true
			}
			for _, record := range records {
				if !ring.Append(media.Cluster{
					Data: record.Data, Timestamp: record.Timestamp, RandomAccess: record.RandomAccess,
				}) {
					writeMediaClose(conn, websocket.CloseMessageTooBig, CodeMediaProtocolError)
					return
				}
				if !live {
					live = s.Media.MarkWebSocketLive(ticket.LinkID, ticket.PublicationID)
				}
			}
		case websocket.TextMessage:
			var control struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(payload, &control) != nil {
				writeMediaClose(conn, 4400, CodeMediaProtocolError)
				return
			}
			switch control.Type {
			case "media.end":
				endedCleanly = true
				_ = writeJSONDeadline(conn, map[string]any{"type": "media.end"})
				return
			case "media.reset":
				next, ok := s.Media.ResetWebSocketPublication(ticket.LinkID, ticket.PublicationID)
				if !ok {
					return
				}
				generation, initSet, live = next, false, false
				parser.Reset()
				_ = writeJSONDeadline(conn, map[string]any{"type": "media.reset", "generation": generation})
			default:
				writeMediaClose(conn, 4400, CodeMediaProtocolError)
				return
			}
		default:
			writeMediaClose(conn, 4400, CodeMediaProtocolError)
			return
		}
	}
}

func (s *Server) runWebSocketViewer(conn *websocket.Conn, ticket media.Ticket) {
	defer conn.Close()
	defer func() { _ = s.Media.CloseSubscriber(ticket.LinkID, ticket.MediaSessionID, ticket.SessionID) }()
	if err := setMediaReadDeadline(conn); err != nil {
		return
	}
	var open mediaOpen
	if err := readMediaOpen(conn, &open); err != nil || open.Type != "subscriber.open" || open.ProtocolVersion != 1 {
		writeMediaClose(conn, 4400, CodeMediaProtocolError)
		return
	}
	ring, publication, err := s.Media.WebSocketBuffer(ticket.LinkID, ticket.Generation)
	if err != nil || publication.ID != ticket.PublicationID {
		writeMediaClose(conn, 4409, CodeTransportMismatch)
		return
	}
	snapshot, queue, err := ring.Subscribe(ticket.MediaSessionID)
	if err != nil {
		writeMediaClose(conn, 4409, CodeMediaStartTimeout)
		return
	}
	defer ring.Unsubscribe(ticket.MediaSessionID)
	if err := s.Media.AttachWebSocketSubscriber(ticket.LinkID, ticket.MediaSessionID, ticket.SessionID, func() { _ = conn.Close() }); err != nil {
		return
	}
	if err := writeJSONDeadline(conn, map[string]any{
		"type": "media.opened", "publicationId": publication.ID,
		"mediaSessionId": ticket.MediaSessionID, "transport": media.TransportWebSocket,
	}); err != nil {
		return
	}
	if err := writeJSONDeadline(conn, map[string]any{
		"type": "media.bootstrap", "generation": snapshot.Generation, "clusterCount": len(snapshot.Clusters),
	}); err != nil {
		return
	}
	if err := writeBinaryDeadline(conn, snapshot.Init); err != nil {
		return
	}
	for _, cluster := range snapshot.Clusters {
		if err := writeBinaryDeadline(conn, cluster.Data); err != nil {
			return
		}
	}
	if err := writeJSONDeadline(conn, map[string]any{"type": "media.live", "generation": snapshot.Generation}); err != nil {
		return
	}
	s.Media.EmitConnected(ticket.LinkID, ticket.SessionID, ticket.MediaSessionID, "viewer")

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
			writeMediaClose(conn, 4400, CodeMediaProtocolError)
			return
		}
	}()
	ping := time.NewTicker(mediaPingInterval)
	defer ping.Stop()
	for {
		select {
		case <-readDone:
			return
		case cluster, ok := <-queue.C:
			if !ok {
				if errors.Is(queue.Err(), media.ErrSlowConsumer) {
					writeMediaClose(conn, 4429, CodeMediaSlowConsumer)
				}
				return
			}
			if queue.Closed() {
				if errors.Is(queue.Err(), media.ErrSlowConsumer) {
					writeMediaClose(conn, 4429, CodeMediaSlowConsumer)
				}
				return
			}
			if err := writeBinaryDeadline(conn, cluster.Data); err != nil {
				return
			}
			queue.Consumed(cluster)
		case <-ping.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(mediaWriteTimeout)); err != nil {
				return
			}
		}
	}
}

func readMediaOpen(conn *websocket.Conn, open *mediaOpen) error {
	messageType, payload, err := conn.ReadMessage()
	if err != nil {
		return err
	}
	if messageType != websocket.TextMessage || len(payload) > 4096 {
		return errors.New("invalid media open")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(open); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("invalid media open")
	}
	return nil
}

func (s *Server) trackTicketReservation(raw string, ticket media.Ticket, cleanup func()) {
	key := ticketKey(raw)
	reservation := &ticketReservation{
		cancel: make(chan struct{}), linkID: ticket.LinkID,
		sessionID: ticket.SessionID, cleanup: cleanup,
	}
	s.ticketReservations.Store(key, reservation)
	go func() {
		timer := time.NewTimer(time.Until(ticket.ExpiresAt))
		defer timer.Stop()
		select {
		case <-timer.C:
			if _, loaded := s.ticketReservations.LoadAndDelete(key); loaded {
				reservation.cleanup()
			}
		case <-reservation.cancel:
		}
	}()
}

func (s *Server) cancelTicketReservation(raw string) {
	key := ticketKey(raw)
	if value, ok := s.ticketReservations.LoadAndDelete(key); ok {
		close(value.(*ticketReservation).cancel)
	}
}

func (s *Server) cleanupTicketReservations(linkID, sessionID string) {
	s.ticketReservations.Range(func(key, value any) bool {
		reservation := value.(*ticketReservation)
		if reservation.linkID == linkID && (sessionID == "" || reservation.sessionID == sessionID) {
			if _, loaded := s.ticketReservations.LoadAndDelete(key); loaded {
				close(reservation.cancel)
				reservation.cleanup()
			}
		}
		return true
	})
	if sessionID == "" {
		s.Tickets.DeleteForLink(linkID)
	} else {
		s.Tickets.DeleteForSession(linkID, sessionID)
	}
}

func (s *Server) cleanupTicketClaim(ticket media.Ticket) {
	if ticket.Role == media.RolePublisher {
		s.Media.ClosePublisher(ticket.LinkID, ticket.PublicationID)
	} else {
		_ = s.Media.CloseSubscriber(ticket.LinkID, ticket.MediaSessionID, ticket.SessionID)
	}
}

func ticketKey(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func hasSubprotocol(request *http.Request, expected string) bool {
	for _, value := range strings.Split(request.Header.Get("Sec-WebSocket-Protocol"), ",") {
		if strings.TrimSpace(value) == expected {
			return true
		}
	}
	return false
}

func setMediaReadDeadline(conn *websocket.Conn) error {
	if err := conn.SetReadDeadline(time.Now().Add(mediaReadTimeout)); err != nil {
		return err
	}
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(mediaReadTimeout))
	})
	return nil
}

func setPublisherReadDeadline(conn *websocket.Conn) error {
	if err := conn.SetReadDeadline(time.Now().Add(mediaReadTimeout)); err != nil {
		return err
	}
	conn.SetPongHandler(func(string) error { return nil })
	return nil
}

func mediaPingPump(conn *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(mediaPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(mediaWriteTimeout)) != nil {
				return
			}
		}
	}
}

func writeJSONDeadline(conn *websocket.Conn, value any) error {
	if err := conn.SetWriteDeadline(time.Now().Add(mediaWriteTimeout)); err != nil {
		return err
	}
	return conn.WriteJSON(value)
}

func writeBinaryDeadline(conn *websocket.Conn, payload []byte) error {
	if err := conn.SetWriteDeadline(time.Now().Add(mediaWriteTimeout)); err != nil {
		return err
	}
	return conn.WriteMessage(websocket.BinaryMessage, payload)
}

func writeMediaClose(conn *websocket.Conn, code int, reason string) {
	_ = conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(code, reason),
		time.Now().Add(time.Second),
	)
}

func protocolCode(err error) string {
	if errors.Is(err, media.ErrWebMUnsupported) {
		return CodeMediaCodecUnsupported
	}
	return CodeMediaProtocolError
}
