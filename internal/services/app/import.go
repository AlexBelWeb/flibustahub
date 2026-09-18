package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/inpx"
	"github.com/alexbelweb/flibustahub/internal/repositories"
	"github.com/alexbelweb/flibustahub/internal/services/inpximport"
)

// ImportPreview is the dump chosen for import, shown before StartImport.
type ImportPreview struct {
	LibraryRoot    string `json:"libraryRoot"`
	INPXPath       string `json:"inpxPath"`
	INPXFileName   string `json:"inpxFileName"`
	FileVersion    string `json:"fileVersion"`
	CatalogVersion string `json:"catalogVersion"`
	SameVersion    bool   `json:"sameVersion"`
	HasCatalog     bool   `json:"hasCatalog"`
}

func (s *Service) SetLibraryRoot(path string) error {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return apperr.New(apperr.CodeINPXNotFound, nil)
	}
	info, err := os.Stat(path)
	if err != nil {
		return apperr.Wrap(apperr.CodeINPXNotFound, err, nil)
	}
	if !info.IsDir() {
		return apperr.New(apperr.CodeINPXNotFound, nil)
	}
	return s.config().Update(func(f *config.File) { f.LibraryRoot = path })
}

func (s *Service) PreviewImport(ctx context.Context) (ImportPreview, error) {
	live := s.config().Live()
	path, err := inpx.FindINPX(live.LibraryRoot, live.INPXPath)
	if err != nil {
		return ImportPreview{LibraryRoot: live.LibraryRoot}, apperr.Wrap(apperr.CodeINPXNotFound, err, nil)
	}
	st, err := os.Stat(path)
	if err != nil {
		return ImportPreview{LibraryRoot: live.LibraryRoot}, apperr.Wrap(apperr.CodeINPXNotFound, err, nil)
	}
	f, err := os.Open(path)
	if err != nil {
		return ImportPreview{LibraryRoot: live.LibraryRoot}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	meta, err := inpx.PeekMeta(f, st.Size())
	_ = f.Close()
	if err != nil {
		return ImportPreview{LibraryRoot: live.LibraryRoot}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	catalogVer := ""
	snap := s.snap()
	if snap.catalog != nil {
		catalogVer, err = db.INPXVersion(ctx, snap.catalog.Read)
		if err != nil {
			return ImportPreview{}, apperr.Wrap(apperr.CodeImportFailed, err, nil)
		}
	}
	return ImportPreview{
		LibraryRoot:    live.LibraryRoot,
		INPXPath:       path,
		INPXFileName:   filepath.Base(path),
		FileVersion:    meta.Version,
		CatalogVersion: catalogVer,
		SameVersion:    catalogVer != "" && catalogVer == meta.Version,
		HasCatalog:     catalogVer != "",
	}, nil
}

func (s *Service) StartImport(ctx context.Context, progress func(inpximport.Progress)) (inpximport.ReportDTO, error) {
	st := s.snap()
	if st.importer == nil || st.catalog == nil {
		return inpximport.ReportDTO{}, apperr.New(apperr.CodeImportFailed, nil)
	}
	s.importMu.Lock()
	if s.importing {
		s.importMu.Unlock()
		return inpximport.ReportDTO{}, apperr.New(apperr.CodeImportFailed, nil)
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.importing = true
	s.importCancel = cancel
	s.lastProgress = inpximport.Progress{}
	s.importMu.Unlock()
	defer func() {
		s.importMu.Lock()
		s.importing = false
		s.importCancel = nil
		s.importMu.Unlock()
		cancel()
	}()

	live := s.config().Live()
	rep, err := st.importer.Import(runCtx, inpximport.Options{
		LibraryRoot: live.LibraryRoot,
		INPXPath:    live.INPXPath,
		Progress: func(p inpximport.Progress) {
			s.importMu.Lock()
			s.lastProgress = p
			s.importMu.Unlock()
			if progress != nil {
				progress(p)
			}
		},
	})
	if err != nil {
		return inpximport.ReportDTO{}, err
	}
	return rep.ToDTO(), nil
}

func (s *Service) CancelImport() {
	s.importMu.Lock()
	cancel := s.importCancel
	s.importMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Service) IsImporting() bool {
	s.importMu.Lock()
	defer s.importMu.Unlock()
	return s.importing
}

func (s *Service) ImportCommitted() bool {
	s.importMu.Lock()
	defer s.importMu.Unlock()
	return s.lastProgress.Committed
}

func (s *Service) LastImportReport(ctx context.Context) (inpximport.ReportDTO, error) {
	empty := inpximport.ReportDTO{}
	st := s.snap()
	if st.catalog == nil {
		return empty, nil
	}
	row, err := repositories.LatestFinishedBatch(ctx, st.catalog.Read)
	if err != nil {
		return empty, apperr.Wrap(apperr.CodeImportFailed, err, nil)
	}
	if row == nil {
		return empty, nil
	}
	rep := inpximport.ReportFromBatch(
		row.ID, row.Status, row.INPXPath, row.INPXVersion,
		row.RecordsSeen, row.WorksAdded, row.EditionsAdded, row.EditionsUpdated,
		row.EditionsDeactivated, row.LibIDCollisions,
		inpximport.ParseNotes(row.NotesJSON),
	)
	return rep.ToDTO(), nil
}
