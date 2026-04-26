package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mainframe/tm-system/iam/internal/models"
	"github.com/mainframe/tm-system/iam/internal/repository"
)

// UserService handles business logic for user management.
type UserService struct {
	users repository.UserStore
	roles repository.RoleStore
}

// NewUserService creates a UserService.
func NewUserService(users repository.UserStore, roles repository.RoleStore) *UserService {
	return &UserService{users: users, roles: roles}
}

// Create validates the request, hashes the password, and persists a new user.
func (s *UserService) Create(ctx context.Context, req *models.CreateUserRequest) (*models.UserResponse, error) {
	if err := validateCreateUser(req); err != nil {
		return nil, err
	}

	// Validate that all requested roles exist.
	for _, roleName := range req.Roles {
		if _, err := s.roles.FindByName(ctx, roleName); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, fmt.Errorf("role %q does not exist", roleName)
			}
			return nil, err
		}
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	isActive := true
	u := &models.User{
		Username:     strings.TrimSpace(req.Username),
		Email:        strings.TrimSpace(req.Email),
		PasswordHash: hash,
		FullName:     strings.TrimSpace(req.FullName),
		Roles:        req.Roles,
		IsActive:     isActive,
	}

	if err := s.users.Create(ctx, u); err != nil {
		if errors.Is(err, repository.ErrDuplicateKey) {
			return nil, fmt.Errorf("username or email already exists")
		}
		return nil, err
	}

	resp := u.ToResponse()
	return &resp, nil
}

// GetByID retrieves a single user by ID.
func (s *UserService) GetByID(ctx context.Context, id string) (*models.UserResponse, error) {
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := u.ToResponse()
	return &resp, nil
}

// List returns all users.
func (s *UserService) List(ctx context.Context) ([]models.UserResponse, error) {
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.UserResponse, len(users))
	for i, u := range users {
		out[i] = u.ToResponse()
	}
	return out, nil
}

// Update applies allowed field changes to a user.
func (s *UserService) Update(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.UserResponse, error) {
	fields := map[string]interface{}{}
	if req.Email != "" {
		fields["email"] = strings.TrimSpace(req.Email)
	}
	if req.FullName != "" {
		fields["full_name"] = strings.TrimSpace(req.FullName)
	}
	if req.IsActive != nil {
		fields["is_active"] = *req.IsActive
	}

	if len(fields) == 0 {
		return s.GetByID(ctx, id)
	}

	if err := s.users.Update(ctx, id, fields); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// Delete removes a user permanently.
func (s *UserService) Delete(ctx context.Context, id string) error {
	return s.users.Delete(ctx, id)
}

// AssignRoles replaces the roles slice on a user after validating each role exists.
func (s *UserService) AssignRoles(ctx context.Context, id string, roleNames []string) (*models.UserResponse, error) {
	for _, name := range roleNames {
		if _, err := s.roles.FindByName(ctx, name); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, fmt.Errorf("role %q does not exist", name)
			}
			return nil, err
		}
	}

	if err := s.users.Update(ctx, id, map[string]interface{}{"roles": roleNames}); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// --- validators --------------------------------------------------------------

func validateCreateUser(req *models.CreateUserRequest) error {
	if strings.TrimSpace(req.Username) == "" {
		return fmt.Errorf("username is required")
	}
	if len(req.Username) < 3 || len(req.Username) > 64 {
		return fmt.Errorf("username must be 3–64 characters")
	}
	if len(req.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
}
