package catalog

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/catalog/alphabet"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/repositories"
	"github.com/alexbelweb/flibustahub/internal/services/downloads"
	"github.com/alexbelweb/flibustahub/internal/textnorm"
)

// narrowBelow is the work_count under which listings drive from the relation
// table. Snapshot ms wide/narrow: 1→113/0, 100→184/1, 501→16/2, 1995→12/5,
// 4953→6/10, 8354→26/25, 9920→6/32, 19609→1/60, 81943→1/119. Threshold 5000
// is the measured 4953 point (6/10), not the 8354 crossing (26/25).
const narrowBelow = 5000

// Service lists and searches the catalog. Search has no side effects.
type Service struct {
	db  *db.DB
	cat *repositories.Catalog
	log *slog.Logger
	now func() time.Time
}

func New(catalogDB *db.DB, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		db:  catalogDB,
		cat: repositories.NewCatalog(catalogDB),
		log: log,
		now: time.Now,
	}
}

func (s *Service) ready() error {
	if s == nil || s.db == nil || s.cat == nil {
		return apperr.New(apperr.CodeDBOpenFailed, nil)
	}
	return nil
}

func (s *Service) ListWorks(ctx context.Context, q ListWorksQuery) (WorkPage, error) {
	if err := s.ready(); err != nil {
		return WorkPage{}, err
	}
	limit := q.Limit
	if limit <= 0 || limit > PageWorks {
		limit = PageWorks
	}
	sort := strings.TrimSpace(strings.ToLower(q.Sort))
	seriesName, visible, narrow, total, err := s.planList(ctx, q)
	if err != nil {
		return WorkPage{}, err
	}
	if q.SeriesID != 0 && seriesName == "" {
		return WorkPage{Total: total}, nil
	}
	if q.SeriesID != 0 && q.GenreID == 0 && q.AuthorID == 0 && (sort == "" || sort == "series" || sort == "seriesno") {
		sort = "seriesno"
	} else {
		sort = normalizeSort(q.Sort, q.Rated, q.Want)
	}
	p := repositories.WorkListParams{
		Sort:       sort,
		Lang:       strings.TrimSpace(q.Lang),
		GenreID:    q.GenreID,
		AuthorID:   q.AuthorID,
		SeriesID:   q.SeriesID,
		SeriesName: seriesName,
		Visible:    visible,
		Narrow:     narrow,
		Rated:      q.Rated,
		Want:       q.Want,
		Limit:      limit + 1,
	}
	switch sort {
	case "seriesno":
		if cur, ok := decodeCursor(q.Cursor, curSeriesNo); ok {
			p.HasCursor = true
			p.AfterTitle = cur.V
			p.AfterID = cur.ID
			p.AfterBucket = cur.B
			p.AfterNo = cur.N
		}
	case SortAdded:
		if cur, ok := decodeCursor(q.Cursor, curAdded); ok {
			p.HasCursor = true
			p.AfterAdded = cur.V
			p.AfterID = cur.ID
		}
	case SortRating:
		if cur, ok := decodeCursor(q.Cursor, curRating); ok {
			p.HasCursor = true
			p.AfterRating = cur.R
			p.AfterRateAt = cur.V
			p.AfterID = cur.ID
		}
	case SortRatedAt:
		if cur, ok := decodeCursor(q.Cursor, curRatedAt); ok {
			p.HasCursor = true
			p.AfterRateAt = cur.V
			p.AfterID = cur.ID
		}
	case SortWantAt:
		if cur, ok := decodeCursor(q.Cursor, curWantAt); ok {
			p.HasCursor = true
			p.AfterWantAt = cur.V
			p.AfterID = cur.ID
		}
	default:
		if cur, ok := decodeCursor(q.Cursor, curTitle); ok {
			p.HasCursor = true
			p.AfterTitle = cur.V
			p.AfterID = cur.ID
		}
	}

	var next string
	var ids []int64
	if sort == "seriesno" {
		keys, err := s.cat.ListSeriesVolumeKeys(ctx, p)
		if err != nil {
			return WorkPage{}, apperr.Wrap(apperr.CodeInternal, err, nil)
		}
		if len(keys) > limit {
			keys = keys[:limit]
			last := keys[len(keys)-1]
			next = encodeCursor(pageCursor{K: curSeriesNo, V: last.Title, ID: last.ID, B: last.Bucket, N: last.No})
		}
		ids = make([]int64, len(keys))
		for i, k := range keys {
			ids[i] = k.ID
		}
	} else {
		rows, err := s.cat.ListWorkIDs(ctx, p)
		if err != nil {
			return WorkPage{}, apperr.Wrap(apperr.CodeInternal, err, nil)
		}
		if len(rows) > limit {
			rows = rows[:limit]
			last := rows[len(rows)-1]
			switch sort {
			case SortAdded:
				next = encodeCursor(pageCursor{K: curAdded, V: last.AddedDate.String, ID: last.ID})
			case SortRating:
				rate := 0
				if last.Rating.Valid {
					rate = int(last.Rating.Int64)
				}
				next = encodeCursor(pageCursor{K: curRating, V: last.RatingAt.String, ID: last.ID, R: rate})
			case SortRatedAt:
				next = encodeCursor(pageCursor{K: curRatedAt, V: last.RatingAt.String, ID: last.ID})
			case SortWantAt:
				next = encodeCursor(pageCursor{K: curWantAt, V: last.WantAt.String, ID: last.ID})
			default:
				next = encodeCursor(pageCursor{K: curTitle, V: last.SortTitle, ID: last.ID})
			}
		}
		ids = make([]int64, len(rows))
		for i, r := range rows {
			ids[i] = r.ID
		}
	}
	items, err := s.hydrate(ctx, ids, seriesName)
	if err != nil {
		return WorkPage{}, err
	}
	return WorkPage{Items: items, NextCursor: next, Total: total}, nil
}

