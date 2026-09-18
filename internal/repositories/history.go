package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/alexbelweb/flibustahub/internal/textnorm"
)

type HistoryRow struct {
	ID         int64
	Query      string
	SearchedAt string
}

func (c *Catalog) ListHistory(ctx context.Context, limit int) ([]HistoryRow, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := c.read.QueryContext(ctx, `SELECT id, query, searched_at FROM search_history ORDER BY searched_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []HistoryRow
	for rows.Next() {
		var r HistoryRow
		if err := rows.Scan(&r.ID, &r.Query, &r.SearchedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (c *Catalog) RecordHistory(ctx context.Context, query string, now time.Time, keep int) error {
	norm := textnorm.Normalize(query)
	if norm == "" {
		return nil
	}
	stamp := now.UTC().Format(time.RFC3339)
	tx, err := c.write.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var id int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM search_history WHERE normalize(query) = ? ORDER BY searched_at DESC LIMIT 1`, norm).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		if _, err := tx.ExecContext(ctx, `INSERT INTO search_history(query, searched_at) VALUES (?, ?)`, query, stamp); err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		if _, err := tx.ExecContext(ctx, `UPDATE search_history SET query = ?, searched_at = ? WHERE id = ?`, query, stamp, id); err != nil {
			return err
		}
	}
	if keep > 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM search_history WHERE id NOT IN (
			SELECT id FROM search_history ORDER BY searched_at DESC, id DESC LIMIT ?)`, keep); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (c *Catalog) ClearHistory(ctx context.Context) error {
	_, err := c.write.ExecContext(ctx, `DELETE FROM search_history`)
	return err
}
