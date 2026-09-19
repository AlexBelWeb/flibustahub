package downloads

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/platform"
	"github.com/alexbelweb/flibustahub/internal/repositories"
)

const ReadingDir = "reading"

// Kind is one user-started file operation. Download and read of different editions may overlap.
type Kind string

const (
	KindDownload Kind = "download"
	KindRead     Kind = "read"
)

// Result is a finished download or unpack.
type Result struct {
	Path     string `json:"path"`
	FileName string `json:"fileName"`
}

// Store loads one edition for download or reading.
type Store interface {
	Edition(ctx context.Context, id int64) (repositories.EditionFile, error)
}

type opKey struct {
	id   int64
	kind Kind
}

type runningOp struct {
	cancel context.CancelFunc
	id     uint64
}

// Service streams a book out of a ZIP and writes it atomically.
type Service struct {
	store        Store
	libraryRoot  func() string
	downloadsDir func() string
	dataDir      func() string
	readerPath   func() string
	ready        func(ctx context.Context) error
	readyForce   func(ctx context.Context) error
	log          *slog.Logger
	now          func() time.Time
	mu           sync.Mutex
	ops          map[opKey]*runningOp
	unpacked     map[int64]string
	seq          uint64
}

func New(
	store Store,
	libraryRoot, downloadsDir, dataDir, readerPath func() string,
	ready, readyForce func(context.Context) error,
	log *slog.Logger,
) *Service {
	if log == nil {
		log = slog.Default()
	}
	if libraryRoot == nil {
		libraryRoot = func() string { return "" }
	}
	if downloadsDir == nil {
		downloadsDir = func() string { return "" }
	}
	if dataDir == nil {
		dataDir = func() string { return "" }
	}
	if readerPath == nil {
		readerPath = func() string { return "" }
	}
	if ready == nil {
		ready = func(context.Context) error { return apperr.New(apperr.CodeLibraryOffline, nil) }
	}
	if readyForce == nil {
		readyForce = ready
	}
	return &Service{
		store:        store,
		libraryRoot:  libraryRoot,
		downloadsDir: downloadsDir,
		dataDir:      dataDir,
		readerPath:   readerPath,
		ready:        ready,
		readyForce:   readyForce,
		log:          log,
		now:          time.Now,
		ops:          make(map[opKey]*runningOp),
		unpacked:     make(map[int64]string),
	}
}

func (s *Service) Stop() {
	s.mu.Lock()
	for _, op := range s.ops {
		if op != nil && op.cancel != nil {
			op.cancel()
		}
	}
	s.mu.Unlock()
}

func (s *Service) Download(ctx context.Context, editionID int64) (Result, error) {
	return s.run(ctx, editionID, KindDownload, s.saveDownload)
}

func (s *Service) Read(ctx context.Context, editionID int64) (Result, error) {
	return s.run(ctx, editionID, KindRead, s.openRead)
}

func (s *Service) Cancel(editionID int64, kind Kind) {
	s.mu.Lock()
	op := s.ops[opKey{id: editionID, kind: kind}]
	s.mu.Unlock()
	if op != nil && op.cancel != nil {
		op.cancel()
	}
}

func (s *Service) ShowInFolder(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return apperr.New(apperr.CodeOpenDirFailed, nil)
	}
	if err := platform.ShowInFolder(path); err != nil {
		return apperr.Wrap(apperr.CodeOpenDirFailed, err, nil)
	}
	return nil
}

func (s *Service) run(ctx context.Context, editionID int64, kind Kind, fn func(context.Context, repositories.EditionFile) (Result, error)) (Result, error) {
	if editionID <= 0 {
		return Result{}, apperr.New(apperr.CodeNotFound, map[string]string{"kind": "edition"})
	}
	if err := s.ready(ctx); err != nil {
		return Result{}, err
	}
	ed, err := s.loadEdition(ctx, editionID)
	if err != nil {
		return Result{}, err
	}
	opCtx, cancel := context.WithCancel(ctx)
	key := opKey{id: editionID, kind: kind}
	s.mu.Lock()
	if prev := s.ops[key]; prev != nil && prev.cancel != nil {
		prev.cancel()
	}
	s.seq++
	token := s.seq
	s.ops[key] = &runningOp{cancel: cancel, id: token}
	s.mu.Unlock()
	defer func() {
		cancel()
		s.mu.Lock()
		if s.ops[key] != nil && s.ops[key].id == token {
			delete(s.ops, key)
		}
		s.mu.Unlock()
	}()

	start := s.now()
	out, err := fn(opCtx, ed)
	elapsed := s.now().Sub(start)
	if elapsed >= time.Second {
		s.log.Info("storage read waited",
			"editionId", editionID,
			"kind", string(kind),
			"archive", ed.ArchiveName,
			"ms", elapsed.Milliseconds(),
		)
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return Result{}, apperr.New(apperr.CodeCancelled, nil)
		}
		return Result{}, err
	}
	return out, nil
}

