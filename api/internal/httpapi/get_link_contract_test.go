package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestGetLinkOmitsPresenterToken(t *testing.T) {
	srv := testServer(t)
	created := createLink(t, srv)
	rec := doJSON(t, srv, http.MethodGet, "/api/v1/links/"+created.ID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), created.PresenterToken) {
		t.Fatal("presenter token leaked")
	}
	if strings.Contains(rec.Body.String(), "presenterToken") {
		t.Fatal("presenterToken field present")
	}
	var out linkPublic
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.ID != created.ID || out.State != "waiting" {
		t.Fatalf("unexpected body %+v", out)
	}
}
