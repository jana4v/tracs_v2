package clients

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates a Redis client with exponential backoff retry on initial connection.
// Satisfies SRS Section 15: fault-tolerant startup with exponential backoff.
func NewRedisClient(ctx context.Context, addr, password string, db int, logger *slog.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	backoff := time.Second
	maxBackoff := 30 * time.Second
	maxRetries := 10

	for i := 0; i < maxRetries; i++ {
		err := client.Ping(ctx).Err()
		if err == nil {
			logger.Info("connected to Redis", "addr", addr)
			return client, nil
		}

		logger.Warn("Redis connection failed, retrying",
			"addr", addr,
			"attempt", i+1,
			"backoff", backoff,
			"error", err,
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}

		backoff = min(backoff*2, maxBackoff)
	}

	return nil, fmt.Errorf("failed to connect to Redis at %s after %d retries", addr, maxRetries)
}
