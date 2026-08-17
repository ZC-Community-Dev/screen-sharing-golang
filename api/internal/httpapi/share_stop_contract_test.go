package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestShareStopKeepsLink(t *testing.T) {
	srv := testServer(t)
	created := createLink(t, srv)
	claim := doJSON(t, srv, http.MethodPost, "/api/v1/links/"+created.ID+"/presenter-sessions", map[string]string{
		"presenterToken": created.PresenterToken,
	})
	var sess sessionResponse
	_ = json.Unmarshal(claim.Body.Bytes(), &sess)
	start := doJSON(t, srv, http.MethodPost, "/api/v1/links/"+created.ID+"/share/start", presenterAuth{
		SessionID: sess.SessionID, PresenterToken: created.PresenterToken,
	})
	if start.Code != http.StatusOK {
		t.Fatalf("start %d %s", start.Code, start.Body.String())
	}
	stop := doJSON(t, srv, http.MethodPost, "/api/v1/links/"+created.ID+"/share/stop", presenterAuth{
		SessionID: sess.SessionID, PresenterToken: created.PresenterToken,
	})
	if stop.Code != http.StatusOK {
		t.Fatalf("stop %d %s", stop.Code, stop.Body.String())
	}
	var pub linkPublic
	if err := json.Unmarshal(stop.Body.Bytes(), &pub); err != nil {
		t.Fatal(err)
	}
	if pub.State != "waiting" {
		t.Fatalf("state %q", pub.State)
	}
	get := doJSON(t, srv, http.MethodGet, "/api/v1/links/"+created.ID, nil)
	if get.Code != http.StatusOK {
		t.Fatalf("get after stop %d", get.Code)
	}
}
