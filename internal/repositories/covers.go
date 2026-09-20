package repositories

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/alexbelweb/flibustahub/internal/visibility"
)

// VisibleEdition is one listable file of a work (archive, size, format).
type VisibleEdition struct {
	ID          int64
	ArchiveName string
	FileName    string
	FileExt     string
	Size        sql.NullInt64
	AddedDate   string
}

// EditionFile is a physical file used for covers, download or reading.
type EditionFile struct {
	ID          int64
	WorkID      int64
	Title       string
	AuthorsText string
	ArchiveName string
	FileName    string
	FileExt     string
	Size        sql.NullInt64
	AddedDate   string
	Active      bool
}

type AnnotationRow struct {
	Text      sql.NullString
	CheckedAt sql.NullString
}

type WarmupItem struct {
	WorkID      int64
	ArchiveName string
}

// PrimaryEdition returns the visible edition used for covers and annotations.
func (c *Catalog) PrimaryEdition(ctx context.Context, workID int64) (EditionFile, error) {
	var e EditionFile
	err := c.read.QueryRowContext(ctx, `
		SELECT e.archive_name, e.file_name, e.file_ext
		  FROM editions e
		 WHERE e.work_id = ? AND `+visibility.VisibleEditionSQL+`
		 ORDER BY e.added_date DESC, e.id DESC
		 LIMIT 1`, workID).Scan(&e.ArchiveName, &e.FileName, &e.FileExt)
	return e, err
}