func (s *Service) planList(ctx context.Context, q ListWorksQuery) (seriesName string, visible, narrow bool, total *Total, err error) {
	visible = q.GenreID != 0 || q.SeriesID != 0
	driveCount := -1
	if q.GenreID != 0 {
		g, e := s.cat.Genre(ctx, q.GenreID)
		if e == sql.ErrNoRows {
			return "", visible, false, &Total{N: 0}, nil
		}
		if e != nil {
			return "", false, false, nil, apperr.Wrap(apperr.CodeInternal, e, nil)
		}
		driveCount = g.WorkCount
		if q.AuthorID == 0 && q.SeriesID == 0 && q.Lang == "" {
			total = &Total{N: g.WorkCount}
		}
	}
	if q.AuthorID != 0 {
		a, e := s.cat.Author(ctx, q.AuthorID)
		if e == sql.ErrNoRows {
			return "", visible, false, &Total{N: 0}, nil
		}
		if e != nil {
			return "", false, false, nil, apperr.Wrap(apperr.CodeInternal, e, nil)
		}
		if driveCount < 0 || a.WorkCount < driveCount {
			driveCount = a.WorkCount
		}
		if q.GenreID == 0 && q.SeriesID == 0 && q.Lang == "" {
			total = &Total{N: a.WorkCount}
		}
	}
	if q.SeriesID != 0 {
		ser, e := s.cat.SeriesByID(ctx, q.SeriesID)
		if e == sql.ErrNoRows {
			return "", visible, false, &Total{N: 0}, nil
		}
		if e != nil {
			return "", false, false, nil, apperr.Wrap(apperr.CodeInternal, e, nil)
		}
		seriesName = ser.Name
		if driveCount < 0 || ser.WorkCount < driveCount {
			driveCount = ser.WorkCount
		}
		if q.GenreID == 0 && q.AuthorID == 0 && q.Lang == "" {
			total = &Total{N: ser.WorkCount}
		}
	}
	if q.GenreID == 0 && q.AuthorID == 0 && q.SeriesID == 0 && q.Lang == "" && !q.Rated && !q.Want {
		if n, ok, e := s.cat.MetaInt(ctx, db.MetaWorksListable); e != nil {
			return "", false, false, nil, apperr.Wrap(apperr.CodeInternal, e, nil)
		} else if ok {
			total = &Total{N: n}
		}
	}
	filters := 0
	if q.GenreID != 0 {
		filters++
	}
	if q.AuthorID != 0 {
		filters++
	}
	if q.SeriesID != 0 {
		filters++
	}
	narrow = driveCount >= 0 && driveCount < narrowBelow
	if q.SeriesID != 0 && q.GenreID == 0 && q.AuthorID == 0 {
		// Series page always reads from editions: it must order by series_no.
		narrow = true
	}
	if filters > 1 || q.Lang != "" || q.Rated || q.Want {
		n, e := s.cat.CountWorksCapped(ctx, repositories.WorkListParams{
			Lang:       strings.TrimSpace(q.Lang),
			GenreID:    q.GenreID,
			AuthorID:   q.AuthorID,
			SeriesName: seriesName,
			Visible:    visible,
			Narrow:     narrow,
			Rated:      q.Rated,
			Want:       q.Want,
		}, SearchDepth+1)
		if e != nil {
			return "", false, false, nil, apperr.Wrap(apperr.CodeInternal, e, nil)
		}
		total = &Total{N: n, Capped: n > SearchDepth}
		if total.Capped {
			total.N = SearchDepth
		}
	}
	return seriesName, visible, narrow, total, nil
}

