package inpx

import (
	"testing"

	"github.com/alexbelweb/flibustahub/internal/inpx/testdata"
)

func mustParse(t *testing.T, fields [14]string) Record {
	t.Helper()
	rec, skip := ParseLine(testdata.Record(fields, true), "d.fb2-000001-000100.zip")
	if skip != SkipNone {
		t.Fatalf("skip %v", skip)
	}
	return rec
}

func TestParseGromovFields(t *testing.T) {
	rec := mustParse(t, testdata.Gromov())
	if rec.LibID != "110119" || rec.File != "110119" {
		t.Fatalf("libid/file = %s/%s", rec.LibID, rec.File)
	}
	if rec.Size == nil || *rec.Size != 1230745 {
		t.Fatalf("size = %v", rec.Size)
	}
	if rec.IsDeleted {
		t.Fatal("DEL 0 must not be deleted")
	}
	if rec.Title != "Первый из могикан" || rec.Series != "Мир матриархата" || rec.SeriesNo != "2" {
		t.Fatalf("title/series = %q %q %q", rec.Title, rec.Series, rec.SeriesNo)
	}
	if rec.Lang != "ru" || rec.Ext != "fb2" || rec.Date != "2008-07-05" {
		t.Fatalf("lang/ext/date = %q %q %q", rec.Lang, rec.Ext, rec.Date)
	}
	if rec.LibRate == nil || *rec.LibRate != 3 {
		t.Fatalf("librate = %v", rec.LibRate)
	}
	if len(rec.Authors) != 1 || rec.Authors[0].Last != "Громов" || rec.Authors[0].First != "Александр" || rec.Authors[0].Middle != "Николаевич" {
		t.Fatalf("authors = %+v", rec.Authors)
	}
	if len(rec.Genres) != 1 || rec.Genres[0] != "sf_social" {
		t.Fatalf("genres = %v", rec.Genres)
	}
}

func TestParseHangingColonAndCRLF(t *testing.T) {
	raw := testdata.Record(testdata.Gromov(), true)
	if raw[len(raw)-1] != '\n' || raw[len(raw)-2] != '\r' {
		t.Fatal("fixture CRLF")
	}
	rec, skip := ParseLine(raw, "a.zip")
	if skip != SkipNone {
		t.Fatal(skip)
	}
	if rec.Keywords != "" {
		t.Fatalf("CR must not leak into keywords: %q", rec.Keywords)
	}
}

func TestParseNoAuthorsFallback(t *testing.T) {
	rec := mustParse(t, testdata.NoAuthors())
	if len(rec.Authors) != 1 || rec.Authors[0].Last != "Неизвестен" || rec.Authors[0].First != "Автор" {
		t.Fatalf("%+v", rec.Authors)
	}
}

func TestParseDeletedAndDirtySerno(t *testing.T) {
	if !mustParse(t, testdata.Deleted()).IsDeleted {
		t.Fatal("DEL=1")
	}
	if mustParse(t, testdata.SernoRange()).SeriesNo != "1-2" {
		t.Fatal("serno 1-2")
	}
	if mustParse(t, testdata.SernoQuestion()).SeriesNo != "?" {
		t.Fatal("serno ?")
	}
}

func TestParseRejectsNoLibIDAndMalformed(t *testing.T) {
	if _, skip := ParseLine(testdata.Record(testdata.NoLibID(), true), "a.zip"); skip != SkipNoLibID {
		t.Fatalf("skip = %v", skip)
	}
	if _, skip := ParseLine(testdata.BrokenRecord("a", "b", "c"), "a.zip"); skip != SkipMalformed {
		t.Fatalf("skip = %v", skip)
	}
}

func TestParseLastRecordWithoutLF(t *testing.T) {
	rec, skip := ParseLine(testdata.Record(testdata.NoLFTail(), false), "a.zip")
	if skip != SkipNone {
		t.Fatal(skip)
	}
	if rec.Title != "Последняя без LF" {
		t.Fatalf("%q", rec.Title)
	}
}

func TestParseManyAuthors(t *testing.T) {
	rec := mustParse(t, testdata.ManyAuthors())
	if len(rec.Authors) != 30 {
		t.Fatalf("got %d authors", len(rec.Authors))
	}
}
