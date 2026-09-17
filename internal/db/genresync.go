package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/alexbelweb/flibustahub/internal/data"
)

const metaINPXVersion = "inpx_version"

// SyncGenreNames updates name_ru for codes already present in genres.
// It does not insert dictionary rows that were never seen in a dump.
func (d *DB) SyncGenreNames(ctx context.Context) error {
	if d == nil || d.Write == nil {
		return nil
	}
	return SyncGenreNames(ctx, d.Write)
}

// SyncGenreNames updates existing genre rows from the effective dictionary.
func SyncGenreNames(ctx context.Context, e Execer) error {
	rows, err := e.QueryContext(ctx, `SELECT code, name_ru FROM genres`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type pair struct{ code, name string }
	var stale []pair
	for rows.Next() {
		var code, name string
		if err := rows.Scan(&code, &name); err != nil {
			return err
		}
		want := data.GenreNameRU(code)
		if want != name {
			stale = append(stale, pair{code, want})
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, p := range stale {
		if _, err := e.ExecContext(ctx, `UPDATE genres SET name_ru = ? WHERE code = ? AND name_ru <> ?`, p.name, p.code, p.name); err != nil {
			return fmt.Errorf("sync genre %s: %w", p.code, err)
		}
	}
	return nil
}

// SetINPXVersion stores the last successfully imported dump version.
func SetINPXVersion(ctx context.Context, e Execer, version string) error {
	if version == "" {
		return nil
	}
	_, err := e.ExecContext(ctx, `INSERT INTO app_meta(key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, metaINPXVersion, version)
	return err
}

// INPXVersion returns the last successfully imported dump version, or empty.
func INPXVersion(ctx context.Context, e Execer) (string, error) {
	var v string
	err := e.QueryRowContext(ctx, `SELECT value FROM app_meta WHERE key = ?`, metaINPXVersion).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}
