package personal

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/repositories"
)

func openPersonal(t *testing.T) (*Service, *db.DB) {
	t.Helper()
	var tick atomic.Int64
	base := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	d, err := db.Open(context.Background(), db.Options{
		Path:       filepath.Join(t.TempDir(), "catalog.sqlite"),
		BackupsDir: filepath.Join(t.TempDir(), "backups"),
		Log:        slog.New(slog.DiscardHandler),
		Now: func() time.Time {
			return base.Add(time.Duration(tick.Add(1)) * time.Second)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if _, err := d.Write.Exec(`INSERT INTO works(id, work_key, title, sort_title, authors_text, created_at, updated_at)
		VALUES (1, 'k-one', 'One', 'one', 'Author', 't', 't'),
		       (2, 'k-two', 'Two', 'two', 'Author', 't', 't')`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write.Exec(`INSERT INTO editions(libid, work_id, archive_name, file_name, is_active, is_deleted)
		VALUES ('1', 1, 'a.zip', 'f', 1, 0), ('2', 2, 'a.zip', 'f', 1, 0)`); err != nil {
		t.Fatal(err)
	}
	svc := New(d, slog.New(slog.DiscardHandler))
	svc.now = func() time.Time { return time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC) }
	return svc, d
}

func TestSetRatingUnchangedDoesNotTouchTimestamp(t *testing.T) {
	svc, d := openPersonal(t)
	ctx := context.Background()
	eight := 8
	if err := svc.SetRating(ctx, 1, &eight); err != nil {
		t.Fatal(err)
	}
	var ts1 string
	if err := d.Read.QueryRow(`SELECT rating_updated_at FROM works WHERE id=1`).Scan(&ts1); err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC) }
	if err := svc.SetRating(ctx, 1, &eight); err != nil {
		t.Fatal(err)
	}
	var ts2 string
	if err := d.Read.QueryRow(`SELECT rating_updated_at FROM works WHERE id=1`).Scan(&ts2); err != nil {
		t.Fatal(err)
	}
	if ts1 != ts2 {
		t.Fatalf("timestamp moved on unchanged rating: %q -> %q", ts1, ts2)
	}
}

func TestClearRatingMovesTimestamp(t *testing.T) {
	svc, d := openPersonal(t)
	ctx := context.Background()
	eight := 8
	if err := svc.SetRating(ctx, 1, &eight); err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC) }
	if err := svc.SetRating(ctx, 1, nil); err != nil {
		t.Fatal(err)
	}
	var rating any
	var ts string
	if err := d.Read.QueryRow(`SELECT rating, rating_updated_at FROM works WHERE id=1`).Scan(&rating, &ts); err != nil {
		t.Fatal(err)
	}
	if rating != nil {
		t.Fatalf("rating still %v", rating)
	}
	if ts != "2026-09-21T12:00:00.000Z" {
		t.Fatalf("ts %q", ts)
	}
}

func TestUnsyncedSnapshotAfterFirstExport(t *testing.T) {
	svc, _ := openPersonal(t)
	ctx := context.Background()
	eight := 8
	if err := svc.SetRating(ctx, 1, &eight); err != nil {
		t.Fatal(err)
	}
	snap, err := svc.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snap.UnsyncedCount != 1 || snap.LastExportAt != "" {
		t.Fatalf("before export %+v", snap)
	}
	path := filepath.Join(t.TempDir(), "out.csv")
	if _, err := svc.Export(ctx, path); err != nil {
		t.Fatal(err)
	}
	snap, err = svc.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snap.UnsyncedCount != 0 || snap.LastExportAt == "" {
		t.Fatalf("after export %+v", snap)
	}
}

func TestExportCSVHasBOMAndCRLF(t *testing.T) {
	svc, _ := openPersonal(t)
	ctx := context.Background()
	eight := 8
	if err := svc.SetRating(ctx, 1, &eight); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "out.csv")
	if _, err := svc.Export(ctx, path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(raw, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatal("missing BOM")
	}
	if !bytes.Contains(raw, []byte("\r\n")) {
		t.Fatal("missing CRLF")
	}
	if !bytes.Contains(raw, []byte("4")) {
		t.Fatalf("expected stars 4 in %s", raw)
	}
}

