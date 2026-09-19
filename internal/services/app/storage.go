package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
	"github.com/alexbelweb/flibustahub/internal/services/downloads"
	"github.com/alexbelweb/flibustahub/internal/services/storage"
)

func (s *Service) SetStorageEvents(fn func(storage.Snapshot)) {
	s.openMu.Lock()
	s.storeEmit = fn
	s.openMu.Unlock()
}

func (s *Service) emitStorage(snap storage.Snapshot) {
	if snap.Available {
		s.openMu.Lock()
		c := s.covers
		s.openMu.Unlock()
		if c != nil {
			c.ForgetOfflineFails()
		}
	}
	s.openMu.Lock()
	fn := s.storeEmit
	s.openMu.Unlock()
	if fn != nil {
		fn(snap)
	}
}

func (s *Service) storageSnapshot() storage.Snapshot {
	if s.storage == nil {
		return storage.Snapshot{}
	}
	return s.storage.Snapshot()
}

func (s *Service) CheckStorage(ctx context.Context, force bool) storage.Snapshot {
	if s.storage == nil {
		return storage.Snapshot{}
	}
	return s.storage.Check(ctx, force)
}

func (s *Service) DismissDumpOffer() error {
	if s.storage == nil {
		return nil
	}
	return s.storage.DismissDumpOffer()
}

func (s *Service) AfterWindow() {
	s.CleanupReadingAsync()
	go s.CheckStorage(context.Background(), true)
}

func (s *Service) CleanupReading() {
	dir := filepath.Join(s.config().Paths().DataDir, downloads.ReadingDir)
	downloads.CleanDir(dir, s.Logger())
}

func (s *Service) CleanupReadingAsync() {
	go s.CleanupReading()
}

func (s *Service) SetDownloadsDir(path string) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return apperr.New(apperr.CodeDownloadsDirUnusable, nil)
	}
	info, err := os.Stat(path)
	if err != nil {
		return apperr.Wrap(apperr.CodeDownloadsDirUnusable, err, nil)
	}
	if !info.IsDir() {
		return apperr.New(apperr.CodeDownloadsDirUnusable, nil)
	}
	return s.config().Update(func(f *config.File) { f.DownloadsDir = path })
}

func (s *Service) SetReaderPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return s.config().Update(func(f *config.File) { f.ReaderPath = "" })
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return apperr.Wrap(apperr.CodeReaderMissing, err, map[string]string{"path": path})
	}
	if info.IsDir() {
		return apperr.New(apperr.CodeReaderMissing, map[string]string{"path": path})
	}
	return s.config().Update(func(f *config.File) { f.ReaderPath = path })
}

func (s *Service) ReaderPath() string {
	return s.config().Live().ReaderPath
}
