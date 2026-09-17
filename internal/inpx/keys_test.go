package inpx

import (
	"strings"
	"testing"

	"github.com/alexbelweb/flibustahub/internal/inpx/testdata"
)

func TestWorkKeyIndependentOfAuthorOrder(t *testing.T) {
	a := mustParse(t, testdata.CoauthorsOrderA())
	b := mustParse(t, testdata.CoauthorsOrderB())
	ka, pa := WorkKey(a.Authors, a.Title)
	kb, pb := WorkKey(b.Authors, b.Title)
	if ka != kb {
		t.Fatalf("keys differ:\n%s\n%s\npreimages:\n%s\n%s", ka, kb, pa, pb)
	}
	if len(ka) != 64 {
		t.Fatalf("key length %d", len(ka))
	}
	if a.Authors[0].Last == b.Authors[0].Last {
		t.Fatal("fixture authors must be listed in different order")
	}
	if AuthorsText(a.Authors) == AuthorsText(b.Authors) {
		t.Fatal("display order must follow AUTHOR field, not the key sort")
	}
}

func TestWorkKeyManyAuthorsIsHexSHA256(t *testing.T) {
	rec := mustParse(t, testdata.ManyAuthors())
	key, pre := WorkKey(rec.Authors, rec.Title)
	if len(key) != 64 {
		t.Fatalf("len=%d key=%s", len(key), key)
	}
	if strings.Count(pre, "|") != 29 {
		t.Fatalf("expected 30 author keys joined, preimage pipes=%d", strings.Count(pre, "|"))
	}
}

func TestWorkKeyFixedVector(t *testing.T) {
	rec := mustParse(t, testdata.Gromov())
	key, pre := WorkKey(rec.Authors, rec.Title)
	const wantPre = "громов,александр,николаевич\tпервый из могикан"
	if pre != wantPre {
		t.Fatalf("preimage %q want %q", pre, wantPre)
	}
	const wantKey = "0b8530d073bdb10c81122dbf4d3c1bee535e99cb62b1736881be35e76d987da7"
	if key != wantKey {
		t.Fatalf("work_key %s want %s", key, wantKey)
	}
}

func TestUnknownAuthorKey(t *testing.T) {
	rec := mustParse(t, testdata.NoAuthors())
	_, pre := WorkKey(rec.Authors, rec.Title)
	if !strings.HasPrefix(pre, "неизвестен,автор,") {
		t.Fatalf("preimage %q", pre)
	}
}

func TestSortTitleYoAndQuotes(t *testing.T) {
	got := []string{
		SortTitle("«Ёлка»"),
		SortTitle("абрикос"),
		SortTitle("Яблоко"),
		SortTitle("елка"),
	}
	want := []string{"елка", "абрикос", "яблоко", "елка"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%d: %q want %q", i, got[i], want[i])
		}
	}
	if SortTitle("«Ёлка»") != SortTitle("елка") {
		t.Fatalf("quotes/yo must fold: %q vs %q", SortTitle("«Ёлка»"), SortTitle("елка"))
	}
	if SortTitle("абрикос") >= SortTitle("елка") {
		t.Fatalf("абрикос should sort before елка: %q %q", SortTitle("абрикос"), SortTitle("елка"))
	}
}

func TestAuthorSortName(t *testing.T) {
	a := Author{Last: "Ёлкин", First: "Ёж", Middle: ""}
	if a.SortName() != "елкин еж" {
		t.Fatalf("%q", a.SortName())
	}
	if a.DisplayName() != "Ёлкин Ёж" {
		t.Fatalf("display %q", a.DisplayName())
	}
}

func TestArchiveRecency(t *testing.T) {
	if ArchiveRecency("f.fb2-200864-203580.zip") != 203580 {
		t.Fatal(ArchiveRecency("f.fb2-200864-203580.zip"))
	}
	if ArchiveRecency(testdata.ArchiveLow) != 100 {
		t.Fatal(ArchiveRecency(testdata.ArchiveLow))
	}
	if ArchiveRecency(testdata.ArchiveHigh) != 300 {
		t.Fatal(ArchiveRecency(testdata.ArchiveHigh))
	}
}

func TestShouldReplaceBranches(t *testing.T) {
	low := testdata.ArchiveLow
	high := testdata.ArchiveHigh
	cases := []struct {
		name      string
		existing  EditionState
		incoming  Record
		replace   bool
	}{
		{
			name:     "deleted to live",
			existing: EditionState{IsDeleted: true, ArchiveName: high},
			incoming: Record{IsDeleted: false, ArchiveName: low},
			replace:  true,
		},
		{
			name:     "live to deleted",
			existing: EditionState{IsDeleted: false, ArchiveName: low},
			incoming: Record{IsDeleted: true, ArchiveName: high},
			replace:  false,
		},
		{
			name:     "same live newer archive",
			existing: EditionState{IsDeleted: false, ArchiveName: low},
			incoming: Record{IsDeleted: false, ArchiveName: high},
			replace:  true,
		},
		{
			name:     "same live older archive",
			existing: EditionState{IsDeleted: false, ArchiveName: high},
			incoming: Record{IsDeleted: false, ArchiveName: low},
			replace:  false,
		},
		{
			name:     "equal recency replaces",
			existing: EditionState{IsDeleted: false, ArchiveName: high},
			incoming: Record{IsDeleted: false, ArchiveName: high},
			replace:  true,
		},
		{
			name:     "both deleted newer archive",
			existing: EditionState{IsDeleted: true, ArchiveName: low},
			incoming: Record{IsDeleted: true, ArchiveName: high},
			replace:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ShouldReplace(tc.existing, tc.incoming); got != tc.replace {
				t.Fatalf("got %v want %v", got, tc.replace)
			}
		})
	}
}
