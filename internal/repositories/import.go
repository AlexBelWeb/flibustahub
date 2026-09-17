package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/alexbelweb/flibustahub/internal/data"
	"github.com/alexbelweb/flibustahub/internal/inpx"
)

// ImportTx holds prepared statements for one import transaction.
type ImportTx struct {
	tx               *sql.Tx
	now              string
	authors          map[string]int64
	genres           map[string]int64
	works            map[string]cachedWork
	unnamed          []string
	unnamedSeen      map[string]struct{}
	UnnamedTotal     int
	GenreNamesMapped int
	ambiguous        []string
	ambiguousSeen    map[string]struct{}

	selWork       *sql.Stmt
	insWork       *sql.Stmt
	updWork       *sql.Stmt
	selAuthor     *sql.Stmt
	insAuthor     *sql.Stmt
	insWorkAuthor *sql.Stmt
	selWA         *sql.Stmt
	selGenre      *sql.Stmt
	insGenre      *sql.Stmt
	selEdition    *sql.Stmt
	insEdition    *sql.Stmt
	updEdition    *sql.Stmt
	delEdGenres   *sql.Stmt
	insEdGenre    *sql.Stmt
	insSeen       *sql.Stmt
}

type cachedWork struct {
	id        int64
	title     string
	sortTitle string
	lang      string
}

type existingEdition struct {
	ID          int64
	WorkID      int64
	ArchiveName string
	IsDeleted   bool
}

