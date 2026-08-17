package room

import (
	"encoding/json"
	"testing"
)

func TestViewerLeaveDecrementsAttachedCount(t *testing.T) {
	h := NewHub()
	pid, err := h.ClaimPresenter("room1")
	if err != nil {
		t.Fatal(err)
	}
	pch := make(chan []byte, 16)
	if !h.Attach("room1", pid, pch) {
		t.Fatal("attach presenter")
	}
	if h.Count("room1") != 1 {
		t.Fatalf("count after presenter attach: %d", h.Count("room1"))
	}

	vid := h.JoinViewer("room1")
	if h.Count("room1") != 1 {
		t.Fatalf("unattached viewer must not count: %d", h.Count("room1"))
	}

	vch := make(chan []byte, 16)
	if !h.Attach("room1", vid, vch) {
		t.Fatal("attach viewer")
	}
	if h.Count("room1") != 2 {
		t.Fatalf("count after viewer attach: %d", h.Count("room1"))
	}

	if h.Disconnect("room1", vid) {
		t.Fatal("viewer leave must not be presenter left")
	}
	if h.Count("room1") != 1 {
		t.Fatalf("count after viewer leave: %d", h.Count("room1"))
	}

	var lastCount int
	var saw bool
	for len(pch) > 0 {
		var msg map[string]any
		if err := json.Unmarshal(<-pch, &msg); err != nil {
			t.Fatal(err)
		}
		if msg["type"] == "presence" {
			payload := msg["payload"].(map[string]any)
			lastCount = int(payload["participantCount"].(float64))
			saw = true
		}
	}
	if !saw || lastCount != 1 {
		t.Fatalf("presence after leave: saw=%v count=%d", saw, lastCount)
	}
}

func TestExpireUnattachedDropsHTTPOnlySessions(t *testing.T) {
	h := NewHub()
	if _, err := h.ClaimPresenter("room1"); err != nil {
		t.Fatal(err)
	}
	_ = h.JoinViewer("room1")
	removed := h.ExpireUnattached(0)
	if removed != 2 {
		t.Fatalf("expected 2 expired, got %d", removed)
	}
	if h.Count("room1") != 0 {
		t.Fatalf("count after expire: %d", h.Count("room1"))
	}
	if h.PresenterID("room1") != "" {
		t.Fatal("expired presenter must release the slot")
	}
	if _, err := h.ClaimPresenter("room1"); err != nil {
		t.Fatalf("claim after expire: %v", err)
	}
}
