package gateway

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

// MapsHandler serves full Redis hash map reads as arrays of {param, value} dicts.
type MapsHandler struct {
	rdb    *redis.Client
	logger *slog.Logger
}

// allowedMaps maps URL name → Redis key.
// Only explicitly listed maps are accessible (no arbitrary key reads).
var allowedMaps = map[string]string{
	"tm":    "TM_MAP",
	"tm1":   "TM1_MAP",
	"tm2":   "TM2_MAP",
	"smon1": "SMON1_MAP",
	"smon2": "SMON2_MAP",
	"adc1":  "ADC1_MAP",
	"adc2":  "ADC2_MAP",
	"udtm":  "UDTM_MAP",
	"dtm":   "DTM_MAP",
}

// MapEntry is a single row in the response — one param/value pair.
type MapEntry struct {
	Param string `json:"param"`
	Value string `json:"value"`
}

// GetMap handles GET /maps/{name}
// Returns the full Redis hash as an array of {param, value} objects.
// Valid names: tm, tm1, tm2, smon1, smon2, adc1, adc2, udtm, dtm
func (h *MapsHandler) GetMap(w http.ResponseWriter, r *http.Request) {
	name := strings.ToLower(chi.URLParam(r, "name"))

	redisKey, ok := allowedMaps[name]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "unknown map: " + name + ". Valid: tm, tm1, tm2, smon1, smon2, adc1, adc2, udtm, dtm",
		})
		return
	}

	data, err := h.rdb.HGetAll(r.Context(), redisKey).Result()
	if err != nil {
		h.logger.Error("failed to read Redis map", "key", redisKey, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	entries := make([]MapEntry, 0, len(data))
	for param, value := range data {
		entries = append(entries, MapEntry{Param: param, Value: value})
	}

	writeJSON(w, http.StatusOK, entries)
}
