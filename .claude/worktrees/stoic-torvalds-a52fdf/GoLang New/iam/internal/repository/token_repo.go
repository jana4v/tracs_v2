package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/iam/internal/models"
)

// TokenRepository provides SQLite operations for refresh token records.
type TokenRepository struct {
	db *sql.DB
}

// NewTokenRepository creates a TokenRepository backed by SQLite.
// Indexes are created at startup via SQLiteDB.migrate().
func NewTokenRepository(sdb *clients.SQLiteDB) *TokenRepository {
	return &TokenRepository{db: sdb.DB}
}

// Create stores a new refresh token.
func (r *TokenRepository) Create(ctx context.Context, t *models.RefreshToken) error {
	t.ID = clients.NewID()
	t.CreatedAt = time.Now().UTC()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO iam_refresh_tokens(id, user_id, token, expires_at, revoked) VALUES(?,?,?,?,0)`,
		t.ID, t.UserID, t.Token, clients.TimeToUnix(t.ExpiresAt),
	)
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

// FindByToken returns a non-revoked, non-expired token record.
func (r *TokenRepository) FindByToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	now := clients.TimeToUnix(time.Now())
	var t models.RefreshToken
	var expiresAtUnix int64
	var revokedInt int

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, token, expires_at, revoked FROM iam_refresh_tokens
		 WHERE token=? AND revoked=0 AND expires_at>?`,
		token, now,
	).Scan(&t.ID, &t.UserID, &t.Token, &expiresAtUnix, &revokedInt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find refresh token: %w", err)
	}

	t.ExpiresAt = clients.UnixToTime(expiresAtUnix)
	t.Revoked = revokedInt != 0
	return &t, nil
}

// Revoke marks a specific token as revoked.
func (r *TokenRepository) Revoke(ctx context.Context, token string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE iam_refresh_tokens SET revoked=1 WHERE token=?`, token,
	)
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// RevokeAllForUser revokes all active refresh tokens for a user.
func (r *TokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE iam_refresh_tokens SET revoked=1 WHERE user_id=? AND revoked=0`, userID,
	)
	if err != nil {
		return fmt.Errorf("revoke user tokens: %w", err)
	}
	return nil
}

// DeleteExpired removes all expired token rows. Called periodically to replace MongoDB TTL index.
func (r *TokenRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM iam_refresh_tokens WHERE expires_at<?`, clients.TimeToUnix(time.Now()),
	)
	return err
}
