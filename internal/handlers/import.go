package handlers

import (
	"context"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/events"
	appsvc "github.com/alexbelweb/flibustahub/internal/services/app"
	"github.com/alexbelweb/flibustahub/internal/services/inpximport"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	CloseKindImport      = "import"
	CloseKindMaintenance = "maintenance"
)

// CloseRequested is the payload of import:closeRequested and maintenance:closeRequested.
type CloseRequested struct {
	Kind      string `json:"kind"`
	Committed bool   `json:"committed"`
}

func (a *App) SelectLibraryRoot(title string) (string, error) {
	if a.rt == nil || a.rt.ctx == nil {
		return "", apperr.New(apperr.CodeOpenDirFailed, nil)
	}
	dir, err := runtime.OpenDirectoryDialog(a.rt.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
	if err != nil {
		return "", apperr.Wrap(apperr.CodeOpenDirFailed, err, nil)
	}
	if dir == "" {
		return "", nil
	}
	if err := a.svc.SetLibraryRoot(dir); err != nil {
		return "", err
	}
	return dir, nil
}

func (a *App) SelectINPXFile(title string) (string, error) {
	if a.rt == nil || a.rt.ctx == nil {
		return "", apperr.New(apperr.CodeOpenFileFailed, nil)
	}
	path, err := runtime.OpenFileDialog(a.rt.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{{
			DisplayName: "INPX",
			Pattern:     "*.inpx",
		}},
	})
	if err != nil {
		return "", apperr.Wrap(apperr.CodeOpenFileFailed, err, nil)
	}
	if path == "" {
		return "", nil
	}
	if err := a.svc.SetINPXPath(path); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) SetINPXPath(path string) error {
	return a.svc.SetINPXPath(path)
}

func (a *App) PreviewImport() (appsvc.ImportPreview, error) {
	return a.svc.PreviewImport(context.Background())
}

func (a *App) StartImport() (inpximport.ReportDTO, error) {
	return a.svc.StartImport(context.Background(), func(p inpximport.Progress) {
		if a.rt == nil || a.rt.ctx == nil {
			return
		}
		runtime.EventsEmit(a.rt.ctx, events.ImportProgress, p)
	})
}

func (a *App) CancelImport() {
	a.svc.CancelImport()
}

func (a *App) LastImportReport() (inpximport.ReportDTO, error) {
	return a.svc.LastImportReport(context.Background())
}

func (a *App) DismissWindowClose() {
	if a.rt != nil {
		a.rt.DismissClose()
	}
}
