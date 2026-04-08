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
	LLMProvider    string
	LLMBaseURL     string
	LLMModel       string
	LLMAPIKey      string
	CacheTTL       time.Duration
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	PostgresDSN    string
	KafkaEnabled   bool
	KafkaBrokers   []string
	KafkaTopic     string
}

func Load() (Config, error) {
	cfg := Config{
		Port:           getEnv("MODERATION_PORT", "8081"),
		RequestTimeout: getDurationEnv("MODERATION_REQUEST_TIMEOUT", 8*time.Second),
		LLMTimeout:     getDurationEnv("LLM_TIMEOUT", 6*time.Second),
		LLMProvider:    strings.ToLower(strings.TrimSpace(getEnv("LLM_PROVIDER", "ollama"))),
		LLMBaseURL:     strings.TrimSuffix(strings.TrimSpace(getEnv("LLM_BASE_URL", getEnv("OLLAMA_BASE_URL", ""))), "/"),
		LLMModel:       strings.TrimSpace(getEnv("LLM_MODEL", getEnv("OLLAMA_MODEL", ""))),
		LLMAPIKey:      strings.TrimSpace(getEnv("LLM_API_KEY", "")),
		CacheTTL:       getDurationEnv("CACHE_TTL", 10*time.Minute),
		RedisAddr:      getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		RedisDB:        getIntEnv("REDIS_DB", 0),
		PostgresDSN:    getEnv("POSTGRES_DSN", "postgres://moderation:moderation@postgres:5432/moderation?sslmode=disable"),
		KafkaEnabled:   getBoolEnv("KAFKA_ENABLED", false),
		KafkaBrokers:   splitCSV(getEnv("KAFKA_BROKERS", "kafka:9092")),
		KafkaTopic:     getEnv("KAFKA_TOPIC", "moderation-events"),
	}

	if cfg.PostgresDSN == "" {
		return Config{}, fmt.Errorf("POSTGRES_DSN is required")
	}
	if cfg.LLMProvider == "" {
		cfg.LLMProvider = "ollama"
	}

	switch cfg.LLMProvider {
	case "ollama":
		if cfg.LLMBaseURL == "" {
			cfg.LLMBaseURL = "http://ollama:11434"
		}
		if cfg.LLMModel == "" {
			cfg.LLMModel = "gemma:2b"
		}
	case "openai":
		if cfg.LLMBaseURL == "" {
			cfg.LLMBaseURL = strings.TrimSuffix(strings.TrimSpace(getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1")), "/")
		}
		if cfg.LLMModel == "" {
			cfg.LLMModel = "gpt-4o-mini"
		}
		if cfg.LLMAPIKey == "" {
			cfg.LLMAPIKey = strings.TrimSpace(getEnv("OPENAI_API_KEY", ""))
		}
		if cfg.LLMAPIKey == "" {
			return Config{}, fmt.Errorf("LLM_API_KEY or OPENAI_API_KEY is required when LLM_PROVIDER=openai")
		}
	case "google", "google-genai", "genai", "gemini":
		cfg.LLMProvider = "google"
		if cfg.LLMBaseURL == "" {
			cfg.LLMBaseURL = strings.TrimSuffix(strings.TrimSpace(getEnv("GOOGLE_GENAI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta")), "/")
		}
		if cfg.LLMModel == "" {
			cfg.LLMModel = "gemini-1.5-flash"
		}
		if cfg.LLMAPIKey == "" {
			cfg.LLMAPIKey = strings.TrimSpace(getEnv("GOOGLE_API_KEY", ""))
		}
		if cfg.LLMAPIKey == "" {
			return Config{}, fmt.Errorf("LLM_API_KEY or GOOGLE_API_KEY is required when LLM_PROVIDER=google")
		}
	default:
		return Config{}, fmt.Errorf("unsupported LLM_PROVIDER: %s", cfg.LLMProvider)
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
