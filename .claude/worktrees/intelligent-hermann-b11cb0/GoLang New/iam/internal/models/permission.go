package models

import "strings"

// PolicyRule is a single Casbin (role, resource, action) tuple.
type PolicyRule struct {
	Role     string `json:"role"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// AddPermissionRequest is the POST body for adding an endpoint permission to a role.
type AddPermissionRequest struct {
	Role     string `json:"role"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// RemovePermissionRequest is the DELETE body for removing an endpoint permission.
type RemovePermissionRequest struct {
	Role     string `json:"role"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// PermissionResource describes an API resource and its supported actions.
type PermissionResource struct {
	Resource    string   `json:"resource"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
}

// PermStorageKey converts a resource + action pair to the storage format used in Role.Permissions.
// Example: PermStorageKey("iam:roles", "write") == "iam:roles:write"
func PermStorageKey(resource, action string) string {
	return resource + ":" + action
}

// SplitPermission splits a stored permission key (e.g. "iam:roles:write") into
// resource ("iam:roles") and action ("write") by splitting at the LAST colon.
func SplitPermission(perm string) (resource, action string, ok bool) {
	idx := strings.LastIndex(perm, ":")
	if idx < 0 || idx == len(perm)-1 {
		return "", "", false
	}
	return perm[:idx], perm[idx+1:], true
}

// KnownResources is the catalog of configurable API endpoint resources.
// Super-admin uses this list when assigning permissions to operator/viewer roles.
var KnownResources = []PermissionResource{
	{Resource: "iam:users", Description: "IAM User management", Actions: []string{"read", "write"}},
	{Resource: "iam:roles", Description: "IAM Role management", Actions: []string{"read", "write"}},
	{Resource: "telemetry", Description: "Telemetry data", Actions: []string{"read", "write"}},
	{Resource: "commands", Description: "Telecommand management", Actions: []string{"read", "write"}},
	{Resource: "simulator", Description: "TM/TC Simulator", Actions: []string{"read", "write"}},
	{Resource: "sessions", Description: "Work sessions", Actions: []string{"read", "write"}},
	{Resource: "umacs", Description: "UMACS connections", Actions: []string{"read", "write"}},
	{Resource: "chainmon", Description: "Chain monitoring", Actions: []string{"read", "write"}},
}

// BuiltinRoles are the roles whose permissions are managed by the system (not editable via the permissions API).
var BuiltinRoles = map[string]bool{
	RoleSuperAdmin: true,
	RoleAdmin:      true,
}
