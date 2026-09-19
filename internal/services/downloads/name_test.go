package downloads

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestStemStripsTraversalAndInvalid(t *testing.T) {
	got := Stem(`A/B\C:?*`, `../title<>|"name".`)
	if strings.ContainsAny(got, `<>:"/\|?*`) {
		t.Fatalf("invalid chars remain: %q", got)
	}
	if strings.HasSuffix(got, ".") || strings.HasSuffix(got, " ") {
		t.Fatalf("trailing dot/space: %q", got)
	}
}

func TestStemTrailingDotAndSpace(t *testing.T) {
	got := Stem("Автор", "Название...")
	if strings.HasSuffix(got, ".") || strings.HasSuffix(got, " ") {
		t.Fatalf("got %q", got)
	}
	if !strings.HasSuffix(got, "Название") {
		t.Fatalf("got %q", got)
	}
}

func TestStemFirstAuthorOnly(t *testing.T) {
	got := Stem("Лев Толстой,  А. Кто-то", "Война и мир")
	if got != "Лев Толстой — Война и мир" {
		t.Fatalf("got %q", got)
	}
}

func TestStemTruncatesTo120Runes(t *testing.T) {
	title := strings.Repeat("я", 200)
	got := Stem("A", title)
	if utf8.RuneCountInString(got) > maxStemRunes {
		t.Fatalf("len=%d", utf8.RuneCountInString(got))
	}
}

func TestFileNameCollisionSuffixes(t *testing.T) {
	dir := t.TempDir()
	n1, p1, err := FileName(dir, "Author", "Title", "fb2", "1")
	if err != nil {
		t.Fatal(err)
	}
	if n1 != "Author — Title.fb2" {
		t.Fatalf("first = %q", n1)
	}
	if err := os.WriteFile(p1, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	n2, p2, err := FileName(dir, "Author", "Title", "fb2", "1")
	if err != nil {
		t.Fatal(err)
	}
	if n2 != "Author — Title (2).fb2" {
		t.Fatalf("second = %q", n2)
	}
	if err := os.WriteFile(p2, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	n3, _, err := FileName(dir, "Author", "Title", "fb2", "1")
	if err != nil {
		t.Fatal(err)
	}
	if n3 != "Author — Title (3).fb2" {
		t.Fatalf("third = %q", n3)
	}
}

func TestFitStemKeepsPathWithinBudget(t *testing.T) {
	dir := strings.Repeat("d", 150)
	long := "Автор — " + strings.Repeat("б", 80)
	stem := fitStem(dir, long, ".fb2")
	full := filepath.Join(dir, stem+".fb2")
	if utf8.RuneCountInString(full)+utf8.RuneCountInString(collisionPad) > maxPathRunes {
		t.Fatalf("path runes=%d stem=%q", utf8.RuneCountInString(full), stem)
	}
	if utf8.RuneCountInString(stem) < minStemRunes {
		t.Fatalf("stem too short: %q", stem)
	}
	if stem == long {
		t.Fatal("expected shrink")
	}
}

func TestFileNameDeepDirectoryShrinksStem(t *testing.T) {
	base := t.TempDir()
	pad := strings.Repeat("d", 70)
	deep := filepath.Join(base, pad, pad)
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	title := strings.Repeat("б", 80)
	name, dest, err := FileName(deep, "Автор", title, "fb2", "9")
	if err != nil {
		t.Fatal(err)
	}
	stem := strings.TrimSuffix(name, ".fb2")
	if utf8.RuneCountInString(stem) < minStemRunes {
		t.Fatalf("stem too short: %q", stem)
	}
	untrimmed := filepath.Join(deep, "Автор — "+title+".fb2")
	if utf8.RuneCountInString(untrimmed) <= maxPathRunes {
		t.Fatal("fixture directory is not deep enough to force a shrink")
	}
	if utf8.RuneCountInString(stem) >= utf8.RuneCountInString("Автор — "+title) {
		t.Fatalf("stem was not shrunk: %q", stem)
	}
	if utf8.RuneCountInString(dest) > maxPathRunes && utf8.RuneCountInString(stem) > minStemRunes {
		t.Fatalf("path runes=%d stem=%q", utf8.RuneCountInString(dest), stem)
	}
	_ = dest
}

func TestFileNameStaysInsideDir(t *testing.T) {
	dir := t.TempDir()
	_, dest, err := FileName(dir, "", `..\..\secret`, "fb2", "7")
	if err != nil {
		t.Fatal(err)
	}
	if !insideDir(dir, dest) {
		t.Fatalf("escaped: %q", dest)
	}
}

func TestEmptyTitleFallsBackToID(t *testing.T) {
	dir := t.TempDir()
	name, _, err := FileName(dir, "", "", "epub", "42")
	if err != nil {
		t.Fatal(err)
	}
	if name != "42.epub" {
		t.Fatalf("got %q", name)
	}
}
