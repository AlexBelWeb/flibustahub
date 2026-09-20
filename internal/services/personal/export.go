package personal

import (
	"context"
	"strings"

	"github.com/alexbelweb/flibustahub/internal/apperr"
)

type ExportResult struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

func (s *Service) Export(ctx context.Context, path string) (ExportResult, error) {
	if err := s.ready(); err != nil {
		return ExportResult{}, err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return ExportResult{}, apperr.New(apperr.CodePersonalExportFailed, nil)
	}
	rows, err := s.cat.ExportableWorks(ctx)
	if err != nil {
		return ExportResult{}, apperr.Wrap(apperr.CodePersonalExportFailed, err, nil)
	}
	dump := make([]dumpRow, 0, len(rows))
	for _, r := range rows {
		dump = append(dump, rowFromPersonal(r))
	}
	var data []byte
	if dumpFormatOf(path) == "json" {
		data, err = encodeJSON(dump)
	} else {
		data, err = encodeCSV(dump)
	}
	if err != nil {
		return ExportResult{}, apperr.Wrap(apperr.CodePersonalExportFailed, err, nil)
	}
	if err := writeAtomicFile(path, data); err != nil {
		return ExportResult{}, apperr.Wrap(apperr.CodePersonalExportFailed, err, nil)
	}
	if err := s.cat.MarkExported(ctx, rows, s.stamp()); err != nil {
		return ExportResult{}, apperr.Wrap(apperr.CodePersonalExportFailed, err, nil)
	}
	return ExportResult{Path: path, Count: len(rows)}, nil
}
