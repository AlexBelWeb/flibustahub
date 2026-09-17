package db

import (
	"strings"
	"unicode"
)

func splitSQL(script string) []string {
	var stmts []string
	var b strings.Builder
	depth := 0
	inSingle := false
	i := 0
	for i < len(script) {
		c := script[i]
		if inSingle {
			b.WriteByte(c)
			if c == '\'' {
				if i+1 < len(script) && script[i+1] == '\'' {
					b.WriteByte(script[i+1])
					i += 2
					continue
				}
				inSingle = false
			}
			i++
			continue
		}
		if c == '\'' {
			inSingle = true
			b.WriteByte(c)
			i++
			continue
		}
		if c == '-' && i+1 < len(script) && script[i+1] == '-' {
			for i < len(script) && script[i] != '\n' {
				i++
			}
			continue
		}
		if wordAt(script, i, "BEGIN") {
			depth++
			b.WriteString(script[i : i+5])
			i += 5
			continue
		}
		if wordAt(script, i, "END") {
			b.WriteString(script[i : i+3])
			i += 3
			if depth > 0 {
				depth--
			}
			continue
		}
		if c == ';' && depth == 0 {
			stmt := strings.TrimSpace(b.String())
			if stmt != "" {
				stmts = append(stmts, stmt)
			}
			b.Reset()
			i++
			continue
		}
		b.WriteByte(c)
		i++
	}
	if rest := strings.TrimSpace(b.String()); rest != "" {
		stmts = append(stmts, rest)
	}
	return stmts
}

func wordAt(s string, i int, word string) bool {
	n := len(word)
	if i+n > len(s) {
		return false
	}
	if !strings.EqualFold(s[i:i+n], word) {
		return false
	}
	if i > 0 && isIdent(rune(s[i-1])) {
		return false
	}
	if i+n < len(s) && isIdent(rune(s[i+n])) {
		return false
	}
	return true
}

func isIdent(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
