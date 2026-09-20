package personal

import (
	"testing"
	"time"
)

func TestStarsFromInternal(t *testing.T) {
	cases := map[int]string{1: "0.5", 2: "1", 3: "1.5", 9: "4.5", 10: "5"}
	for n, want := range cases {
		if got := StarsFromInternal(n); got != want {
			t.Errorf("%d -> %q, want %q", n, got, want)
		}
	}
}

func TestParseStars(t *testing.T) {
	ok := map[string]int{
		"0.5": 1, "1": 2, "1.5": 3, "4.5": 9, "5": 10,
		"4,5": 9, " 2 ": 4,
	}
	for raw, want := range ok {
		n, good := ParseStars(raw)
		if !good || n != want {
			t.Errorf("%q -> %d ok=%v, want %d", raw, n, good, want)
		}
	}
	empty, good := ParseStars("")
	if !good || empty != 0 {
		t.Fatalf("empty should parse as unset, got %d ok=%v", empty, good)
	}
	for _, bad := range []string{"0", "5.5", "4.25", "7", "abc", "1.0.0"} {
		if _, good := ParseStars(bad); good {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestParseWantToRead(t *testing.T) {
	for _, raw := range []string{"1", "true", "TRUE", "0", "false", "False"} {
		v, ok := ParseWantToRead(raw)
		if !ok || v == nil {
			t.Fatalf("%q: ok=%v v=%v", raw, ok, v)
		}
		wantTrue := raw == "1" || stringsEqualFoldTrue(raw)
		if *v != wantTrue {
			t.Errorf("%q -> %v, want %v", raw, *v, wantTrue)
		}
	}
	v, ok := ParseWantToRead("")
	if !ok || v != nil {
		t.Fatalf("empty want: %v %v", v, ok)
	}
	if _, ok := ParseWantToRead("yes"); ok {
		t.Fatal("yes must be invalid")
	}
}

func TestStampLayoutIsFixedWidthUTC(t *testing.T) {
	if got := Stamp(time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)); got != "2026-09-20T12:00:00.000Z" {
		t.Fatalf("zero ns: %q", got)
	}
	if got := Stamp(time.Date(2026, 9, 20, 12, 0, 0, 123456789, time.UTC)); got != "2026-09-20T12:00:00.123Z" {
		t.Fatalf("frac: %q", got)
	}
}

func stringsEqualFoldTrue(s string) bool {
	switch s {
	case "true", "TRUE", "True":
		return true
	default:
		return false
	}
}
