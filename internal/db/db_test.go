package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/textnorm"
	"github.com/alexbelweb/flibustahub/migrations"
	"golang.org/x/text/encoding/charmap"
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
	if n != 6 {
		t.Fatalf("schema_migrations rows = %d", n)
	}
}

func TestWorksAddedDateMigration(t *testing.T) {
	d := openTest(t)
	var n int
	if err := d.Read.QueryRow(`SELECT count(*) FROM pragma_table_info('works') WHERE name = 'added_date'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("works.added_date missing")
	}
	var notnull int
	var dflt sql.NullString
	if err := d.Read.QueryRow(`SELECT "notnull", dflt_value FROM pragma_table_info('works') WHERE name = 'added_date'`).Scan(&notnull, &dflt); err != nil {
		t.Fatal(err)
	}
	if notnull != 1 {
		t.Fatal("works.added_date must be NOT NULL")
	}
	if err := d.Read.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='index' AND name='idx_works_added'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("idx_works_added missing")
	}
	if err := d.Read.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='index' AND name='idx_series_sort'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("idx_series_sort missing")
	}
	if err := d.Read.QueryRow(`SELECT count(*) FROM pragma_table_info('works') WHERE name = 'want_to_read'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("works.want_to_read missing")
	}
	if err := d.Read.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='index' AND name='idx_works_unsynced_rating'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal("idx_works_unsynced_rating missing")
	}
	var total, listable string
	if err := d.Read.QueryRow(`SELECT value FROM app_meta WHERE key = ?`, MetaWorksTotal).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if err := d.Read.QueryRow(`SELECT value FROM app_meta WHERE key = ?`, MetaWorksListable).Scan(&listable); err != nil {
		t.Fatal(err)
	}
	if total != "0" || listable != "0" {
		t.Fatalf("empty catalog counters total=%s listable=%s", total, listable)
	}
}

