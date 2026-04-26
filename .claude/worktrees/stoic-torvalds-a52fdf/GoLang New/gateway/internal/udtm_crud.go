package gateway

import (
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository"
	"github.com/redis/go-redis/v9"
)

// UDTMCrudHandler serves SQLite-backed UD_TM CRUD endpoints.
type UDTMCrudHandler struct {
	udtm   repository.UDTMStore
	tm     repository.TMMnemonicStore
	rdb    *redis.Client
	logger *slog.Logger
}

// udtmSaveRequest is the body for POST /ud-tm.
type udtmSaveRequest struct {
	Rows          []models.UDTMRow `json:"rows"`
	Project       string           `json:"project"`
	CreatedBy     string           `json:"created_by"`
	ChangeMessage string           `json:"change_message"`
}

// GetUDTM handles GET /ud-tm
func (h *UDTMCrudHandler) GetUDTM(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		project = "default"
	}

	doc, err := h.udtm.Get(r.Context(), project)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"rows":           []models.UDTMRow{},
			"latest_version": 0,
		})
		return
	}
	if err != nil {
		h.logger.Error("user_telemetry find failed", "project", project, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rows":           doc.Rows,
		"latest_version": doc.Version,
	})
}

// SaveUDTM handles POST /ud-tm
func (h *UDTMCrudHandler) SaveUDTM(w http.ResponseWriter, r *http.Request) {
	var req udtmSaveRequest
	if err := jsonDecode(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Project == "" {
		req.Project = "default"
	}

	now := time.Now().UTC()
	for i := range req.Rows {
		req.Rows[i].LastUpdated = now
	}

	// 1. Determine next version
	nextVersion := 1
	if existing, err := h.udtm.Get(r.Context(), req.Project); err == nil {
		nextVersion = existing.Version + 1
	}

	// 2. Upsert current document
	doc := models.UserTelemetry{
		Project:   req.Project,
		Rows:      req.Rows,
		Version:   nextVersion,
		CreatedBy: req.CreatedBy,
		CreatedAt: now,
	}
	if err := h.udtm.Save(r.Context(), doc); err != nil {
		h.logger.Error("user_telemetry upsert failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	// 3. Insert version snapshot
	verDoc := models.UserTelemetryVersion{
		Project:       req.Project,
		Version:       nextVersion,
		Rows:          req.Rows,
		CreatedBy:     req.CreatedBy,
		CreatedAt:     now,
		ChangeMessage: req.ChangeMessage,
	}
	if err := h.udtm.SaveVersion(r.Context(), verDoc); err != nil {
		h.logger.Warn("user_telemetry_versions insert failed", "error", err)
	}

	// 4. Sync non-empty rows → tm_mnemonics with subsystem="UDTM"
	syncedCount := 0
	for _, row := range req.Rows {
		if row.Mnemonic == "" {
			continue
		}
		tmDoc := map[string]any{
			"_id":               row.Mnemonic,
			"subsystem":         "UDTM",
			"type":              row.Type,
			"unit":              row.Unit,
			"cdbMnemonic":       row.Description,
			"range":             []any{},
			"tolerance":         0.0,
			"enable_comparison": false,
			"enable_limit":      false,
			"enable_storage":    false,
		}
		if err := h.tm.SaveDoc(r.Context(), row.Mnemonic, "UDTM", tmDoc); err != nil {
			h.logger.Warn("tm_mnemonics udtm sync failed", "mnemonic", row.Mnemonic, "error", err)
			continue
		}
		syncedCount++
	}

	// 5. Publish
	if err := h.rdb.Publish(r.Context(), models.MdbTmMnemonicsUpdated, "udtm_updated").Err(); err != nil {
		h.logger.Warn("failed to publish MDB_TM_MNEMONICS_UPDATED", "error", err)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true, "version": nextVersion,
		"synced_mnemonics": syncedCount, "message": "saved",
	})
}

// GetUDTMVersions handles GET /ud-tm/versions
func (h *UDTMCrudHandler) GetUDTMVersions(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		project = "default"
	}

	versions, err := h.udtm.ListVersions(r.Context(), project)
	if err != nil {
		h.logger.Error("user_telemetry_versions query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	type versionSummary struct {
		Version       int       `json:"version"`
		CreatedBy     string    `json:"created_by"`
		CreatedAt     time.Time `json:"created_at"`
		ChangeMessage string    `json:"change_message"`
		RowCount      int       `json:"row_count"`
	}
	summaries := make([]versionSummary, len(versions))
	for i, v := range versions {
		summaries[i] = versionSummary{
			Version:       v.Version,
			CreatedBy:     v.CreatedBy,
			CreatedAt:     v.CreatedAt,
			ChangeMessage: v.ChangeMessage,
			RowCount:      len(v.Rows),
		}
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Version > summaries[j].Version
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{"versions": summaries})
}

// GetUDTMVersion handles GET /ud-tm/versions/{version}
func (h *UDTMCrudHandler) GetUDTMVersion(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		project = "default"
	}
	version, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid version"})
		return
	}

	doc, err := h.udtm.GetVersion(r.Context(), project, version)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "version not found"})
		return
	}
	if err != nil {
		h.logger.Error("user_telemetry_versions find failed", "version", version, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"version": doc})
}
