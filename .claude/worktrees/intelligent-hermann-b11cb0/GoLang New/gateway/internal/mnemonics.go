package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mainframe/tm-system/gateway/internal/parsers"
	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository"
	"github.com/redis/go-redis/v9"
)

// MnemonicsHandler serves autocomplete catalog endpoints backed by SQLite and Redis.
type MnemonicsHandler struct {
	tm     repository.TMMnemonicStore
	tc     repository.TCMnemonicStore
	sco    repository.SCOCommandStore
	rdb    *redis.Client
	logger *slog.Logger
}

// GetTMMnemonics handles GET /mnemonics/tm
// Optional query param: ?subsystem=PAYLOAD
func (h *MnemonicsHandler) GetTMMnemonics(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("GetTMMnemonics called", "query", r.URL.Query())

	if sub := r.URL.Query().Get("subsystem"); sub != "" {
		mnemonics, err := h.tm.FindBySubsystem(r.Context(), sub)
		if err != nil {
			h.logger.Error("tm_mnemonics query failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			return
		}
		h.logger.Info("GetTMMnemonics result", "count", len(mnemonics))
		writeJSON(w, http.StatusOK, mnemonics)
		return
	}

	// No subsystem filter — return all except UDTM, DTM, SMON*, ADC*
	all, err := h.tm.FindAll(r.Context())
	if err != nil {
		h.logger.Error("tm_mnemonics query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	smonRe := regexp.MustCompile(`(?i)^SMON([1-9]|10)$`)
	adcRe := regexp.MustCompile(`(?i)^ADC([1-9]|10)$`)
	filtered := all[:0]
	for _, m := range all {
		sub := strings.ToUpper(strings.TrimSpace(m.Subsystem))
		if sub == "UDTM" || sub == "DTM" {
			continue
		}
		if smonRe.MatchString(m.Subsystem) || adcRe.MatchString(m.Subsystem) {
			continue
		}
		filtered = append(filtered, m)
	}

	h.logger.Info("GetTMMnemonics result", "count", len(filtered))
	writeJSON(w, http.StatusOK, filtered)
}

// GetTMMnemonicsBySubsystem handles GET /mnemonics/tm/{subsystem}
func (h *MnemonicsHandler) GetTMMnemonicsBySubsystem(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subsystem")
	h.logger.Debug("GetTMMnemonicsBySubsystem called", "subsystem", sub)
	mnemonics, err := h.tm.FindBySubsystemPattern(r.Context(), "%"+sub+"%")
	if err != nil {
		h.logger.Error("tm_mnemonics subsystem query failed", "subsystem", sub, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	h.logger.Debug("GetTMMnemonicsBySubsystem result", "subsystem", sub, "count", len(mnemonics))
	writeJSON(w, http.StatusOK, mnemonics)
}

// GetAllTMMnemonics handles GET /get/mnemonics/tm
// Returns an array of id_mnemonic strings in the format "{_id}_{cdbMnemonic}".
func (h *MnemonicsHandler) GetAllTMMnemonics(w http.ResponseWriter, r *http.Request) {
	mnemonics, err := h.tm.FindAll(r.Context())
	if err != nil {
		h.logger.Error("get/mnemonics/tm query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	result := make([]string, len(mnemonics))
	for i, m := range mnemonics {
		result[i] = m.ID + "_" + m.CdbMnemonic
	}
	writeJSON(w, http.StatusOK, result)
}

// GetTMParamIDMnemonicMapping handles GET /mnemonics/tm/id_to_mnemonic_mapping
func (h *MnemonicsHandler) GetTMParamIDMnemonicMapping(w http.ResponseWriter, r *http.Request) {
	var (
		mnemonics []models.TmMnemonic
		err       error
	)
	if sub := strings.TrimSpace(r.URL.Query().Get("subsystem")); sub != "" {
		mnemonics, err = h.tm.FindBySubsystem(r.Context(), sub)
	} else {
		mnemonics, err = h.tm.FindAll(r.Context())
	}
	if err != nil {
		h.logger.Error("mnemonics/tm/id_to_mnemonic_mapping query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	mapping := make(map[string]string, len(mnemonics))
	for _, m := range mnemonics {
		paramID := strings.TrimSpace(m.ID)
		mnemonic := strings.TrimSpace(m.CdbMnemonic)
		if paramID == "" || mnemonic == "" {
			continue
		}
		mapping[paramID] = mnemonic
	}

	writeJSON(w, http.StatusOK, mapping)
}

// GetTMMnemonicsBySubsystemGET handles GET /get/mnemonics/tm/{subsystem}
func (h *MnemonicsHandler) GetTMMnemonicsBySubsystemGET(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subsystem")
	h.logger.Debug("GetTMMnemonicsBySubsystemGET called", "subsystem", sub)
	mnemonics, err := h.tm.FindBySubsystemPattern(r.Context(), "%"+sub+"%")
	if err != nil {
		h.logger.Error("get/mnemonics/tm/{subsystem} query failed", "subsystem", sub, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	result := make([]string, len(mnemonics))
	for i, m := range mnemonics {
		result[i] = m.ID + "_" + m.CdbMnemonic
	}
	writeJSON(w, http.StatusOK, result)
}

// GetMnemonicRange handles GET /get/mnemonics/tm/{subsystem}/{mnemonic}/range
func (h *MnemonicsHandler) GetMnemonicRange(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subsystem")
	mne := chi.URLParam(r, "mnemonic")
	h.logger.Debug("GetMnemonicRange called", "subsystem", sub, "mnemonic", mne)

	m, err := h.tm.FindBySubsystemAndMnemonic(r.Context(), "%"+sub+"%", mne)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "mnemonic not found"})
		return
	}
	if err != nil {
		h.logger.Error("get/mnemonics/tm/{subsystem}/{mnemonic}/range query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, m.GetRangeStr())
}

// GetTCMnemonics handles GET /mnemonics/tc
func (h *MnemonicsHandler) GetTCMnemonics(w http.ResponseWriter, r *http.Request) {
	docs, err := h.tc.FindAll(r.Context())
	if err != nil {
		h.logger.Error("tc_mnemonics query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	// Compute FullRef for each document's command field
	result := make([]models.TcMnemonic, 0, len(docs))
	for _, doc := range docs {
		var m models.TcMnemonic
		if b, err := json.Marshal(doc); err == nil {
			if err := json.Unmarshal(b, &m); err == nil {
				m.FullRef = "TC." + m.Command
				result = append(result, m)
			}
		}
	}
	writeJSON(w, http.StatusOK, result)
}

// GetSCOCommands handles GET /mnemonics/sco
func (h *MnemonicsHandler) GetSCOCommands(w http.ResponseWriter, r *http.Request) {
	cmds, err := h.sco.FindAll(r.Context())
	if err != nil {
		h.logger.Error("sco_commands query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	for i := range cmds {
		cmds[i].FullRef = "SCO." + cmds[i].Command
	}
	writeJSON(w, http.StatusOK, cmds)
}

// GetAllMnemonics handles GET /mnemonics/all — combined catalog for Monaco autocomplete.
func (h *MnemonicsHandler) GetAllMnemonics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tm, err := h.tm.FindAll(ctx)
	if err != nil {
		h.logger.Error("tm_mnemonics query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	tcDocs, err := h.tc.FindAll(ctx)
	if err != nil {
		h.logger.Error("tc_mnemonics query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	tc := make([]models.TcMnemonic, 0, len(tcDocs))
	for _, doc := range tcDocs {
		var m models.TcMnemonic
		if b, err := json.Marshal(doc); err == nil {
			if err := json.Unmarshal(b, &m); err == nil {
				m.FullRef = "TC." + m.Command
				tc = append(tc, m)
			}
		}
	}

	sco, err := h.sco.FindAll(ctx)
	if err != nil {
		h.logger.Error("sco_commands query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	for i := range sco {
		sco[i].FullRef = "SCO." + sco[i].Command
	}

	// UD_TM mnemonics live in tm_mnemonics with subsystem="UDTM"
	udtm, err := h.tm.FindBySubsystem(ctx, "UDTM")
	if err != nil {
		h.logger.Error("udtm query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tm_mnemonics":    tm,
		"tc_mnemonics":    tc,
		"sco_commands":    sco,
		"ud_tm_mnemonics": udtm,
	})
}

// GetSubsystems handles GET /telemetry/subsystems — distinct subsystem names from tm_mnemonics.
func (h *MnemonicsHandler) GetSubsystems(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("GetSubsystems called, querying tm_mnemonics")
	subs, err := h.tm.GetSubsystems(r.Context())
	if err != nil {
		h.logger.Error("subsystems query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	h.logger.Debug("GetSubsystems result", "count", len(subs))
	writeJSON(w, http.StatusOK, map[string][]string{"subsystems": subs})
}

// GetTCSubsystems handles GET /telecommand/subsystems — distinct subsystem names from tc_mnemonics.
func (h *MnemonicsHandler) GetTCSubsystems(w http.ResponseWriter, r *http.Request) {
	subs, err := h.tc.GetSubsystems(r.Context())
	if err != nil {
		h.logger.Error("tc subsystems query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string][]string{"subsystems": subs})
}

// GetTCMnemonicsBySubsystem handles GET /mnemonics/tc/{subsystem}.
func (h *MnemonicsHandler) GetTCMnemonicsBySubsystem(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subsystem")

	var (
		cmdDescs []string
		err      error
	)
	if strings.EqualFold(sub, "all") || sub == "" {
		cmdDescs, err = h.tc.GetAllCmdDescs(r.Context())
	} else {
		cmdDescs, err = h.tc.GetCmdDescsBySubsystem(r.Context(), "%"+sub+"%")
	}
	if err != nil {
		h.logger.Error("tc_mnemonics subsystem query failed", "subsystem", sub, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	sort.Strings(cmdDescs)
	writeJSON(w, http.StatusOK, cmdDescs)
}

// GetTCRecord handles GET /telecommand/record
func (h *MnemonicsHandler) GetTCRecord(w http.ResponseWriter, r *http.Request) {
	cmdDesc := strings.TrimSpace(r.URL.Query().Get("cmdDesc"))
	cmdId := strings.TrimSpace(r.URL.Query().Get("cmdId"))
	if cmdDesc == "" && cmdId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "cmdDesc or cmdId query param is required",
		})
		return
	}

	var (
		doc map[string]any
		err error
	)
	if cmdDesc != "" {
		doc, err = h.tc.FindByCmdDesc(r.Context(), cmdDesc)
	} else {
		doc, err = h.tc.GetByIDRaw(r.Context(), cmdId)
	}

	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "record not found"})
		return
	}
	if err != nil {
		h.logger.Error("tc_mnemonics record query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, doc)
}

// GetLiveTMMnemonics handles GET /tm/mnemonics — live TM_MAP key list from Redis.
func (h *MnemonicsHandler) GetLiveTMMnemonics(w http.ResponseWriter, r *http.Request) {
	keys, err := h.rdb.HKeys(r.Context(), models.TMMap).Result()
	if err != nil {
		h.logger.Error("failed to read TM_MAP keys", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string][]string{"mnemonics": keys})
}

type updateTMLimitsRequest struct {
	Subsystem string          `json:"subsystem"`
	Mnemonic  string          `json:"mnemonic"`
	Limits    json.RawMessage `json:"limits"`
}

type tmLimitsRow struct {
	Mnemonic string `json:"mnemonic"`
	Limits   []any  `json:"limits"`
}

type updateTMLimitsBulkRequest struct {
	Subsystem string                   `json:"subsystem"`
	Items     []updateTMLimitsBulkItem `json:"items"`
}

type updateTMLimitsBulkItem struct {
	Mnemonic string          `json:"mnemonic"`
	Limits   json.RawMessage `json:"limits"`
}

type updateTMFlagRequest struct {
	Subsystem string `json:"subsystem"`
	Mnemonic  string `json:"mnemonic"`
	Value     *bool  `json:"value"`
}

type updateTMToleranceRequest struct {
	Subsystem string   `json:"subsystem"`
	Mnemonic  string   `json:"mnemonic"`
	Tolerance *float64 `json:"tolerance"`
}

type updateTMExpectedValueRequest struct {
	Subsystem     string `json:"subsystem"`
	Mnemonic      string `json:"mnemonic"`
	ExpectedValue string `json:"expectedValue"`
}

type updateTMAvailableChainsRequest struct {
	Subsystem string          `json:"subsystem"`
	Mnemonic  string          `json:"mnemonic"`
	Value     json.RawMessage `json:"value"`
}

// GetTMLimitsBySubsystem handles GET /api/go/v1/telemetry/limits/{subsystem}.
func (h *MnemonicsHandler) GetTMLimitsBySubsystem(w http.ResponseWriter, r *http.Request) {
	sub := strings.TrimSpace(chi.URLParam(r, "subsystem"))
	if sub == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "subsystem is required",
		})
		return
	}

	rows, err := h.tm.FindBySubsystem(r.Context(), sub)
	if err != nil {
		h.logger.Error("failed to fetch telemetry limits", "error", err, "subsystem", sub)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"error":   "failed to fetch telemetry limits",
		})
		return
	}

	result := make([]tmLimitsRow, 0, len(rows))
	for _, m := range rows {
		if !strings.EqualFold(strings.TrimSpace(m.Type), "ANALOG") {
			continue
		}
		var limitsAny []any
		for _, v := range m.Limits {
			limitsAny = append(limitsAny, v)
		}
		result = append(result, tmLimitsRow{Mnemonic: m.CdbMnemonic, Limits: limitsAny})
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.ToUpper(result[i].Mnemonic) < strings.ToUpper(result[j].Mnemonic)
	})

	writeJSON(w, http.StatusOK, result)
}

// UpdateTMLimits handles PUT /api/v1/telemetry/limits.
func (h *MnemonicsHandler) UpdateTMLimits(w http.ResponseWriter, r *http.Request) {
	var req updateTMLimitsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid request body"})
		return
	}
	req.Subsystem = strings.TrimSpace(req.Subsystem)
	req.Mnemonic = strings.TrimSpace(req.Mnemonic)
	if req.Subsystem == "" || req.Mnemonic == "" || len(req.Limits) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "subsystem, mnemonic and limits are required"})
		return
	}

	limits, err := parsers.ParseLimitsPayload(req.Limits)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	matched, err := h.tm.PatchBySubsystemMnemonic(r.Context(), req.Subsystem, req.Mnemonic, func(doc map[string]any) {
		doc["limits"] = limits
		doc["updatedAt"] = now
	})
	if err != nil {
		h.logger.Error("failed to update tm_mnemonics limits", "error", err, "subsystem", req.Subsystem, "mnemonic", req.Mnemonic)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": "failed to update limits"})
		return
	}
	if matched == 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "mnemonic not found"})
		return
	}

	if err := h.rdb.Publish(r.Context(), models.MdbTmMnemonicsUpdated, "tm_limits_updated").Err(); err != nil {
		h.logger.Warn("failed to publish MDB_TM_MNEMONICS_UPDATED", "error", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "subsystem": req.Subsystem, "mnemonic": req.Mnemonic, "limits": limits})
}

// UpdateTMLimitsBulk handles PUT /api/go/v1/telemetry/limits/bulk.
func (h *MnemonicsHandler) UpdateTMLimitsBulk(w http.ResponseWriter, r *http.Request) {
	var req updateTMLimitsBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid request body"})
		return
	}
	req.Subsystem = strings.TrimSpace(req.Subsystem)
	if req.Subsystem == "" || len(req.Items) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "subsystem and non-empty items are required"})
		return
	}

	updated := make([]string, 0, len(req.Items))
	missing := make([]string, 0)
	now := time.Now().UTC().Format(time.RFC3339)

	for _, item := range req.Items {
		mnemonic := strings.TrimSpace(item.Mnemonic)
		if mnemonic == "" || len(item.Limits) == 0 {
			continue
		}

		limits, err := parsers.ParseLimitsPayload(item.Limits)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"error":   fmt.Sprintf("invalid limits for mnemonic %s: %s", mnemonic, err.Error()),
			})
			return
		}

		matched, err := h.tm.PatchBySubsystemMnemonic(r.Context(), req.Subsystem, mnemonic, func(doc map[string]any) {
			doc["limits"] = limits
			doc["updatedAt"] = now
		})
		if err != nil {
			h.logger.Error("failed bulk update for tm_mnemonics limits", "error", err, "subsystem", req.Subsystem, "mnemonic", mnemonic)
			writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": "failed to update limits"})
			return
		}
		if matched == 0 {
			missing = append(missing, mnemonic)
			continue
		}
		updated = append(updated, mnemonic)
	}

	if len(updated) > 0 {
		if err := h.rdb.Publish(r.Context(), models.MdbTmMnemonicsUpdated, "tm_limits_bulk_updated").Err(); err != nil {
			h.logger.Warn("failed to publish MDB_TM_MNEMONICS_UPDATED", "error", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true, "subsystem": req.Subsystem,
		"updated_count": len(updated), "updated": updated, "missing": missing,
	})
}

