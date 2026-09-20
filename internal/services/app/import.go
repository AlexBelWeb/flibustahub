package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/inpx"
	"github.com/alexbelweb/flibustahub/internal/repositories"
	"github.com/alexbelweb/flibustahub/internal/services/covers"
	"github.com/alexbelweb/flibustahub/internal/services/inpximport"
)

// INPXFile is one dump file found in the library root.
type INPXFile struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

// ImportPreview is the dump chosen for import, shown before StartImport.
type ImportPreview struct {
	LibraryRoot    string     `json:"libraryRoot"`
	ZipCount       int        `json:"zipCount"`
	INPXFiles      []INPXFile `json:"inpxFiles"`
	INPXPath       string     `json:"inpxPath"`
	INPXFileName   string     `json:"inpxFileName"`
	FileVersion    string     `json:"fileVersion"`
	CatalogVersion string     `json:"catalogVersion"`
	SameVersion    bool       `json:"sameVersion"`
	HasCatalog     bool       `json:"hasCatalog"`
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
	if err := s.config().Update(func(f *config.File) { f.LibraryRoot = path }); err != nil {
		return err
	}
	if s.storage != nil {
		s.storage.PersistVolume(context.Background(), path)
		go s.CheckStorage(context.Background(), true)
	}
	return nil
}

func (s *Service) SetINPXPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return s.config().Update(func(f *config.File) { f.INPXPath = "" })
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return apperr.Wrap(apperr.CodeINPXNotFound, err, nil)
	}
	if info.IsDir() || !strings.EqualFold(filepath.Ext(path), ".inpx") {
		return apperr.New(apperr.CodeINPXNotFound, nil)
	}
	return s.config().Update(func(f *config.File) { f.INPXPath = path })
}

func (s *Service) PreviewImport(ctx context.Context) (ImportPreview, error) {
	live := s.config().Live()
	empty := ImportPreview{LibraryRoot: live.LibraryRoot, INPXFiles: []INPXFile{}}
	if strings.TrimSpace(live.LibraryRoot) == "" {
		return empty, nil
	}
	zipCount, dumps, err := inpx.InspectLibraryRoot(live.LibraryRoot)
	if err != nil {
		return empty, apperr.Wrap(apperr.CodeLibraryUnreadable, err, nil)
	}
	files := make([]INPXFile, 0, len(dumps))
	for _, d := range dumps {
		files = append(files, INPXFile{Path: d.Path, Name: d.Name})
	}
	out := ImportPreview{
		LibraryRoot: live.LibraryRoot,
		ZipCount:    zipCount,
		INPXFiles:   files,
	}
	selected := strings.TrimSpace(live.INPXPath)
	if selected == "" {
		if newest, findErr := inpx.FindINPX(live.LibraryRoot, ""); findErr == nil {
			selected = newest
		}
	}
	if selected != "" {
		out.INPXPath = selected
		out.INPXFileName = filepath.Base(selected)
		st, statErr := os.Stat(selected)
		if statErr == nil && !st.IsDir() {
			f, openErr := os.Open(selected)
			if openErr == nil {
				meta, peekErr := inpx.PeekMeta(f, st.Size())
				_ = f.Close()
				if peekErr == nil {
					out.FileVersion = meta.Version
				}
			}
		}
	}
	snap := s.snap()
	if snap.catalog != nil {
		catalogVer, verErr := db.INPXVersion(ctx, snap.catalog.Read)
		if verErr != nil {
			return ImportPreview{}, apperr.Wrap(apperr.CodeImportFailed, verErr, nil)
		}
		out.CatalogVersion = catalogVer
		out.HasCatalog = catalogVer != ""
		out.SameVersion = catalogVer != "" && catalogVer == out.FileVersion
	}
	return out, nil
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
	if s.DatabaseMaintenanceRunning() {
		s.importMu.Unlock()
		return inpximport.ReportDTO{}, apperr.New(apperr.CodeDBMaintenanceBusy, nil)
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
	coversDir := s.config().Paths().CoversDir
	if err := covers.ClearNoneMarkers(coversDir); err != nil {
		s.Logger().Warn("cover none markers not cleared", "err", err)
	}
	s.openMu.Lock()
	c := s.covers
	s.openMu.Unlock()
	if c != nil {
		c.ForgetSessionFails()
	}
	go s.CheckStorage(context.Background(), true)
	return rep.ToDTO(), nil
}

func (s *Service) CancelImport() {
	s.importMu.Lock()
	cancel := s.importCancel
	s.importMu.Unlock()
	if cancel != nil {
		s.Logger().Info("import cancelled")
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
		row.ID, row.Status, row.INPXPath, row.INPXVersion, row.FinishedAt,
		row.RecordsSeen, row.WorksAdded, row.EditionsAdded, row.EditionsUpdated,
		row.EditionsDeactivated, row.LibIDCollisions,
		inpximport.ParseNotes(row.NotesJSON),
	)
	return rep.ToDTO(), nil
}
