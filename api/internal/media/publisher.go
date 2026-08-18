package media

import (
	"fmt"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

// Keep this below the client's final bounded retry (3.5s cumulative) so the
// same presenter can establish a fresh publisher after a broken ICE session.
const publisherDisconnectGrace = 2 * time.Second

type PublisherSession struct {
	ID             string
	LinkID         string
	OwnerSessionID string
	Transport      Transport
	Peer           *webrtc.PeerConnection
	closeSocket    func()

	closeOnce sync.Once
	timerMu   sync.Mutex
	timer     *time.Timer
}

func (m *Manager) CreatePublisher(linkID, owner string, offer webrtc.SessionDescription) (string, webrtc.SessionDescription, error) {
	id, err := m.reservePublisher(linkID, owner)
	if err != nil {
		return "", webrtc.SessionDescription{}, err
	}
	api, err := m.engineAPI()
	if err != nil {
		m.ClosePublisher(linkID, id)
		return "", webrtc.SessionDescription{}, err
	}
	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		m.ClosePublisher(linkID, id)
		return "", webrtc.SessionDescription{}, fmt.Errorf("create publisher peer: %w", err)
	}
	session := &PublisherSession{
		ID: id, LinkID: linkID, OwnerSessionID: owner,
		Transport: TransportWebRTC, Peer: pc,
	}
	if !m.installPublisher(linkID, id, session) {
		_ = pc.Close()
		return "", webrtc.SessionDescription{}, ErrPublisherConflict
	}

	pc.OnTrack(func(remote *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		if remote.Kind() != webrtc.RTPCodecTypeVideo || remote.Codec().MimeType != webrtc.MimeTypeVP8 {
			m.ClosePublisher(linkID, id)
			return
		}
		relay, relayErr := NewRelay(remote, pc, func() { m.ClosePublisher(linkID, id) })
		if relayErr != nil || !m.setRelay(linkID, id, relay) {
			if relay != nil {
				relay.Close()
			}
			return
		}
		relay.Start()
	})
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateConnected:
			session.cancelTimer()
			m.emitMedia(linkID, owner, id, "presenter", "connected")
		case webrtc.PeerConnectionStateDisconnected:
			m.emitState(linkID, StateReconnecting)
			m.emitMedia(linkID, owner, id, "presenter", "reconnecting")
			session.startTimer(func() {
				m.emitState(linkID, StateFailed)
				m.emitMedia(linkID, owner, id, "presenter", "failed")
				m.ClosePublisher(linkID, id)
			})
		case webrtc.PeerConnectionStateFailed:
			m.emitState(linkID, StateFailed)
			m.emitMedia(linkID, owner, id, "presenter", "failed")
			m.ClosePublisher(linkID, id)
		case webrtc.PeerConnectionStateClosed:
			m.ClosePublisher(linkID, id)
		}
	})
	if err := pc.SetRemoteDescription(offer); err != nil {
		m.ClosePublisher(linkID, id)
		return "", webrtc.SessionDescription{}, ErrInvalidSDP
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		m.ClosePublisher(linkID, id)
		return "", webrtc.SessionDescription{}, ErrInvalidSDP
	}
	gathering := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(answer); err != nil {
		m.ClosePublisher(linkID, id)
		return "", webrtc.SessionDescription{}, ErrInvalidSDP
	}
	<-gathering
	m.emitState(linkID, StateConnecting)
	m.emitMedia(linkID, owner, id, "presenter", "connecting")
	return id, *pc.LocalDescription(), nil
}

func (m *Manager) emitState(linkID string, state State) {
	m.mu.Lock()
	room := m.rooms[linkID]
	if room == nil {
		m.mu.Unlock()
		return
	}
	var publication Publication
	var hasPublication bool
	room.state = state
	switch state {
	case StateConnecting:
		room.publication.State = PublicationConnecting
	case StateSharing:
		room.publication.State = PublicationLive
	case StateReconnecting:
		room.publication.State = PublicationReconnecting
	case StateFailed:
		room.publication.State = PublicationFailed
	}
	publication = room.publication
	hasPublication = room.publisher != nil
	cb, publicationCB := m.onState, m.onPublication
	m.mu.Unlock()
	if hasPublication && publicationCB != nil {
		publicationCB(linkID, publication)
	}
	if cb != nil {
		cb(linkID, state)
	}
}

func (m *Manager) emitMedia(linkID, owner, id, role, state string) {
	m.mu.Lock()
	cb := m.onMedia
	m.mu.Unlock()
	if cb != nil {
		cb(linkID, owner, id, role, state)
	}
}

func (p *PublisherSession) startTimer(fn func()) {
	p.timerMu.Lock()
	defer p.timerMu.Unlock()
	if p.timer != nil {
		p.timer.Stop()
	}
	p.timer = time.AfterFunc(publisherDisconnectGrace, fn)
}

func (p *PublisherSession) cancelTimer() {
	p.timerMu.Lock()
	defer p.timerMu.Unlock()
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}
}

func (p *PublisherSession) closePeer() {
	if p == nil {
		return
	}
	p.closeOnce.Do(func() {
		p.cancelTimer()
		if p.Peer != nil {
			_ = p.Peer.Close()
		}
		if p.closeSocket != nil {
			p.closeSocket()
		}
	})
}