// UpdateTMTolerance handles PUT /api/go/v1/telemetry/tolerance.
func (h *MnemonicsHandler) UpdateTMTolerance(w http.ResponseWriter, r *http.Request) {
	var req updateTMToleranceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid request body"})
		return
	}
	req.Subsystem = strings.TrimSpace(req.Subsystem)
	req.Mnemonic = strings.TrimSpace(req.Mnemonic)
	if req.Subsystem == "" || req.Mnemonic == "" || req.Tolerance == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "subsystem, mnemonic and tolerance are required"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	tol := *req.Tolerance
	matched, err := h.tm.PatchBySubsystemMnemonic(r.Context(), req.Subsystem, req.Mnemonic, func(doc map[string]any) {
		doc["tolerance"] = tol
		doc["updatedAt"] = now
	})
	if err != nil {
		h.logger.Error("failed to update tm_mnemonics tolerance", "error", err, "subsystem", req.Subsystem, "mnemonic", req.Mnemonic)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": "failed to update tolerance"})
		return
	}
	if matched == 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "mnemonic not found"})
		return
	}

	if err := h.rdb.Publish(r.Context(), models.MdbTmMnemonicsUpdated, "tm_tolerance_updated").Err(); err != nil {
		h.logger.Warn("failed to publish MDB_TM_MNEMONICS_UPDATED", "error", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "subsystem": req.Subsystem, "mnemonic": req.Mnemonic, "tolerance": tol})
}

