package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/mainframe/tm-system/iam/internal/service"
)

// contextKey is a private type to avoid key collisions in context values.
type contextKey string

const claimsKey contextKey = "iam_claims"

// Authenticator validates Bearer tokens on protected routes.
type Authenticator struct {
	auth *service.AuthService
}

// NewAuthenticator creates an Authenticator middleware.
func NewAuthenticator(auth *service.AuthService) *Authenticator {
	return &Authenticator{auth: auth}
}

// RequireAuth is an http.Handler middleware that enforces a valid JWT.
// On success it stores the parsed Claims in the request context.
func (a *Authenticator) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := extractBearer(r)
		if err != nil {
			writeUnauthorized(w, err.Error())
			return
		}

		claims, err := a.auth.ValidateAccessToken(token)
		if err != nil {
			if errors.Is(err, service.ErrTokenExpired) {
				writeUnauthorized(w, "token expired")
				return
			}
			writeUnauthorized(w, "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ClaimsFromContext retrieves the JWT Claims stored by RequireAuth.
func ClaimsFromContext(ctx context.Context) *service.Claims {
	v, _ := ctx.Value(claimsKey).(*service.Claims)
	return v
}

// --- helpers -----------------------------------------------------------------

func extractBearer(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", errors.New("missing Authorization header")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("authorization header must be: Bearer <token>")
	}
	if strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("bearer token is empty")
	}
	return parts[1], nil
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Bearer realm="mainframe-iam"`)
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"` + msg + `"}`)) //nolint:errcheck
}

func writeForbidden(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"error":"` + msg + `"}`)) //nolint:errcheck
}
