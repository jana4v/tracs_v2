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

// RoleRepository provides SQLite CRUD for Role documents.
type RoleRepository struct {
	db *sql.DB
}

// NewRoleRepository creates a RoleRepository backed by SQLite.
func NewRoleRepository(sdb *clients.SQLiteDB) *RoleRepository {
	return &RoleRepository{db: sdb.DB}
}

// Create inserts a new role. Returns ErrDuplicateKey if name already exists.
func (r *RoleRepository) Create(ctx context.Context, role *models.Role) error {
	role.ID = clients.NewID()
	role.CreatedAt = time.Now().UTC()
	role.UpdatedAt = role.CreatedAt
	if role.Permissions == nil {
		role.Permissions = []string{}
	}

	data, err := json.Marshal(role)
	if err != nil {
		return fmt.Errorf("marshal role: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO iam_roles(id, name, data) VALUES(?,?,?)`,
		role.ID, role.Name, string(data),
	)
	if err != nil {
		if clients.IsDuplicateKey(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("insert role: %w", err)
	}
	return nil
}

// FindByID returns a role by UUID string.
func (r *RoleRepository) FindByID(ctx context.Context, id string) (*models.Role, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT data FROM iam_roles WHERE id=?`, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find role by id: %w", err)
	}
	var role models.Role
	if err := json.Unmarshal([]byte(raw), &role); err != nil {
		return nil, fmt.Errorf("decode role: %w", err)
	}
	return &role, nil
}

// FindByName returns a role by its unique name.
func (r *RoleRepository) FindByName(ctx context.Context, name string) (*models.Role, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT data FROM iam_roles WHERE name=?`, name).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find role by name: %w", err)
	}
	var role models.Role
	if err := json.Unmarshal([]byte(raw), &role); err != nil {
		return nil, fmt.Errorf("decode role: %w", err)
	}
	return &role, nil
}

// List returns all roles sorted by name.
func (r *RoleRepository) List(ctx context.Context) ([]models.Role, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT data FROM iam_roles ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var role models.Role
		if json.Unmarshal([]byte(raw), &role) == nil {
			roles = append(roles, role)
		}
	}
	if roles == nil {
		roles = []models.Role{}
	}
	return roles, nil
}

// Update applies partial changes to an existing role (read-modify-write).
func (r *RoleRepository) Update(ctx context.Context, id string, fields map[string]interface{}) error {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT data FROM iam_roles WHERE id=?`, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("find role for update: %w", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return fmt.Errorf("decode role for update: %w", err)
	}

	for k, v := range fields {
		doc[k] = v
	}
	doc["updated_at"] = time.Now().UTC()

	updated, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("re-marshal role: %w", err)
	}

	result, err := r.db.ExecContext(ctx,
		`UPDATE iam_roles SET data=? WHERE id=?`, string(updated), id,
	)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a role by ID.
func (r *RoleRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM iam_roles WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Count returns the total number of roles.
func (r *RoleRepository) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM iam_roles`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count roles: %w", err)
	}
	return n, nil
}
