package downloads

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxStemRunes = 120
	minStemRunes = 20
	maxPathRunes = 240
	collisionPad = " (99)"
)

var osStat = os.Stat

var errNoSlot = errors.New("no unique filename available")

var winReserved = map[string]struct{}{
	"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
	"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {},
	"COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
	"LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {},
	"LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
}

func firstAuthor(authors string) string {
	authors = strings.TrimSpace(authors)
	if i := strings.IndexRune(authors, ','); i >= 0 {
		authors = authors[:i]
	}
	return cleanComponent(authors)
}

// Stem builds a sanitized basename (no extension) from author and title.
func Stem(author, title string) string {
	author = firstAuthor(author)
	title = cleanComponent(title)
	var stem string
	switch {
	case author != "" && title != "":
		stem = author + " — " + title
	case title != "":
		stem = title
	case author != "":
		stem = author
	default:
		return ""
	}
	stem = trimWinTail(stem)
	stem = truncateRunes(stem, maxStemRunes)
	stem = trimWinTail(stem)
	return avoidReserved(stem)
}

// FileName returns a unique file name in dir. dest is the full path.
func FileName(dir, author, title, ext, fallback string) (name, dest string, err error) {
	stem := Stem(author, title)
	if stem == "" {
		stem = cleanComponent(fallback)
	}
	if stem == "" {
		stem = "book"
	}
	ext = normalizeExt(ext)
	stem = fitStem(dir, stem, ext)
	return uniqueName(dir, stem, ext)
}

func normalizeExt(ext string) string {
	ext = strings.TrimSpace(strings.TrimPrefix(ext, "."))
	ext = cleanComponent(ext)
	if ext == "" {
		return ".fb2"
	}
	return "." + strings.ToLower(ext)
}

func cleanComponent(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		if r < 32 || invalidFileRune(r) {
			continue
		}
		b.WriteRune(r)
	}
	s = strings.Join(strings.Fields(b.String()), " ")
	return trimWinTail(s)
}

func invalidFileRune(r rune) bool {
	switch r {
	case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
		return true
	}
	return unicode.IsControl(r)
}

func trimWinTail(s string) string {
	return strings.TrimRight(s, " .")
}

func avoidReserved(stem string) string {
	upper := strings.ToUpper(stem)
	if _, ok := winReserved[upper]; ok {
		return stem + "_"
	}
	if i := strings.IndexByte(upper, '.'); i > 0 {
		if _, ok := winReserved[upper[:i]]; ok {
			return stem + "_"
		}
	}
	return stem
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n])
}

func fitStem(dir, stem, ext string) string {
	for utf8.RuneCountInString(stem) > minStemRunes {
		full := filepath.Join(dir, stem+ext)
		if utf8.RuneCountInString(full)+utf8.RuneCountInString(collisionPad) <= maxPathRunes {
			return stem
		}
		runes := []rune(stem)
		stem = trimWinTail(string(runes[:len(runes)-1]))
		if stem == "" {
			break
		}
	}
	if stem == "" {
		return "book"
	}
	if utf8.RuneCountInString(stem) > minStemRunes {
		stem = truncateRunes(stem, minStemRunes)
		stem = trimWinTail(stem)
	}
	return stem
}

func uniqueName(dir, stem, ext string) (string, string, error) {
	try := func(name string) (string, string, bool) {
		dest := filepath.Join(dir, name)
		if !insideDir(dir, dest) {
			return "", "", false
		}
		return name, dest, true
	}
	name := stem + ext
	if n, dest, ok := try(name); ok {
		if _, err := osStat(dest); err != nil {
			return n, dest, nil
		}
	}
	for i := 2; i < 10000; i++ {
		name = stem + " (" + itoa(i) + ")" + ext
		n, dest, ok := try(name)
		if !ok {
			continue
		}
		if _, err := osStat(dest); err != nil {
			return n, dest, nil
		}
	}
	return "", "", errNoSlot
}

func insideDir(dir, dest string) bool {
	dir = filepath.Clean(dir)
	dest = filepath.Clean(dest)
	rel, err := filepath.Rel(dir, dest)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
