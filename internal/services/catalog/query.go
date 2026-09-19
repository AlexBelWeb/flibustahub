package catalog

import (
	"regexp"
	"strings"

	"github.com/alexbelweb/flibustahub/internal/textnorm"
)

var tokenJunk = regexp.MustCompile(`["^:*()%_]`)

// Tokens splits a user query into FTS/LIKE tokens using the same normalize as the index.
func Tokens(q string) []string {
	var out []string
	for _, raw := range strings.Fields(q) {
		raw = tokenJunk.ReplaceAllString(raw, "")
		n := textnorm.Normalize(raw)
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}

func ftsPhrase(tokens []string, columns string) string {
	if len(tokens) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tokens))
	for _, t := range tokens {
		parts = append(parts, `"`+t+`"*`)
	}
	body := strings.Join(parts, " AND ")
	if columns == "" {
		return body
	}
	return "{" + columns + "}: " + body
}

const worksFTSColumns = "title authors series"

func likePatterns(tokens []string) []string {
	out := make([]string, len(tokens))
	for i, t := range tokens {
		out[i] = "% " + t + "%"
	}
	return out
}
