package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"

	"github.com/mainframe/tm-system/iam/internal/models"
	"github.com/mainframe/tm-system/iam/internal/repository"
)

// PermissionService manages runtime endpoint permission policies for configurable roles.
// It keeps the Casbin in-memory enforcer and the SQLite Role documents in sync.
type PermissionService struct {
	enforcer *casbin.Enforcer
	roles    repository.RoleStore
}

// NewPermissionService creates a PermissionService.
func NewPermissionService(enforcer *casbin.Enforcer, roles repository.RoleStore) *PermissionService {
	return &PermissionService{enforcer: enforcer, roles: roles}
}

// ListPolicies returns all current Casbin policy rules.
func (s *PermissionService) ListPolicies() []models.PolicyRule {
	all, _ := s.enforcer.GetPolicy()
	rules := make([]models.PolicyRule, 0, len(all))
	for _, p := range all {
		if len(p) >= 3 {
			rules = append(rules, models.PolicyRule{
				Role:     p[0],
				Resource: p[1],
				Action:   p[2],
			})
		}
	}
	return rules
}

// AddPermission adds a (role, resource, action) policy to Casbin and persists it
// in the role's MongoDB permissions array. Built-in roles (super_admin, admin)
// cannot be modified through this API.
func (s *PermissionService) AddPermission(ctx context.Context, req *models.AddPermissionRequest) error {
	role := strings.TrimSpace(req.Role)
	resource := strings.TrimSpace(req.Resource)
	action := strings.TrimSpace(req.Action)

	if role == "" || resource == "" || action == "" {
		return fmt.Errorf("role, resource and action are required")
	}
	if models.BuiltinRoles[role] {
		return fmt.Errorf("cannot modify built-in role %q", role)
	}

	// Add to in-memory Casbin enforcer.
	if _, err := s.enforcer.AddPolicy(role, resource, action); err != nil {
		return fmt.Errorf("add casbin policy: %w", err)
	}

	// Persist in the MongoDB role document.
	roleDoc, err := s.roles.FindByName(ctx, role)
	if err != nil {
		return fmt.Errorf("role %q not found: %w", role, err)
	}

	permKey := models.PermStorageKey(resource, action)
	for _, p := range roleDoc.Permissions {
		if p == permKey {
			return nil // already persisted
		}
	}

	updated := append(roleDoc.Permissions, permKey) //nolint:gocritic
	return s.roles.Update(ctx, roleDoc.ID, map[string]interface{}{"permissions": updated})
}

// RemovePermission removes a (role, resource, action) policy from Casbin and from
// the role's MongoDB permissions array.
func (s *PermissionService) RemovePermission(ctx context.Context, req *models.RemovePermissionRequest) error {
	role := strings.TrimSpace(req.Role)
	resource := strings.TrimSpace(req.Resource)
	action := strings.TrimSpace(req.Action)

	if role == "" || resource == "" || action == "" {
		return fmt.Errorf("role, resource and action are required")
	}
	if models.BuiltinRoles[role] {
		return fmt.Errorf("cannot modify built-in role %q", role)
	}

	// Remove from in-memory Casbin enforcer.
	if _, err := s.enforcer.RemovePolicy(role, resource, action); err != nil {
		return fmt.Errorf("remove casbin policy: %w", err)
	}

	// Remove from the MongoDB role document.
	roleDoc, err := s.roles.FindByName(ctx, role)
	if err != nil {
		return fmt.Errorf("role %q not found: %w", role, err)
	}

	permKey := models.PermStorageKey(resource, action)
	perms := make([]string, 0, len(roleDoc.Permissions))
	for _, p := range roleDoc.Permissions {
		if p != permKey {
			perms = append(perms, p)
		}
	}

	return s.roles.Update(ctx, roleDoc.ID, map[string]interface{}{"permissions": perms})
}
