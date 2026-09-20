package repositories

import (
	"context"
	"database/sql"
	"strings"

	"github.com/alexbelweb/flibustahub/internal/visibility"
)

type PersonalRow struct {
	ID                  int64
	WorkKey             string
	Title               string
	AuthorsText         string
	Rating              sql.NullInt64
	Comment             sql.NullString
	WantToRead          int
	RatingUpdatedAt     sql.NullString
	CommentUpdatedAt    sql.NullString
	WantToReadUpdatedAt sql.NullString
	ExportedAt          sql.NullString
}

type PersonalSnapshot struct {
	UnsyncedCount int
	LastExportAt  string
}

func (c *Catalog) Personal(ctx context.Context, id int64) (PersonalRow, error) {
	var r PersonalRow
	err := c.read.QueryRowContext(ctx, `
SELECT id, work_key, title, authors_text, rating, comment, want_to_read,
       rating_updated_at, comment_updated_at, want_to_read_updated_at, exported_at
  FROM works WHERE id = ?`, id).Scan(
		&r.ID, &r.WorkKey, &r.Title, &r.AuthorsText, &r.Rating, &r.Comment, &r.WantToRead,
		&r.RatingUpdatedAt, &r.CommentUpdatedAt, &r.WantToReadUpdatedAt, &r.ExportedAt,
	)
	return r, err
}

func (c *Catalog) WorkByKey(ctx context.Context, key string) (PersonalRow, error) {
	var r PersonalRow
	err := c.read.QueryRowContext(ctx, `
SELECT id, work_key, title, authors_text, rating, comment, want_to_read,
       rating_updated_at, comment_updated_at, want_to_read_updated_at, exported_at
  FROM works WHERE work_key = ?`, key).Scan(
		&r.ID, &r.WorkKey, &r.Title, &r.AuthorsText, &r.Rating, &r.Comment, &r.WantToRead,
		&r.RatingUpdatedAt, &r.CommentUpdatedAt, &r.WantToReadUpdatedAt, &r.ExportedAt,
	)
	return r, err
}

func (c *Catalog) SetRating(ctx context.Context, id int64, rating *int, ts string) (bool, error) {
	var (
		res sql.Result
		err error
	)
	if rating == nil {
		res, err = c.write.ExecContext(ctx, `
UPDATE works SET rating = NULL, rating_updated_at = ?
 WHERE id = ? AND rating IS NOT NULL`, ts, id)
	} else {
		res, err = c.write.ExecContext(ctx, `
UPDATE works SET rating = ?, rating_updated_at = ?
 WHERE id = ? AND (rating IS NULL OR rating != ?)`, *rating, ts, id, *rating)
	}
	if err != nil {
		return false, err
	}
	return c.personalChanged(ctx, res, id)
}

func (c *Catalog) SetComment(ctx context.Context, id int64, comment string, ts string) (bool, error) {
	comment = strings.TrimSpace(comment)
	var (
		res sql.Result
		err error
	)
	if comment == "" {
		res, err = c.write.ExecContext(ctx, `
UPDATE works SET comment = NULL, comment_updated_at = ?
 WHERE id = ? AND comment IS NOT NULL`, ts, id)
	} else {
		res, err = c.write.ExecContext(ctx, `
UPDATE works SET comment = ?, comment_updated_at = ?
 WHERE id = ? AND (comment IS NULL OR comment != ?)`, comment, ts, id, comment)
	}
	if err != nil {
		return false, err
	}
	return c.personalChanged(ctx, res, id)
}

func (c *Catalog) SetWantToRead(ctx context.Context, id int64, want bool, ts string) (bool, error) {
	next := 0
	if want {
		next = 1
	}
	res, err := c.write.ExecContext(ctx, `
UPDATE works SET want_to_read = ?, want_to_read_updated_at = ?
 WHERE id = ? AND want_to_read != ?`, next, ts, id, next)
	if err != nil {
		return false, err
	}
	return c.personalChanged(ctx, res, id)
}

func (c *Catalog) personalChanged(ctx context.Context, res sql.Result, id int64) (bool, error) {
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	var one int
	err = c.write.QueryRowContext(ctx, `SELECT 1 FROM works WHERE id = ?`, id).Scan(&one)
	if err != nil {
		return false, err
	}
	return false, nil
}

