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
	if v := os.Getenv("FLIBUSTAHUB_DATADIR"); v != "" {
		if err := os.MkdirAll(v, 0o755); err != nil {
			t.Fatal(err)
		}
		dir = v
	}
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

	t.Logf("selected inpx=%s version=%s", rep.INPXPath, rep.INPXVersion)
	if inpxPath == "" {
		if strings.Contains(strings.ToLower(filepath.ToSlash(rep.INPXPath)), "/update/") {
			t.Fatalf("autosearch picked a nested dump: %s", rep.INPXPath)
		}
		if filepath.Dir(rep.INPXPath) != filepath.Clean(root) {
			t.Fatalf("autosearch must stay in library root, got %s", rep.INPXPath)
		}
	}

	var deleted, editions int
	if err := d.Read.QueryRow(`SELECT count(*) FROM editions`).Scan(&editions); err != nil {
		t.Fatal(err)
	}
	if err := d.Read.QueryRow(`SELECT count(*) FROM editions WHERE is_deleted=1`).Scan(&deleted); err != nil {
		t.Fatal(err)
	}
	t.Logf("import status=%s records=%d works_added=%d editions=%d editions_added=%d editions_updated=%d deactivated=%d collisions=%d unnamed=%d missing_archives=%d deleted=%d",
		rep.Status, rep.RecordsSeen, rep.WorksAdded, editions, rep.EditionsAdded, rep.EditionsUpdated, rep.EditionsDeactivated, rep.LibIDCollisions, rep.Notes.UnnamedGenresTotal, rep.Notes.MissingArchivesTotal, deleted)
	t.Logf("unnamed_genres=%v skipped_malformed=%d skipped_no_libid=%d", rep.Notes.UnnamedGenres, rep.Notes.SkippedMalformed, rep.Notes.SkippedNoLibID)
	t.Logf("phases_ms backup=%d reading=%d records=%d fts=%d warmup=%d analyze=%d wall=%s db_bytes=%d",
		rep.Notes.PhasesMS["backup"], rep.Notes.PhasesMS["reading"], rep.Notes.PhasesMS["records"],
		rep.Notes.PhasesMS["fts"], rep.Notes.PhasesMS["warmup"], rep.Notes.PhasesMS["analyze"], elapsed, st.Size())
	if editions > 0 {
		t.Logf("deleted_share=%.1f%%", 100*float64(deleted)/float64(editions))
	}

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
