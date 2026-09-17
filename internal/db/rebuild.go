package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Execer is the subset of *sql.DB / *sql.Conn / *sql.Tx used by catalog maintenance.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// DropWorksFTSTriggers removes works FTS update/delete triggers. Call inside the import transaction.
func DropWorksFTSTriggers(ctx context.Context, e Execer) error {
	for _, name := range []string{"works_fts_au", "works_fts_ad"} {
		if _, err := e.ExecContext(ctx, "DROP TRIGGER IF EXISTS "+name); err != nil {
			return err
		}
	}
	return nil
}

// EnsureWorksFTSTriggers creates works_fts triggers from 001_initial.sql if missing.
func EnsureWorksFTSTriggers(ctx context.Context, e Execer) error {
	stmts, err := WorksFTSTriggers()
	if err != nil {
		return err
	}
	if _, err := e.ExecContext(ctx, "DROP TRIGGER IF EXISTS works_fts_au"); err != nil {
		return err
	}
	if _, err := e.ExecContext(ctx, "DROP TRIGGER IF EXISTS works_fts_ad"); err != nil {
		return err
	}
	for _, s := range stmts {
		if _, err := e.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func worksFTSTriggerCount(ctx context.Context, e Execer) (int, error) {
	var n int
	err := e.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name IN ('works_fts_au','works_fts_ad')`).Scan(&n)
	return n, err
}

// RebuildWorksFTS drops and recreates works_fts, filling it with normalize(...) values.
func RebuildWorksFTS(ctx context.Context, e Execer) error {
	create, err := FTSCreate("works_fts")
	if err != nil {
		return err
	}
	if _, err := e.ExecContext(ctx, "DROP TABLE IF EXISTS works_fts"); err != nil {
		return err
	}
	if _, err := e.ExecContext(ctx, create); err != nil {
		return err
	}
	_, err = e.ExecContext(ctx, `
INSERT INTO works_fts(rowid, title, authors, series)
SELECT w.id,
       normalize(w.title),
       normalize(w.authors_text),
       (SELECT normalize(group_concat(DISTINCT e.series))
          FROM editions e
         WHERE e.work_id = w.id
           AND e.is_active = 1 AND e.is_deleted = 0
           AND e.series IS NOT NULL AND trim(e.series) != '')
  FROM works w`)
	if err != nil {
		return fmt.Errorf("works_fts fill: %w", err)
	}
	return EnsureWorksFTSTriggers(ctx, e)
}

// SetFTSDirty writes app_meta.fts_dirty.
func SetFTSDirty(ctx context.Context, e Execer, dirty bool) error {
	v := "0"
	if dirty {
		v = "1"
	}
	_, err := e.ExecContext(ctx, `INSERT INTO app_meta(key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, MetaFTSDirty, v)
	return err
}

func ftsDirty(ctx context.Context, e Execer) (bool, error) {
	var v string
	err := e.QueryRowContext(ctx, `SELECT value FROM app_meta WHERE key = ?`, MetaFTSDirty).Scan(&v)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v == "1", nil
}