func (c *Catalog) VisibleEditions(ctx context.Context, workID int64) ([]VisibleEdition, error) {
	rows, err := c.read.QueryContext(ctx, `
		SELECT e.id, e.archive_name, e.file_name, e.file_ext, e.size, COALESCE(e.added_date, '')
		  FROM editions e
		 WHERE e.work_id = ? AND `+visibility.VisibleEditionSQL+`
		 ORDER BY e.archive_name, e.id`, workID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []VisibleEdition
	for rows.Next() {
		var e VisibleEdition
		if err := rows.Scan(&e.ID, &e.ArchiveName, &e.FileName, &e.FileExt, &e.Size, &e.AddedDate); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Edition returns one edition joined with its work, including inactive rows.
func (c *Catalog) Edition(ctx context.Context, id int64) (EditionFile, error) {
	var e EditionFile
	var active int
	err := c.read.QueryRowContext(ctx, `
		SELECT e.id, e.work_id, w.title, w.authors_text, e.archive_name, e.file_name, e.file_ext,
		       e.size, COALESCE(e.added_date, ''), e.is_active
		  FROM editions e JOIN works w ON w.id = e.work_id
		 WHERE e.id = ?`, id).Scan(
		&e.ID, &e.WorkID, &e.Title, &e.AuthorsText, &e.ArchiveName, &e.FileName, &e.FileExt,
		&e.Size, &e.AddedDate, &active,
	)
	e.Active = active != 0
	return e, err
}

func (c *Catalog) Annotation(ctx context.Context, workID int64) (AnnotationRow, error) {
	var r AnnotationRow
	err := c.read.QueryRowContext(ctx, `SELECT annotation, annotation_checked_at FROM works WHERE id = ?`, workID).
		Scan(&r.Text, &r.CheckedAt)
	return r, err
}

func (c *Catalog) SaveAnnotation(ctx context.Context, workID int64, text *string, at time.Time) error {
	stamp := at.UTC().Format(time.RFC3339)
	var arg any
	if text != nil {
		s := strings.TrimSpace(*text)
		if s != "" {
			arg = s
		}
	}
	_, err := c.write.ExecContext(ctx, `UPDATE works SET annotation = ?, annotation_checked_at = ?, updated_at = ? WHERE id = ?`,
		arg, stamp, stamp, workID)
	return err
}

func (c *Catalog) RecordViewed(ctx context.Context, workID int64, at time.Time) error {
	stamp := at.UTC().Format(time.RFC3339)
	_, err := c.write.ExecContext(ctx, `
		INSERT INTO recently_viewed(work_id, viewed_at) VALUES (?, ?)
		ON CONFLICT(work_id) DO UPDATE SET viewed_at = excluded.viewed_at`, workID, stamp)
	return err
}

func (c *Catalog) WorkAuthors(ctx context.Context, workID int64) ([]AuthorRow, error) {
	rows, err := c.read.QueryContext(ctx, `
		SELECT a.id, a.display_name, a.sort_name, a.work_count
		  FROM work_authors wa
		  JOIN authors a ON a.id = wa.author_id
		 WHERE wa.work_id = ?
		 ORDER BY wa.position, a.id`, workID)
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

func (c *Catalog) WorkGenres(ctx context.Context, workID int64) ([]GenreRow, error) {
	rows, err := c.read.QueryContext(ctx, `
		SELECT g.id, g.code, g.name_ru, g.work_count
		  FROM work_genres wg
		  JOIN genres g ON g.id = wg.genre_id
		 WHERE wg.work_id = ?
		 ORDER BY g.name_ru, g.id`, workID)
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

func (c *Catalog) SeriesIDByName(ctx context.Context, name string) (int64, error) {
	var id int64
	err := c.read.QueryRowContext(ctx, `SELECT id FROM series WHERE name = ?`, name).Scan(&id)
	return id, err
}

type SeriesNeighbor struct {
	PrevID  int64
	NextID  int64
	HasPrev bool
	HasNext bool
}

func (c *Catalog) SeriesNeighbors(ctx context.Context, workID int64, seriesName string) (SeriesNeighbor, error) {
	var n SeriesNeighbor
	if strings.TrimSpace(seriesName) == "" {
		return n, nil
	}
	keys, err := c.ListSeriesVolumeKeys(ctx, WorkListParams{
		SeriesName: seriesName,
		Visible:    true,
		Limit:      100000,
	})
	if err != nil {
		return n, err
	}
	for i, k := range keys {
		if k.ID != workID {
			continue
		}
		if i > 0 {
			n.HasPrev = true
			n.PrevID = keys[i-1].ID
		}
		if i+1 < len(keys) {
			n.HasNext = true
			n.NextID = keys[i+1].ID
		}
		break
	}
	return n, nil
}

const warmupAdded = 500

// WarmupItems lists rated/noted works plus the newest warmupAdded listable works,
// each with the archive of its primary edition. Ordered by archive then work id.
func (c *Catalog) WarmupItems(ctx context.Context) ([]WarmupItem, error) {
	q := `
WITH primary_ed AS (
  SELECT work_id, archive_name FROM (
    SELECT e.work_id, e.archive_name,
           row_number() OVER (PARTITION BY e.work_id ORDER BY e.added_date DESC, e.id DESC) AS rn
      FROM editions e
     WHERE ` + visibility.VisibleEditionSQL + `
  ) WHERE rn = 1
),
marked AS (
  SELECT w.id AS work_id
    FROM works w
   WHERE w.rating IS NOT NULL
      OR (w.comment IS NOT NULL AND trim(w.comment) != '')
      OR w.want_to_read = 1
),
recent AS (
  SELECT w.id AS work_id
    FROM works w
   WHERE ` + visibility.ListableWorkSQL + `
   ORDER BY w.added_date DESC, w.id DESC
   LIMIT ?
)
SELECT p.work_id, p.archive_name
  FROM primary_ed p
  JOIN (
    SELECT work_id FROM marked
    UNION
    SELECT work_id FROM recent
  ) u ON u.work_id = p.work_id
 ORDER BY p.archive_name, p.work_id`
	rows, err := c.read.QueryContext(ctx, q, warmupAdded)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []WarmupItem
	for rows.Next() {
		var it WarmupItem
		if err := rows.Scan(&it.WorkID, &it.ArchiveName); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
