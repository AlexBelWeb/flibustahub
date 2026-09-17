// Package data holds static catalog lookup tables.
package data

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

//go:embed genres.json
var genresJSON []byte

const LocalFileName = "genres.local.json"

var (
	genreMu    sync.RWMutex
	genreNames map[string]string
)

func init() {
	loadEmbeddedLocked()
}

func loadEmbeddedLocked() {
	names := map[string]string{}
	if err := json.Unmarshal(genresJSON, &names); err != nil {
		panic("data: genres.json: " + err.Error())
	}
	genreMu.Lock()
	genreNames = names
	genreMu.Unlock()
}

// Load installs the embedded dictionary, then optionally merges {dataDir}/genres.local.json.
// A missing file is ignored. Damaged JSON is a warning; the built-in map stays in effect.
func Load(dataDir string, log *slog.Logger) {
	if log == nil {
		log = slog.Default()
	}
	names := map[string]string{}
	if err := json.Unmarshal(genresJSON, &names); err != nil {
		panic("data: genres.json: " + err.Error())
	}
	if dataDir != "" {
		overlay := filepath.Join(dataDir, LocalFileName)
		raw, err := os.ReadFile(overlay)
		switch {
		case err == nil:
			var extra map[string]string
			if err := json.Unmarshal(raw, &extra); err != nil {
				log.Warn("genre override ignored, using built-in dictionary", "path", overlay, "err", err)
			} else {
				for code, name := range extra {
					if code == "" || name == "" {
						continue
					}
					names[code] = name
				}
			}
		case errors.Is(err, fs.ErrNotExist):
			// optional file
		default:
			log.Warn("genre override unreadable, using built-in dictionary", "path", overlay, "err", err)
		}
	}
	genreMu.Lock()
	genreNames = names
	genreMu.Unlock()
}

// GenreNameRU returns the Russian label for a genre code, or the code itself.
func GenreNameRU(code string) string {
	genreMu.RLock()
	defer genreMu.RUnlock()
	if name, ok := genreNames[code]; ok && name != "" {
		return name
	}
	return code
}

// KnownGenre reports whether the code is in the effective dictionary.
func KnownGenre(code string) bool {
	genreMu.RLock()
	defer genreMu.RUnlock()
	_, ok := genreNames[code]
	return ok
}

// GenreCount is the number of codes in the effective dictionary.
func GenreCount() int {
	genreMu.RLock()
	defer genreMu.RUnlock()
	return len(genreNames)
}
