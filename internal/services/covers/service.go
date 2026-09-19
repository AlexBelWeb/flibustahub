package covers

import (
	"archive/zip"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/fb2"
	"github.com/alexbelweb/flibustahub/internal/repositories"
)

const workers = 2

// Store is the catalog persistence used while extracting covers and annotations.
type Store interface {
	PrimaryEdition(ctx context.Context, workID int64) (repositories.EditionFile, error)
	Annotation(ctx context.Context, workID int64) (repositories.AnnotationRow, error)
	SaveAnnotation(ctx context.Context, workID int64, text *string, at time.Time) error
	WarmupItems(ctx context.Context) ([]repositories.WarmupItem, error)
}

// Annotation is the stored (or just extracted) text for a work.
type Annotation struct {
	Text    *string `json:"text,omitempty"`
	Checked bool    `json:"checked"`
}

// Progress is the warmup counters shown in settings.
type Progress struct {
	Total   int  `json:"total"`
	Done    int  `json:"done"`
	Running bool `json:"running"`
}

type sessionFail struct {
	err    error
	warned bool
}

// Service extracts covers and annotations through a two-worker priority queue.
type Service struct {
	store       Store
	coversDir   func() string
	libraryRoot func() string
	log         *slog.Logger
	now         func() time.Time
	onProgress  func(Progress)
	q           *queue
	runCtx      context.Context
	runCancel   context.CancelFunc
	workers     sync.WaitGroup
	stop        chan struct{}
	stopOnce    sync.Once
	failMu      sync.Mutex
	fails       map[int64]sessionFail
	warmMu      sync.Mutex
	warmCancel  context.CancelFunc
	warmRunning atomic.Bool
	warmTotal   atomic.Int64
	warmDone    atomic.Int64
}

func New(
	store Store,
	coversDir func() string,
	libraryRoot func() string,
	log *slog.Logger,
	now func() time.Time,
	onProgress func(Progress),
) *Service {
	if log == nil {
		log = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	if coversDir == nil {
		coversDir = func() string { return "" }
	}
	if libraryRoot == nil {
		libraryRoot = func() string { return "" }
	}
	runCtx, cancel := context.WithCancel(context.Background())
	s := &Service{
		store:       store,
		coversDir:   coversDir,
		libraryRoot: libraryRoot,
		log:         log,
		now:         now,
		onProgress:  throttleProgress(onProgress),
		q:           newQueue(),
		runCtx:      runCtx,
		runCancel:   cancel,
		stop:        make(chan struct{}),
		fails:       make(map[int64]sessionFail),
	}
	for i := 0; i < workers; i++ {
		s.workers.Add(1)
		go s.worker()
	}
	return s
}

func (s *Service) Stop() {
	s.stopOnce.Do(func() {
		s.StopWarmup()
		if s.runCancel != nil {
			s.runCancel()
		}
		close(s.stop)
		s.q.Close()
	})
}

// Wait blocks until workers exit or timeout, then WARNs with task "covers".
func (s *Service) Wait(timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		s.workers.Wait()
		close(done)
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return
	case <-timer.C:
	}
	select {
	case <-done:
	default:
		s.log.Warn("shutdown timed out", "task", "covers")
	}
}

func (s *Service) ForgetSessionFails() {
	s.failMu.Lock()
	s.fails = make(map[int64]sessionFail)
	s.failMu.Unlock()
}

func (s *Service) worker() {
	defer s.workers.Done()
	for {
		j := s.q.Take()
		if j == nil {
			return
		}
		s.q.Done(j, s.run(s.runCtx, j.workID, j.need))
	}
}

func (s *Service) ServeCover(ctx context.Context, workID int64, prio Priority) (Hit, error) {
	if workID <= 0 {
		return Hit{Missing: true}, apperr.New(apperr.CodeNotFound, map[string]string{"kind": "work"})
	}
	if hit := Lookup(s.coversDir(), workID); hit.Found {
		return hit, nil
	}
	if err := s.ensure(ctx, workID, NeedCover, prio); err != nil {
		return Hit{Missing: true}, err
	}
	hit := Lookup(s.coversDir(), workID)
	if !hit.Found {
		hit.Missing = true
	}
	return hit, nil
}

func (s *Service) Annotation(ctx context.Context, workID int64) (Annotation, error) {
	if workID <= 0 {
		return Annotation{}, apperr.New(apperr.CodeNotFound, map[string]string{"kind": "work"})
	}
	need := NeedAnnotation
	if hit := Lookup(s.coversDir(), workID); !hit.Found {
		need |= NeedCover
	}
	if err := s.ensure(ctx, workID, need, PrioOpen); err != nil {
		return Annotation{}, err
	}
	return s.readAnnotation(ctx, workID)
}

