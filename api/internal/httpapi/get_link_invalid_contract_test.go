package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetLinkInvalidAndUnknown(t *testing.T) {
	srv := testServer(t)

	bad := doJSON(t, srv, http.MethodGet, "/api/v1/links/short", nil)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("400 got %d %s", bad.Code, bad.Body.String())
	}
	var body errorBody
	if err := json.Unmarshal(bad.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != CodeLinkInvalid {
		t.Fatalf("code %q", body.Error.Code)
	}

	missing := doJSON(t, srv, http.MethodGet, "/api/v1/links/Abcdefgh12", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("404 got %d %s", missing.Code, missing.Body.String())
	}
}
