package gateway

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/redis/go-redis/v9"
)

// DTMHandler serves PUT /dtm/values.
type DTMHandler struct {
	rdb    *redis.Client
	logger *slog.Logger
}

// PutValues accepts [{mnemonic, value}] and writes to both DTM_MAP and TM_MAP.
func (h *DTMHandler) PutValues(w http.ResponseWriter, r *http.Request) {
	var items []mnemonicValue
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		h.logger.Warn("invalid request body", "error", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(items) == 0 {
		writeError(w, http.StatusBadRequest, "empty values array")
		return
	}

	ctx := r.Context()

	// Build field-value pairs for HSET.
	fields := make([]interface{}, 0, len(items)*2)
	for _, item := range items {
		fields = append(fields, item.Mnemonic, item.Value)
	}

	// Write to DTM_MAP.
	if err := h.rdb.HSet(ctx, models.DTMMap, fields...).Err(); err != nil {
		h.logger.Error("Redis HSET failed", "key", models.DTMMap, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to write DTM_MAP")
		return
	}

	// Write to TM_MAP (merged view).
	if err := h.rdb.HSet(ctx, models.TMMap, fields...).Err(); err != nil {
		h.logger.Error("Redis HSET failed", "key", models.TMMap, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to write TM_MAP")
		return
	}

	h.logger.Info("DTM values updated", "count", len(items))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"updated": len(items),
	})
}
