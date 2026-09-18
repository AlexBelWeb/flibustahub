package catalog

import "strings"

const (
	SortTitle = "title"
	SortAdded = "added"

	PageWorks   = 50
	PageAuthors = 40
	PageSeries  = 50
	PageGenres  = 500

	SearchDepth   = 500
	HistoryKeep   = 200
	HistoryRecent = 10
)

// Total is a cheap or capped count. Nil on a page means the UI must not show N.
type Total struct {
	N      int  `json:"n"`
	Capped bool `json:"capped,omitempty"`
}

// Work is one catalog card. HasFile is false for ghost cards (LISTABLE without a visible edition).
type Work struct {
	ID           int64  `json:"id"`
	WorkKey      string `json:"workKey"`
	Title        string `json:"title"`
	SortTitle    string `json:"sortTitle"`
	AuthorsText  string `json:"authorsText"`
	Lang         string `json:"lang,omitempty"`
	Rating       *int   `json:"rating,omitempty"`
	AddedDate    string `json:"addedDate,omitempty"`
	Series       string `json:"series,omitempty"`
	SeriesNo     string `json:"seriesNo,omitempty"`
	EditionCount int    `json:"editionCount"`
	HasFile      bool   `json:"hasFile"`
	Size         *int64 `json:"size,omitempty"`
	Librate      *int   `json:"librate,omitempty"`
}

type Author struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"displayName"`
	SortName    string `json:"sortName"`
	WorkCount   int    `json:"workCount"`
}

type Series struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SortName  string `json:"sortName"`
	WorkCount int    `json:"workCount"`
}

type Genre struct {
	ID        int64  `json:"id"`
	Code      string `json:"code"`
	NameRU    string `json:"nameRu"`
	WorkCount int    `json:"workCount"`
}

type WorkPage struct {
	Items      []Work `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
	Total      *Total `json:"total,omitempty"`
}

type AuthorPage struct {
	Items      []Author `json:"items"`
	NextCursor string   `json:"nextCursor,omitempty"`
	Total      *Total   `json:"total,omitempty"`
}

type SeriesPage struct {
	Items      []Series `json:"items"`
	NextCursor string   `json:"nextCursor,omitempty"`
	Total      *Total   `json:"total,omitempty"`
}

type ListWorksQuery struct {
	Sort     string
	Cursor   string
	Lang     string
	GenreID  int64
	AuthorID int64
	SeriesID int64
	Limit    int
}

type ListPeopleQuery struct {
	Letter string
	Query  string
	Cursor string
	Limit  int
}

type SearchQuery struct {
	Q        string
	Lang     string
	GenreID  int64
	AuthorID int64
	SeriesID int64
	Offset   int
	Limit    int
}

type SearchResult struct {
	Authors  []Author `json:"authors,omitempty"`
	Series   []Series `json:"series,omitempty"`
	Works    WorkPage `json:"works"`
	Fallback bool     `json:"fallback,omitempty"`
}

func normalizeSort(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == SortAdded {
		return SortAdded
	}
	return SortTitle
}