func (c *Catalog) ExportableWorks(ctx context.Context) ([]PersonalRow, error) {
	q := `SELECT w.id, w.work_key, w.title, w.authors_text, w.rating, w.comment, w.want_to_read,
	             w.rating_updated_at, w.comment_updated_at, w.want_to_read_updated_at, w.exported_at
	        FROM works w
	       WHERE ` + visibility.ExportableWorkSQL + `
	       ORDER BY w.id`
	rows, err := c.read.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []PersonalRow
	for rows.Next() {
		var r PersonalRow
		if err := rows.Scan(&r.ID, &r.WorkKey, &r.Title, &r.AuthorsText, &r.Rating, &r.Comment, &r.WantToRead,
			&r.RatingUpdatedAt, &r.CommentUpdatedAt, &r.WantToReadUpdatedAt, &r.ExportedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (c *Catalog) PersonalSnapshot(ctx context.Context) (PersonalSnapshot, error) {
	var snap PersonalSnapshot
	err := c.read.QueryRowContext(ctx, `
SELECT count(*) FROM (
  SELECT id FROM works WHERE rating_updated_at IS NOT NULL AND (exported_at IS NULL OR rating_updated_at > exported_at)
  UNION
  SELECT id FROM works WHERE comment_updated_at IS NOT NULL AND (exported_at IS NULL OR comment_updated_at > exported_at)
  UNION
  SELECT id FROM works WHERE want_to_read_updated_at IS NOT NULL AND (exported_at IS NULL OR want_to_read_updated_at > exported_at)
)`).Scan(&snap.UnsyncedCount)
	if err != nil {
		return snap, err
	}
	var last sql.NullString
	if err := c.read.QueryRowContext(ctx, `SELECT max(exported_at) FROM works WHERE exported_at IS NOT NULL`).Scan(&last); err != nil {
		return snap, err
	}
	if last.Valid {
		snap.LastExportAt = last.String
	}
	return snap, nil
}

func (c *Catalog) MarkExported(ctx context.Context, rows []PersonalRow, ts string) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := c.write.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, `
UPDATE works SET exported_at = ?
 WHERE id = ?
   AND coalesce(rating_updated_at, '') = ?
   AND coalesce(comment_updated_at, '') = ?
   AND coalesce(want_to_read_updated_at, '') = ?`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()
	for _, r := range rows {
		if _, err := stmt.ExecContext(ctx, ts, r.ID, nullStamp(r.RatingUpdatedAt), nullStamp(r.CommentUpdatedAt), nullStamp(r.WantToReadUpdatedAt)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func nullStamp(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

func (c *Catalog) WithWriteTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := c.write.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func ApplyPersonalTx(ctx context.Context, tx *sql.Tx, id int64, rating *int, ratingTS string, comment *string, commentTS string, want *bool, wantTS string) error {
	sets := make([]string, 0, 6)
	args := make([]any, 0, 7)
	if ratingTS != "" {
		if rating == nil {
			sets = append(sets, `rating = NULL`, `rating_updated_at = ?`)
			args = append(args, ratingTS)
		} else {
			sets = append(sets, `rating = ?`, `rating_updated_at = ?`)
			args = append(args, *rating, ratingTS)
		}
	}
	if commentTS != "" {
		if comment == nil || *comment == "" {
			sets = append(sets, `comment = NULL`, `comment_updated_at = ?`)
			args = append(args, commentTS)
		} else {
			sets = append(sets, `comment = ?`, `comment_updated_at = ?`)
			args = append(args, *comment, commentTS)
		}
	}
	if wantTS != "" {
		v := 0
		if want != nil && *want {
			v = 1
		}
		sets = append(sets, `want_to_read = ?`, `want_to_read_updated_at = ?`)
		args = append(args, v, wantTS)
	}
	if len(sets) == 0 {
		return nil
	}
	args = append(args, id)
	_, err := tx.ExecContext(ctx, `UPDATE works SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...)
	return err
}

func (c *Catalog) WantToReadCount(ctx context.Context) (int, error) {
	var n int
	err := c.read.QueryRowContext(ctx, `SELECT count(*) FROM works WHERE want_to_read = 1`).Scan(&n)
	return n, err
}

func (c *Catalog) LatestViewedID(ctx context.Context) (int64, error) {
	var id int64
	err := c.read.QueryRowContext(ctx, `SELECT work_id FROM recently_viewed ORDER BY viewed_at DESC LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

func (c *Catalog) TopGenres(ctx context.Context, limit int) ([]GenreRow, error) {
	if limit <= 0 {
		limit = 8
	}
	rows, err := c.read.QueryContext(ctx, `
SELECT id, code, name_ru, work_count FROM genres
 WHERE work_count > 0
 ORDER BY work_count DESC, id
 LIMIT ?`, limit)
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

func (c *Catalog) TopSeries(ctx context.Context, limit int) ([]SeriesRow, error) {
	if limit <= 0 {
		limit = 6
	}
	rows, err := c.read.QueryContext(ctx, `
SELECT id, name, sort_name, work_count FROM series
 WHERE work_count > 0
 ORDER BY work_count DESC, id
 LIMIT ?`, limit)
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
