package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOptimizeRestoresTempStore(t *testing.T) {
	d := openTest(t)
	if err := d.Optimize(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := readPragma(context.Background(), d.Write, "temp_store")
	if err != nil {
		t.Fatal(err)
	}
	if got != "2" {
		t.Fatalf("temp_store after optimize = %q, want MEMORY (2)", got)
	}
}

func TestOpenSetsTempStoreDirectory(t *testing.T) {
	d := openTest(t)
	got, err := readPragma(context.Background(), d.Write, "temp_store_directory")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.ToSlash(filepath.Dir(d.path))
	if filepath.ToSlash(got) != want && !strings.EqualFold(filepath.ToSlash(got), want) {
		t.Fatalf("temp_store_directory = %q, want %q", got, want)
	}
}

func TestOpenRemovesOrphanEtilqs(t *testing.T) {
	dir := t.TempDir()
	orphan := filepath.Join(dir, "etilqs_leftover")
	if err := os.WriteFile(orphan, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(dir, "catalog.sqlite")
	d, err := Open(context.Background(), Options{
		Path:       keep,
		BackupsDir: filepath.Join(dir, "backups"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("orphan temp db still present: %v", err)
	}
}

func TestQuoteSQLStringEscapesQuotes(t *testing.T) {
	if got := quoteSQLString(`O'Reilly`); got != `'O''Reilly'` {
		t.Fatalf("%s", got)
	}
}

func TestOptimizePinsThenRestoresOnSameConn(t *testing.T) {
	d := openTest(t)
	conn, err := d.Write.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.ExecContext(context.Background(), "PRAGMA temp_store=FILE"); err != nil {
		t.Fatal(err)
	}
	var store any
	if err := conn.QueryRowContext(context.Background(), "PRAGMA temp_store").Scan(&store); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(store) != "1" {
		t.Fatalf("temp_store = %v, want FILE (1)", store)
	}
	if err := RestoreWorkPragmas(context.Background(), conn); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRowContext(context.Background(), "PRAGMA temp_store").Scan(&store); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(store) != "2" {
		t.Fatalf("temp_store after restore = %v, want MEMORY (2)", store)
	}
}
