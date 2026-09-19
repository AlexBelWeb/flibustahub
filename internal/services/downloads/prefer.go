package downloads

import (
	"strings"
	"time"
)

// Edition is a visible file of a work used to pick a preferred download.
type Edition struct {
	ID        int64
	FileExt   string
	AddedDate string
}

func extRank(ext string) int {
	switch strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), ".")) {
	case "fb2":
		return 0
	case "epub":
		return 1
	default:
		return 2
	}
}

// Preferred returns the preferred visible edition, or zero if the list is empty.
func Preferred(eds []Edition) Edition {
	var best Edition
	var have bool
	for _, e := range eds {
		if e.ID <= 0 {
			continue
		}
		if !have || betterEdition(e, best) {
			best = e
			have = true
		}
	}
	return best
}

func betterEdition(a, b Edition) bool {
	ra, rb := extRank(a.FileExt), extRank(b.FileExt)
	if ra != rb {
		return ra < rb
	}
	if a.AddedDate != b.AddedDate {
		return a.AddedDate > b.AddedDate
	}
	return a.ID > b.ID
}

func parseDumpVersion(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int64(r-'0')
	}
	return n, true
}

// DumpNewer reports whether the on-disk dump should be offered as an update.
func DumpNewer(fileVer, catalogVer, dismissed string, fileMtime, importedAt time.Time) bool {
	if strings.TrimSpace(fileVer) == "" && fileMtime.IsZero() {
		return false
	}
	fv, fOK := parseDumpVersion(fileVer)
	cv, cOK := parseDumpVersion(catalogVer)
	dv, dOK := parseDumpVersion(dismissed)
	if fOK && cOK {
		if fv <= cv {
			return false
		}
		if dismissed != "" && dOK && fv <= dv {
			return false
		}
		if dismissed != "" && !dOK && fileVer == dismissed {
			return false
		}
		return true
	}
	if dismissed != "" && fileVer == dismissed {
		return false
	}
	if catalogVer != "" && fileVer == catalogVer {
		return false
	}
	if importedAt.IsZero() {
		return catalogVer != "" && fileVer != catalogVer
	}
	return fileMtime.After(importedAt)
}