// UpdateTMExpectedValue handles PUT /api/go/v1/telemetry/expected-value.
func (h *MnemonicsHandler) UpdateTMExpectedValue(w http.ResponseWriter, r *http.Request) {
	var req updateTMExpectedValueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid request body"})
		return
	}
	req.Subsystem = strings.TrimSpace(req.Subsystem)
	req.Mnemonic = strings.TrimSpace(req.Mnemonic)
	req.ExpectedValue = strings.TrimSpace(req.ExpectedValue)
	if req.Subsystem == "" || req.Mnemonic == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "subsystem and mnemonic are required"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	ev := req.ExpectedValue
	matched, err := h.tm.PatchBySubsystemMnemonic(r.Context(), req.Subsystem, req.Mnemonic, func(doc map[string]any) {
		doc["expected_value"] = ev
		doc["updatedAt"] = now
	})
	if err != nil {
		h.logger.Error("failed to update tm_mnemonics expected value", "error", err, "subsystem", req.Subsystem, "mnemonic", req.Mnemonic)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": "failed to update expected value"})
		return
	}
	if matched == 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "mnemonic not found"})
		return
	}

	if err := h.rdb.Publish(r.Context(), models.MdbTmMnemonicsUpdated, "tm_expected_value_updated").Err(); err != nil {
		h.logger.Warn("failed to publish MDB_TM_MNEMONICS_UPDATED", "error", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "subsystem": req.Subsystem, "mnemonic": req.Mnemonic, "expected_value": ev})
}

