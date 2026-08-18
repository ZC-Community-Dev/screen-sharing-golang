package httpapi

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"api/internal/links"
	"api/internal/media"
	"api/internal/room"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	Engine             *gin.Engine
	Links              *links.Service
	Hub                *room.Hub
	Media              *media.Manager
	Tickets            *media.TicketStore
	Config             Config
	ticketReservations sync.Map
	Log                *slog.Logger
}

func New(cfg Config, database *sql.DB, logger *slog.Logger) *Server {
	return NewWithFrontend(cfg, database, logger, nil)
}

func NewWithFrontend(
	cfg Config,
	database *sql.DB,
	logger *slog.Logger,
	frontend http.Handler,
) *Server {
	cfg = normalizeMediaConfig(cfg)
	if logger == nil {
		logger = NewLogger()
	}
	s := &Server{
		Links:   links.NewService(database, cfg.Salt),
		Hub:     room.NewHub(),
		Log:     logger,
		Config:  cfg,
		Tickets: media.NewTicketStore(30*time.Second, time.Now),
	}
	s.Media = media.NewManagerWithFactory(func() (*media.Engine, error) {
		return media.NewEngine(media.EngineConfig{UDPPort: cfg.MediaUDPPort, PublicIP: cfg.MediaPublicIP})
	}, media.Limits{MaxRooms: cfg.MediaMaxRooms, MaxViewersPerRoom: cfg.MediaMaxViewersPerRoom})
	s.Media.SetCallbacks(func(linkID string, state media.State) {
		s.Hub.BroadcastState(linkID, string(state))
		_ = s.Links.SetState(linkID, string(state))
	}, func(linkID, owner, mediaID, role, state string) {
		s.Hub.SendMediaState(linkID, owner, mediaID, role, state)
	})
	s.Media.SetPublicationCallback(func(linkID string, publication media.Publication) {
		s.Hub.BroadcastPublication(linkID, room.Publication{
			ID: publication.ID, Transport: string(publication.Transport), State: string(publication.State),
			Generation: publication.Generation,
		})
	})
	r := gin.New()
	r.Use(gin.Recovery(), RequestLog(logger))
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "X-Room-Session-ID"},
		AllowCredentials: false,
	}))
	v1 := r.Group("/api/v1")
	v1.POST("/links", s.createLink)
	v1.GET("/links/:id", s.getLink)
	v1.POST("/links/:id/presenter-sessions", s.claimPresenter)
	v1.POST("/links/:id/viewer-sessions", s.joinViewer)
	v1.POST("/links/:id/share/start", s.startShare)
	v1.POST("/links/:id/share/stop", s.stopShare)
	v1.GET("/links/:id/events", s.eventsV1)
	v2 := r.Group("/api/v2")
	v2.GET("/media/config", s.getMediaConfig)
	v2.POST("/links", s.createLink)
	v2.GET("/links/:id", s.getLink)
	v2.POST("/links/:id/presenter-sessions", s.claimPresenter)
	v2.POST("/links/:id/viewer-sessions", s.joinViewer)
	v2.POST("/links/:id/share/start", s.prepareShareV2)
	v2.POST("/links/:id/share/stop", s.stopShare)
	v2.GET("/links/:id/events", s.eventsV2)
	v2.POST("/links/:id/media/publisher", s.createPublisher)
	v2.POST("/links/:id/media/subscribers", s.createSubscriber)
	v2.DELETE("/links/:id/media/subscribers/:mediaSessionId", s.deleteSubscriber)
	v2.POST("/links/:id/media/websocket-tickets", s.createMediaWebSocketTicket)
	v2.GET("/links/:id/media/websocket", s.mediaWebSocket)
	if frontend != nil {
		r.NoRoute(func(c *gin.Context) {
			requestPath := c.Request.URL.Path
			if requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") {
				writeError(c, http.StatusNotFound, CodeRouteNotFound, "route not found")
				return
			}
			if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
				c.Status(http.StatusNotFound)
				return
			}
			frontend.ServeHTTP(c.Writer, c.Request)
		})
	}
	s.Engine = r
	return s
}

func (s *Server) Close() error {
	s.ticketReservations.Range(func(key, value any) bool {
		reservation := value.(*ticketReservation)
		if _, loaded := s.ticketReservations.LoadAndDelete(key); loaded {
			close(reservation.cancel)
			reservation.cleanup()
		}
		return true
	})
	s.Tickets.Close()
	return s.Media.Close()
}
