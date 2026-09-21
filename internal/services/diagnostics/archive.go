package diagnostics

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
)

// ArchiveResult is the written zip path.
type ArchiveResult struct {
	Path string `json:"path"`
}

type versionsFile struct {
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	BuildDate   string `json:"buildDate"`
	INPXVersion string `json:"inpxVersion"`
}

type dbStatsFile struct {
	Works         int    `json:"works"`
	Authors       int    `json:"authors"`
	ImportedAt    string `json:"importedAt"`
	UnnamedGenres int    `json:"unnamedGenres"`
}

// WriteArchive saves a zip into dir. Personal ratings, comments and secrets stay out.
func (s *Service) WriteArchive(ctx context.Context, dir string) (ArchiveResult, error) {
	dir = filepath.Clean(dir)
	if dir == "" || dir == "." {
		return ArchiveResult{}, apperr.New(apperr.CodeDiagArchiveFailed, nil)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return ArchiveResult{}, apperr.Wrap(apperr.CodeDiagArchiveFailed, err, nil)
	}
	snap, err := s.Snapshot(ctx)
	if err != nil {
		return ArchiveResult{}, err
	}
	name := "flibustahub-diagnostics-" + time.Now().UTC().Format("20060102-150405") + ".zip"
	dest := filepath.Join(dir, name)
	tmp := dest + ".tmp"
	if err := s.writeArchiveFile(tmp, snap); err != nil {
		_ = os.Remove(tmp)
		return ArchiveResult{}, err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return ArchiveResult{}, apperr.Wrap(apperr.CodeDiagArchiveFailed, err, nil)
	}
	return ArchiveResult{Path: dest}, nil
}

func (s *Service) writeArchiveFile(path string, snap Snapshot) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return apperr.Wrap(apperr.CodeDiagArchiveFailed, err, nil)
	}
	zw := zip.NewWriter(f)
	closeAll := func() error {
		zerr := zw.Close()
		ferr := f.Close()
		if zerr != nil {
			return apperr.Wrap(apperr.CodeDiagArchiveFailed, zerr, nil)
		}
		if ferr != nil {
			return apperr.Wrap(apperr.CodeDiagArchiveFailed, ferr, nil)
		}
		return nil
	}

	if err := addJSON(zw, "versions.json", versionsFile{
		Version:     snap.Version,
		Commit:      snap.Commit,
		BuildDate:   snap.BuildDate,
		INPXVersion: snap.INPXVersion,
	}); err != nil {
		_ = zw.Close()
		_ = f.Close()
		return err
	}
	if err := addJSON(zw, "db-stats.json", dbStatsFile{
		Works:         snap.Works,
		Authors:       snap.Authors,
		ImportedAt:    snap.ImportedAt,
		UnnamedGenres: snap.UnnamedGenres,
	}); err != nil {
		_ = zw.Close()
		_ = f.Close()
		return err
	}
	if err := addConfig(zw, s.config()); err != nil {
		_ = zw.Close()
		_ = f.Close()
		return err
	}
	if err := addLogs(zw, snap.Paths.LogsDir); err != nil {
		_ = zw.Close()
		_ = f.Close()
		return err
	}
	return closeAll()
}

func addJSON(zw *zip.Writer, name string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.CodeDiagArchiveFailed, err, nil)
	}
	return addBytes(zw, name, raw)
}

func addBytes(zw *zip.Writer, name string, raw []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return apperr.Wrap(apperr.CodeDiagArchiveFailed, err, nil)
	}
	if _, err := w.Write(raw); err != nil {
		return apperr.Wrap(apperr.CodeDiagArchiveFailed, err, nil)
	}
	return nil
}

func addConfig(zw *zip.Writer, store *config.Store) error {
	if store == nil {
		return addBytes(zw, "config.json", []byte("{}\n"))
	}
	raw, err := os.ReadFile(store.Path())
	if err != nil {
		if os.IsNotExist(err) {
			live, mErr := json.MarshalIndent(store.Live(), "", "  ")
			if mErr != nil {
				return apperr.Wrap(apperr.CodeDiagArchiveFailed, mErr, nil)
			}
			return addBytes(zw, "config.json", live)
		}
		return apperr.Wrap(apperr.CodeDiagArchiveFailed, err, nil)
	}
	return addBytes(zw, "config.json", raw)
}

func addLogs(zw *zip.Writer, logsDir string) error {
	if logsDir == "" {
		return nil
	}
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return apperr.Wrap(apperr.CodeDiagArchiveFailed, err, nil)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isLogFile(name) {
			continue
		}
		src := filepath.Join(logsDir, name)
		if err := addFile(zw, "logs/"+name, src); err != nil {
			return err
		}
	}
	return nil
}

func isLogFile(name string) bool {
	return name == "app.log" || (len(name) > 7 && name[:7] == "app.log")
}

func addFile(zw *zip.Writer, name, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return apperr.Wrap(apperr.CodeDiagArchiveFailed, err, nil)
	}
	defer func() { _ = in.Close() }()
	w, err := zw.Create(name)
	if err != nil {
		return apperr.Wrap(apperr.CodeDiagArchiveFailed, err, nil)
	}
	if _, err := io.Copy(w, in); err != nil {
		return apperr.Wrap(apperr.CodeDiagArchiveFailed, err, nil)
	}
	return nil
}

func zipContains(path, needle string) (bool, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = zr.Close() }()
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return false, err
		}
		raw, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return false, err
		}
		if bytes.Contains(raw, []byte(needle)) {
			return true, nil
		}
	}
	return false, nil
}
