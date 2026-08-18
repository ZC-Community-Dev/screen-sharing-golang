package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestV2PreservesLinkAndSessionContracts(t *testing.T) {
	srv := testServer(t)
	create := doJSON(t, srv, http.MethodPost, "/api/v2/links", nil)
	if create.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", create.Code, create.Body.String())
	}
	var link createLinkResponse
	if err := json.Unmarshal(create.Body.Bytes(), &link); err != nil {
		t.Fatal(err)
	}
	if link.ID == "" || link.PresenterToken == "" {
		t.Fatal("missing compatible link fields")
	}
	for _, request := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v2/links/" + link.ID},
		{method: http.MethodPost, path: "/api/v2/links/" + link.ID + "/viewer-sessions"},
	} {
		if rec := doJSON(t, srv, request.method, request.path, nil); rec.Code >= 300 {
			t.Fatalf("%s: %d %s", request.path, rec.Code, rec.Body.String())
		}
	}
	claim := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/presenter-sessions", map[string]string{
		"presenterToken": link.PresenterToken,
	})
	if claim.Code != http.StatusCreated {
		t.Fatalf("claim: %d %s", claim.Code, claim.Body.String())
	}
}
