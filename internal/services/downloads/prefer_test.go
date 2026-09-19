package downloads

import (
	"testing"
	"time"
)

func TestPreferredFormatThenDateThenID(t *testing.T) {
	eds := []Edition{
		{ID: 1, FileExt: "epub", AddedDate: "2026-01-02"},
		{ID: 2, FileExt: "fb2", AddedDate: "2026-01-01"},
		{ID: 3, FileExt: "pdf", AddedDate: "2026-02-01"},
	}
	got := Preferred(eds)
	if got.ID != 2 {
		t.Fatalf("want fb2 id=2, got %+v", got)
	}

	same := []Edition{
		{ID: 10, FileExt: "fb2", AddedDate: "2026-01-01"},
		{ID: 11, FileExt: "fb2", AddedDate: "2026-02-01"},
	}
	got = Preferred(same)
	if got.ID != 11 {
		t.Fatalf("want newer fb2 id=11, got %+v", got)
	}

	tie := []Edition{
		{ID: 4, FileExt: "fb2", AddedDate: "2026-01-01"},
		{ID: 8, FileExt: "fb2", AddedDate: "2026-01-01"},
	}
	got = Preferred(tie)
	if got.ID != 8 {
		t.Fatalf("want higher id, got %+v", got)
	}
}

func TestPreferredSkipsZeroID(t *testing.T) {
	got := Preferred([]Edition{{ID: 0, FileExt: "fb2"}, {ID: 5, FileExt: "epub"}})
	if got.ID != 5 {
		t.Fatalf("got %+v", got)
	}
}

func TestDumpNewerNumeric(t *testing.T) {
	if !DumpNewer("20261001", "20260901", "", time.Time{}, time.Time{}) {
		t.Fatal("newer numeric version")
	}
	if DumpNewer("20260901", "20260901", "", time.Now(), time.Time{}) {
		t.Fatal("same version must be silent even with newer mtime")
	}
	if DumpNewer("20260801", "20260901", "", time.Time{}, time.Time{}) {
		t.Fatal("older version")
	}
	if DumpNewer("20261001", "20260901", "20261001", time.Time{}, time.Time{}) {
		t.Fatal("dismissed version")
	}
	if !DumpNewer("20261101", "20260901", "20261001", time.Time{}, time.Time{}) {
		t.Fatal("newer than dismissed")
	}
}

func TestDumpNewerUnparseableUsesMtime(t *testing.T) {
	imported := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	older := imported.Add(-time.Hour)
	newer := imported.Add(time.Hour)
	if DumpNewer("abc", "abc", "", newer, imported) {
		t.Fatal("same unparseable version is silent")
	}
	if DumpNewer("xyz", "old", "", older, imported) {
		t.Fatal("older mtime")
	}
	if !DumpNewer("xyz", "old", "", newer, imported) {
		t.Fatal("newer mtime of unparseable version")
	}
	if DumpNewer("xyz", "old", "xyz", newer, imported) {
		t.Fatal("dismissed unparseable")
	}
}
