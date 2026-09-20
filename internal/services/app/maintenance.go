package app

import (
	"context"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/services/maintenance"
)

func (s *Service) OptimizeDatabase(ctx context.Context) (maintenance.OptimizeResult, error) {
	if s == nil || s.maint == nil {
		return maintenance.OptimizeResult{}, apperr.New(apperr.CodeDBOpenFailed, nil)
	}
	return s.maint.Optimize(ctx)
}

func (s *Service) CreateCatalogBackup(ctx context.Context) (maintenance.BackupResult, error) {
	if s == nil || s.maint == nil {
		return maintenance.BackupResult{}, apperr.New(apperr.CodeDBOpenFailed, nil)
	}
	return s.maint.Backup(ctx)
}

func (s *Service) DatabaseMaintenanceRunning() bool {
	if s == nil || s.maint == nil {
		return false
	}
	return s.maint.Running()
}

func (s *Service) CancelMaintenance() {
	if s == nil || s.maint == nil {
		return
	}
	s.maint.Cancel()
}
