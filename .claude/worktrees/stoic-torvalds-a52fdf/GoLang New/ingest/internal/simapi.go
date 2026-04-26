package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// SimHandler provides the HTTP simulation API.
//
// Value injection (used by FIXED mode and DTM/UDTM):
//
//	PUT /streams/{streamID}/values          — inject paramId→value map
//
// Backend random generation (used by RANDOM mode):
//
//	POST /streams/{streamID}/sim/start      — start server-side random loop (body: SimRunConfig)
//	POST /streams/{streamID}/sim/stop       — stop server-side random loop
//	GET  /streams/{streamID}/sim/status     — running status for a stream
//	GET  /streams/sim/status                — status of all running simulators
//
// Inspection:
//
//	GET /streams                            — list all registered stream buffers
//	GET /streams/{streamID}                 — get current values for a stream
type SimHandler struct {
	Store   *StreamStore
	Manager *SimManager
	Logger  *slog.Logger
	svcCtx  context.Context // service-level context passed to SimRunner goroutines
}

// NewSimMux creates an http.ServeMux with all simulation API routes registered.
func NewSimMux(store *StreamStore, manager *SimManager, svcCtx context.Context, logger *slog.Logger) *http.ServeMux {
	h := &SimHandler{
		Store:   store,
		Manager: manager,
		Logger:  logger,
		svcCtx:  svcCtx,
	}
	mux := http.NewServeMux()

	// Value injection
	mux.HandleFunc("PUT /streams/{streamID}/values", h.PutValues)

	// Cross-stream value query (used by gateway get-telemetry — avoids Redis round-trip)
	mux.HandleFunc("POST /streams/query", h.QueryValues)

	// Backend random sim control
	mux.HandleFunc("POST /streams/{streamID}/sim/start", h.StartSim)
	mux.HandleFunc("POST /streams/{streamID}/sim/stop", h.StopSim)
	mux.HandleFunc("GET /streams/{streamID}/sim/status", h.GetStreamSimStatus)
	mux.HandleFunc("GET /streams/sim/status", h.GetAllSimStatus)

	// Inspection
	mux.HandleFunc("GET /streams", h.GetStreams)
	mux.HandleFunc("GET /streams/{streamID}", h.GetStream)

	return mux
}

// maxSimBodyBytes caps the request body size for all sim API endpoints (1 MiB).
const maxSimBodyBytes = 1 * 1024 * 1024

// ─── Value injection ──────────────────────────────────────────────────────────

// PutValues handles PUT /streams/{streamID}/values.
// Body: map[string]string — id → value (paramID for TM, param name for SCOS/DTM/UDTM).
func (h *SimHandler) PutValues(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSimBodyBytes)
	streamID := strings.ToUpper(r.PathValue("streamID"))

	var data map[string]string
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	if len(data) == 0 {
		http.Error(w, `{"error":"empty values map"}`, http.StatusBadRequest)
		return
	}

	buf := h.Store.GetOrCreate(StreamMeta{ID: streamID, ChainType: "programmatic", ChainName: streamID})
	buf.Update(data)
	h.Store.Notify()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"stream": streamID, "updated": len(data)})
}

// ─── Cross-stream query ───────────────────────────────────────────────────────

// QueryValues handles POST /streams/query.
// Body: {"ids": ["ACM05521", "cpu_mode", ...]}
// Response: {"ACM05521": "PRESENT", "cpu_mode": "3.14", ...} — only found IDs are included.
// Searches ALL registered stream buffers in-memory; no Redis round-trip needed.
func (h *SimHandler) QueryValues(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSimBodyBytes)
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	if len(req.IDs) == 0 {
		http.Error(w, `{"error":"ids array is required"}`, http.StatusBadRequest)
		return
	}

	// Build wanted-ID set for O(1) lookup.
	wanted := make(map[string]struct{}, len(req.IDs))
	for _, id := range req.IDs {
		wanted[id] = struct{}{}
	}

	// Scan all stream buffers and collect matching entries.
	result := make(map[string]string, len(req.IDs))
	for _, buf := range h.Store.All() {
		for k, v := range buf.Snapshot() {
			if _, ok := wanted[k]; ok {
				result[k] = v
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ─── Backend random sim control ───────────────────────────────────────────────

// StartSim handles POST /streams/{streamID}/sim/start.
// Body: SimRunConfig — interval_ms + params map with ranges from MongoDB.
func (h *SimHandler) StartSim(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSimBodyBytes)
	streamID := strings.ToUpper(r.PathValue("streamID"))

	var cfg SimRunConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, `{"error":"invalid SimRunConfig body"}`, http.StatusBadRequest)
		return
	}
	if len(cfg.Params) == 0 {
		http.Error(w, `{"error":"params map is empty — no mnemonics to simulate"}`, http.StatusBadRequest)
		return
	}
	if cfg.IntervalMs <= 0 {
		cfg.IntervalMs = 1000
	}

	h.Manager.Start(h.svcCtx, streamID, cfg)
	h.Logger.Info("sim started via API", "stream", streamID, "params", len(cfg.Params), "interval_ms", cfg.IntervalMs)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stream":      streamID,
		"running":     true,
		"params":      len(cfg.Params),
		"interval_ms": cfg.IntervalMs,
	})
}

// StopSim handles POST /streams/{streamID}/sim/stop.
func (h *SimHandler) StopSim(w http.ResponseWriter, r *http.Request) {
	streamID := strings.ToUpper(r.PathValue("streamID"))
	h.Manager.Stop(streamID)
	h.Logger.Info("sim stopped via API", "stream", streamID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"stream": streamID, "running": false})
}

// GetStreamSimStatus handles GET /streams/{streamID}/sim/status.
func (h *SimHandler) GetStreamSimStatus(w http.ResponseWriter, r *http.Request) {
	streamID := strings.ToUpper(r.PathValue("streamID"))
	running := h.Manager.IsRunning(streamID)
	intervalMs := 0
	if running {
		if all := h.Manager.Status(); all != nil {
			intervalMs = all[streamID]
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stream":      streamID,
		"running":     running,
		"interval_ms": intervalMs,
	})
}

// GetAllSimStatus handles GET /streams/sim/status — all running simulators.
func (h *SimHandler) GetAllSimStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.Manager.Status())
}

// ─── Inspection ───────────────────────────────────────────────────────────────

// GetStreams handles GET /streams — lists all registered stream buffers.
func (h *SimHandler) GetStreams(w http.ResponseWriter, r *http.Request) {
	type streamInfo struct {
		ID        string `json:"id"`
		ChainType string `json:"chain_type"`
		Params    int    `json:"params"`
		SimActive bool   `json:"sim_active"`
	}
	result := make([]streamInfo, 0)
	for _, b := range h.Store.All() {
		result = append(result, streamInfo{
			ID:        b.Meta.ID,
			ChainType: b.Meta.ChainType,
			Params:    b.Len(),
			SimActive: h.Manager.IsRunning(b.Meta.ID),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetStream handles GET /streams/{streamID} — current values for a stream.
func (h *SimHandler) GetStream(w http.ResponseWriter, r *http.Request) {
	streamID := strings.ToUpper(r.PathValue("streamID"))
	buf, ok := h.Store.Get(streamID)
	if !ok {
		http.Error(w, fmt.Sprintf(`{"error":"stream %q not found"}`, streamID), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         buf.Meta.ID,
		"sim_active": h.Manager.IsRunning(streamID),
		"values":     buf.Snapshot(),
	})
}
