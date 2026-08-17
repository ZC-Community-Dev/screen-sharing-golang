package httpapi

import (
	"database/sql"
	"log/slog"

	"api/internal/links"
	"api/internal/room"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	Engine *gin.Engine
	Links  *links.Service
	Hub    *room.Hub
	Log    *slog.Logger
}

func New(cfg Config, database *sql.DB, logger *slog.Logger) *Server {
	if logger == nil {
		logger = NewLogger()
	}
	s := &Server{
		Links: links.NewService(database, cfg.Salt),
		Hub:   room.NewHub(),
		Log:   logger,
	}
	r := gin.New()
	r.Use(gin.Recovery(), RequestLog(logger))
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		AllowCredentials: false,
	}))
	v1 := r.Group("/api/v1")
	v1.POST("/links", s.createLink)
	v1.GET("/links/:id", s.getLink)
	v1.POST("/links/:id/presenter-sessions", s.claimPresenter)
	v1.POST("/links/:id/viewer-sessions", s.joinViewer)
	v1.POST("/links/:id/share/start", s.startShare)
	v1.POST("/links/:id/share/stop", s.stopShare)
	v1.GET("/links/:id/events", s.events)
	s.Engine = r
	return s
}
