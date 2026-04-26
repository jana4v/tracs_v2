package gateway

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository"
	"github.com/redis/go-redis/v9"
)

// DTMCrudHandler serves SQLite-backed DTM procedure CRUD endpoints.
type DTMCrudHandler struct {
	dtm    repository.DTMStore
	tm     repository.TMMnemonicStore
	rdb    *redis.Client
	logger *slog.Logger
}

// dtmSaveRequest is the body for POST /dtm/procedures.
type dtmSaveRequest struct {
	Rows      []models.DTMProcedureRow `json:"rows"`
	Project   string                   `json:"project"`
	CreatedBy string                   `json:"created_by"`
}

// GetDTMProcedures handles GET /dtm/procedures
// Optional query param: ?project=default
func (h *DTMCrudHandler) GetDTMProcedures(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		project = "default"
	}

	doc, err := h.dtm.Get(r.Context(), project)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"rows":    []models.DTMProcedureRow{},
			"version": 0,
		})
		return
	}
	if err != nil {
		h.logger.Error("dtm_procedures find failed", "project", project, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rows":    doc.Rows,
		"version": doc.Version,
	})
}

// SaveDTMProcedures handles POST /dtm/procedures
func (h *DTMCrudHandler) SaveDTMProcedures(w http.ResponseWriter, r *http.Request) {
	var req dtmSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Project == "" {
		req.Project = "default"
	}

	now := time.Now().UTC()

	// 1. Determine next version
	nextVersion := 1
	if existing, err := h.dtm.Get(r.Context(), req.Project); err == nil {
		nextVersion = existing.Version + 1
	}

	// 2. Upsert current document
	doc := models.DTMProcedures{
		Project:   req.Project,
		Rows:      req.Rows,
		Version:   nextVersion,
		CreatedBy: req.CreatedBy,
		CreatedAt: now,
	}
	if err := h.dtm.Save(r.Context(), doc); err != nil {
		h.logger.Error("dtm_procedures upsert failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	// 3. Sync enabled rows with non-empty mnemonic → tm_mnemonics with subsystem="DTM"
	syncedCount := 0
	for _, row := range req.Rows {
		if row.Mnemonic == "" || !row.Enabled {
			continue
		}

		rangeVal := row.Range

		tmDoc := map[string]any{
			"_id":               row.Mnemonic,
			"subsystem":         "DTM",
			"type":              row.Type,
			"unit":              row.Unit,
			"cdbMnemonic":       row.Description,
			"range":             rangeVal,
			"tolerance":         row.Tolerance,
			"enable_comparison": false,
			"enable_limit":      true,
			"enable_storage":    false,
		}
		if err := h.tm.SaveDoc(r.Context(), row.Mnemonic, "DTM", tmDoc); err != nil {
			h.logger.Warn("tm_mnemonics dtm sync failed", "mnemonic", row.Mnemonic, "error", err)
			continue
		}
		syncedCount++
	}

	// 4. Publish events
	if err := h.rdb.Publish(r.Context(), models.MdbTmMnemonicsUpdated, "dtm_updated").Err(); err != nil {
		h.logger.Warn("failed to publish MDB_TM_MNEMONICS_UPDATED", "error", err)
	}
	if err := h.rdb.Publish(r.Context(), models.DTMProceduresUpdated, "").Err(); err != nil {
		h.logger.Warn("failed to publish DTM_PROCEDURES_UPDATED", "error", err)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":          true,
		"version":          nextVersion,
		"synced_mnemonics": syncedCount,
	})
}
