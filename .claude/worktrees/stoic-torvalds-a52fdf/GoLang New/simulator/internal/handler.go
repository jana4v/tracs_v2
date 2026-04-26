package internal

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mainframe/tm-system/internal/models"
)

// SimulatorHandler provides HTTP handlers for the Simulator API (SRS Section 3.4).
type SimulatorHandler struct {
	sim    *Simulator
	logger *slog.Logger
}

// NewSimulatorHandler creates a new handler.
func NewSimulatorHandler(sim *Simulator, logger *slog.Logger) *SimulatorHandler {
	return &SimulatorHandler{sim: sim, logger: logger}
}

// cors wraps a handler with CORS headers.
func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

// RegisterRoutes registers all simulator HTTP routes.
func (h *SimulatorHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /api/go/v1/simulator/values", cors(h.UpdateValues))
	mux.HandleFunc("GET /api/go/v1/simulator/values", cors(h.GetValues))
	mux.HandleFunc("GET /api/go/v1/subsystems", cors(h.GetSubsystems))
	mux.HandleFunc("GET /api/go/v1/simulator/subsystems", cors(h.GetSubsystems))
	mux.HandleFunc("POST /api/go/v1/simulator/reset", cors(h.Reset))
	mux.HandleFunc("GET /api/go/v1/simulator-status", cors(h.GetStatus))
	mux.HandleFunc("GET /api/go/v1/simulator/mnemonics", cors(h.GetMnemonics))
	mux.HandleFunc("GET /api/go/v1/simulator/mnemonic/range", cors(h.GetMnemonicRange))
	mux.HandleFunc("GET /api/go/v1/simulator/mode", cors(h.GetSimulatorMode))
	mux.HandleFunc("PUT /api/go/v1/simulator/mode", cors(h.SetSimulatorMode))
	mux.HandleFunc("POST /api/go/v1/simulator/start", cors(h.StartSimulator))
	mux.HandleFunc("POST /api/go/v1/simulator/stop", cors(h.StopSimulator))
	mux.HandleFunc("OPTIONS /api/go/v1/simulator/values", cors(func(w http.ResponseWriter, r *http.Request) {}))
	mux.HandleFunc("OPTIONS /api/go/v1/subsystems", cors(func(w http.ResponseWriter, r *http.Request) {}))
	mux.HandleFunc("OPTIONS /api/go/v1/simulator/subsystems", cors(func(w http.ResponseWriter, r *http.Request) {}))
	mux.HandleFunc("OPTIONS /api/go/v1/simulator/reset", cors(func(w http.ResponseWriter, r *http.Request) {}))
	mux.HandleFunc("OPTIONS /api/go/v1/simulator-status", cors(func(w http.ResponseWriter, r *http.Request) {}))
	mux.HandleFunc("OPTIONS /api/go/v1/simulator/mnemonics", cors(func(w http.ResponseWriter, r *http.Request) {}))
	mux.HandleFunc("OPTIONS /api/go/v1/simulator/mnemonic/range", cors(func(w http.ResponseWriter, r *http.Request) {}))
	mux.HandleFunc("OPTIONS /api/go/v1/simulator/mode", cors(func(w http.ResponseWriter, r *http.Request) {}))
	mux.HandleFunc("OPTIONS /api/go/v1/simulator/start", cors(func(w http.ResponseWriter, r *http.Request) {}))
	mux.HandleFunc("OPTIONS /api/go/v1/simulator/stop", cors(func(w http.ResponseWriter, r *http.Request) {}))
}

