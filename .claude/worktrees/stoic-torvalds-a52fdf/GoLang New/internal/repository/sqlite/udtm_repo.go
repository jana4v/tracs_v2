package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/internal/models"
	"github.com/mainframe/tm-system/internal/repository"
)

// UDTMRepo implements repository.UDTMStore against the user_telemetry and
// user_telemetry_versions SQLite tables.
type UDTMRepo struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewUDTMRepo creates a UDTMRepo backed by the given SQLiteDB.
func NewUDTMRepo(sdb *clients.SQLiteDB, logger *slog.Logger) *UDTMRepo {
	return &UDTMRepo{db: sdb.DB, logger: logger}
}

func (r *UDTMRepo) Get(ctx context.Context, project string) (*models.UserTelemetry, error) {
	var raw string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM user_telemetry WHERE project=?`, project,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var doc models.UserTelemetry
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *UDTMRepo) GetRaw(ctx context.Context, project string) (map[string]any, error) {
	var raw string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM user_telemetry WHERE project=?`, project,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (r *UDTMRepo) Save(ctx context.Context, doc models.UserTelemetry) error {
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO user_telemetry(project, data) VALUES(?,?)
		 ON CONFLICT(project) DO UPDATE SET data=excluded.data`,
		doc.Project, string(data),
	)
	return err
}

func (r *UDTMRepo) SaveVersion(ctx context.Context, ver models.UserTelemetryVersion) error {
	data, err := json.Marshal(ver)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO user_telemetry_versions(project, version, data) VALUES(?,?,?)`,
		ver.Project, ver.Version, string(data),
	)
	return err
}

func (r *UDTMRepo) ListVersions(ctx context.Context, project string) ([]models.UserTelemetryVersion, error) {
	return queryAll[models.UserTelemetryVersion](ctx, r.db,
		`SELECT data FROM user_telemetry_versions WHERE project=?`, project)
}

func (r *UDTMRepo) GetVersion(ctx context.Context, project string, version int) (*models.UserTelemetryVersion, error) {
	var raw string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM user_telemetry_versions WHERE project=? AND version=?`,
		project, version,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var doc models.UserTelemetryVersion
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}
