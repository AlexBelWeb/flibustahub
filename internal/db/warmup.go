package db

import (
	"context"
	"fmt"
	"time"
)

// WarmUpCatalog rebuilds materialized genre/series tables and authors/series FTS.
func WarmUpCatalog(ctx context.Context, e Execer) error {
	stmts := []string{
		`DELETE FROM work_genres`,
		`INSERT INTO work_genres(work_id, genre_id)
		 SELECT DISTINCT e.work_id, eg.genre_id
		   FROM edition_genres eg
		   JOIN editions e ON e.id = eg.edition_id
		  WHERE e.is_active = 1 AND e.is_deleted = 0`,
		`UPDATE genres SET work_count = (
		   SELECT count(*) FROM work_genres wg WHERE wg.genre_id = genres.id)`,
		`DROP TABLE IF EXISTS temp.listable`,
		`CREATE TEMP TABLE listable (work_id INTEGER PRIMARY KEY)`,
		`INSERT OR IGNORE INTO listable(work_id)
		 SELECT DISTINCT work_id FROM editions WHERE is_active = 1 AND is_deleted = 0`,
		`INSERT OR IGNORE INTO listable(work_id)
		 SELECT id FROM works
		  WHERE rating IS NOT NULL
		     OR (comment IS NOT NULL AND trim(comment) <> '')`,
		`DROP TABLE IF EXISTS temp.acount`,
		`CREATE TEMP TABLE acount (author_id INTEGER PRIMARY KEY, n INTEGER NOT NULL)`,
		`INSERT INTO acount(author_id, n)
		 SELECT wa.author_id, count(*)
		   FROM work_authors wa
		   JOIN listable l ON l.work_id = wa.work_id
		  GROUP BY wa.author_id`,
		`UPDATE authors SET work_count =
		   IFNULL((SELECT n FROM acount a WHERE a.author_id = authors.id), 0)`,
		`DROP TABLE IF EXISTS temp.listable`,
		`DROP TABLE IF EXISTS temp.acount`,
		`DELETE FROM series`,
		`INSERT INTO series(name, sort_name, work_count)
		 SELECT e.series,
		        normalize(e.series),
		        count(DISTINCT e.work_id)
		   FROM editions e
		  WHERE e.is_active = 1 AND e.is_deleted = 0
		    AND e.series IS NOT NULL AND trim(e.series) != ''
		  GROUP BY e.series`,
	}
	for _, s := range stmts {
		if _, err := e.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("warmup: %w", err)
		}
	}
	if err := rebuildNamedFTS(ctx, e, "authors_fts", `
INSERT INTO authors_fts(rowid, display_name, sort_name)
SELECT id, normalize(display_name), normalize(sort_name) FROM authors`); err != nil {
		return err
	}
	if err := rebuildNamedFTS(ctx, e, "series_fts", `
INSERT INTO series_fts(rowid, name)
SELECT id, normalize(name) FROM series`); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := e.ExecContext(ctx, `INSERT INTO app_meta(key, value) VALUES ('last_warmup_at', ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, now)
	return err
}

func rebuildNamedFTS(ctx context.Context, e Execer, name, fill string) error {
	create, err := FTSCreate(name)
	if err != nil {
		return err
	}
	if _, err := e.ExecContext(ctx, "DROP TABLE IF EXISTS "+name); err != nil {
		return err
	}
	if _, err := e.ExecContext(ctx, create); err != nil {
		return err
	}
	if _, err := e.ExecContext(ctx, fill); err != nil {
		return fmt.Errorf("%s fill: %w", name, err)
	}
	return nil
}

// Analyze runs ANALYZE on the connection.
func Analyze(ctx context.Context, e Execer) error {
	_, err := e.ExecContext(ctx, "ANALYZE")
	return err
}

// Optimize runs PRAGMA optimize after materialized tables have been rebuilt.
func Optimize(ctx context.Context, e Execer) error {
	_, err := e.ExecContext(ctx, "PRAGMA optimize")
	return err
}

// WarmCache runs the post-import cache touch queries.
func WarmCache(ctx context.Context, e Execer) error {
	var n int
	if err := e.QueryRowContext(ctx, `SELECT count(*) FROM works`).Scan(&n); err != nil {
		return err
	}
	_ = n
	var dummy int
	_ = e.QueryRowContext(ctx, `SELECT count(*) FROM works_fts`).Scan(&dummy)
	return nil
}
