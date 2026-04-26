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

// TMMnemonicRepo implements repository.TMMnemonicStore against the SQLite
// tm_mnemonics and tm_mnemonics_change_history tables.
type TMMnemonicRepo struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewTMMnemonicRepo creates a TMMnemonicRepo backed by the given SQLiteDB.
func NewTMMnemonicRepo(sdb *clients.SQLiteDB, logger *slog.Logger) *TMMnemonicRepo {
	return &TMMnemonicRepo{db: sdb.DB, logger: logger}
}

func (r *TMMnemonicRepo) FindAll(ctx context.Context) ([]models.TmMnemonic, error) {
	return queryAll[models.TmMnemonic](ctx, r.db, `SELECT data FROM tm_mnemonics`)
}

func (r *TMMnemonicRepo) FindBySubsystem(ctx context.Context, subsystem string) ([]models.TmMnemonic, error) {
	return queryAll[models.TmMnemonic](ctx, r.db,
		`SELECT data FROM tm_mnemonics WHERE UPPER(subsystem)=UPPER(?)`, subsystem)
}

func (r *TMMnemonicRepo) FindBySubsystemPattern(ctx context.Context, pattern string) ([]models.TmMnemonic, error) {
	return queryAll[models.TmMnemonic](ctx, r.db,
		`SELECT data FROM tm_mnemonics WHERE UPPER(subsystem) LIKE UPPER(?)`, pattern)
}

func (r *TMMnemonicRepo) FindBySubsystemAndMnemonic(ctx context.Context, subsystem, mnemonic string) (*models.TmMnemonic, error) {
	var raw string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM tm_mnemonics
		 WHERE UPPER(subsystem) LIKE UPPER(?) AND UPPER(json_extract(data,'$.cdbMnemonic'))=UPPER(?)
		 LIMIT 1`,
		subsystem, mnemonic,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var m models.TmMnemonic
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *TMMnemonicRepo) GetSubsystems(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT subsystem FROM tm_mnemonics ORDER BY subsystem`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	if subs == nil {
		subs = []string{}
	}
	return subs, rows.Err()
}

func (r *TMMnemonicRepo) FindWithComparisonEnabled(ctx context.Context) ([]models.TmMnemonic, error) {
	return queryAll[models.TmMnemonic](ctx, r.db,
		`SELECT data FROM tm_mnemonics WHERE json_extract(data,'$.enable_comparison')=1`)
}

func (r *TMMnemonicRepo) GetByIDRaw(ctx context.Context, id string) (map[string]any, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT data FROM tm_mnemonics WHERE id=?`, id).Scan(&raw)
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

func (r *TMMnemonicRepo) SaveDoc(ctx context.Context, id, subsystem string, doc map[string]any) error {
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO tm_mnemonics(id, subsystem, data) VALUES(?,?,?)
		 ON CONFLICT(id) DO UPDATE SET subsystem=excluded.subsystem, data=excluded.data`,
		id, subsystem, string(data),
	)
	return err
}

func (r *TMMnemonicRepo) PatchBySubsystemMnemonic(ctx context.Context, subsystem, mnemonic string, patchFn func(map[string]any)) (int64, error) {
	var id, raw string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, data FROM tm_mnemonics
		 WHERE UPPER(subsystem)=UPPER(?) AND UPPER(json_extract(data,'$.cdbMnemonic'))=UPPER(?)
		 LIMIT 1`,
		subsystem, mnemonic,
	).Scan(&id, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return 0, err
	}

	patchFn(doc)

	updated, err := json.Marshal(doc)
	if err != nil {
		return 0, err
	}
	if _, err := r.db.ExecContext(ctx,
		`UPDATE tm_mnemonics SET data=? WHERE id=?`, string(updated), id,
	); err != nil {
		return 0, err
	}
	return 1, nil
}

func (r *TMMnemonicRepo) AppendHistory(ctx context.Context, id string, entry any) error {
	return appendHistory(ctx, r.db, "tm_mnemonics_change_history", id, entry)
}
