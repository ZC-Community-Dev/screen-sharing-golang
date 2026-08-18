package httpapi

import (
	"testing"
)

func TestMediaConfigDefaultsAndValidation(t *testing.T) {
	t.Setenv("LINK_ID_SALT", "test-salt")
	for _, key := range []string{"MEDIA_UDP_PORT", "MEDIA_PUBLIC_IP", "MEDIA_MAX_ROOMS", "MEDIA_MAX_VIEWERS_PER_ROOM"} {
		t.Setenv(key, "")
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MediaUDPPort != 5000 || cfg.MediaMaxRooms != 20 || cfg.MediaMaxViewersPerRoom != 10 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}

	for _, tc := range []struct {
		name, key, value string
	}{
		{"port", "MEDIA_UDP_PORT", "0"},
		{"public ip", "MEDIA_PUBLIC_IP", "not-an-ip"},
		{"rooms", "MEDIA_MAX_ROOMS", "0"},
		{"viewers", "MEDIA_MAX_VIEWERS_PER_ROOM", "9"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LINK_ID_SALT", "test-salt")
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestDualTransportConfigValidationAndLazyUDP(t *testing.T) {
	t.Setenv("LINK_ID_SALT", "test-salt")
	t.Setenv("MEDIA_ALLOWED_TRANSPORTS", "websocket")
	t.Setenv("MEDIA_DEFAULT_TRANSPORT", "websocket")
	t.Setenv("MEDIA_UDP_PORT", "not-a-port")
	t.Setenv("MEDIA_PUBLIC_IP", "not-an-ip")
	t.Setenv("MEDIA_WS_MAX_CHUNK_BYTES", "1048576")
	t.Setenv("MEDIA_WS_MAX_BUFFER_BYTES", "2097152")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("websocket-only config initialized UDP: %v", err)
	}
	if len(cfg.MediaAllowedTransports) != 1 || cfg.MediaAllowedTransports[0] != "websocket" ||
		cfg.MediaWSMaxChunkBytes != 1048576 || cfg.MediaWSMaxBufferBytes != 2097152 {
		t.Fatalf("config=%+v", cfg)
	}

	for _, test := range []struct {
		name, allowed, defaultTransport, chunk, buffer string
	}{
		{"unknown", "websocket,tcp", "websocket", "1", "1"},
		{"duplicate", "websocket,websocket", "websocket", "1", "1"},
		{"default outside allowed", "websocket", "webrtc", "1", "1"},
		{"invalid chunk", "websocket", "websocket", "0", "1"},
		{"invalid buffer", "websocket", "websocket", "1", "0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LINK_ID_SALT", "test-salt")
			t.Setenv("MEDIA_ALLOWED_TRANSPORTS", test.allowed)
			t.Setenv("MEDIA_DEFAULT_TRANSPORT", test.defaultTransport)
			t.Setenv("MEDIA_WS_MAX_CHUNK_BYTES", test.chunk)
			t.Setenv("MEDIA_WS_MAX_BUFFER_BYTES", test.buffer)
			if _, err := Load(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
