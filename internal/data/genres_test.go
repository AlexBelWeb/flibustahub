package data

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedDictionaryIntegrity(t *testing.T) {
	dec := json.NewDecoder(bytes.NewReader(genresJSON))
	tok, err := dec.Token()
	if err != nil {
		t.Fatal(err)
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		t.Fatalf("expected object, got %v", tok)
	}
	seen := map[string]struct{}{}
	prev := ""
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			t.Fatal(err)
		}
		code, ok := keyTok.(string)
		if !ok {
			t.Fatalf("key type %T", keyTok)
		}
		var name string
		if err := dec.Decode(&name); err != nil {
			t.Fatal(err)
		}
		if code == "" || name == "" {
			t.Errorf("empty code or name: %q=%q", code, name)
		}
		if _, dup := seen[code]; dup {
			t.Errorf("duplicate key %q", code)
		}
		seen[code] = struct{}{}
		if prev != "" && code <= prev {
			t.Errorf("keys not sorted: %q after %q", code, prev)
		}
		prev = code
	}
	if _, err := dec.Token(); err != nil {
		t.Fatal(err)
	}
	if len(seen) == 0 {
		t.Fatal("empty dictionary")
	}
}

func TestOfficialFB2Names(t *testing.T) {
	Load("", slog.New(slog.DiscardHandler))
	if GenreNameRU("sf_social") != "Социально-психологическая фантастика" {
		t.Fatalf("%q", GenreNameRU("sf_social"))
	}
	if GenreCount() < 200 {
		t.Fatalf("dictionary too small: %d", GenreCount())
	}
}

func TestDictionaryCoversCommonGenreCodes(t *testing.T) {
	Load("", slog.New(slog.DiscardHandler))
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
			t.Errorf("code %s missing from dictionary", code)
		}
		if GenreNameRU(code) == code {
			t.Errorf("%s fell back to code", code)
		}
	}
}

func TestUnknownGenreFallsBackToCode(t *testing.T) {
	Load("", slog.New(slog.DiscardHandler))
	if GenreNameRU("totally_unknown_genre") != "totally_unknown_genre" {
		t.Fatal(GenreNameRU("totally_unknown_genre"))
	}
	if KnownGenre("totally_unknown_genre") {
		t.Fatal("unknown must not be marked known")
	}
}

func TestLocalOverrideMergesAndBrokenFileKeepsBuiltin(t *testing.T) {
	t.Cleanup(func() { Load("", slog.New(slog.DiscardHandler)) })

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, LocalFileName), []byte(`{"sf_social":"Тестовое имя","brand_new_code":"Новый жанр"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	Load(dir, slog.New(slog.DiscardHandler))
	if GenreNameRU("sf_social") != "Тестовое имя" {
		t.Fatalf("override: %q", GenreNameRU("sf_social"))
	}
	if !KnownGenre("brand_new_code") || GenreNameRU("brand_new_code") != "Новый жанр" {
		t.Fatal("overlay code missing")
	}

	broken := t.TempDir()
	if err := os.WriteFile(filepath.Join(broken, LocalFileName), []byte(`{not json`), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	Load(broken, log)
	if GenreNameRU("sf_social") != "Социально-психологическая фантастика" {
		t.Fatalf("broken overlay must not replace built-in, got %q", GenreNameRU("sf_social"))
	}
	if !bytes.Contains(buf.Bytes(), []byte("genre override")) {
		t.Fatalf("expected warn log, got %s", buf.String())
	}
}
