package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mainframe/tm-system/iam/internal/models"
	"github.com/mainframe/tm-system/iam/internal/repository"
)

// Seed bootstraps the IAM database with default roles and an admin user
// if neither exists yet. Safe to call on every startup.
func Seed(
	ctx context.Context,
	users repository.UserStore,
	roles repository.RoleStore,
	adminPassword string,
	logger *slog.Logger,
) error {
	// Seed default roles.
	for _, def := range models.DefaultRoles {
		if _, err := roles.FindByName(ctx, def.Name); err != nil {
			if !errors.Is(err, repository.ErrNotFound) {
				return fmt.Errorf("seed check role %s: %w", def.Name, err)
			}
			role := def // copy
			if err := roles.Create(ctx, &role); err != nil && !errors.Is(err, repository.ErrDuplicateKey) {
				return fmt.Errorf("seed role %s: %w", def.Name, err)
			}
			logger.Info("seeded role", "name", def.Name)
		}
	}

	// Ensure built-in admin/superadmin users exist.

	hash, err := HashPassword(adminPassword)
	if err != nil {
		return err
	}

	admin := &models.User{
		Username:     "admin",
		Email:        "admin@mainframe.local",
		PasswordHash: hash,
		FullName:     "System Administrator",
		Roles:        []string{models.RoleAdmin},
		IsActive:     true,
	}

	superAdmin := &models.User{
		Username:     "superadmin",
		Email:        "superadmin@mainframe.local",
		PasswordHash: hash,
		FullName:     "IAM Super Administrator",
		Roles:        []string{models.RoleSuperAdmin},
		IsActive:     true,
	}

	if _, err := users.FindByUsername(ctx, admin.Username); err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("seed check admin user: %w", err)
		}
		if err := users.Create(ctx, admin); err != nil && !errors.Is(err, repository.ErrDuplicateKey) {
			return fmt.Errorf("seed admin user: %w", err)
		}
		logger.Info("seeded admin user", "username", admin.Username)
	}

	if _, err := users.FindByUsername(ctx, superAdmin.Username); err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return fmt.Errorf("seed check superadmin user: %w", err)
		}
		if err := users.Create(ctx, superAdmin); err != nil && !errors.Is(err, repository.ErrDuplicateKey) {
			return fmt.Errorf("seed superadmin user: %w", err)
		}
		logger.Info("seeded superadmin user", "username", superAdmin.Username)
	}

	return nil
}