func (s *Service) Search(ctx context.Context, q SearchQuery) (SearchResult, error) {
	if err := s.ready(); err != nil {
		return SearchResult{}, err
	}
	tokens := Tokens(q.Q)
	if len(tokens) == 0 {
		page, err := s.ListWorks(ctx, ListWorksQuery{
			Sort: SortTitle, Lang: q.Lang, GenreID: q.GenreID, AuthorID: q.AuthorID, SeriesID: q.SeriesID,
			Rated: q.Rated, Want: q.Want, Limit: q.Limit,
		})
		return SearchResult{Works: page}, err
	}
	limit := q.Limit
	if limit <= 0 || limit > PageWorks {
		limit = PageWorks
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= SearchDepth {
		return SearchResult{Works: WorkPage{Total: &Total{N: SearchDepth, Capped: true}}}, nil
	}
	if offset+limit > SearchDepth {
		limit = SearchDepth - offset
	}

	seriesName := ""
	visible := q.GenreID != 0 || q.SeriesID != 0
	if q.SeriesID != 0 {
		ser, err := s.cat.SeriesByID(ctx, q.SeriesID)
		if err == sql.ErrNoRows {
			return SearchResult{Works: WorkPage{Total: &Total{N: 0}}}, nil
		}
		if err != nil {
			return SearchResult{}, apperr.Wrap(apperr.CodeInternal, err, nil)
		}
		seriesName = ser.Name
	}
	sp := repositories.WorkSearchParams{
		Lang:       strings.TrimSpace(q.Lang),
		GenreID:    q.GenreID,
		AuthorID:   q.AuthorID,
		SeriesName: seriesName,
		Visible:    visible,
		Rated:      q.Rated,
		Want:       q.Want,
		Offset:     offset,
		Limit:      limit,
		Like:       likePatterns(tokens),
	}

	useFTS := s.db.SearchIndexReady()
	var (
		ids      []int64
		fallback bool
		err      error
	)
	if useFTS {
		sp.FTS = ftsPhrase(tokens, worksFTSColumns)
		ids, err = s.cat.SearchWorkIDsFTS(ctx, sp)
		if err != nil {
			s.log.Warn("fts search failed, using like fallback", "err", err)
			fallback = true
			ids, err = s.cat.SearchWorkIDsLIKE(ctx, sp)
		}
	} else {
		s.log.Info("search index not ready, using like fallback")
		fallback = true
		ids, err = s.cat.SearchWorkIDsLIKE(ctx, sp)
	}
	if err != nil {
		return SearchResult{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	items, err := s.hydrate(ctx, ids, seriesName)
	if err != nil {
		return SearchResult{}, err
	}

	var total *Total
	if !fallback {
		capN := SearchDepth + 1
		var n int
		n, err = s.cat.CountWorksCappedFTS(ctx, sp, capN)
		if err != nil {
			return SearchResult{}, apperr.Wrap(apperr.CodeInternal, err, nil)
		}
		total = &Total{N: n, Capped: n > SearchDepth}
		if total.Capped {
			total.N = SearchDepth
		}
	}

	out := SearchResult{
		Works:    WorkPage{Items: items, Total: total},
		Fallback: fallback,
	}
	if offset == 0 {
		out.Authors, out.Series, out.AuthorsTotal, out.SeriesTotal = s.searchPeople(ctx, tokens, fallback || !useFTS)
	}
	return out, nil
}

func cappedSearchTotal(n int) *Total {
	t := &Total{N: n, Capped: n > SearchDepth}
	if t.Capped {
		t.N = SearchDepth
	}
	return t
}

func (s *Service) searchPeople(ctx context.Context, tokens []string, like bool) ([]Author, []Series, *Total, *Total) {
	fetchN := SearchPeoplePreview + 1
	authorMatch := ftsPhrase(tokens, "display_name sort_name")
	seriesMatch := ftsPhrase(tokens, "name")
	var aids, sids []int64
	var authorsTotal, seriesTotal *Total
	var err error
	if !like {
		aids, err = s.cat.SearchAuthorIDsFTS(ctx, authorMatch, fetchN)
		if err != nil {
			s.log.Warn("authors fts failed, using like fallback", "err", err)
			like = true
		} else {
			sids, err = s.cat.SearchSeriesIDsFTS(ctx, seriesMatch, fetchN)
			if err != nil {
				s.log.Warn("series fts failed, using like fallback", "err", err)
				like = true
			}
		}
	}
	if like {
		pats := likePatterns(tokens)
		aids, err = s.cat.SearchAuthorIDsLIKE(ctx, pats, fetchN)
		if err != nil {
			s.log.Warn("authors like search failed", "err", err)
			aids = nil
		}
		sids, err = s.cat.SearchSeriesIDsLIKE(ctx, pats, fetchN)
		if err != nil {
			s.log.Warn("series like search failed", "err", err)
			sids = nil
		}
		if len(aids) > SearchPeoplePreview {
			authorsTotal = &Total{N: SearchPeoplePreview, Capped: true}
		}
		if len(sids) > SearchPeoplePreview {
			seriesTotal = &Total{N: SearchPeoplePreview, Capped: true}
		}
	} else {
		if n, err := s.cat.CountAuthorsCappedFTS(ctx, authorMatch, SearchDepth+1); err != nil {
			s.log.Warn("authors count failed", "err", err)
		} else {
			authorsTotal = cappedSearchTotal(n)
		}
		if n, err := s.cat.CountSeriesCappedFTS(ctx, seriesMatch, SearchDepth+1); err != nil {
			s.log.Warn("series count failed", "err", err)
		} else {
			seriesTotal = cappedSearchTotal(n)
		}
	}
	if len(aids) > SearchPeoplePreview {
		aids = aids[:SearchPeoplePreview]
	}
	if len(sids) > SearchPeoplePreview {
		sids = sids[:SearchPeoplePreview]
	}
	arows, _ := s.cat.AuthorsByIDs(ctx, aids)
	srows, _ := s.cat.SeriesByIDs(ctx, sids)
	authors := make([]Author, len(arows))
	for i, a := range arows {
		authors[i] = Author{ID: a.ID, DisplayName: a.DisplayName, SortName: a.SortName, WorkCount: a.WorkCount}
	}
	series := make([]Series, len(srows))
	for i, r := range srows {
		series[i] = Series{ID: r.ID, Name: r.Name, SortName: r.SortName, WorkCount: r.WorkCount}
	}
	return authors, series, authorsTotal, seriesTotal
}

func (s *Service) ListAuthors(ctx context.Context, q ListPeopleQuery) (AuthorPage, error) {
	if err := s.ready(); err != nil {
		return AuthorPage{}, err
	}
	if q.Query != "" {
		tokens := Tokens(q.Query)
		if len(tokens) == 0 {
			return AuthorPage{}, nil
		}
		limit := q.Limit
		if limit <= 0 || limit > PageAuthors {
			limit = PageAuthors
		}
		ids, err := s.cat.SearchAuthorIDsFTS(ctx, ftsPhrase(tokens, "display_name sort_name"), limit)
		if err != nil {
			s.log.Warn("authors fts failed, using like fallback", "err", err)
			ids, err = s.cat.SearchAuthorIDsLIKE(ctx, likePatterns(tokens), limit)
		}
		if err != nil {
			return AuthorPage{}, apperr.Wrap(apperr.CodeInternal, err, nil)
		}
		rows, err := s.cat.AuthorsByIDs(ctx, ids)
		if err != nil {
			return AuthorPage{}, apperr.Wrap(apperr.CodeInternal, err, nil)
		}
		items := make([]Author, len(rows))
		for i, a := range rows {
			items[i] = Author{ID: a.ID, DisplayName: a.DisplayName, SortName: a.SortName, WorkCount: a.WorkCount}
		}
		return AuthorPage{Items: items}, nil
	}
	limit := q.Limit
	if limit <= 0 || limit > PageAuthors {
		limit = PageAuthors
	}
	letter := strings.ToLower(strings.TrimSpace(q.Letter))
	if letter != "" && !alphabet.Known(letter) {
		return AuthorPage{}, nil
	}
	p := repositories.NameListParams{Letter: letter, Limit: limit + 1}
	if cur, ok := decodeCursor(q.Cursor, curName); ok {
		p.HasCursor = true
		p.AfterName = cur.V
		p.AfterID = cur.ID
	}
	rows, err := s.cat.ListAuthors(ctx, p)
	if err != nil {
		return AuthorPage{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	var next string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		next = encodeCursor(pageCursor{K: curName, V: last.SortName, ID: last.ID})
	}
	items := make([]Author, len(rows))
	for i, a := range rows {
		items[i] = Author{ID: a.ID, DisplayName: a.DisplayName, SortName: a.SortName, WorkCount: a.WorkCount}
	}
	return AuthorPage{Items: items, NextCursor: next}, nil
}

func (s *Service) ListSeries(ctx context.Context, q ListPeopleQuery) (SeriesPage, error) {
	if err := s.ready(); err != nil {
		return SeriesPage{}, err
	}
	limit := q.Limit
	if limit <= 0 || limit > PageSeries {
		limit = PageSeries
	}
	if q.Query != "" {
		tokens := Tokens(q.Query)
		if len(tokens) == 0 {
			return SeriesPage{}, nil
		}
		ids, err := s.cat.SearchSeriesIDsFTS(ctx, ftsPhrase(tokens, "name"), limit)
		if err != nil {
			s.log.Warn("series fts failed, using like fallback", "err", err)
			ids, err = s.cat.SearchSeriesIDsLIKE(ctx, likePatterns(tokens), limit)
		}
		if err != nil {
			return SeriesPage{}, apperr.Wrap(apperr.CodeInternal, err, nil)
		}
		rows, err := s.cat.SeriesByIDs(ctx, ids)
		if err != nil {
			return SeriesPage{}, apperr.Wrap(apperr.CodeInternal, err, nil)
		}
		items := make([]Series, len(rows))
		for i, r := range rows {
			items[i] = Series{ID: r.ID, Name: r.Name, SortName: r.SortName, WorkCount: r.WorkCount}
		}
		return SeriesPage{Items: items}, nil
	}
	letter := strings.ToLower(strings.TrimSpace(q.Letter))
	if letter != "" && !alphabet.Known(letter) {
		return SeriesPage{}, nil
	}
	p := repositories.NameListParams{Letter: letter, Limit: limit + 1}
	if cur, ok := decodeCursor(q.Cursor, curName); ok {
		p.HasCursor = true
		p.AfterName = cur.V
		p.AfterID = cur.ID
	}
	rows, err := s.cat.ListSeries(ctx, p)
	if err != nil {
		return SeriesPage{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	var next string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		next = encodeCursor(pageCursor{K: curName, V: last.SortName, ID: last.ID})
	}
	items := make([]Series, len(rows))
	for i, r := range rows {
		items[i] = Series{ID: r.ID, Name: r.Name, SortName: r.SortName, WorkCount: r.WorkCount}
	}
	return SeriesPage{Items: items, NextCursor: next}, nil
}

func (s *Service) ListGenres(ctx context.Context, query string) ([]Genre, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	q := ""
	if t := Tokens(query); len(t) > 0 {
		q = t[0]
	}
	rows, err := s.cat.ListGenres(ctx, q, PageGenres)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	out := make([]Genre, len(rows))
	for i, g := range rows {
		out[i] = Genre{ID: g.ID, Code: g.Code, NameRU: g.NameRU, WorkCount: g.WorkCount}
	}
	return out, nil
}

func notFound(kind string) error {
	return apperr.New(apperr.CodeNotFound, map[string]string{"kind": kind})
}

func (s *Service) GetWork(ctx context.Context, id int64) (Work, error) {
	if err := s.ready(); err != nil {
		return Work{}, err
	}
	if id <= 0 {
		return Work{}, notFound("work")
	}
	items, err := s.hydrate(ctx, []int64{id}, "")
	if err != nil {
		return Work{}, err
	}
	if len(items) == 0 {
		return Work{}, notFound("work")
	}
	return items[0], nil
}

func (s *Service) GetWorkDetails(ctx context.Context, id int64) (WorkDetails, error) {
	work, err := s.GetWork(ctx, id)
	if err != nil {
		return WorkDetails{}, err
	}
	out := WorkDetails{Work: work}
	personal, perr := s.cat.Personal(ctx, id)
	if perr != nil && perr != sql.ErrNoRows {
		return WorkDetails{}, apperr.Wrap(apperr.CodeInternal, perr, nil)
	}
	if perr == nil && personal.Comment.Valid {
		c := personal.Comment.String
		out.Comment = &c
	}
	authors, err := s.cat.WorkAuthors(ctx, id)
	if err != nil {
		return WorkDetails{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	out.Authors = make([]Author, len(authors))
	for i, a := range authors {
		out.Authors[i] = Author{ID: a.ID, DisplayName: a.DisplayName, SortName: a.SortName, WorkCount: a.WorkCount}
	}
	genres, err := s.cat.WorkGenres(ctx, id)
	if err != nil {
		return WorkDetails{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	out.Genres = make([]Genre, len(genres))
	for i, g := range genres {
		out.Genres[i] = Genre{ID: g.ID, Code: g.Code, NameRU: g.NameRU, WorkCount: g.WorkCount}
	}
	if work.Series != "" {
		sid, err := s.cat.SeriesIDByName(ctx, work.Series)
		if err == nil {
			out.SeriesID = sid
			nb, nerr := s.cat.SeriesNeighbors(ctx, id, work.Series)
			if nerr != nil {
				return WorkDetails{}, apperr.Wrap(apperr.CodeInternal, nerr, nil)
			}
			if nb.HasPrev {
				v := nb.PrevID
				out.PrevWorkID = &v
			}
			if nb.HasNext {
				v := nb.NextID
				out.NextWorkID = &v
			}
		} else if err != sql.ErrNoRows {
			return WorkDetails{}, apperr.Wrap(apperr.CodeInternal, err, nil)
		}
	}
	ann, err := s.cat.Annotation(ctx, id)
	if err != nil && err != sql.ErrNoRows {
		return WorkDetails{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	out.AnnotationChecked = ann.CheckedAt.Valid
	if ann.Text.Valid {
		t := ann.Text.String
		out.Annotation = &t
	}
	if work.HasFile {
		ed, err := s.cat.PrimaryEdition(ctx, id)
		if err == nil {
			out.FileExt = strings.ToUpper(strings.TrimPrefix(strings.TrimSpace(ed.FileExt), "."))
		} else if err != sql.ErrNoRows {
			return WorkDetails{}, apperr.Wrap(apperr.CodeInternal, err, nil)
		}
		eds, err := s.cat.VisibleEditions(ctx, id)
		if err != nil {
			return WorkDetails{}, apperr.Wrap(apperr.CodeInternal, err, nil)
		}
		candidates := make([]downloads.Edition, 0, len(eds))
		out.Editions = make([]WorkEdition, len(eds))
		for i, e := range eds {
			item := WorkEdition{
				ID:          e.ID,
				ArchiveName: e.ArchiveName,
				FileName:    e.FileName,
				FileExt:     strings.ToUpper(strings.TrimPrefix(strings.TrimSpace(e.FileExt), ".")),
				AddedDate:   e.AddedDate,
			}
			if e.Size.Valid {
				v := e.Size.Int64
				item.Size = &v
			}
			out.Editions[i] = item
			candidates = append(candidates, downloads.Edition{
				ID: e.ID, FileExt: e.FileExt, AddedDate: e.AddedDate,
			})
		}
		if pref := downloads.Preferred(candidates); pref.ID != 0 {
			out.PreferredEditionID = pref.ID
			for i := range out.Editions {
				if out.Editions[i].ID == pref.ID {
					out.Editions[i].Preferred = true
					out.FileExt = out.Editions[i].FileExt
					if out.Editions[i].Size != nil {
						out.Size = out.Editions[i].Size
					}
					break
				}
			}
		}
	}
	return out, nil
}

func (s *Service) RecordViewed(ctx context.Context, id int64) error {
	if err := s.ready(); err != nil {
		return err
	}
	if id <= 0 {
		return notFound("work")
	}
	if err := s.cat.RecordViewed(ctx, id, s.now()); err != nil {
		return apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	return nil
}

func (s *Service) GetAuthor(ctx context.Context, id int64) (Author, error) {
	if err := s.ready(); err != nil {
		return Author{}, err
	}
	if id <= 0 {
		return Author{}, notFound("author")
	}
	a, err := s.cat.Author(ctx, id)
	if err == sql.ErrNoRows {
		return Author{}, notFound("author")
	}
	if err != nil {
		return Author{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	return Author{ID: a.ID, DisplayName: a.DisplayName, SortName: a.SortName, WorkCount: a.WorkCount}, nil
}

func (s *Service) GetGenre(ctx context.Context, id int64) (Genre, error) {
	if err := s.ready(); err != nil {
		return Genre{}, err
	}
	if id <= 0 {
		return Genre{}, notFound("genre")
	}
	g, err := s.cat.Genre(ctx, id)
	if err == sql.ErrNoRows {
		return Genre{}, notFound("genre")
	}
	if err != nil {
		return Genre{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	return Genre{ID: g.ID, Code: g.Code, NameRU: g.NameRU, WorkCount: g.WorkCount}, nil
}

func (s *Service) GetSeries(ctx context.Context, id int64) (Series, error) {
	if err := s.ready(); err != nil {
		return Series{}, err
	}
	if id <= 0 {
		return Series{}, notFound("series")
	}
	ser, err := s.cat.SeriesByID(ctx, id)
	if err == sql.ErrNoRows {
		return Series{}, notFound("series")
	}
	if err != nil {
		return Series{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	return Series{ID: ser.ID, Name: ser.Name, SortName: ser.SortName, WorkCount: ser.WorkCount}, nil
}

func (s *Service) RandomWork(ctx context.Context) (Work, error) {
	if err := s.ready(); err != nil {
		return Work{}, err
	}
	id, err := s.cat.RandomListableID(ctx)
	if err != nil {
		return Work{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	if id == 0 {
		return Work{}, nil
	}
	items, err := s.hydrate(ctx, []int64{id}, "")
	if err != nil || len(items) == 0 {
		return Work{}, err
	}
	return items[0], nil
}

func (s *Service) Alphabet() []string {
	return alphabet.Letters()
}

func (s *Service) RecordSearch(ctx context.Context, query string) error {
	if err := s.ready(); err != nil {
		return err
	}
	query = strings.TrimSpace(query)
	if query == "" || textnorm.Normalize(query) == "" {
		return nil
	}
	if err := s.cat.RecordHistory(ctx, query, s.now(), HistoryKeep); err != nil {
		return apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	return nil
}

func (s *Service) SearchHistory(ctx context.Context) ([]string, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.cat.ListHistory(ctx, HistoryRecent)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Query
	}
	return out, nil
}

func (s *Service) ClearSearchHistory(ctx context.Context) error {
	if err := s.ready(); err != nil {
		return err
	}
	if err := s.cat.ClearHistory(ctx); err != nil {
		return apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	return nil
}

func (s *Service) hydrate(ctx context.Context, ids []int64, seriesHint string) ([]Work, error) {
	rows, err := s.cat.HydrateWorks(ctx, ids, seriesHint)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	out := make([]Work, len(rows))
	for i, r := range rows {
		out[i] = workFromRow(r)
	}
	return out, nil
}

func workFromRow(r repositories.WorkRow) Work {
	w := Work{
		ID:           r.ID,
		WorkKey:      r.WorkKey,
		Title:        r.Title,
		SortTitle:    r.SortTitle,
		AuthorsText:  r.AuthorsText,
		EditionCount: r.EditionCount,
		HasFile:      r.EditionCount > 0,
	}
	if r.Lang.Valid {
		w.Lang = r.Lang.String
	}
	if r.Rating.Valid {
		v := int(r.Rating.Int64)
		w.Rating = &v
	}
	w.WantToRead = r.WantToRead == 1
	if r.AddedDate.Valid {
		w.AddedDate = r.AddedDate.String
	}
	if r.Series.Valid {
		w.Series = r.Series.String
	}
	if r.SeriesNo.Valid {
		w.SeriesNo = r.SeriesNo.String
	}
	if r.Size.Valid {
		v := r.Size.Int64
		w.Size = &v
	}
	if r.Librate.Valid {
		v := int(r.Librate.Int64)
		w.Librate = &v
	}
	return w
}
