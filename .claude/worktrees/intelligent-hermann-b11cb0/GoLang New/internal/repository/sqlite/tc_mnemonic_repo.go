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

// TCMnemonicRepo implements repository.TCMnemonicStore against the SQLite
// tc_mnemonics and tc_mnemonics_change_history tables.
type TCMnemonicRepo struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewTCMnemonicRepo creates a TCMnemonicRepo backed by the given SQLiteDB.
func NewTCMnemonicRepo(sdb *clients.SQLiteDB, logger *slog.Logger) *TCMnemonicRepo {
	return &TCMnemonicRepo{db: sdb.DB, logger: logger}
}

func (r *TCMnemonicRepo) FindAll(ctx context.Context) ([]map[string]any, error) {
	return queryAllRaw(ctx, r.db, `SELECT data FROM tc_mnemonics`)
}

func (r *TCMnemonicRepo) FindBySubsystem(ctx context.Context, subsystem string) ([]map[string]any, error) {
	return queryAllRaw(ctx, r.db,
		`SELECT data FROM tc_mnemonics WHERE UPPER(subsystem)=UPPER(?)`, subsystem)
}

func (r *TCMnemonicRepo) GetSubsystems(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT subsystem FROM tc_mnemonics ORDER BY subsystem`)
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

func (r *TCMnemonicRepo) GetAllCmdDescs(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT json_extract(data,'$.cmdDesc') FROM tc_mnemonics WHERE json_extract(data,'$.cmdDesc') IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var descs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		if s != "" {
			descs = append(descs, s)
		}
	}
	if descs == nil {
		descs = []string{}
	}
	return descs, rows.Err()
}

func (r *TCMnemonicRepo) GetCmdDescsBySubsystem(ctx context.Context, subsystem string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT json_extract(data,'$.cmdDesc') FROM tc_mnemonics WHERE UPPER(subsystem) LIKE UPPER(?)`,
		subsystem,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var descs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		if s != "" {
			descs = append(descs, s)
		}
	}
	if descs == nil {
		descs = []string{}
	}
	return descs, rows.Err()
}

func (r *TCMnemonicRepo) FindByCmdDesc(ctx context.Context, cmdDesc string) (map[string]any, error) {
	var raw string
	err := r.db.QueryRowContext(ctx,
		`SELECT data FROM tc_mnemonics WHERE UPPER(json_extract(data,'$.cmdDesc'))=UPPER(?) LIMIT 1`,
		cmdDesc,
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

func (r *TCMnemonicRepo) GetByIDRaw(ctx context.Context, id string) (map[string]any, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT data FROM tc_mnemonics WHERE id=?`, id).Scan(&raw)
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

func (r *TCMnemonicRepo) SaveDoc(ctx context.Context, id, subsystem string, doc map[string]any) error {
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO tc_mnemonics(id, subsystem, data) VALUES(?,?,?)
		 ON CONFLICT(id) DO UPDATE SET subsystem=excluded.subsystem, data=excluded.data`,
		id, subsystem, string(data),
	)
	return err
}

func (r *TCMnemonicRepo) AppendHistory(ctx context.Context, id string, entry any) error {
	return appendHistory(ctx, r.db, "tc_mnemonics_change_history", id, entry)
}

// ─── SCO commands ────────────────────────────────────────────────────────────

// SCOCommandRepo implements repository.SCOCommandStore against the sco_commands table.
type SCOCommandRepo struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewSCOCommandRepo creates an SCOCommandRepo backed by the given SQLiteDB.
func NewSCOCommandRepo(sdb *clients.SQLiteDB, logger *slog.Logger) *SCOCommandRepo {
	return &SCOCommandRepo{db: sdb.DB, logger: logger}
}

func (r *SCOCommandRepo) FindAll(ctx context.Context) ([]models.ScoCommand, error) {
	return queryAll[models.ScoCommand](ctx, r.db, `SELECT data FROM sco_commands`)
}
