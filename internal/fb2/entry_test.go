package fb2

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestValidEntryName(t *testing.T) {
	ok := []string{"110119.fb2", "dir/book.fb2", "book..fb2"}
	bad := []string{"../book.fb2", "foo/../book.fb2", "/abs.fb2", `C:\book.fb2`, `\\server\book.fb2`, ""}
	for _, n := range ok {
		if !ValidEntryName(n) {
			t.Fatalf("rejected valid %q", n)
		}
	}
	for _, n := range bad {
		if ValidEntryName(n) {
			t.Fatalf("accepted %q", n)
		}
	}
}

func zipWith(t *testing.T, names map[string][]byte) *zip.Reader {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range names {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestFindEntryFallbacks(t *testing.T) {
	doc := fb2Doc("utf-8", "l", "image/jpeg", jpegBytes, "n")
	zr := zipWith(t, map[string][]byte{
		"110119.FB2":  doc,
		"../evil.fb2": []byte("nope"),
	})
	got := FindEntry(zr.File, "110119", "fb2")
	if got == nil || got.Name != "110119.FB2" {
		t.Fatalf("got %+v", got)
	}
	if FindEntry(zr.File, "missing", "fb2") != nil {
		t.Fatal("missing entry found")
	}
	if FindEntry(zr.File, "evil", "fb2") != nil {
		t.Fatal("traversal name must not be selected")
	}
}

func TestFindEntryByBaseName(t *testing.T) {
	doc := fb2Doc("utf-8", "href", "image/jpeg", jpegBytes, "n")
	zr := zipWith(t, map[string][]byte{"nested/110119.fb2": doc})
	got := FindEntry(zr.File, "110119", "epub")
	if got == nil || got.Name != "nested/110119.fb2" {
		t.Fatalf("fallback by stem: %+v", got)
	}
}

func TestParseZipRoundtrip(t *testing.T) {
	doc := fb2Doc("utf-8", "l", "image/jpeg", pngBytes, "Note")
	zr := zipWith(t, map[string][]byte{"110119.fb2": doc})
	got, err := ParseZip(zr, "110119", "fb2")
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasCover || got.CoverKind != ImagePNG || got.Annotation != "Note" {
		t.Fatalf("%+v", got)
	}
}
