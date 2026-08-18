package media

import (
	"fmt"
	"sync"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
)

type SubscriberSession struct {
	ID             string
	LinkID         string
	OwnerSessionID string
	Transport      Transport
	Peer           *webrtc.PeerConnection
	Sender         *webrtc.RTPSender
	closeSocket    func()
	closeOnce      sync.Once
}

func (m *Manager) CreateSubscriber(linkID, owner string, offer webrtc.SessionDescription) (string, webrtc.SessionDescription, error) {
	relay, err := m.relayFor(linkID)
	if err != nil {
		return "", webrtc.SessionDescription{}, err
	}
	id, err := m.reserveSubscriber(linkID, owner)
	if err != nil {
		return "", webrtc.SessionDescription{}, err
	}
	api, err := m.engineAPI()
	if err != nil {
		_ = m.CloseSubscriber(linkID, id, owner)
		return "", webrtc.SessionDescription{}, err
	}
	pc, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		_ = m.CloseSubscriber(linkID, id, owner)
		return "", webrtc.SessionDescription{}, fmt.Errorf("create subscriber peer: %w", err)
	}
	sender, err := pc.AddTrack(relay.Track)
	if err != nil {
		_ = pc.Close()
		_ = m.CloseSubscriber(linkID, id, owner)
		return "", webrtc.SessionDescription{}, ErrInvalidSDP
	}
	session := &SubscriberSession{
		ID: id, LinkID: linkID, OwnerSessionID: owner,
		Transport: TransportWebRTC, Peer: pc, Sender: sender,
	}
	if !m.installSubscriber(linkID, id, session) {
		session.closePeer()
		return "", webrtc.SessionDescription{}, ErrSessionNotFound
	}
	pc.OnTrack(func(_ *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		_ = m.CloseSubscriber(linkID, id, owner)
	})
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateConnected:
			m.emitMedia(linkID, owner, id, "viewer", "connected")
			relay.RequestKeyframe()
		case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateClosed:
			_ = m.CloseSubscriber(linkID, id, owner)
		}
	})
	go drainRTCP(sender, relay)
	if err := pc.SetRemoteDescription(offer); err != nil {
		_ = m.CloseSubscriber(linkID, id, owner)
		return "", webrtc.SessionDescription{}, ErrInvalidSDP
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		_ = m.CloseSubscriber(linkID, id, owner)
		return "", webrtc.SessionDescription{}, ErrInvalidSDP
	}
	gathering := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(answer); err != nil {
		_ = m.CloseSubscriber(linkID, id, owner)
		return "", webrtc.SessionDescription{}, ErrInvalidSDP
	}
	<-gathering
	m.emitMedia(linkID, owner, id, "viewer", "connecting")
	return id, *pc.LocalDescription(), nil
}

func drainRTCP(sender *webrtc.RTPSender, relay *Relay) {
	for {
		packets, _, err := sender.ReadRTCP()
		if err != nil {
			return
		}
		for _, packet := range packets {
			switch packet.(type) {
			case *rtcp.PictureLossIndication, *rtcp.FullIntraRequest:
				relay.RequestKeyframe()
			}
		}
	}
}

func (s *SubscriberSession) closePeer() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		if s.Peer != nil {
			_ = s.Peer.Close()
		}
		if s.closeSocket != nil {
			s.closeSocket()
		}
	})
}
