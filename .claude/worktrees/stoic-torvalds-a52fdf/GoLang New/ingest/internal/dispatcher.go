package ingest

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/internal/config"
	"github.com/mainframe/tm-system/internal/models"
)

const (
	dispatchInterval = 100 * time.Millisecond
	statusInterval   = 1 * time.Second
)

// StreamDispatcher is the single NATS publishing point for the ingest service.
//
// It runs three loops on independent ticks:
//
//  1. 100 ms (or on notify) — delta: writes changed paramId-keyed values to
//     UNIFIED_TM_MAP and publishes both {prefix}.tm_map (unified) and
//     {prefix}.tm_map.<chain> (per-chain)
//
//  2. 1 s — status: reads Redis status maps and publishes
//     {prefix}.limit-failures, {prefix}.chain-mismatches, {prefix}.chain-status
//
// Full snapshots are no longer broadcast on a timer. Instead, clients send a
// NATS request to {prefix}.tm_map/full once on startup and receive the current
// snapshot as the reply. After that they subscribe to {prefix}.tm_map for
// live delta updates.
type StreamDispatcher struct {
	store        *StreamStore
	rdb          *redis.Client
	nats         *clients.NATSClient // may be nil (NATS disabled)
	cfg          config.NATSConfig
	logger       *slog.Logger
	statusChains []string // chain names for heartbeat status
}

// NewStreamDispatcher creates a dispatcher. nats may be nil.
// statusChains is the list of chain names whose heartbeats are published
// (e.g. ["TM1", "TM2", "SMON1"]).
func NewStreamDispatcher(store *StreamStore, rdb *redis.Client, nats *clients.NATSClient, cfg config.NATSConfig, logger *slog.Logger, statusChains []string) *StreamDispatcher {
	return &StreamDispatcher{
		store:        store,
		rdb:          rdb,
		nats:         nats,
		cfg:          cfg,
		logger:       logger,
		statusChains: statusChains,
	}
}

// Run starts the dispatch loop and blocks until ctx is cancelled.
func (d *StreamDispatcher) Run(ctx context.Context) {
	dataTicker := time.NewTicker(dispatchInterval)
	statusTicker := time.NewTicker(statusInterval)
	defer dataTicker.Stop()
	defer statusTicker.Stop()

	// Seed UNIFIED_TM_MAP with the current snapshot on startup.
	d.seedUnifiedMap(ctx)

	// Register the full-map request-reply handler so clients can request the
	// current snapshot on demand instead of waiting for a broadcast.
	d.ServeFullMapRequests(ctx)

	d.logger.Info("stream dispatcher started",
		"data_interval", dispatchInterval,
		"status_chains", len(d.statusChains),
	)

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("stream dispatcher stopped")
			return
		case <-dataTicker.C:
			d.dispatchDelta(ctx)
		case <-d.store.NotifyCh():
			d.dispatchDelta(ctx)
		case <-statusTicker.C:
			d.publishStatus(ctx)
		}
	}
}

// -------------------------------------------------------------------
// TM data paths
// -------------------------------------------------------------------

// dispatchDelta collects changed values from all buffers, writes them to
// UNIFIED_TM_MAP, and publishes unified plus per-chain NATS deltas.
func (d *StreamDispatcher) dispatchDelta(ctx context.Context) {
	allChanged := make(map[string]string)
	changedByChain := make(map[string]map[string]string)

	for _, buf := range d.store.All() {
		changed, _ := buf.Delta()
		if len(changed) == 0 {
			continue
		}
		if _, ok := changedByChain[buf.Meta.ID]; !ok {
			changedByChain[buf.Meta.ID] = make(map[string]string)
		}
		for k, v := range changed {
			allChanged[k] = v
			changedByChain[buf.Meta.ID][k] = v
		}
		buf.CommitDelta()
	}

	if len(allChanged) == 0 {
		return
	}

	// Write to Redis UNIFIED_TM_MAP.
	pipe := d.rdb.Pipeline()
	for k, v := range allChanged {
		pipe.HSet(ctx, models.UnifiedTMMap, k, v)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		d.logger.Warn("dispatcher: Redis pipeline failed", "error", err)
	}

	// Publish NATS unified + per-chain deltas using paramId (or source key) as-is.
	if d.nats != nil {
		if data, err := json.Marshal(numericPayload(allChanged)); err == nil {
			d.nats.Publish(d.subject("tm_map"), data)
		}
		for chain, payloadData := range changedByChain {
			if len(payloadData) == 0 {
				continue
			}
			if chainData, err := json.Marshal(numericPayload(payloadData)); err == nil {
				d.nats.Publish(d.subject("tm_map."+chain), chainData)
			}
		}
	}
}

// seedUnifiedMap writes all current values to UNIFIED_TM_MAP in Redis on
// startup so the hash is populated before any delta updates arrive.
func (d *StreamDispatcher) seedUnifiedMap(ctx context.Context) {
	full := make(map[string]string)
	for _, buf := range d.store.All() {
		for k, v := range buf.Snapshot() {
			full[k] = v
		}
	}
	if len(full) == 0 {
		return
	}
	pipe := d.rdb.Pipeline()
	for k, v := range full {
		pipe.HSet(ctx, models.UnifiedTMMap, k, v)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		d.logger.Warn("dispatcher: seed Redis write failed", "error", err)
	}
	d.logger.Debug("dispatcher: UNIFIED_TM_MAP seeded", "params", len(full))
}

