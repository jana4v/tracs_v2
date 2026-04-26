package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/internal/config"
	"github.com/mainframe/tm-system/internal/models"
)

// ChainSubscriber manages a single chain's WebSocket ingestion.
//
// StreamStore writes feed StreamDispatcher, which writes to UNIFIED_TM_MAP and
// publishes NATS subjects.
type ChainSubscriber struct {
	chain            config.ChainConfig
	rdb              *redis.Client
	store            *StreamStore
	buf              *StreamBuffer // pre-fetched buffer for this chain
	logger           *slog.Logger
	lastDataReceived time.Time
}

// NewChainSubscriber creates a subscriber for the given chain configuration.
func NewChainSubscriber(chain config.ChainConfig, rdb *redis.Client, store *StreamStore, logger *slog.Logger) *ChainSubscriber {
	buf := store.GetOrCreate(StreamMeta{
		ID:        chain.Name,
		ChainType: chain.Type,
		ChainName: chain.Name,
	})
	return &ChainSubscriber{
		chain:  chain,
		rdb:    rdb,
		store:  store,
		buf:    buf,
		logger: logger,
	}
}

// Run starts the WebSocket subscriber with auto-reconnect. It blocks until
// the context is cancelled. Intended to be called in a goroutine.
func (cs *ChainSubscriber) Run(ctx context.Context) error {
	cs.clearChainRedis(ctx)

	wsURL := fmt.Sprintf("ws://%s:%d/ws", cs.chain.Host, cs.chain.Port)

	ws := &clients.WSSubscriber{
		URL:           wsURL,
		ChainName:     cs.chain.Name,
		Logger:        cs.logger,
		SendSubscribe: cs.chain.Type == "TM",

		OnConnect: func(ctx context.Context) {
			cs.logger.Info("connected, setting heartbeat to CONNECTED")
			cs.setHeartbeat(ctx, models.StatusConnected, 0)
		},

		OnDisconnect: func(ctx context.Context, err error) {
			cs.logger.Warn("disconnected, setting heartbeat to CONNECTION_FAILED", "error", err)
			cs.setHeartbeat(ctx, models.StatusConnectionFailed, 0)
			cs.clearChainRedis(ctx)
		},

		OnMessage: func(ctx context.Context, msg []byte) error {
			return cs.handleMessage(ctx, msg)
		},
	}

	return ws.Run(ctx)
}

// handleMessage dispatches the raw WebSocket message to the appropriate parser.
func (cs *ChainSubscriber) handleMessage(ctx context.Context, msg []byte) error {
	switch cs.chain.Type {
	case "TM":
		return cs.handleTMMessage(ctx, msg)
	case "SCOS":
		return cs.handleSCOSMessage(ctx, msg)
	default:
		return fmt.Errorf("unknown chain type: %s", cs.chain.Type)
	}
}

// handleTMMessage parses a TmPacket and writes to StreamStore.
// The StreamStore key is paramID only.
func (cs *ChainSubscriber) handleTMMessage(ctx context.Context, msg []byte) error {
	paramID, param, value, errDesc, err := ParseTMPacket(msg)
	if err != nil {
		return fmt.Errorf("parse TM packet: %w", err)
	}

	if param == "" {
		return nil
	}

	if strings.Contains(errDesc, "break") {
		cs.logger.Debug("data break detected", "param", param)
		cs.setHeartbeat(ctx, models.StatusDataBreak, 0)
		cs.clearChainRedis(ctx)
		return nil
	}

	// Keep heartbeat status current for chain-status publishing.
	cs.setHeartbeat(ctx, models.StatusOK, 2*time.Second)

	// StreamStore key is paramID only.
	cs.buf.Update(map[string]string{paramID: value})
	cs.store.Notify()

	cs.lastDataReceived = time.Now()
	return nil
}

// handleSCOSMessage parses a ScosPkt and writes all params to StreamStore.
// SCOS chains have no per-param numeric ID; the StreamStore key is the param name only.
func (cs *ChainSubscriber) handleSCOSMessage(ctx context.Context, msg []byte) error {
	params, errDesc, err := ParseSCOSPacket(msg)
	if err != nil {
		return fmt.Errorf("parse SCOS packet: %w", err)
	}

	if strings.Contains(errDesc, "break") || len(params) == 0 {
		cs.logger.Debug("data break or empty packet detected")
		cs.setHeartbeat(ctx, models.StatusDataBreak, 0)
		cs.clearChainRedis(ctx)
		return nil
	}

	bufData := make(map[string]string, len(params))
	for _, p := range params {
		// StreamStore key: param name only (no chain prefix — buffer is per-stream already).
		bufData[p.Param] = p.Value
	}

	// Keep heartbeat status current for chain-status publishing.
	cs.setHeartbeat(ctx, models.StatusOK, 2*time.Second)

	// StreamStore write.
	cs.buf.Update(bufData)
	cs.store.Notify()

	cs.lastDataReceived = time.Now()
	return nil
}

// setHeartbeat sets the heartbeat status key in Redis. ttl=0 means no expiry.
func (cs *ChainSubscriber) setHeartbeat(ctx context.Context, status string, ttl time.Duration) {
	if err := cs.rdb.Set(ctx, models.HeartbeatStatusKey(cs.chain.Name), status, ttl).Err(); err != nil {
		cs.logger.Error("failed to set heartbeat", "status", status, "error", err)
	}
}

// clearChainRedis deletes the chain-specific MAP and PKT keys from Redis.
func (cs *ChainSubscriber) clearChainRedis(ctx context.Context) {
	pipe := cs.rdb.Pipeline()
	pipe.Del(ctx, models.ChainMapKeyByName(cs.chain.Name))
	pipe.Del(ctx, models.ChainPktKey(cs.chain.Name))
	if _, err := pipe.Exec(ctx); err != nil {
		cs.logger.Error("failed to clear chain Redis keys", "error", err)
	}
}
