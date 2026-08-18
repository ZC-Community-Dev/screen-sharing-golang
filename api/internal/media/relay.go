package media

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
)

const (
	pliInterval    = 3 * time.Second
	rtpIdleTimeout = 15 * time.Second
)

// Relay forwards compressed RTP directly. It deliberately retains no packets.
type Relay struct {
	Track *webrtc.TrackLocalStaticRTP

	remote    *webrtc.TrackRemote
	publisher *webrtc.PeerConnection
	stop      chan struct{}
	once      sync.Once
	lastRTP   atomic.Int64
	onStopped func()
}

func NewRelay(remote *webrtc.TrackRemote, publisher *webrtc.PeerConnection, onStopped func()) (*Relay, error) {
	if remote == nil || remote.Kind() != webrtc.RTPCodecTypeVideo || remote.Codec().MimeType != webrtc.MimeTypeVP8 {
		return nil, ErrInvalidSDP
	}
	track, err := webrtc.NewTrackLocalStaticRTP(remote.Codec().RTPCodecCapability, "screen", "relay")
	if err != nil {
		return nil, err
	}
	r := &Relay{Track: track, remote: remote, publisher: publisher, stop: make(chan struct{}), onStopped: onStopped}
	r.lastRTP.Store(time.Now().UnixNano())
	return r, nil
}

func (r *Relay) Start() {
	go r.forward()
	go r.feedback()
}

func (r *Relay) forward() {
	for {
		packet, _, err := r.remote.ReadRTP()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				r.Close()
			}
			return
		}
		r.lastRTP.Store(time.Now().UnixNano())
		if err := r.Track.WriteRTP(packet); err != nil && !errors.Is(err, io.ErrClosedPipe) {
			r.Close()
			return
		}
	}
}

func (r *Relay) feedback() {
	ticker := time.NewTicker(pliInterval)
	defer ticker.Stop()
	for {
		select {
		case <-r.stop:
			return
		case <-ticker.C:
			if time.Since(time.Unix(0, r.lastRTP.Load())) > rtpIdleTimeout {
				r.Close()
				return
			}
			if r.publisher != nil {
				_ = r.publisher.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(r.remote.SSRC())}})
			}
		}
	}
}

func (r *Relay) Close() {
	if r == nil {
		return
	}
	r.once.Do(func() {
		close(r.stop)
		if r.onStopped != nil {
			go r.onStopped()
		}
	})
}
