// Package alphabet is the shared letter index for the UI and OPDS.
// Buckets use sort_name, where ё is already folded to е, so there is no Ё shelf.
package alphabet

import "unicode"

// Other is the catch-all bucket for names that do not start with a catalog letter.
const Other = "other"

// Letters returns catalog shelves in display order: А–Я (no Ё), A–Z, then Other.
// The Е shelf key is "е"; the UI and OPDS label it "Е (Ё)".
func Letters() []string {
	out := make([]string, 0, 33+26+1)
	for r := 'а'; r <= 'я'; r++ {
		if r == 'ё' {
			continue
		}
		out = append(out, string(r))
	}
	for r := 'a'; r <= 'z'; r++ {
		out = append(out, string(r))
	}
	out = append(out, Other)
	return out
}

// Bucket returns the shelf key for a normalized sort_name.
func Bucket(sortName string) string {
	if sortName == "" {
		return Other
	}
	r := []rune(sortName)[0]
	if r == 'ё' {
		return "е"
	}
	if r >= 'а' && r <= 'я' {
		return string(r)
	}
	if r >= 'a' && r <= 'z' {
		return string(r)
	}
	return Other
}

// PrefixRange is the half-open [lo, hi) interval on sort_name for a letter key.
// ok is false for Other, which cannot be expressed as a single prefix range.
func PrefixRange(letter string) (lo, hi string, ok bool) {
	if letter == "" || letter == Other {
		return "", "", false
	}
	rs := []rune(letter)
	if len(rs) != 1 {
		return "", "", false
	}
	r := rs[0]
	if !isShelfRune(r) {
		return "", "", false
	}
	return string(r), string(r + 1), true
}

func isShelfRune(r rune) bool {
	if r == 'ё' {
		return false
	}
	if r >= 'а' && r <= 'я' {
		return true
	}
	return r >= 'a' && r <= 'z'
}

// Known reports whether letter is a shelf key returned by Letters, including Other.
func Known(letter string) bool {
	if letter == Other {
		return true
	}
	rs := []rune(letter)
	return len(rs) == 1 && isShelfRune(rs[0])
}

// OtherSQL is the predicate "first character is not a catalog letter" on column `sort_name`.
// SQLite GLOB character classes are ASCII, so this uses unicode code points instead.
func OtherSQL(column string) string {
	// Latin a–z, then a gap, then Cyrillic а–я. Anything else (digits, empty, other scripts) is Other.
	return "(" + column + " = '' OR " + column + " < 'a' OR (" + column + " >= '{' AND " + column + " < 'а') OR " + column + " >= '" + string(rune('я'+1)) + "')"
}

// IsLetter reports whether r is a catalog shelf rune (ё is not; it belongs on е).
func IsLetter(r rune) bool {
	return isShelfRune(unicode.ToLower(r))
}
