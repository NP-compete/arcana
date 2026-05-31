package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// DistributedRateLimiter uses a Redis sorted-set sliding window to enforce
// per-key rate limits across multiple API replicas. When Redis is unavailable
// it falls back to the in-process token-bucket RateLimiter so the service
// remains operational (fail-open for availability).
type DistributedRateLimiter struct {
	client *redis.Client
	// fallback to local limiter if Redis unavailable
	local *RateLimiter
}

// NewDistributedRateLimiter creates a Redis-backed rate limiter.
// If the REDIS_URL environment variable is unset or Redis is unreachable,
// the limiter falls back to the in-memory RateLimiter.
func NewDistributedRateLimiter() *DistributedRateLimiter {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://redis.arcana.svc.cluster.local:6379"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("WARNING: Redis URL parse error, falling back to local rate limiter: %v", err)
		return &DistributedRateLimiter{local: NewRateLimiter()}
	}

	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("WARNING: Redis unreachable, falling back to local rate limiter: %v", err)
		return &DistributedRateLimiter{local: NewRateLimiter()}
	}

	return &DistributedRateLimiter{client: client}
}

// Allow checks if a request is allowed using a Redis sliding window algorithm.
// Falls back to local in-memory rate limiter if Redis is unavailable.
func (rl *DistributedRateLimiter) Allow(key string, ratePerSecond int) bool {
	if rl.client == nil {
		return rl.local.Allow(key, ratePerSecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	now := time.Now()
	windowKey := fmt.Sprintf("rl:%s", key)
	windowStart := now.Add(-1 * time.Second)

	pipe := rl.client.Pipeline()

	// Remove expired entries outside the 1-second sliding window
	pipe.ZRemRangeByScore(ctx, windowKey, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count current window entries
	countCmd := pipe.ZCard(ctx, windowKey)

	_, err := pipe.Exec(ctx)
	if err != nil {
		// Redis error -- fail open for availability
		return true
	}

	count := countCmd.Val()
	if count >= int64(ratePerSecond) {
		return false
	}

	// Add this request to the sorted set
	member := redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	}
	pipe2 := rl.client.Pipeline()
	pipe2.ZAdd(ctx, windowKey, member)
	pipe2.Expire(ctx, windowKey, 2*time.Second)
	_, _ = pipe2.Exec(ctx)

	return true
}
