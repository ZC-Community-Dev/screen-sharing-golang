package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestShareStartContract(t *testing.T) {
	srv := testServer(t)
	created := createLink(t, srv)
	claim := doJSON(t, srv, http.MethodPost, "/api/v1/links/"+created.ID+"/presenter-sessions", map[string]string{
		"presenterToken": created.PresenterToken,
	})
	var sess sessionResponse
	if err := json.Unmarshal(claim.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}

	unauth := doJSON(t, srv, http.MethodPost, "/api/v1/links/"+created.ID+"/share/start", presenterAuth{
		SessionID: sess.SessionID, PresenterToken: "wrong",
	})
	if unauth.Code != http.StatusUnauthorized {
		t.Fatalf("401 got %d %s", unauth.Code, unauth.Body.String())
	}

	ok := doJSON(t, srv, http.MethodPost, "/api/v1/links/"+created.ID+"/share/start", presenterAuth{
		SessionID: sess.SessionID, PresenterToken: created.PresenterToken,
	})
	if ok.Code != http.StatusOK {
		t.Fatalf("200 got %d %s", ok.Code, ok.Body.String())
	}
	var pub linkPublic
	if err := json.Unmarshal(ok.Body.Bytes(), &pub); err != nil {
		t.Fatal(err)
	}
	if pub.State != "sharing" {
		t.Fatalf("state %q", pub.State)
	}

	again := doJSON(t, srv, http.MethodPost, "/api/v1/links/"+created.ID+"/share/start", presenterAuth{
		SessionID: sess.SessionID, PresenterToken: created.PresenterToken,
	})
	if again.Code != http.StatusConflict {
		t.Fatalf("409 got %d %s", again.Code, again.Body.String())
	}
}
