package httpapi

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			c.Next()
			return
		}
		if !s.originAllowed(c.Request, false) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, X-Room-Session-ID")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (s *Server) allowWebSocketOrigin(request *http.Request) bool {
	return s.originAllowed(request, false)
}

func (s *Server) validMediaOrigin(request *http.Request) bool {
	return s.originAllowed(request, true)
}

func (s *Server) originAllowed(request *http.Request, requireOrigin bool) bool {
	raw := request.Header.Get("Origin")
	if raw == "" {
		return !requireOrigin
	}
	origin, err := url.Parse(raw)
	if err != nil || origin.Host == "" || (origin.Scheme != "http" && origin.Scheme != "https") {
		return false
	}
	if sameHost(origin.Host, request.Host) {
		return true
	}
	for _, allowed := range s.Config.CORSOrigins {
		if raw == allowed {
			return true
		}
	}
	return false
}

func sameHost(originHost, requestHost string) bool {
	return canonicalHost(originHost) != "" && canonicalHost(originHost) == canonicalHost(requestHost)
}

func canonicalHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return ""
	}
	if parsed, err := url.Parse("http://" + host); err == nil && parsed.Host != "" {
		host = parsed.Host
	}
	if h, port, err := net.SplitHostPort(host); err == nil {
		if port == "80" || port == "443" {
			return h
		}
		return strings.ToLower(h) + ":" + port
	}
	return strings.Trim(host, "[]")
}
