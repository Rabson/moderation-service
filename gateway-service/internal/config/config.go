package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	GatewayPort           int
	UpstreamURL           string
	GatewayRequestTimeout time.Duration
	KeyCacheTTL           time.Duration
	AdminSecret           string
	PostgresURL           string
	RedisURL              string
	CORSAllowedOrigins    []string
}

func Load() (*Config, error) {
	gatewayPort := 8080
	if port := os.Getenv("GATEWAY_PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid GATEWAY_PORT: %w", err)
		}
		gatewayPort = p
	}

	upstreamURL := os.Getenv("UPSTREAM_URL")
	if upstreamURL == "" {
		upstreamURL = "http://api-service:8080"
	}

	gatewayRequestTimeout := 130 * time.Second
	if timeout := os.Getenv("GATEWAY_REQUEST_TIMEOUT"); timeout != "" {
		d, err := time.ParseDuration(timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid GATEWAY_REQUEST_TIMEOUT: %w", err)
		}
		gatewayRequestTimeout = d
	}

	keyCacheTTL := 60 * time.Second
	if ttl := os.Getenv("KEY_CACHE_TTL"); ttl != "" {
		d, err := time.ParseDuration(ttl)
		if err != nil {
			return nil, fmt.Errorf("invalid KEY_CACHE_TTL: %w", err)
		}
		keyCacheTTL = d
	}

	adminSecret := os.Getenv("ADMIN_SECRET")
	if adminSecret == "" {
		return nil, fmt.Errorf("ADMIN_SECRET not set")
	}

	postgresURL := os.Getenv("DATABASE_URL")
	if postgresURL == "" {
		postgresURL = "postgres://postgres:postgres@postgres:5432/moderation?sslmode=disable"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://redis:6379"
	}

	corsAllowedOrigins := splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if len(corsAllowedOrigins) == 0 {
		corsAllowedOrigins = []string{"http://localhost:8088", "http://localhost:8080", "https://localhost:443"}
	}

	return &Config{
		GatewayPort:           gatewayPort,
		UpstreamURL:           upstreamURL,
		GatewayRequestTimeout: gatewayRequestTimeout,
		KeyCacheTTL:           keyCacheTTL,
		AdminSecret:           adminSecret,
		PostgresURL:           postgresURL,
		RedisURL:              redisURL,
		CORSAllowedOrigins:    corsAllowedOrigins,
	}, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
