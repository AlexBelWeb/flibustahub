package inpximport

import (
	"encoding/json"

	"github.com/alexbelweb/flibustahub/internal/inpx"
)

// ParseNotes decodes import_batches.notes JSON (snake_case storage).
func ParseNotes(raw string) Notes {
	var n Notes
	if raw == "" {
		return n
	}
	_ = json.Unmarshal([]byte(raw), &n)
	return n
}

// ReportDTO is the Wails-facing import report. JSON is camelCase.
// Storage in import_batches.notes stays snake_case.
type ReportDTO struct {
	ID                  int64    `json:"id"`
	Status              string   `json:"status"`
	INPXPath            string   `json:"inpxPath"`
	INPXVersion         string   `json:"inpxVersion"`
	FinishedAt          string   `json:"finishedAt,omitempty"`
	RecordsSeen         int      `json:"recordsSeen"`
	WorksAdded          int      `json:"worksAdded"`
	EditionsAdded       int      `json:"editionsAdded"`
	EditionsUpdated     int      `json:"editionsUpdated"`
	EditionsDeactivated int      `json:"editionsDeactivated"`
	LibIDCollisions     int      `json:"libidCollisions"`
	Notes               NotesDTO `json:"notes"`
}

// NotesDTO is notes for the UI.
type NotesDTO struct {
	MissingArchives      []string       `json:"missingArchives"`
	MissingArchivesTotal int            `json:"missingArchivesTotal"`
	UnnamedGenres        []string       `json:"unnamedGenres"`
	UnnamedGenresTotal   int            `json:"unnamedGenresTotal"`
	SkippedMalformed     int            `json:"skippedMalformed"`
	SkippedNoLibid       int            `json:"skippedNoLibid"`
	Encodings            EncodingsDTO   `json:"encodings"`
	GenreNamesMapped     int            `json:"genreNamesMapped"`
	PhasesMs             map[string]int `json:"phasesMs,omitempty"`
}

// EncodingsDTO is encoding stats for the UI.
type EncodingsDTO struct {
	UTF8           int    `json:"utf8"`
	CP1251         int    `json:"cp1251"`
	VersionInfo    string `json:"versionInfo,omitempty"`
	CollectionInfo string `json:"collectionInfo,omitempty"`
}

// ToDTO maps a stored report to the Wails payload.
func (r Report) ToDTO() ReportDTO {
	return ReportDTO{
		ID:                  r.ID,
		Status:              r.Status,
		INPXPath:            r.INPXPath,
		INPXVersion:         r.INPXVersion,
		FinishedAt:          r.FinishedAt,
		RecordsSeen:         r.RecordsSeen,
		WorksAdded:          r.WorksAdded,
		EditionsAdded:       r.EditionsAdded,
		EditionsUpdated:     r.EditionsUpdated,
		EditionsDeactivated: r.EditionsDeactivated,
		LibIDCollisions:     r.LibIDCollisions,
		Notes:               r.Notes.ToDTO(),
	}
}

// ToDTO maps stored notes to camelCase.
func (n Notes) ToDTO() NotesDTO {
	missing := n.MissingArchives
	if missing == nil {
		missing = []string{}
	}
	unnamed := n.UnnamedGenres
	if unnamed == nil {
		unnamed = []string{}
	}
	return NotesDTO{
		MissingArchives:      missing,
		MissingArchivesTotal: n.MissingArchivesTotal,
		UnnamedGenres:        unnamed,
		UnnamedGenresTotal:   n.UnnamedGenresTotal,
		SkippedMalformed:     n.SkippedMalformed,
		SkippedNoLibid:       n.SkippedNoLibID,
		Encodings:            encodingsDTO(n.Encodings),
		GenreNamesMapped:     n.GenreNamesMapped,
		PhasesMs:             n.PhasesMS,
	}
}

func encodingsDTO(e inpx.EncodingStats) EncodingsDTO {
	return EncodingsDTO{
		UTF8:           e.UTF8,
		CP1251:         e.CP1251,
		VersionInfo:    e.VersionInfo,
		CollectionInfo: e.CollectionInfo,
	}
}

// ReportFromBatch rebuilds a Report from a finished import_batches row.
func ReportFromBatch(id int64, status, path, version, finishedAt string, records, works, editionsAdded, editionsUpdated, deactivated, collisions int, notes Notes) Report {
	return Report{
		ID:                  id,
		Status:              status,
		INPXPath:            path,
		INPXVersion:         version,
		FinishedAt:          finishedAt,
		RecordsSeen:         records,
		WorksAdded:          works,
		EditionsAdded:       editionsAdded,
		EditionsUpdated:     editionsUpdated,
		EditionsDeactivated: deactivated,
		LibIDCollisions:     collisions,
		Notes:               notes,
	}
}
