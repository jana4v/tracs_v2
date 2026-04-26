package internal

import (
	"context"
	"log/slog"
	"time"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/redis/go-redis/v9"
)

// ChainMonitor monitors a single chain's heartbeat status and publishes
// heartbeat messages when the chain is active.
type ChainMonitor struct {
	chainName      string
	chainType      string
	timeoutSeconds int
	rdb            *redis.Client
	publisher      *HeartbeatPublisher
	logger         *slog.Logger
}

// NewChainMonitor creates a new per-chain monitor instance.
func NewChainMonitor(name, chainType string, timeoutSeconds int, rdb *redis.Client, publisher *HeartbeatPublisher, logger *slog.Logger) *ChainMonitor {
	return &ChainMonitor{
		chainName:      name,
		chainType:      chainType,
		timeoutSeconds: timeoutSeconds,
		rdb:            rdb,
		publisher:      publisher,
		logger:         logger.With("chain", name, "chainType", chainType),
	}
}

// Run starts the monitoring loop. Every 5 seconds it reads the chain's heartbeat
// status key from Redis. If status is "OK", it publishes a heartbeat with ACTIVE status.
// If status is not OK, no heartbeat is published (absence signals inactive).
func (m *ChainMonitor) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	m.logger.Info("chain monitor started", "timeout_seconds", m.timeoutSeconds)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("chain monitor stopping")
			return
		case <-ticker.C:
			m.check(ctx)
		}
	}
}

// check reads the heartbeat status key and publishes if OK.
func (m *ChainMonitor) check(ctx context.Context) {
	statusKey := models.HeartbeatStatusKey(m.chainName)
	status, err := m.rdb.Get(ctx, statusKey).Result()
	if err != nil {
		if err != redis.Nil {
			m.logger.Warn("failed to read heartbeat status", "key", statusKey, "error", err)
		}
		return
	}

	if status != models.StatusOK {
		m.logger.Debug("chain not OK, skipping heartbeat publish", "status", status)
		return
	}

	// Read last data timestamp for the heartbeat payload
	lastDataKey := models.LastDataTimeKey(m.chainName)
	lastDataTs, err := m.rdb.Get(ctx, lastDataKey).Result()
	if err != nil {
		if err != redis.Nil {
			m.logger.Warn("failed to read last data time", "key", lastDataKey, "error", err)
		}
		lastDataTs = ""
	}

	// Publish heartbeat
	channel := models.HeartbeatChannelKey(m.chainName)
	payload := models.HeartbeatPayload{
		Chain:      m.chainName,
		Status:     models.StatusActive,
		LastDataTs: lastDataTs,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	if err := m.publisher.Publish(ctx, channel, payload); err != nil {
		m.logger.Error("failed to publish heartbeat", "channel", channel, "error", err)
		return
	}

	m.logger.Debug("heartbeat published", "channel", channel, "status", models.StatusActive)
}
