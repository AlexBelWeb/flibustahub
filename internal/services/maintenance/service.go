// Package maintenance runs long catalog file operations: VACUUM and manual backups.
package maintenance

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/platform"
)

// OptimizeResult is the on-disk size change after VACUUM.
type OptimizeResult struct {
	BytesBefore int64 `json:"bytesBefore"`
	BytesAfter  int64 `json:"bytesAfter"`
	BytesFreed  int64 `json:"bytesFreed"`
}

// BackupResult is the path of a manual snapshot.
type BackupResult struct {
	Path string `json:"path"`
}

// Options wire catalog access and the operations that must block maintenance.
type Options struct {
	Catalog   func() *db.DB
	Importing func() bool
	Warming   func() bool
	DiskFree  func(string) (uint64, error)
	Hold      <-chan struct{}
	Log       *slog.Logger
}

// Service serializes optimize and backup.
type Service struct {
	catalog   func() *db.DB
	importing func() bool
	warming   func() bool
	diskFree  func(string) (uint64, error)
	hold      <-chan struct{}
	log       *slog.Logger
	running   atomic.Bool
	cancelMu  sync.Mutex
	cancel    context.CancelFunc
}

func New(opt Options) *Service {
	s := &Service{
		catalog:   opt.Catalog,
		importing: opt.Importing,
		warming:   opt.Warming,
		diskFree:  opt.DiskFree,
		hold:      opt.Hold,
		log:       opt.Log,
	}
	if s.catalog == nil {
		s.catalog = func() *db.DB { return nil }
	}
	if s.importing == nil {
		s.importing = func() bool { return false }
	}
	if s.warming == nil {
		s.warming = func() bool { return false }
	}
	if s.diskFree == nil {
		s.diskFree = platform.DiskFree
	}
	if s.log == nil {
		s.log = slog.Default()
	}
	return s
}

// Running is true while VACUUM or a manual backup is in flight.
func (s *Service) Running() bool {
	return s.running.Load()
}

// Cancel interrupts an in-flight optimize or backup so shutdown can finish.
func (s *Service) Cancel() {
	s.cancelMu.Lock()
	cancel := s.cancel
	s.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Service) setCancel(cancel context.CancelFunc) {
	s.cancelMu.Lock()
	s.cancel = cancel
	s.cancelMu.Unlock()
}

func (s *Service) begin() error {
	if s.importing() {
		return apperr.New(apperr.CodeImportInProgress, nil)
	}
	if s.warming() {
		return apperr.New(apperr.CodeCoverWarmupInProgress, nil)
	}
	if !s.running.CompareAndSwap(false, true) {
		return apperr.New(apperr.CodeDBMaintenanceBusy, nil)
	}
	if s.importing() {
		s.running.Store(false)
		return apperr.New(apperr.CodeImportInProgress, nil)
	}
	if s.warming() {
		s.running.Store(false)
		return apperr.New(apperr.CodeCoverWarmupInProgress, nil)
	}
	return nil
}

func (s *Service) end() {
	s.setCancel(nil)
	s.running.Store(false)
}

func (s *Service) waitHold() {
	if s.hold != nil {
		<-s.hold
	}
}

func spaceNeed(before int64) uint64 {
	if before <= 0 {
		return 0
	}
	return uint64(before) * 2
}

func clampFreed(before, after int64) int64 {
	freed := before - after
	if freed < 0 {
		return 0
	}
	return freed
}

func (s *Service) run(ctx context.Context, fn func(context.Context) error) error {
	if err := s.begin(); err != nil {
		return err
	}
	defer s.end()
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.setCancel(cancel)
	s.waitHold()
	return fn(runCtx)
}

// Optimize runs VACUUM, ANALYZE and a WAL truncate. There is no finer progress.
func (s *Service) Optimize(ctx context.Context) (OptimizeResult, error) {
	var out OptimizeResult
	err := s.run(ctx, func(ctx context.Context) error {
		d := s.catalog()
		if d == nil {
			return apperr.New(apperr.CodeDBOpenFailed, nil)
		}
		before, err := d.FileSize()
		if err != nil {
			return err
		}
		free, err := s.diskFree(d.Path())
		if err != nil {
			return apperr.Wrap(apperr.CodeDBOptimizeFailed, err, nil)
		}
		need := spaceNeed(before)
		if need > 0 && free < need {
			return apperr.New(apperr.CodeDBNoSpace, map[string]string{
				"need": strconv.FormatUint(need, 10),
				"have": strconv.FormatUint(free, 10),
			})
		}
		if err := d.Optimize(ctx); err != nil {
			return err
		}
		after, err := d.FileSize()
		if err != nil {
			return err
		}
		out = OptimizeResult{
			BytesBefore: before,
			BytesAfter:  after,
			BytesFreed:  clampFreed(before, after),
		}
		s.log.Info("catalog optimized", "bytes_before", before, "bytes_after", after, "bytes_freed", out.BytesFreed)
		return nil
	})
	if err != nil {
		return OptimizeResult{}, err
	}
	return out, nil
}

// Backup writes a snapshot with reason manual.
func (s *Service) Backup(ctx context.Context) (BackupResult, error) {
	var out BackupResult
	err := s.run(ctx, func(ctx context.Context) error {
		d := s.catalog()
		if d == nil {
			return apperr.New(apperr.CodeDBOpenFailed, nil)
		}
		path, err := d.Backup(ctx, db.BackupReasonManual)
		if err != nil {
			return err
		}
		out = BackupResult{Path: path}
		s.log.Info("catalog backup created", "reason", db.BackupReasonManual)
		return nil
	})
	if err != nil {
		return BackupResult{}, err
	}
	return out, nil
}
