package app

import (
	"context"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/services/catalog"
	"github.com/alexbelweb/flibustahub/internal/services/covers"
)

func (s *Service) coversSvc() (*covers.Service, error) {
	s.openMu.Lock()
	c := s.covers
	s.openMu.Unlock()
	if c == nil {
		return nil, apperr.New(apperr.CodeDBOpenFailed, nil)
	}
	return c, nil
}

func (s *Service) SetCoverEvents(fn func(covers.Progress)) {
	s.openMu.Lock()
	s.coverEmit = fn
	s.openMu.Unlock()
}

func (s *Service) SetHTTPAddr(addr string) {
	s.openMu.Lock()
	s.httpAddr = addr
	s.openMu.Unlock()
}

func (s *Service) emitCoverProgress(p covers.Progress) {
	s.openMu.Lock()
	fn := s.coverEmit
	s.openMu.Unlock()
	if fn != nil {
		fn(p)
	}
}

func parsePrio(s string) covers.Priority {
	switch s {
	case "open":
		return covers.PrioOpen
	case "prefetch":
		return covers.PrioPrefetch
	default:
		return covers.PrioVisible
	}
}

func (s *Service) ServeCover(ctx context.Context, workID int64, prio string) (covers.Hit, error) {
	c, err := s.coversSvc()
	if err != nil {
		return covers.Hit{}, err
	}
	return c.ServeCover(ctx, workID, parsePrio(prio))
}

func (s *Service) GetAnnotation(ctx context.Context, workID int64) (covers.Annotation, error) {
	c, err := s.coversSvc()
	if err != nil {
		return covers.Annotation{}, err
	}
	return c.Annotation(ctx, workID)
}

func (s *Service) ClearCoverCache() error {
	c, err := s.coversSvc()
	if err != nil {
		return err
	}
	if err := c.ClearCache(); err != nil {
		return apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	return nil
}

func (s *Service) CoverWarmupPreview(ctx context.Context) (int, error) {
	c, err := s.coversSvc()
	if err != nil {
		return 0, err
	}
	return c.WarmupPreview(ctx)
}

func (s *Service) StartCoverWarmup(ctx context.Context) error {
	if s.DatabaseMaintenanceRunning() {
		return apperr.New(apperr.CodeDBMaintenanceBusy, nil)
	}
	c, err := s.coversSvc()
	if err != nil {
		return err
	}
	s.holdCache()
	started, err := c.StartWarmup(ctx)
	if err != nil || !started {
		s.releaseAfterHeavy()
		return err
	}
	return nil
}

func (s *Service) StopCoverWarmup() {
	c, err := s.coversSvc()
	if err != nil {
		return
	}
	c.StopWarmup()
}

func (s *Service) CoverWarmupProgress() covers.Progress {
	c, err := s.coversSvc()
	if err != nil {
		return covers.Progress{}
	}
	return c.WarmupProgress()
}

func (s *Service) GetWorkDetails(ctx context.Context, id int64) (catalog.WorkDetails, error) {
	c, err := s.catalogSvc()
	if err != nil {
		return catalog.WorkDetails{}, err
	}
	return c.GetWorkDetails(ctx, id)
}

func (s *Service) RecordViewed(ctx context.Context, id int64) error {
	c, err := s.catalogSvc()
	if err != nil {
		return err
	}
	return c.RecordViewed(ctx, id)
}
