package handlers

import (
	"context"

	"github.com/alexbelweb/flibustahub/internal/services/maintenance"
)

func (a *App) OptimizeDatabase() (maintenance.OptimizeResult, error) {
	return a.svc.OptimizeDatabase(context.Background())
}

func (a *App) CreateCatalogBackup() (maintenance.BackupResult, error) {
	return a.svc.CreateCatalogBackup(context.Background())
}

func (a *App) DatabaseMaintenanceRunning() bool {
	return a.svc.DatabaseMaintenanceRunning()
}
