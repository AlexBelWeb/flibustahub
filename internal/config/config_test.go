package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	store, err := Load(dir, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	live := store.Live()
	if live.Theme != ThemeSystem {
		t.Fatalf("theme = %q", live.Theme)
	}
	if live.LibraryRoot != "" {
		t.Fatalf("libraryRoot must start empty, got %q", live.LibraryRoot)
	}
	if live.OPDSPort != DefaultOPDSPort {
		t.Fatalf("port = %d", live.OPDSPort)
	}
	if live.DataDir != dir {
		t.Fatalf("dataDir = %q, want %q", live.DataDir, dir)
	}
}

func TestLoadBrokenFileRenamed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := Load(dir, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".broken"); err != nil {
		t.Fatalf("expected broken sidecar: %v", err)
	}
	if store.Live().Theme != ThemeSystem {
		t.Fatal("expected defaults after damaged config")
	}
}

func TestUpdateDoesNotPersistEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FLIBUSTAHUB_THEME", ThemeLight)
	store, err := Load(dir, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if store.Live().Theme != ThemeLight {
		t.Fatalf("live theme = %q", store.Live().Theme)
	}
	if err := store.Update(func(f *File) { f.Locale = "en" }); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	var disk File
	if err := json.Unmarshal(raw, &disk); err != nil {
		t.Fatal(err)
	}
	if disk.Locale != "en" {
		t.Fatalf("saved locale = %q", disk.Locale)
	}
	if disk.Theme == ThemeLight {
		t.Fatal("env overlay must not be written to config.json")
	}
}

func TestAtomicSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := Load(dir, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(f *File) {
		f.Locale = "ru"
		f.Window = WindowState{X: 10, Y: 20, Width: 1280, Height: 840}
	}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load(dir, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	got := reloaded.Live()
	if got.Locale != "ru" || got.Window.Width != 1280 {
		t.Fatalf("reloaded = %+v", got)
	}
	if got.CatalogView != CatalogViewTable {
		t.Fatalf("default catalog view = %q", got.CatalogView)
	}
	if got.SidebarCollapsed {
		t.Fatal("sidebar must start expanded")
	}
}

func TestSidebarAndCatalogViewPersist(t *testing.T) {
	dir := t.TempDir()
	store, err := Load(dir, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(f *File) {
		f.SidebarCollapsed = true
		f.CatalogView = CatalogViewTile
	}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load(dir, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	got := reloaded.Live()
	if !got.SidebarCollapsed || got.CatalogView != CatalogViewTile {
		t.Fatalf("reloaded = %+v", got)
	}
}
