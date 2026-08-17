package httpapi

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"api/internal/db"

	"github.com/gin-gonic/gin"
)

func TestLinkPersistsAcrossReopen(t *testing.T) {
	gin.SetMode(gin.TestMode)
	path := filepath.Join(t.TempDir(), "links.db")
	db1, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	srv1 := New(Config{Salt: "test-salt-value", CORSOrigins: []string{"http://localhost:4200"}}, db1, NewLogger())
	created := createLink(t, srv1)
	_ = db1.Close()

	db2, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db2.Close() })
	srv2 := New(Config{Salt: "test-salt-value", CORSOrigins: []string{"http://localhost:4200"}}, db2, NewLogger())
	rec := doJSON(t, srv2, http.MethodGet, "/api/v1/links/"+created.ID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var out linkPublic
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.ID != created.ID {
		t.Fatalf("got %q", out.ID)
	}
}
