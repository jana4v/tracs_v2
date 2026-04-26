package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// queryAll executes query and deserialises every `data TEXT` column row into T.
func queryAll[T any](ctx context.Context, db *sql.DB, query string, args ...any) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var item T
		if err := json.Unmarshal([]byte(raw), &item); err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if results == nil {
		results = []T{}
	}
	return results, nil
}

// queryAllRaw executes query and deserialises every row into map[string]any.
func queryAllRaw(ctx context.Context, db *sql.DB, query string, args ...any) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var doc map[string]any
		if err := json.Unmarshal([]byte(raw), &doc); err != nil {
			return nil, err
		}
		results = append(results, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if results == nil {
		results = []map[string]any{}
	}
	return results, nil
}

// allowedHistoryTables is the whitelist of tables accepted by appendHistory.
var allowedHistoryTables = map[string]bool{
	"tm_mnemonics_change_history": true,
	"tc_mnemonics_change_history": true,
}

// appendHistory appends a JSON entry to the history array in a *_change_history table.
// tableName must be one of: tm_mnemonics_change_history, tc_mnemonics_change_history.
func appendHistory(ctx context.Context, db *sql.DB, tableName, id string, entry any) error {
	if !allowedHistoryTables[tableName] {
		return fmt.Errorf("appendHistory: unknown table %q", tableName)
	}

	var histRaw string
	err := db.QueryRowContext(ctx,
		`SELECT history FROM `+tableName+` WHERE id=?`, id,
	).Scan(&histRaw)

	if errors.Is(err, sql.ErrNoRows) {
		b, _ := json.Marshal([]any{entry})
		_, err2 := db.ExecContext(ctx,
			`INSERT INTO `+tableName+`(id, history) VALUES(?,?)`,
			id, string(b),
		)
		return err2
	}
	if err != nil {
		return err
	}

	var hist []any
	if err := json.Unmarshal([]byte(histRaw), &hist); err != nil {
		hist = []any{}
	}
	hist = append(hist, entry)
	b, _ := json.Marshal(hist)
	_, err = db.ExecContext(ctx,
		`UPDATE `+tableName+` SET history=? WHERE id=?`,
		string(b), id,
	)
	return err
}