func TestExportJSONWantBool(t *testing.T) {
	svc, _ := openPersonal(t)
	ctx := context.Background()
	if err := svc.SetWantToRead(ctx, 1, true); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "out.json")
	if _, err := svc.Export(ctx, path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rows []dumpRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].WantToRead == nil || !*rows[0].WantToRead {
		t.Fatalf("%+v", rows)
	}
}

func TestClearThenExportThenImportClearsOtherDB(t *testing.T) {
	a, _ := openPersonal(t)
	b, dbB := openPersonal(t)
	ctx := context.Background()
	eight := 8
	if err := a.SetRating(ctx, 1, &eight); err != nil {
		t.Fatal(err)
	}
	if err := b.SetRating(ctx, 1, &eight); err != nil {
		t.Fatal(err)
	}
	a.now = func() time.Time { return time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC) }
	if err := a.SetRating(ctx, 1, nil); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "out.json")
	if _, err := a.Export(ctx, path); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Import(ctx, path); err != nil {
		t.Fatal(err)
	}
	var rating any
	if err := dbB.Read.QueryRow(`SELECT rating FROM works WHERE id=1`).Scan(&rating); err != nil {
		t.Fatal(err)
	}
	if rating != nil {
		t.Fatalf("other db rating survived: %v", rating)
	}
}

