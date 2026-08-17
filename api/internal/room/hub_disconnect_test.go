package room

import (
	"encoding/json"
	"testing"
)

func TestPresenterDisconnectBroadcastsWaiting(t *testing.T) {
	h := NewHub()
	pid, err := h.ClaimPresenter("room1")
	if err != nil {
		t.Fatal(err)
	}
	vid := h.JoinViewer("room1")
	ch := make(chan []byte, 8)
	if !h.Attach("room1", vid, ch) {
		t.Fatal("attach")
	}
	if err := h.StartShare("room1", pid); err != nil {
		t.Fatal(err)
	}
	// drain
	for len(ch) > 0 {
		<-ch
	}
	if !h.Disconnect("room1", pid) {
		t.Fatal("expected presenter left")
	}
	var sawWaiting bool
	for len(ch) > 0 {
		var msg map[string]any
		if err := json.Unmarshal(<-ch, &msg); err != nil {
			t.Fatal(err)
		}
		if msg["type"] == "room.state" {
			payload := msg["payload"].(map[string]any)
			if payload["state"] == "waiting" {
				sawWaiting = true
			}
		}
	}
	if !sawWaiting {
		t.Fatal("missing waiting broadcast")
	}
}
