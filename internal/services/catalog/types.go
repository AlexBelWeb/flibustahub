package catalog

import "strings"

const (
	SortTitle   = "title"
	SortAdded   = "added"
	SortRating  = "rating"
	SortRatedAt = "ratedat"
	SortWantAt  = "wantat"

	PageWorks   = 50
	PageAuthors = 40
	PageSeries  = 50
	PageGenres  = 500

	SearchDepth   = 500
	HistoryKeep   = 200
	HistoryRecent = 10

	SearchPeoplePreview = 3
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
	WantToRead   bool   `json:"wantToRead,omitempty"`
	AddedDate    string `json:"addedDate,omitempty"`
	Series       string `json:"series,omitempty"`
	SeriesNo     string `json:"seriesNo,omitempty"`
	EditionCount int    `json:"editionCount"`
	HasFile      bool   `json:"hasFile"`
	Size         *int64 `json:"size,omitempty"`
	Librate      *int   `json:"librate,omitempty"`
}

type WorkDetails struct {
	Work
	Comment            *string       `json:"comment,omitempty"`
	Authors            []Author      `json:"authors,omitempty"`
	Genres             []Genre       `json:"genres,omitempty"`
	SeriesID           int64         `json:"seriesId,omitempty"`
	PrevWorkID         *int64        `json:"prevWorkId,omitempty"`
	NextWorkID         *int64        `json:"nextWorkId,omitempty"`
	Annotation         *string       `json:"annotation,omitempty"`
	AnnotationChecked  bool          `json:"annotationChecked,omitempty"`
	FileExt            string        `json:"fileExt,omitempty"`
	PreferredEditionID int64         `json:"preferredEditionId,omitempty"`
	Editions           []WorkEdition `json:"editions,omitempty"`
}

type WorkEdition struct {
	ID          int64  `json:"id"`
	ArchiveName string `json:"archiveName"`
	FileName    string `json:"fileName,omitempty"`
	FileExt     string `json:"fileExt,omitempty"`
	Size        *int64 `json:"size,omitempty"`
	AddedDate   string `json:"addedDate,omitempty"`
	Preferred   bool   `json:"preferred,omitempty"`
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
	Sort     string `json:"sort"`
	Cursor   string `json:"cursor"`
	Lang     string `json:"lang"`
	GenreID  int64  `json:"genreId"`
	AuthorID int64  `json:"authorId"`
	SeriesID int64  `json:"seriesId"`
	Rated    bool   `json:"rated"`
	Want     bool   `json:"want"`
	Limit    int    `json:"limit"`
}

type ListPeopleQuery struct {
	Letter string `json:"letter"`
	Query  string `json:"query"`
	Cursor string `json:"cursor"`
	Limit  int    `json:"limit"`
}

type SearchQuery struct {
	Q        string `json:"q"`
	Lang     string `json:"lang"`
	GenreID  int64  `json:"genreId"`
	AuthorID int64  `json:"authorId"`
	SeriesID int64  `json:"seriesId"`
	Rated    bool   `json:"rated"`
	Want     bool   `json:"want"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
}

type SearchResult struct {
	Authors      []Author `json:"authors,omitempty"`
	Series       []Series `json:"series,omitempty"`
	AuthorsTotal *Total   `json:"authorsTotal,omitempty"`
	SeriesTotal  *Total   `json:"seriesTotal,omitempty"`
	Works        WorkPage `json:"works"`
	Fallback     bool     `json:"fallback,omitempty"`
}

func normalizeSort(s string, rated, want bool) string {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case SortAdded:
		return SortAdded
	case SortRating:
		if rated {
			return SortRating
		}
	case SortRatedAt:
		if rated {
			return SortRatedAt
		}
	case SortWantAt:
		if want {
			return SortWantAt
		}
	}
	return SortTitle
}
