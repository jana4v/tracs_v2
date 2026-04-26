package clients

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/mainframe/tm-system/internal/config"
	_ "modernc.org/sqlite"
)

// SQLiteDB wraps a *sql.DB and runs the schema migration on startup.
type SQLiteDB struct {
	DB     *sql.DB
	logger *slog.Logger
}

// NewSQLiteDB opens (or creates) the SQLite database file, applies WAL mode,
// and runs the schema migration.
func NewSQLiteDB(ctx context.Context, cfg config.SQLiteConfig, logger *slog.Logger) (*SQLiteDB, error) {
	path := cfg.Path
	if path == "" {
		path = "./astra.db"
	}
	if cfg.InMemory {
		path = ":memory:"
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite open %s: %w", path, err)
	}

	// Single writer; WAL allows concurrent readers.
	db.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		var err error
		maxRetries := 5
		for attempt := 0; attempt < maxRetries; attempt++ {
			_, err = db.ExecContext(ctx, p)
			if err == nil {
				break
			}
			// Retry on busy/locked errors
			if attempt < maxRetries-1 {
				backoff := time.Duration((1<<uint(attempt))*100) * time.Millisecond
				time.Sleep(backoff)
				continue
			}
		}
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("sqlite pragma %q: %w", p, err)
		}
	}

	s := &SQLiteDB{DB: db, logger: logger}
	if err := s.migrate(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite migrate: %w", err)
	}

	logger.Info("connected to SQLite", "path", path)
	return s, nil
}

// Close closes the underlying database connection.
func (s *SQLiteDB) Close() error {
	return s.DB.Close()
}

// migrate creates all required tables if they do not already exist.
func (s *SQLiteDB) migrate(ctx context.Context) error {
	for _, stmt := range allSchemas {
		if _, err := s.DB.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate statement failed: %w\nSQL: %s", err, stmt)
		}
	}
	return nil
}

// LoadMnemonics loads all documents from tm_mnemonics as raw maps.
// Used by comparator and ingest services.
func (s *SQLiteDB) LoadMnemonics(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT data FROM tm_mnemonics`)
	if err != nil {
		return nil, fmt.Errorf("query tm_mnemonics: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("scan tm_mnemonics: %w", err)
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return nil, fmt.Errorf("unmarshal tm_mnemonics row: %w", err)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}
