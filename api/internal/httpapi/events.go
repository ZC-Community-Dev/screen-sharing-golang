package httpapi

import (
	"net/http"

	"api/internal/ids"
	"api/internal/links"
	"api/internal/room"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "" ||
			origin == "http://localhost:4200" ||
			origin == "http://127.0.0.1:4200"
	},
}

func (s *Server) events(c *gin.Context) {
	id := c.Param("id")
	if !ids.ValidPublicID(id) {
		writeError(c, http.StatusBadRequest, CodeLinkInvalid, "invalid link")
		return
	}
	if _, err := s.Links.GetByID(id); err != nil {
		writeError(c, http.StatusNotFound, CodeLinkNotFound, "link not found")
		return
	}
	sessionID := c.Query("sessionId")
	if !s.Hub.HasSession(id, sessionID) {
		writeError(c, http.StatusUnauthorized, CodePresenterUnauthorized, "unknown session")
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	send := make(chan []byte, 64)
	if !s.Hub.Attach(id, sessionID, send) {
		return
	}
	defer func() {
		if s.Hub.Disconnect(id, sessionID) {
			_ = s.Links.SetState(id, links.StateWaiting)
		}
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			out, to, ok := room.RelaySignal(raw, sessionID)
			if !ok {
				continue
			}
			to = room.ResolveSignalTo(to, s.Hub.PresenterID(id))
			if to == "" {
				continue
			}
			s.Hub.SendTo(id, sessionID, to, out)
		}
	}()

	for {
		select {
		case <-done:
			return
		case msg, ok := <-send:
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}
}
