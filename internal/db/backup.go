package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alexbelweb/flibustahub/internal/apperr"
)

const backupKeep = 3

// Backup writes a consistent snapshot via VACUUM INTO and keeps the last 3 files.
func (d *DB) Backup(ctx context.Context) (string, error) {
	if d == nil || d.Write == nil {
		return "", apperr.New(apperr.CodeDBBackupFailed, nil)
	}
	if err := os.MkdirAll(d.backupsDir, 0o755); err != nil {
		return "", apperr.Wrap(apperr.CodeDBBackupFailed, err, nil)
	}
	name := fmt.Sprintf("catalog-%s.sqlite", d.now().Format("20060102-150405"))
	dest := filepath.Join(d.backupsDir, name)
	slash := filepath.ToSlash(dest)
	if _, err := d.Write.ExecContext(ctx, "VACUUM INTO ?", slash); err != nil {
		quoted := strings.ReplaceAll(slash, "'", "''")
		if _, err2 := d.Write.ExecContext(ctx, "VACUUM INTO '"+quoted+"'"); err2 != nil {
			return "", apperr.Wrap(apperr.CodeDBBackupFailed, err, nil)
		}
	}
	if err := pruneBackups(d.backupsDir); err != nil {
		return dest, apperr.Wrap(apperr.CodeDBBackupFailed, err, nil)
	}
	return dest, nil
}

func pruneBackups(dir string) error {
	matches, err := filepath.Glob(filepath.Join(dir, "catalog-*.sqlite"))
	if err != nil {
		return err
	}
	sort.Strings(matches)
	if len(matches) <= backupKeep {
		return nil
	}
	for _, extra := range matches[:len(matches)-backupKeep] {
		if err := os.Remove(extra); err != nil {
			return err
		}
	}
	return nil
}
