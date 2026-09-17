package inpx

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexbelweb/flibustahub/internal/inpx/testdata"
)

func TestWalkCatalogINPX(t *testing.T) {
	raw := testdata.CatalogINPX()
	var recs []Record
	meta, err := WalkRecords(bytes.NewReader(raw), int64(len(raw)), func(r Record) error {
		recs = append(recs, r)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if meta.Version != "20260901" {
		t.Fatalf("version %q", meta.Version)
	}
	if meta.SkippedMalformed != 1 {
		t.Fatalf("malformed = %d", meta.SkippedMalformed)
	}
	if meta.SkippedNoLibID != 1 {
		t.Fatalf("no libid = %d", meta.SkippedNoLibID)
	}
	var sawNoLF, sawMissing, sawHighShared bool
	var order []string
	seenArch := map[string]int{}
	for _, r := range recs {
		order = append(order, r.ArchiveName+"/"+r.LibID)
		seenArch[r.ArchiveName]++
		if r.Title == "Последняя без LF" {
			sawNoLF = true
		}
		if r.LibID == "800001" {
			sawMissing = true
			if r.ArchiveName != testdata.ArchiveMissing {
				t.Fatalf("missing archive name %s", r.ArchiveName)
			}
		}
		if r.LibID == "900001" && r.ArchiveName == testdata.ArchiveHigh {
			sawHighShared = true
		}
	}
	if !sawNoLF {
		t.Fatal("record without LF was dropped")
	}
	if !sawMissing {
		t.Fatal("missing-archive record was dropped")
	}
	if !sawHighShared {
		t.Fatal("high-recency duplicate libid missing; zip order vs name order?")
	}
	lowIdx, highIdx := -1, -1
	for i, r := range recs {
		if r.ArchiveName == testdata.ArchiveLow && lowIdx < 0 {
			lowIdx = i
		}
		if r.ArchiveName == testdata.ArchiveHigh && highIdx < 0 {
			highIdx = i
		}
	}
	if lowIdx < 0 || highIdx < 0 || lowIdx > highIdx {
		t.Fatalf("inp files must be processed by name: first low=%d high=%d recs=%v", lowIdx, highIdx, order)
	}
	if meta.Encodings.UTF8 != 0 || meta.Encodings.CP1251 != 3 {
		t.Fatalf("catalog fixture encodings utf8=%d cp1251=%d", meta.Encodings.UTF8, meta.Encodings.CP1251)
	}
	if meta.Encodings.VersionInfo != string(EncodingUTF8) {
		t.Fatalf("version.info encoding %q", meta.Encodings.VersionInfo)
	}
	if meta.Encodings.CollectionInfo != string(EncodingUTF8) {
		t.Fatalf("collection.info encoding %q", meta.Encodings.CollectionInfo)
	}
}

func TestCollectionInfoFallback(t *testing.T) {
	raw := testdata.CollectionOnlyINPX()
	meta, err := WalkRecords(bytes.NewReader(raw), int64(len(raw)), func(Record) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if meta.Version != "from_collection_20250101" {
		t.Fatalf("version %q", meta.Version)
	}
}

func TestUTF8CollectionInfo(t *testing.T) {
	raw := testdata.UTF8CollectionINPX()
	var title string
	meta, err := WalkRecords(bytes.NewReader(raw), int64(len(raw)), func(r Record) error {
		title = r.Title
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if meta.Version != "from_collection_utf8_20250101" {
		t.Fatalf("version %q", meta.Version)
	}
	if meta.Encodings.CollectionInfo != string(EncodingUTF8) || meta.Encodings.UTF8 != 1 {
		t.Fatalf("encodings %+v", meta.Encodings)
	}
	if title != "Первый из могикан" {
		t.Fatalf("title %q", title)
	}
}

func TestCP1251CollectionInfo(t *testing.T) {
	raw := testdata.CP1251CollectionINPX()
	meta, err := WalkRecords(bytes.NewReader(raw), int64(len(raw)), func(Record) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if meta.Version != "from_collection_cp1251_20250101" {
		t.Fatalf("version %q", meta.Version)
	}
	if meta.Encodings.CollectionInfo != string(EncodingCP1251) {
		t.Fatalf("collection encoding %q", meta.Encodings.CollectionInfo)
	}
}

func TestCP1251VersionInfo(t *testing.T) {
	raw := testdata.CP1251VersionINPX()
	meta, err := WalkRecords(bytes.NewReader(raw), int64(len(raw)), func(Record) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if meta.Version != "20260901" {
		t.Fatalf("version %q", meta.Version)
	}
	if meta.Encodings.VersionInfo != string(EncodingCP1251) {
		t.Fatalf("version encoding %q", meta.Encodings.VersionInfo)
	}
}

func TestMixedInpEncodings(t *testing.T) {
	raw := testdata.MixedEncodingINPX()
	got := map[string]Record{}
	meta, err := WalkRecords(bytes.NewReader(raw), int64(len(raw)), func(r Record) error {
		got[r.LibID] = r
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if meta.Encodings.UTF8 != 1 || meta.Encodings.CP1251 != 1 {
		t.Fatalf("mixed encodings %+v", meta.Encodings)
	}
	if meta.Encodings.VersionInfo != string(EncodingUTF8) || meta.Encodings.CollectionInfo != string(EncodingCP1251) {
		t.Fatalf("info encodings %+v", meta.Encodings)
	}
	g := got["110119"]
	if g.Authors[0].Last != "Громов" || g.Title != "Первый из могикан" {
		t.Fatalf("utf-8 member last=%q title=%q", g.Authors[0].Last, g.Title)
	}
	y := got["930001"]
	if y.Title != "Ёлка" || y.Authors[0].Last != "Ёлкин" {
		t.Fatalf("cp1251 member title=%q last=%q", y.Title, y.Authors[0].Last)
	}
}

func TestFindINPXNonRecursive(t *testing.T) {
	dir := t.TempDir()
	if err := testdata.WriteLibraryRoot(dir); err != nil {
		t.Fatal(err)
	}
	got, err := FindINPX(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != testdata.DumpName {
		t.Fatalf("got %s", got)
	}
	if filepath.Dir(got) != dir {
		t.Fatalf("must not recurse into update: %s", got)
	}
}

func TestMissingArchives(t *testing.T) {
	dir := t.TempDir()
	if err := testdata.WriteLibraryRoot(dir); err != nil {
		t.Fatal(err)
	}
	miss := MissingArchives(dir, []string{testdata.ArchiveLow, testdata.ArchiveHigh, testdata.ArchiveMissing})
	if len(miss) != 1 || miss[0] != testdata.ArchiveMissing {
		t.Fatalf("%v", miss)
	}
}

func TestFindINPXExplicitPath(t *testing.T) {
	dir := t.TempDir()
	if err := testdata.WriteLibraryRoot(dir); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "custom.inpx")
	if err := testdata.WriteCollectionOnlyDump(dir, "custom.inpx"); err != nil {
		t.Fatal(err)
	}
	got, err := FindINPX(dir, want)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s", got)
	}
}

func TestFindINPXEmptyRoot(t *testing.T) {
	_, err := FindINPX("", "")
	if !os.IsNotExist(err) {
		t.Fatalf("err=%v", err)
	}
}
