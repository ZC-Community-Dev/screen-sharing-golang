package media

import (
	"errors"
	"testing"
)

func TestManagerReservesOnePublisherAndViewerCapacity(t *testing.T) {
	m := NewManager(nil, Limits{MaxRooms: 1, MaxViewersPerRoom: 10})
	id, err := m.reservePublisher("link", "presenter")
	if err != nil || id == "" || id == "presenter" {
		t.Fatalf("reserve publisher: %q %v", id, err)
	}
	if _, err := m.reservePublisher("link", "other"); !errors.Is(err, ErrPublisherConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if _, err := m.reservePublisher("second-link", "other"); !errors.Is(err, ErrCapacity) {
		t.Fatalf("expected room capacity, got %v", err)
	}
	for range 10 {
		if _, err := m.reserveSubscriber("link", "viewer"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := m.reserveSubscriber("link", "viewer"); !errors.Is(err, ErrCapacity) {
		t.Fatalf("expected capacity, got %v", err)
	}
	m.ClosePublisher("link", id)
	m.ClosePublisher("link", id)
	if got := m.Stats(); got.ActiveRooms != 0 || got.ActiveSubscribers != 0 {
		t.Fatalf("sessions retained: %+v", got)
	}
}
