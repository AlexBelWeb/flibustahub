package db

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alexbelweb/flibustahub/migrations"
)

var ftsCreateRe = regexp.MustCompile(`(?is)CREATE VIRTUAL TABLE\s+(works_fts|authors_fts|series_fts)\s+USING\s+fts5\s*\([^;]*?\)`)
var worksFTSTriggerRe = regexp.MustCompile(`(?is)CREATE TRIGGER\s+(works_fts_au|works_fts_ad)\b.*?END;`)

const (
	MetaFTSDirty      = "fts_dirty"
	MetaWorksTotal    = "works_total"
	MetaWorksListable = "works_listable"
	MetaLastWarmup    = "last_warmup_at"
	importCacheSize   = -128000
	workingCacheSize  = -64000
)

func initialSQL() (string, error) {
	raw, err := migrations.FS.ReadFile("001_initial.sql")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// FTSCreate returns the CREATE VIRTUAL TABLE statement for name from 001_initial.sql.
func FTSCreate(name string) (string, error) {
	raw, err := initialSQL()
	if err != nil {
		return "", err
	}
	matches := ftsCreateRe.FindAllString(raw, -1)
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

// WorksFTSTriggers returns CREATE TRIGGER statements for works_fts from 001_initial.sql.
func WorksFTSTriggers() ([]string, error) {
	raw, err := initialSQL()
	if err != nil {
		return nil, err
	}
	matches := worksFTSTriggerRe.FindAllString(raw, -1)
	if len(matches) != 2 {
		return nil, fmt.Errorf("works_fts triggers: got %d, want 2", len(matches))
	}
	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = strings.TrimSpace(m)
	}
	return out, nil
}
