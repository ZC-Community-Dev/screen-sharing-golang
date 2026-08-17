package room

import (
	"encoding/json"
	"testing"
)

func TestRelaySignalKeepsReadyKind(t *testing.T) {
	raw := []byte(`{"type":"signal","to":"presenter","payload":{"kind":"ready"}}`)
	out, to, ok := RelaySignal(raw, "viewer1")
	if !ok {
		t.Fatal("expected relay")
	}
	if to != "presenter" {
		t.Fatalf("to=%s", to)
	}
	var msg map[string]any
	if err := json.Unmarshal(out, &msg); err != nil {
		t.Fatal(err)
	}
	payload, _ := msg["payload"].(map[string]any)
	if payload["kind"] != "ready" {
		t.Fatalf("kind=%v", payload["kind"])
	}
	if msg["from"] != "viewer1" {
		t.Fatalf("from=%v", msg["from"])
	}
}

func TestResolveSignalToRewritesPresenterAlias(t *testing.T) {
	if got := ResolveSignalTo("presenter", "abc"); got != "abc" {
		t.Fatalf("got %s", got)
	}
	if got := ResolveSignalTo("sess-9", "abc"); got != "sess-9" {
		t.Fatalf("got %s", got)
	}
	if got := ResolveSignalTo("presenter", ""); got != "" {
		t.Fatalf("empty presenter should stay empty, got %s", got)
	}
}