// ServeFullMapRequests subscribes to full-map request subjects and replies with
// either a unified snapshot or a per-chain snapshot.
//
// Supported request subjects:
//   - {prefix}.tm_map/full          (unified)
//   - {prefix}.tm_map/full.<chain>  (per-chain, e.g. TM1, TM2, SMON1)
//
// Clients can request full data once on startup, then subscribe to live deltas.
func (d *StreamDispatcher) ServeFullMapRequests(ctx context.Context) {
	if d.nats == nil {
		return
	}
	unifiedSubject := d.subject("tm_map/full")
	perChainSubject := d.subject("tm_map/full.*")

	cancelUnified, err := d.nats.SubscribeRequests(unifiedSubject, func(reply string, _ []byte) {
		full := make(map[string]string)
		for _, buf := range d.store.All() {
			for k, v := range buf.Snapshot() {
				full[k] = v
			}
		}
		data, err := json.Marshal(numericPayload(full))
		if err != nil {
			d.logger.Warn("dispatcher: unified full map request marshal failed", "error", err)
			return
		}
		d.nats.Reply(reply, data)
		d.logger.Debug("dispatcher: unified full map request served", "params", len(full))
	})
	if err != nil {
		d.logger.Warn("dispatcher: failed to register full map request handler", "error", err)
		return
	}

	cancelPerChain, err := d.nats.SubscribeRequestsWithSubject(perChainSubject, func(requestSubject string, reply string, _ []byte) {
		chain := strings.TrimPrefix(requestSubject, d.subject("tm_map/full."))
		if chain == "" {
			return
		}

		full := d.fullMapForChain(chain)
		data, err := json.Marshal(numericPayload(full))
		if err != nil {
			d.logger.Warn("dispatcher: per-chain full map request marshal failed", "chain", chain, "error", err)
			return
		}
		d.nats.Reply(reply, data)
		d.logger.Debug("dispatcher: per-chain full map request served", "chain", chain, "params", len(full))
	})
	if err != nil {
		cancelUnified()
		d.logger.Warn("dispatcher: failed to register per-chain full map request handler", "error", err)
		return
	}

	// Unsubscribe when the context is cancelled.
	go func() {
		<-ctx.Done()
		cancelUnified()
		cancelPerChain()
	}()
}

func (d *StreamDispatcher) fullMapForChain(chain string) map[string]string {
	full := make(map[string]string)
	for _, buf := range d.store.All() {
		if !strings.EqualFold(buf.Meta.ID, chain) {
			continue
		}
		for k, v := range buf.Snapshot() {
			full[k] = v
		}
	}
	return full
}

// -------------------------------------------------------------------
// Status data path (limit failures, mismatches, chain heartbeats)
// -------------------------------------------------------------------

// publishStatus reads status maps from Redis and publishes to NATS.
func (d *StreamDispatcher) publishStatus(ctx context.Context) {
	if d.nats == nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)

	// Limit failures.
	if limitData, err := d.rdb.HGetAll(ctx, models.TMLimitFailuresMap).Result(); err == nil {
		if payload, err := json.Marshal(map[string]interface{}{"timestamp": now, "data": limitData}); err == nil {
			d.nats.Publish(d.subject("limit-failures"), payload)
		}
	}

	// Chain mismatches.
	if mismatchData, err := d.rdb.HGetAll(ctx, models.TMChainMismatchesMap).Result(); err == nil {
		if payload, err := json.Marshal(map[string]interface{}{"timestamp": now, "data": mismatchData}); err == nil {
			d.nats.Publish(d.subject("chain-mismatches"), payload)
		}
	}

	// Chain heartbeat status.
	if len(d.statusChains) > 0 {
		type chainEntry struct {
			Chain  string `json:"chain"`
			Status string `json:"status"`
		}
		entries := make([]chainEntry, 0, len(d.statusChains))
		for _, name := range d.statusChains {
			val, err := d.rdb.Get(ctx, models.HeartbeatStatusKey(name)).Result()
			status := "inactive"
			if err == nil && val != "" {
				status = "active"
			}
			entries = append(entries, chainEntry{Chain: name, Status: status})
		}
		if payload, err := json.Marshal(map[string]interface{}{"timestamp": now, "chains": entries}); err == nil {
			d.nats.Publish(d.subject("chain-status"), payload)
		}
	}
}

// -------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------

// numericPayload converts string values to float64 where possible so
// NATS consumers receive JSON numbers for analog telemetry.
func numericPayload(data map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(data))
	for k, v := range data {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			out[k] = f
		} else {
			out[k] = v
		}
	}
	return out
}

// subject returns the fully-qualified NATS subject for the given suffix.
func (d *StreamDispatcher) subject(suffix string) string {
	if d.cfg.SubjectPrefix == "" {
		return suffix
	}
	return d.cfg.SubjectPrefix + "." + suffix
}
