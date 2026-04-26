package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mainframe/tm-system/iam/internal/middleware"
	"github.com/mainframe/tm-system/iam/internal/models"
	"github.com/mainframe/tm-system/iam/internal/repository"
	"github.com/mainframe/tm-system/iam/internal/service"
)

// UserHandler handles CRUD operations on user resources.
type UserHandler struct {
	users  *service.UserService
	logger *slog.Logger
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(users *service.UserService, logger *slog.Logger) *UserHandler {
	return &UserHandler{users: users, logger: logger}
}

// List godoc
// GET /iam/users
// Requires: admin role
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		h.logger.Error("list users error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// Create godoc
// POST /iam/users
// Requires: admin role
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, err := h.users.Create(r.Context(), &req)
	if err != nil {
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("create user error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

// Get godoc
// GET /iam/users/{id}
// Requires: admin role OR same user
func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Allow users to retrieve their own profile.
	if !isAdminOrSelf(r, id) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	user, err := h.users.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("get user error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// Update godoc
// PUT /iam/users/{id}
// Requires: admin role OR same user
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if !isAdminOrSelf(r, id) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	var req models.UpdateUserRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, err := h.users.Update(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("update user error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// Delete godoc
// DELETE /iam/users/{id}
// Requires: admin role
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.users.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Error("delete user error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AssignRoles godoc
// PUT /iam/users/{id}/roles
// Requires: admin role
func (h *UserHandler) AssignRoles(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req models.AssignRolesRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, err := h.users.AssignRoles(r.Context(), id, req.Roles)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		if isValidationError(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("assign roles error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// --- helpers -----------------------------------------------------------------

// isAdminOrSelf returns true if the caller is an admin or is accessing their own resource.
func isAdminOrSelf(r *http.Request, resourceUserID string) bool {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return false
	}
	for _, role := range claims.Roles {
		if role == "admin" {
			return true
		}
	}
	return claims.UserID == resourceUserID
}

// isValidationError returns true for known user-input validation errors
// (those that should be returned as 400 Bad Request).
func isValidationError(err error) bool {
	// Simple heuristic: service validation errors are plain fmt.Errorf strings
	// without wrapping sentinel errors that map to 5xx.
	return err != nil &&
		!errors.Is(err, repository.ErrNotFound) &&
		!errors.Is(err, repository.ErrDuplicateKey)
}
