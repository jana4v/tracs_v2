package gateway

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/redis/go-redis/v9"
)

// SimulatorHandler serves GET /simulator-status.
type SimulatorHandler struct {
	rdb    *redis.Client
	logger *slog.Logger
}

// GetSimulatorStatus returns HGETALL of TM_SIMULATOR_CFG_MAP as JSON.
func (h *SimulatorHandler) GetSimulatorStatus(w http.ResponseWriter, r *http.Request) {
	data, err := h.rdb.HGetAll(r.Context(), models.TMSimulatorCfgMap).Result()
	if err != nil {
		h.logger.Error("Redis HGETALL failed",
			"key", models.TMSimulatorCfgMap,
			"error", err,
		)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to read %s", models.TMSimulatorCfgMap))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// StopLegacySimulator handles POST /simulator/stop.
// It mirrors the standalone tm-simulator service HTTP stop: publish to TM_SIMULATOR_CTRL_CHANNEL,
// set ENABLE=0 in TM_SIMULATOR_CFG_MAP, and clear SIMULATED_TM_MAP. Use this when the GUI
// stops "random" simulation but the launcher-built simulator process is still running.
func (h *SimulatorHandler) StopLegacySimulator(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := h.rdb.Publish(ctx, models.TMSimulatorCtrlChannel, "stop").Err(); err != nil {
		h.logger.Warn("failed to publish TM_SIMULATOR_CTRL_CHANNEL stop", "error", err)
	}
	if err := h.rdb.HSet(ctx, models.TMSimulatorCfgMap, models.SimCfgEnable, "0").Err(); err != nil {
		h.logger.Error("failed to set simulator ENABLE=0", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := h.rdb.Del(ctx, models.SimulatedTMMap).Err(); err != nil {
		h.logger.Warn("failed to delete SIMULATED_TM_MAP", "error", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "running": false})
}
