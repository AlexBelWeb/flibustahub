package db

import (
	"context"
	"database/sql"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/textnorm"
)

func openTest(t *testing.T, opts ...func(*Options)) *DB {
	t.Helper()
	dir := t.TempDir()
	opt := Options{
		Path:       filepath.Join(dir, "catalog.sqlite"),
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
		Now:        func() time.Time { return time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC) },
	}
	for _, fn := range opts {
		fn(&opt)
	}
	d, err := Open(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func TestOpenAppliesSchemaAndPragmas(t *testing.T) {
	d := openTest(t)
	var n int
	if err := d.Read.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='works'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("works table missing")
	}
	want := map[string]string{
		"journal_mode": "wal",
		"foreign_keys": "1",
		"busy_timeout": "30000",
		"cache_size":   "-64000",
		"temp_store":   "2",
	}
	for _, pool := range []*sql.DB{d.Read, d.Write} {
		for name, expect := range want {
			got, err := readPragma(context.Background(), pool, name)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if !strings.EqualFold(got, expect) {
				t.Fatalf("PRAGMA %s = %q, want %q", name, got, expect)
			}
		}
		mmap, err := readPragma(context.Background(), pool, "mmap_size")
		if err != nil {
			t.Fatal(err)
		}
		if mmap != "268435456" {
			t.Logf("mmap_size not applied (got %s); recorded, not treated as set", mmap)
		}
	}
}

func TestMigrateTwiceIsNoop(t *testing.T) {
	dir := t.TempDir()
	opt := Options{
		Path:       filepath.Join(dir, "catalog.sqlite"),
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
	}
	d, err := Open(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	_ = d.Close()
	d2, err := Open(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d2.Close() })
	var n int
	if err := d2.Write.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("schema_migrations rows = %d", n)
	}
}

func TestChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	src1 := fstest.MapFS{
		"001_initial.sql": &fstest.MapFile{Data: []byte("CREATE TABLE t(id INTEGER PRIMARY KEY);")},
	}
	opt := Options{
		Path:       filepath.Join(dir, "catalog.sqlite"),
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
		Migrations: src1,
	}
	d, err := Open(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	_ = d.Close()

	src2 := fstest.MapFS{
		"001_initial.sql": &fstest.MapFile{Data: []byte("CREATE TABLE t(id INTEGER PRIMARY KEY); CREATE TABLE u(id INTEGER);")},
	}
	opt.Migrations = src2
	_, err = Open(context.Background(), opt)
	if err == nil {
		t.Fatal("expected checksum error")
	}
	if apperr.As(err).Code != apperr.CodeDBMigrateFailed {
		t.Fatalf("code = %s", apperr.As(err).Code)
	}
}

func TestForeignKeysAndCascade(t *testing.T) {
	d := openTest(t)
	_, err := d.Write.Exec(`INSERT INTO editions(libid, work_id, archive_name, file_name) VALUES ('x', 999, 'a.zip', 'f')`)
	if err == nil {
		t.Fatal("expected FK failure")
	}
	if _, err := d.Write.Exec(`INSERT INTO works(work_key, title, sort_title, authors_text, created_at, updated_at)
		VALUES ('k', 'T', 't', 'A', 'now', 'now')`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write.Exec(`INSERT INTO editions(libid, work_id, archive_name, file_name) VALUES ('x', 1, 'a.zip', 'f')`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write.Exec(`DELETE FROM works WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := d.Write.QueryRow(`SELECT count(*) FROM editions`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("editions left = %d", n)
	}
}

func TestFTSTriggers(t *testing.T) {
	d := openTest(t)
	if _, err := d.Write.Exec(`INSERT INTO works(work_key, title, sort_title, authors_text, created_at, updated_at)
		VALUES ('k', 'Ёлка', 'елка', 'Автор', 'now', 'now')`); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := d.Write.QueryRow(`SELECT count(*) FROM works_fts`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("INSERT must not fill FTS, got %d", n)
	}
	if _, err := d.Write.Exec(`UPDATE works SET title = 'Сосна' WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	var title string
	if err := d.Write.QueryRow(`SELECT title FROM works_fts WHERE rowid = 1`).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != textnorm.Normalize("Сосна") {
		t.Fatalf("fts title = %q", title)
	}
	if err := d.Write.QueryRow(`SELECT count(*) FROM works_fts WHERE title MATCH 'елка'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("old title still in FTS")
	}
	if _, err := d.Write.Exec(`DELETE FROM works WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	if err := d.Write.QueryRow(`SELECT count(*) FROM works_fts`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("fts leftover %d", n)
	}
}

func TestNoInsertFTSTrigger(t *testing.T) {
	d := openTest(t)
	var n int
	if err := d.Read.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='trigger' AND sql LIKE '%AFTER INSERT%'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("unexpected INSERT triggers: %d", n)
	}
}

func TestSQLFunctions(t *testing.T) {
	d := openTest(t)
	var got string
	if err := d.Read.QueryRow(`SELECT normalize('Ёлка')`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != "елка" {
		t.Fatalf("normalize = %q", got)
	}
	if err := d.Read.QueryRow(`SELECT search_norm('Ёлка')`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != " елка " {
		t.Fatalf("search_norm = %q", got)
	}
	var n sql.NullString
	if err := d.Read.QueryRow(`SELECT normalize(NULL)`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n.Valid {
		t.Fatalf("expected NULL, got %q", n.String)
	}
}

func TestBackupVacuumAndRetention(t *testing.T) {
	t0 := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	clock := t0
	d := openTest(t, func(o *Options) {
		o.Now = func() time.Time { return clock }
	})
	if _, err := d.Write.Exec(`INSERT INTO works(work_key, title, sort_title, authors_text, created_at, updated_at)
		VALUES ('k', 'T', 't', 'A', 'now', 'now')`); err != nil {
		t.Fatal(err)
	}
	var files []string
	for i := 0; i < 5; i++ {
		clock = t0.Add(time.Duration(i) * time.Second)
		path, err := d.Backup(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, path)
	}
	matches, err := filepath.Glob(filepath.Join(d.backupsDir, "catalog-*.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 3 {
		t.Fatalf("kept %d backups, want 3: %v", len(matches), matches)
	}
	latest := files[len(files)-1]
	snap, err := sqlOpen(fileDSN(latest))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = snap.Close() }()
	var n int
	if err := snap.QueryRow(`SELECT count(*) FROM works`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("backup works = %d", n)
	}
}

func TestFTSCreateSource(t *testing.T) {
	for _, name := range []string{"works_fts", "authors_fts", "series_fts"} {
		sql, err := FTSCreate(name)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(sql, name) || !strings.Contains(sql, "unicode61") {
			t.Fatalf("unexpected %s: %s", name, sql)
		}
	}
	tr, err := WorksFTSTriggers()
	if err != nil {
		t.Fatal(err)
	}
	if len(tr) != 2 {
		t.Fatalf("triggers %d", len(tr))
	}
}
