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
// exit path. The focus channel is stopped first: this process will not show its
// window again, so a relaunch must not treat the acknowledgement as "already running".
// The lock stays until the catalog file is closed. A close that only times
// out does not drop the lock: the process exit releases it together with
// the database descriptors.
func (s *Service) Shutdown(stopHTTP func(context.Context) error) {
	s.shutdownOnce.Do(func() {
		s.stopInstanceFocus()
		deadline := time.Now().Add(s.stopBudget)
		s.beginShutdown()
		s.CancelImport()
		s.CancelMaintenance()
		s.stopCovers()
		s.stopDownloads()
		s.CleanupReading()
		s.waitNamedUntil(deadline, "import", func() bool { return !s.IsImporting() })
		s.waitNamedUntil(deadline, "maintenance", func() bool { return !s.DatabaseMaintenanceRunning() })
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
		// The lock guards the catalog. Drop it only after a confirmed close.
		// The focus channel was stopped at the start of this function.
		if err := s.closeCatalogUntil(deadline); err != nil {
			s.Logger().Warn("catalog close did not finish; instance lock held until exit", "err", err)
			return
		}
		s.unlockInstance()
	})
}

func (s *Service) beginShutdown() {
	s.openMu.Lock()
	s.stopping = true
	s.openMu.Unlock()
	s.Logger().Info("shutting down")
}

func (s *Service) stopDownloads() {
	s.openMu.Lock()
	d := s.downloads
	s.openMu.Unlock()
	if d == nil {
		return
	}
	d.Stop()
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

func (s *Service) closeCatalogUntil(deadline time.Time) error {
	s.openMu.Lock()
	defer s.openMu.Unlock()
	return s.closeCatalogLocked(remaining(deadline))
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
