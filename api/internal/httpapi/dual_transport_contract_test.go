package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"api/internal/media"

	"github.com/gorilla/websocket"
)

func TestPublicMediaConfigContainsOnlySafeFields(t *testing.T) {
	srv := testServer(t)
	rec := doJSON(t, srv, http.MethodGet, "/api/v2/media/config", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, forbidden := range []string{"udpPort", "publicIP", "maxRooms", "maxViewers", "maxBufferBytes"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("private field %q exposed in %s", forbidden, body)
		}
	}
	var out publicMediaConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.AllowedTransports) == 0 || out.DefaultTransport == "" ||
		out.WebSocket.MIMEType != "video/webm;codecs=vp8" ||
		out.WebSocket.StartupBufferMS > 2000 {
		t.Fatalf("config=%+v", out)
	}
}

func TestWebSocketTicketRequiresAuthorizedBoundRole(t *testing.T) {
	srv := testServer(t)
	link := createLink(t, srv)
	claim := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/presenter-sessions", map[string]string{
		"presenterToken": link.PresenterToken,
	})
	var presenter sessionResponse
	if err := json.Unmarshal(claim.Body.Bytes(), &presenter); err != nil {
		t.Fatal(err)
	}

	unauthorized := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/media/websocket-tickets", map[string]any{
		"sessionId": presenter.SessionID, "role": "publisher", "presenterToken": "wrong",
	})
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized=%d %s", unauthorized.Code, unauthorized.Body.String())
	}

	issued := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/media/websocket-tickets", map[string]any{
		"sessionId": presenter.SessionID, "role": "publisher", "presenterToken": link.PresenterToken,
	})
	if issued.Code != http.StatusCreated {
		t.Fatalf("issued=%d %s", issued.Code, issued.Body.String())
	}
	var ticket webSocketTicketResponse
	if err := json.Unmarshal(issued.Body.Bytes(), &ticket); err != nil {
		t.Fatal(err)
	}
	if ticket.Ticket == "" || strings.Contains(ticket.WebSocketPath, ticket.Ticket) {
		t.Fatalf("unsafe ticket response: %+v", ticket)
	}
}

func TestWebSocketPublisherToLateViewerBootstrap(t *testing.T) {
	srv := testServer(t)
	link := createLink(t, srv)
	claim := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/presenter-sessions", map[string]string{
		"presenterToken": link.PresenterToken,
	})
	var presenter sessionResponse
	if err := json.Unmarshal(claim.Body.Bytes(), &presenter); err != nil {
		t.Fatal(err)
	}
	publisherTicket := issueMediaTicket(t, srv, link.ID, map[string]any{
		"sessionId": presenter.SessionID, "role": "publisher", "presenterToken": link.PresenterToken,
	})

	httpServer := httptest.NewServer(srv.Engine)
	defer httpServer.Close()
	publisher := dialMediaSocket(t, httpServer.URL, link.ID, publisherTicket.Ticket)
	defer publisher.Close()
	if err := publisher.WriteJSON(map[string]any{
		"type": "publisher.open", "protocolVersion": 1,
		"mimeType": webSocketMIMEType, "timesliceMs": 250,
	}); err != nil {
		t.Fatal(err)
	}
	var opened map[string]any
	if err := publisher.ReadJSON(&opened); err != nil || opened["type"] != "media.opened" {
		t.Fatalf("publisher opened=%v err=%v", opened, err)
	}
	if err := publisher.WriteMessage(websocket.BinaryMessage, deterministicHTTPWebMFixture()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		publication, ok := srv.Media.Publication(link.ID)
		if ok && publication.State == media.PublicationLive {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("publication did not become live")
		}
		time.Sleep(5 * time.Millisecond)
	}

	join := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/viewer-sessions", nil)
	var viewerSession sessionResponse
	if err := json.Unmarshal(join.Body.Bytes(), &viewerSession); err != nil {
		t.Fatal(err)
	}
	viewerTicket := issueMediaTicket(t, srv, link.ID, map[string]any{
		"sessionId": viewerSession.SessionID, "role": "viewer",
	})
	viewer := dialMediaSocket(t, httpServer.URL, link.ID, viewerTicket.Ticket)
	defer viewer.Close()
	if err := viewer.WriteJSON(map[string]any{"type": "subscriber.open", "protocolVersion": 1}); err != nil {
		t.Fatal(err)
	}
	_ = viewer.SetReadDeadline(time.Now().Add(2 * time.Second))
	for index, expectedType := range []int{
		websocket.TextMessage, websocket.TextMessage, websocket.BinaryMessage,
		websocket.BinaryMessage, websocket.TextMessage,
	} {
		messageType, payload, err := viewer.ReadMessage()
		if err != nil {
			t.Fatalf("message %d: %v", index, err)
		}
		if messageType != expectedType {
			t.Fatalf("message %d type=%d payload=%q", index, messageType, payload)
		}
	}
}