// UpdateValues handles PUT /simulator/values (SRS 3.4.1).
// Request body: [{"mnemonic": "ACM05521", "value": "ABSENT"}]
func (h *SimulatorHandler) UpdateValues(w http.ResponseWriter, r *http.Request) {
	var items []struct {
		Mnemonic string `json:"mnemonic"`
		Value    string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	updates := make(map[string]string, len(items))
	for _, item := range items {
		updates[item.Mnemonic] = item.Value
	}

	if err := h.sim.UpdateValues(r.Context(), updates); err != nil {
		h.logger.Error("update values failed", "error", err)
		http.Error(w, `{"error":"update failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"updated": len(updates),
	})
}

// Reset handles POST /simulator/reset (SRS 3.4.2).
func (h *SimulatorHandler) Reset(w http.ResponseWriter, r *http.Request) {
	if err := h.sim.Reset(r.Context()); err != nil {
		h.logger.Error("reset failed", "error", err)
		http.Error(w, `{"error":"reset failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "all values reset to initial state",
	})
}

// GetMnemonics handles GET /simulator/mnemonics.
// Returns the full list of mnemonic definitions loaded by the simulator.
func (h *SimulatorHandler) GetMnemonics(w http.ResponseWriter, r *http.Request) {
	mnemonics := h.sim.GetMnemonics()
	if rawSubsystems := strings.TrimSpace(r.URL.Query().Get("subsystem")); rawSubsystems != "" {
		allowed := make(map[string]struct{})
		for _, token := range strings.Split(rawSubsystems, ",") {
			name := strings.ToUpper(strings.TrimSpace(token))
			if name == "" {
				continue
			}
			allowed[name] = struct{}{}
		}

		filtered := make([]models.TmMnemonic, 0, len(mnemonics))
		for _, mnemonic := range mnemonics {
			if _, ok := allowed[strings.ToUpper(strings.TrimSpace(mnemonic.Subsystem))]; ok {
				filtered = append(filtered, mnemonic)
			}
		}
		mnemonics = filtered
	}
	if mnemonics == nil {
		mnemonics = []models.TmMnemonic{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mnemonics)
}

// GetMnemonicRange handles GET /simulator/mnemonic/range?mnemonic=XXX.
// Returns the range values for a specific mnemonic.
func (h *SimulatorHandler) GetMnemonicRange(w http.ResponseWriter, r *http.Request) {
	mnemonic := r.URL.Query().Get("mnemonic")
	if mnemonic == "" {
		http.Error(w, `{"error":"mnemonic parameter required"}`, http.StatusBadRequest)
		return
	}

	mnemonics := h.sim.GetMnemonics()
	for _, m := range mnemonics {
		if m.CdbMnemonic == mnemonic {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"mnemonic": m.CdbMnemonic,
				"type":     m.Type,
				"range":    m.Range,
			})
			return
		}
	}

	http.Error(w, `{"error":"mnemonic not found"}`, http.StatusNotFound)
}

// GetSimulatorMode handles GET /simulator/mode.
// Returns the current simulation mode (FIXED or RANDOM).
func (h *SimulatorHandler) GetSimulatorMode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mode, err := h.sim.GetMode(ctx)
	if err != nil {
		h.logger.Error("get mode failed", "error", err)
		http.Error(w, `{"error":"failed to get mode"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"mode": mode,
	})
}

// SetSimulatorMode handles PUT /simulator/mode.
// Sets the simulation mode (FIXED or RANDOM).
func (h *SimulatorHandler) SetSimulatorMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	mode := strings.ToUpper(strings.TrimSpace(req.Mode))
	if mode != models.SimModeFixed && mode != models.SimModeRandom {
		http.Error(w, `{"error":"mode must be FIXED or RANDOM"}`, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := h.sim.SetMode(ctx, mode); err != nil {
		h.logger.Error("set mode failed", "error", err)
		http.Error(w, `{"error":"failed to set mode"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"mode":    mode,
	})
}

func (h *SimulatorHandler) StartSimulator(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := h.sim.Start(ctx); err != nil {
		h.logger.Error("start simulator failed", "error", err)
		http.Error(w, `{"error":"failed to start simulator"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"running": true,
	})
}

func (h *SimulatorHandler) StopSimulator(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := h.sim.Stop(ctx); err != nil {
		h.logger.Error("stop simulator failed", "error", err)
		http.Error(w, `{"error":"failed to stop simulator"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"running": false,
	})
}

// GetStatus handles GET /simulator-status (SRS 3.4.3).
func (h *SimulatorHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.sim.GetStatus(r.Context())
	if err != nil {
		h.logger.Error("get status failed", "error", err)
		http.Error(w, `{"error":"status unavailable"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (h *SimulatorHandler) GetValues(w http.ResponseWriter, r *http.Request) {
	subsystem := r.URL.Query().Get("subsystem")
	subsystems := []string{}
	if subsystem != "" {
		subsystems = strings.Split(subsystem, ",")
	}
	values, err := h.sim.GetValuesBySubsystems(r.Context(), subsystems)
	if err != nil {
		h.logger.Error("get values failed", "error", err)
		http.Error(w, `{"error":"values unavailable"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(values)
}

func (h *SimulatorHandler) GetSubsystems(w http.ResponseWriter, r *http.Request) {
	subsystems := h.sim.GetSubsystems()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subsystems)
}
