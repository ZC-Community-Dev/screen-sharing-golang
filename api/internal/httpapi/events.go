package httpapi

import (
	"net/http"
	"time"

	"api/internal/ids"
	"api/internal/links"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "" ||
			origin == "http://localhost:4200" ||
			origin == "http://127.0.0.1:4200"
	},
}

func (s *Server) eventsV1(c *gin.Context) {
	id := c.Param("id")
	if !ids.ValidPublicID(id) {
		writeError(c, http.StatusBadRequest, CodeLinkInvalid, "invalid link")
		return
	}
	if _, err := s.Links.GetByID(id); err != nil {
		writeError(c, http.StatusNotFound, CodeLinkNotFound, "link not found")
		return
	}
	writeError(c, http.StatusGone, "api_version_retired", "participant signaling has moved to API v2 media endpoints")
}

func (s *Server) eventsV2(c *gin.Context) {
	id := c.Param("id")
	if !ids.ValidPublicID(id) {
		writeError(c, http.StatusBadRequest, CodeLinkInvalid, "invalid link")
		return
	}
	if _, err := s.Links.GetByID(id); err != nil {
		writeError(c, http.StatusNotFound, CodeLinkNotFound, "link not found")
		return
	}
	sessionID := c.Query("sessionId")
	if !s.Hub.HasSession(id, sessionID) {
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "unknown session")
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	send := make(chan []byte, 64)
	if !s.Hub.Attach(id, sessionID, send) {
		return
	}
	defer func() {
		s.cleanupTicketReservations(id, sessionID)
		s.Media.CloseRoomSession(id, sessionID)
		if s.Hub.Disconnect(id, sessionID) {
			_ = s.Links.SetState(id, links.StateWaiting)
		}
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "client event frames are not accepted"),
				time.Now().Add(time.Second),
			)
			return
		}
	}()

	for {
		select {
		case <-done:
			return
		case msg, ok := <-send:
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}
}
