package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository"
)

// ─── types shared with ingest (matching JSON tags) ────────────────────────────

// simParamRange mirrors ingest.ParamRange for JSON serialisation.
type simParamRange struct {
	Type   string   `json:"type"`
	Min    float64  `json:"min"`
	Max    float64  `json:"max"`
	States []string `json:"states"`
}

// simRunConfig mirrors ingest.SimRunConfig for JSON serialisation.
type simRunConfig struct {
	IntervalMs int                      `json:"interval_ms"`
	Params     map[string]simParamRange `json:"params"`
}

// ─── handler ─────────────────────────────────────────────────────────────────

// SimStartHandler builds the full SimRunConfig from SQLite/Redis and forwards
// start/stop/status calls to the ingest sim API.
type SimStartHandler struct {
	tm           repository.TMMnemonicStore
	dtm          repository.DTMStore
	udtm         repository.UDTMStore
	ingestSimURL string
	logger       *slog.Logger
}

// StartSim handles POST /sim/streams/{streamID}/start.
func (h *SimStartHandler) StartSim(w http.ResponseWriter, r *http.Request) {
	streamID := strings.ToUpper(chi.URLParam(r, "streamID"))

	var req struct {
		IntervalMs int `json:"interval_ms"`
	}
	json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck — body is optional
	if req.IntervalMs <= 0 {
		req.IntervalMs = 1000
	}

	params, err := h.loadParamRanges(r.Context(), streamID)
	if err != nil {
		h.logger.Error("simstart: failed to load param ranges", "stream", streamID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if len(params) == 0 {
		switch {
		case strings.HasPrefix(streamID, "SMON") || strings.HasPrefix(streamID, "ADC"):
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("no mnemonics found for stream %s (Redis %s is empty)", streamID, models.ChainMapKeyByName(streamID)),
			})
		case streamID == "DTM":
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "no mnemonics found for stream DTM (dtm_procedures not configured)",
			})
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("no mnemonics found for stream %s", streamID),
			})
		}
		return
	}

	cfg := simRunConfig{IntervalMs: req.IntervalMs, Params: params}
	h.forwardToIngest(w, r.Context(), "POST", "/streams/"+streamID+"/sim/start", cfg)
}

// StopSim handles POST /sim/streams/{streamID}/stop.
func (h *SimStartHandler) StopSim(w http.ResponseWriter, r *http.Request) {
	streamID := strings.ToUpper(chi.URLParam(r, "streamID"))
	h.forwardToIngest(w, r.Context(), "POST", "/streams/"+streamID+"/sim/stop", nil)
}

// GetSimStatus handles GET /sim/streams/{streamID}/status.
func (h *SimStartHandler) GetSimStatus(w http.ResponseWriter, r *http.Request) {
	streamID := strings.ToUpper(chi.URLParam(r, "streamID"))
	h.forwardToIngest(w, r.Context(), "GET", "/streams/"+streamID+"/sim/status", nil)
}

// GetAllSimStatus handles GET /sim/streams/status.
func (h *SimStartHandler) GetAllSimStatus(w http.ResponseWriter, r *http.Request) {
	h.forwardToIngest(w, r.Context(), "GET", "/streams/sim/status", nil)
}

// ─── range loading ────────────────────────────────────────────────────────────

func (h *SimStartHandler) loadParamRanges(ctx context.Context, streamID string) (map[string]simParamRange, error) {
	upper := strings.ToUpper(streamID)

	switch {
	case strings.HasPrefix(upper, "TM"):
		return h.loadTMRanges(ctx)
	case strings.HasPrefix(upper, "SMON") || strings.HasPrefix(upper, "ADC"):
		return nil, fmt.Errorf("SCOS sim start not supported via this endpoint (use Redis-driven sim)")
	case upper == "DTM":
		return h.loadDTMRanges(ctx)
	case upper == "UDTM":
		return h.loadUDTMRanges(ctx)
	default:
		return nil, fmt.Errorf("unknown stream type: %s", streamID)
	}
}

func (h *SimStartHandler) loadTMRanges(ctx context.Context) (map[string]simParamRange, error) {
	mnemonics, err := h.tm.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("query tm_mnemonics: %w", err)
	}

	params := make(map[string]simParamRange, len(mnemonics))
	for _, m := range mnemonics {
		params[m.ID] = mnemonicToParamRange(m)
	}
	return params, nil
}

