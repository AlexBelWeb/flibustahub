package covers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alexbelweb/flibustahub/internal/fb2"
)

func TestLookupAndWrite(t *testing.T) {
	dir := t.TempDir()
	if hit := Lookup(dir, 1); hit.Found {
		t.Fatal("empty cache hit")
	}
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0}
	if err := WriteCover(dir, 1, jpeg, fb2.ImageJPEG); err != nil {
		t.Fatal(err)
	}
	hit := Lookup(dir, 1)
	if !hit.Found || hit.None || hit.MIME != "image/jpeg" {
		t.Fatalf("%+v", hit)
	}
	if err := WriteNone(dir, 1); err != nil {
		t.Fatal(err)
	}
	hit = Lookup(dir, 1)
	if !hit.Found || !hit.None {
		t.Fatal("expected .none")
	}
	if _, err := os.Stat(filepath.Join(dir, "1.jpg")); !os.IsNotExist(err) {
		t.Fatal("jpg should be replaced by none")
	}
}

func TestClearDir(t *testing.T) {
	dir := t.TempDir()
	if err := WriteNone(dir, 2); err != nil {
		t.Fatal(err)
	}
	if err := ClearDir(dir); err != nil {
		t.Fatal(err)
	}
	if hit := Lookup(dir, 2); hit.Found {
		t.Fatal("cleared cache still hits")
	}
}

func TestClearNoneMarkersLeavesImages(t *testing.T) {
	dir := t.TempDir()
	jpg := filepath.Join(dir, "7.jpg")
	none := filepath.Join(dir, "7.none")
	if err := os.WriteFile(jpg, []byte{0xff, 0xd8, 0xff, 0xe0}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(none, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ClearNoneMarkers(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(none); !os.IsNotExist(err) {
		t.Fatal(".none must be removed after a successful import")
	}
	if _, err := os.Stat(jpg); err != nil {
		t.Fatal("image must stay")
	}
}
