package iam

import (
	"log/slog"

	"github.com/casbin/casbin/v2"
	"github.com/go-chi/chi/v5"

	"github.com/mainframe/tm-system/iam/internal/handlers"
	"github.com/mainframe/tm-system/iam/internal/middleware"
	"github.com/mainframe/tm-system/iam/internal/service"
)

// RegisterRoutes mounts all IAM API routes under /iam on the given chi.Router.
//
// Route overview:
//
//	Public       POST   /iam/auth/login
//	Public       POST   /iam/auth/refresh
//
//	Auth         POST   /iam/auth/logout
//	Auth         GET    /iam/auth/me
//	Auth         POST   /iam/auth/change-password
//
//	Auth         GET    /iam/roles
//	Auth         GET    /iam/roles/{id}
//	SuperAdmin   POST   /iam/roles
//	SuperAdmin   PUT    /iam/roles/{id}
//	SuperAdmin   DELETE /iam/roles/{id}
//
//	Admin        GET    /iam/users
//	Admin        POST   /iam/users
//	Auth         GET    /iam/users/{id}    (admin or self)
//	Auth         PUT    /iam/users/{id}    (admin or self)
//	Admin        DELETE /iam/users/{id}
//	Admin        PUT    /iam/users/{id}/roles
//
//	SuperAdmin   GET    /iam/permissions
//	SuperAdmin   POST   /iam/permissions
//	SuperAdmin   DELETE /iam/permissions
//	SuperAdmin   GET    /iam/permissions/resources
func RegisterRoutes(
	r chi.Router,
	authSvc *service.AuthService,
	userSvc *service.UserService,
	roleSvc *service.RoleService,
	permSvc *service.PermissionService,
	authz *casbin.Enforcer,
	logger *slog.Logger,
) {
	authn := middleware.NewAuthenticator(authSvc)
	authorizer := middleware.NewAuthorizer(authz)

	authH := handlers.NewAuthHandler(authSvc, userSvc, logger)
	userH := handlers.NewUserHandler(userSvc, logger)
	roleH := handlers.NewRoleHandler(roleSvc, logger)
	permH := handlers.NewPermissionHandler(permSvc, logger)

	r.Route("/iam", func(r chi.Router) {
		// ── Auth ──────────────────────────────────────────────────────────
		r.Route("/auth", func(r chi.Router) {
			// Public endpoints (no token required)
			r.Post("/login", authH.Login)
			r.Post("/refresh", authH.Refresh)

			// Protected endpoints
			r.Group(func(r chi.Router) {
				r.Use(authn.RequireAuth)
				r.Post("/logout", authH.Logout)
				r.Get("/me", authH.Me)
				r.Post("/change-password", authH.ChangePassword)
			})
		})

		// ── Roles ─────────────────────────────────────────────────────────
		r.Route("/roles", func(r chi.Router) {
			r.Use(authn.RequireAuth)

			r.Get("/", roleH.List)
			r.Get("/{id}", roleH.Get)

			// Super-admin-only mutations
			r.Group(func(r chi.Router) {
				r.Use(authorizer.RequireSuperAdmin())
				r.Post("/", roleH.Create)
				r.Put("/{id}", roleH.Update)
				r.Delete("/{id}", roleH.Delete)
			})
		})

		// ── Users ─────────────────────────────────────────────────────────
		r.Route("/users", func(r chi.Router) {
			r.Use(authn.RequireAuth)

			// Casbin-controlled: list/read/create/delete/assign roles.
			r.Group(func(r chi.Router) {
				r.Use(authorizer.RequirePermission("iam:users", "read"))
				r.Get("/", userH.List)
			})

			r.Group(func(r chi.Router) {
				r.Use(authorizer.RequirePermission("iam:users", "write"))
				r.Post("/", userH.Create)
				r.Delete("/{id}", userH.Delete)
				r.Put("/{id}/roles", userH.AssignRoles)
			})

			// Admin or self: read and update
			r.With(authorizer.RequireSelfOrPermission("iam:users", "read", "id")).Get("/{id}", userH.Get)
			r.With(authorizer.RequireSelfOrPermission("iam:users", "write", "id")).Put("/{id}", userH.Update)
		})

		// ── Permissions (super_admin only) ────────────────────────────────
		r.Route("/permissions", func(r chi.Router) {
			r.Use(authn.RequireAuth)
			r.Use(authorizer.RequireSuperAdmin())

			r.Get("/", permH.List)
			r.Get("/resources", permH.Resources)
			r.Post("/", permH.Add)
			r.Delete("/", permH.Remove)
		})
	})
}
