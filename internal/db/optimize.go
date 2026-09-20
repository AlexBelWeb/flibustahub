package db

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexbelweb/flibustahub/internal/apperr"
)

// FileSize is the catalog file size on disk, not including -wal/-shm.
func (d *DB) FileSize() (int64, error) {
	if d == nil || d.path == "" {
		return 0, apperr.New(apperr.CodeDBOpenFailed, nil)
	}
	info, err := os.Stat(d.path)
	if err != nil {
		return 0, apperr.Wrap(apperr.CodeDBOpenFailed, err, nil)
	}
	return info.Size(), nil
}

// Optimize rebuilds the catalog file, refreshes stats, then truncates the WAL.
// VACUUM copies into a temporary database; the maintenance connection sets
// temp_store=FILE so DSN temp_store=MEMORY cannot send a 700 MB rebuild into
// RAM. The process temp directory is already the catalog folder (set once
// before the pools open). temp_store is restored before the connection
// returns to the write pool.
func (d *DB) Optimize(ctx context.Context) error {
	return d.optimize(ctx, true)
}

func (d *DB) optimize(ctx context.Context, pinTempToDisk bool) error {
	if d == nil || d.Write == nil {
		return apperr.New(apperr.CodeDBOpenFailed, nil)
	}
	conn, err := d.Write.Conn(ctx)
	if err != nil {
		return apperr.Wrap(apperr.CodeDBOptimizeFailed, err, nil)
	}
	defer func() { _ = conn.Close() }()
	if pinTempToDisk {
		if _, err := conn.ExecContext(ctx, "PRAGMA temp_store=FILE"); err != nil {
			return apperr.Wrap(apperr.CodeDBOptimizeFailed, err, nil)
		}
		defer func() { _ = RestoreWorkPragmas(ctx, conn) }()
	}
	if _, err := conn.ExecContext(ctx, "VACUUM"); err != nil {
		return apperr.Wrap(apperr.CodeDBOptimizeFailed, err, nil)
	}
	if _, err := conn.ExecContext(ctx, "ANALYZE"); err != nil {
		return apperr.Wrap(apperr.CodeDBOptimizeFailed, err, nil)
	}
	var busy, logFrames, checkpointed int
	if err := conn.QueryRowContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &logFrames, &checkpointed); err != nil {
		return apperr.Wrap(apperr.CodeDBOptimizeFailed, err, nil)
	}
	return nil
}

func setTempStoreDirectory(dir string) error {
	bootstrap, err := sql.Open("sqlite", "file:flibustahub-tempdir?mode=memory")
	if err != nil {
		return err
	}
	defer func() { _ = bootstrap.Close() }()
	if err := bootstrap.Ping(); err != nil {
		return err
	}
	stmt := "PRAGMA temp_store_directory=" + quoteSQLString(filepath.ToSlash(dir))
	_, err = bootstrap.Exec(stmt)
	return err
}

func removeOrphanTempDBs(dir string, log *slog.Logger) {
	if log == nil {
		log = slog.Default()
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(strings.ToLower(name), "etilqs_") {
			continue
		}
		path := filepath.Join(dir, name)
		if err := os.Remove(path); err != nil {
			log.Warn("orphan sqlite temp file not removed", "name", name, "err", err)
			continue
		}
		log.Info("removed orphan sqlite temp file", "name", name)
	}
}

func quoteSQLString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
