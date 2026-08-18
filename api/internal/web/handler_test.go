package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

func TestNewHandlerRequiresIndex(t *testing.T) {
	_, err := NewHandler(fstest.MapFS{
		"main.js": &fstest.MapFile{Data: []byte("console.log('x')")},
	})
	if err == nil || !strings.Contains(err.Error(), "frontend bundle") {
		t.Fatalf("expected frontend bundle error, got %v", err)
	}
}

func TestHandlerServesRootAndExactAsset(t *testing.T) {
	handler := fixtureHandler(t)

	root := request(handler, http.MethodGet, "/")
	if root.Code != http.StatusOK {
		t.Fatalf("root status=%d body=%s", root.Code, root.Body.String())
	}
	if got := root.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("root content-type=%q", got)
	}
	if !strings.Contains(root.Body.String(), "screen-share-fixture") {
		t.Fatal("root did not serve index.html")
	}

	asset := request(handler, http.MethodGet, "/main.abc123.js")
	if asset.Code != http.StatusOK {
		t.Fatalf("asset status=%d body=%s", asset.Code, asset.Body.String())
	}
	if got := asset.Header().Get("Content-Type"); !strings.Contains(got, "javascript") {
		t.Fatalf("asset content-type=%q", got)
	}
}

func TestHandlerSupportsHeadAndRejectsUnsupportedMethod(t *testing.T) {
	handler := fixtureHandler(t)

	head := request(handler, http.MethodHead, "/")
	if head.Code != http.StatusOK {
		t.Fatalf("HEAD status=%d", head.Code)
	}
	if head.Body.Len() != 0 {
		t.Fatalf("HEAD returned body=%q", head.Body.String())
	}

	post := request(handler, http.MethodPost, "/")
	if post.Code != http.StatusNotFound {
		t.Fatalf("POST status=%d", post.Code)
	}
}

func TestHandlerFallsBackForSPARoutes(t *testing.T) {
	handler := fixtureHandler(t)
	for _, target := range []string{"/r/Abcdefgh12", "/r/invalid", "//r//Abcdefgh12"} {
		rec := request(handler, http.MethodGet, target)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", target, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "screen-share-fixture") {
			t.Fatalf("%s did not receive index.html", target)
		}
	}
}

func TestHandlerDoesNotMaskMissingAssets(t *testing.T) {
	handler := fixtureHandler(t)
	rec := request(handler, http.MethodGet, "/main-missing.js")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerRejectsTraversal(t *testing.T) {
	handler := fixtureHandler(t)
	rec := request(handler, http.MethodGet, "/%2e%2e/index.html")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerSetsFrontendCachePolicy(t *testing.T) {
	handler := fixtureHandler(t)

	index := request(handler, http.MethodGet, "/")
	if got := index.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("index cache-control=%q", got)
	}

	asset := request(handler, http.MethodGet, "/main.abc123.js")
	if got := asset.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("asset cache-control=%q", got)
	}
}

func TestNewHandlerRejectsSecretFilenames(t *testing.T) {
	_, err := NewHandler(fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html></html>")},
		".env":       &fstest.MapFile{Data: []byte("LINK_ID_SALT=secret")},
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "forbidden") {
		t.Fatalf("expected forbidden bundle error, got %v", err)
	}
}

func fixtureHandler(t *testing.T) http.Handler {
	t.Helper()
	handler, err := NewHandler(os.DirFS("testdata/site"))
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func request(handler http.Handler, method, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