type simDTMProcRow struct {
	Mnemonic string        `json:"mnemonic"`
	Type     string        `json:"type"`
	Range    []interface{} `json:"range"`
}

type simDTMProcDoc struct {
	Rows []simDTMProcRow `json:"rows"`
}

func (h *SimStartHandler) loadDTMRanges(ctx context.Context) (map[string]simParamRange, error) {
	rawDoc, err := h.dtm.GetRaw(ctx, "default")
	if errors.Is(err, repository.ErrNotFound) {
		return map[string]simParamRange{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query dtm_procedures: %w", err)
	}

	// Re-encode and decode into the local simDTMProcDoc type to extract rows with []interface{} range.
	b, _ := json.Marshal(rawDoc)
	var doc simDTMProcDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("decode dtm_procedures: %w", err)
	}

	params := make(map[string]simParamRange, len(doc.Rows))
	for _, row := range doc.Rows {
		if row.Mnemonic == "" {
			continue
		}
		params[row.Mnemonic] = rawRangeToParamRange(row.Type, row.Range)
	}
	return params, nil
}

type simUDTMRow struct {
	Mnemonic string `json:"mnemonic"`
	Type     string `json:"type"`
}

type simUDTMDoc struct {
	Rows []simUDTMRow `json:"rows"`
}

func (h *SimStartHandler) loadUDTMRanges(ctx context.Context) (map[string]simParamRange, error) {
	doc, err := h.udtm.Get(ctx, "default")
	if errors.Is(err, repository.ErrNotFound) {
		return map[string]simParamRange{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query user_telemetry: %w", err)
	}

	params := make(map[string]simParamRange, len(doc.Rows))
	for _, row := range doc.Rows {
		if row.Mnemonic == "" {
			continue
		}
		t := strings.ToUpper(row.Type)
		pr := simParamRange{Type: t, Min: 0, Max: 1}
		if t == "DIGITAL" || t == "BINARY" {
			pr.States = []string{"0", "1"}
		}
		params[row.Mnemonic] = pr
	}
	return params, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// mnemonicToParamRange converts a TmMnemonic to a simParamRange.
func mnemonicToParamRange(m models.TmMnemonic) simParamRange {
	t := strings.ToUpper(strings.TrimSpace(m.Type))
	if t == "ANALOG" {
		pr := simParamRange{Type: "ANALOG", Max: 100}
		if len(m.Range) >= 2 {
			pr.Min = toFloat64(m.Range[0])
			pr.Max = toFloat64(m.Range[1])
		} else if len(m.Range) == 1 {
			pr.Max = toFloat64(m.Range[0])
		}
		return pr
	}
	states := make([]string, 0, len(m.Range))
	for _, v := range m.Range {
		if s := fmt.Sprintf("%v", v); s != "" {
			states = append(states, s)
		}
	}
	return simParamRange{Type: "DIGITAL", States: states}
}

// rawRangeToParamRange handles generic []interface{} range fields from DTM docs.
func rawRangeToParamRange(typ string, raw []interface{}) simParamRange {
	t := strings.ToUpper(strings.TrimSpace(typ))
	if t == "ANALOG" {
		pr := simParamRange{Type: "ANALOG", Max: 100}
		if len(raw) >= 2 {
			pr.Min = toFloat64(raw[0])
			pr.Max = toFloat64(raw[1])
		}
		return pr
	}
	states := make([]string, 0, len(raw))
	for _, v := range raw {
		if s := fmt.Sprintf("%v", v); s != "" {
			states = append(states, s)
		}
	}
	return simParamRange{Type: "DIGITAL", States: states}
}

// toFloat64 converts an interface{} (int, float64, string, etc.) to float64.
func toFloat64(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case int32:
		return float64(x)
	case string:
		var f float64
		fmt.Sscanf(x, "%f", &f)
		return f
	default:
		var f float64
		fmt.Sscanf(fmt.Sprintf("%v", x), "%f", &f)
		return f
	}
}

// forwardToIngest sends method+path to the ingest sim API and relays the response.
func (h *SimStartHandler) forwardToIngest(w http.ResponseWriter, ctx context.Context, method, path string, body interface{}) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "marshal error"})
			return
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, h.ingestSimURL+path, bodyReader)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "proxy error"})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := simHTTPClient.Do(req)
	if err != nil {
		h.logger.Warn("simstart: ingest unreachable", "path", path, "error", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "ingest sim API unavailable"})
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		h.logger.Warn("simstart: error copying upstream response body", "path", path, "error", err)
	}
}
