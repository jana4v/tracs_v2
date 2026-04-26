package middleware

import (
	"net/http"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/go-chi/chi/v5"

	"github.com/mainframe/tm-system/iam/internal/models"
)

const iamCasbinModel = `
[request_definition]
r = sub, res, act

[policy_definition]
p = sub, res, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && (p.res == "*" || r.res == p.res) && (p.act == "*" || r.act == p.act)
`

// NewIAMEnforcer creates an in-memory Casbin enforcer.
// Built-in policies:
//
//	super_admin → iam:roles, iam:users, iam:permissions with action "*"
//	admin       → *, * (full access)
//
// Dynamic policies are loaded from the permissions stored in each role document.
// Roles in models.BuiltinRoles are skipped (their policies are hardcoded above).
func NewIAMEnforcer(roles []models.Role) (*casbin.Enforcer, error) {
	m, err := model.NewModelFromString(iamCasbinModel)
	if err != nil {
		return nil, err
	}

	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, err
	}

	// Built-in role policies.
	_, _ = e.AddPolicy(models.RoleSuperAdmin, "iam:roles", "*")
	_, _ = e.AddPolicy(models.RoleSuperAdmin, "iam:users", "*")
	_, _ = e.AddPolicy(models.RoleSuperAdmin, "iam:permissions", "*")
	_, _ = e.AddPolicy(models.RoleAdmin, "*", "*")

	// Load dynamic policies from role documents (operator, viewer, and any custom roles).
	for _, role := range roles {
		if models.BuiltinRoles[role.Name] {
			continue // built-in policies are fixed above
		}
		for _, perm := range role.Permissions {
			if perm == "*" {
				continue // skip wildcard stored in admin doc
			}
			resource, action, ok := models.SplitPermission(perm)
			if !ok {
				continue
			}
			_, _ = e.AddPolicy(role.Name, resource, action)
		}
	}

	return e, nil
}

// Authorizer enforces resource/action policies for authenticated callers.
type Authorizer struct {
	enforcer *casbin.Enforcer
}

// NewAuthorizer creates a Casbin-backed Authorizer middleware.
func NewAuthorizer(enforcer *casbin.Enforcer) *Authorizer {
	return &Authorizer{enforcer: enforcer}
}

// RequirePermission enforces that at least one caller role can access resource/action.
func (a *Authorizer) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writeUnauthorized(w, "missing authentication")
				return
			}

			for _, role := range claims.Roles {
				ok, err := a.enforcer.Enforce(role, resource, action)
				if err != nil {
					writeForbidden(w, "authorization engine failure")
					return
				}
				if ok {
					next.ServeHTTP(w, r)
					return
				}
			}

			writeForbidden(w, "insufficient permissions")
		})
	}
}

// RequireSelfOrPermission allows access to own resource, else checks Casbin permission.
func (a *Authorizer) RequireSelfOrPermission(resource, action, idParam string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writeUnauthorized(w, "missing authentication")
				return
			}

			if resourceID := chi.URLParam(r, idParam); resourceID != "" && resourceID == claims.UserID {
				next.ServeHTTP(w, r)
				return
			}

			for _, role := range claims.Roles {
				ok, err := a.enforcer.Enforce(role, resource, action)
				if err != nil {
					writeForbidden(w, "authorization engine failure")
					return
				}
				if ok {
					next.ServeHTTP(w, r)
					return
				}
			}

			writeForbidden(w, "insufficient permissions")
		})
	}
}

// RequireSuperAdmin enforces that the caller holds the super_admin role in their JWT claims.
// This is a direct role check used to guard permission-management endpoints.
func (a *Authorizer) RequireSuperAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writeUnauthorized(w, "missing authentication")
				return
			}
			for _, role := range claims.Roles {
				if role == models.RoleSuperAdmin {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeForbidden(w, "super_admin role required")
		})
	}
}
