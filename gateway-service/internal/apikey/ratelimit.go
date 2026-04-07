package apikey

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

// Allow checks if a request is allowed under the rate limit.
// Returns (allowed, remaining, error).
// If Redis errors, fails open (allowed=true) to avoid blocking traffic.
func (rl *RateLimiter) Allow(ctx context.Context, keyID string, limit int) (bool, int, error) {
	now := time.Now()
	minuteStart := now.Truncate(time.Minute).Unix()
	windowKey := fmt.Sprintf("rl:%s:%d", keyID, minuteStart)

	// Increment counter
	curr, err := rl.client.Incr(ctx, windowKey).Result()
	if err != nil {
		// Redis error: fail open
		return true, limit, err
	}

	// Set expiration on first request of the window
	if curr == 1 {
		_ = rl.client.Expire(ctx, windowKey, time.Minute)
	}

	// Check limit
	allowed := curr <= int64(limit)
	remaining := int(limit) - int(curr)
	if remaining < 0 {
		remaining = 0
	}

	return allowed, remaining, nil
}
