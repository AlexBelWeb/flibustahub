package platform

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShowInFolderQuotesSpacesAndRejectsMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "My Books")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "War and Peace.fb2")
	if err := os.WriteFile(path, []byte("fb2"), 0o644); err != nil {
		t.Fatal(err)
	}

	prev := revealPath
	t.Cleanup(func() { revealPath = prev })
	var got string
	revealPath = func(abs string) error {
		got = abs
		return nil
	}

	if err := ShowInFolder(path); err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("reveal was not called")
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("path is not absolute: %q", got)
	}
	if !strings.Contains(got, " ") {
		t.Fatalf("expected a path with spaces, got %q", got)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	got = ""
	err := ShowInFolder(path)
	if err == nil {
		t.Fatal("deleted file must fail before the file manager is launched")
	}
	if !errors.Is(err, os.ErrNotExist) && !os.IsNotExist(err) {
		t.Fatalf("want not-exist, got %v", err)
	}
	if got != "" {
		t.Fatalf("file manager must not be launched for a missing file, got %q", got)
	}
}

func TestOpenDirRejectsMissingAndFile(t *testing.T) {
	prev := openDir
	t.Cleanup(func() { openDir = prev })
	var got string
	openDir = func(abs string) error {
		got = abs
		return nil
	}

	dir := filepath.Join(t.TempDir(), "My Downloads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := OpenDir(dir); err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(got) || !strings.Contains(got, " ") {
		t.Fatalf("expected absolute path with spaces, got %q", got)
	}

	got = ""
	missing := filepath.Join(dir, "gone")
	if err := OpenDir(missing); err == nil {
		t.Fatal("missing directory must fail")
	}
	if got != "" {
		t.Fatal("file manager must not be launched for a missing directory")
	}

	file := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = ""
	if err := OpenDir(file); err == nil {
		t.Fatal("a file is not a directory")
	}
	if got != "" {
		t.Fatal("file manager must not be launched for a file")
	}
}
