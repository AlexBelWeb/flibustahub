package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/visibility"
)

// Catalog is the read/write persistence for listings, search and history.
type Catalog struct {
	read  *sql.DB
	write *sql.DB
}

func NewCatalog(d *db.DB) *Catalog {
	if d == nil {
		return nil
	}
	return &Catalog{read: d.Read, write: d.Write}
}

const seriesNoNumeric = `e.series_no GLOB '[0-9]*' AND e.series_no NOT GLOB '*[^0-9]*'`

type WorkRow struct {
	ID           int64
	WorkKey      string
	Title        string
	SortTitle    string
	AuthorsText  string
	Lang         sql.NullString
	Rating       sql.NullInt64
	AddedDate    sql.NullString
	Series       sql.NullString
	SeriesNo     sql.NullString
	EditionCount int
	Size         sql.NullInt64
	Librate      sql.NullInt64
}

type AuthorRow struct {
	ID          int64
	DisplayName string
	SortName    string
	WorkCount   int
}

type SeriesRow struct {
	ID        int64
	Name      string
	SortName  string
	WorkCount int
}

type GenreRow struct {
	ID        int64
	Code      string
	NameRU    string
	WorkCount int
}

type WorkListParams struct {
	Sort        string
	Lang        string
	GenreID     int64
	AuthorID    int64
	SeriesID    int64
	SeriesName  string
	Visible     bool
	Narrow      bool
	AfterTitle  string
	AfterAdded  string
	AfterID     int64
	HasCursor   bool
	AfterBucket int
	AfterNo     *float64
	Limit       int
}

type SeriesVolumeMeta struct {
	ID     int64
	Title  string
	Bucket int
	No     *float64
}

