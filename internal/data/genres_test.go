package data

import "testing"

func TestOfficialFB2Names(t *testing.T) {
	if GenreNameRU("sf_social") != "Социально-психологическая фантастика" {
		t.Fatalf("%q", GenreNameRU("sf_social"))
	}
	if GenreCount() < 200 {
		t.Fatalf("dictionary too small: %d", GenreCount())
	}
}

func TestJournalDumpCodesCovered(t *testing.T) {
	for _, code := range []string{
		"network_literature",
		"love_sf",
		"sf_fantasy",
		"love_contemporary",
		"prose_contemporary",
		"popadancy",
		"sf_action",
		"sf",
	} {
		if !KnownGenre(code) {
			t.Errorf("journal code %s missing from dictionary", code)
		}
		if GenreNameRU(code) == code {
			t.Errorf("%s fell back to code", code)
		}
	}
}

func TestUnknownGenreFallsBackToCode(t *testing.T) {
	if GenreNameRU("totally_unknown_genre") != "totally_unknown_genre" {
		t.Fatal(GenreNameRU("totally_unknown_genre"))
	}
	if KnownGenre("totally_unknown_genre") {
		t.Fatal("unknown must not be marked known")
	}
}
