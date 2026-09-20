package catalog

import (
	"context"
	"database/sql"
	"errors"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/db"
)

const (
	HomeCarousel   = 18
	HomeTagGenres  = 8
	HomeTagSeries  = 6
	HeroFromViewed = "viewed"
	HeroFromRandom = "random"
)

type HomeDashboard struct {
	WorksListable   int      `json:"worksListable"`
	AuthorsTotal    int      `json:"authorsTotal"`
	SeriesTotal     int      `json:"seriesTotal"`
	INPXVersion     string   `json:"inpxVersion,omitempty"`
	ImportedAt      string   `json:"importedAt,omitempty"`
	Hero            *Work    `json:"hero,omitempty"`
	HeroSource      string   `json:"heroSource,omitempty"`
	Arrivals        []Work   `json:"arrivals,omitempty"`
	Rated           []Work   `json:"rated,omitempty"`
	WantToReadCount int      `json:"wantToReadCount"`
	PopularGenres   []Genre  `json:"popularGenres,omitempty"`
	PopularSeries   []Series `json:"popularSeries,omitempty"`
}

func (s *Service) Home(ctx context.Context) (HomeDashboard, error) {
	if err := s.ready(); err != nil {
		return HomeDashboard{}, err
	}
	out := HomeDashboard{}
	if n, ok, err := s.cat.MetaInt(ctx, db.MetaWorksListable); err != nil {
		return HomeDashboard{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	} else if ok {
		out.WorksListable = n
	}
	if n, ok, err := s.cat.MetaInt(ctx, db.MetaAuthorsTotal); err != nil {
		return HomeDashboard{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	} else if ok {
		out.AuthorsTotal = n
	} else if err := s.db.Read.QueryRowContext(ctx, `SELECT count(*) FROM authors`).Scan(&out.AuthorsTotal); err != nil {
		return HomeDashboard{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	if n, ok, err := s.cat.MetaInt(ctx, db.MetaSeriesTotal); err != nil {
		return HomeDashboard{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	} else if ok {
		out.SeriesTotal = n
	} else if err := s.db.Read.QueryRowContext(ctx, `SELECT count(*) FROM series`).Scan(&out.SeriesTotal); err != nil {
		return HomeDashboard{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	if v, err := db.Meta(ctx, s.db.Read, "inpx_version"); err != nil {
		return HomeDashboard{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	} else {
		out.INPXVersion = v
	}
	if v, err := db.Meta(ctx, s.db.Read, db.MetaINPXImportedAt); err != nil {
		return HomeDashboard{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	} else {
		out.ImportedAt = v
	}

	if id, err := s.cat.LatestViewedID(ctx); err != nil {
		return HomeDashboard{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	} else if id > 0 {
		w, err := s.GetWork(ctx, id)
		switch {
		case err == nil:
			out.Hero = &w
			out.HeroSource = HeroFromViewed
		case isAbsentWork(err):
			// recently_viewed can outlive a deleted row; pick a random book below
		default:
			return HomeDashboard{}, err
		}
	}
	if out.Hero == nil {
		if w, err := s.RandomWork(ctx); err != nil && err != sql.ErrNoRows {
			return HomeDashboard{}, err
		} else if w.ID != 0 {
			out.Hero = &w
			out.HeroSource = HeroFromRandom
		}
	}

	arrivals, err := s.ListWorks(ctx, ListWorksQuery{Sort: SortAdded, Limit: HomeCarousel})
	if err != nil {
		return HomeDashboard{}, err
	}
	out.Arrivals = arrivals.Items

	rated, err := s.ListWorks(ctx, ListWorksQuery{Sort: SortRatedAt, Rated: true, Limit: HomeCarousel})
	if err != nil {
		return HomeDashboard{}, err
	}
	out.Rated = rated.Items

	wantN, err := s.cat.WantToReadCount(ctx)
	if err != nil {
		return HomeDashboard{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	out.WantToReadCount = wantN

	genres, err := s.cat.TopGenres(ctx, HomeTagGenres)
	if err != nil {
		return HomeDashboard{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	out.PopularGenres = make([]Genre, len(genres))
	for i, g := range genres {
		out.PopularGenres[i] = Genre{ID: g.ID, Code: g.Code, NameRU: g.NameRU, WorkCount: g.WorkCount}
	}
	series, err := s.cat.TopSeries(ctx, HomeTagSeries)
	if err != nil {
		return HomeDashboard{}, apperr.Wrap(apperr.CodeInternal, err, nil)
	}
	out.PopularSeries = make([]Series, len(series))
	for i, r := range series {
		out.PopularSeries[i] = Series{ID: r.ID, Name: r.Name, SortName: r.SortName, WorkCount: r.WorkCount}
	}
	return out, nil
}

func isAbsentWork(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	var typed *apperr.Error
	return errors.As(err, &typed) && typed.Code == apperr.CodeNotFound
}
