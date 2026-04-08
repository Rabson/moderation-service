package server

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/moderation-llm/gateway-service/internal/apikey"
)

type Server struct {
	mux         chi.Router
	upstreamURL string
	timeout     time.Duration
	store       *apikey.Store
	limiter     *apikey.RateLimiter
	adminSecret string
	corsOrigins []string
}

func NewServer(upstreamURL string, timeout time.Duration, store *apikey.Store, limiter *apikey.RateLimiter, adminSecret string, corsOrigins []string) *Server {
	s := &Server{
		mux:         chi.NewRouter(),
		upstreamURL: upstreamURL,
		timeout:     timeout,
		store:       store,
		limiter:     limiter,
		adminSecret: adminSecret,
		corsOrigins: corsOrigins,
	}

	s.buildRouter()
	return s
}

func (s *Server) buildRouter() {
	s.mux.Use(middleware.RequestID)
	s.mux.Use(middleware.RealIP)
	s.mux.Use(middleware.Logger)
	s.mux.Use(middleware.Timeout(s.timeout))
	s.mux.Use(cors.Handler(cors.Options{
		AllowedOrigins: s.corsOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-API-Key", "X-Admin-Secret"},
		ExposedHeaders: []string{"X-RateLimit-Limit", "X-RateLimit-Remaining"},
		MaxAge:         300,
	}))

	// Public health check
	s.mux.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Protected API endpoints
	s.mux.Group(func(r chi.Router) {
		r.Use(apikeyMiddleware(s.store, s.limiter))
		r.Post("/moderate", s.proxy)
		r.Post("/moderate/batch", s.proxy)
		r.Post("/transcribe", s.proxy)
		r.Post("/transcribe/audio", s.proxy)
		r.Post("/translate", s.proxy)
	})

	// Admin endpoints
	s.mux.Route("/admin", func(r chi.Router) {
		r.Use(adminAuthMiddleware(s.adminSecret))
		r.Post("/keys", s.createKey)
		r.Get("/keys", s.listKeys)
		r.Put("/keys/{id}", s.updateKey)
		r.Delete("/keys/{id}", s.deactivateKey)
	})
}

func (s *Server) proxy(w http.ResponseWriter, r *http.Request) {
	targetURL, err := url.Parse(s.upstreamURL)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "invalid upstream URL"})
		return
	}

	// Strip X-API-Key before forwarding
	r.Header.Del("X-API-Key")

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ServeHTTP(w, r)
}

func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

func jsonResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// Admin handlers
func (s *Server) createKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name              string `json:"name"`
		RequestsPerMinute int    `json:"requests_per_minute"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.RequestsPerMinute == 0 {
		req.RequestsPerMinute = 100
	}

	keyInfo, plaintext, err := s.store.Create(r.Context(), req.Name, req.RequestsPerMinute)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"api_key":             plaintext,
		"note":                "Store this key securely. You won't be able to see it again.",
		"id":                  keyInfo.ID,
		"name":                keyInfo.Name,
		"requests_per_minute": keyInfo.RequestsPerMinute,
	})
}

func (s *Server) listKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.store.List(r.Context())
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"keys": keys})
}

func (s *Server) deactivateKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.Deactivate(r.Context(), id); err != nil {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "key deactivated"})
}

func (s *Server) updateKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Name              string `json:"name"`
		RequestsPerMinute int    `json:"requests_per_minute"`
		IsActive          bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.Name == "" || req.RequestsPerMinute <= 0 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "name and requests_per_minute must be provided"})
		return
	}

	key, err := s.store.Update(r.Context(), id, req.Name, req.RequestsPerMinute, req.IsActive)
	if err != nil {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"id":                  key.ID,
		"name":                key.Name,
		"requests_per_minute": key.RequestsPerMinute,
		"is_active":           key.IsActive,
	})
}
