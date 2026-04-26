package handlers

import (
	"log/slog"
	"net/http"

	"github.com/mainframe/tm-system/iam/internal/models"
	"github.com/mainframe/tm-system/iam/internal/service"
)

// PermissionHandler handles permission policy management endpoints.
type PermissionHandler struct {
	perms  *service.PermissionService
	logger *slog.Logger
}

// NewPermissionHandler creates a PermissionHandler.
func NewPermissionHandler(perms *service.PermissionService, logger *slog.Logger) *PermissionHandler {
	return &PermissionHandler{perms: perms, logger: logger}
}

// List godoc
// GET /iam/permissions
// Returns all current Casbin policy rules.
func (h *PermissionHandler) List(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.perms.ListPolicies())
}

// Resources godoc
// GET /iam/permissions/resources
// Returns the catalog of configurable endpoint resources.
func (h *PermissionHandler) Resources(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, models.KnownResources)
}

// Add godoc
// POST /iam/permissions
// Body: AddPermissionRequest{role, resource, action}
func (h *PermissionHandler) Add(w http.ResponseWriter, r *http.Request) {
	var req models.AddPermissionRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := h.perms.AddPermission(r.Context(), &req); err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("add permission error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

// Remove godoc
// DELETE /iam/permissions
// Body: RemovePermissionRequest{role, resource, action}
func (h *PermissionHandler) Remove(w http.ResponseWriter, r *http.Request) {
	var req models.RemovePermissionRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := h.perms.RemovePermission(r.Context(), &req); err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("remove permission error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