func TestMigration002FillsAddedDateAndCounts(t *testing.T) {
	initial, err := migrations.FS.ReadFile("001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	opt := Options{
		Path:       filepath.Join(dir, "catalog.sqlite"),
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
		Migrations: fstest.MapFS{"001_initial.sql": &fstest.MapFile{Data: initial}},
	}
	d, err := Open(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.Write.Exec(`INSERT INTO works(id, work_key, title, sort_title, authors_text, created_at, updated_at, rating)
		VALUES (1, 'k1', 'Has date', 'has date', 'A', 't', 't', NULL),
		       (2, 'k2', 'Ghost', 'ghost', 'B', 't', 't', 9)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.Write.Exec(`INSERT INTO editions(libid, work_id, archive_name, file_name, added_date, is_deleted, is_active)
		VALUES ('1', 1, 'a.zip', 'f.fb2', '2024-02-03', 0, 1)`)
	if err != nil {
		t.Fatal(err)
	}
	_ = d.Close()

	d2, err := Open(context.Background(), Options{
		Path:       opt.Path,
		BackupsDir: opt.BackupsDir,
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d2.Close() })

	var dated, ghost string
	if err := d2.Read.QueryRow(`SELECT added_date FROM works WHERE id = 1`).Scan(&dated); err != nil {
		t.Fatal(err)
	}
	if dated != "2024-02-03" {
		t.Fatalf("visible work added_date = %q", dated)
	}
	if err := d2.Read.QueryRow(`SELECT added_date FROM works WHERE id = 2`).Scan(&ghost); err != nil {
		t.Fatal(err)
	}
	if ghost != "" {
		t.Fatalf("missing date must be empty string, got %q", ghost)
	}
	var total, listable string
	if err := d2.Read.QueryRow(`SELECT value FROM app_meta WHERE key = ?`, MetaWorksTotal).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if err := d2.Read.QueryRow(`SELECT value FROM app_meta WHERE key = ?`, MetaWorksListable).Scan(&listable); err != nil {
		t.Fatal(err)
	}
	if total != "2" || listable != "2" {
		t.Fatalf("counters total=%s listable=%s", total, listable)
	}
}

func TestMigration003ClearsImplausibleAnnotations(t *testing.T) {
	m1, err := migrations.FS.ReadFile("001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	m2, err := migrations.FS.ReadFile("002_works_added_date.sql")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	opt := Options{
		Path:       filepath.Join(dir, "catalog.sqlite"),
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
		Migrations: fstest.MapFS{
			"001_initial.sql":          &fstest.MapFile{Data: m1},
			"002_works_added_date.sql": &fstest.MapFile{Data: m2},
		},
	}
	d, err := Open(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.Write.Exec(`INSERT INTO works(id, work_key, title, sort_title, authors_text, created_at, updated_at, annotation, annotation_checked_at)
		VALUES (1, 'k1', 'Good', 'good', 'A', 't', 't', 'Обычный текст аннотации.', '2026-01-01T00:00:00Z'),
		       (2, 'k2', 'Bad', 'bad', 'B', 't', 't', '╔══╗ © ¤ ░▒▓│┤', '2026-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	_ = d.Close()

	d2, err := Open(context.Background(), Options{
		Path:       opt.Path,
		BackupsDir: opt.BackupsDir,
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d2.Close() })

	var good sql.NullString
	var goodAt sql.NullString
	if err := d2.Read.QueryRow(`SELECT annotation, annotation_checked_at FROM works WHERE id = 1`).Scan(&good, &goodAt); err != nil {
		t.Fatal(err)
	}
	if !good.Valid || good.String != "Обычный текст аннотации." || !goodAt.Valid {
		t.Fatalf("good annotation lost: %v %v", good, goodAt)
	}
	var bad sql.NullString
	var badAt sql.NullString
	if err := d2.Read.QueryRow(`SELECT annotation, annotation_checked_at FROM works WHERE id = 2`).Scan(&bad, &badAt); err != nil {
		t.Fatal(err)
	}
	if bad.Valid || badAt.Valid {
		t.Fatalf("implausible annotation must be cleared, got %v %v", bad, badAt)
	}
}

func TestMigration004ClearsMojibakeWrittenAfter003(t *testing.T) {
	m1, err := migrations.FS.ReadFile("001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	m2, err := migrations.FS.ReadFile("002_works_added_date.sql")
	if err != nil {
		t.Fatal(err)
	}
	m3, err := migrations.FS.ReadFile("003_reset_implausible_annotations.sql")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	opt := Options{
		Path:       filepath.Join(dir, "catalog.sqlite"),
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
		Migrations: fstest.MapFS{
			"001_initial.sql":                       &fstest.MapFile{Data: m1},
			"002_works_added_date.sql":              &fstest.MapFile{Data: m2},
			"003_reset_implausible_annotations.sql": &fstest.MapFile{Data: m3},
		},
	}
	d, err := Open(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	utf := []byte("Он говорил, что я его пара, его Истинная.")
	mojibake, err := charmap.CodePage866.NewDecoder().Bytes(utf)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.Write.Exec(`INSERT INTO works(id, work_key, title, sort_title, authors_text, created_at, updated_at, annotation, annotation_checked_at)
		VALUES (1, 'k1', 'T', 't', 'A', 't', 't', ?, '2026-09-19T09:10:00Z')`, string(mojibake))
	if err != nil {
		t.Fatal(err)
	}
	_ = d.Close()

	d2, err := Open(context.Background(), Options{
		Path:       opt.Path,
		BackupsDir: opt.BackupsDir,
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d2.Close() })
	var ann sql.NullString
	var at sql.NullString
	if err := d2.Read.QueryRow(`SELECT annotation, annotation_checked_at FROM works WHERE id = 1`).Scan(&ann, &at); err != nil {
		t.Fatal(err)
	}
	if ann.Valid || at.Valid {
		t.Fatalf("mojibake written after 003 must be cleared, got %v %v", ann, at)
	}
}

func TestMigration005MovesEmptySortKeysLast(t *testing.T) {
	limited := fstest.MapFS{}
	for _, name := range []string{
		"001_initial.sql",
		"002_works_added_date.sql",
		"003_reset_implausible_annotations.sql",
		"004_reset_implausible_annotations.sql",
	} {
		data, err := migrations.FS.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		limited[name] = &fstest.MapFile{Data: data}
	}
	dir := t.TempDir()
	opt := Options{
		Path:       filepath.Join(dir, "catalog.sqlite"),
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
		Migrations: limited,
	}
	d, err := Open(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.Write.Exec(`INSERT INTO works(id, work_key, title, sort_title, authors_text, created_at, updated_at)
		VALUES (1, 'k1', '_', '', 'A', 't', 't'),
		       (2, 'k2', 'Ёлка', 'елка', 'A', 't', 't')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.Write.Exec(`INSERT INTO authors(id, author_key, last_name, first_name, middle_name, display_name, sort_name)
		VALUES (1, 'empty', '', '', '', '', '')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.Write.Exec(`INSERT INTO series(id, name, sort_name, work_count) VALUES (1, '...', '', 0)`)
	if err != nil {
		t.Fatal(err)
	}
	_ = d.Close()

	d2, err := Open(context.Background(), Options{
		Path:       opt.Path,
		BackupsDir: opt.BackupsDir,
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d2.Close() })

	const sentinel = "\uFFFF"
	var sortTitle, sortAuthor, sortSeries string
	if err := d2.Read.QueryRow(`SELECT sort_title FROM works WHERE id = 1`).Scan(&sortTitle); err != nil {
		t.Fatal(err)
	}
	if err := d2.Read.QueryRow(`SELECT sort_name FROM authors WHERE id = 1`).Scan(&sortAuthor); err != nil {
		t.Fatal(err)
	}
	if err := d2.Read.QueryRow(`SELECT sort_name FROM series WHERE id = 1`).Scan(&sortSeries); err != nil {
		t.Fatal(err)
	}
	if sortTitle != sentinel || sortAuthor != sentinel || sortSeries != sentinel {
		t.Fatalf("sort keys %q %q %q", sortTitle, sortAuthor, sortSeries)
	}
	var firstTitle string
	if err := d2.Read.QueryRow(`SELECT title FROM works ORDER BY sort_title, id LIMIT 1`).Scan(&firstTitle); err != nil {
		t.Fatal(err)
	}
	if firstTitle != "Ёлка" {
		t.Fatalf("first title %q, empty key still sorts first", firstTitle)
	}
}

func TestHasPendingMigrations(t *testing.T) {
	missing, err := HasPendingMigrations(context.Background(), filepath.Join(t.TempDir(), "no.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if missing {
		t.Fatal("missing file is not a pending migration")
	}

	d := openTest(t)
	path := d.Path()
	_ = d.Close()
	got, err := HasPendingMigrations(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Fatal("fully migrated catalog must not be pending")
	}

	initial, err := migrations.FS.ReadFile("001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	opt := Options{
		Path:       filepath.Join(dir, "catalog.sqlite"),
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
		Migrations: fstest.MapFS{"001_initial.sql": &fstest.MapFile{Data: initial}},
	}
	d1, err := Open(context.Background(), opt)
	if err != nil {
		t.Fatal(err)
	}
	onePath := d1.Path()
	_ = d1.Close()
	pending, err := HasPendingMigrations(context.Background(), onePath)
	if err != nil {
		t.Fatal(err)
	}
	if !pending {
		t.Fatal("catalog with only 001 applied must be pending")
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

func TestOccupiedCatalogIsRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.sqlite")
	raw, err := sql.Open("sqlite", fileDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`CREATE TABLE works (id INTEGER PRIMARY KEY, key TEXT)`); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()

	_, err = Open(context.Background(), Options{
		Path:       path,
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
	})
	if err == nil {
		t.Fatal("expected incompatible catalog")
	}
	if apperr.As(err).Code != apperr.CodeDBIncompatible {
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
	var ok int64
	if err := d.Read.QueryRow(`SELECT text_plausible('Обычный текст аннотации.')`).Scan(&ok); err != nil {
		t.Fatal(err)
	}
	if ok != 1 {
		t.Fatalf("plausible = %d", ok)
	}
	if err := d.Read.QueryRow(`SELECT text_plausible('╔══╗ © ¤ ░▒▓│┤')`).Scan(&ok); err != nil {
		t.Fatal(err)
	}
	if ok != 0 {
		t.Fatalf("implausible = %d", ok)
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
		path, err := d.Backup(context.Background(), BackupReasonINPX)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, path)
	}
	matches, err := filepath.Glob(filepath.Join(d.backupsDir, "catalog-inpx-*.sqlite"))
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

func TestBackupRetentionIsPerReason(t *testing.T) {
	t0 := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	clock := t0
	d := openTest(t, func(o *Options) {
		o.Now = func() time.Time { return clock }
	})
	if _, err := d.Write.Exec(`INSERT INTO works(work_key, title, sort_title, authors_text, created_at, updated_at)
		VALUES ('k', 'T', 't', 'A', 'now', 'now')`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		clock = t0.Add(time.Duration(i) * time.Second)
		if _, err := d.Backup(context.Background(), BackupReasonPersonal); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 5; i++ {
		clock = t0.Add(time.Duration(5+i) * time.Second)
		if _, err := d.Backup(context.Background(), BackupReasonMigration); err != nil {
			t.Fatal(err)
		}
	}
	personal, err := filepath.Glob(filepath.Join(d.backupsDir, "catalog-personal-*.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	migration, err := filepath.Glob(filepath.Join(d.backupsDir, "catalog-migration-*.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if len(personal) != 3 {
		t.Fatalf("personal kept %d: %v", len(personal), personal)
	}
	if len(migration) != 3 {
		t.Fatalf("migration kept %d: %v", len(migration), migration)
	}
}

func TestBackupRemovesLegacyNames(t *testing.T) {
	t0 := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	d := openTest(t, func(o *Options) {
		o.Now = func() time.Time { return t0 }
	})
	if _, err := d.Write.Exec(`INSERT INTO works(work_key, title, sort_title, authors_text, created_at, updated_at)
		VALUES ('k', 'T', 't', 'A', 'now', 'now')`); err != nil {
		t.Fatal(err)
	}
	legacy := []string{
		filepath.Join(d.backupsDir, "catalog-20260915-120000.sqlite"),
		filepath.Join(d.backupsDir, "catalog-20260916-180000.sqlite"),
	}
	for _, p := range legacy {
		if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.Backup(context.Background(), BackupReasonINPX); err != nil {
		t.Fatal(err)
	}
	for _, p := range legacy {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("legacy %s still there: %v", p, err)
		}
	}
	kept, err := filepath.Glob(filepath.Join(d.backupsDir, "catalog-inpx-*.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 1 {
		t.Fatalf("inpx kept %d: %v", len(kept), kept)
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

func TestEditionLookupPlanUsesWorkIndex(t *testing.T) {
	d := openTest(t)
	tx, err := d.Write.Begin()
	if err != nil {
		t.Fatal(err)
	}
	const n = 4000
	insW, err := tx.Prepare(`INSERT INTO works(work_key, title, sort_title, authors_text, created_at, updated_at) VALUES (?, ?, 't', 'A', 'now', 'now')`)
	if err != nil {
		t.Fatal(err)
	}
	insE, err := tx.Prepare(`INSERT INTO editions(libid, work_id, archive_name, file_name, series, is_active, is_deleted) VALUES (?, ?, 'a.zip', 'f', 's', 1, 0)`)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= n; i++ {
		if _, err := insW.Exec(fmt.Sprintf("k%04d", i), "T"); err != nil {
			t.Fatal(err)
		}
		if _, err := insE.Exec(fmt.Sprintf("l%04d", i), i); err != nil {
			t.Fatal(err)
		}
	}
	_ = insW.Close()
	_ = insE.Close()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := Analyze(context.Background(), d.Write); err != nil {
		t.Fatal(err)
	}
	var one int
	if err := d.Write.QueryRow(`SELECT 1 FROM sqlite_stat1 LIMIT 1`).Scan(&one); err != nil {
		t.Fatal(err)
	}
	plan := explainQueryPlan(t, d.Write, `SELECT group_concat(DISTINCT e.series)
  FROM editions e
 WHERE e.work_id = 1
   AND e.is_active = 1 AND e.is_deleted = 0
   AND e.series IS NOT NULL AND trim(e.series) != ''`)
	if strings.Contains(plan, "idx_editions_active") {
		t.Fatalf("hot series lookup walked idx_editions_active:\n%s", plan)
	}
	if !strings.Contains(plan, "idx_editions_work") {
		t.Fatalf("expected idx_editions_work, got:\n%s", plan)
	}
}

func explainQueryPlan(t *testing.T, db *sql.DB, query string) string {
	t.Helper()
	rows, err := db.Query("EXPLAIN QUERY PLAN " + query)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var b strings.Builder
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatal(err)
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestOpenRecoversSearchIndexInBackground(t *testing.T) {
	d := openTest(t)
	if _, err := d.Write.Exec(`INSERT INTO works(work_key, title, sort_title, authors_text, created_at, updated_at)
		VALUES ('k', 'Ёлка', 'елка', 'Автор', 'now', 'now')`); err != nil {
		t.Fatal(err)
	}
	if err := SetFTSDirty(context.Background(), d.Write, true); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write.Exec(`DROP TRIGGER IF EXISTS works_fts_au`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write.Exec(`DROP TRIGGER IF EXISTS works_fts_ad`); err != nil {
		t.Fatal(err)
	}
	path := d.Path()
	backups := d.backupsDir
	_ = d.Close()

	d2, err := Open(context.Background(), Options{
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
		t.Fatal("expected search index ready")
	}
	var n int
	if err := d2.Read.QueryRow(`SELECT count(*) FROM works_fts`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("works_fts rows = %d", n)
	}
}

func TestRebuildWorksFTSFillsInBatches(t *testing.T) {
	d := openTest(t)
	tx, err := d.Write.Begin()
	if err != nil {
		t.Fatal(err)
	}
	ins, err := tx.Prepare(`INSERT INTO works(work_key, title, sort_title, authors_text, created_at, updated_at) VALUES (?, 'T', 't', 'A', 'now', 'now')`)
	if err != nil {
		t.Fatal(err)
	}
	const n = worksFTSBatch + 3
	for i := 1; i <= n; i++ {
		if _, err := ins.Exec(fmt.Sprintf("k%05d", i)); err != nil {
			t.Fatal(err)
		}
	}
	_ = ins.Close()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	conn, err := d.Write.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if err := RebuildWorksFTS(context.Background(), conn); err != nil {
		t.Fatal(err)
	}
	var got int
	if err := d.Read.QueryRow(`SELECT count(*) FROM works_fts`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != n {
		t.Fatalf("works_fts rows = %d, want %d", got, n)
	}
}

func TestWALCheckpointBusyWithLiveReader(t *testing.T) {
	d := openTest(t)
	if _, err := d.Write.Exec(`INSERT INTO app_meta(key, value) VALUES ('k', 'v')`); err != nil {
		t.Fatal(err)
	}
	rows, err := d.Read.Query(`SELECT value FROM app_meta`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var busy, logFrames, checkpointed int
	if err := d.Write.QueryRow(`PRAGMA wal_checkpoint(TRUNCATE)`).Scan(&busy, &logFrames, &checkpointed); err != nil {
		t.Fatal(err)
	}
	if busy == 0 {
		t.Log("checkpoint was not busy with a live reader")
	}
}

func TestCloseTruncatesWALAfterReadersClosed(t *testing.T) {
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
	if _, err := d.Write.Exec(`INSERT INTO app_meta(key, value) VALUES ('k', 'v')`); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	wal := d.Path() + "-wal"
	st, statErr := os.Stat(wal)
	if statErr == nil && st.Size() > 0 {
		t.Fatalf("wal still %d bytes after Close", st.Size())
	}
}
