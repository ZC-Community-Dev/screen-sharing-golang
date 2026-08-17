package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestClaimPresenterContract(t *testing.T) {
	srv := testServer(t)
	created := createLink(t, srv)

	rec := doJSON(t, srv, http.MethodPost, "/api/v1/links/"+created.ID+"/presenter-sessions", map[string]string{
		"presenterToken": created.PresenterToken,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("201 got %d %s", rec.Code, rec.Body.String())
	}
	var sess sessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	if sess.Role != "presenter" || sess.SessionID == "" {
		t.Fatalf("%+v", sess)
	}

	bad := doJSON(t, srv, http.MethodPost, "/api/v1/links/"+created.ID+"/presenter-sessions", map[string]string{
		"presenterToken": "not-the-token",
	})
	if bad.Code != http.StatusUnauthorized {
		t.Fatalf("401 got %d %s", bad.Code, bad.Body.String())
	}

	conflict := doJSON(t, srv, http.MethodPost, "/api/v1/links/"+created.ID+"/presenter-sessions", map[string]string{
		"presenterToken": created.PresenterToken,
	})
	if conflict.Code != http.StatusConflict {
		t.Fatalf("409 got %d %s", conflict.Code, conflict.Body.String())
	}
}
