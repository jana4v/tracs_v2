package ingest

import (
	"context"
	"log/slog"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ParamRange describes the value space for a single simulated parameter.
// Populated by the gateway from the tm_mnemonics MongoDB collection.
type ParamRange struct {
	Type   string   `json:"type"`   // "ANALOG" or "DIGITAL"
	Min    float64  `json:"min"`    // ANALOG: lower bound
	Max    float64  `json:"max"`    // ANALOG: upper bound
	States []string `json:"states"` // DIGITAL: possible state strings
}

// SimRunConfig is the payload sent by the gateway when starting a stream simulator.
type SimRunConfig struct {
	IntervalMs int                   `json:"interval_ms"`
	Params     map[string]ParamRange `json:"params"` // id → range (key is paramID for TM, param name for SCOS/DTM/UDTM)
}

// randomValue generates a single random value string for the given ParamRange.
//
//   - DIGITAL: picks a random element from States.
//   - ANALOG:  uniform float64 in [Min, Max], formatted to 4 decimal places.
func randomValue(pr ParamRange) string {
	t := strings.ToUpper(pr.Type)
	if t == "DIGITAL" || t == "BINARY" || len(pr.States) > 0 {
		if len(pr.States) == 0 {
			return "0"
		}
		return pr.States[rand.Intn(len(pr.States))]
	}
	// ANALOG
	span := pr.Max - pr.Min
	if span <= 0 {
		return strconv.FormatFloat(pr.Min, 'f', 4, 64)
	}
	v := pr.Min + rand.Float64()*span
	return strconv.FormatFloat(v, 'f', 4, 64)
}

// ─── SimRunner ────────────────────────────────────────────────────────────────

// SimRunner generates random telemetry values for a single stream on a fixed
// interval and injects them into the StreamStore.
// The values flow through StreamDispatcher → UNIFIED_TM_MAP → NATS tm_map.
type SimRunner struct {
	streamID string
	store    *StreamStore
	cfg      SimRunConfig
	logger   *slog.Logger
	cancel   context.CancelFunc
}

func newSimRunner(streamID string, store *StreamStore, cfg SimRunConfig, logger *slog.Logger) *SimRunner {
	return &SimRunner{
		streamID: streamID,
		store:    store,
		cfg:      cfg,
		logger:   logger,
	}
}

// start spawns the background goroutine. parent is the service-level context;
// the runner's own cancel is layered on top so Stop() can kill it independently.
func (r *SimRunner) start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	r.cancel = cancel
	go r.loop(ctx)
}

func (r *SimRunner) stop() {
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *SimRunner) loop(ctx context.Context) {
	ms := r.cfg.IntervalMs
	if ms <= 0 {
		ms = 1000
	}

	buf := r.store.GetOrCreate(StreamMeta{
		ID:        r.streamID,
		ChainType: "simulated",
		ChainName: r.streamID,
	})

	ticker := time.NewTicker(time.Duration(ms) * time.Millisecond)
	defer ticker.Stop()

	r.logger.Info("sim runner started",
		"stream", r.streamID,
		"params", len(r.cfg.Params),
		"interval_ms", ms,
	)

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("sim runner stopped", "stream", r.streamID)
			return
		case <-ticker.C:
			values := make(map[string]string, len(r.cfg.Params))
			for key, pr := range r.cfg.Params {
				values[key] = randomValue(pr)
			}
			buf.Update(values)
			r.store.Notify()
		}
	}
}

// ─── SimManager ───────────────────────────────────────────────────────────────

// SimManager keeps track of all active SimRunners, at most one per stream.
type SimManager struct {
	mu      sync.RWMutex
	runners map[string]*SimRunner
	store   *StreamStore
	logger  *slog.Logger
}

// NewSimManager creates an empty SimManager.
func NewSimManager(store *StreamStore, logger *slog.Logger) *SimManager {
	return &SimManager{
		runners: make(map[string]*SimRunner),
		store:   store,
		logger:  logger,
	}
}

// Start starts (or restarts) the random simulator for streamID using cfg.
// parent should be the service-level context so all runners stop on shutdown.
func (m *SimManager) Start(parent context.Context, streamID string, cfg SimRunConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.runners[streamID]; ok {
		r.stop() // stop previous runner before replacing
	}
	r := newSimRunner(streamID, m.store, cfg, m.logger)
	r.start(parent)
	m.runners[streamID] = r
}

// Stop halts the random simulator for streamID (no-op if not running).
func (m *SimManager) Stop(streamID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.runners[streamID]; ok {
		r.stop()
		delete(m.runners, streamID)
	}
}

// Status returns a map of running stream → interval_ms.
func (m *SimManager) Status() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]int, len(m.runners))
	for id, r := range m.runners {
		out[id] = r.cfg.IntervalMs
	}
	return out
}

// IsRunning reports whether a simulator is active for streamID.
func (m *SimManager) IsRunning(streamID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.runners[streamID]
	return ok
}