func (s *Service) readAnnotation(ctx context.Context, workID int64) (Annotation, error) {
	row, err := s.store.Annotation(ctx, workID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Annotation{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	out := Annotation{Checked: row.CheckedAt.Valid}
	if row.Text.Valid {
		t := row.Text.String
		out.Text = &t
	}
	return out, nil
}

func (s *Service) ensure(ctx context.Context, workID int64, need Need, prio Priority) error {
	if err := s.sessionErr(workID); err != nil {
		return err
	}
	need = s.unsatisfied(ctx, workID, need)
	if need == 0 {
		return nil
	}
	ch := s.q.Enqueue(ctx, workID, need, prio)
	select {
	case <-ctx.Done():
		s.q.Cancel(workID, ch)
		return ctx.Err()
	case err := <-ch:
		if err != nil {
			return err
		}
		if left := s.unsatisfied(ctx, workID, need); left != 0 {
			if se := s.sessionErr(workID); se != nil {
				return se
			}
		}
		return nil
	}
}

func (s *Service) unsatisfied(ctx context.Context, workID int64, need Need) Need {
	var left Need
	if need.has(NeedCover) {
		if hit := Lookup(s.coversDir(), workID); !hit.Found {
			left |= NeedCover
		}
	}
	if need.has(NeedAnnotation) {
		row, err := s.store.Annotation(ctx, workID)
		if err != nil || !row.CheckedAt.Valid {
			left |= NeedAnnotation
		}
	}
	return left
}

func (s *Service) run(ctx context.Context, workID int64, need Need) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.sessionErr(workID); err != nil {
		return err
	}
	need = s.unsatisfied(ctx, workID, need)
	if need == 0 {
		return nil
	}
	if err := s.libraryReady(); err != nil {
		s.remember(workID, err, false)
		return err
	}
	ed, err := s.store.PrimaryEdition(ctx, workID)
	if errors.Is(err, sql.ErrNoRows) {
		out := apperr.New(apperr.CodeNotFound, map[string]string{"kind": "work"})
		s.remember(workID, out, false)
		return out
	}
	if err != nil {
		out := apperr.Wrap(apperr.CodeInternal, err, nil)
		s.remember(workID, out, true)
		return out
	}
	zipPath, err := archivePath(s.libraryRoot(), ed.ArchiveName)
	if err != nil {
		s.remember(workID, err, false)
		return err
	}
	if _, err := os.Stat(zipPath); err != nil {
		out := apperr.New(apperr.CodeArchiveMissing, map[string]string{"archive": ed.ArchiveName})
		s.remember(workID, out, false)
		return out
	}
	start := time.Now()
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		out := apperr.Wrap(apperr.CodeFB2Unreadable, err, map[string]string{"workId": formatID(workID)})
		s.remember(workID, out, true)
		return out
	}
	defer func() { _ = zr.Close() }()
	if err := ctx.Err(); err != nil {
		return err
	}

	book, err := fb2.ParseZip(&zr.Reader, ed.FileName, ed.FileExt)
	elapsed := time.Since(start)
	if elapsed >= time.Second {
		s.log.Info("storage read waited",
			"workId", workID,
			"archive", ed.ArchiveName,
			"ms", elapsed.Milliseconds(),
		)
	}
	if err != nil {
		out := apperr.Wrap(apperr.CodeFB2Unreadable, err, map[string]string{"workId": formatID(workID)})
		s.remember(workID, out, true)
		return out
	}

	dir := s.coversDir()
	if need.has(NeedCover) {
		if book.HasCover {
			if err := WriteCover(dir, workID, book.Cover, book.CoverKind); err != nil {
				return apperr.Wrap(apperr.CodeInternal, err, nil)
			}
		} else {
			if book.UnrecognizedCover {
				s.log.Warn("cover bytes not a known image", "workId", workID)
			}
			if err := WriteNone(dir, workID); err != nil {
				return apperr.Wrap(apperr.CodeInternal, err, nil)
			}
		}
	}
	if need.has(NeedAnnotation) {
		var text *string
		if book.AnnotationRejected {
			s.log.Warn("annotation rejected as implausible",
				"workId", workID,
				"archive", ed.ArchiveName,
				"encoding", book.Encoding,
			)
		} else if strings.TrimSpace(book.Annotation) != "" {
			a := book.Annotation
			text = &a
		}
		if err := s.store.SaveAnnotation(ctx, workID, text, s.now()); err != nil {
			return apperr.Wrap(apperr.CodeInternal, err, nil)
		}
	}
	return nil
}

