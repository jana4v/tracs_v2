package gateway

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
)

// TelemetryHandler serves POST /get-telemetry (SRS 10.1).
// Values are read directly from the ingest service's in-memory StreamStore via
// POST /streams/query — no Redis round-trip required.
type TelemetryHandler struct {
	ingestSimURL string // e.g. "http://localhost:21004"
	logger       *slog.Logger
}

// telemetryRequest is the expected JSON body for get-telemetry.
type telemetryRequest struct {
	IDs []string `json:"ids"` // parameter IDs (paramID for TM, param name for SCOS/DTM/UDTM)
}

// telemetryResponse is the JSON response for get-telemetry.
type telemetryResponse struct {
	Results map[string]*string `json:"results"`
}

// GetTelemetry fetches current values for the requested parameter IDs from the
// ingest service's in-memory StreamStore. Returns null for any ID not found.
func (h *TelemetryHandler) GetTelemetry(w http.ResponseWriter, r *http.Request) {
	var req telemetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("invalid request body", "error", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "ids array is required")
		return
	}

	// Forward to ingest /streams/query — same JSON shape.
	body, _ := json.Marshal(map[string][]string{"ids": req.IDs})
	ingestResp, err := simHTTPClient.Post(
		h.ingestSimURL+"/streams/query",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		h.logger.Warn("get-telemetry: ingest unreachable", "error", err)
		// Return all nulls so the UI degrades gracefully rather than hard-erroring.
		results := make(map[string]*string, len(req.IDs))
		for _, id := range req.IDs {
			results[id] = nil
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(telemetryResponse{Results: results})
		return
	}
	defer ingestResp.Body.Close()

	// Parse the flat id→value map returned by ingest.
	var vals map[string]string
	if err := json.NewDecoder(ingestResp.Body).Decode(&vals); err != nil {
		h.logger.Error("get-telemetry: failed to decode ingest response", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Build results: include every requested ID; null for those not yet in any stream.
	results := make(map[string]*string, len(req.IDs))
	for _, id := range req.IDs {
		if v, ok := vals[id]; ok {
			s := v
			results[id] = &s
		} else {
			results[id] = nil
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(telemetryResponse{Results: results})
}
