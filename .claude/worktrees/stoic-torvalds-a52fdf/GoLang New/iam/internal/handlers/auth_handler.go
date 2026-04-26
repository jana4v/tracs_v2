package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/mainframe/tm-system/iam/internal/middleware"
	"github.com/mainframe/tm-system/iam/internal/models"
	"github.com/mainframe/tm-system/iam/internal/service"
)

// AuthHandler handles all authentication endpoints.
type AuthHandler struct {
	auth   *service.AuthService
	users  *service.UserService
	logger *slog.Logger
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(auth *service.AuthService, users *service.UserService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, users: users, logger: logger}
}

// Login godoc
// POST /iam/auth/login
// Body: { "username": "...", "password": "..." }
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	resp, err := h.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid username or password")
		case errors.Is(err, service.ErrAccountDisabled):
			writeError(w, http.StatusForbidden, "account is disabled")
		default:
			h.logger.Error("login error", "error", err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Refresh godoc
// POST /iam/auth/refresh
// Body: { "refresh_token": "..." }
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	resp, err := h.auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTokenRevoked), errors.Is(err, service.ErrTokenExpired):
			writeError(w, http.StatusUnauthorized, "refresh token is invalid or expired")
		case errors.Is(err, service.ErrAccountDisabled):
			writeError(w, http.StatusForbidden, "account is disabled")
		default:
			h.logger.Error("refresh error", "error", err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// Logout godoc
// POST /iam/auth/logout
// Body: { "refresh_token": "..." }
// Requires: Bearer token
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := h.auth.Logout(r.Context(), req.RefreshToken); err != nil {
		h.logger.Error("logout error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// Me godoc
// GET /iam/auth/me
// Requires: Bearer token
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "missing authentication")
		return
	}

	user, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// ChangePassword godoc
// POST /iam/auth/change-password
// Body: { "old_password": "...", "new_password": "..." }
// Requires: Bearer token
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "missing authentication")
		return
	}

	var req models.ChangePasswordRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "old_password and new_password are required")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "new_password must be at least 8 characters")
		return
	}

	if err := h.auth.ChangePassword(r.Context(), claims.UserID, req.OldPassword, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "old password is incorrect")
			return
		}
		h.logger.Error("change password error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password changed successfully"})
}
