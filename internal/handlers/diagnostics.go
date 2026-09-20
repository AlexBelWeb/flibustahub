package handlers

import (
	"context"
	"os"

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

func (a *App) BuildIssueReport(kind, description string) (diagnostics.IssueReport, error) {
	return a.svc.BuildIssueReport(context.Background(), kind, description)
}

func (a *App) SaveIssueReport(title, body string) (diagnostics.ArchiveResult, error) {
	if a.rt == nil || a.rt.ctx == nil {
		return diagnostics.ArchiveResult{}, apperr.New(apperr.CodeOpenFileFailed, nil)
	}
	path, err := runtime.SaveFileDialog(a.rt.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: "flibustahub-issue.md",
		Filters: []runtime.FileFilter{
			{DisplayName: "Markdown (*.md)", Pattern: "*.md"},
		},
	})
	if err != nil {
		return diagnostics.ArchiveResult{}, apperr.Wrap(apperr.CodeOpenFileFailed, err, nil)
	}
	if path == "" {
		return diagnostics.ArchiveResult{}, nil
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return diagnostics.ArchiveResult{}, apperr.Wrap(apperr.CodeOpenFileFailed, err, nil)
	}
	return diagnostics.ArchiveResult{Path: path}, nil
}
