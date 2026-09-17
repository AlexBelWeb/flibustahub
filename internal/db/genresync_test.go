package db

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexbelweb/flibustahub/internal/data"
)

func TestSyncGenreNamesUpdatesExistingRowsOnly(t *testing.T) {
	t.Cleanup(func() { data.Load("", slog.New(slog.DiscardHandler)) })
	d := openTest(t)
	if _, err := d.Write.Exec(`INSERT INTO genres(code, name_ru) VALUES ('sf_social', 'старое')`); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, data.LocalFileName), []byte(`{"sf_social":"новое имя","never_seen":"не должен попасть"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	data.Load(dir, slog.New(slog.DiscardHandler))
	if err := d.SyncGenreNames(context.Background()); err != nil {
		t.Fatal(err)
	}
	var name string
	if err := d.Read.QueryRow(`SELECT name_ru FROM genres WHERE code='sf_social'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "новое имя" {
		t.Fatalf("name_ru=%q", name)
	}
	var n int
	if err := d.Read.QueryRow(`SELECT count(*) FROM genres WHERE code='never_seen'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("dictionary must not insert unseen codes")
	}
}
