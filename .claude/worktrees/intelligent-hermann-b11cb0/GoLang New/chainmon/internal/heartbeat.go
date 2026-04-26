package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/redis/go-redis/v9"
)

// HeartbeatPublisher publishes heartbeat payloads to Redis pub/sub channels.
type HeartbeatPublisher struct {
	rdb    *redis.Client
	logger *slog.Logger
}

// NewHeartbeatPublisher creates a new HeartbeatPublisher.
func NewHeartbeatPublisher(rdb *redis.Client, logger *slog.Logger) *HeartbeatPublisher {
	return &HeartbeatPublisher{
		rdb:    rdb,
		logger: logger.With("component", "heartbeat-publisher"),
	}
}

// Publish serializes the heartbeat payload to JSON and publishes it to the
// specified Redis pub/sub channel.
func (p *HeartbeatPublisher) Publish(ctx context.Context, channel string, payload models.HeartbeatPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal heartbeat payload: %w", err)
	}

	if err := p.rdb.Publish(ctx, channel, string(data)).Err(); err != nil {
		return fmt.Errorf("publish to %s: %w", channel, err)
	}

	return nil
}
