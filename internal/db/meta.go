package db

import (
	"context"
	"database/sql"
)

const (
	MetaLibraryVolume  = "library_volume_label"
	MetaLibraryRel     = "library_volume_rel"
	MetaINPXImportedAt = "inpx_imported_at"
)

// SetMeta writes a key/value row in app_meta.
func SetMeta(ctx context.Context, e Execer, key, value string) error {
	_, err := e.ExecContext(ctx, `INSERT INTO app_meta(key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// Meta reads a key from app_meta. Missing keys return empty, not an error.
func Meta(ctx context.Context, e Execer, key string) (string, error) {
	var v string
	err := e.QueryRowContext(ctx, `SELECT value FROM app_meta WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}
