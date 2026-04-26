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

// SpasdacsRepo implements repository.SpasdacsStore against the spasdacs SQLite table.
type SpasdacsRepo struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewSpasdacsRepo creates a SpasdacsRepo backed by the given SQLiteDB.
func NewSpasdacsRepo(sdb *clients.SQLiteDB, logger *slog.Logger) *SpasdacsRepo {
	return &SpasdacsRepo{db: sdb.DB, logger: logger}
}

func (r *SpasdacsRepo) List(ctx context.Context) ([]models.SpasdacsMeta, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT data FROM spasdacs ORDER BY json_extract(data,'$.updatedAt') DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.SpasdacsMeta
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var m models.SpasdacsMeta
		if json.Unmarshal([]byte(raw), &m) == nil {
			result = append(result, m)
		}
	}
	if result == nil {
		result = []models.SpasdacsMeta{}
	}
	return result, rows.Err()
}

func (r *SpasdacsRepo) Get(ctx context.Context, id string) (*models.SpasdacsDiagram, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT data FROM spasdacs WHERE id=?`, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var d models.SpasdacsDiagram
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *SpasdacsRepo) Save(ctx context.Context, d models.SpasdacsDiagram) error {
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO spasdacs(id, name, data) VALUES(?,?,?)
		 ON CONFLICT(id) DO UPDATE SET name=excluded.name, data=excluded.data`,
		d.ID, d.Name, string(data),
	)
	return err
}

func (r *SpasdacsRepo) Patch(ctx context.Context, id string, patchFn func(map[string]any)) error {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT data FROM spasdacs WHERE id=?`, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return repository.ErrNotFound
	}
	if err != nil {
		return err
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return err
	}

	patchFn(doc)

	updated, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE spasdacs SET data=? WHERE id=?`, string(updated), id,
	)
	return err
}

func (r *SpasdacsRepo) Delete(ctx context.Context, id string) (bool, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM spasdacs WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}
