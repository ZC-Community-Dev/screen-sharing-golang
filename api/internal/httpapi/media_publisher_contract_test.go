package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/pion/webrtc/v4"
)

func TestPublisherContractAuthorizationSDPAndConflict(t *testing.T) {
	srv := testServer(t)
	link := createLink(t, srv)
	viewer := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/viewer-sessions", nil)
	var viewerSession sessionResponse
	_ = json.Unmarshal(viewer.Body.Bytes(), &viewerSession)
	offer := publisherOffer(t)

	rejected := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/media/publisher", publisherOfferRequest{
		SessionID: viewerSession.SessionID, PresenterToken: link.PresenterToken, Offer: offer,
	})
	if rejected.Code != http.StatusUnauthorized {
		t.Fatalf("viewer publish status=%d body=%s", rejected.Code, rejected.Body.String())
	}

	claim := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/presenter-sessions", map[string]string{"presenterToken": link.PresenterToken})
	var presenter sessionResponse
	_ = json.Unmarshal(claim.Body.Bytes(), &presenter)
	invalid := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/media/publisher", publisherOfferRequest{
		SessionID: presenter.SessionID, PresenterToken: link.PresenterToken, Offer: sdpPayload{Type: "offer", SDP: "secret-invalid-sdp"},
	})
	if invalid.Code != http.StatusBadRequest || strings.Contains(invalid.Body.String(), "secret-invalid-sdp") {
		t.Fatalf("unsafe invalid response=%d %s", invalid.Code, invalid.Body.String())
	}

	created := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/media/publisher", publisherOfferRequest{
		SessionID: presenter.SessionID, PresenterToken: link.PresenterToken, Offer: offer,
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("publisher status=%d body=%s", created.Code, created.Body.String())
	}
	var answer mediaAnswerResponse
	if err := json.Unmarshal(created.Body.Bytes(), &answer); err != nil {
		t.Fatal(err)
	}
	if answer.MediaSessionID == "" || answer.Answer.Type != "answer" || !strings.Contains(answer.Answer.SDP, "a=ice-lite") {
		t.Fatalf("invalid answer metadata")
	}
	conflict := doJSON(t, srv, http.MethodPost, "/api/v2/links/"+link.ID+"/media/publisher", publisherOfferRequest{
		SessionID: presenter.SessionID, PresenterToken: link.PresenterToken, Offer: offer,
	})
	if conflict.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", conflict.Code, conflict.Body.String())
	}
}

func TestMediaErrorsAndLogsDoNotEchoSensitivePayload(t *testing.T) {
	var logs bytes.Buffer
	srv := testServer(t)
	srv.Log = slog.New(slog.NewJSONHandler(&logs, nil))
	secret := "token-secret 203.0.113.7 RTP-payload"
	rec := doJSON(t, srv, http.MethodPost, "/api/v2/links/Abcdefgh12/media/publisher", map[string]any{
		"sessionId": secret, "presenterToken": secret, "offer": map[string]string{"type": "offer", "sdp": secret},
	})
	if strings.Contains(rec.Body.String(), secret) || strings.Contains(logs.String(), secret) {
		t.Fatal("sensitive media request was exposed")
	}
}

func publisherOffer(t *testing.T) sdpPayload {
	t.Helper()
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pc.Close() })
	track, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}, "screen", "publisher",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pc.AddTrack(track); err != nil {
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