// PrepareImport creates TEMP seen_libids and statement cache on tx.
func PrepareImport(ctx context.Context, tx *sql.Tx, now time.Time) (*ImportTx, error) {
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS temp.seen_libids`); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `CREATE TEMP TABLE seen_libids (libid TEXT PRIMARY KEY) WITHOUT ROWID`); err != nil {
		return nil, err
	}
	p := &ImportTx{
		tx:            tx,
		now:           now.UTC().Format(time.RFC3339),
		authors:       map[string]int64{},
		genres:        map[string]int64{},
		works:         map[string]cachedWork{},
		unnamedSeen:   map[string]struct{}{},
		ambiguousSeen: map[string]struct{}{},
	}
	var err error
	p.selWork, err = tx.PrepareContext(ctx, `SELECT id, title, sort_title, IFNULL(lang,'') FROM works WHERE work_key = ?`)
	if err != nil {
		return nil, err
	}
	p.insWork, err = tx.PrepareContext(ctx, `INSERT INTO works (work_key, title, sort_title, authors_text, lang, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, err
	}
	p.updWork, err = tx.PrepareContext(ctx, `UPDATE works SET title = ?, sort_title = ?, lang = ?, updated_at = ?
WHERE id = ? AND (title <> ? OR sort_title <> ? OR IFNULL(lang,'') <> IFNULL(?,''))`)
	if err != nil {
		return nil, err
	}
	p.selAuthor, err = tx.PrepareContext(ctx, `SELECT id FROM authors WHERE author_key = ?`)
	if err != nil {
		return nil, err
	}
	p.insAuthor, err = tx.PrepareContext(ctx, `INSERT INTO authors (author_key, last_name, first_name, middle_name, display_name, sort_name)
VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, err
	}
	p.insWorkAuthor, err = tx.PrepareContext(ctx, `INSERT OR IGNORE INTO work_authors (work_id, author_id, position) VALUES (?, ?, ?)`)
	if err != nil {
		return nil, err
	}
	p.selWA, err = tx.PrepareContext(ctx, `SELECT 1 FROM work_authors WHERE work_id = ? LIMIT 1`)
	if err != nil {
		return nil, err
	}
	p.selGenre, err = tx.PrepareContext(ctx, `SELECT id FROM genres WHERE code = ?`)
	if err != nil {
		return nil, err
	}
	p.insGenre, err = tx.PrepareContext(ctx, `INSERT INTO genres (code, name_ru) VALUES (?, ?)
ON CONFLICT(code) DO UPDATE SET name_ru = excluded.name_ru
WHERE genres.name_ru <> excluded.name_ru`)
	if err != nil {
		return nil, err
	}
	p.selEdition, err = tx.PrepareContext(ctx, `SELECT id, work_id, archive_name, is_deleted FROM editions WHERE libid = ?`)
	if err != nil {
		return nil, err
	}
	p.insEdition, err = tx.PrepareContext(ctx, `INSERT INTO editions (libid, work_id, archive_name, file_name, file_ext, size, series, series_no, lang, librate, keywords, added_date, is_deleted, is_active)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`)
	if err != nil {
		return nil, err
	}
	p.updEdition, err = tx.PrepareContext(ctx, `UPDATE editions SET
  work_id = ?, archive_name = ?, file_name = ?, file_ext = ?, size = ?, series = ?, series_no = ?,
  lang = ?, librate = ?, keywords = ?, added_date = ?, is_deleted = ?, is_active = 1
WHERE id = ? AND (
  work_id <> ? OR archive_name <> ? OR file_name <> ? OR file_ext <> ?
  OR IFNULL(size,-1) <> IFNULL(?, -1) OR IFNULL(series,'') <> IFNULL(?, '')
  OR IFNULL(series_no,'') <> IFNULL(?, '') OR IFNULL(lang,'') <> IFNULL(?, '')
  OR IFNULL(librate,-1) <> IFNULL(?, -1) OR IFNULL(keywords,'') <> IFNULL(?, '')
  OR IFNULL(added_date,'') <> IFNULL(?, '') OR is_deleted <> ?
)`)
	if err != nil {
		return nil, err
	}
	p.delEdGenres, err = tx.PrepareContext(ctx, `DELETE FROM edition_genres WHERE edition_id = ?`)
	if err != nil {
		return nil, err
	}
	p.insEdGenre, err = tx.PrepareContext(ctx, `INSERT OR IGNORE INTO edition_genres (edition_id, genre_id) VALUES (?, ?)`)
	if err != nil {
		return nil, err
	}
	p.insSeen, err = tx.PrepareContext(ctx, `INSERT OR IGNORE INTO seen_libids (libid) VALUES (?)`)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *ImportTx) Close() {
	for _, s := range []*sql.Stmt{
		p.selWork, p.insWork, p.updWork, p.selAuthor, p.insAuthor, p.insWorkAuthor, p.selWA,
		p.selGenre, p.insGenre, p.selEdition, p.insEdition, p.updEdition,
		p.delEdGenres, p.insEdGenre, p.insSeen,
	} {
		if s != nil {
			_ = s.Close()
		}
	}
}

// ApplyResult counts mutations for one record.
type ApplyResult struct {
	WorkAdded      bool
	EditionAdded   bool
	EditionUpdated bool
	Collision      bool
}

func (p *ImportTx) Apply(ctx context.Context, rec inpx.Record) (ApplyResult, error) {
	var out ApplyResult
	if _, err := p.insSeen.ExecContext(ctx, rec.LibID); err != nil {
		return out, err
	}
	authorIDs, err := p.ensureAuthors(ctx, rec.Authors)
	if err != nil {
		return out, err
	}
	workKey, _ := inpx.WorkKey(rec.Authors, rec.Title)
	workID, added, err := p.ensureWork(ctx, rec, workKey, authorIDs)
	if err != nil {
		return out, err
	}
	out.WorkAdded = added
	ex, found, err := p.lookupEdition(ctx, rec.LibID)
	if err != nil {
		return out, err
	}
	if found && ex.ArchiveName != rec.ArchiveName {
		out.Collision = true
	}
	if !found {
		if err := p.insertEdition(ctx, rec, workID); err != nil {
			return out, err
		}
		out.EditionAdded = true
		return out, nil
	}
	incoming := rec
	if !inpx.ShouldReplace(inpx.EditionState{IsDeleted: ex.IsDeleted, ArchiveName: ex.ArchiveName}, incoming) {
		return out, nil
	}
	changed, err := p.updateEdition(ctx, ex.ID, rec, workID)
	if err != nil {
		return out, err
	}
	out.EditionUpdated = changed
	return out, nil
}

func (p *ImportTx) ensureAuthors(ctx context.Context, authors []inpx.Author) ([]int64, error) {
	ids := make([]int64, len(authors))
	for i, a := range authors {
		key := a.Key()
		if id, ok := p.authors[key]; ok {
			ids[i] = id
			continue
		}
		var id int64
		err := p.selAuthor.QueryRowContext(ctx, key).Scan(&id)
		if err == sql.ErrNoRows {
			res, err := p.insAuthor.ExecContext(ctx, key, a.Last, a.First, a.Middle, a.DisplayName(), a.SortName())
			if err != nil {
				return nil, err
			}
			id, err = res.LastInsertId()
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
		p.authors[key] = id
		ids[i] = id
	}
	return ids, nil
}

func (p *ImportTx) ensureWork(ctx context.Context, rec inpx.Record, workKey string, authorIDs []int64) (int64, bool, error) {
	sortTitle := inpx.SortTitle(rec.Title)
	if cached, ok := p.works[workKey]; ok {
		if !rec.IsDeleted {
			if err := p.updateWorkIfChanged(ctx, workKey, cached, rec.Title, sortTitle, rec.Lang); err != nil {
				return 0, false, err
			}
		}
		return p.works[workKey].id, false, nil
	}
	var cached cachedWork
	err := p.selWork.QueryRowContext(ctx, workKey).Scan(&cached.id, &cached.title, &cached.sortTitle, &cached.lang)
	if err == sql.ErrNoRows {
		res, err := p.insWork.ExecContext(ctx, workKey, rec.Title, sortTitle, inpx.AuthorsText(rec.Authors), nullStr(rec.Lang), p.now, p.now)
		if err != nil {
			return 0, false, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return 0, false, err
		}
		for pos, aid := range authorIDs {
			if _, err := p.insWorkAuthor.ExecContext(ctx, id, aid, pos); err != nil {
				return 0, false, err
			}
		}
		p.works[workKey] = cachedWork{id: id, title: rec.Title, sortTitle: sortTitle, lang: rec.Lang}
		return id, true, nil
	}
	if err != nil {
		return 0, false, err
	}
	p.works[workKey] = cached
	if !rec.IsDeleted {
		if err := p.updateWorkIfChanged(ctx, workKey, cached, rec.Title, sortTitle, rec.Lang); err != nil {
			return 0, false, err
		}
	}
	id := p.works[workKey].id
	var one int
	if err := p.selWA.QueryRowContext(ctx, id).Scan(&one); err == sql.ErrNoRows {
		for pos, aid := range authorIDs {
			if _, err := p.insWorkAuthor.ExecContext(ctx, id, aid, pos); err != nil {
				return 0, false, err
			}
		}
	} else if err != nil {
		return 0, false, err
	}
	return id, false, nil
}

func (p *ImportTx) updateWorkIfChanged(ctx context.Context, workKey string, cached cachedWork, title, sortTitle, lang string) error {
	if cached.title == title && cached.sortTitle == sortTitle && cached.lang == lang {
		return nil
	}
	if _, err := p.updWork.ExecContext(ctx, title, sortTitle, nullStr(lang), p.now,
		cached.id, title, sortTitle, nullStr(lang)); err != nil {
		return err
	}
	cached.title = title
	cached.sortTitle = sortTitle
	cached.lang = lang
	p.works[workKey] = cached
	return nil
}

func (p *ImportTx) lookupEdition(ctx context.Context, libid string) (existingEdition, bool, error) {
	var ex existingEdition
	var del int
	err := p.selEdition.QueryRowContext(ctx, libid).Scan(&ex.ID, &ex.WorkID, &ex.ArchiveName, &del)
	if err == sql.ErrNoRows {
		return ex, false, nil
	}
	if err != nil {
		return ex, false, err
	}
	ex.IsDeleted = del != 0
	return ex, true, nil
}

func (p *ImportTx) insertEdition(ctx context.Context, rec inpx.Record, workID int64) error {
	res, err := p.insEdition.ExecContext(ctx,
		rec.LibID, workID, rec.ArchiveName, rec.File, rec.Ext, rec.Size, nullStr(rec.Series), rec.SeriesNo,
		nullStr(rec.Lang), rec.LibRate, rec.Keywords, rec.Date, boolInt(rec.IsDeleted),
	)
	if err != nil {
		return err
	}
	edID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	return p.insertGenres(ctx, edID, rec.Genres)
}

func (p *ImportTx) updateEdition(ctx context.Context, id int64, rec inpx.Record, workID int64) (bool, error) {
	res, err := p.updEdition.ExecContext(ctx,
		workID, rec.ArchiveName, rec.File, rec.Ext, rec.Size, nullStr(rec.Series), rec.SeriesNo,
		nullStr(rec.Lang), rec.LibRate, rec.Keywords, rec.Date, boolInt(rec.IsDeleted),
		id,
		workID, rec.ArchiveName, rec.File, rec.Ext, rec.Size, nullStr(rec.Series), rec.SeriesNo,
		nullStr(rec.Lang), rec.LibRate, rec.Keywords, rec.Date, boolInt(rec.IsDeleted),
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if err := p.replaceGenres(ctx, id, rec.Genres); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (p *ImportTx) replaceGenres(ctx context.Context, editionID int64, codes []string) error {
	if _, err := p.delEdGenres.ExecContext(ctx, editionID); err != nil {
		return err
	}
	return p.insertGenres(ctx, editionID, codes)
}

func (p *ImportTx) insertGenres(ctx context.Context, editionID int64, codes []string) error {
	for _, code := range codes {
		gid, err := p.ensureGenre(ctx, code)
		if err != nil {
			return err
		}
		if _, err := p.insEdGenre.ExecContext(ctx, editionID, gid); err != nil {
			return err
		}
	}
	return nil
}

func (p *ImportTx) ensureGenre(ctx context.Context, raw string) (int64, error) {
	code, mapped, ambiguous := data.ResolveGenre(raw)
	if mapped {
		p.GenreNamesMapped++
	}
	if ambiguous {
		p.noteAmbiguous(raw)
	}
	if id, ok := p.genres[code]; ok {
		return id, nil
	}
	if !data.KnownGenre(code) {
		p.UnnamedTotal++
		if _, ok := p.unnamedSeen[code]; !ok {
			p.unnamedSeen[code] = struct{}{}
			if len(p.unnamed) < 20 {
				p.unnamed = append(p.unnamed, code)
			}
		}
	}
	name := data.GenreNameRU(code)
	if _, err := p.insGenre.ExecContext(ctx, code, name); err != nil {
		return 0, err
	}
	var id int64
	if err := p.selGenre.QueryRowContext(ctx, code).Scan(&id); err != nil {
		return 0, err
	}
	p.genres[code] = id
	return id, nil
}

func (p *ImportTx) noteAmbiguous(raw string) {
	if _, ok := p.ambiguousSeen[raw]; ok {
		return
	}
	p.ambiguousSeen[raw] = struct{}{}
	p.ambiguous = append(p.ambiguous, raw)
}

// UnnamedGenres returns at most 20 codes that had no dictionary name.
func (p *ImportTx) UnnamedGenres() []string {
	return append([]string(nil), p.unnamed...)
}

// AmbiguousGenres returns dump GENRE values that matched several dictionary names.
func (p *ImportTx) AmbiguousGenres() []string {
	return append([]string(nil), p.ambiguous...)
}

// DeactivateMissing flips is_active only for rows that actually change.
func DeactivateMissing(ctx context.Context, tx *sql.Tx) (deactivated int64, err error) {
	res, err := tx.ExecContext(ctx, `UPDATE editions SET is_active = 0
WHERE is_active = 1 AND libid NOT IN (SELECT libid FROM seen_libids)`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	if _, err := tx.ExecContext(ctx, `UPDATE editions SET is_active = 1
WHERE is_active = 0 AND libid IN (SELECT libid FROM seen_libids)`); err != nil {
		return 0, err
	}
	return n, nil
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
