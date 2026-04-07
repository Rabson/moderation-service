package gateway

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"moderation-llm/api-service/internal/config"
)

type Server struct {
	cfg    config.Config
	logger *slog.Logger
	router chi.Router
	client *http.Client
}

func NewServer(cfg config.Config, logger *slog.Logger) *Server {
	s := &Server{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{Timeout: cfg.RequestTimeout},
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
	r.Use(s.rateLimitMiddleware())

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Post("/moderate", s.proxy("/moderate"))
	r.Post("/moderate/batch", s.proxy("/moderate/batch"))
	r.Post("/transcribe", s.proxy("/transcribe"))
	r.Post("/transcribe/audio", s.proxy("/transcribe/audio"))
	r.Post("/translate", s.proxy("/translate"))

	return r
}

func (s *Server) proxy(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		maxRequestBody := int64(2 << 20)
		if path == "/transcribe/audio" {
			maxRequestBody = 20 << 20
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody))
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		target := strings.TrimSuffix(s.cfg.ModerationServiceURL, "/") + path
		ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
		if err != nil {
			http.Error(w, "failed to create upstream request", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-ID", middleware.GetReqID(r.Context()))

		started := time.Now()
		resp, err := s.client.Do(req)
		if err != nil {
			s.logger.Error("upstream call failed", "path", path, "error", err)
			http.Error(w, "moderation service unavailable", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		if err != nil {
			http.Error(w, "failed to read upstream response", http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(respBody)

		s.logger.Info("proxy request complete",
			"path", path,
			"status", resp.StatusCode,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	}
}
