package catalog

import "testing"

func TestTokensNormalizeYoAndCase(t *testing.T) {
	a := Tokens("Ёлка")
	b := Tokens("елка")
	c := Tokens("ЕЛКА")
	if len(a) != 1 || a[0] != "елка" {
		t.Fatalf("Ёлка → %v", a)
	}
	if a[0] != b[0] || a[0] != c[0] {
		t.Fatalf("tokens differ: %v %v %v", a, b, c)
	}
}

func TestTokensStripAndDropEmpty(t *testing.T) {
	got := Tokens(`  "foo" ^bar: baz*  """  `)
	if len(got) != 3 || got[0] != "foo" || got[1] != "bar" || got[2] != "baz" {
		t.Fatalf("got %v", got)
	}
	if Tokens("***") != nil && len(Tokens("***")) != 0 {
		t.Fatalf("junk tokens: %v", Tokens("***"))
	}
}

func TestFTSPhrase(t *testing.T) {
	q := ftsPhrase([]string{"елка", "гром"}, "title series")
	want := `{title series}: "елка"* AND "гром"*`
	if q != want {
		t.Fatalf("got %q", q)
	}
}
