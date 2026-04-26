package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mainframe/tm-system/iam/internal/models"
	"github.com/mainframe/tm-system/iam/internal/repository"
	"github.com/mainframe/tm-system/iam/internal/service"
)

// RoleHandler handles CRUD operations on role resources.
type RoleHandler struct {
	roles  *service.RoleService
	logger *slog.Logger
}

// NewRoleHandler creates a RoleHandler.
func NewRoleHandler(roles *service.RoleService, logger *slog.Logger) *RoleHandler {
	return &RoleHandler{roles: roles, logger: logger}
}

// List godoc
// GET /iam/roles
// Requires: any authenticated user
func (h *RoleHandler) List(w http.ResponseWriter, r *http.Request) {
	roles, err := h.roles.List(r.Context())
	if err != nil {
		h.logger.Error("list roles error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, roles)
}

// Create godoc
// POST /iam/roles
// Requires: admin role
func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateRoleRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	role, err := h.roles.Create(r.Context(), &req)
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("create role error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, role)
}

// Get godoc
// GET /iam/roles/{id}
// Requires: any authenticated user
func (h *RoleHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	role, err := h.roles.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "role not found")
			return
		}
		h.logger.Error("get role error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, role)
}

// Update godoc
// PUT /iam/roles/{id}
// Requires: admin role
func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req models.UpdateRoleRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	role, err := h.roles.Update(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "role not found")
			return
		}
		h.logger.Error("update role error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, role)
}

// Delete godoc
// DELETE /iam/roles/{id}
// Requires: admin role
func (h *RoleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.roles.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "role not found")
			return
		}
		h.logger.Error("delete role error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
