package media

import (
	"strings"
	"testing"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

func TestRTPFlowsPublisherThroughServerToSubscriber(t *testing.T) {
	engine, err := NewEngine(EngineConfig{UDPPort: 0})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	manager := NewManager(engine, Limits{MaxRooms: 1, MaxViewersPerRoom: 10})

	publisher, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = publisher.Close() })
	source, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000},
		"screen", "publisher",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := publisher.AddTrack(source); err != nil {
		t.Fatal(err)
	}
	pubOffer := completeOffer(t, publisher)
	_, pubAnswer, err := manager.CreatePublisher("link", "presenter", pubOffer)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pubAnswer.SDP, "a=ice-lite") || !strings.Contains(pubAnswer.SDP, "VP8/90000") ||
		strings.Contains(pubAnswer.SDP, "H264/") || strings.Contains(pubAnswer.SDP, "VP9/") {
		t.Fatal("server answer is not ICE Lite and VP8-only")
	}
	if err := publisher.SetRemoteDescription(pubAnswer); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		_ = source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 96, SequenceNumber: 1, Timestamp: 3000, SSRC: 1234}, Payload: []byte{0x10, 0x00}})
		if _, err := manager.relayFor("link"); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("publisher relay was not created")
		}
		time.Sleep(20 * time.Millisecond)
	}

	received := make(chan int, 1024)
	subscribers := make([]*webrtc.PeerConnection, 0, 10)
	for index := range 10 {
		subscriber, err := webrtc.NewPeerConnection(webrtc.Configuration{})
		if err != nil {
			t.Fatal(err)
		}
		subscribers = append(subscribers, subscriber)
		t.Cleanup(func() { _ = subscriber.Close() })
		if _, err := subscriber.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			t.Fatal(err)
		}
		viewerIndex := index
		subscriber.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
			if track.Codec().MimeType != webrtc.MimeTypeVP8 {
				return
			}
			for {
				if _, _, err := track.ReadRTP(); err != nil {
					return
				}
				select {
				case received <- viewerIndex:
				default:
				}
			}
		})
		subOffer := completeOffer(t, subscriber)
		_, subAnswer, err := manager.CreateSubscriber("link", "viewer-"+string(rune('A'+index)), subOffer)
		if err != nil {
			t.Fatal(err)
		}
		if err := subscriber.SetRemoteDescription(subAnswer); err != nil {
			t.Fatal(err)
		}
	}
	if manager.Stats().ActiveRooms != 1 || manager.Stats().ActiveSubscribers != 10 {
		t.Fatalf("unexpected fan-out stats: %+v", manager.Stats())
	}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(10 * time.Second)
	var sequence uint16 = 2
	seen := make(map[int]bool, 10)
	for len(seen) < 10 {
		select {
		case viewer := <-received:
			seen[viewer] = true
		case <-ticker.C:
			_ = source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 96, SequenceNumber: sequence, Timestamp: uint32(sequence) * 3000, SSRC: 1234}, Payload: []byte{0x10, byte(sequence)}})
			sequence++
		case <-timeout:
			t.Fatalf("only %d subscribers received relayed RTP", len(seen))
		}
	}

	if err := subscribers[0].Close(); err != nil {
		t.Fatal(err)
	}
	for len(received) > 0 {
		<-received
	}
	remaining := make(map[int]bool, 9)
	timeout = time.After(10 * time.Second)
	for len(remaining) < 9 {
		select {
		case viewer := <-received:
			if viewer != 0 {
				remaining[viewer] = true
			}
		case <-ticker.C:
			_ = source.WriteRTP(&rtp.Packet{Header: rtp.Header{Version: 2, PayloadType: 96, SequenceNumber: sequence, Timestamp: uint32(sequence) * 3000, SSRC: 1234}, Payload: []byte{0x10, byte(sequence)}})
			sequence++
		case <-timeout:
			t.Fatalf("only %d of 9 remaining subscribers continued receiving RTP", len(remaining))
		}
	}
}

func completeOffer(t *testing.T, pc *webrtc.PeerConnection) webrtc.SessionDescription {
	t.Helper()
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	done := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(offer); err != nil {
		t.Fatal(err)
	}
	<-done
	return *pc.LocalDescription()
}
