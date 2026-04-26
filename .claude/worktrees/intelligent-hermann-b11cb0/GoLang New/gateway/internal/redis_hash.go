package gateway

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
)

// allowedHashPrefixes restricts WriteHash to a known set of key namespaces,
// preventing authenticated callers from writing to arbitrary Redis keys.
var allowedHashPrefixes = []string{
	"GUI_STATE:",
}

// RedisHashHandler provides generic Redis hash read/write endpoints.
type RedisHashHandler struct {
	rdb    *redis.Client
	logger *slog.Logger
}

type redisHashWriteRequest struct {
	Hash   string                 `json:"hash"`
	Values map[string]interface{} `json:"values"`
}

type redisHashReadRequest struct {
	Hash string   `json:"hash"`
	Keys []string `json:"keys"`
}

// WriteHash handles POST /redis/hash/write.
// Request body: {"hash":"GUI_STATE:tm/chain-simulator","values":{"mode":"FIXED"}}
func (h *RedisHashHandler) WriteHash(w http.ResponseWriter, r *http.Request) {
	var req redisHashWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Hash == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hash is required"})
		return
	}

	// Only allow writes to explicitly whitelisted key namespaces.
	allowed := false
	for _, prefix := range allowedHashPrefixes {
		if strings.HasPrefix(req.Hash, prefix) {
			allowed = true
			break
		}
	}
	if !allowed {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "hash key namespace not permitted"})
		return
	}

	if len(req.Values) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "values cannot be empty"})
		return
	}

	payload := make(map[string]string, len(req.Values))
	for k, v := range req.Values {
		if k == "" {
			continue
		}
		switch t := v.(type) {
		case string:
			payload[k] = t
		default:
			b, err := json.Marshal(t)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "values must be JSON-serializable"})
				return
			}
			payload[k] = string(b)
		}
	}

	if len(payload) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no valid fields to write"})
		return
	}

	if err := h.rdb.HSet(r.Context(), req.Hash, payload).Err(); err != nil {
		h.logger.Error("failed to write Redis hash", "hash", req.Hash, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	// Notify subscribers that this hash changed. Payload is intentionally generic.
	if err := h.rdb.Publish(r.Context(), req.Hash, "changed").Err(); err != nil {
		h.logger.Error("failed to publish Redis hash change", "hash", req.Hash, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hash written but publish failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"hash":           req.Hash,
		"written_fields": len(payload),
	})
}

// ReadHash handles POST /redis/hash/read.
// Request body: {"hash":"GUI_STATE:tm/chain-simulator","keys":["mode"]}
// If keys is empty, the full hash is returned.
func (h *RedisHashHandler) ReadHash(w http.ResponseWriter, r *http.Request) {
	var req redisHashReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Hash == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hash is required"})
		return
	}

	values := map[string]string{}
	if len(req.Keys) == 0 {
		all, err := h.rdb.HGetAll(r.Context(), req.Hash).Result()
		if err != nil {
			h.logger.Error("failed to read Redis hash", "hash", req.Hash, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			return
		}
		values = all
	} else {
		results, err := h.rdb.HMGet(r.Context(), req.Hash, req.Keys...).Result()
		if err != nil {
			h.logger.Error("failed to read Redis hash keys", "hash", req.Hash, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			return
		}

		for i, v := range results {
			if i >= len(req.Keys) || req.Keys[i] == "" || v == nil {
				continue
			}
			if s, ok := v.(string); ok {
				values[req.Keys[i]] = s
				continue
			}
			b, err := json.Marshal(v)
			if err != nil {
				continue
			}
			values[req.Keys[i]] = string(b)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"hash":   req.Hash,
		"values": values,
	})
}
