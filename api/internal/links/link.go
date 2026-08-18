package links

import "time"

const (
	StateWaiting      = "waiting"
	StateConnecting   = "connecting"
	StateSharing      = "sharing"
	StateReconnecting = "reconnecting"
	StateFailed       = "failed"
)

type Link struct {
	ID                 string
	PresenterTokenHash string
	CreatedAt          time.Time
	State              string
}

type Repository interface {
	Insert(link Link) error
	GetByID(id string) (Link, error)
	SetState(id, state string) error
	Exists(id string) (bool, error)
}
