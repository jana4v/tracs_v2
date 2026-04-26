package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mainframe/tm-system/iam/internal/models"
	"github.com/mainframe/tm-system/iam/internal/repository"
)

// RoleService handles business logic for role management.
type RoleService struct {
	roles repository.RoleStore
}

// NewRoleService creates a RoleService.
func NewRoleService(roles repository.RoleStore) *RoleService {
	return &RoleService{roles: roles}
}

// Create validates and persists a new role.
func (s *RoleService) Create(ctx context.Context, req *models.CreateRoleRequest) (*models.RoleResponse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("role name is required")
	}

	role := &models.Role{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Permissions: req.Permissions,
	}

	if err := s.roles.Create(ctx, role); err != nil {
		if errors.Is(err, repository.ErrDuplicateKey) {
			return nil, fmt.Errorf("role %q already exists", req.Name)
		}
		return nil, err
	}

	resp := role.ToResponse()
	return &resp, nil
}

// GetByID retrieves a single role.
func (s *RoleService) GetByID(ctx context.Context, id string) (*models.RoleResponse, error) {
	role, err := s.roles.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := role.ToResponse()
	return &resp, nil
}

// List returns all roles.
func (s *RoleService) List(ctx context.Context) ([]models.RoleResponse, error) {
	roles, err := s.roles.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.RoleResponse, len(roles))
	for i, r := range roles {
		out[i] = r.ToResponse()
	}
	return out, nil
}

// ListRaw returns all role documents (internal use, e.g. Casbin policy loading).
func (s *RoleService) ListRaw(ctx context.Context) ([]models.Role, error) {
	return s.roles.List(ctx)
}

// Update applies allowed field changes to a role.
func (s *RoleService) Update(ctx context.Context, id string, req *models.UpdateRoleRequest) (*models.RoleResponse, error) {
	fields := map[string]interface{}{}
	if req.Description != "" {
		fields["description"] = strings.TrimSpace(req.Description)
	}
	if req.Permissions != nil {
		fields["permissions"] = req.Permissions
	}

	if len(fields) == 0 {
		return s.GetByID(ctx, id)
	}

	if err := s.roles.Update(ctx, id, fields); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// Delete removes a role permanently.
// Note: this does NOT cascade-remove the role name from existing users.
func (s *RoleService) Delete(ctx context.Context, id string) error {
	return s.roles.Delete(ctx, id)
}
