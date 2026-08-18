package media

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/pion/ice/v4"
	"github.com/pion/interceptor"
	"github.com/pion/logging"
	"github.com/pion/webrtc/v4"
)

type EngineConfig struct {
	UDPPort  int
	PublicIP string
	// MTU is the target UDP datagram size for WebRTC/ICE. Default 1200
	// avoids IP fragmentation on typical internet paths.
	MTU int
}

// Engine owns the process-wide UDP4 mux used by every media peer.
type Engine struct {
	API    *webrtc.API
	MTU    int
	conn   *net.UDPConn
	mux    ice.UDPMux
	once   sync.Once
	closed error
}

func NewEngine(cfg EngineConfig) (*Engine, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: cfg.UDPPort})
	if err != nil {
		return nil, fmt.Errorf("open media UDP socket: %w", err)
	}

	var mediaEngine webrtc.MediaEngine
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeVP8,
			ClockRate: 90000,
			RTCPFeedback: []webrtc.RTCPFeedback{
				{Type: "goog-remb"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
			},
		},
		PayloadType: 96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("register VP8: %w", err)
	}
	var registry interceptor.Registry
	if err := webrtc.RegisterDefaultInterceptors(&mediaEngine, &registry); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("register interceptors: %w", err)
	}
	var settings webrtc.SettingEngine
	settings.SetLite(true)
	settings.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4})
	settings.SetICETimeouts(8*time.Second, 20*time.Second, 2*time.Second)
	// Receive buffer stays at Ethernet size so a slightly larger datagram is
	// not truncated; senders are asked to stay at cfg.MTU (default 1200).
	settings.SetReceiveMTU(1500)
	loggerFactory := logging.NewDefaultLoggerFactory()
	loggerFactory.Writer = io.Discard
	settings.LoggerFactory = loggerFactory
	mux := webrtc.NewICEUDPMux(loggerFactory.NewLogger("ice"), conn)
	settings.SetICEUDPMux(mux)
	if cfg.PublicIP != "" {
		settings.SetNAT1To1IPs([]string{cfg.PublicIP}, webrtc.ICECandidateTypeHost)
	}
	mtu := cfg.MTU
	if mtu <= 0 {
		mtu = 1200
	}
	if mtu < 576 {
		mtu = 576
	}
	if mtu > 1200 {
		mtu = 1200
	}
	return &Engine{
		API:  webrtc.NewAPI(webrtc.WithMediaEngine(&mediaEngine), webrtc.WithInterceptorRegistry(&registry), webrtc.WithSettingEngine(settings)),
		MTU:  mtu,
		conn: conn,
		mux:  mux,
	}, nil
}

func (e *Engine) Close() error {
	if e == nil {
		return nil
	}
	e.once.Do(func() {
		if e.mux != nil {
			e.closed = e.mux.Close()
		} else if e.conn != nil {
			e.closed = e.conn.Close()
		}
	})
	return e.closed
}