// UpdateTMIgnoreLimitCheck handles PUT /api/v1/telemetry/ignore-limit-check.
func (h *MnemonicsHandler) UpdateTMIgnoreLimitCheck(w http.ResponseWriter, r *http.Request) {
	h.updateTMBooleanFlag(w, r, "ignore_limit_check")
}

// UpdateTMIgnoreChangeDetection handles PUT /api/v1/telemetry/ignore-change-detection.
func (h *MnemonicsHandler) UpdateTMIgnoreChangeDetection(w http.ResponseWriter, r *http.Request) {
	h.updateTMBooleanFlag(w, r, "ignore_change_detection")
}

// UpdateTMIgnoreChainComparision handles PUT /api/v1/telemetry/ignore-chain-comparision.
func (h *MnemonicsHandler) UpdateTMIgnoreChainComparision(w http.ResponseWriter, r *http.Request) {
	h.updateTMBooleanFlag(w, r, "ignore_chain_comparision")
}

// UpdateTMAvailableChains handles PUT /api/v1/telemetry/available-chains.
func (h *MnemonicsHandler) UpdateTMAvailableChains(w http.ResponseWriter, r *http.Request) {
	var req updateTMAvailableChainsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid request body"})
		return
	}
	req.Subsystem = strings.TrimSpace(req.Subsystem)
	req.Mnemonic = strings.TrimSpace(req.Mnemonic)
	if req.Subsystem == "" || req.Mnemonic == "" || len(req.Value) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "subsystem, mnemonic and value are required"})
		return
	}

	chains, err := parsers.ParseAvailableChains(req.Value)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	matched, err := h.tm.PatchBySubsystemMnemonic(r.Context(), req.Subsystem, req.Mnemonic, func(doc map[string]any) {
		doc["available_chains"] = chains
		doc["updatedAt"] = now
	})
	if err != nil {
		h.logger.Error("failed to update tm_mnemonics available_chains", "error", err, "subsystem", req.Subsystem, "mnemonic", req.Mnemonic)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": "failed to update available_chains"})
		return
	}
	if matched == 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "mnemonic not found"})
		return
	}

	if err := h.rdb.Publish(r.Context(), models.MdbTmMnemonicsUpdated, "tm_available_chains_updated").Err(); err != nil {
		h.logger.Warn("failed to publish MDB_TM_MNEMONICS_UPDATED", "error", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "subsystem": req.Subsystem, "mnemonic": req.Mnemonic, "available_chains": chains})
}

