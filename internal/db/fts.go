package db

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alexbelweb/flibustahub/migrations"
)

var ftsCreateRe = regexp.MustCompile(`(?is)CREATE VIRTUAL TABLE\s+(works_fts|authors_fts|series_fts)\s+USING\s+fts5\s*\([^;]*?\)`)

// FTSCreate returns the CREATE VIRTUAL TABLE statement for name from 001_initial.sql.
// rebuildFts in a later slice must use this so DROP+CREATE stays byte-identical.
func FTSCreate(name string) (string, error) {
	raw, err := migrations.FS.ReadFile("001_initial.sql")
	if err != nil {
		return "", err
	}
	matches := ftsCreateRe.FindAllString(string(raw), -1)
	want := strings.ToLower(name)
	for _, m := range matches {
		if strings.Contains(strings.ToLower(m), want+" ") || strings.Contains(strings.ToLower(m), want+"\t") {
			return strings.TrimSpace(m), nil
		}
		fields := strings.Fields(strings.ToLower(m))
		for i, f := range fields {
			if f == "table" && i+1 < len(fields) && fields[i+1] == want {
				return strings.TrimSpace(m), nil
			}
		}
	}
	return "", fmt.Errorf("fts create for %s not found", name)
}
