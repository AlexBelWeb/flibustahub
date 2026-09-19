package downloads

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/repositories"
)

type memStore struct {
	ed repositories.EditionFile
}

func (m memStore) Edition(_ context.Context, id int64) (repositories.EditionFile, error) {
	if id != m.ed.ID {
		return repositories.EditionFile{}, os.ErrNotExist
	}
	return m.ed, nil
}

func writeZip(t *testing.T, dir, archive, entry, body string) {
	t.Helper()
	path := filepath.Join(dir, archive)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(entry)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testSvc(t *testing.T, root, dest string, ed repositories.EditionFile) *Service {
	t.Helper()
	ready := func(context.Context) error { return nil }
	return New(memStore{ed: ed}, func() string { return root }, func() string { return dest },
		func() string { return t.TempDir() }, func() string { return "" }, ready, ready, nil)
}

func TestDownloadWritesCompleteFile(t *testing.T) {
	root := t.TempDir()
	dest := t.TempDir()
	writeZip(t, root, "a.zip", "book.fb2", "hello-book")
	ed := repositories.EditionFile{
		ID: 7, WorkID: 1, Title: "Title", AuthorsText: "Author",
		ArchiveName: "a.zip", FileName: "book", FileExt: "fb2", Active: true,
	}
	svc := testSvc(t, root, dest, ed)
	out, err := svc.Download(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if out.FileName != "Author — Title.fb2" {
		t.Fatalf("name %q", out.FileName)
	}
	body, err := os.ReadFile(out.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello-book" {
		t.Fatalf("body %q", body)
	}
	if _, err := os.Stat(out.Path + ".part"); !os.IsNotExist(err) {
		t.Fatal("part file left behind")
	}
}

func TestWriteAtomicDeletesOnCancel(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "book.fb2")
	ctx, cancel := context.WithCancel(context.Background())
	r, w := io.Pipe()
	go func() {
		_, _ = w.Write([]byte("abc"))
		cancel()
		time.Sleep(20 * time.Millisecond)
		_, _ = w.Write(bytes.Repeat([]byte("x"), 64_000))
		_ = w.Close()
	}()
	err := writeAtomic(ctx, dest, r)
	_ = r.Close()
	if err == nil {
		t.Fatal("expected cancel")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("complete file appeared")
	}
	if _, err := os.Stat(dest + ".part"); !os.IsNotExist(err) {
		t.Fatal("part file left behind")
	}
}

func TestDownloadCancelledContext(t *testing.T) {
	root := t.TempDir()
	dest := t.TempDir()
	writeZip(t, root, "a.zip", "book.fb2", "hello")
	ed := repositories.EditionFile{
		ID: 8, WorkID: 1, Title: "T", AuthorsText: "A",
		ArchiveName: "a.zip", FileName: "book", FileExt: "fb2", Active: true,
	}
	svc := testSvc(t, root, dest, ed)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.Download(ctx, 8)
	if apperr.As(err).Code != apperr.CodeCancelled {
		t.Fatalf("code %s err=%v", apperr.As(err).Code, err)
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("leftover %v", entries)
	}
}

func TestDownloadMissingArchive(t *testing.T) {
	root := t.TempDir()
	dest := t.TempDir()
	ed := repositories.EditionFile{
		ID: 9, WorkID: 1, Title: "T", AuthorsText: "A",
		ArchiveName: "gone.zip", FileName: "book", FileExt: "fb2", Active: true,
	}
	svc := testSvc(t, root, dest, ed)
	_, err := svc.Download(context.Background(), 9)
	if apperr.As(err).Code != apperr.CodeArchiveMissing {
		t.Fatalf("code %s", apperr.As(err).Code)
	}
}

func TestOpenStreamClosesZip(t *testing.T) {
	root := t.TempDir()
	writeZip(t, root, "a.zip", "book.fb2", "body")
	ed := repositories.EditionFile{
		ID: 1, WorkID: 1, ArchiveName: "a.zip", FileName: "book", FileExt: "fb2", Active: true,
	}
	stream, err := openStream(root, ed)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4)
	if _, err := stream.Read(buf); err != nil && err != io.EOF {
		t.Fatal(err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestShowInFolderMissingFile(t *testing.T) {
	svc := New(nil, nil, nil, nil, nil, nil, nil, nil)
	err := svc.ShowInFolder(filepath.Join(t.TempDir(), "gone.fb2"))
	if apperr.As(err).Code != apperr.CodeOpenDirFailed {
		t.Fatalf("code %s", apperr.As(err).Code)
	}
}

func TestCancelUnknownOpIsNoop(t *testing.T) {
	svc := testSvc(t, t.TempDir(), t.TempDir(), repositories.EditionFile{ID: 1, Active: true})
	svc.Cancel(1, KindDownload)
	svc.Cancel(1, KindRead)
}

func TestCleanDirRemovesFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.fb2")
	if err := os.WriteFile(path, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	CleanDir(dir, nil)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file remained")
	}
}

func TestFileNameFromStreamMatchesDownload(t *testing.T) {
	root := t.TempDir()
	dest := t.TempDir()
	writeZip(t, root, "a.zip", "n.fb2", "data")
	ed := repositories.EditionFile{
		ID: 3, WorkID: 1, Title: "Name.", AuthorsText: "Author",
		ArchiveName: "a.zip", FileName: "n", FileExt: "fb2", Active: true,
	}
	svc := testSvc(t, root, dest, ed)
	out, err := svc.Download(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasSuffix(out.FileName, ".") || strings.Contains(out.FileName, "..") {
		t.Fatalf("bad name %q", out.FileName)
	}
	if out.FileName != "Author — Name.fb2" {
		t.Fatalf("name %q", out.FileName)
	}
}
