package gateway

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/redis/go-redis/v9"
)

// LimitsHandler serves GET /limit-failures.
type LimitsHandler struct {
	rdb    *redis.Client
	logger *slog.Logger
}

// GetLimitFailures returns HGETALL of TM_LIMIT_FAILURES_MAP as JSON.
func (h *LimitsHandler) GetLimitFailures(w http.ResponseWriter, r *http.Request) {
	data, err := h.rdb.HGetAll(r.Context(), models.TMLimitFailuresMap).Result()
	if err != nil {
		h.logger.Error("Redis HGETALL failed",
			"key", models.TMLimitFailuresMap,
			"error", err,
		)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to read %s", models.TMLimitFailuresMap))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
