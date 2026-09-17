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

const worksFTSBatch = 20000

// RebuildWorksFTS drops and recreates works_fts, filling it with normalize(...) values.
func RebuildWorksFTS(ctx context.Context, conn *sql.Conn) error {
	create, err := FTSCreate("works_fts")
	if err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "DROP TABLE IF EXISTS works_fts"); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, create); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `DROP TABLE IF EXISTS temp.work_series`); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `CREATE TEMP TABLE work_series (
		work_id INTEGER PRIMARY KEY,
		series  TEXT
	)`); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, `
INSERT INTO work_series(work_id, series)
SELECT e.work_id, group_concat(DISTINCT e.series)
  FROM editions e
 WHERE e.is_active = 1 AND e.is_deleted = 0
   AND e.series IS NOT NULL AND trim(e.series) != ''
 GROUP BY e.work_id`); err != nil {
		return fmt.Errorf("work_series: %w", err)
	}
	var lastID int64
	for {
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `
INSERT INTO works_fts(rowid, title, authors, series)
SELECT w.id,
       w.sort_title,
       normalize(w.authors_text),
       normalize(s.series)
  FROM works w
  LEFT JOIN work_series s ON s.work_id = w.id
 WHERE w.id > ?
 ORDER BY w.id
 LIMIT ?`, lastID, worksFTSBatch)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("works_fts fill: %w", err)
		}
		n, _ := res.RowsAffected()
		if err := tx.Commit(); err != nil {
			return err
		}
		if n == 0 {
			break
		}
		if err := conn.QueryRowContext(ctx, `SELECT max(rowid) FROM works_fts`).Scan(&lastID); err != nil {
			return err
		}
	}
	_, _ = conn.ExecContext(ctx, `DROP TABLE IF EXISTS temp.work_series`)
	return EnsureWorksFTSTriggers(ctx, conn)
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