func TestMediaUpgradeEnforcesOriginProtocolAndSingleUse(t *testing.T) {
	srv := testServer(t)
	link := createLink(t, srv)
	claim := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/presenter-sessions", map[string]string{
		"presenterToken": link.PresenterToken,
	})
	var presenter sessionResponse
	if err := json.Unmarshal(claim.Body.Bytes(), &presenter); err != nil {
		t.Fatal(err)
	}
	ticket := issueMediaTicket(t, srv, link.ID, map[string]any{
		"sessionId": presenter.SessionID, "role": "publisher", "presenterToken": link.PresenterToken,
	})
	path := "/api/v2/links/" + link.ID + "/media/websocket?ticket=" + url.QueryEscape(ticket.Ticket)
	missingProtocol := doJSON(t, srv, http.MethodGet, path, nil)
	if missingProtocol.Code != http.StatusUpgradeRequired {
		t.Fatalf("missing protocol=%d %s", missingProtocol.Code, missingProtocol.Body.String())
	}

	httpServer := httptest.NewServer(srv.Engine)
	defer httpServer.Close()
	dialer := websocket.Dialer{Subprotocols: []string{mediaSubprotocol}, EnableCompression: false}
	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + path
	_, response, err := dialer.Dial(wsURL, http.Header{"Origin": []string{"https://attacker.invalid"}})
	if err == nil || response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong origin response=%v err=%v", response, err)
	}

	conn := dialMediaSocket(t, httpServer.URL, link.ID, ticket.Ticket)
	defer conn.Close()
	_, response, err = dialer.Dial(wsURL, http.Header{"Origin": []string{httpServer.URL}})
	if err == nil || response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("ticket replay response=%v err=%v", response, err)
	}
}

func issueMediaTicket(t *testing.T, srv *Server, linkID string, body any) webSocketTicketResponse {
	t.Helper()
	response := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+linkID+"/media/websocket-tickets", body)
	if response.Code != http.StatusCreated {
		t.Fatalf("ticket=%d %s", response.Code, response.Body.String())
	}
	var ticket webSocketTicketResponse
	if err := json.Unmarshal(response.Body.Bytes(), &ticket); err != nil {
		t.Fatal(err)
	}
	return ticket
}

func dialMediaSocket(t *testing.T, serverURL, linkID, ticket string) *websocket.Conn {
	t.Helper()
	dialer := websocket.Dialer{Subprotocols: []string{mediaSubprotocol}, EnableCompression: false}
	header := http.Header{"Origin": []string{serverURL}}
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") +
		"/api/v2/links/" + linkID + "/media/websocket?ticket=" + url.QueryEscape(ticket)
	conn, response, err := dialer.Dial(wsURL, header)
	if err != nil {
		if response != nil {
			t.Fatalf("dial status=%d err=%v", response.StatusCode, err)
		}
		t.Fatal(err)
	}
	if conn.Subprotocol() != mediaSubprotocol {
		t.Fatalf("subprotocol=%q", conn.Subprotocol())
	}
	return conn
}

func deterministicHTTPWebMFixture() []byte {
	return []byte{
		0x1a, 0x45, 0xdf, 0xa3, 0x80,
		0x18, 0x53, 0x80, 0x67, 0xaf,
		0x15, 0x49, 0xa9, 0x66, 0x87, 0x2a, 0xd7, 0xb1, 0x83, 0x0f, 0x42, 0x40,
		0x16, 0x54, 0xae, 0x6b, 0x8f,
		0xae, 0x8d, 0xd7, 0x81, 0x01, 0x83, 0x81, 0x01, 0x86, 0x85, 'V', '_', 'V', 'P', '8',
		0x1f, 0x43, 0xb6, 0x75, 0x8a,
		0xe7, 0x81, 0x00,
		0xa3, 0x85, 0x81, 0x00, 0x00, 0x80, 0x00,
	}
}
