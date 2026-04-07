package gateway

import (
	"net/http"

	"golang.org/x/time/rate"
)

func (s *Server) rateLimitMiddleware() func(http.Handler) http.Handler {
	limiter := rate.NewLimiter(rate.Limit(s.cfg.RateLimitRPS), s.cfg.RateLimitBurst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
