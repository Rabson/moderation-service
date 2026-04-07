package server

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strconv"

	"github.com/moderation-llm/gateway-service/internal/apikey"
)

func apikeyMiddleware(store *apikey.Store, limiter *apikey.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "missing X-API-Key header"})
				return
			}

			// Validate key in Postgres/Redis
			apiKey, err := store.Validate(r.Context(), key)
			if err != nil {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "invalid or inactive API key"})
				return
			}

			// Check rate limit
			allowed, remaining, _ := limiter.Allow(r.Context(), apiKey.ID, apiKey.RequestsPerMinute)
			if !allowed {
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(apiKey.RequestsPerMinute))
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
				jsonResponse(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
				return
			}

			// Add rate limit headers to response
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(apiKey.RequestsPerMinute))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			// Store in context for downstream handlers
			ctx := context.WithValue(r.Context(), "api_key_id", apiKey.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func adminAuthMiddleware(adminSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			secret := r.Header.Get("X-Admin-Secret")
			if secret == "" {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "missing X-Admin-Secret header"})
				return
			}

			// Constant-time compare to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(secret), []byte(adminSecret)) != 1 {
				jsonResponse(w, http.StatusForbidden, map[string]string{"error": "invalid admin secret"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
