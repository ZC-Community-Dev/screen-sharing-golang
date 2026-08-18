package httpapi

import (
	"net/http"

	"api/internal/media"

	"github.com/gin-gonic/gin"
)

const (
	webSocketMIMEType        = "video/webm;codecs=vp8"
	webSocketTimesliceMS     = 250
	webSocketStartupBufferMS = 2000
)

type publicMediaConfig struct {
	AllowedTransports []media.Transport `json:"allowedTransports"`
	DefaultTransport  media.Transport   `json:"defaultTransport"`
	WebSocket         struct {
		MIMEType        string `json:"mimeType"`
		TimesliceMS     int    `json:"timesliceMs"`
		StartupBufferMS int    `json:"startupBufferMs"`
		MaxChunkBytes   int    `json:"maxChunkBytes"`
	} `json:"websocket"`
}

func normalizeMediaConfig(cfg Config) Config {
	if len(cfg.MediaAllowedTransports) == 0 {
		cfg.MediaAllowedTransports = []media.Transport{media.TransportWebRTC, media.TransportWebSocket}
	}
	if cfg.MediaDefaultTransport == "" {
		cfg.MediaDefaultTransport = media.TransportWebRTC
	}
	if cfg.MediaWSMaxChunkBytes <= 0 {
		cfg.MediaWSMaxChunkBytes = 4 << 20
	}
	if cfg.MediaWSMaxBufferBytes <= 0 {
		cfg.MediaWSMaxBufferBytes = 8 << 20
	}
	return cfg
}

func (s *Server) getMediaConfig(c *gin.Context) {
	var response publicMediaConfig
	response.AllowedTransports = append([]media.Transport(nil), s.Config.MediaAllowedTransports...)
	response.DefaultTransport = s.Config.MediaDefaultTransport
	response.WebSocket.MIMEType = webSocketMIMEType
	response.WebSocket.TimesliceMS = webSocketTimesliceMS
	response.WebSocket.StartupBufferMS = webSocketStartupBufferMS
	response.WebSocket.MaxChunkBytes = s.Config.MediaWSMaxChunkBytes
	c.JSON(http.StatusOK, response)
}
