package app

import (
	"context"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/services/catalog"
)

func (s *Service) catalogSvc() (*catalog.Service, error) {
	st := s.snap()
	if st.catalog == nil {
		return nil, apperr.New(apperr.CodeDBOpenFailed, nil)
	}
	return catalog.New(st.catalog, s.Logger()), nil
}

func (s *Service) ListWorks(ctx context.Context, q catalog.ListWorksQuery) (catalog.WorkPage, error) {
	c, err := s.catalogSvc()
	if err != nil {
		return catalog.WorkPage{}, err
	}
	return c.ListWorks(ctx, q)
}

func (s *Service) SearchCatalog(ctx context.Context, q catalog.SearchQuery) (catalog.SearchResult, error) {
	c, err := s.catalogSvc()
	if err != nil {
		return catalog.SearchResult{}, err
	}
	return c.Search(ctx, q)
}

func (s *Service) ListAuthors(ctx context.Context, q catalog.ListPeopleQuery) (catalog.AuthorPage, error) {
	c, err := s.catalogSvc()
	if err != nil {
		return catalog.AuthorPage{}, err
	}
	return c.ListAuthors(ctx, q)
}

func (s *Service) ListSeries(ctx context.Context, q catalog.ListPeopleQuery) (catalog.SeriesPage, error) {
	c, err := s.catalogSvc()
	if err != nil {
		return catalog.SeriesPage{}, err
	}
	return c.ListSeries(ctx, q)
}

func (s *Service) ListGenres(ctx context.Context, query string) ([]catalog.Genre, error) {
	c, err := s.catalogSvc()
	if err != nil {
		return nil, err
	}
	return c.ListGenres(ctx, query)
}

func (s *Service) RandomWork(ctx context.Context) (catalog.Work, error) {
	c, err := s.catalogSvc()
	if err != nil {
		return catalog.Work{}, err
	}
	return c.RandomWork(ctx)
}

func (s *Service) CatalogAlphabet() []string {
	st := s.snap()
	return catalog.New(st.catalog, s.Logger()).Alphabet()
}

func (s *Service) RecordSearch(ctx context.Context, query string) error {
	c, err := s.catalogSvc()
	if err != nil {
		return err
	}
	return c.RecordSearch(ctx, query)
}

func (s *Service) SearchHistory(ctx context.Context) ([]string, error) {
	c, err := s.catalogSvc()
	if err != nil {
		return nil, err
	}
	return c.SearchHistory(ctx)
}

func (s *Service) ClearSearchHistory(ctx context.Context) error {
	c, err := s.catalogSvc()
	if err != nil {
		return err
	}
	return c.ClearSearchHistory(ctx)
}
