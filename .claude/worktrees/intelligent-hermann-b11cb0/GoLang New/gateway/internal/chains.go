package gateway

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/redis/go-redis/v9"
)

// ChainsHandler serves chain-related endpoints.
type ChainsHandler struct {
	rdb    *redis.Client
	logger *slog.Logger
}

// chainStatusEntry represents the heartbeat status of a single chain.
type chainStatusEntry struct {
	Chain  string `json:"chain"`
	Status string `json:"status"` // "active" or "inactive"
}

// configuredChains lists the default chain names to check heartbeats for.
// In production this would come from configuration; kept simple for Phase 2.
var configuredChains = []string{"TM1", "TM2", "TM3", "TM4", "SMON1", "ADC1"}

// GetChainStatus reads heartbeat status keys from Redis for all configured chains
// and returns active/inactive per chain.
func (h *ChainsHandler) GetChainStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	entries := make([]chainStatusEntry, 0, len(configuredChains))

	for _, chain := range configuredChains {
		key := models.HeartbeatStatusKey(chain)
		val, err := h.rdb.Get(ctx, key).Result()

		status := "inactive"
		if err == nil && val != "" {
			status = "active"
		} else if err != nil && err != redis.Nil {
			h.logger.Warn("Redis GET failed for heartbeat key",
				"key", key,
				"error", err,
			)
		}

		entries = append(entries, chainStatusEntry{
			Chain:  chain,
			Status: status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"chains": entries,
	})
}

// GetChainMismatches returns HGETALL of TM_CHAIN_MISMATCHES_MAP as JSON.
func (h *ChainsHandler) GetChainMismatches(w http.ResponseWriter, r *http.Request) {
	data, err := h.rdb.HGetAll(r.Context(), models.TMChainMismatchesMap).Result()
	if err != nil {
		h.logger.Error("Redis HGETALL failed",
			"key", models.TMChainMismatchesMap,
			"error", err,
		)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to read %s", models.TMChainMismatchesMap))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
