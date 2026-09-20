package inpximport

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
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

const seriesLookupSQL = `SELECT group_concat(DISTINCT e.series)
  FROM editions e
 WHERE e.work_id = 1
   AND e.is_active = 1 AND e.is_deleted = 0
   AND e.series IS NOT NULL AND trim(e.series) != ''`

func explainQueryPlan(e interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, query string) (string, error) {
	rows, err := e.QueryContext(context.Background(), "EXPLAIN QUERY PLAN "+query)
	if err != nil {
		return "", err
	}
	defer func() { _ = rows.Close() }()
	var b strings.Builder
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			return "", err
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(detail)
	}
	return b.String(), rows.Err()
}

func TestImportHasStatsBeforeFTS(t *testing.T) {
	d := openCatalog(t)
	lib := t.TempDir()
	if err := testdata.WriteLibraryRoot(lib); err != nil {
		t.Fatal(err)
	}
	svc := New(d, slog.New(slog.DiscardHandler))
	_, err := svc.Import(context.Background(), Options{
		LibraryRoot: lib,
		BeforeFTS: func(conn *sql.Conn) error {
			var one int
			if err := conn.QueryRowContext(context.Background(), `SELECT 1 FROM sqlite_stat1 LIMIT 1`).Scan(&one); err != nil {
				return err
			}
			plan, err := explainQueryPlan(conn, seriesLookupSQL)
			if err != nil {
				return err
			}
			if strings.Contains(plan, "idx_editions_active") {
				return fmt.Errorf("hot series lookup walked idx_editions_active:\n%s", plan)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestImportLogsFinishedSummary(t *testing.T) {
	d := openCatalog(t)
	lib := t.TempDir()
	if err := testdata.WriteLibraryRoot(lib); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	svc := New(d, slog.New(slog.NewJSONHandler(&buf, nil)))
	if _, err := svc.Import(context.Background(), Options{LibraryRoot: lib}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"msg":"import finished"`) {
		t.Fatalf("expected import finished log, got %s", out)
	}
	for _, key := range []string{
		`"backup"`, `"reading"`, `"records"`, `"fts"`, `"warmup"`, `"analyze"`,
		`"wall_ms"`, `"db_bytes"`,
	} {
		if !strings.Contains(out, key) {
			t.Errorf("finished log missing %s", key)
		}
	}
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
	if rep.Notes.Encodings.CP1251 != 3 || rep.Notes.Encodings.UTF8 != 0 {
		t.Fatalf("fixture encodings %+v", rep.Notes.Encodings)
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

func TestImportMapsGenreLabelsToCodes(t *testing.T) {
	d := openCatalog(t)
	lib := t.TempDir()
	if err := testdata.WriteGenreLabelsDump(lib); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	svc := New(d, slog.New(slog.NewTextHandler(&buf, nil)))
	rep, err := svc.Import(context.Background(), Options{LibraryRoot: lib})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Notes.GenreNamesMapped != 2 {
		t.Fatalf("genre_names_mapped=%d, want 2", rep.Notes.GenreNamesMapped)
	}

	codeOf := func(libid string) string {
		t.Helper()
		var code string
		err := d.Read.QueryRow(`SELECT g.code FROM editions e
JOIN edition_genres eg ON eg.edition_id = e.id
JOIN genres g ON g.id = eg.genre_id
WHERE e.libid = ?`, libid).Scan(&code)
		if err != nil {
			t.Fatalf("libid %s: %v", libid, err)
		}
		return code
	}
	if got := codeOf("950001"); got != "nonf_biography" {
		t.Fatalf("biography label stored as %q", got)
	}
	if got := codeOf("950002"); got != "det_espionage" {
		t.Fatalf("espionage label stored as %q", got)
	}
	if got := codeOf("950003"); got != "Дамский детективный роман" {
		t.Fatalf("ambiguous label stored as %q", got)
	}

	var rawLabels int
	if err := d.Read.QueryRow(`SELECT count(*) FROM genres WHERE code IN ('Биографии и мемуары','Шпионский Детектив')`).Scan(&rawLabels); err != nil {
		t.Fatal(err)
	}
	if rawLabels != 0 {
		t.Fatalf("dictionary names must not be stored as codes, got %d", rawLabels)
	}
	if !strings.Contains(buf.String(), "genre name matches several codes") {
		t.Fatalf("expected ambiguous-name log, got %s", buf.String())
	}
}

func TestImportIdempotentAndKeepsPersonalData(t *testing.T) {
	d := openCatalog(t)
	importFixture(t, d, nil)

	var workID int64
	if err := d.Write.QueryRow(`SELECT id FROM works LIMIT 1`).Scan(&workID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write.Exec(`UPDATE works SET rating=8, rating_updated_at='rate-ts', comment='note', comment_updated_at='comment-ts', exported_at='export-ts', want_to_read=1, want_to_read_updated_at='want-ts' WHERE id=?`, workID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write.Exec(`INSERT INTO recently_viewed(work_id, viewed_at) VALUES (?, 'viewed-ts')`, workID); err != nil {
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

	var rating, want int
	var comment, rateAt, commentAt, exported, wantAt string
	if err := d.Read.QueryRow(`SELECT rating, comment, rating_updated_at, comment_updated_at, exported_at, want_to_read, want_to_read_updated_at FROM works WHERE id=?`, workID).Scan(&rating, &comment, &rateAt, &commentAt, &exported, &want, &wantAt); err != nil {
		t.Fatal(err)
	}
	if rating != 8 || comment != "note" || rateAt != "rate-ts" || commentAt != "comment-ts" || exported != "export-ts" || want != 1 || wantAt != "want-ts" {
		t.Fatalf("personal data lost: rating=%d comment=%q rateAt=%q commentAt=%q exported=%q want=%d wantAt=%q", rating, comment, rateAt, commentAt, exported, want, wantAt)
	}
	var viewed string
	if err := d.Read.QueryRow(`SELECT viewed_at FROM recently_viewed WHERE work_id=?`, workID).Scan(&viewed); err != nil {
		t.Fatal(err)
	}
	if viewed != "viewed-ts" {
		t.Fatalf("recently_viewed lost: %q", viewed)
	}

	rep3 := importFixture(t, d, nil)
	_ = rep3
	if err := d.Read.QueryRow(`SELECT rating, comment, rating_updated_at, comment_updated_at, exported_at, want_to_read, want_to_read_updated_at FROM works WHERE id=?`, workID).Scan(&rating, &comment, &rateAt, &commentAt, &exported, &want, &wantAt); err != nil {
		t.Fatal(err)
	}
	if rating != 8 || comment != "note" || rateAt != "rate-ts" || commentAt != "comment-ts" || exported != "export-ts" || want != 1 || wantAt != "want-ts" {
		t.Fatalf("personal data lost after third import")
	}
	if err := d.Read.QueryRow(`SELECT viewed_at FROM recently_viewed WHERE work_id=?`, workID).Scan(&viewed); err != nil {
		t.Fatal(err)
	}
	if viewed != "viewed-ts" {
		t.Fatalf("recently_viewed lost after third import: %q", viewed)
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

func TestImportCancelDuringFTSKeepsData(t *testing.T) {
	d := openCatalog(t)
	lib := t.TempDir()
	if err := testdata.WriteLibraryRoot(lib); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	svc := New(d, slog.New(slog.DiscardHandler))
	rep, err := svc.Import(ctx, Options{
		LibraryRoot: lib,
		BeforeFTS: func(*sql.Conn) error {
			cancel()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("cancel after commit must finish: %v", err)
	}
	if rep.Status != StatusDone {
		t.Fatalf("status %s", rep.Status)
	}
	var works, fts int
	if err := d.Read.QueryRow(`SELECT count(*) FROM works`).Scan(&works); err != nil {
		t.Fatal(err)
	}
	if err := d.Read.QueryRow(`SELECT count(*) FROM works_fts`).Scan(&fts); err != nil {
		t.Fatal(err)
	}
	if works == 0 || fts != works {
		t.Fatalf("works=%d fts=%d", works, fts)
	}
	var dirty string
	if err := d.Read.QueryRow(`SELECT value FROM app_meta WHERE key=?`, db.MetaFTSDirty).Scan(&dirty); err != nil {
		t.Fatal(err)
	}
	if dirty != "0" {
		t.Fatalf("fts_dirty=%s", dirty)
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
	if err := d2.WaitSearchIndex(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !d2.SearchIndexReady() {
		t.Fatal("search index should be ready after recovery")
	}
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

func TestLibIDCollisionsCountsArchiveChange(t *testing.T) {
	d := openCatalog(t)
	rep := importFixture(t, d, nil)
	if rep.LibIDCollisions != 2 {
		t.Fatalf("libid_collisions = %d, want 2 (shared libids across two archives)", rep.LibIDCollisions)
	}
	var arch900, arch910 string
	if err := d.Read.QueryRow(`SELECT archive_name FROM editions WHERE libid='900001'`).Scan(&arch900); err != nil {
		t.Fatal(err)
	}
	if err := d.Read.QueryRow(`SELECT archive_name FROM editions WHERE libid='910001'`).Scan(&arch910); err != nil {
		t.Fatal(err)
	}
	if arch900 != testdata.ArchiveHigh || arch910 != testdata.ArchiveHigh {
		t.Fatalf("kept archives 900001=%s 910001=%s, want %s", arch900, arch910, testdata.ArchiveHigh)
	}
}

func TestRecordsPhaseLeavesWorksFTSEmpty(t *testing.T) {
	d := openCatalog(t)
	var fts int
	importFixture(t, d, func(tx *sql.Tx) error {
		if err := tx.QueryRow(`SELECT count(*) FROM works_fts`).Scan(&fts); err != nil {
			return err
		}
		var works int
		if err := tx.QueryRow(`SELECT count(*) FROM works`).Scan(&works); err != nil {
			return err
		}
		if works == 0 {
			return errors.New("probe ran before any works were inserted")
		}
		return nil
	})
	if fts != 0 {
		t.Fatalf("works_fts rows during records = %d", fts)
	}
}

func TestReimportDoesNotRewriteUnchangedWorks(t *testing.T) {
	d := openCatalog(t)
	importFixture(t, d, nil)
	type row struct {
		id        int64
		updatedAt string
		title     string
	}
	rs, err := d.Read.Query(`SELECT id, updated_at, title FROM works ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	var before []row
	for rs.Next() {
		var r row
		if err := rs.Scan(&r.id, &r.updatedAt, &r.title); err != nil {
			t.Fatal(err)
		}
		before = append(before, r)
	}
	_ = rs.Close()
	if len(before) == 0 {
		t.Fatal("no works")
	}
	importFixture(t, d, nil)
	rs, err = d.Read.Query(`SELECT id, updated_at, title FROM works ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rs.Close() }()
	var i int
	for rs.Next() {
		var r row
		if err := rs.Scan(&r.id, &r.updatedAt, &r.title); err != nil {
			t.Fatal(err)
		}
		if i >= len(before) {
			t.Fatal("work count grew")
		}
		if r != before[i] {
			t.Fatalf("work rewritten: before %+v after %+v", before[i], r)
		}
		i++
	}
	if i != len(before) {
		t.Fatalf("work count %d -> %d", len(before), i)
	}
}

func TestImportUpsertsGenreName(t *testing.T) {
	d := openCatalog(t)
	importFixture(t, d, nil)
	if _, err := d.Write.Exec(`UPDATE genres SET name_ru = 'устарело' WHERE code = 'sf_social'`); err != nil {
		t.Fatal(err)
	}
	importFixture(t, d, nil)
	var name string
	if err := d.Read.QueryRow(`SELECT name_ru FROM genres WHERE code='sf_social'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Социально-психологическая фантастика" {
		t.Fatalf("name_ru not upserted: %q", name)
	}
}

func TestShouldEmitRecordsRequiresBothGates(t *testing.T) {
	if shouldEmitRecords(2000, 50*time.Millisecond, 2000) {
		t.Fatal("must not emit before 100ms")
	}
	if shouldEmitRecords(100, 200*time.Millisecond, 100) {
		t.Fatal("must not emit before 2000 records")
	}
	if !shouldEmitRecords(2000, 100*time.Millisecond, 2000) {
		t.Fatal("must emit when both gates pass")
	}
	if !shouldEmitRecords(0, 0, 0) {
		t.Fatal("recordsSeen 0 is the phase entry")
	}
}

func TestImportProgressBytesAndCommitted(t *testing.T) {
	d := openCatalog(t)
	lib := t.TempDir()
	if err := testdata.WriteLibraryRoot(lib); err != nil {
		t.Fatal(err)
	}
	var last Progress
	var sawBytes, sawCommitted bool
	svc := New(d, slog.New(slog.DiscardHandler))
	_, err := svc.Import(context.Background(), Options{
		LibraryRoot: lib,
		Progress: func(p Progress) {
			last = p
			if p.Phase == PhaseRecords && p.BytesTotal > 0 && p.BytesDone >= 0 {
				sawBytes = true
			}
			if p.Committed {
				sawCommitted = true
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sawBytes {
		t.Fatal("records progress never carried bytes")
	}
	if !sawCommitted {
		t.Fatal("committed flag never set after COMMIT")
	}
	if last.Phase != PhaseWarmup || !last.Committed {
		t.Fatalf("last progress %+v", last)
	}
}

func TestRunWithPulseEmitsEachSecond(t *testing.T) {
	var n atomic.Int32
	emit := func(Progress) { n.Add(1) }
	err := runWithPulse(emit, PhaseFTS, 10, 100, func() error {
		time.Sleep(1100 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n.Load() < 1 {
		t.Fatalf("pulse events = %d, want at least 1 during a 1.1s phase", n.Load())
	}
}
