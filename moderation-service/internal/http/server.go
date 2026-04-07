package http

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"moderation-llm/moderation-service/internal/moderation"
)

type Server struct {
	engine  *moderation.Engine
	logger  *slog.Logger
	router  chi.Router
	timeout time.Duration
}

func NewServer(engine *moderation.Engine, logger *slog.Logger, timeout time.Duration) *Server {
	s := &Server{
		engine:  engine,
		logger:  logger,
		timeout: timeout,
	}
	s.router = s.buildRouter()
	return s
}

func (s *Server) Router() chi.Router {
	return s.router
}

func (s *Server) buildRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(s.timeout))

	r.Get("/healthz", s.health)
	r.Post("/moderate", s.moderate)
	r.Post("/moderate/batch", s.moderateBatch)

	return r
}
