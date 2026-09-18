package repositories

import (
	"context"
	"database/sql"

	"github.com/alexbelweb/flibustahub/internal/catalog/alphabet"
)

type NameListParams struct {
	Letter    string
	AfterName string
	AfterID   int64
	HasCursor bool
	Limit     int
}

func letterClause(column, letter string) (sqlFrag string, args []any, ok bool) {
	if letter == "" {
		return "", nil, true
	}
	if letter == alphabet.Other {
		return " AND " + alphabet.OtherSQL(column), nil, true
	}
	lo, hi, in := alphabet.PrefixRange(letter)
	if !in {
		return "", nil, false
	}
	return " AND " + column + " >= ? AND " + column + " < ?", []any{lo, hi}, true
}

func (c *Catalog) ListAuthors(ctx context.Context, p NameListParams) ([]AuthorRow, error) {
	q, args, ok := AuthorListSQL(p)
	if !ok {
		return nil, nil
	}
	return c.scanAuthors(ctx, q, args...)
}

func (c *Catalog) ListSeries(ctx context.Context, p NameListParams) ([]SeriesRow, error) {
	q, args, ok := SeriesListSQL(p)
	if !ok {
		return nil, nil
	}
	return c.scanSeries(ctx, q, args...)
}

// AuthorListSQL is the listing query ListAuthors runs. Tests EXPLAIN this SQL, not a copy.
func AuthorListSQL(p NameListParams) (string, []any, bool) {
	return nameListSQL(`SELECT id, display_name, sort_name, work_count FROM authors WHERE work_count > 0`, p)
}

// SeriesListSQL is the listing query ListSeries runs. Tests EXPLAIN this SQL, not a copy.
func SeriesListSQL(p NameListParams) (string, []any, bool) {
	return nameListSQL(`SELECT id, name, sort_name, work_count FROM series WHERE work_count > 0`, p)
}

func nameListSQL(head string, p NameListParams) (string, []any, bool) {
	q := head
	args := []any{}
	clause, a, ok := letterClause("sort_name", p.Letter)
	if !ok {
		return "", nil, false
	}
	q += clause
	args = append(args, a...)
	if p.HasCursor {
		q += ` AND sort_name >= ? AND (sort_name, id) > (?, ?)`
		args = append(args, p.AfterName, p.AfterName, p.AfterID)
	}
	q += ` ORDER BY sort_name, id LIMIT ?`
	args = append(args, p.Limit)
	return q, args, true
}

func (c *Catalog) ListGenres(ctx context.Context, nameQuery string, limit int) ([]GenreRow, error) {
	q := `SELECT id, code, name_ru, work_count FROM genres WHERE work_count > 0`
	args := []any{}
	if nameQuery != "" {
		q += ` AND (search_norm(name_ru) LIKE ? OR search_norm(code) LIKE ?)`
		pat := "% " + nameQuery + "%"
		args = append(args, pat, pat)
	}
	q += ` ORDER BY normalize(name_ru), id`
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := c.read.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []GenreRow
	for rows.Next() {
		var g GenreRow
		if err := rows.Scan(&g.ID, &g.Code, &g.NameRU, &g.WorkCount); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (c *Catalog) RandomListableID(ctx context.Context) (int64, error) {
	var maxID int64
	if err := c.read.QueryRowContext(ctx, `SELECT ifnull(max(id), 0) FROM works`).Scan(&maxID); err != nil {
		return 0, err
	}
	if maxID <= 0 {
		return 0, nil
	}
	var id int64
	err := c.read.QueryRowContext(ctx, `
SELECT w.id FROM works w
 WHERE `+workPresenceSQL("w", false)+`
   AND w.id >= abs(random() % ?)
 ORDER BY w.id LIMIT 1`, maxID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	err = c.read.QueryRowContext(ctx, `
SELECT w.id FROM works w WHERE `+workPresenceSQL("w", false)+` ORDER BY w.id LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

func (c *Catalog) scanAuthors(ctx context.Context, q string, args ...any) ([]AuthorRow, error) {
	rows, err := c.read.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []AuthorRow
	for rows.Next() {
		var a AuthorRow
		if err := rows.Scan(&a.ID, &a.DisplayName, &a.SortName, &a.WorkCount); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (c *Catalog) scanSeries(ctx context.Context, q string, args ...any) ([]SeriesRow, error) {
	rows, err := c.read.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []SeriesRow
	for rows.Next() {
		var s SeriesRow
		if err := rows.Scan(&s.ID, &s.Name, &s.SortName, &s.WorkCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
