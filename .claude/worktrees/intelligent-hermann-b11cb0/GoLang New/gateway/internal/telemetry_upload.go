package gateway

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mainframe/tm-system/gateway/internal/parsers"
	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository"
	"github.com/redis/go-redis/v9"
)

type TelemetryUploadHandler struct {
	tm     repository.TMMnemonicStore
	rdb    *redis.Client
	logger *slog.Logger
}

type telemetryUploadRequest struct {
	Filename string `json:"filename"`
	Data     string `json:"data"`
}

type telemetryUploadStats struct {
	Total    int      `json:"total"`
	Inserted int      `json:"inserted"`
	Updated  int      `json:"updated"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

func (h *TelemetryUploadHandler) UploadTelemetry(w http.ResponseWriter, r *http.Request) {
	var req telemetryUploadRequest
	if err := jsonDecode(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "invalid request body",
		})
		return
	}

	req.Filename = strings.TrimSpace(req.Filename)
	req.Data = strings.TrimSpace(req.Data)
	if req.Filename == "" || req.Data == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "No file data provided. Send JSON with 'filename' and 'data' (base64).",
		})
		return
	}

	ext := strings.ToLower(filepath.Ext(req.Filename))
	if ext != ".xlsx" && ext != ".out" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Unsupported file type: %s. Supported: .xlsx, .out", ext),
		})
		return
	}

	fileBytes, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "Invalid base64 file data",
		})
		return
	}

	tmpFile, err := os.CreateTemp("", "tm_upload_*"+ext)
	if err != nil {
		h.logger.Error("failed to create temp file", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"error":   "failed to create temp file",
		})
		return
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(fileBytes); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		h.logger.Error("failed to write temp upload file", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"error":   "failed to persist upload",
		})
		return
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"success": false,
			"error":   "failed to finalize upload",
		})
		return
	}
	defer os.Remove(tmpPath)

	h.logger.Info("TM upload saved temp file", "path", tmpPath, "bytes", len(fileBytes), "filename", req.Filename)

	var records []map[string]any
	if ext == ".xlsx" {
		records, err = parsers.ParseXLSX(tmpPath)
	} else {
		records, err = parsers.ParseOUT(tmpPath)
	}
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Failed to parse %s: %v", strings.ToUpper(strings.TrimPrefix(ext, ".")), err),
		})
		return
	}

	h.logger.Info("TM upload parsed records", "filename", req.Filename, "records", len(records))

	stats := h.upsertTMPIDsBulk(r, records, req.Filename)
	if stats.Inserted > 0 || stats.Updated > 0 {
		if err := h.rdb.Publish(r.Context(), models.MdbTmMnemonicsUpdated, "telemetry_upload").Err(); err != nil {
			h.logger.Warn("failed to publish MDB_TM_MNEMONICS_UPDATED", "error", err)
		}
	}

	status := http.StatusOK
	if len(stats.Errors) > 0 {
		status = http.StatusMultiStatus
	}
	writeJSON(w, status, map[string]any{"success": true, "filename": req.Filename, "stats": stats})
}

func (h *TelemetryUploadHandler) upsertTMPIDsBulk(r *http.Request, records []map[string]any, sourceFile string) telemetryUploadStats {
	stats := telemetryUploadStats{Total: len(records), Errors: make([]string, 0)}

	for _, rec := range records {
		result, err := h.upsertTMPID(r, rec, sourceFile)
		if err != nil {
			pid := toString(rec["cdbPidNo"])
			if pid == "" {
				pid = "UNKNOWN"
			}
			stats.Errors = append(stats.Errors, fmt.Sprintf("[%s] %v", pid, err))
			continue
		}
		switch result {
		case "inserted":
			stats.Inserted++
		case "updated":
			stats.Updated++
		default:
			stats.Skipped++
		}
	}

	h.logger.Info("TM bulk upsert completed",
		"total", stats.Total, "inserted", stats.Inserted,
		"updated", stats.Updated, "skipped", stats.Skipped, "errors", len(stats.Errors))

	return stats
}

func (h *TelemetryUploadHandler) upsertTMPID(r *http.Request, record map[string]any, sourceFile string) (string, error) {
	pid := toString(record["cdbPidNo"])
	if pid == "" {
		return "skipped", nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	sub := firstNonEmpty(toString(record["sourceSheet"]), toString(record["subsystem"]))

	doc := map[string]any{
		"_id":                pid,
		"subsystem":          sub,
		"cdbPidNo":           pid,
		"cdbMnemonic":        toString(record["cdbMnemonic"]),
		"type":               toString(record["type"]),
		"processingType":     toString(record["processingType"]),
		"samplingRate":       recordOrEmpty(record, "samplingRate"),
		"dwellAddress":       toString(record["dwellAddress"]),
		"pidAddress":         toString(record["pidAddress"]),
		"pt":                 toString(record["pt"]),
		"range":              parsers.NormalizeRangeLikeValue(recordOrEmpty(record, "range")),
		"expected_value":     parsers.DeriveExpectedValue(record),
		"resolutionA1":       recordOrEmpty(record, "resolutionA1"),
		"offsetA0":           recordOrEmpty(record, "offsetA0"),
		"tolerance":          recordOrEmpty(record, "tolerance"),
		"unit":               toString(record["unit"]),
		"digitalStatus":      toString(record["digitalStatus"]),
		"condSrc":            toString(record["condSrc"]),
		"condSts":            toString(record["condSts"]),
		"gcoMnemonic":        toString(record["gcoMnemonic"]),
		"pidScope":           toString(record["pidScope"]),
		"lutRef":             toString(record["lutRef"]),
		"qualificationLimit": recordOrEmpty(record, "qualificationLimit"),
		"storageLimit":       recordOrEmpty(record, "storageLimit"),
		"description":        toString(record["description"]),
		"descUpdate":         toString(record["descUpdate"]),
		"sourceSheet":        toString(record["sourceSheet"]),
		"sourceFile":         firstNonEmpty(sourceFile, toString(record["sourceFile"])),
	}

	existing, getErr := h.tm.GetByIDRaw(r.Context(), pid)

	if errors.Is(getErr, repository.ErrNotFound) {
		doc["limits"] = parsers.CloneRangeLikeValue(doc["range"])
		doc["ignore_limit_check"] = false
		doc["ignore_change_detection"] = false
		doc["ignore_chain_comparision"] = false
		doc["available_chains"] = []int{1, 2}
		doc["createdAt"] = now

		if err := h.tm.SaveDoc(r.Context(), pid, sub, doc); err != nil {
			return "", err
		}
		return "inserted", nil
	}
	if getErr != nil {
		return "", getErr
	}

	if existing == nil {
		existing = map[string]any{}
	}

	changes := map[string]any{}
	oldMnemonic := toString(existing["cdbMnemonic"])
	newMnemonic := toString(doc["cdbMnemonic"])
	if oldMnemonic != "" && oldMnemonic != newMnemonic {
		changes["mnemonic"] = map[string]any{oldMnemonic: newMnemonic}
	}
	oldDigitalStatus := toString(existing["digitalStatus"])
	newDigitalStatus := toString(doc["digitalStatus"])
	if oldDigitalStatus != "" && oldDigitalStatus != newDigitalStatus {
		changes["digital_tm"] = map[string]any{oldDigitalStatus: newDigitalStatus}
	}

	doc["limits"] = parsers.CloneRangeLikeValue(doc["range"])
	if _, ok := existing["ignore_limit_check"]; ok {
		doc["ignore_limit_check"] = existing["ignore_limit_check"]
	} else {
		doc["ignore_limit_check"] = false
	}
	if _, ok := existing["ignore_change_detection"]; ok {
		doc["ignore_change_detection"] = existing["ignore_change_detection"]
	} else {
		doc["ignore_change_detection"] = false
	}
	if _, ok := existing["ignore_chain_comparision"]; ok {
		doc["ignore_chain_comparision"] = existing["ignore_chain_comparision"]
	} else {
		doc["ignore_chain_comparision"] = false
	}
	if _, ok := existing["available_chains"]; ok {
		doc["available_chains"] = existing["available_chains"]
	} else {
		doc["available_chains"] = []int{1, 2}
	}
	doc["updatedAt"] = now

	if err := h.tm.SaveDoc(r.Context(), pid, sub, doc); err != nil {
		return "", err
	}

	if len(changes) > 0 {
		changeEntry := map[string]any{"timestamp": now, "changes": changes}
		if err := h.tm.AppendHistory(r.Context(), pid, changeEntry); err != nil {
			h.logger.Warn("tm change history append failed", "pid", pid, "error", err)
		}
	}

	return "updated", nil
}

func jsonDecode(r *http.Request, out any) error {
	dec := json.NewDecoder(r.Body)
	return dec.Decode(out)
}

