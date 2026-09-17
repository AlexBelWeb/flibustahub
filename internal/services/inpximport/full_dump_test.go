package inpximport

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/db"
)

func TestImportFullDump(t *testing.T) {
	root := os.Getenv("FLIBUSTAHUB_LIBRARYROOT")
	inpxPath := os.Getenv("FLIBUSTAHUB_INPXPATH")
	if root == "" && inpxPath == "" {
		t.Skip("FLIBUSTAHUB_LIBRARYROOT / FLIBUSTAHUB_INPXPATH not set")
	}
	dir := t.TempDir()
	d, err := db.Open(context.Background(), db.Options{
		Path:       filepath.Join(dir, "catalog.sqlite"),
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	t0 := time.Now()
	svc := New(d, slog.Default())
	rep, err := svc.Import(context.Background(), Options{
		LibraryRoot: root,
		INPXPath:    inpxPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(t0)
	st, err := os.Stat(d.Path())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("import status=%s records=%d works_added=%d editions_added=%d collisions=%d unnamed=%d missing_archives=%d",
		rep.Status, rep.RecordsSeen, rep.WorksAdded, rep.EditionsAdded, rep.LibIDCollisions, rep.Notes.UnnamedGenresTotal, rep.Notes.MissingArchivesTotal)
	t.Logf("phases_ms=%v wall=%s db_bytes=%d", rep.Notes.PhasesMS, elapsed, st.Size())

	filled := filepath.Join(filepath.Dir(d.Path()), "filled-catalog.sqlite")
	slash := filepath.ToSlash(filled)
	if _, err := d.Write.ExecContext(context.Background(), "VACUUM INTO ?", slash); err != nil {
		quoted := strings.ReplaceAll(slash, "'", "''")
		if _, err2 := d.Write.ExecContext(context.Background(), "VACUUM INTO '"+quoted+"'"); err2 != nil {
			t.Fatalf("snapshot: %v / %v", err, err2)
		}
	}
	t.Logf("filled snapshot %s", filled)

	if elapsed > 10*time.Minute {
		t.Errorf("import exceeded 10m budget: %s (driver decision is the owner's)", elapsed)
	}
}
