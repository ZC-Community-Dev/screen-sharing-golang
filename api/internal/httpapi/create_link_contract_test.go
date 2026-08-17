package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"api/internal/ids"
)

func TestCreateLinkContract(t *testing.T) {
	srv := testServer(t)
	rec := doJSON(t, srv, http.MethodPost, "/api/v1/links", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var out createLinkResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !ids.ValidPublicID(out.ID) {
		t.Fatalf("invalid id %q", out.ID)
	}
	if out.PublicURL != "/r/"+out.ID {
		t.Fatalf("publicUrl %q", out.PublicURL)
	}
	if out.PresenterToken == "" {
		t.Fatal("missing presenterToken")
	}
	if out.PublicURL == out.PresenterToken {
		t.Fatal("token leaked into public url")
	}
}
