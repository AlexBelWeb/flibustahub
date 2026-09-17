package platform

import "testing"

func TestMatchLocale(t *testing.T) {
	cases := map[string]string{
		"ru":          "ru",
		"ru-RU":       "ru",
		"ru_RU.UTF-8": "ru",
		"en-US":       "en",
		"uk":          "en",
		"":            "en",
	}
	for in, want := range cases {
		if got := MatchLocale(in); got != want {
			t.Fatalf("MatchLocale(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDetectLocaleFromEnv(t *testing.T) {
	t.Setenv("LC_ALL", "ru_RU.UTF-8")
	t.Setenv("LANG", "en_US.UTF-8")
	if got := DetectLocale(); got != "ru" {
		t.Fatalf("got %q", got)
	}
}
