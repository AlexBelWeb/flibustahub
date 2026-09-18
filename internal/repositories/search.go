package repositories

import (
	"context"
	"strings"

	"github.com/alexbelweb/flibustahub/internal/visibility"
)

type WorkSearchParams struct {
	FTS        string
	Like       []string
	Lang       string
	GenreID    int64
	AuthorID   int64
	SeriesName string
	Visible    bool
	Offset     int
	Limit      int
}

func (c *Catalog) SearchWorkIDsFTS(ctx context.Context, p WorkSearchParams) ([]int64, error) {
	q := `SELECT w.id FROM works_fts f
	 JOIN works w ON w.id = f.rowid
	 WHERE works_fts MATCH ? AND ` + workPresenceSQL("w", p.Visible)
	args := []any{p.FTS}
	q, args = appendWorkFilters(q, args, p)
	q += ` ORDER BY bm25(works_fts), w.id LIMIT ? OFFSET ?`
	args = append(args, p.Limit, p.Offset)
	return c.scanIDs(ctx, q, args...)
}

func (c *Catalog) SearchWorkIDsLIKE(ctx context.Context, p WorkSearchParams) ([]int64, error) {
	q := `SELECT w.id FROM works w WHERE ` + workPresenceSQL("w", p.Visible)
	args := []any{}
	for range p.Like {
		q += ` AND (' ' || w.sort_title) LIKE ?`
	}
	for _, pat := range p.Like {
		args = append(args, pat)
	}
	q, args = appendWorkFilters(q, args, p)
	q += ` ORDER BY w.sort_title, w.id LIMIT ? OFFSET ?`
	args = append(args, p.Limit, p.Offset)
	return c.scanIDs(ctx, q, args...)
}

func (c *Catalog) CountWorksCappedFTS(ctx context.Context, p WorkSearchParams, capN int) (int, error) {
	q := `SELECT count(*) FROM (
		SELECT w.id FROM works_fts f
		 JOIN works w ON w.id = f.rowid
		 WHERE works_fts MATCH ? AND ` + workPresenceSQL("w", p.Visible)
	args := []any{p.FTS}
	q, args = appendWorkFilters(q, args, p)
	q += ` LIMIT ?)`
	args = append(args, capN)
	return c.scanCount(ctx, q, args...)
}

func (c *Catalog) CountWorksCappedLIKE(ctx context.Context, p WorkSearchParams, capN int) (int, error) {
	q := `SELECT count(*) FROM (SELECT w.id FROM works w WHERE ` + workPresenceSQL("w", p.Visible)
	args := []any{}
	for range p.Like {
		q += ` AND (' ' || w.sort_title) LIKE ?`
	}
	for _, pat := range p.Like {
		args = append(args, pat)
	}
	q, args = appendWorkFilters(q, args, p)
	q += ` LIMIT ?)`
	args = append(args, capN)
	return c.scanCount(ctx, q, args...)
}

func appendWorkFilters(q string, args []any, p WorkSearchParams) (string, []any) {
	if p.Lang != "" {
		q += ` AND w.lang = ?`
		args = append(args, p.Lang)
	}
	if p.GenreID != 0 {
		q += ` AND EXISTS (SELECT 1 FROM work_genres wg WHERE wg.work_id = w.id AND wg.genre_id = ?)`
		args = append(args, p.GenreID)
	}
	if p.AuthorID != 0 {
		q += ` AND EXISTS (SELECT 1 FROM work_authors wa WHERE wa.work_id = w.id AND wa.author_id = ?)`
		args = append(args, p.AuthorID)
	}
	if p.SeriesName != "" {
		q += ` AND EXISTS (SELECT 1 FROM editions e WHERE e.work_id = w.id AND e.series = ? AND ` + visibility.VisibleEditionSQL + `)`
		args = append(args, p.SeriesName)
	}
	return q, args
}

func (c *Catalog) scanIDs(ctx context.Context, q string, args ...any) ([]int64, error) {
	rows, err := c.read.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (c *Catalog) scanCount(ctx context.Context, q string, args ...any) (int, error) {
	var n int
	err := c.read.QueryRowContext(ctx, q, args...).Scan(&n)
	return n, err
}

func (c *Catalog) SearchAuthorIDsFTS(ctx context.Context, match string, limit int) ([]int64, error) {
	return c.scanIDs(ctx, `SELECT a.id FROM authors_fts f
		JOIN authors a ON a.id = f.rowid
		WHERE authors_fts MATCH ? AND a.work_count > 0
		ORDER BY bm25(authors_fts) LIMIT ?`, match, limit)
}

func (c *Catalog) SearchAuthorIDsLIKE(ctx context.Context, patterns []string, limit int) ([]int64, error) {
	q := `SELECT a.id FROM authors a WHERE a.work_count > 0`
	args := []any{}
	for range patterns {
		q += ` AND (' ' || a.sort_name) LIKE ?`
	}
	for _, p := range patterns {
		args = append(args, p)
	}
	q += ` ORDER BY a.sort_name, a.id LIMIT ?`
	args = append(args, limit)
	return c.scanIDs(ctx, q, args...)
}

func (c *Catalog) SearchSeriesIDsFTS(ctx context.Context, match string, limit int) ([]int64, error) {
	return c.scanIDs(ctx, `SELECT s.id FROM series_fts f
		JOIN series s ON s.id = f.rowid
		WHERE series_fts MATCH ? AND s.work_count > 0
		ORDER BY bm25(series_fts) LIMIT ?`, match, limit)
}

func (c *Catalog) SearchSeriesIDsLIKE(ctx context.Context, patterns []string, limit int) ([]int64, error) {
	q := `SELECT s.id FROM series s WHERE s.work_count > 0`
	args := []any{}
	for range patterns {
		q += ` AND (' ' || s.sort_name) LIKE ?`
	}
	for _, p := range patterns {
		args = append(args, p)
	}
	q += ` ORDER BY s.sort_name, s.id LIMIT ?`
	args = append(args, limit)
	return c.scanIDs(ctx, q, args...)
}

func (c *Catalog) AuthorsByIDs(ctx context.Context, ids []int64) ([]AuthorRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	ph := strings.Repeat("?,", len(ids))
	ph = ph[:len(ph)-1]
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := c.read.QueryContext(ctx, `SELECT id, display_name, sort_name, work_count FROM authors WHERE id IN (`+ph+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	byID := map[int64]AuthorRow{}
	for rows.Next() {
		var a AuthorRow
		if err := rows.Scan(&a.ID, &a.DisplayName, &a.SortName, &a.WorkCount); err != nil {
			return nil, err
		}
		byID[a.ID] = a
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]AuthorRow, 0, len(ids))
	for _, id := range ids {
		if a, ok := byID[id]; ok {
			out = append(out, a)
		}
	}
	return out, nil
}

func (c *Catalog) SeriesByIDs(ctx context.Context, ids []int64) ([]SeriesRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	ph := strings.Repeat("?,", len(ids))
	ph = ph[:len(ph)-1]
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := c.read.QueryContext(ctx, `SELECT id, name, sort_name, work_count FROM series WHERE id IN (`+ph+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	byID := map[int64]SeriesRow{}
	for rows.Next() {
		var s SeriesRow
		if err := rows.Scan(&s.ID, &s.Name, &s.SortName, &s.WorkCount); err != nil {
			return nil, err
		}
		byID[s.ID] = s
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]SeriesRow, 0, len(ids))
	for _, id := range ids {
		if s, ok := byID[id]; ok {
			out = append(out, s)
		}
	}
	return out, nil
}
