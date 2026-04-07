package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                 string
	ModerationServiceURL string
	RequestTimeout       time.Duration
	RateLimitRPS         int
	RateLimitBurst       int
}

func Load() (Config, error) {
	cfg := Config{
		Port:                 getEnv("API_PORT", "8080"),
		ModerationServiceURL: getEnv("MODERATION_SERVICE_URL", "http://moderation-service:8081"),
		RequestTimeout:       getDurationEnv("API_REQUEST_TIMEOUT", 5*time.Second),
		RateLimitRPS:         getIntEnv("RATE_LIMIT_RPS", 20),
		RateLimitBurst:       getIntEnv("RATE_LIMIT_BURST", 40),
	}

	if cfg.ModerationServiceURL == "" {
		return Config{}, fmt.Errorf("MODERATION_SERVICE_URL is required")
	}
	if cfg.RateLimitRPS <= 0 {
		return Config{}, fmt.Errorf("RATE_LIMIT_RPS must be > 0")
	}
	if cfg.RateLimitBurst <= 0 {
		return Config{}, fmt.Errorf("RATE_LIMIT_BURST must be > 0")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	v := getEnv(key, "")
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	v := getEnv(key, "")
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
