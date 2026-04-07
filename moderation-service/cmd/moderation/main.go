package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"moderation-llm/moderation-service/internal/config"
	transporthttp "moderation-llm/moderation-service/internal/http"
	"moderation-llm/moderation-service/internal/kafka"
	"moderation-llm/moderation-service/internal/llm"
	"moderation-llm/moderation-service/internal/moderation"
	"moderation-llm/moderation-service/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx := context.Background()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("redis ping failed", "error", err)
		os.Exit(1)
	}

	postgres, err := storage.NewPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		logger.Error("postgres init failed", "error", err)
		os.Exit(1)
	}
	defer postgres.Close()

	cache := storage.NewCache(redisClient, cfg.CacheTTL)
	llmClient := llm.NewClient(cfg.OllamaBaseURL, cfg.OllamaModel, cfg.LLMTimeout)
	producer := kafka.NewProducer(cfg.KafkaEnabled, cfg.KafkaBrokers, cfg.KafkaTopic)
	defer func() {
		if err := producer.Close(); err != nil {
			logger.Warn("kafka close failed", "error", err)
		}
	}()

	engine := moderation.NewEngine(
		moderation.NewRuleEngine(),
		llmClient,
		cache,
		postgres,
		producer,
		logger,
		cfg.LLMTimeout,
	)

	server := transporthttp.NewServer(engine, logger, cfg.RequestTimeout)
	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.Router(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("moderation service started", "port", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("moderation service failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown failed", "error", err)
	}
	if err := redisClient.Close(); err != nil {
		logger.Warn("redis close failed", "error", err)
	}
	logger.Info("moderation service stopped")
}
