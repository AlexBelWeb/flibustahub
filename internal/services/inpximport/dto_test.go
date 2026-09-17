package inpximport

import (
	"encoding/json"
	"testing"

	"github.com/alexbelweb/flibustahub/internal/inpx"
)

func TestReportDTOCamelCase(t *testing.T) {
	rep := Report{
		ID:              3,
		Status:          StatusDone,
		INPXPath:        `/library/dump.inpx`,
		INPXVersion:     "20260901",
		RecordsSeen:     10,
		EditionsUpdated: 2,
		Notes: Notes{
			MissingArchives:      []string{"a.zip"},
			MissingArchivesTotal: 1,
			UnnamedGenres:        []string{"sf_writing"},
			UnnamedGenresTotal:   1,
			SkippedNoLibID:       4,
			GenreNamesMapped:     8,
			Encodings: inpx.EncodingStats{
				UTF8:        219,
				CP1251:      0,
				VersionInfo: "utf-8",
			},
			PhasesMS: map[string]int{"records": 12},
		},
	}
	raw, err := json.Marshal(rep.ToDTO())
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"inpxPath", "inpxVersion", "recordsSeen", "editionsUpdated", "libidCollisions"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("missing %s in %s", key, raw)
		}
	}
	notes, _ := m["notes"].(map[string]any)
	if notes == nil {
		t.Fatalf("notes: %s", raw)
	}
	for _, key := range []string{
		"missingArchives", "missingArchivesTotal", "unnamedGenres", "unnamedGenresTotal",
		"skippedMalformed", "skippedNoLibid", "genreNamesMapped", "encodings", "phasesMs",
	} {
		if _, ok := notes[key]; !ok {
			t.Fatalf("missing notes.%s in %s", key, raw)
		}
	}
	enc, _ := notes["encodings"].(map[string]any)
	if enc["utf8"] != float64(219) || enc["versionInfo"] != "utf-8" {
		t.Fatalf("encodings %+v", enc)
	}
}

func TestParseNotesRoundTrip(t *testing.T) {
	stored := `{"missing_archives":["a.zip"],"missing_archives_total":1,"skipped_no_libid":2,"encodings":{"utf-8":3,"cp1251":1}}`
	n := ParseNotes(stored)
	if n.MissingArchivesTotal != 1 || n.SkippedNoLibID != 2 || n.Encodings.UTF8 != 3 {
		t.Fatalf("%+v", n)
	}
	dto := n.ToDTO()
	if dto.SkippedNoLibid != 2 || dto.Encodings.CP1251 != 1 {
		t.Fatalf("%+v", dto)
	}
}
