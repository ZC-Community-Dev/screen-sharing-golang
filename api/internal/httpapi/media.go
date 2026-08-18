package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"api/internal/links"
	"api/internal/media"
	"api/internal/room"

	"github.com/gin-gonic/gin"
	"github.com/pion/webrtc/v4"
)

const maxSDPSize = 131072

type sdpPayload struct {
	Type string `json:"type"`
	SDP  string `json:"sdp"`
}

type publisherOfferRequest struct {
	SessionID      string     `json:"sessionId"`
	PresenterToken string     `json:"presenterToken"`
	Offer          sdpPayload `json:"offer"`
}

type subscriberOfferRequest struct {
	SessionID string     `json:"sessionId"`
	Offer     sdpPayload `json:"offer"`
}

type mediaAnswerResponse struct {
	MediaSessionID string     `json:"mediaSessionId"`
	Answer         sdpPayload `json:"answer"`
}

func (s *Server) createPublisher(c *gin.Context) {
	id, ok := s.parseID(c)
	if !ok {
		return
	}
	if !containsTransport(s.Config.MediaAllowedTransports, media.TransportWebRTC) {
		writeError(c, http.StatusForbidden, CodeTransportNotAllowed, "webrtc media transport is not enabled")
		return
	}
	var req publisherOfferRequest
	if !decodeMediaJSON(c, &req) || !validOffer(req.Offer) || !vp8VideoOnly(req.Offer.SDP) {
		writeError(c, http.StatusBadRequest, CodeInvalidSDP, "invalid media offer")
		return
	}
	if err := s.Links.VerifyToken(id, req.PresenterToken); err != nil {
		if errors.Is(err, links.ErrNotFound) {
			writeError(c, http.StatusNotFound, CodeLinkNotFound, "link not found")
		} else {
			writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "presenter authorization failed")
		}
		return
	}
	if role, exists := s.Hub.SessionRole(id, req.SessionID); !exists || role != room.RolePresenter {
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "presenter authorization failed")
		return
	}
	mediaID, answer, err := s.Media.CreatePublisher(id, req.SessionID, webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer, SDP: req.Offer.SDP,
	})
	if err != nil {
		s.writeMediaError(c, err)
		return
	}
	c.JSON(http.StatusCreated, mediaAnswerResponse{
		MediaSessionID: mediaID,
		Answer:         sdpPayload{Type: "answer", SDP: answer.SDP},
	})
}

func (s *Server) createSubscriber(c *gin.Context) {
	id, ok := s.parseID(c)
	if !ok {
		return
	}
	if !containsTransport(s.Config.MediaAllowedTransports, media.TransportWebRTC) {
		writeError(c, http.StatusForbidden, CodeTransportNotAllowed, "webrtc media transport is not enabled")
		return
	}
	var req subscriberOfferRequest
	if !decodeMediaJSON(c, &req) || !validOffer(req.Offer) || !vp8VideoOnly(req.Offer.SDP) {
		writeError(c, http.StatusBadRequest, CodeInvalidSDP, "invalid media offer")
		return
	}
	if _, err := s.Links.GetByID(id); err != nil {
		writeError(c, http.StatusNotFound, CodeLinkNotFound, "link not found")
		return
	}
	if role, exists := s.Hub.SessionRole(id, req.SessionID); !exists || role != room.RoleViewer {
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "viewer session is invalid")
		return
	}
	mediaID, answer, err := s.Media.CreateSubscriber(id, req.SessionID, webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer, SDP: req.Offer.SDP,
	})
	if err != nil {
		s.writeMediaError(c, err)
		return
	}
	c.JSON(http.StatusCreated, mediaAnswerResponse{
		MediaSessionID: mediaID,
		Answer:         sdpPayload{Type: "answer", SDP: answer.SDP},
	})
}

func (s *Server) deleteSubscriber(c *gin.Context) {
	id, ok := s.parseID(c)
	if !ok {
		return
	}
	owner := c.GetHeader("X-Room-Session-ID")
	if role, exists := s.Hub.SessionRole(id, owner); !exists || role != room.RoleViewer {
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "viewer session is invalid")
		return
	}
	if err := s.Media.CloseSubscriber(id, c.Param("mediaSessionId"), owner); err != nil {
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "media session ownership failed")
		return
	}
	c.Status(http.StatusNoContent)
}

func decodeMediaJSON(c *gin.Context, target any) bool {
	reader := http.MaxBytesReader(c.Writer, c.Request.Body, maxSDPSize+8192)
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return false
	}
	return decoder.Decode(&struct{}{}) == io.EOF
}

func validOffer(offer sdpPayload) bool {
	return offer.Type == "offer" && strings.TrimSpace(offer.SDP) != "" && len(offer.SDP) <= maxSDPSize
}

func vp8VideoOnly(sdp string) bool {
	upper := strings.ToUpper(sdp)
	return strings.Contains(upper, "\nM=VIDEO ") &&
		strings.Contains(upper, "VP8/90000") &&
		!strings.Contains(upper, "\nM=AUDIO ")
}

func (s *Server) writeMediaError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, media.ErrInvalidSDP):
		writeError(c, http.StatusBadRequest, CodeInvalidSDP, "invalid media offer")
	case errors.Is(err, media.ErrPublisherConflict):
		writeError(c, http.StatusConflict, CodeMediaConflict, "media publisher already exists")
	case errors.Is(err, media.ErrNotReady):
		writeError(c, http.StatusConflict, CodeMediaNotReady, "publisher media is not ready")
	case errors.Is(err, media.ErrCapacity):
		writeError(c, http.StatusTooManyRequests, CodeMediaCapacity, "media capacity reached")
	case errors.Is(err, media.ErrSessionNotFound):
		writeError(c, http.StatusNotFound, CodeMediaSessionNotFound, "media session not found")
	case errors.Is(err, media.ErrTransportInvalid):
		writeError(c, http.StatusBadRequest, CodeTransportInvalid, "invalid media transport")
	case errors.Is(err, media.ErrTransportMismatch):
		writeError(c, http.StatusConflict, CodeTransportMismatch, "publication uses another transport")
	case errors.Is(err, media.ErrUnauthorized):
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "media session ownership failed")
	default:
		s.Log.Error("media operation failed", "category", "internal")
		writeError(c, http.StatusInternalServerError, CodeInternalError, "media operation failed")
	}
}
