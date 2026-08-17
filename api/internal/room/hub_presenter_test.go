package room

import (
	"errors"
	"testing"
)

func TestOnlyOneActivePresenter(t *testing.T) {
	h := NewHub()
	if _, err := h.ClaimPresenter("room1"); err != nil {
		t.Fatal(err)
	}
	_, err := h.ClaimPresenter("room1")
	if !errors.Is(err, ErrPresenterConflict) {
		t.Fatalf("got %v", err)
	}
}
