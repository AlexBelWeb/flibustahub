package testdata

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestGromovHasTrailingEOTAndCRLF(t *testing.T) {
	raw := Record(Gromov(), true)
	if !bytes.Contains(raw, []byte{eot}) {
		t.Fatal("missing field separator 0x04")
	}
	if !bytes.HasSuffix(raw, []byte{eot, '\r', '\n'}) {
		t.Fatalf("want trailing 0x04 CRLF, got %q", raw[len(raw)-8:])
	}
	line := bytes.TrimSuffix(raw, []byte{'\r', '\n'})
	parts := bytes.Split(line, []byte{eot})
	if len(parts) != 15 {
		t.Fatalf("segments = %d, want 15 (14 fields + empty tail)", len(parts))
	}
	if len(parts[14]) != 0 {
		t.Fatalf("tail segment %q", parts[14])
	}
	if string(parts[5]) != "110119" || string(parts[6]) != "1230745" || string(parts[7]) != "110119" || string(parts[8]) != "0" {
		t.Fatalf("FILE/SIZE/LIBID/DEL = %q %q %q %q", parts[5], parts[6], parts[7], parts[8])
	}
}

func TestBrokenRecordHasFewerThan14Fields(t *testing.T) {
	raw := BrokenRecord("битая", "запись", "мало")
	line := bytes.TrimSuffix(raw, []byte{'\r', '\n'})
	parts := bytes.Split(line, []byte{eot})
	if len(parts) >= 14 {
		t.Fatalf("broken record has %d segments, want < 14 fields", len(parts))
	}
}

func TestLastRecordMayOmitLF(t *testing.T) {
	raw := Record(noLFTail(), false)
	if bytes.Contains(raw, []byte{'\n'}) {
		t.Fatal("fixture must omit LF")
	}
	if !bytes.HasSuffix(raw, []byte{eot}) {
		t.Fatal("still needs trailing 0x04")
	}
}

func TestCatalogINPXZipOrderIsNotNameOrder(t *testing.T) {
	r, err := zip.NewReader(bytes.NewReader(CatalogINPX()), int64(len(CatalogINPX())))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range r.File {
		if len(f.Name) > 4 && f.Name[len(f.Name)-4:] == ".inp" {
			names = append(names, f.Name)
		}
	}
	if len(names) < 3 {
		t.Fatalf("inp files = %v", names)
	}
	if names[0] < names[1] && names[1] < names[2] {
		t.Fatalf("zip already in name order %v; fixture must shuffle", names)
	}
}

func TestVersionInfoBOMAndCollectionFallback(t *testing.T) {
	raw := CatalogINPX()
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	var sawBOM bool
	for _, f := range zr.File {
		if f.Name != "version.info" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b := make([]byte, 16)
		n, _ := rc.Read(b)
		_ = rc.Close()
		if n < 3 || b[0] != 0xEF || b[1] != 0xBB || b[2] != 0xBF {
			t.Fatalf("version.info must start with UTF-8 BOM, got %x", b[:n])
		}
		sawBOM = true
	}
	if !sawBOM {
		t.Fatal("version.info missing")
	}
	fb := CollectionOnlyINPX()
	fr, err := zip.NewReader(bytes.NewReader(fb), int64(len(fb)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fr.File {
		if f.Name == "version.info" {
			t.Fatal("collection-only fixture must not include version.info")
		}
	}
}
