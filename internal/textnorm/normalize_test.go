package textnorm

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Ёлка", "елка"},
		{"  Hello,   world! ", "hello world"},
		{"«Ёлка»", "елка"},
		{"", ""},
		{"   ", ""},
		{"е\u0308лка", "елка"}, // combining diaeresis → NFC ёлка → елка
	}
	for _, tc := range cases {
		if got := Normalize(tc.in); got != tc.want {
			t.Fatalf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSearchNormWraps(t *testing.T) {
	if got := SearchNorm("Ёлка"); got != " елка " {
		t.Fatalf("got %q", got)
	}
	if got := SearchNorm(""); got != "  " {
		t.Fatalf("empty: %q", got)
	}
}

func TestNormalizeIdempotent(t *testing.T) {
	s := Normalize("«Hello, Ёлка!»")
	if Normalize(s) != s {
		t.Fatalf("not idempotent: %q", s)
	}
}