func (s *Service) libraryReady() error {
	root := strings.TrimSpace(s.libraryRoot())
	if root == "" {
		return apperr.New(apperr.CodeLibraryOffline, nil)
	}
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return apperr.New(apperr.CodeLibraryOffline, nil)
	}
	return nil
}

func (s *Service) sessionErr(workID int64) error {
	s.failMu.Lock()
	defer s.failMu.Unlock()
	if f, ok := s.fails[workID]; ok {
		return f.err
	}
	return nil
}

func (s *Service) remember(workID int64, err error, warn bool) {
	if err == nil {
		return
	}
	s.failMu.Lock()
	defer s.failMu.Unlock()
	prev, ok := s.fails[workID]
	if ok && prev.warned {
		warn = false
	}
	s.fails[workID] = sessionFail{err: err, warned: prev.warned || warn}
	if warn {
		s.log.Warn("fb2 extract failed", "workId", workID, "err", err)
	}
}

func (s *Service) ClearCache() error {
	s.failMu.Lock()
	s.fails = make(map[int64]sessionFail)
	s.failMu.Unlock()
	return ClearDir(s.coversDir())
}

func (s *Service) WarmupPreview(ctx context.Context) (int, error) {
	items, err := s.store.WarmupItems(ctx)
	if err != nil {
		return 0, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	n := 0
	dir := s.coversDir()
	for _, it := range items {
		if !Lookup(dir, it.WorkID).Found {
			n++
		}
	}
	return n, nil
}

func (s *Service) StartWarmup(ctx context.Context) error {
	if !s.warmRunning.CompareAndSwap(false, true) {
		return nil
	}
	items, err := s.store.WarmupItems(ctx)
	if err != nil {
		s.warmRunning.Store(false)
		return apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	dir := s.coversDir()
	pending := make([]int64, 0, len(items))
	for _, it := range items {
		if Lookup(dir, it.WorkID).Found {
			continue
		}
		pending = append(pending, it.WorkID)
	}
	s.warmTotal.Store(int64(len(pending)))
	s.warmDone.Store(0)
	s.emitWarmup()
	if len(pending) == 0 {
		s.warmRunning.Store(false)
		s.emitWarmup()
		return nil
	}
	wctx, cancel := context.WithCancel(context.Background())
	s.warmMu.Lock()
	s.warmCancel = cancel
	s.warmMu.Unlock()
	go s.runWarmup(wctx, pending)
	return nil
}

func (s *Service) runWarmup(ctx context.Context, ids []int64) {
	defer func() {
		s.warmMu.Lock()
		s.warmCancel = nil
		s.warmMu.Unlock()
		s.warmRunning.Store(false)
		s.emitWarmup()
	}()
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		err := s.ensure(ctx, id, NeedCover, PrioWarmup)
		if errors.Is(err, context.Canceled) {
			return
		}
		s.warmDone.Add(1)
		s.emitWarmup()
	}
}

func (s *Service) StopWarmup() {
	s.warmMu.Lock()
	cancel := s.warmCancel
	s.warmMu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.q.DropUnstartedWarmup()
}

func (s *Service) WarmupProgress() Progress {
	return Progress{
		Total:   int(s.warmTotal.Load()),
		Done:    int(s.warmDone.Load()),
		Running: s.warmRunning.Load(),
	}
}

func (s *Service) emitWarmup() {
	if s.onProgress == nil {
		return
	}
	s.onProgress(s.WarmupProgress())
}

func archivePath(root, name string) (string, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	name = strings.TrimSpace(name)
	if root == "" {
		return "", apperr.New(apperr.CodeLibraryOffline, nil)
	}
	if name == "" || !fb2.ValidEntryName(name) {
		return "", apperr.New(apperr.CodeArchiveMissing, map[string]string{"archive": name})
	}
	p := filepath.Join(root, filepath.FromSlash(name))
	rel, err := filepath.Rel(root, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", apperr.New(apperr.CodeArchiveMissing, map[string]string{"archive": name})
	}
	return p, nil
}

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func throttleProgress(fn func(Progress)) func(Progress) {
	if fn == nil {
		return nil
	}
	var mu sync.Mutex
	var last time.Time
	return func(p Progress) {
		now := time.Now()
		mu.Lock()
		defer mu.Unlock()
		if !p.Running || p.Done >= p.Total {
			last = now
			fn(p)
			return
		}
		if !last.IsZero() && now.Sub(last) < 100*time.Millisecond {
			return
		}
		last = now
		fn(p)
	}
}
