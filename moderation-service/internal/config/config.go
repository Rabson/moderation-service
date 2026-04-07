package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port           string
	RequestTimeout time.Duration
	LLMTimeout     time.Duration
	CacheTTL       time.Duration
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	PostgresDSN    string
	OllamaBaseURL  string
	OllamaModel    string
	KafkaEnabled   bool
	KafkaBrokers   []string
	KafkaTopic     string
}

func Load() (Config, error) {
	cfg := Config{
		Port:           getEnv("MODERATION_PORT", "8081"),
		RequestTimeout: getDurationEnv("MODERATION_REQUEST_TIMEOUT", 8*time.Second),
		LLMTimeout:     getDurationEnv("LLM_TIMEOUT", 6*time.Second),
		CacheTTL:       getDurationEnv("CACHE_TTL", 10*time.Minute),
		RedisAddr:      getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		RedisDB:        getIntEnv("REDIS_DB", 0),
		PostgresDSN:    getEnv("POSTGRES_DSN", "postgres://moderation:moderation@postgres:5432/moderation?sslmode=disable"),
		OllamaBaseURL:  strings.TrimSuffix(getEnv("OLLAMA_BASE_URL", "http://ollama:11434"), "/"),
		OllamaModel:    getEnv("OLLAMA_MODEL", "gemma:2b"),
		KafkaEnabled:   getBoolEnv("KAFKA_ENABLED", false),
		KafkaBrokers:   splitCSV(getEnv("KAFKA_BROKERS", "kafka:9092")),
		KafkaTopic:     getEnv("KAFKA_TOPIC", "moderation-events"),
	}

	if cfg.PostgresDSN == "" {
		return Config{}, fmt.Errorf("POSTGRES_DSN is required")
	}
	if cfg.OllamaBaseURL == "" {
		return Config{}, fmt.Errorf("OLLAMA_BASE_URL is required")
	}
	if cfg.OllamaModel == "" {
		return Config{}, fmt.Errorf("OLLAMA_MODEL is required")
	}
	if cfg.KafkaEnabled && len(cfg.KafkaBrokers) == 0 {
		return Config{}, fmt.Errorf("KAFKA_BROKERS must be set when KAFKA_ENABLED=true")
	}

	return cfg, nil
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

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
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

func getBoolEnv(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(getEnv(key, "")))
	if v == "" {
		return fallback
	}
	return v == "1" || v == "true" || v == "yes"
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
