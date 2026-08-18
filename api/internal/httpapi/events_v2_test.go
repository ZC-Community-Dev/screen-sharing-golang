package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestV2EventsRejectParticipantSignalingAndV1IsRetired(t *testing.T) {
	srv := testServer(t)
	link := createLink(t, srv)
	join := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/viewer-sessions", nil)
	var session sessionResponse
	if err := json.Unmarshal(join.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}

	httpServer := httptest.NewServer(srv.Engine)
	defer httpServer.Close()
	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/v2/links/" + link.ID + "/events?sessionId=" + session.SessionID
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for range 2 {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		var event struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &event); err != nil {
			t.Fatal(err)
		}
		if event.Type != "room.state" && event.Type != "presence" && event.Type != "media.state" {
			t.Fatalf("unexpected server event %q", event.Type)
		}
	}
	if err := conn.WriteJSON(map[string]any{"type": "signal", "to": "presenter", "payload": map[string]string{"kind": "offer"}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("participant signaling frame was accepted")
	}

	v1 := doJSON(t, srv, http.MethodGet, "/api/v1/links/"+link.ID+"/events?sessionId="+session.SessionID, nil)
	if v1.Code != http.StatusGone {
		t.Fatalf("v1 status=%d body=%s", v1.Code, v1.Body.String())
	}
}
