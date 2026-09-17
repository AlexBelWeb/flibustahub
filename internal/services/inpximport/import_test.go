package inpximport

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/inpx/testdata"
	"github.com/alexbelweb/flibustahub/internal/textnorm"
)

func openCatalog(t *testing.T) *db.DB {
	t.Helper()
	dir := t.TempDir()
	var tick atomic.Int64
	base := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	d, err := db.Open(context.Background(), db.Options{
		Path:       filepath.Join(dir, "catalog.sqlite"),
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
		Now: func() time.Time {
			return base.Add(time.Duration(tick.Add(1)) * time.Second)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func importFixture(t *testing.T, d *db.DB, probe func(*sql.Tx) error) Report {
	t.Helper()
	lib := t.TempDir()
	if err := testdata.WriteLibraryRoot(lib); err != nil {
		t.Fatal(err)
	}
	svc := New(d, slog.New(slog.DiscardHandler))
	rep, err := svc.Import(context.Background(), Options{
		LibraryRoot:  lib,
		RecordsProbe: probe,
	})
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func TestImportCatalogFixture(t *testing.T) {
	d := openCatalog(t)
	var ftsDuringRecords int
	rep := importFixture(t, d, func(tx *sql.Tx) error {
		return tx.QueryRow(`SELECT count(*) FROM works_fts`).Scan(&ftsDuringRecords)
	})
	if ftsDuringRecords != 0 {
		t.Fatalf("works_fts during records = %d", ftsDuringRecords)
	}
	if rep.Status != StatusDone {
		t.Fatalf("status %s", rep.Status)
	}
	if rep.INPXVersion != "20260901" {
		t.Fatalf("version %s", rep.INPXVersion)
	}
	if rep.Notes.MissingArchivesTotal != 1 || len(rep.Notes.MissingArchives) != 1 {
		t.Fatalf("missing archives %+v", rep.Notes)
	}
	if rep.Notes.SkippedMalformed != 1 || rep.Notes.SkippedNoLibID != 1 {
		t.Fatalf("skips %+v", rep.Notes)
	}
	if rep.Notes.UnnamedGenresTotal < 1 {
		t.Fatalf("unnamed genres %+v", rep.Notes)
	}
	if rep.LibIDCollisions < 1 {
		t.Fatalf("collisions %d", rep.LibIDCollisions)
	}

	var works, editions, fts int
	if err := d.Read.QueryRow(`SELECT count(*) FROM works`).Scan(&works); err != nil {
		t.Fatal(err)
	}
	if err := d.Read.QueryRow(`SELECT count(*) FROM editions`).Scan(&editions); err != nil {
		t.Fatal(err)
	}
	if err := d.Read.QueryRow(`SELECT count(*) FROM works_fts`).Scan(&fts); err != nil {
		t.Fatal(err)
	}
	if works == 0 || editions == 0 {
		t.Fatalf("works=%d editions=%d", works, editions)
	}
	if fts != works {
		t.Fatalf("fts %d works %d", fts, works)
	}

	var yo int
	if err := d.Read.QueryRow(`SELECT count(*) FROM works_fts WHERE works_fts MATCH ?`, textnorm.Normalize("ёлка")).Scan(&yo); err != nil {
		t.Fatal(err)
	}
	if yo < 1 {
		t.Fatal("rebuildFts must index normalize(title) so ёлка matches елка")
	}

	var serno string
	if err := d.Read.QueryRow(`SELECT series_no FROM editions WHERE libid='700001'`).Scan(&serno); err != nil {
		t.Fatal(err)
	}
	if serno != "1-2" {
		t.Fatalf("serno %q", serno)
	}

	var active, deleted int
	if err := d.Read.QueryRow(`SELECT is_active, is_deleted FROM editions WHERE libid='800001'`).Scan(&active, &deleted); err != nil {
		t.Fatal(err)
	}
	if active != 1 {
		t.Fatal("missing archive edition must still be imported")
	}

	var title string
	var isDel int
	if err := d.Read.QueryRow(`SELECT w.title, e.is_deleted FROM editions e JOIN works w ON w.id=e.work_id WHERE e.libid='910001'`).Scan(&title, &isDel); err != nil {
		t.Fatal(err)
	}
	if isDel != 0 {
		t.Fatalf("DEL 1→0 should replace, is_deleted=%d title=%s", isDel, title)
	}
}

func TestImportIdempotentAndKeepsPersonalData(t *testing.T) {
	d := openCatalog(t)
	importFixture(t, d, nil)

	var workID int64
	if err := d.Write.QueryRow(`SELECT id FROM works LIMIT 1`).Scan(&workID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write.Exec(`UPDATE works SET rating=8, rating_updated_at='t', comment='note', comment_updated_at='t' WHERE id=?`, workID); err != nil {
		t.Fatal(err)
	}

	var works1, editions1 int
	_ = d.Read.QueryRow(`SELECT count(*) FROM works`).Scan(&works1)
	_ = d.Read.QueryRow(`SELECT count(*) FROM editions`).Scan(&editions1)

	rep2 := importFixture(t, d, nil)
	if rep2.WorksAdded != 0 {
		t.Fatalf("second import works_added=%d", rep2.WorksAdded)
	}
	var works2, editions2 int
	_ = d.Read.QueryRow(`SELECT count(*) FROM works`).Scan(&works2)
	_ = d.Read.QueryRow(`SELECT count(*) FROM editions`).Scan(&editions2)
	if works1 != works2 || editions1 != editions2 {
		t.Fatalf("counts changed %d/%d -> %d/%d", works1, editions1, works2, editions2)
	}

	var rating int
	var comment string
	if err := d.Read.QueryRow(`SELECT rating, comment FROM works WHERE id=?`, workID).Scan(&rating, &comment); err != nil {
		t.Fatal(err)
	}
	if rating != 8 || comment != "note" {
		t.Fatalf("personal data lost: %d %q", rating, comment)
	}

	rep3 := importFixture(t, d, nil)
	_ = rep3
	if err := d.Read.QueryRow(`SELECT rating, comment FROM works WHERE id=?`, workID).Scan(&rating, &comment); err != nil {
		t.Fatal(err)
	}
	if rating != 8 || comment != "note" {
		t.Fatalf("personal data lost after third import")
	}
}

func TestImportPragmaIsolation(t *testing.T) {
	d := openCatalog(t)
	lib := t.TempDir()
	if err := testdata.WriteLibraryRoot(lib); err != nil {
		t.Fatal(err)
	}
	svc := New(d, slog.New(slog.DiscardHandler))
	_, err := svc.Import(context.Background(), Options{
		LibraryRoot: lib,
		RecordsProbe: func(*sql.Tx) error {
			fk, err := db.ReadPragma(context.Background(), d.Read, "foreign_keys")
			if err != nil {
				return err
			}
			if fk != "1" {
				return errors.New("read pool foreign_keys leaked: " + fk)
			}
			cs, err := db.ReadPragma(context.Background(), d.Read, "cache_size")
			if err != nil {
				return err
			}
			if cs != "-64000" {
				return errors.New("read pool cache_size leaked: " + cs)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fk, err := db.ReadPragma(context.Background(), d.Write, "foreign_keys")
	if err != nil {
		t.Fatal(err)
	}
	if fk != "1" {
		t.Fatalf("write pool after import foreign_keys=%s", fk)
	}
	cs, err := db.ReadPragma(context.Background(), d.Write, "cache_size")
	if err != nil {
		t.Fatal(err)
	}
	if cs != "-64000" {
		t.Fatalf("write pool after import cache_size=%s", cs)
	}
}

func TestImportCancelRollsBack(t *testing.T) {
	d := openCatalog(t)
	lib := t.TempDir()
	if err := testdata.WriteLibraryRoot(lib); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	svc := New(d, slog.New(slog.DiscardHandler))
	_, err := svc.Import(ctx, Options{
		LibraryRoot: lib,
		RecordsProbe: func(*sql.Tx) error {
			cancel()
			return nil
		},
	})
	if err == nil {
		t.Fatal("expected cancel")
	}
	if apperr.As(err).Code != apperr.CodeImportCancelled {
		t.Fatalf("code %s", apperr.As(err).Code)
	}
	var n int
	_ = d.Read.QueryRow(`SELECT count(*) FROM works`).Scan(&n)
	if n != 0 {
		t.Fatalf("rolled back works=%d", n)
	}
	var status string
	if err := d.Read.QueryRow(`SELECT status FROM import_batches ORDER BY id DESC LIMIT 1`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != StatusCancelled {
		t.Fatalf("status %s", status)
	}
}

func TestFTSDirtyRecovery(t *testing.T) {
	d := openCatalog(t)
	importFixture(t, d, nil)
	if _, err := d.Write.Exec(`DROP TRIGGER IF EXISTS works_fts_au`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write.Exec(`DROP TRIGGER IF EXISTS works_fts_ad`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write.Exec(`DELETE FROM works_fts`); err != nil {
		t.Fatal(err)
	}
	if err := db.SetFTSDirty(context.Background(), d.Write, true); err != nil {
		t.Fatal(err)
	}
	path := d.Path()
	backups := filepath.Join(filepath.Dir(path), "backups")
	_ = d.Close()

	d2, err := db.Open(context.Background(), db.Options{
		Path:       path,
		BackupsDir: backups,
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d2.Close() })
	var yo int
	if err := d2.Read.QueryRow(`SELECT count(*) FROM works_fts WHERE works_fts MATCH ?`, textnorm.Normalize("ёлка")).Scan(&yo); err != nil {
		t.Fatal(err)
	}
	if yo < 1 {
		t.Fatal("startup recovery must rebuild searchable FTS")
	}
	var trig int
	if err := d2.Read.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name IN ('works_fts_au','works_fts_ad')`).Scan(&trig); err != nil {
		t.Fatal(err)
	}
	if trig != 2 {
		t.Fatalf("triggers %d", trig)
	}
}

func TestDeletedRecordDoesNotOverwriteTitle(t *testing.T) {
	d := openCatalog(t)
	importFixture(t, d, nil)
	var id int64
	var title string
	if err := d.Write.QueryRow(`SELECT w.id, w.title FROM works w JOIN editions e ON e.work_id=w.id WHERE e.libid='500001'`).Scan(&id, &title); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write.Exec(`UPDATE works SET title='Keep me', sort_title='keep me' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	importFixture(t, d, nil)
	var after string
	if err := d.Read.QueryRow(`SELECT title FROM works WHERE id=?`, id).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != "Keep me" {
		t.Fatalf("deleted incoming overwrote title: %q", after)
	}
}

func TestDeactivateMissingLibid(t *testing.T) {
	d := openCatalog(t)
	importFixture(t, d, nil)
	if _, err := d.Write.Exec(`INSERT INTO editions(libid, work_id, archive_name, file_name, is_active) VALUES ('gone', 1, 'x.zip', 'f', 1)`); err != nil {
		t.Fatal(err)
	}
	importFixture(t, d, nil)
	var active int
	if err := d.Read.QueryRow(`SELECT is_active FROM editions WHERE libid='gone'`).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != 0 {
		t.Fatal("expected deactivation")
	}
}
