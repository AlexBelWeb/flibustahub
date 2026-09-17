package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	_ "modernc.org/sqlite"
)

const (
	readMaxOpen  = 4
	writeMaxOpen = 1
)

// DB holds the reader and writer pools for one catalog file.
type DB struct {
	Read  *sql.DB
	Write *sql.DB

	path       string
	backupsDir string
	log        *slog.Logger
	now        func() time.Time
	source     fs.FS

	mu            sync.Mutex
	recovering    bool
	recoverErr    error
	recoverCancel context.CancelFunc
	recoverDone   chan struct{}
}

// Options control Open.
type Options struct {
	Path       string
	BackupsDir string
	Log        *slog.Logger
	Now        func() time.Time
	Migrations fs.FS
}

// Open registers SQL functions, opens read/write pools, applies PRAGMA via DSN, and migrates.
func Open(ctx context.Context, opt Options) (*DB, error) {
	if err := registerFunctions(); err != nil {
		return nil, apperr.Wrap(apperr.CodeDBOpenFailed, err, nil)
	}
	if opt.Log == nil {
		opt.Log = slog.Default()
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	if opt.Path == "" {
		return nil, apperr.New(apperr.CodeDBOpenFailed, nil)
	}
	if err := os.MkdirAll(filepath.Dir(opt.Path), 0o755); err != nil {
		return nil, apperr.Wrap(apperr.CodeDBOpenFailed, err, nil)
	}
	if opt.BackupsDir != "" {
		if err := os.MkdirAll(opt.BackupsDir, 0o755); err != nil {
			return nil, apperr.Wrap(apperr.CodeDBOpenFailed, err, nil)
		}
	}

	dsn := fileDSN(opt.Path)
	write, err := sqlOpen(dsn)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeDBOpenFailed, err, nil)
	}
	configurePool(write, writeMaxOpen)
	read, err := sqlOpen(dsn)
	if err != nil {
		_ = write.Close()
		return nil, apperr.Wrap(apperr.CodeDBOpenFailed, err, nil)
	}
	configurePool(read, readMaxOpen)

	d := &DB{
		Read:       read,
		Write:      write,
		path:       opt.Path,
		backupsDir: opt.BackupsDir,
		log:        opt.Log,
		now:        opt.Now,
		source:     opt.Migrations,
	}
	d.reportPragmas(ctx)
	if err := applyMigrations(ctx, d, opt.Migrations); err != nil {
		_ = d.Close()
		return nil, err
	}
	if err := d.startIndexRecovery(ctx); err != nil {
		_ = d.Close()
		return nil, apperr.Wrap(apperr.CodeDBOpenFailed, err, nil)
	}
	return d, nil
}

func sqlOpen(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func configurePool(db *sql.DB, maxOpen int) {
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxOpen)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)
}

func (d *DB) reportPragmas(ctx context.Context) {
	type row struct {
		name string
		want string
	}
	want := []row{
		{"journal_mode", "wal"},
		{"foreign_keys", "1"},
		{"busy_timeout", "30000"},
		{"cache_size", "-64000"},
		{"mmap_size", "268435456"},
		{"temp_store", "2"},
	}
	for _, pool := range []*sql.DB{d.Write, d.Read} {
		for _, p := range want {
			got, err := readPragma(ctx, pool, p.name)
			if err != nil {
				d.log.Warn("pragma read failed", "pragma", p.name, "err", err)
				continue
			}
			if !strings.EqualFold(got, p.want) {
				d.log.Warn("pragma not applied", "pragma", p.name, "want", p.want, "got", got)
			}
		}
	}
}

func readPragma(ctx context.Context, pool *sql.DB, name string) (string, error) {
	var v any
	if err := pool.QueryRowContext(ctx, "PRAGMA "+name).Scan(&v); err != nil {
		return "", err
	}
	switch t := v.(type) {
	case []byte:
		return string(t), nil
	default:
		return fmt.Sprint(t), nil
	}
}

// Close checkpoints WAL on the writer and closes both pools.
func (d *DB) Close() error {
	if d == nil {
		return nil
	}
	d.stopIndexRecovery()
	var first error
	if d.Write != nil {
		if _, err := d.Write.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil && first == nil {
			first = err
		}
		if err := d.Write.Close(); err != nil && first == nil {
			first = err
		}
		d.Write = nil
	}
	if d.Read != nil {
		if err := d.Read.Close(); err != nil && first == nil {
			first = err
		}
		d.Read = nil
	}
	return first
}

// Path is the catalog file path.
func (d *DB) Path() string {
	if d == nil {
		return ""
	}
	return d.path
}
