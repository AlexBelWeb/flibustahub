package fb2

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const minPlausible = 0.82

// PlausibleText reports whether s looks like human language rather than mojibake.
func PlausibleText(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	if strings.ContainsRune(s, '\uFFFD') {
		return false
	}
	return TextScore(s) >= minPlausible
}

// TextScore is the share of letters, digits, space and ordinary punctuation
// versus controls, box drawing and rare signs. Higher is better.
func TextScore(s string) float64 {
	if s == "" {
		return 1
	}
	good, bad := 0, 0
	for _, r := range s {
		switch {
		case r == '\uFFFD':
			bad += 8
		case r == '\n' || r == '\r' || r == '\t':
			good++
		case unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r):
			good++
		case isOrdinaryPunct(r):
			good++
		case unicode.IsControl(r) || isBoxDrawing(r):
			bad += 4
		default:
			bad++
		}
	}
	total := good + bad
	if total == 0 {
		return 0
	}
	return float64(good) / float64(total)
}

func isOrdinaryPunct(r rune) bool {
	switch r {
	case '.', ',', ';', ':', '!', '?', '-', '—', '–', '\'', '"', '«', '»',
		'(', ')', '[', ']', '{', '}', '/', '\\', '&', '%', '+', '=', '*',
		'#', '@', '_', '~', '…', '“', '”', '‘', '’', '№':
		return true
	default:
		return false
	}
}

func isBoxDrawing(r rune) bool {
	return (r >= 0x2500 && r <= 0x257F) || (r >= 0x2580 && r <= 0x259F)
}

func utf16WithoutBOM(raw []byte) (charset, bool) {
	if len(raw) < 32 || len(raw)%2 != 0 {
		return charset{}, false
	}
	if utf8.Valid(raw) {
		zeros := 0
		for _, b := range raw[:min(len(raw), 512)] {
			if b == 0 {
				zeros++
			}
		}
		if zeros < 8 {
			return charset{}, false
		}
	}
	n := min(len(raw), 4096)
	evenZ, oddZ := 0, 0
	for i := 0; i < n; i++ {
		if raw[i] != 0 {
			continue
		}
		if i%2 == 0 {
			evenZ++
		} else {
			oddZ++
		}
	}
	pairs := n / 2
	if pairs == 0 {
		return charset{}, false
	}
	if oddZ >= pairs/4 && oddZ > evenZ*2 {
		return charset{name: "utf-16le", enc: unicodeUTF16LE}, true
	}
	if evenZ >= pairs/4 && evenZ > oddZ*2 {
		return charset{name: "utf-16be", enc: unicodeUTF16BE}, true
	}
	return charset{}, false
}
