package media

import "errors"

type Transport string

const (
	TransportWebRTC    Transport = "webrtc"
	TransportWebSocket Transport = "websocket"
)

func (t Transport) Valid() bool {
	return t == TransportWebRTC || t == TransportWebSocket
}

type PublicationState string

const (
	PublicationConnecting   PublicationState = "connecting"
	PublicationLive         PublicationState = "live"
	PublicationReconnecting PublicationState = "reconnecting"
	PublicationFailed       PublicationState = "failed"
	PublicationEnded        PublicationState = "ended"
)

type Publication struct {
	ID             string
	LinkID         string
	OwnerSessionID string
	Transport      Transport
	State          PublicationState
	Generation     uint64
}

var (
	ErrTransportInvalid  = errors.New("invalid media transport")
	ErrTransportMismatch = errors.New("publication transport mismatch")
)
