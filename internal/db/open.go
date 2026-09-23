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

	waitIndexRecovery = 3 * time.Second
	waitPoolClose     = 3 * time.Second
)

func remaining(deadline time.Time) time.Duration {
	d := time.Until(deadline)
	if d < 0 {
		return 0
	}
	return d
}

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
	closed        bool
	recovering    bool
	recoverErr    error
	recoverCancel context.CancelFunc
	recoverDone   chan struct{}
	recoverWaited bool

	idle *cacheGate
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

	dir := filepath.Dir(opt.Path)
	removeOrphanTempDBs(dir, opt.Log)
	if err := setTempStoreDirectory(dir); err != nil {
		return nil, apperr.Wrap(apperr.CodeDBOpenFailed, err, nil)
	}

	dsn := fileDSN(opt.Path)
	write, read, err := openPools(ctx, dsn)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeDBOpenFailed, err, nil)
	}
	configurePool(write, writeMaxOpen)
	configurePool(read, readMaxOpen)

	d := &DB{
		Read:       read,
		Write:      write,
		path:       opt.Path,
		backupsDir: opt.BackupsDir,
		log:        opt.Log,
		now:        opt.Now,
		source:     opt.Migrations,
		idle:       newCacheGate(idleCacheDelay),
	}
	d.idle.owner = d
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

func openPools(ctx context.Context, dsn string) (*sql.DB, *sql.DB, error) {
	var write, read *sql.DB
	err := retryTransient(ctx, transientOpenBudget, transientLockIO, func() error {
		if write != nil {
			_ = write.Close()
			write = nil
		}
		if read != nil {
			_ = read.Close()
			read = nil
		}
		var openErr error
		write, openErr = sqlOpenOnce(ctx, dsn)
		if openErr != nil {
			return openErr
		}
		read, openErr = sqlOpenOnce(ctx, dsn)
		if openErr != nil {
			_ = write.Close()
			write = nil
			return openErr
		}
		return nil
	})
	if err != nil {
		if write != nil {
			_ = write.Close()
		}
		if read != nil {
			_ = read.Close()
		}
		return nil, nil, err
	}
	return write, read, nil
}

func sqlOpenOnce(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
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

// Close closes the read pool, checkpoints WAL on the writer, then closes the write pool.
func (d *DB) Close() error {
	return d.CloseWithin(waitIndexRecovery + waitPoolClose)
}

// CloseWithin is Close with a shared wait budget for shutdown.
func (d *DB) CloseWithin(budget time.Duration) error {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	idle := d.idle
	d.mu.Unlock()
	if idle != nil {
		idle.stop()
	}

	deadline := time.Now().Add(budget)
	d.stopIndexRecoveryUntil(deadline)

	d.mu.Lock()
	read, write := d.Read, d.Write
	d.Read, d.Write = nil, nil
	d.mu.Unlock()

	var first error
	if err := closePoolUntil(read, d.log, "db-read", deadline); err != nil {
		first = err
	}
	if write != nil {
		checkpointWAL(write, d.log)
		if err := closePoolUntil(write, d.log, "db-write", deadline); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func checkpointWAL(write *sql.DB, log *slog.Logger) {
	var busy, logFrames, checkpointed int
	err := write.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &logFrames, &checkpointed)
	if err != nil {
		if log != nil {
			log.Warn("wal checkpoint failed", "err", err)
		}
		return
	}
	if busy != 0 && log != nil {
		log.Warn("could not truncate WAL", "busy", busy, "log", logFrames, "checkpointed", checkpointed)
	}
}

func closePoolUntil(pool *sql.DB, log *slog.Logger, task string, deadline time.Time) error {
	if pool == nil {
		return nil
	}
	done := make(chan error, 1)
	go func() { done <- pool.Close() }()
	timer := time.NewTimer(remaining(deadline))
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-timer.C:
		select {
		case err := <-done:
			return err
		default:
			if log != nil {
				log.Warn("shutdown timed out", "task", task)
			}
			return nil
		}
	}
}

// Path is the catalog file path.
func (d *DB) Path() string {
	if d == nil {
		return ""
	}
	return d.path
}
