package handlers

import (
	"context"
	"runtime"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/services/downloads"
	"github.com/alexbelweb/flibustahub/internal/services/storage"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) CheckStorage(force bool) storage.Snapshot {
	return a.svc.CheckStorage(context.Background(), force)
}

func (a *App) DismissDumpOffer() error {
	return a.svc.DismissDumpOffer()
}

func (a *App) DownloadEdition(id int64) (downloads.Result, error) {
	return a.svc.DownloadEdition(context.Background(), id)
}

func (a *App) ReadEdition(id int64) (downloads.Result, error) {
	return a.svc.ReadEdition(context.Background(), id)
}

func (a *App) CancelFileOp(id int64, kind string) {
	a.svc.CancelFileOp(id, kind)
}

func (a *App) ShowInFolder(path string) error {
	return a.svc.ShowInFolder(path)
}

func (a *App) SelectDownloadsDir(title string) (string, error) {
	if a.rt == nil || a.rt.ctx == nil {
		return "", apperr.New(apperr.CodeOpenDirFailed, nil)
	}
	dir, err := wailsruntime.OpenDirectoryDialog(a.rt.ctx, wailsruntime.OpenDialogOptions{
		Title: title,
	})
	if err != nil {
		return "", apperr.Wrap(apperr.CodeOpenDirFailed, err, nil)
	}
	if dir == "" {
		return "", nil
	}
	if err := a.svc.SetDownloadsDir(dir); err != nil {
		return "", err
	}
	return dir, nil
}

func (a *App) SelectReaderPath(title string) (string, error) {
	if a.rt == nil || a.rt.ctx == nil {
		return "", apperr.New(apperr.CodeOpenFileFailed, nil)
	}
	filter := wailsruntime.FileFilter{DisplayName: "Application", Pattern: "*"}
	if runtime.GOOS == "windows" {
		filter = wailsruntime.FileFilter{DisplayName: "Application", Pattern: "*.exe"}
	}
	path, err := wailsruntime.OpenFileDialog(a.rt.ctx, wailsruntime.OpenDialogOptions{
		Title:   title,
		Filters: []wailsruntime.FileFilter{filter},
	})
	if err != nil {
		return "", apperr.Wrap(apperr.CodeOpenFileFailed, err, nil)
	}
	if path == "" {
		return "", nil
	}
	if err := a.svc.SetReaderPath(path); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) ClearReaderPath() error {
	return a.svc.SetReaderPath("")
}

func (a *App) ReaderPath() string {
	return a.svc.ReaderPath()
}
