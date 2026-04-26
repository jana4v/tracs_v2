package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/iam/internal/models"
)

// UserRepository provides SQLite CRUD for User documents.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a UserRepository backed by SQLite.
// Table indexes are created at startup via SQLiteDB.migrate().
func NewUserRepository(sdb *clients.SQLiteDB) *UserRepository {
	return &UserRepository{db: sdb.DB}
}

// Create inserts a new user. Returns ErrDuplicateKey if username/email already exists.
func (r *UserRepository) Create(ctx context.Context, u *models.User) error {
	u.ID = clients.NewID()
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = u.CreatedAt
	if u.Roles == nil {
		u.Roles = []string{}
	}

	data, err := json.Marshal(u)
	if err != nil {
		return fmt.Errorf("marshal user: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO iam_users(id, username, email, data) VALUES(?,?,?,?)`,
		u.ID, u.Username, u.Email, string(data),
	)
	if err != nil {
		if clients.IsDuplicateKey(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// FindByID returns a user by their UUID string.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT data FROM iam_users WHERE id=?`, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	var u models.User
	if err := json.Unmarshal([]byte(raw), &u); err != nil {
		return nil, fmt.Errorf("decode user: %w", err)
	}
	return &u, nil
}

// FindByUsername returns a user by their username (case-sensitive).
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT data FROM iam_users WHERE username=?`, username).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by username: %w", err)
	}
	var u models.User
	if err := json.Unmarshal([]byte(raw), &u); err != nil {
		return nil, fmt.Errorf("decode user: %w", err)
	}
	return &u, nil
}

// List returns all users sorted by username.
func (r *UserRepository) List(ctx context.Context) ([]models.User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT data FROM iam_users ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var u models.User
		if json.Unmarshal([]byte(raw), &u) == nil {
			users = append(users, u)
		}
	}
	if users == nil {
		users = []models.User{}
	}
	return users, nil
}

// Update applies partial changes to an existing user (read-modify-write).
func (r *UserRepository) Update(ctx context.Context, id string, fields map[string]interface{}) error {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT data FROM iam_users WHERE id=?`, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("find user for update: %w", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return fmt.Errorf("decode user for update: %w", err)
	}

	for k, v := range fields {
		doc[k] = v
	}
	doc["updated_at"] = time.Now().UTC()

	updated, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("re-marshal user: %w", err)
	}

	result, err := r.db.ExecContext(ctx,
		`UPDATE iam_users SET data=? WHERE id=?`, string(updated), id,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a user by ID.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM iam_users WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Count returns the total number of users.
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM iam_users`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

// UpdateLastLogin sets the last_login timestamp for a user.
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id string) error {
	return r.Update(ctx, id, map[string]interface{}{
		"last_login": time.Now().UTC(),
	})
}
