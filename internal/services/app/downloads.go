package app

import (
	"context"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/repositories"
	"github.com/alexbelweb/flibustahub/internal/services/covers"
	"github.com/alexbelweb/flibustahub/internal/services/downloads"
)

func (s *Service) downloadsSvc() (*downloads.Service, error) {
	s.openMu.Lock()
	d := s.downloads
	s.openMu.Unlock()
	if d == nil {
		return nil, apperr.New(apperr.CodeDBOpenFailed, nil)
	}
	return d, nil
}

func (s *Service) DownloadEdition(ctx context.Context, editionID int64) (downloads.Result, error) {
	d, err := s.downloadsSvc()
	if err != nil {
		return downloads.Result{}, err
	}
	return d.Download(ctx, editionID)
}

func (s *Service) ReadEdition(ctx context.Context, editionID int64) (downloads.Result, error) {
	d, err := s.downloadsSvc()
	if err != nil {
		return downloads.Result{}, err
	}
	return d.Read(ctx, editionID)
}

func (s *Service) CancelFileOp(editionID int64, kind string) {
	d, err := s.downloadsSvc()
	if err != nil {
		return
	}
	d.Cancel(editionID, downloads.Kind(kind))
}

func (s *Service) ShowInFolder(path string) error {
	return downloads.New(nil, nil, nil, nil, nil, nil, nil, s.Logger()).ShowInFolder(path)
}

func (s *Service) newDownloadsLocked() *downloads.Service {
	if s.catalog == nil {
		return nil
	}
	cfg := s.cfg
	cat := repositories.NewCatalog(s.catalog)
	ready := func(ctx context.Context) error {
		if s.storage == nil {
			return apperr.New(apperr.CodeLibraryOffline, nil)
		}
		return s.storage.Probe(ctx)
	}
	force := func(ctx context.Context) error {
		if s.storage == nil {
			return apperr.New(apperr.CodeLibraryOffline, nil)
		}
		return s.storage.ProbeForce(ctx)
	}
	return downloads.New(cat, func() string {
		return cfg.Live().LibraryRoot
	}, func() string {
		return cfg.Paths().DownloadsDir
	}, func() string {
		return cfg.Paths().DataDir
	}, func() string {
		return cfg.Live().ReaderPath
	}, ready, force, s.log)
}

func (s *Service) newCoversLocked() *covers.Service {
	if s.catalog == nil {
		return nil
	}
	cfg := s.cfg
	cat := repositories.NewCatalog(s.catalog)
	ready := func() error {
		if s.storage == nil {
			return apperr.New(apperr.CodeLibraryOffline, nil)
		}
		return s.storage.Probe(context.Background())
	}
	return covers.New(cat, func() string {
		return cfg.Paths().CoversDir
	}, func() string {
		return cfg.Live().LibraryRoot
	}, ready, s.log, nil, s.emitCoverProgress)
}
