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

// legacyBackupGlob is the pre-reason name catalog-YYYYMMDD-HHMMSS.sqlite.
// It must not match catalog-{reason}-*.sqlite.
const legacyBackupGlob = "catalog-????????-??????.sqlite"

const (
	BackupReasonMigration = "migration"
	BackupReasonINPX      = "inpx"
	BackupReasonPersonal  = "personal"
	BackupReasonManual    = "manual"
)

// Backup writes a consistent snapshot via VACUUM INTO and keeps the last 3 files per reason.
func (d *DB) Backup(ctx context.Context, reason string) (string, error) {
	if d == nil || d.Write == nil {
		return "", apperr.New(apperr.CodeDBBackupFailed, nil)
	}
	if !validBackupReason(reason) {
		return "", apperr.New(apperr.CodeDBBackupFailed, nil)
	}
	if err := os.MkdirAll(d.backupsDir, 0o755); err != nil {
		return "", apperr.Wrap(apperr.CodeDBBackupFailed, err, nil)
	}
	name := fmt.Sprintf("catalog-%s-%s.sqlite", reason, d.now().Format("20060102-150405"))
	dest := filepath.Join(d.backupsDir, name)
	slash := filepath.ToSlash(dest)
	if _, err := d.Write.ExecContext(ctx, "VACUUM INTO ?", slash); err != nil {
		quoted := strings.ReplaceAll(slash, "'", "''")
		if _, err2 := d.Write.ExecContext(ctx, "VACUUM INTO '"+quoted+"'"); err2 != nil {
			return "", apperr.Wrap(apperr.CodeDBBackupFailed, err, nil)
		}
	}
	if err := pruneBackups(d.backupsDir, reason); err != nil {
		return dest, apperr.Wrap(apperr.CodeDBBackupFailed, err, nil)
	}
	return dest, nil
}

func validBackupReason(reason string) bool {
	if reason == "" {
		return false
	}
	for _, r := range reason {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

func pruneBackups(dir, reason string) error {
	if err := pruneGlob(dir, "catalog-"+reason+"-*.sqlite", backupKeep); err != nil {
		return err
	}
	return pruneGlob(dir, legacyBackupGlob, 0)
}

func pruneGlob(dir, pattern string, keep int) error {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return err
	}
	sort.Strings(matches)
	if keep < 0 {
		keep = 0
	}
	if len(matches) <= keep {
		return nil
	}
	for _, extra := range matches[:len(matches)-keep] {
		if err := os.Remove(extra); err != nil {
			return err
		}
	}
	return nil
}
