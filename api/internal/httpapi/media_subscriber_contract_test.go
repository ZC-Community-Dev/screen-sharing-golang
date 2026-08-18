package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/pion/webrtc/v4"
)

func TestSubscriberRequiresViewerInSameLinkAndReadyPublisher(t *testing.T) {
	srv := testServer(t)
	link := createLink(t, srv)
	other := createLink(t, srv)
	join := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/viewer-sessions", nil)
	var viewer sessionResponse
	_ = json.Unmarshal(join.Body.Bytes(), &viewer)
	offer := subscriberOffer(t)

	wrongLink := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+other.ID+"/media/subscribers", subscriberOfferRequest{
		SessionID: viewer.SessionID, Offer: offer,
	})
	if wrongLink.Code != http.StatusUnauthorized {
		t.Fatalf("cross-room status=%d body=%s", wrongLink.Code, wrongLink.Body.String())
	}
	notReady := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/media/subscribers", subscriberOfferRequest{
		SessionID: viewer.SessionID, Offer: offer,
	})
	if notReady.Code != http.StatusConflict {
		t.Fatalf("not-ready status=%d body=%s", notReady.Code, notReady.Body.String())
	}
	for range 2 {
		deleted := doJSONWithHeader(t, srv, http.MethodDelete,
			"/api/v2/links/"+link.ID+"/media/subscribers/opaque-missing", nil,
			"X-Room-Session-ID", viewer.SessionID,
		)
		if deleted.Code != http.StatusNoContent {
			t.Fatalf("idempotent delete status=%d body=%s", deleted.Code, deleted.Body.String())
		}
	}
}

func subscriberOffer(t *testing.T) sdpPayload {
	t.Helper()
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pc.Close() })
	if _, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionRecvonly,
	}); err != nil {
		t.Fatal(err)
	}
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	done := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(offer); err != nil {
		t.Fatal(err)
	}
	<-done
	return sdpPayload{Type: "offer", SDP: pc.LocalDescription().SDP}
}
