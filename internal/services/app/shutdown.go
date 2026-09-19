package app

import (
	"context"
	"time"
)

const shutdownBudget = 5 * time.Second

func remaining(deadline time.Time) time.Duration {
	d := time.Until(deadline)
	if d < 0 {
		return 0
	}
	return d
}

// Shutdown cancels in-flight work, waits with a shared budget, stops HTTP, then
// closes the catalog. It is safe to call more than once and from more than one
// exit path.
func (s *Service) Shutdown(stopHTTP func(context.Context) error) {
	s.shutdownOnce.Do(func() {
		deadline := time.Now().Add(shutdownBudget)
		s.beginShutdown()
		s.CancelImport()
		s.stopCovers()
		s.waitNamedUntil(deadline, "import", func() bool { return !s.IsImporting() })
		s.waitCoversUntil(deadline)
		s.waitNamedUntil(deadline, "catalog-open", func() bool { return !s.isOpening() })
		if stopHTTP != nil {
			ctx, cancel := context.WithTimeout(context.Background(), remaining(deadline))
			err := stopHTTP(ctx)
			cancel()
			if err != nil {
				s.Logger().Warn("shutdown timed out", "task", "http", "err", err)
			}
		}
		s.closeCatalogUntil(deadline)
	})
}

func (s *Service) beginShutdown() {
	s.openMu.Lock()
	s.stopping = true
	s.openMu.Unlock()
	s.Logger().Info("shutting down")
}

func (s *Service) stopCovers() {
	s.openMu.Lock()
	c := s.covers
	s.openMu.Unlock()
	if c == nil {
		return
	}
	s.Logger().Info("covers cancelled")
	c.Stop()
}

func (s *Service) waitCoversUntil(deadline time.Time) {
	s.openMu.Lock()
	c := s.covers
	s.openMu.Unlock()
	if c == nil {
		return
	}
	c.Wait(remaining(deadline))
}

func (s *Service) closeCatalogUntil(deadline time.Time) {
	s.openMu.Lock()
	defer s.openMu.Unlock()
	s.closeCatalogLocked(remaining(deadline))
}

func (s *Service) isOpening() bool {
	s.openMu.Lock()
	defer s.openMu.Unlock()
	return s.opening
}

func (s *Service) waitNamedUntil(deadline time.Time, task string, done func() bool) {
	if done() {
		return
	}
	left := remaining(deadline)
	if left == 0 {
		s.Logger().Warn("shutdown timed out", "task", task)
		return
	}
	end := time.Now().Add(left)
	for !done() {
		if time.Now().After(end) {
			s.Logger().Warn("shutdown timed out", "task", task)
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}