func (s *Service) loadEdition(ctx context.Context, id int64) (repositories.EditionFile, error) {
	ed, err := s.store.Edition(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return repositories.EditionFile{}, apperr.New(apperr.CodeNotFound, map[string]string{"kind": "edition"})
	}
	if err != nil {
		return repositories.EditionFile{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	if !ed.Active {
		return repositories.EditionFile{}, apperr.New(apperr.CodeNotFound, map[string]string{"kind": "edition"})
	}
	return ed, nil
}

func (s *Service) saveDownload(ctx context.Context, ed repositories.EditionFile) (Result, error) {
	dir := strings.TrimSpace(s.downloadsDir())
	if err := ensureDir(dir); err != nil {
		return Result{}, err
	}
	name, dest, err := FileName(dir, ed.AuthorsText, ed.Title, ed.FileExt, strconv.FormatInt(ed.ID, 10))
	if err != nil {
		return Result{}, apperr.Wrap(apperr.CodeDownloadsDirUnusable, err, nil)
	}
	if err := s.extractTo(ctx, ed, dest); err != nil {
		return Result{}, err
	}
	return Result{Path: dest, FileName: name}, nil
}

func (s *Service) openRead(ctx context.Context, ed repositories.EditionFile) (Result, error) {
	s.mu.Lock()
	prev := s.unpacked[ed.ID]
	s.mu.Unlock()
	if prev != "" {
		if st, err := os.Stat(prev); err == nil && !st.IsDir() {
			if err := s.launch(prev); err != nil {
				return Result{Path: prev, FileName: filepath.Base(prev)}, withFile(err, prev)
			}
			return Result{Path: prev, FileName: filepath.Base(prev)}, nil
		}
	}
	dir := filepath.Join(s.dataDir(), ReadingDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	name, dest, err := FileName(dir, ed.AuthorsText, ed.Title, ed.FileExt, strconv.FormatInt(ed.ID, 10))
	if err != nil {
		return Result{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	if err := s.extractTo(ctx, ed, dest); err != nil {
		return Result{}, err
	}
	s.mu.Lock()
	s.unpacked[ed.ID] = dest
	s.mu.Unlock()
	if err := s.launch(dest); err != nil {
		return Result{Path: dest, FileName: name}, withFile(err, dest)
	}
	return Result{Path: dest, FileName: name}, nil
}

func (s *Service) extractTo(ctx context.Context, ed repositories.EditionFile, dest string) error {
	stream, err := openStream(s.libraryRoot(), ed)
	if err != nil {
		if apperr.As(err).Code == apperr.CodeArchiveMissing {
			if probeErr := s.readyForce(ctx); probeErr != nil {
				return probeErr
			}
		}
		return err
	}
	defer func() { _ = stream.Close() }()
	if err := writeAtomic(ctx, dest, stream); err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		return apperr.Wrap(apperr.CodeFB2Unreadable, err, map[string]string{
			"workId": strconv.FormatInt(ed.WorkID, 10),
		})
	}
	return nil
}

func (s *Service) launch(path string) error {
	reader := strings.TrimSpace(s.readerPath())
	if reader != "" {
		if err := platform.OpenFileWith(reader, path); err != nil {
			return apperr.Wrap(apperr.CodeReaderMissing, err, map[string]string{"path": reader})
		}
		return nil
	}
	if err := platform.OpenFile(path); err != nil {
		if errors.Is(err, platform.ErrNoAssociation) || runtime.GOOS != "windows" {
			return apperr.Wrap(apperr.CodeReaderUnavailable, err, nil)
		}
		return apperr.Wrap(apperr.CodeReaderUnavailable, err, nil)
	}
	return nil
}

func withFile(err error, file string) error {
	typed := apperr.As(err)
	if typed.Params == nil {
		typed.Params = map[string]string{}
	}
	typed.Params["file"] = file
	return typed
}

func ensureDir(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return apperr.New(apperr.CodeDownloadsDirUnusable, nil)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.CodeDownloadsDirUnusable, err, nil)
	}
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return apperr.New(apperr.CodeDownloadsDirUnusable, nil)
	}
	return nil
}

// CleanDir removes files from the session reading folder. Failures are WARN only.
func CleanDir(dir string, log *slog.Logger) {
	if log == nil {
		log = slog.Default()
	}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Warn("reading dir cleanup failed", "dir", dir, "err", err)
		}
		return
	}
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if err := os.RemoveAll(path); err != nil {
			log.Warn("reading file not removed", "path", path, "err", err)
		}
	}
}
