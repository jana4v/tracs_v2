package models

import "time"

// Role represents a named set of permissions stored in the iam_roles SQLite table.
type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoleResponse is the public view of a role.
type RoleResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse converts a Role to its public representation.
func (r *Role) ToResponse() RoleResponse {
	return RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Permissions: r.Permissions,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// CreateRoleRequest is the payload for creating a new role.
type CreateRoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// UpdateRoleRequest is the payload for updating an existing role.
type UpdateRoleRequest struct {
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// Well-known built-in role names.
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleOperator   = "operator"
	RoleViewer     = "viewer"
)

// DefaultRoles are seeded on first startup.
var DefaultRoles = []Role{
	{
		Name:        RoleSuperAdmin,
		Description: "IAM administrator – manages roles and endpoint permissions",
		Permissions: []string{"iam:roles:*", "iam:users:*", "iam:permissions:*"},
	},
	{
		Name:        RoleAdmin,
		Description: "Full system access across all services",
		Permissions: []string{"*"},
	},
	{
		Name:        RoleOperator,
		Description: "Read/write telemetry and telecommands",
		Permissions: []string{
			"telemetry:read",
			"telemetry:write",
			"commands:read",
			"commands:write",
			"simulator:read",
			"simulator:write",
		},
	},
	{
		Name:        RoleViewer,
		Description: "Read-only access to telemetry",
		Permissions: []string{
			"telemetry:read",
			"commands:read",
			"simulator:read",
		},
	},
}
