package covers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/fb2"
	"github.com/alexbelweb/flibustahub/internal/repositories"
)

var jpegCover = []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F'}

func openCoverSvc(t *testing.T, lib, covers string) (*Service, *db.DB) {
	t.Helper()
	d, err := db.Open(context.Background(), db.Options{
		Path:       filepath.Join(t.TempDir(), "catalog.sqlite"),
		BackupsDir: filepath.Join(t.TempDir(), "backups"),
		Log:        slog.New(slog.DiscardHandler),
		Now:        func() time.Time { return time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	svc := New(repositories.NewCatalog(d), func() string { return covers }, func() string { return lib }, nil, slog.New(slog.DiscardHandler), time.Now, nil)
	t.Cleanup(svc.Stop)
	return svc, d
}

func seedWork(t *testing.T, d *db.DB, archive, file string) {
	t.Helper()
	stmts := []string{
		`INSERT INTO authors(id, author_key, last_name, first_name, middle_name, display_name, sort_name) VALUES (1,'a','A','B','','A B','a b')`,
		`INSERT INTO works(id, work_key, title, sort_title, authors_text, lang, created_at, updated_at) VALUES (1,'k','Title','title','A B','ru','t','t')`,
		`INSERT INTO work_authors(work_id, author_id, position) VALUES (1,1,0)`,
		`INSERT INTO editions(id, libid, work_id, archive_name, file_name, file_ext, added_date, is_deleted, is_active)
		 VALUES (1,'1',1,'` + archive + `','` + file + `','fb2','2020-01-01',0,1)`,
	}
	for _, s := range stmts {
		if _, err := d.Write.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
}

func writeFB2Zip(t *testing.T, zipPath, entry string, cover []byte, annotation string, closeTag bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		t.Fatal(err)
	}
	b64 := base64.StdEncoding.EncodeToString(cover)
	body := `<?xml version="1.0" encoding="utf-8"?>` +
		`<FictionBook><description><title-info>` +
		`<annotation><p>` + annotation + `</p></annotation>` +
		`<coverpage><image href="#cover.jpg"/></coverpage>` +
		`</title-info></description>` +
		`<binary id="cover.jpg" content-type="image/png">` + b64 + `</binary>`
	if closeTag {
		body += `</FictionBook>`
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	fw, err := w.Create(entry)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(zipPath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestServeCoverAndAnnotationOneArchive(t *testing.T) {
	lib := t.TempDir()
	covers := t.TempDir()
	writeFB2Zip(t, filepath.Join(lib, "a.zip"), "110119.fb2", jpegCover, "Hello", true)
	svc, d := openCoverSvc(t, lib, covers)
	seedWork(t, d, "a.zip", "110119")

	hit, err := svc.ServeCover(context.Background(), 1, PrioOpen)
	if err != nil {
		t.Fatal(err)
	}
	if hit.None || hit.MIME != "image/jpeg" {
		t.Fatalf("%+v", hit)
	}
	if fb2.DetectImage(mustRead(t, hit.Path)) != fb2.ImageJPEG {
		t.Fatal("cached bytes are not jpeg")
	}

	ann, err := svc.Annotation(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if ann.Text == nil || *ann.Text != "Hello" || !ann.Checked {
		t.Fatalf("%+v", ann)
	}
}

func TestUnknownMagicWritesNone(t *testing.T) {
	lib := t.TempDir()
	covers := t.TempDir()
	writeFB2Zip(t, filepath.Join(lib, "a.zip"), "110119.fb2", []byte("XXXX"), "A", true)
	svc, d := openCoverSvc(t, lib, covers)
	seedWork(t, d, "a.zip", "110119")
	hit, err := svc.ServeCover(context.Background(), 1, PrioVisible)
	if err != nil {
		t.Fatal(err)
	}
	if !hit.None {
		t.Fatal("expected none marker")
	}
	if _, err := os.Stat(filepath.Join(covers, "1.none")); err != nil {
		t.Fatal(err)
	}
}

func TestMissingArchiveDoesNotWriteNone(t *testing.T) {
	lib := t.TempDir()
	covers := t.TempDir()
	svc, d := openCoverSvc(t, lib, covers)
	seedWork(t, d, "missing.zip", "110119")
	_, err := svc.ServeCover(context.Background(), 1, PrioVisible)
	if apperr.As(err).Code != apperr.CodeArchiveMissing {
		t.Fatalf("err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(covers, "1.none")); !os.IsNotExist(err) {
		t.Fatal(".none must not be written for a missing archive")
	}
	_, err = svc.ServeCover(context.Background(), 1, PrioVisible)
	if apperr.As(err).Code != apperr.CodeArchiveMissing {
		t.Fatal("session must remember the miss")
	}
}

func TestTruncatedFB2DoesNotWriteNone(t *testing.T) {
	lib := t.TempDir()
	covers := t.TempDir()
	writeFB2Zip(t, filepath.Join(lib, "a.zip"), "110119.fb2", jpegCover, "A", false)
	svc, d := openCoverSvc(t, lib, covers)
	seedWork(t, d, "a.zip", "110119")
	_, err := svc.ServeCover(context.Background(), 1, PrioVisible)
	if apperr.As(err).Code != apperr.CodeFB2Unreadable {
		t.Fatalf("err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(covers, "1.none")); !os.IsNotExist(err) {
		t.Fatal(".none must not be written for truncated fb2")
	}
}

func TestOfflineSkipsExtract(t *testing.T) {
	covers := t.TempDir()
	svc, d := openCoverSvc(t, filepath.Join(t.TempDir(), "no-such"), covers)
	seedWork(t, d, "a.zip", "110119")
	_, err := svc.ServeCover(context.Background(), 1, PrioVisible)
	if apperr.As(err).Code != apperr.CodeLibraryOffline {
		t.Fatalf("err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(covers, "1.none")); !os.IsNotExist(err) {
		t.Fatal(".none must not be written when the library is offline")
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestStopWaitReturns(t *testing.T) {
	svc, _ := openCoverSvc(t, t.TempDir(), t.TempDir())
	svc.Stop()
	start := time.Now()
	svc.Wait(2 * time.Second)
	if time.Since(start) > time.Second {
		t.Fatalf("Wait after Stop took %s", time.Since(start))
	}
}
