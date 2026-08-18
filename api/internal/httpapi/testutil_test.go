package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"api/internal/db"
	"api/internal/web"

	"github.com/gin-gonic/gin"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	database, err := db.Open(filepath.Join(t.TempDir(), "links.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	frontend, err := web.NewHandler(os.DirFS("../web/testdata/site"))
	if err != nil {
		t.Fatal(err)
	}
	srv := NewWithFrontend(
		Config{Salt: "test-salt-value", CORSOrigins: []string{"http://localhost:4200"}},
		database,
		NewLogger(),
		frontend,
	)
	t.Cleanup(func() { _ = srv.Close() })
	return srv
}

func doJSON(t *testing.T, srv *Server, method, path string, body any) *httptest.ResponseRecorder {
	return doJSONWithHeader(t, srv, method, path, body, "", "")
}

func doJSONWithHeader(t *testing.T, srv *Server, method, path string, body any, key, value string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	srv.Engine.ServeHTTP(rec, req)
	return rec
}

func createLink(t *testing.T, srv *Server) createLinkResponse {
	t.Helper()
	rec := doJSON(t, srv, http.MethodPost, "/api/v1/links", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d body %s", rec.Code, rec.Body.String())
	}
	var out createLinkResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}
