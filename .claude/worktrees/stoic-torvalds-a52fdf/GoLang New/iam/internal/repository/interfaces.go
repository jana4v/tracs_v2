package repository

import (
	"context"

	"github.com/mainframe/tm-system/iam/internal/models"
)

// UserStore is the read/write interface for IAM users.
type UserStore interface {
	Create(ctx context.Context, u *models.User) error
	FindByID(ctx context.Context, id string) (*models.User, error)
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	List(ctx context.Context) ([]models.User, error)
	Update(ctx context.Context, id string, fields map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
	UpdateLastLogin(ctx context.Context, id string) error
}

// RoleStore is the read/write interface for IAM roles.
type RoleStore interface {
	Create(ctx context.Context, role *models.Role) error
	FindByID(ctx context.Context, id string) (*models.Role, error)
	FindByName(ctx context.Context, name string) (*models.Role, error)
	List(ctx context.Context) ([]models.Role, error)
	Update(ctx context.Context, id string, fields map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

// TokenStore is the read/write interface for IAM refresh tokens.
type TokenStore interface {
	Create(ctx context.Context, t *models.RefreshToken) error
	FindByToken(ctx context.Context, token string) (*models.RefreshToken, error)
	Revoke(ctx context.Context, token string) error
	RevokeAllForUser(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
}
