package handlers

import (
	"context"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/services/diagnostics"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) Diagnostics() (diagnostics.Snapshot, error) {
	return a.svc.Diagnostics(context.Background())
}

func (a *App) SaveDiagnosticArchive(title string) (diagnostics.ArchiveResult, error) {
	if a.rt == nil || a.rt.ctx == nil {
		return diagnostics.ArchiveResult{}, apperr.New(apperr.CodeOpenDirFailed, nil)
	}
	dir, err := runtime.OpenDirectoryDialog(a.rt.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
	if err != nil {
		return diagnostics.ArchiveResult{}, apperr.Wrap(apperr.CodeOpenDirFailed, err, nil)
	}
	if dir == "" {
		return diagnostics.ArchiveResult{}, nil
	}
	return a.svc.SaveDiagnosticArchive(context.Background(), dir)
}

func (a *App) PreviewIssue(kind, description string) (diagnostics.IssueReport, error) {
	return a.svc.PreviewIssue(context.Background(), kind, description)
}