func (h *MnemonicsHandler) updateTMBooleanFlag(w http.ResponseWriter, r *http.Request, field string) {
	var req updateTMFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "invalid request body"})
		return
	}
	req.Subsystem = strings.TrimSpace(req.Subsystem)
	req.Mnemonic = strings.TrimSpace(req.Mnemonic)
	if req.Subsystem == "" || req.Mnemonic == "" || req.Value == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "subsystem, mnemonic and boolean value are required"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	val := *req.Value
	matched, err := h.tm.PatchBySubsystemMnemonic(r.Context(), req.Subsystem, req.Mnemonic, func(doc map[string]any) {
		doc[field] = val
		doc["updatedAt"] = now
	})
	if err != nil {
		h.logger.Error("failed to update tm_mnemonics flag", "error", err, "field", field, "subsystem", req.Subsystem, "mnemonic", req.Mnemonic)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": "failed to update flag"})
		return
	}
	if matched == 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{"success": false, "error": "mnemonic not found"})
		return
	}

	if err := h.rdb.Publish(r.Context(), models.MdbTmMnemonicsUpdated, "tm_flags_updated").Err(); err != nil {
		h.logger.Warn("failed to publish MDB_TM_MNEMONICS_UPDATED", "error", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "subsystem": req.Subsystem, "mnemonic": req.Mnemonic, "field": field, "value": val})
}

// writeJSON writes v as JSON with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// writeError writes {"error": msg} as JSON with the given HTTP status code.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
