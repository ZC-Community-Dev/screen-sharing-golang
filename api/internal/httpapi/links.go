package httpapi

import (
	"errors"
	"net/http"

	"api/internal/ids"
	"api/internal/links"

	"github.com/gin-gonic/gin"
)

type createLinkResponse struct {
	ID             string `json:"id"`
	PublicURL      string `json:"publicUrl"`
	PresenterToken string `json:"presenterToken"`
}

type linkPublic struct {
	ID               string `json:"id"`
	State            string `json:"state"`
	ParticipantCount int    `json:"participantCount"`
	Publication      any    `json:"publication,omitempty"`
}

func (s *Server) createLink(c *gin.Context) {
	created, err := s.Links.Create()
	if err != nil {
		s.Log.Error("create link failed", "err", err.Error())
		writeError(c, http.StatusInternalServerError, CodeInternalError, "could not create link")
		return
	}
	c.JSON(http.StatusCreated, createLinkResponse{
		ID:             created.ID,
		PublicURL:      created.PublicURL,
		PresenterToken: created.PresenterToken,
	})
}

func (s *Server) getLink(c *gin.Context) {
	id := c.Param("id")
	if !ids.ValidPublicID(id) {
		writeError(c, http.StatusBadRequest, CodeLinkInvalid, "invalid link")
		return
	}
	link, err := s.Links.GetByID(id)
	if errors.Is(err, links.ErrNotFound) {
		writeError(c, http.StatusNotFound, CodeLinkNotFound, "link not found")
		return
	}
	if err != nil {
		s.Log.Error("get link failed", "err", err.Error())
		writeError(c, http.StatusInternalServerError, CodeInternalError, "could not load link")
		return
	}
	c.JSON(http.StatusOK, s.publicLink(link))
}

func (s *Server) publicLink(link links.Link) linkPublic {
	return linkPublic{
		ID:               link.ID,
		State:            link.State,
		ParticipantCount: s.Hub.Count(link.ID),
		Publication:      s.Hub.Publication(link.ID),
	}
}
