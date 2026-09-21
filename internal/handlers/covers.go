package handlers

import (
	"context"

	"github.com/alexbelweb/flibustahub/internal/services/catalog"
	"github.com/alexbelweb/flibustahub/internal/services/covers"
)

func (a *App) GetWorkDetails(id int64) (catalog.WorkDetails, error) {
	return a.svc.GetWorkDetails(context.Background(), id)
}

func (a *App) RecordViewed(id int64) error {
	return a.svc.RecordViewed(context.Background(), id)
}

func (a *App) GetAnnotation(id int64) (covers.Annotation, error) {
	return a.svc.GetAnnotation(context.Background(), id)
}

func (a *App) ClearCoverCache() error {
	return a.svc.ClearCoverCache()
}

func (a *App) CoverWarmupPreview() (int, error) {
	return a.svc.CoverWarmupPreview(context.Background())
}

func (a *App) StartCoverWarmup() error {
	return a.svc.StartCoverWarmup(context.Background())
}

func (a *App) StopCoverWarmup() {
	a.svc.StopCoverWarmup()
}

func (a *App) WindowAway() {
	a.svc.WindowAway()
}

func (a *App) WindowBack() {
	a.svc.WindowBack()
}

func (a *App) CoverWarmupProgress() covers.Progress {
	return a.svc.CoverWarmupProgress()
}
