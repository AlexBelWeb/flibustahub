package handlers

import (
	"context"

	"github.com/alexbelweb/flibustahub/internal/services/catalog"
)

func (a *App) ListWorks(q catalog.ListWorksQuery) (catalog.WorkPage, error) {
	return a.svc.ListWorks(context.Background(), q)
}

func (a *App) SearchCatalog(q catalog.SearchQuery) (catalog.SearchResult, error) {
	return a.svc.SearchCatalog(context.Background(), q)
}

func (a *App) ListAuthors(q catalog.ListPeopleQuery) (catalog.AuthorPage, error) {
	return a.svc.ListAuthors(context.Background(), q)
}

func (a *App) ListSeries(q catalog.ListPeopleQuery) (catalog.SeriesPage, error) {
	return a.svc.ListSeries(context.Background(), q)
}

func (a *App) ListGenres(query string) ([]catalog.Genre, error) {
	return a.svc.ListGenres(context.Background(), query)
}

func (a *App) GetWork(id int64) (catalog.Work, error) {
	return a.svc.GetWork(context.Background(), id)
}

func (a *App) GetAuthor(id int64) (catalog.Author, error) {
	return a.svc.GetAuthor(context.Background(), id)
}

func (a *App) GetGenre(id int64) (catalog.Genre, error) {
	return a.svc.GetGenre(context.Background(), id)
}

func (a *App) GetSeries(id int64) (catalog.Series, error) {
	return a.svc.GetSeries(context.Background(), id)
}

func (a *App) RandomWork() (catalog.Work, error) {
	return a.svc.RandomWork(context.Background())
}

func (a *App) CatalogAlphabet() []string {
	return a.svc.CatalogAlphabet()
}

func (a *App) RecordSearch(query string) error {
	return a.svc.RecordSearch(context.Background(), query)
}

func (a *App) SearchHistory() ([]string, error) {
	return a.svc.SearchHistory(context.Background())
}

func (a *App) ClearSearchHistory() error {
	return a.svc.ClearSearchHistory(context.Background())
}