func (c *Catalog) MetaInt(ctx context.Context, key string) (int, bool, error) {
	var v string
	err := c.read.QueryRowContext(ctx, `SELECT value FROM app_meta WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	var n int
	if _, convErr := fmt.Sscanf(v, "%d", &n); convErr != nil {
		return 0, false, nil
	}
	return n, true, nil
}

func (c *Catalog) Genre(ctx context.Context, id int64) (GenreRow, error) {
	var g GenreRow
	err := c.read.QueryRowContext(ctx, `SELECT id, code, name_ru, work_count FROM genres WHERE id = ?`, id).
		Scan(&g.ID, &g.Code, &g.NameRU, &g.WorkCount)
	return g, err
}

func (c *Catalog) Author(ctx context.Context, id int64) (AuthorRow, error) {
	var a AuthorRow
	err := c.read.QueryRowContext(ctx, `SELECT id, display_name, sort_name, work_count FROM authors WHERE id = ?`, id).
		Scan(&a.ID, &a.DisplayName, &a.SortName, &a.WorkCount)
	return a, err
}

func (c *Catalog) SeriesByID(ctx context.Context, id int64) (SeriesRow, error) {
	var s SeriesRow
	err := c.read.QueryRowContext(ctx, `SELECT id, name, sort_name, work_count FROM series WHERE id = ?`, id).
		Scan(&s.ID, &s.Name, &s.SortName, &s.WorkCount)
	return s, err
}

func workPresenceSQL(alias string, visible bool) string {
	if visible {
		return `EXISTS (SELECT 1 FROM editions e WHERE e.work_id = ` + alias + `.id AND ` + visibility.VisibleEditionSQL + `)`
	}
	return strings.ReplaceAll(visibility.ListableWorkSQL, "w.", alias+".")
}

func (c *Catalog) ListWorkIDs(ctx context.Context, p WorkListParams) ([]WorkRow, error) {
	if p.Limit <= 0 {
		p.Limit = 50
	}
	if p.Sort == "seriesno" {
		keys, err := c.ListSeriesVolumeKeys(ctx, p)
		if err != nil {
			return nil, err
		}
		out := make([]WorkRow, len(keys))
		for i, k := range keys {
			out[i] = WorkRow{ID: k.ID, SortTitle: k.Title}
		}
		return out, nil
	}
	q, args := WorkListSQL(p)
	rows, err := c.read.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []WorkRow
	for rows.Next() {
		var r WorkRow
		if err := rows.Scan(&r.ID, &r.SortTitle, &r.AddedDate); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// WorkListSQL is the listing query ListWorkIDs runs. Tests EXPLAIN this SQL, not a copy.
func WorkListSQL(p WorkListParams) (string, []any) {
	from, args := workListFromWhere(p)
	var b strings.Builder
	if p.Narrow && p.SeriesName != "" {
		b.WriteString(`SELECT DISTINCT w.id, w.sort_title, w.added_date`)
	} else {
		b.WriteString(`SELECT w.id, w.sort_title, w.added_date`)
	}
	b.WriteString(from)
	if p.Sort == "added" {
		if p.HasCursor {
			// Leftmost bound seeks idx_works_added; the tuple is the keyset predicate.
			b.WriteString(` AND w.added_date <= ? AND (w.added_date, w.id) < (?, ?)`)
			args = append(args, p.AfterAdded, p.AfterAdded, p.AfterID)
		}
		b.WriteString(` ORDER BY w.added_date DESC, w.id DESC LIMIT ?`)
	} else {
		if p.HasCursor {
			b.WriteString(` AND w.sort_title >= ? AND (w.sort_title, w.id) > (?, ?)`)
			args = append(args, p.AfterTitle, p.AfterTitle, p.AfterID)
		}
		b.WriteString(` ORDER BY w.sort_title, w.id LIMIT ?`)
	}
	args = append(args, p.Limit)
	return b.String(), args
}

func (c *Catalog) CountWorksCapped(ctx context.Context, p WorkListParams, capN int) (int, error) {
	if capN <= 0 {
		capN = 501
	}
	from, args := workListFromWhere(p)
	inner := `SELECT w.id`
	if p.Narrow && p.SeriesName != "" {
		inner = `SELECT DISTINCT w.id`
	}
	q := `SELECT count(*) FROM (` + inner + from + ` LIMIT ?)`
	args = append(args, capN)
	return c.scanCount(ctx, q, args...)
}

func workListFromWhere(p WorkListParams) (string, []any) {
	var b strings.Builder
	args := make([]any, 0, 8)
	switch {
	case p.Narrow && p.GenreID != 0:
		b.WriteString(` FROM work_genres wg JOIN works w ON w.id = wg.work_id WHERE wg.genre_id = ?`)
		args = append(args, p.GenreID)
	case p.Narrow && p.AuthorID != 0:
		b.WriteString(` FROM work_authors wa JOIN works w ON w.id = wa.work_id WHERE wa.author_id = ?`)
		args = append(args, p.AuthorID)
	case p.Narrow && p.SeriesName != "":
		b.WriteString(` FROM editions e JOIN works w ON w.id = e.work_id WHERE e.series = ? AND ` + visibility.VisibleEditionSQL)
		args = append(args, p.SeriesName)
	default:
		b.WriteString(` FROM works w WHERE 1=1`)
	}
	b.WriteString(` AND ` + workPresenceSQL("w", p.Visible))
	if p.Lang != "" {
		b.WriteString(` AND w.lang = ?`)
		args = append(args, p.Lang)
	}
	if !p.Narrow && p.GenreID != 0 {
		b.WriteString(` AND EXISTS (SELECT 1 FROM work_genres wg WHERE wg.work_id = w.id AND wg.genre_id = ?)`)
		args = append(args, p.GenreID)
	}
	if !p.Narrow && p.AuthorID != 0 {
		b.WriteString(` AND EXISTS (SELECT 1 FROM work_authors wa WHERE wa.work_id = w.id AND wa.author_id = ?)`)
		args = append(args, p.AuthorID)
	}
	if !p.Narrow && p.SeriesName != "" {
		b.WriteString(` AND EXISTS (SELECT 1 FROM editions e WHERE e.work_id = w.id AND e.series = ? AND ` + visibility.VisibleEditionSQL + `)`)
		args = append(args, p.SeriesName)
	}
	return b.String(), args
}

func (c *Catalog) ListSeriesVolumeKeys(ctx context.Context, p WorkListParams) ([]SeriesVolumeMeta, error) {
	q := `
SELECT w.id, w.sort_title,
       MIN(CASE WHEN ` + seriesNoNumeric + ` THEN 0 ELSE 1 END) AS bucket,
       MIN(CASE WHEN ` + seriesNoNumeric + ` THEN CAST(e.series_no AS REAL) END) AS sn
  FROM editions e
  JOIN works w ON w.id = e.work_id
 WHERE e.series = ? AND ` + visibility.VisibleEditionSQL
	args := []any{p.SeriesName}
	if p.Lang != "" {
		q += ` AND w.lang = ?`
		args = append(args, p.Lang)
	}
	q += ` GROUP BY w.id, w.sort_title`
	if p.HasCursor {
		q += ` HAVING (MIN(CASE WHEN ` + seriesNoNumeric + ` THEN 0 ELSE 1 END) > ?)
            OR (MIN(CASE WHEN ` + seriesNoNumeric + ` THEN 0 ELSE 1 END) = ? AND ifnull(MIN(CASE WHEN ` + seriesNoNumeric + ` THEN CAST(e.series_no AS REAL) END), 1e100) > ?)
            OR (MIN(CASE WHEN ` + seriesNoNumeric + ` THEN 0 ELSE 1 END) = ? AND ifnull(MIN(CASE WHEN ` + seriesNoNumeric + ` THEN CAST(e.series_no AS REAL) END), 1e100) = ? AND (w.sort_title > ? OR (w.sort_title = ? AND w.id > ?)))`
		sn := 1e100
		if p.AfterNo != nil {
			sn = *p.AfterNo
		}
		args = append(args, p.AfterBucket, p.AfterBucket, sn, p.AfterBucket, sn, p.AfterTitle, p.AfterTitle, p.AfterID)
	}
	q += ` ORDER BY bucket, ifnull(sn, 1e100), w.sort_title, w.id LIMIT ?`
	args = append(args, p.Limit)
	rs, err := c.read.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rs.Close() }()
	var out []SeriesVolumeMeta
	for rs.Next() {
		var m SeriesVolumeMeta
		var sn sql.NullFloat64
		if err := rs.Scan(&m.ID, &m.Title, &m.Bucket, &sn); err != nil {
			return nil, err
		}
		if sn.Valid {
			v := sn.Float64
			m.No = &v
		}
		out = append(out, m)
	}
	return out, rs.Err()
}

func (c *Catalog) HydrateWorks(ctx context.Context, ids []int64, seriesHint string) ([]WorkRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	ph := make([]string, len(ids))
	args := make([]any, 0, len(ids)+2)
	if seriesHint != "" {
		args = append(args, seriesHint, seriesHint)
	}
	for i, id := range ids {
		ph[i] = "?"
		args = append(args, id)
	}
	seriesPick := `SELECT e.series FROM editions e WHERE e.work_id = w.id AND ` + visibility.VisibleEditionSQL + `
	   AND e.series IS NOT NULL AND trim(e.series) != '' ORDER BY e.id LIMIT 1`
	seriesNoPick := `SELECT e.series_no FROM editions e WHERE e.work_id = w.id AND ` + visibility.VisibleEditionSQL + `
	   AND e.series IS NOT NULL AND trim(e.series) != '' ORDER BY e.id LIMIT 1`
	if seriesHint != "" {
		seriesPick = `SELECT ?`
		seriesNoPick = `SELECT e.series_no FROM editions e WHERE e.work_id = w.id AND e.series = ? AND ` + visibility.VisibleEditionSQL + `
		 ORDER BY CASE WHEN ` + seriesNoNumeric + ` THEN 0 ELSE 1 END, CAST(e.series_no AS REAL), e.id LIMIT 1`
	}
	q := `SELECT w.id, w.work_key, w.title, w.sort_title, w.authors_text, w.lang, w.rating, w.added_date,
	        (` + seriesPick + `),
	        (` + seriesNoPick + `),
	        (SELECT count(*) FROM editions e WHERE e.work_id = w.id AND ` + visibility.VisibleEditionSQL + `),
	        (SELECT max(e.size) FROM editions e WHERE e.work_id = w.id AND ` + visibility.VisibleEditionSQL + `),
	        (SELECT max(e.librate) FROM editions e WHERE e.work_id = w.id AND ` + visibility.VisibleEditionSQL + `)
	   FROM works w
	  WHERE w.id IN (` + strings.Join(ph, ",") + `)`
	rows, err := c.read.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	byID := make(map[int64]WorkRow, len(ids))
	for rows.Next() {
		var r WorkRow
		if err := rows.Scan(&r.ID, &r.WorkKey, &r.Title, &r.SortTitle, &r.AuthorsText, &r.Lang, &r.Rating, &r.AddedDate,
			&r.Series, &r.SeriesNo, &r.EditionCount, &r.Size, &r.Librate); err != nil {
			return nil, err
		}
		byID[r.ID] = r
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]WorkRow, 0, len(ids))
	for _, id := range ids {
		if r, ok := byID[id]; ok {
			out = append(out, r)
		}
	}
	return out, nil
}
