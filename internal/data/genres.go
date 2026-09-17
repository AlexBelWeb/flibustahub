// Package data holds static catalog lookup tables.
package data

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed genres.json
var genresJSON []byte

var (
	genresOnce sync.Once
	genreNames map[string]string
)

func loadGenres() {
	genreNames = map[string]string{}
	if err := json.Unmarshal(genresJSON, &genreNames); err != nil {
		panic("data: genres.json: " + err.Error())
	}
}

// GenreNameRU returns the Russian label for a genre code, or the code itself.
func GenreNameRU(code string) string {
	genresOnce.Do(loadGenres)
	if name, ok := genreNames[code]; ok && name != "" {
		return name
	}
	return code
}

// KnownGenre reports whether the code is in the static dictionary.
func KnownGenre(code string) bool {
	genresOnce.Do(loadGenres)
	_, ok := genreNames[code]
	return ok
}

// GenreCount is the number of codes in the embedded dictionary.
func GenreCount() int {
	genresOnce.Do(loadGenres)
	return len(genreNames)
}
