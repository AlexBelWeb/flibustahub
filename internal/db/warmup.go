package db

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/alexbelweb/flibustahub/internal/visibility"
)

// WarmUpCatalog rebuilds materialized genre/series tables and authors/series FTS.
func WarmUpCatalog(ctx context.Context, e Execer) error {
	stmts := []string{
		`DELETE FROM work_genres`,
		`INSERT INTO work_genres(work_id, genre_id)
		 SELECT DISTINCT e.work_id, eg.genre_id
		   FROM edition_genres eg
		   JOIN editions e ON e.id = eg.edition_id
		  WHERE ` + visibility.VisibleEditionSQL,
		`UPDATE genres SET work_count = (
		   SELECT count(*) FROM work_genres wg WHERE wg.genre_id = genres.id)`,
	}
	stmts = append(stmts, visibility.ListableTempSQL...)
	stmts = append(stmts,
		`DROP TABLE IF EXISTS temp.acount`,
		`CREATE TEMP TABLE acount (author_id INTEGER PRIMARY KEY, n INTEGER NOT NULL)`,
		`INSERT INTO acount(author_id, n)
		 SELECT wa.author_id, count(*)
		   FROM work_authors wa
		   JOIN listable l ON l.work_id = wa.work_id
		  GROUP BY wa.author_id`,
		`UPDATE authors SET work_count =
		   IFNULL((SELECT n FROM acount a WHERE a.author_id = authors.id), 0)`,
		`DROP TABLE IF EXISTS temp.acount`,
	)
	for _, s := range stmts {
		if _, err := e.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("warmup: %w", err)
		}
	}
	if err := refreshWorkAddedDates(ctx, e); err != nil {
		return err
	}
	if err := storeCatalogCounts(ctx, e); err != nil {
		return err
	}
	if _, err := e.ExecContext(ctx, `DROP TABLE IF EXISTS temp.listable`); err != nil {
		return fmt.Errorf("warmup: %w", err)
	}

	seriesStmts := []string{
		`DELETE FROM series`,
		`INSERT INTO series(name, sort_name, work_count)
		 SELECT e.series,
		        CASE WHEN normalize(e.series) = '' THEN char(65535) ELSE normalize(e.series) END,
		        count(DISTINCT e.work_id)
		   FROM editions e
		  WHERE ` + visibility.VisibleEditionSQL + `
		    AND e.series IS NOT NULL AND trim(e.series) != ''
		  GROUP BY e.series`,
	}
	for _, s := range seriesStmts {
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
	_, err := e.ExecContext(ctx, `INSERT INTO app_meta(key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`, MetaLastWarmup, now)
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

// refreshWorkAddedDates sets works.added_date to max(visible edition added_date)
// with one grouped pass into a keyed temp table. Missing dates become ”.
func refreshWorkAddedDates(ctx context.Context, e Execer) error {
	stmts := []string{
		`DROP TABLE IF EXISTS temp.work_added`,
		`CREATE TEMP TABLE work_added (work_id INTEGER PRIMARY KEY, added_date TEXT NOT NULL DEFAULT '')`,
		`INSERT INTO work_added(work_id) SELECT id FROM works`,
		`UPDATE work_added SET added_date = src.added_date
		   FROM (
		     SELECT e.work_id, max(e.added_date) AS added_date
		       FROM editions e
		      WHERE ` + visibility.VisibleEditionSQL + `
		        AND e.added_date IS NOT NULL AND trim(e.added_date) != ''
		      GROUP BY e.work_id
		   ) AS src
		  WHERE work_added.work_id = src.work_id`,
		`UPDATE works SET added_date = a.added_date FROM work_added AS a WHERE works.id = a.work_id`,
		`DROP TABLE IF EXISTS temp.work_added`,
	}
	for _, s := range stmts {
		if _, err := e.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("warmup added_date: %w", err)
		}
	}
	return nil
}

func storeCatalogCounts(ctx context.Context, e Execer) error {
	var total int
	if err := e.QueryRowContext(ctx, `SELECT count(*) FROM works`).Scan(&total); err != nil {
		return fmt.Errorf("warmup works_total: %w", err)
	}
	listable := 0
	err := e.QueryRowContext(ctx, `SELECT count(*) FROM temp.listable`).Scan(&listable)
	if err != nil {
		if err := e.QueryRowContext(ctx, `SELECT count(*) FROM works w WHERE `+visibility.ListableWorkSQL).Scan(&listable); err != nil {
			return fmt.Errorf("warmup works_listable: %w", err)
		}
	}
	if err := putMeta(ctx, e, MetaWorksTotal, strconv.Itoa(total)); err != nil {
		return err
	}
	return putMeta(ctx, e, MetaWorksListable, strconv.Itoa(listable))
}

func putMeta(ctx context.Context, e Execer, key, value string) error {
	return SetMeta(ctx, e, key, value)
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
