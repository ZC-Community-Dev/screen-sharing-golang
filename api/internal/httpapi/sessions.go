package httpapi

import (
	"errors"
	"net/http"

	"api/internal/ids"
	"api/internal/links"
	"api/internal/room"

	"github.com/gin-gonic/gin"
)

type claimPresenterRequest struct {
	PresenterToken string `json:"presenterToken"`
}

type presenterAuth struct {
	SessionID      string `json:"sessionId"`
	PresenterToken string `json:"presenterToken"`
}

type sessionResponse struct {
	SessionID string `json:"sessionId"`
	Role      string `json:"role"`
	ID        string `json:"id"`
	State     string `json:"state"`
}

func (s *Server) claimPresenter(c *gin.Context) {
	id, ok := s.parseID(c)
	if !ok {
		return
	}
	var req claimPresenterRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PresenterToken == "" {
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "presenter token required")
		return
	}
	if err := s.Links.VerifyToken(id, req.PresenterToken); err != nil {
		if errors.Is(err, links.ErrNotFound) {
			writeError(c, http.StatusNotFound, CodeLinkNotFound, "link not found")
			return
		}
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "invalid presenter token")
		return
	}
	sessionID, err := s.Hub.ClaimPresenter(id)
	if errors.Is(err, room.ErrPresenterConflict) {
		writeError(c, http.StatusConflict, CodePresenterConflict, "a presenter is already active")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, CodeInternalError, "could not claim presenter")
		return
	}
	link, _ := s.Links.GetByID(id)
	c.JSON(http.StatusCreated, sessionResponse{
		SessionID: sessionID,
		Role:      string(room.RolePresenter),
		ID:        id,
		State:     link.State,
	})
}

func (s *Server) joinViewer(c *gin.Context) {
	id, ok := s.parseID(c)
	if !ok {
		return
	}
	if _, err := s.Links.GetByID(id); err != nil {
		if errors.Is(err, links.ErrNotFound) {
			writeError(c, http.StatusNotFound, CodeLinkNotFound, "link not found")
			return
		}
		writeError(c, http.StatusInternalServerError, CodeInternalError, "could not join")
		return
	}
	sessionID := s.Hub.JoinViewer(id)
	link, _ := s.Links.GetByID(id)
	c.JSON(http.StatusCreated, sessionResponse{
		SessionID: sessionID,
		Role:      string(room.RoleViewer),
		ID:        id,
		State:     link.State,
	})
}

func (s *Server) startShare(c *gin.Context) {
	s.mutateShare(c, true)
}

func (s *Server) stopShare(c *gin.Context) {
	s.mutateShare(c, false)
}

func (s *Server) mutateShare(c *gin.Context, start bool) {
	id, ok := s.parseID(c)
	if !ok {
		return
	}
	var req presenterAuth
	if err := c.ShouldBindJSON(&req); err != nil || req.SessionID == "" || req.PresenterToken == "" {
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "presenter credentials required")
		return
	}
	if err := s.Links.VerifyToken(id, req.PresenterToken); err != nil {
		if errors.Is(err, links.ErrNotFound) {
			writeError(c, http.StatusNotFound, CodeLinkNotFound, "link not found")
			return
		}
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "invalid presenter token")
		return
	}
	var err error
	if start {
		err = s.Hub.StartShare(id, req.SessionID)
	} else {
		err = s.Hub.StopShare(id, req.SessionID)
	}
	switch {
	case errors.Is(err, room.ErrUnauthorized), errors.Is(err, room.ErrUnknownSession):
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "invalid presenter session")
		return
	case errors.Is(err, room.ErrShareConflict):
		writeError(c, http.StatusConflict, CodeShareConflict, "share already active")
		return
	case err != nil:
		writeError(c, http.StatusInternalServerError, CodeInternalError, "could not update share")
		return
	}
	state := links.StateWaiting
	if start {
		state = links.StateSharing
	}
	if err := s.Links.SetState(id, state); err != nil {
		writeError(c, http.StatusInternalServerError, CodeInternalError, "could not persist state")
		return
	}
	link, _ := s.Links.GetByID(id)
	c.JSON(http.StatusOK, s.publicLink(link))
}

func (s *Server) parseID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if !ids.ValidPublicID(id) {
		writeError(c, http.StatusBadRequest, CodeLinkInvalid, "invalid link")
		return "", false
	}
	return id, true
}
