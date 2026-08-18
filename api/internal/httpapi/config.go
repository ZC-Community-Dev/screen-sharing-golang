package httpapi

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"api/internal/media"
	"github.com/joho/godotenv"
)

// LoadEnvFile reads path into the process environment without overriding
// variables already set. A missing file is ignored.
func LoadEnvFile(path string) error {
	if path == "" {
		path = ".env"
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return godotenv.Load(path)
}

type Config struct {
	Salt                   string
	DBPath                 string
	Port                   string
	CORSOrigins            []string
	MediaUDPPort           int
	MediaPublicIP          string
	MediaUDPMTU            int
	MediaMaxRooms          int
	MediaMaxViewersPerRoom int
	MediaAllowedTransports []media.Transport
	MediaDefaultTransport  media.Transport
	MediaWSMaxChunkBytes   int
	MediaWSMaxBufferBytes  int
}

func Load() (Config, error) {
	salt := os.Getenv("LINK_ID_SALT")
	if salt == "" {
		return Config{}, fmt.Errorf("LINK_ID_SALT is required")
	}
	dbPath := os.Getenv("LINKS_DB_PATH")
	if dbPath == "" {
		dbPath = "data/links.db"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	allowed, err := parseTransports(os.Getenv("MEDIA_ALLOWED_TRANSPORTS"))
	if err != nil {
		return Config{}, err
	}
	defaultTransport := media.Transport(strings.TrimSpace(os.Getenv("MEDIA_DEFAULT_TRANSPORT")))
	if defaultTransport == "" {
		defaultTransport = media.TransportWebRTC
	}
	if !containsTransport(allowed, defaultTransport) {
		return Config{}, fmt.Errorf("MEDIA_DEFAULT_TRANSPORT must be enabled")
	}
	mediaPort := 5000
	publicIP := ""
	if containsTransport(allowed, media.TransportWebRTC) {
		mediaPort, err = envInt("MEDIA_UDP_PORT", 5000)
		if err != nil || mediaPort < 1 || mediaPort > 65535 {
			return Config{}, fmt.Errorf("MEDIA_UDP_PORT must be between 1 and 65535")
		}
		publicIP = os.Getenv("MEDIA_PUBLIC_IP")
		if publicIP != "" && net.ParseIP(publicIP).To4() == nil {
			return Config{}, fmt.Errorf("MEDIA_PUBLIC_IP must be a valid IPv4 address")
		}
	}
	udpMTU, err := envInt("MEDIA_UDP_MTU", 1200)
	if err != nil || udpMTU < 576 || udpMTU > 1200 {
		return Config{}, fmt.Errorf("MEDIA_UDP_MTU must be between 576 and 1200")
	}
	maxRooms, err := envInt("MEDIA_MAX_ROOMS", 20)
	if err != nil || maxRooms < 1 {
		return Config{}, fmt.Errorf("MEDIA_MAX_ROOMS must be greater than zero")
	}
	maxViewers, err := envInt("MEDIA_MAX_VIEWERS_PER_ROOM", 10)
	if err != nil || maxViewers < 10 {
		return Config{}, fmt.Errorf("MEDIA_MAX_VIEWERS_PER_ROOM must be at least 10")
	}
	maxChunk, err := envInt("MEDIA_WS_MAX_CHUNK_BYTES", 4<<20)
	if err != nil || maxChunk < 1 || maxChunk > 64<<20 {
		return Config{}, fmt.Errorf("MEDIA_WS_MAX_CHUNK_BYTES must be between 1 and 67108864")
	}
	maxBuffer, err := envInt("MEDIA_WS_MAX_BUFFER_BYTES", 8<<20)
	if err != nil || maxBuffer < 1 || maxBuffer > 128<<20 {
		return Config{}, fmt.Errorf("MEDIA_WS_MAX_BUFFER_BYTES must be between 1 and 134217728")
	}
	origins, err := parseCORSOrigins(os.Getenv("CORS_ORIGINS"))
	if err != nil {
		return Config{}, err
	}
	return Config{
		Salt:                   salt,
		DBPath:                 dbPath,
		Port:                   port,
		CORSOrigins:            origins,
		MediaUDPPort:           mediaPort,
		MediaPublicIP:          publicIP,
		MediaUDPMTU:            udpMTU,
		MediaMaxRooms:          maxRooms,
		MediaMaxViewersPerRoom: maxViewers,
		MediaAllowedTransports: allowed,
		MediaDefaultTransport:  defaultTransport,
		MediaWSMaxChunkBytes:   maxChunk,
		MediaWSMaxBufferBytes:  maxBuffer,
	}, nil
}

func envInt(name string, fallback int) (int, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}

func parseTransports(raw string) ([]media.Transport, error) {
	if strings.TrimSpace(raw) == "" {
		return []media.Transport{media.TransportWebRTC, media.TransportWebSocket}, nil
	}
	seen := make(map[media.Transport]bool)
	var transports []media.Transport
	for _, item := range strings.Split(raw, ",") {
		transport := media.Transport(strings.TrimSpace(item))
		if !transport.Valid() || seen[transport] {
			return nil, fmt.Errorf("MEDIA_ALLOWED_TRANSPORTS must contain unique webrtc/websocket values")
		}
		seen[transport] = true
		transports = append(transports, transport)
	}
	if len(transports) == 0 {
		return nil, fmt.Errorf("MEDIA_ALLOWED_TRANSPORTS must not be empty")
	}
	return transports, nil
}

func parseCORSOrigins(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return []string{"http://localhost:4200", "http://127.0.0.1:4200"}, nil
	}
	seen := make(map[string]bool)
	var origins []string
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parsed, err := url.Parse(item)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.Path != "" {
			return nil, fmt.Errorf("CORS_ORIGINS must be a comma-separated list of http(s) origins")
		}
		if seen[item] {
			return nil, fmt.Errorf("CORS_ORIGINS must contain unique origins")
		}
		seen[item] = true
		origins = append(origins, item)
	}
	if len(origins) == 0 {
		return nil, fmt.Errorf("CORS_ORIGINS must not be empty")
	}
	return origins, nil
}

func containsTransport(transports []media.Transport, target media.Transport) bool {
	for _, transport := range transports {
		if transport == target {
			return true
		}
	}
	return false
}