func TestFieldWithoutTimestamp(t *testing.T) {
	svc, d := openPersonal(t)
	ctx := context.Background()
	dir := t.TempDir()
	emptyLocal := filepath.Join(dir, "empty-local.json")
	if err := os.WriteFile(emptyLocal, []byte(`[{"workKey":"k-one","workId":1,"authors":"A","title":"One","rating":5,"comment":null,"wantToRead":null,"ratingUpdatedAt":"","commentUpdatedAt":"","wantToReadUpdatedAt":""}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := svc.Import(ctx, emptyLocal)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Applied != 1 {
		t.Fatalf("empty local should apply, %+v", rep)
	}
	var rating int
	if err := d.Read.QueryRow(`SELECT rating FROM works WHERE id=1`).Scan(&rating); err != nil || rating != 10 {
		t.Fatalf("rating %d %v", rating, err)
	}

	eight := 8
	if err := svc.SetRating(ctx, 2, &eight); err != nil {
		t.Fatal(err)
	}
	conflict := filepath.Join(dir, "conflict.json")
	if err := os.WriteFile(conflict, []byte(`[{"workKey":"k-two","workId":2,"authors":"A","title":"Two","rating":5,"comment":null,"wantToRead":null,"ratingUpdatedAt":"","commentUpdatedAt":"","wantToReadUpdatedAt":""}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err = svc.Import(ctx, conflict)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Applied != 0 || len(rep.Notes) == 0 || rep.Notes[0].Reason != NoteNoTimestamp {
		t.Fatalf("local value must survive: %+v", rep)
	}
}

func TestWantToReadSpellingsAndInvalidRatings(t *testing.T) {
	svc, d := openPersonal(t)
	ctx := context.Background()
	dir := t.TempDir()
	body := "workKey,workId,authors,title,rating,comment,wantToRead,ratingUpdatedAt,commentUpdatedAt,wantToReadUpdatedAt\r\n" +
		"k-one,1,A,One,,,true,2026-09-20T12:00:00Z,,2026-09-20T12:00:00Z\r\n"
	path := filepath.Join(dir, "want.csv")
	if err := os.WriteFile(path, append([]byte{0xEF, 0xBB, 0xBF}, []byte(body)...), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Import(ctx, path); err != nil {
		t.Fatal(err)
	}
	var want int
	if err := d.Read.QueryRow(`SELECT want_to_read FROM works WHERE id=1`).Scan(&want); err != nil || want != 1 {
		t.Fatalf("want %d %v", want, err)
	}

	for _, spelling := range []string{"1", "TRUE", "false", "0"} {
		row := "workKey,workId,authors,title,rating,comment,wantToRead,ratingUpdatedAt,commentUpdatedAt,wantToReadUpdatedAt\r\n" +
			"k-two,2,A,Two,,," + spelling + ",,,2026-09-22T12:00:00Z\r\n"
		p := filepath.Join(dir, spelling+".csv")
		if err := os.WriteFile(p, []byte(row), 0o644); err != nil {
			t.Fatal(err)
		}
		rep, err := svc.PreviewImport(ctx, p)
		if err != nil {
			t.Fatal(err)
		}
		if rep.Invalid != 0 {
			t.Fatalf("%s invalid %+v", spelling, rep)
		}
	}

	for _, bad := range []string{"0", "5.5", "4.25", "7"} {
		row := "workKey,workId,authors,title,rating,comment,wantToRead,ratingUpdatedAt,commentUpdatedAt,wantToReadUpdatedAt\r\n" +
			"k-one,1,A,One," + bad + ",,,2026-09-20T12:00:00Z,,\r\n"
		p := filepath.Join(dir, "bad-"+bad+".csv")
		if err := os.WriteFile(p, []byte(row), 0o644); err != nil {
			t.Fatal(err)
		}
		rep, err := svc.PreviewImport(ctx, p)
		if err != nil {
			t.Fatal(err)
		}
		if rep.Invalid != 1 {
			t.Fatalf("%s should be invalid: %+v", bad, rep)
		}
	}
}

func TestImportDoesNotTouchCatalog(t *testing.T) {
	svc, d := openPersonal(t)
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "in.json")
	if err := os.WriteFile(path, []byte(`[{"workKey":"k-one","workId":1,"authors":"Hacked","title":"Nope","rating":5,"comment":"x","wantToRead":true,"ratingUpdatedAt":"2026-09-20T12:00:00Z","commentUpdatedAt":"2026-09-20T12:00:00Z","wantToReadUpdatedAt":"2026-09-20T12:00:00Z"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Import(ctx, path); err != nil {
		t.Fatal(err)
	}
	var title, authors string
	if err := d.Read.QueryRow(`SELECT title, authors_text FROM works WHERE id=1`).Scan(&title, &authors); err != nil {
		t.Fatal(err)
	}
	if title != "One" || authors != "Author" {
		t.Fatalf("catalog mutated: %q %q", title, authors)
	}
}

func TestCommentUnchangedSkipped(t *testing.T) {
	svc, d := openPersonal(t)
	ctx := context.Background()
	if err := svc.SetComment(ctx, 1, "note"); err != nil {
		t.Fatal(err)
	}
	var ts1 string
	if err := d.Read.QueryRow(`SELECT comment_updated_at FROM works WHERE id=1`).Scan(&ts1); err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return time.Date(2026, 9, 22, 1, 0, 0, 0, time.UTC) }
	if err := svc.SetComment(ctx, 1, "note"); err != nil {
		t.Fatal(err)
	}
	var ts2 string
	if err := d.Read.QueryRow(`SELECT comment_updated_at FROM works WHERE id=1`).Scan(&ts2); err != nil {
		t.Fatal(err)
	}
	if ts1 != ts2 {
		t.Fatal(ts2)
	}
}

func TestExportableIncludesClearedValues(t *testing.T) {
	svc, _ := openPersonal(t)
	ctx := context.Background()
	eight := 8
	if err := svc.SetRating(ctx, 1, &eight); err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC) }
	if err := svc.SetRating(ctx, 1, nil); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "out.json")
	res, err := svc.Export(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 1 {
		t.Fatalf("cleared rating must still export, count=%d", res.Count)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), `"rating": null`) && !strings.Contains(string(raw), `"rating":null`) {
		t.Fatalf("expected null rating in %s", raw)
	}
}

func TestBlankCommentSavedAsNull(t *testing.T) {
	svc, d := openPersonal(t)
	ctx := context.Background()
	if err := svc.SetComment(ctx, 1, "note"); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetComment(ctx, 1, " \t "); err != nil {
		t.Fatal(err)
	}
	var comment any
	var ts string
	if err := d.Read.QueryRow(`SELECT comment, comment_updated_at FROM works WHERE id=1`).Scan(&comment, &ts); err != nil {
		t.Fatal(err)
	}
	if comment != nil {
		t.Fatalf("blank comment stored as %v", comment)
	}
	if ts != "2026-09-20T12:00:00.000Z" {
		t.Fatalf("ts %q", ts)
	}
}

func TestMarkExportedSkipsRowsChangedAfterRead(t *testing.T) {
	svc, d := openPersonal(t)
	ctx := context.Background()
	eight := 8
	ten := 10
	if err := svc.SetRating(ctx, 1, &eight); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetRating(ctx, 2, &eight); err != nil {
		t.Fatal(err)
	}
	cat := repositories.NewCatalog(d)
	rows, err := cat.ExportableWorks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC) }
	if err := svc.SetRating(ctx, 1, &ten); err != nil {
		t.Fatal(err)
	}
	if err := cat.MarkExported(ctx, rows, Stamp(time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC))); err != nil {
		t.Fatal(err)
	}
	var exp1, exp2 any
	if err := d.Read.QueryRow(`SELECT exported_at FROM works WHERE id=1`).Scan(&exp1); err != nil {
		t.Fatal(err)
	}
	if err := d.Read.QueryRow(`SELECT exported_at FROM works WHERE id=2`).Scan(&exp2); err != nil {
		t.Fatal(err)
	}
	if exp1 != nil {
		t.Fatalf("changed row marked exported: %v", exp1)
	}
	if exp2 == nil {
		t.Fatal("unchanged row not marked")
	}
}

func TestImportWithoutWritesSkipsBackup(t *testing.T) {
	ctx := context.Background()
	backups := filepath.Join(t.TempDir(), "backups")
	d, err := db.Open(context.Background(), db.Options{
		Path:       filepath.Join(t.TempDir(), "catalog.sqlite"),
		BackupsDir: backups,
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if _, err := d.Write.Exec(`INSERT INTO works(id, work_key, title, sort_title, authors_text, created_at, updated_at)
		VALUES (1, 'k-one', 'One', 'one', 'Author', 't', 't')`); err != nil {
		t.Fatal(err)
	}
	svc := New(d, slog.New(slog.DiscardHandler))
	path := filepath.Join(t.TempDir(), "in.json")
	if err := os.WriteFile(path, []byte(`[{"workKey":"missing","title":"Gone"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := svc.Import(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.NotFound != 1 || rep.Applied != 0 {
		t.Fatalf("%+v", rep)
	}
	matches, err := filepath.Glob(filepath.Join(backups, "catalog-personal-*.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("backup on empty apply: %v", matches)
	}
}

func TestImportSucceedsWhenNotFoundFileUnwritable(t *testing.T) {
	svc, d := openPersonal(t)
	ctx := context.Background()
	eight := 8
	if err := svc.SetRating(ctx, 1, &eight); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "in.json")
	body := `[{"workKey":"k-one","workId":1,"authors":"A","title":"One","rating":5,"comment":null,"wantToRead":null,"ratingUpdatedAt":"2026-09-22T12:00:00.000Z","commentUpdatedAt":"","wantToReadUpdatedAt":""},{"workKey":"missing","title":"Gone"}]`
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "in.notfound.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	rep, err := svc.Import(ctx, src)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Applied != 1 || rep.NotFound != 1 {
		t.Fatalf("%+v", rep)
	}
	if rep.NotFoundPath != "" {
		t.Fatalf("path leaked after write failure: %q", rep.NotFoundPath)
	}
	var rating int
	if err := d.Read.QueryRow(`SELECT rating FROM works WHERE id=1`).Scan(&rating); err != nil || rating != 10 {
		t.Fatalf("applied rating %d %v", rating, err)
	}
}
