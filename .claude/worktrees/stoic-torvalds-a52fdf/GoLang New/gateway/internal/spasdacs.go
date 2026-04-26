package gateway

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository"
)

type SpasdacsHandler struct {
	store  repository.SpasdacsStore
	logger *slog.Logger
}

// SpasdacsDiagram and SpasdacsMeta are defined in internal/models/spasdacs.go.
// Re-exported here as type aliases so existing handler code compiles without change.
type SpasdacsDiagram = models.SpasdacsDiagram
type SpasdacsMeta = models.SpasdacsMeta

func (h *SpasdacsHandler) GetDiagrams(w http.ResponseWriter, r *http.Request) {
	diagrams, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("spasdacs list query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	writeJSON(w, http.StatusOK, diagrams)
}

func (h *SpasdacsHandler) GetDiagram(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
		return
	}

	diagram, err := h.store.Get(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Diagram not found"})
		return
	}
	if err != nil {
		h.logger.Error("spasdacs get failed", "id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, diagram)
}

func (h *SpasdacsHandler) PostDiagram(w http.ResponseWriter, r *http.Request) {
	var diagram models.SpasdacsDiagram
	if err := json.NewDecoder(r.Body).Decode(&diagram); err != nil {
		h.logger.Error("spasdacs decode failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if diagram.ID == "" || diagram.Name == "" || diagram.ModelData == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required fields: id, name, modelData"})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Preserve createdAt from existing record if present.
	if existing, err := h.store.Get(r.Context(), diagram.ID); err == nil && existing.CreatedAt != "" {
		diagram.CreatedAt = existing.CreatedAt
	}
	if diagram.CreatedAt == "" {
		diagram.CreatedAt = now
	}
	diagram.UpdatedAt = now

	if err := h.store.Save(r.Context(), diagram); err != nil {
		h.logger.Error("spasdacs save failed", "id", diagram.ID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save diagram"})
		return
	}

	writeJSON(w, http.StatusOK, diagram)
}

func (h *SpasdacsHandler) PatchDiagram(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.Error("spasdacs patch decode failed", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	_, hasAutoViewInclude := payload["autoViewInclude"]
	_, hasAutoViewDuration := payload["autoViewDuration"]
	if !hasAutoViewInclude && !hasAutoViewDuration {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no patchable fields provided"})
		return
	}

	err := h.store.Patch(r.Context(), id, func(doc map[string]any) {
		if v, ok := payload["autoViewInclude"]; ok {
			doc["autoViewInclude"] = v
		}
		if v, ok := payload["autoViewDuration"]; ok {
			doc["autoViewDuration"] = v
		}
		doc["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
	})
	if errors.Is(err, repository.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Diagram not found"})
		return
	}
	if err != nil {
		h.logger.Error("spasdacs patch failed", "id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"success": "true"})
}

func (h *SpasdacsHandler) DeleteDiagram(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
		return
	}

	deleted, err := h.store.Delete(r.Context(), id)
	if err != nil {
		h.logger.Error("spasdacs delete failed", "id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if !deleted {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Diagram not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"success": "true", "message": "Diagram deleted"})
}
