package handlers

import (
	"context"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/services/catalog"
	"github.com/alexbelweb/flibustahub/internal/services/personal"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetHome() (catalog.HomeDashboard, error) {
	return a.svc.GetHome(context.Background())
}

func (a *App) PersonalSnapshot() (personal.Snapshot, error) {
	return a.svc.PersonalSnapshot(context.Background())
}

func (a *App) SetWorkRating(id int64, rating int) error {
	var p *int
	if rating > 0 {
		p = &rating
	}
	return a.svc.SetWorkRating(context.Background(), id, p)
}

func (a *App) SetWorkComment(id int64, comment string) error {
	return a.svc.SetWorkComment(context.Background(), id, comment)
}

func (a *App) SetWorkWantToRead(id int64, want bool) error {
	return a.svc.SetWorkWantToRead(context.Background(), id, want)
}

func (a *App) ExportPersonal(path string) (personal.ExportResult, error) {
	return a.svc.ExportPersonal(context.Background(), path)
}

func (a *App) PreviewPersonalImport(path string) (personal.ImportReport, error) {
	return a.svc.PreviewPersonalImport(context.Background(), path)
}

func (a *App) ImportPersonal(path string) (personal.ImportReport, error) {
	return a.svc.ImportPersonal(context.Background(), path)
}

func (a *App) SelectPersonalExportPath(title string) (string, error) {
	if a.rt == nil || a.rt.ctx == nil {
		return "", apperr.New(apperr.CodeOpenFileFailed, nil)
	}
	path, err := wailsruntime.SaveFileDialog(a.rt.ctx, wailsruntime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: "flibustahub-personal.csv",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "CSV (*.csv)", Pattern: "*.csv"},
			{DisplayName: "JSON (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		return "", apperr.Wrap(apperr.CodeOpenFileFailed, err, nil)
	}
	return path, nil
}

func (a *App) SelectPersonalImportPath(title string) (string, error) {
	if a.rt == nil || a.rt.ctx == nil {
		return "", apperr.New(apperr.CodeOpenFileFailed, nil)
	}
	path, err := wailsruntime.OpenFileDialog(a.rt.ctx, wailsruntime.OpenDialogOptions{
		Title: title,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "CSV / JSON", Pattern: "*.csv;*.json"},
		},
	})
	if err != nil {
		return "", apperr.Wrap(apperr.CodeOpenFileFailed, err, nil)
	}
	return path, nil
}
