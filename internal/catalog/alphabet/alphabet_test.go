package alphabet

import "testing"

func TestLettersHasNoYoShelf(t *testing.T) {
	seen := map[string]bool{}
	for _, k := range Letters() {
		if k == "ё" {
			t.Fatal("Ё must not be a separate shelf")
		}
		if seen[k] {
			t.Fatalf("duplicate %q", k)
		}
		seen[k] = true
	}
	if !seen["е"] {
		t.Fatal("missing Е shelf")
	}
	if !seen["й"] {
		t.Fatal("missing Й shelf")
	}
	if !seen["a"] || !seen["z"] || !seen[Other] {
		t.Fatal("missing latin or Other")
	}
	if got, want := len(Letters()), 32+26+1; got != want {
		t.Fatalf("len=%d want %d", got, want)
	}
}

func TestBucketFoldsYoIntoYe(t *testing.T) {
	if Bucket("елкин") != "е" {
		t.Fatalf("елкин: %q", Bucket("елкин"))
	}
	if Bucket("ёжик") != "е" {
		t.Fatalf("raw ё should still land on е, got %q", Bucket("ёжик"))
	}
	if Bucket("йенсен") != "й" {
		t.Fatalf("й: %q", Bucket("йенсен"))
	}
	if Bucket("asimov") != "a" {
		t.Fatalf("latin: %q", Bucket("asimov"))
	}
	if Bucket("") != Other || Bucket("42") != Other {
		t.Fatal("empty and digits belong in Other")
	}
}

func TestPrefixRangeYe(t *testing.T) {
	lo, hi, ok := PrefixRange("е")
	if !ok || lo != "е" || hi != "ж" {
		t.Fatalf("е range %q %q ok=%v", lo, hi, ok)
	}
	if _, _, ok := PrefixRange(Other); ok {
		t.Fatal("Other is not a prefix range")
	}
	if _, _, ok := PrefixRange("ё"); ok {
		t.Fatal("ё is not a shelf key")
	}
}
