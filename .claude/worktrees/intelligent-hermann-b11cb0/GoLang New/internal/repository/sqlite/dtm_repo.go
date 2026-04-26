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

// DTMRepo implements repository.DTMStore against the dtm_procedures SQLite table.
type DTMRepo struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewDTMRepo creates a DTMRepo backed by the given SQLiteDB.
func NewDTMRepo(sdb *clients.SQLiteDB, logger *slog.Logger) *DTMRepo {
	return &DTMRepo{db: sdb.DB, logger: logger}
}

func (r *DTMRepo) Get(ctx context.Context, project string) (*models.DTMProcedures, error) {
	var raw string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM dtm_procedures WHERE project=?`, project,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var doc models.DTMProcedures
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *DTMRepo) GetRaw(ctx context.Context, project string) (map[string]any, error) {
	var raw string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM dtm_procedures WHERE project=?`, project,
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

func (r *DTMRepo) Save(ctx context.Context, doc models.DTMProcedures) error {
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO dtm_procedures(project, data) VALUES(?,?)
		 ON CONFLICT(project) DO UPDATE SET data=excluded.data`,
		doc.Project, string(data),
	)
	return err
}
