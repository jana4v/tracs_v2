package gateway

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository"
	"github.com/redis/go-redis/v9"
)

type TelecommandUploadHandler struct {
	tc     repository.TCMnemonicStore
	rdb    *redis.Client
	logger *slog.Logger
}

type telecommandUploadRequest struct {
	Filename string   `json:"filename"`
	Data     string   `json:"data"`
	DataPart []string `json:"dataPart"`
}

type telecommandUploadStats struct {
	Total    int      `json:"total"`
	Inserted int      `json:"inserted"`
	Updated  int      `json:"updated"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

func (h *TelecommandUploadHandler) UploadTelecommand(w http.ResponseWriter, r *http.Request) {
	var req telecommandUploadRequest
	if err := jsonDecode(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "invalid request body",
		})
		return
	}

	req.Filename = strings.TrimSpace(req.Filename)
	req.Data = strings.TrimSpace(req.Data)
	if req.DataPart == nil {
		req.DataPart = []string{}
	}
	if req.Filename == "" || req.Data == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "No file data provided. Send JSON with 'filename' and 'data' (base64).",
		})
		return
	}

	ext := strings.ToLower(filepath.Ext(req.Filename))
	if ext != ".out" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Unsupported file type: %s. Supported: .out", ext),
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

	tmpFile, err := os.CreateTemp("", "tc_upload_"+"*"+ext)
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

	h.logger.Info("TC upload saved temp file", "path", tmpPath, "bytes", len(fileBytes), "filename", req.Filename)

	records, err := parseTCOUTFile(tmpPath)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Failed to parse OUT: %v", err),
		})
		return
	}

	h.logger.Info("TC upload parsed records", "filename", req.Filename, "records", len(records))

	stats := h.upsertTCCommandsBulk(r, records, req.Filename, req.DataPart)
	if stats.Inserted > 0 || stats.Updated > 0 {
		if err := h.rdb.Publish(r.Context(), models.MdbTcCommandsUpdated, "telecommand_upload").Err(); err != nil {
			h.logger.Warn("failed to publish MDB_TC_COMMANDS_UPDATED", "error", err)
		}
	}

	status := http.StatusOK
	if len(stats.Errors) > 0 {
		status = http.StatusMultiStatus
	}
	writeJSON(w, status, map[string]any{
		"success":  true,
		"filename": req.Filename,
		"stats":    stats,
	})
}

func parseTCOUTFile(path string) ([]map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lines := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sourceFile := filepath.Base(path)
	records := make([]map[string]any, 0)

	// Skip header lines (typically first 7 lines)
	startLine := 7
	for i := startLine; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.Contains(line, "---") {
			continue
		}

		tokens := strings.Fields(line)
		if len(tokens) < 5 {
			continue
		}

		// First 6 meaningful columns: CmdId, CommandDescription, CommandCode, Type, Field5, Field6
		cmdId := tokens[0]

		// CmdId pattern check (e.g., ACM53801, AOC44017)
		if !isTelecommandId(cmdId) {
			continue
		}

		// Extract first 6 columns
		subsystem := extractSubsystemFromCmdId(cmdId)
		rec := map[string]any{
			"cmdId":       cmdId,
			"commandDesc": tokens[1],
			"commandCode": tokens[2],
			"type":        tokens[3], // Usually "Normal"
			"decoder":     tokens[4], // Decoder field
			"subsystem":   subsystem, // Extract from cmdId (e.g., "AOC44037" → "AOC")
			"sourceFile":  sourceFile,
		}

		// Optional 6th field if available (mapId)
		if len(tokens) > 5 {
			rec["mapId"] = tokens[5]
		}

		records = append(records, rec)
	}

	return records, nil
}

func isTelecommandId(s string) bool {
	// TC IDs typically start with 3 uppercase letters followed by 5 digits
	// Examples: ACM53801, AOC44017, etc.
	if len(s) < 8 {
		return false
	}
	for i := 0; i < 3; i++ {
		if s[i] < 'A' || s[i] > 'Z' {
			return false
		}
	}
	for i := 3; i < 8; i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func extractSubsystemFromCmdId(cmdId string) string {
	// Extract prefix letters from CmdId (strip trailing digits)
	// Example: "AOC44037" → "AOC"
	for i := 0; i < len(cmdId); i++ {
		if cmdId[i] < 'A' || cmdId[i] > 'Z' {
			return cmdId[:i]
		}
	}
	return cmdId
}

func (h *TelecommandUploadHandler) upsertTCCommandsBulk(r *http.Request, records []map[string]any, sourceFile string, dataPart []string) telecommandUploadStats {
	stats := telecommandUploadStats{
		Total:  len(records),
		Errors: make([]string, 0),
	}

	for _, rec := range records {
		result, err := h.upsertTCCommand(r, rec, sourceFile, dataPart)
		if err != nil {
			cmdId := toString(rec["cmdId"])
			if cmdId == "" {
				cmdId = "UNKNOWN"
			}
			stats.Errors = append(stats.Errors, fmt.Sprintf("[%s] %v", cmdId, err))
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

	h.logger.Info("TC bulk upsert completed",
		"total", stats.Total,
		"inserted", stats.Inserted,
		"updated", stats.Updated,
		"skipped", stats.Skipped,
		"errors", len(stats.Errors),
	)

	return stats
}

func (h *TelecommandUploadHandler) upsertTCCommand(r *http.Request, record map[string]any, sourceFile string, dataPart []string) (string, error) {
	cmdId := toString(record["cmdId"])
	if cmdId == "" {
		return "skipped", nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	subsystem := toString(record["subsystem"])

	doc := map[string]any{
		"cmdId":      cmdId,
		"cmdDesc":    toString(record["commandDesc"]),
		"cmdCode":    toString(record["commandCode"]),
		"type":       toString(record["type"]),
		"decoder":    toString(record["decoder"]),
		"mapId":      recordOrEmpty(record, "mapId"),
		"subsystem":  subsystem,
		"sourceFile": firstNonEmpty(sourceFile, toString(record["sourceFile"])),
		"dataPart":   dataPart,
	}

	// Check if record exists
	existing, getErr := h.tc.GetByIDRaw(r.Context(), cmdId)

	if errors.Is(getErr, repository.ErrNotFound) {
		doc["_id"] = cmdId
		doc["createdAt"] = now
		if ierr := h.tc.SaveDoc(r.Context(), cmdId, subsystem, doc); ierr != nil {
			return "", ierr
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

	oldDesc := toString(existing["cmdDesc"])
	newDesc := toString(doc["cmdDesc"])
	if oldDesc != "" && oldDesc != newDesc {
		changes["description"] = map[string]any{oldDesc: newDesc}
	}

	oldCode := toString(existing["cmdCode"])
	newCode := toString(doc["cmdCode"])
	if oldCode != "" && oldCode != newCode {
		changes["code"] = map[string]any{oldCode: newCode}
	}

	doc["updatedAt"] = now
	// Preserve createdAt from existing
	if ca, ok := existing["createdAt"]; ok {
		doc["createdAt"] = ca
	}

	if uerr := h.tc.SaveDoc(r.Context(), cmdId, subsystem, doc); uerr != nil {
		return "", uerr
	}

	if len(changes) > 0 {
		changeEntry := map[string]any{
			"timestamp": now,
			"changes":   changes,
		}
		if aerr := h.tc.AppendHistory(r.Context(), cmdId, changeEntry); aerr != nil {
			h.logger.Warn("tc history append failed", "cmdId", cmdId, "error", aerr)
		}
	}

	return "updated", nil
}
