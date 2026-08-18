package httpapi

import (
	"net/http"
	"testing"
)

func TestOriginAllowedAcceptsCloudflareSameOrigin(t *testing.T) {
	srv := &Server{Config: Config{CORSOrigins: []string{"http://localhost:4200"}}}
	req := httptestRequest("example.local", "https://example.local")
	if !srv.originAllowed(req, true) {
		t.Fatal("https public origin behind Cloudflare must be accepted as same-origin")
	}
	req = httptestRequest("example.local:443", "https://example.local")
	if !srv.originAllowed(req, true) {
		t.Fatal("default https port must match hostname-only Origin")
	}
}

func TestOriginAllowedRejectsCrossSiteAndAllowsConfiguredCORS(t *testing.T) {
	srv := &Server{Config: Config{CORSOrigins: []string{"https://example.local"}}}
	if srv.originAllowed(httptestRequest("192.168.10.108:8080", "https://attacker.invalid"), false) {
		t.Fatal("cross-site origin must be rejected")
	}
	if !srv.originAllowed(httptestRequest("192.168.10.108:8080", "https://example.local"), true) {
		t.Fatal("configured CORS origin must be accepted when Host is the origin IP")
	}
	if srv.originAllowed(httptestRequest("example.local", ""), true) {
		t.Fatal("media sockets require Origin")
	}
	if !srv.originAllowed(httptestRequest("example.local", ""), false) {
		t.Fatal("event sockets may omit Origin")
	}
}

func httptestRequest(host, origin string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "https://"+host+"/api/v2/links/x/events", nil)
	req.Host = host
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	return req
}
