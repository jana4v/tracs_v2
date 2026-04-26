package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository"
	"github.com/redis/go-redis/v9"
)

const chainSimulatorConfigHash = "GUI_STATE:tm/chain-simulator"

// SimConfigSync listens for GUI simulator config changes in Redis and applies
// RANDOM-mode start/stop/interval updates to SimManager.
type SimConfigSync struct {
	rdb     *redis.Client
	tm      repository.TMMnemonicStore
	dtm     repository.DTMStore
	udtm    repository.UDTMStore
	store   *StreamStore
	manager *SimManager
	logger  *slog.Logger
}

func NewSimConfigSync(rdb *redis.Client, tm repository.TMMnemonicStore, dtm repository.DTMStore, udtm repository.UDTMStore, store *StreamStore, manager *SimManager, logger *slog.Logger) *SimConfigSync {
	return &SimConfigSync{rdb: rdb, tm: tm, dtm: dtm, udtm: udtm, store: store, manager: manager, logger: logger}
}

func (s *SimConfigSync) Run(ctx context.Context) {
	pubsub := s.rdb.Subscribe(ctx, chainSimulatorConfigHash)
	defer pubsub.Close()

	if _, err := pubsub.Receive(ctx); err != nil {
		s.logger.Error("sim config sync subscribe failed", "error", err)
		return
	}

	// Apply persisted state once on startup.
	s.applyHashState(ctx)

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ch:
			if !ok {
				return
			}
			s.applyHashState(ctx)
		}
	}
}

func (s *SimConfigSync) applyHashState(ctx context.Context) {
	data, err := s.rdb.HGetAll(ctx, chainSimulatorConfigHash).Result()
	if err != nil {
		s.logger.Warn("sim config sync read failed", "hash", chainSimulatorConfigHash, "error", err)
		return
	}

	mode := strings.ToUpper(strings.TrimSpace(data["mode"]))
	running := parseBool(data["isRunning"])
	interval := parseInterval(data["intervalMs"], 1000)
	selectedChains := parseSelectedChains(data["selectedChains"])

	if !running || mode != "RANDOM" {
		s.stopAll()
		return
	}

	desired := make(map[string]struct{})
	for _, chain := range selectedChains {
		streamID := strings.ToUpper(strings.TrimSpace(chain))
		if streamID == "" {
			continue
		}
		desired[streamID] = struct{}{}
	}

	if len(desired) == 0 {
		s.stopAll()
		return
	}

	current := s.manager.Status()
	for streamID := range current {
		if _, ok := desired[streamID]; !ok {
			s.manager.Stop(streamID)
			s.logger.Info("sim config sync stopped stream", "stream", streamID)
		}
	}

	for streamID := range desired {
		cfg, ok := s.buildConfigForStream(streamID, interval)
		if !ok {
			s.logger.Warn("sim config sync skipped stream (no params yet)", "stream", streamID)
			continue
		}

		if curInterval, ok := current[streamID]; ok && curInterval == interval {
			continue
		}

		s.manager.Start(ctx, streamID, cfg)
		s.logger.Info("sim config sync started stream", "stream", streamID, "interval_ms", interval, "params", len(cfg.Params))
	}
}

func (s *SimConfigSync) stopAll() {
	for streamID := range s.manager.Status() {
		s.manager.Stop(streamID)
		s.logger.Info("sim config sync stopped stream", "stream", streamID)
	}
}

func parseSelectedChains(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	// Handle normal string arrays first: ["TM1","TM2"]
	var chains []string
	if err := json.Unmarshal([]byte(raw), &chains); err == nil {
		return normalizeChainList(chains)
	}

	// Handle object arrays from UI controls:
	// [{"value":"TM1"}] or [{"id":"TM1"}] or [{"label":"TM1"}]
	var objects []map[string]any
	if err := json.Unmarshal([]byte(raw), &objects); err == nil {
		picked := make([]string, 0, len(objects))
		for _, obj := range objects {
			for _, key := range []string{"value", "id", "label"} {
				if v, ok := obj[key]; ok {
					picked = append(picked, fmt.Sprintf("%v", v))
					break
				}
			}
		}
		return normalizeChainList(picked)
	}

	// Handle double-encoded payloads: "[\"TM1\"]"
	var wrapped string
	if err := json.Unmarshal([]byte(raw), &wrapped); err == nil && strings.TrimSpace(wrapped) != "" {
		return parseSelectedChains(wrapped)
	}

	return nil
}

