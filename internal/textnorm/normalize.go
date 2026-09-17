// Package textnorm implements the single catalog string normalization used
// for FTS, LIKE fallback, sort keys and work_key.
package textnorm

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Normalize applies NFC, lowercase, yo→ye, punctuation to spaces, and
// collapsed/trimmed whitespace.
func Normalize(s string) string {
	s = norm.NFC.String(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "ё", "е")

	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			if b.Len() == 0 || prevSpace {
				continue
			}
			b.WriteByte(' ')
			prevSpace = true
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return strings.TrimSpace(b.String())
}

// SearchNorm wraps Normalize with surrounding spaces for start-of-word LIKE.
func SearchNorm(s string) string {
	return " " + Normalize(s) + " "
}
