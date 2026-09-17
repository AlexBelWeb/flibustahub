package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/migrations"
)

const schemaTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version    INTEGER PRIMARY KEY,
  name       TEXT    NOT NULL,
  applied_at TEXT    NOT NULL,
  checksum   TEXT    NOT NULL
);
`

// Migration is one schema file.
type Migration struct {
	Version  int
	Name     string
	SQL      string
	Checksum string
}

func loadMigrations(source fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(source, ".")
	if err != nil {
		return nil, err
	}
	var out []Migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		m, err := parseMigration(source, e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func parseMigration(source fs.FS, name string) (Migration, error) {
	raw, err := fs.ReadFile(source, name)
	if err != nil {
		return Migration{}, err
	}
	base := strings.TrimSuffix(name, ".sql")
	parts := strings.SplitN(base, "_", 2)
	if len(parts) != 2 {
		return Migration{}, fmt.Errorf("migration name %q must be NNN_slug.sql", name)
	}
	ver, err := strconv.Atoi(parts[0])
	if err != nil || ver <= 0 {
		return Migration{}, fmt.Errorf("migration version in %q", name)
	}
	sum := sha256.Sum256(raw)
	return Migration{
		Version:  ver,
		Name:     base,
		SQL:      string(raw),
		Checksum: hex.EncodeToString(sum[:]),
	}, nil
}

func applyMigrations(ctx context.Context, d *DB, source fs.FS) error {
	if source == nil {
		source = migrations.FS
	}
	files, err := loadMigrations(source)
	if err != nil {
		return apperr.Wrap(apperr.CodeDBMigrateFailed, err, nil)
	}
	if err := rejectOccupiedCatalog(ctx, d.Write, files); err != nil {
		return err
	}
	if _, err := d.Write.ExecContext(ctx, schemaTable); err != nil {
		return apperr.Wrap(apperr.CodeDBMigrateFailed, err, nil)
	}

	applied, err := readApplied(ctx, d.Write)
	if err != nil {
		return apperr.Wrap(apperr.CodeDBMigrateFailed, err, nil)
	}
	for _, f := range files {
		row, ok := applied[f.Version]
		if !ok {
			continue
		}
		if row.Checksum != f.Checksum {
			d.log.Error("migration checksum mismatch",
				"version", f.Version,
				"name", f.Name,
				"applied", row.Checksum,
				"file", f.Checksum,
			)
			return apperr.New(apperr.CodeDBMigrateFailed, map[string]string{"name": f.Name})
		}
	}

	var pending []Migration
	for _, f := range files {
		if _, ok := applied[f.Version]; !ok {
			pending = append(pending, f)
		}
	}
	if len(pending) == 0 {
		return nil
	}
	if len(applied) > 0 {
		if _, err := d.Backup(ctx); err != nil {
			return err
		}
	}
	for _, m := range pending {
		if err := runMigration(ctx, d.Write, m, d.now()); err != nil {
			return err
		}
		d.log.Info("applied migration", "version", m.Version, "name", m.Name)
	}
	return nil
}

type appliedRow struct {
	Checksum string
}

func readApplied(ctx context.Context, w *sql.DB) (map[int]appliedRow, error) {
	rows, err := w.QueryContext(ctx, `SELECT version, checksum FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[int]appliedRow{}
	for rows.Next() {
		var v int
		var sum string
		if err := rows.Scan(&v, &sum); err != nil {
			return nil, err
		}
		out[v] = appliedRow{Checksum: sum}
	}
	return out, rows.Err()
}

// rejectOccupiedCatalog refuses to initialize a file that already has a works
// table but no recorded 001_initial: that is some other catalog, not an empty v2 database.
func rejectOccupiedCatalog(ctx context.Context, w *sql.DB, files []Migration) error {
	hasInitial := false
	for _, f := range files {
		if f.Version == 1 {
			hasInitial = true
			break
		}
	}
	if !hasInitial {
		return nil
	}
	var table string
	err := w.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'works'`).Scan(&table)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return apperr.Wrap(apperr.CodeDBMigrateFailed, err, nil)
	}
	var hasMigrations int
	if err := w.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'`).Scan(&hasMigrations); err != nil {
		return apperr.Wrap(apperr.CodeDBMigrateFailed, err, nil)
	}
	if hasMigrations == 1 {
		var stamped int
		if err := w.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version = 1`).Scan(&stamped); err != nil {
			return apperr.Wrap(apperr.CodeDBMigrateFailed, err, nil)
		}
		if stamped > 0 {
			return nil
		}
	}
	return apperr.New(apperr.CodeDBIncompatible, nil)
}

func runMigration(ctx context.Context, w *sql.DB, m Migration, at time.Time) error {
	tx, err := w.BeginTx(ctx, nil)
	if err != nil {
		return apperr.Wrap(apperr.CodeDBMigrateFailed, err, nil)
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range splitSQL(m.SQL) {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return apperr.Wrap(apperr.CodeDBMigrateFailed, fmt.Errorf("%s: %w", preview(stmt), err), map[string]string{"name": m.Name})
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations(version, name, applied_at, checksum) VALUES (?, ?, ?, ?)`,
		m.Version, m.Name, at.UTC().Format(time.RFC3339), m.Checksum,
	); err != nil {
		return apperr.Wrap(apperr.CodeDBMigrateFailed, err, map[string]string{"name": m.Name})
	}
	if err := tx.Commit(); err != nil {
		return apperr.Wrap(apperr.CodeDBMigrateFailed, err, map[string]string{"name": m.Name})
	}
	return nil
}

func preview(stmt string) string {
	stmt = strings.Join(strings.Fields(stmt), " ")
	if len(stmt) > 80 {
		return stmt[:80]
	}
	return stmt
}
