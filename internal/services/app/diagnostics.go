package app

import (
	"context"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/services/diagnostics"
)

func (s *Service) Diagnostics(ctx context.Context) (diagnostics.Snapshot, error) {
	if s == nil || s.diag == nil {
		return diagnostics.Snapshot{}, apperr.New(apperr.CodeDiagFailed, nil)
	}
	return s.diag.Snapshot(ctx)
}

func (s *Service) SaveDiagnosticArchive(ctx context.Context, dir string) (diagnostics.ArchiveResult, error) {
	if s == nil || s.diag == nil {
		return diagnostics.ArchiveResult{}, apperr.New(apperr.CodeDiagArchiveFailed, nil)
	}
	return s.diag.WriteArchive(ctx, dir)
}

func (s *Service) PreviewIssue(ctx context.Context, kind, description string) (diagnostics.IssueReport, error) {
	if s == nil || s.diag == nil {
		return diagnostics.IssueReport{}, apperr.New(apperr.CodeDiagFailed, nil)
	}
	snap, err := s.diag.Snapshot(ctx)
	if err != nil {
		return diagnostics.IssueReport{}, err
	}
	return s.diag.BuildReport(kind, description, snap), nil
}
