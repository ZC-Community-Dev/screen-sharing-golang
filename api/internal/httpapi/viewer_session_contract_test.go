package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestJoinViewerContract(t *testing.T) {
	srv := testServer(t)
	created := createLink(t, srv)
	rec := doJSON(t, srv, http.MethodPost, "/api/v1/links/"+created.ID+"/viewer-sessions", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	var sess sessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	if sess.Role != "viewer" || sess.ID != created.ID || sess.SessionID == "" {
		t.Fatalf("%+v", sess)
	}
}
