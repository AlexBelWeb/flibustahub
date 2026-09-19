package app

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/inpx/testdata"
	"github.com/alexbelweb/flibustahub/internal/services/inpximport"
)

func TestSetLibraryRootPersists(t *testing.T) {
	svc := newTestService(t)
	dir := t.TempDir()
	if err := svc.SetLibraryRoot(dir); err != nil {
		t.Fatal(err)
	}
	if svc.cfg.Live().LibraryRoot != dir {
		t.Fatalf("libraryRoot=%q", svc.cfg.Live().LibraryRoot)
	}
}

func TestPreviewImportAndLastReport(t *testing.T) {
	svc := newTestService(t)
	lib := t.TempDir()
	if err := testdata.WriteLibraryRoot(lib); err != nil {
		t.Fatal(err)
	}
	catalog, err := db.Open(context.Background(), db.Options{
		Path:       filepath.Join(t.TempDir(), "catalog.sqlite"),
		BackupsDir: filepath.Join(t.TempDir(), "backups"),
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = catalog.Close() })
	svc.AttachCatalog(catalog, nil)
	if err := svc.SetLibraryRoot(lib); err != nil {
		t.Fatal(err)
	}

	prev, err := svc.PreviewImport(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if prev.INPXFileName != testdata.DumpName || prev.FileVersion != "20260901" {
		t.Fatalf("%+v", prev)
	}
	if prev.ZipCount != 2 {
		t.Fatalf("zipCount=%d", prev.ZipCount)
	}
	if len(prev.INPXFiles) != 1 || prev.INPXFiles[0].Name != testdata.DumpName {
		t.Fatalf("inpxFiles=%+v", prev.INPXFiles)
	}
	if prev.HasCatalog || prev.SameVersion {
		t.Fatalf("empty catalog flagged as imported: %+v", prev)
	}

	empty, err := svc.LastImportReport(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if empty.Status != "" {
		t.Fatalf("expected empty report, got %+v", empty)
	}

	rep, err := svc.StartImport(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Status != inpximport.StatusDone {
		t.Fatalf("status %s", rep.Status)
	}
	if len(rep.Notes.MissingArchives) != 1 {
		t.Fatalf("notes %+v", rep.Notes)
	}

	prev2, err := svc.PreviewImport(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !prev2.HasCatalog || !prev2.SameVersion {
		t.Fatalf("after import: %+v", prev2)
	}

	last, err := svc.LastImportReport(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if last.ID != rep.ID || last.Status != inpximport.StatusDone {
		t.Fatalf("last %+v", last)
	}
}

func TestImportClearsNoneMarkers(t *testing.T) {
	svc := newTestService(t)
	lib := t.TempDir()
	if err := testdata.WriteLibraryRoot(lib); err != nil {
		t.Fatal(err)
	}
	catalog, err := db.Open(context.Background(), db.Options{
		Path:       filepath.Join(t.TempDir(), "catalog.sqlite"),
		BackupsDir: filepath.Join(t.TempDir(), "backups"),
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = catalog.Close() })
	svc.AttachCatalog(catalog, nil)
	if err := svc.SetLibraryRoot(lib); err != nil {
		t.Fatal(err)
	}

	dir := svc.cfg.Paths().CoversDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	jpg := filepath.Join(dir, "7.jpg")
	none := filepath.Join(dir, "7.none")
	if err := os.WriteFile(jpg, []byte{0xff, 0xd8, 0xff, 0xe0}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(none, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.StartImport(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(none); !os.IsNotExist(err) {
		t.Fatal(".none must be removed after a successful import")
	}
	if _, err := os.Stat(jpg); err != nil {
		t.Fatal("cached image must stay")
	}
}

func TestPreviewImportEmptyFolder(t *testing.T) {
	svc := newTestService(t)
	dir := t.TempDir()
	if err := svc.SetLibraryRoot(dir); err != nil {
		t.Fatal(err)
	}
	prev, err := svc.PreviewImport(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if prev.ZipCount != 0 || len(prev.INPXFiles) != 0 || prev.INPXPath != "" {
		t.Fatalf("%+v", prev)
	}
}

func TestPreviewImportUnreadableFolder(t *testing.T) {
	svc := newTestService(t)
	missing := filepath.Join(t.TempDir(), "gone")
	if err := svc.cfg.Update(func(f *config.File) { f.LibraryRoot = missing }); err != nil {
		t.Fatal(err)
	}
	_, err := svc.PreviewImport(context.Background())
	if apperr.As(err).Code != apperr.CodeLibraryUnreadable {
		t.Fatalf("got %v", err)
	}
}

func TestSetLibraryRootRejectsFile(t *testing.T) {
	svc := newTestService(t)
	f := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetLibraryRoot(f); err == nil {
		t.Fatal("expected error")
	}
}
