package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moderation-llm/gateway-service/internal/apikey"
	"github.com/moderation-llm/gateway-service/internal/config"
	"github.com/moderation-llm/gateway-service/internal/server"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to Postgres
	dbPool, err := pgxpool.New(context.Background(), cfg.PostgresURL)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	defer dbPool.Close()

	// Test connection
	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("Postgres ping failed: %v", err)
	}
	log.Println("✓ Connected to Postgres")

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Redis ping failed: %v", err)
	}
	log.Println("✓ Connected to Redis")

	// Initialize API key store
	store, err := apikey.NewStore(dbPool, redisClient, int64(cfg.KeyCacheTTL.Seconds()))
	if err != nil {
		log.Fatalf("Failed to initialize API key store: %v", err)
	}
	log.Println("✓ API key store initialized")

	// Create rate limiter
	limiter := apikey.NewRateLimiter(redisClient)

	// Create server
	srv := server.NewServer(
		cfg.UpstreamURL,
		cfg.GatewayRequestTimeout,
		store,
		limiter,
		cfg.AdminSecret,
		cfg.CORSAllowedOrigins,
	)

	addr := fmt.Sprintf(":%d", cfg.GatewayPort)
	log.Printf("Starting gateway on %s (upstream: %s)\n", addr, cfg.UpstreamURL)
	if err := srv.ListenAndServe(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
