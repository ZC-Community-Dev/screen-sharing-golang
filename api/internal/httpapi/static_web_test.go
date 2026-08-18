package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInjectedFrontendServesRootThroughGin(t *testing.T) {
	srv := testServerWithFrontend(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.Engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("content-type=%q", got)
	}
	if !strings.Contains(rec.Body.String(), "screen-share-fixture") {
		t.Fatal("missing embedded fixture")
	}
}

func TestInjectedFrontendServesDirectRoomRoutes(t *testing.T) {
	srv := testServerWithFrontend(t)
	for _, target := range []string{"/r/Abcdefgh12", "/r/invalid"} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		srv.Engine.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", target, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "screen-share-fixture") {
			t.Fatalf("%s did not receive index.html", target)
		}
	}
}

func TestFrontendDoesNotChangeKnownAPIContract(t *testing.T) {
	srv := testServerWithFrontend(t)
	created := createLink(t, srv)
	if created.ID == "" || created.PresenterToken == "" {
		t.Fatalf("invalid create response: %+v", created)
	}
}

func TestUnknownAPIPathReturnsJSONNotSPA(t *testing.T) {
	srv := testServerWithFrontend(t)
	rec := doJSON(t, srv, http.MethodGet, "/api/v1/inexistente", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("content-type=%q body=%s", got, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "screen-share-fixture") {
		t.Fatal("API error returned SPA")
	}
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
}

func TestUnknownNonGETPathNeverReturnsSPA(t *testing.T) {
	srv := testServerWithFrontend(t)
	rec := doJSON(t, srv, http.MethodPost, "/rota-inexistente", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "screen-share-fixture") {
		t.Fatal("POST error returned SPA")
	}
}

func TestWebSocketRoutePrecedesFrontendFallback(t *testing.T) {
	srv := testServerWithFrontend(t)
	rec := doJSON(t, srv, http.MethodGet, "/api/v1/links/Abcdefgh12/events?sessionId=missing", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "screen-share-fixture") {
		t.Fatal("WebSocket API route returned SPA")
	}
}

func testServerWithFrontend(t *testing.T) *Server {
	t.Helper()
	return testServer(t)
}