func normalizeChainList(chains []string) []string {
	result := make([]string, 0, len(chains))
	seen := make(map[string]struct{}, len(chains))
	for _, chain := range chains {
		normalized := strings.ToUpper(strings.TrimSpace(chain))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	return result
}

func (s *SimConfigSync) buildConfigForStream(streamID string, intervalMs int) (SimRunConfig, bool) {
	params, err := s.loadParamRanges(context.Background(), streamID)
	if err != nil {
		s.logger.Warn("sim config sync range load failed", "stream", streamID, "error", err)
		return SimRunConfig{}, false
	}
	if len(params) == 0 {
		return SimRunConfig{}, false
	}

	return SimRunConfig{IntervalMs: intervalMs, Params: params}, true
}

func (s *SimConfigSync) loadParamRanges(ctx context.Context, streamID string) (map[string]ParamRange, error) {
	upper := strings.ToUpper(strings.TrimSpace(streamID))

	switch {
	case strings.HasPrefix(upper, "TM"):
		return s.loadTMRanges(ctx)
	case strings.HasPrefix(upper, "SMON") || strings.HasPrefix(upper, "ADC"):
		return s.loadSCOSRanges(ctx, upper)
	case upper == "DTM":
		return s.loadDTMRanges(ctx)
	case upper == "UDTM":
		return s.loadUDTMRanges(ctx)
	default:
		return nil, fmt.Errorf("unknown stream type: %s", streamID)
	}
}

func (s *SimConfigSync) loadTMRanges(ctx context.Context) (map[string]ParamRange, error) {
	mnemonics, err := s.tm.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("load tm_mnemonics: %w", err)
	}

	params := make(map[string]ParamRange, len(mnemonics))
	for _, m := range mnemonics {
		key := strings.TrimSpace(m.ID)
		if key == "" {
			continue
		}
		params[key] = mnemonicToParamRange(m)
	}
	return params, nil
}

func (s *SimConfigSync) loadSCOSRanges(ctx context.Context, streamID string) (map[string]ParamRange, error) {
	redisKey := models.ChainMapKeyByName(streamID)
	data, err := s.rdb.HGetAll(ctx, redisKey).Result()
	if err != nil {
		return nil, fmt.Errorf("HGetAll %s: %w", redisKey, err)
	}

	params := make(map[string]ParamRange, len(data))
	for param := range data {
		params[param] = ParamRange{Type: "ANALOG", Min: 0, Max: 100}
	}
	return params, nil
}

func (s *SimConfigSync) loadDTMRanges(ctx context.Context) (map[string]ParamRange, error) {
	doc, err := s.dtm.Get(ctx, "default")
	if errors.Is(err, repository.ErrNotFound) {
		return map[string]ParamRange{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load dtm_procedures: %w", err)
	}

	params := make(map[string]ParamRange, len(doc.Rows))
	for _, row := range doc.Rows {
		if strings.TrimSpace(row.Mnemonic) == "" {
			continue
		}
		pr, err := parseDTMRange(row.Type, row.Range)
		if err != nil {
			s.logger.Warn("simconfigsync: invalid DTM range, using defaults",
				"mnemonic", row.Mnemonic, "range", row.Range, "error", err)
		}
		params[row.Mnemonic] = pr
	}
	return params, nil
}

func (s *SimConfigSync) loadUDTMRanges(ctx context.Context) (map[string]ParamRange, error) {
	doc, err := s.udtm.Get(ctx, "default")
	if errors.Is(err, repository.ErrNotFound) {
		return map[string]ParamRange{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load user_telemetry: %w", err)
	}

	params := make(map[string]ParamRange, len(doc.Rows))
	for _, row := range doc.Rows {
		if strings.TrimSpace(row.Mnemonic) == "" {
			continue
		}
		t := strings.ToUpper(strings.TrimSpace(row.Type))
		pr := ParamRange{Type: t, Min: 0, Max: 1}
		if t == "DIGITAL" || t == "BINARY" {
			pr.States = []string{"0", "1"}
		}
		params[row.Mnemonic] = pr
	}
	return params, nil
}

func mnemonicToParamRange(m models.TmMnemonic) ParamRange {
	t := strings.ToUpper(strings.TrimSpace(m.Type))
	if t == "ANALOG" {
		pr := ParamRange{Type: "ANALOG", Max: 100}
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
		if st := fmt.Sprintf("%v", v); st != "" {
			states = append(states, st)
		}
	}
	return ParamRange{Type: "DIGITAL", States: states}
}

// parseDTMRange converts DTMProcedureRow.Range string to ParamRange.
// ANALOG format: "min:max" (e.g. "-5:5"). DIGITAL format: comma-separated states.
// Returns an error if ANALOG range values cannot be parsed, but still returns usable defaults.
func parseDTMRange(typ, rangeStr string) (ParamRange, error) {
	t := strings.ToUpper(strings.TrimSpace(typ))
	if t == "ANALOG" {
		pr := ParamRange{Type: "ANALOG", Max: 100}
		parts := strings.SplitN(rangeStr, ":", 2)
		if len(parts) == 2 {
			min, errMin := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			max, errMax := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if errMin != nil || errMax != nil {
				return pr, fmt.Errorf("parse range %q: min=%v max=%v", rangeStr, errMin, errMax)
			}
			pr.Min = min
			pr.Max = max
		}
		return pr, nil
	}
	states := make([]string, 0)
	for _, s := range strings.Split(rangeStr, ",") {
		if st := strings.TrimSpace(s); st != "" {
			states = append(states, st)
		}
	}
	return ParamRange{Type: "DIGITAL", States: states}, nil
}

func toFloat64(v interface{}) float64 {
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseStringArray(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func parseBool(raw string) bool {
	s := strings.ToLower(strings.TrimSpace(raw))
	return s == "1" || s == "true" || s == "yes"
}

func parseInterval(raw string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return def
	}
	if n < 100 {
		return 100
	}
	if n > int((60 * time.Second).Milliseconds()) {
		return int((60 * time.Second).Milliseconds())
	}
	return n
}
