package inpximport

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/textnorm"
	"github.com/alexbelweb/flibustahub/internal/visibility"
)

const personalProbeComment = "reimport-probe-keep"

func TestImportFullDump(t *testing.T) {
	root := os.Getenv("FLIBUSTAHUB_LIBRARYROOT")
	inpxPath := os.Getenv("FLIBUSTAHUB_INPXPATH")
	if root == "" && inpxPath == "" {
		t.Skip("FLIBUSTAHUB_LIBRARYROOT / FLIBUSTAHUB_INPXPATH not set")
	}
	dir := t.TempDir()
	if v := os.Getenv("FLIBUSTAHUB_DATADIR"); v != "" {
		if err := os.MkdirAll(v, 0o755); err != nil {
			t.Fatal(err)
		}
		dir = v
	}
	d, err := db.Open(context.Background(), db.Options{
		Path:       filepath.Join(dir, "catalog.sqlite"),
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.WaitSearchIndex(context.Background()); err != nil {
		t.Fatal(err)
	}

	before, probes := prepareReimport(t, d)

	t0 := time.Now()
	svc := New(d, slog.Default())
	var ticks []RecordsTick
	var importSync, importJournal, importFK string
	rep, err := svc.Import(context.Background(), Options{
		LibraryRoot: root,
		INPXPath:    inpxPath,
		OnRecordsTick: func(tick RecordsTick) {
			ticks = append(ticks, tick)
			t.Logf("records seen=%d elapsed_ms=%d bucket_ms=%d", tick.Seen, tick.ElapsedMS, tick.DeltaMS)
		},
		RecordsProbe: func(tx *sql.Tx) error {
			var e error
			if importSync, e = readNamedPragma(tx, "synchronous"); e != nil {
				return e
			}
			if importJournal, e = readNamedPragma(tx, "journal_mode"); e != nil {
				return e
			}
			if importFK, e = readNamedPragma(tx, "foreign_keys"); e != nil {
				return e
			}
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(t0)
	st, err := os.Stat(d.Path())
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("selected inpx=%s version=%s", rep.INPXPath, rep.INPXVersion)
	if inpxPath == "" {
		if strings.Contains(strings.ToLower(filepath.ToSlash(rep.INPXPath)), "/update/") {
			t.Fatalf("autosearch picked a nested dump: %s", rep.INPXPath)
		}
		if filepath.Dir(rep.INPXPath) != filepath.Clean(root) {
			t.Fatalf("autosearch must stay in library root, got %s", rep.INPXPath)
		}
	}

	var deleted, editions int
	if err := d.Read.QueryRow(`SELECT count(*) FROM editions`).Scan(&editions); err != nil {
		t.Fatal(err)
	}
	if err := d.Read.QueryRow(`SELECT count(*) FROM editions WHERE is_deleted=1`).Scan(&deleted); err != nil {
		t.Fatal(err)
	}
	t.Logf("import status=%s records=%d works_added=%d editions=%d editions_added=%d editions_updated=%d deactivated=%d collisions=%d unnamed=%d genre_names_mapped=%d missing_archives=%d deleted=%d",
		rep.Status, rep.RecordsSeen, rep.WorksAdded, editions, rep.EditionsAdded, rep.EditionsUpdated, rep.EditionsDeactivated, rep.LibIDCollisions, rep.Notes.UnnamedGenresTotal, rep.Notes.GenreNamesMapped, rep.Notes.MissingArchivesTotal, deleted)
	t.Logf("unnamed_genres=%v skipped_malformed=%d skipped_no_libid=%d encodings=%+v",
		rep.Notes.UnnamedGenres, rep.Notes.SkippedMalformed, rep.Notes.SkippedNoLibID, rep.Notes.Encodings)
	t.Logf("phases_ms backup=%d reading=%d records=%d fts=%d warmup=%d analyze=%d wall=%s db_bytes=%d",
		rep.Notes.PhasesMS["backup"], rep.Notes.PhasesMS["reading"], rep.Notes.PhasesMS["records"],
		rep.Notes.PhasesMS["fts"], rep.Notes.PhasesMS["warmup"], rep.Notes.PhasesMS["analyze"], elapsed, st.Size())
	if editions > 0 {
		t.Logf("deleted_share=%.1f%%", 100*float64(deleted)/float64(editions))
	}
	if len(ticks) > 0 {
		t.Logf("records_profile ticks=%d first_bucket_ms=%d last_bucket_ms=%d", len(ticks), ticks[0].DeltaMS, ticks[len(ticks)-1].DeltaMS)
	}
	t.Logf("import_conn pragmas synchronous=%s journal_mode=%s foreign_keys=%s", importSync, importJournal, importFK)
	if importJournal != "" && !strings.EqualFold(importJournal, "wal") {
		t.Errorf("journal_mode=%s, want wal", importJournal)
	}
	if importSync != "" && importSync != "1" && !strings.EqualFold(importSync, "normal") {
		t.Errorf("synchronous=%s, want NORMAL (1); OFF is forbidden", importSync)
	}

	assertEnvCount(t, "FLIBUSTAHUB_EXPECT_RECORDS_SEEN", rep.RecordsSeen)
	assertEnvCount(t, "FLIBUSTAHUB_EXPECT_EDITIONS", editions)
	assertEnvCount(t, "FLIBUSTAHUB_EXPECT_LIBID_COLLISIONS", rep.LibIDCollisions)
	assertEnvCount(t, "FLIBUSTAHUB_EXPECT_SKIPPED_MALFORMED", rep.Notes.SkippedMalformed)
	assertEnvCount(t, "FLIBUSTAHUB_EXPECT_SKIPPED_NO_LIBID", rep.Notes.SkippedNoLibID)

	var works, authors int
	if err := d.Read.QueryRow(`SELECT count(*) FROM works`).Scan(&works); err != nil {
		t.Fatal(err)
	}
	if err := d.Read.QueryRow(`SELECT count(*) FROM authors`).Scan(&authors); err != nil {
		t.Fatal(err)
	}
	t.Logf("catalog counts works=%d editions=%d authors=%d", works, editions, authors)
	assertEnvCount(t, "FLIBUSTAHUB_EXPECT_WORKS", works)
	assertEnvCount(t, "FLIBUSTAHUB_EXPECT_AUTHORS", authors)
	assertEnvCount(t, "FLIBUSTAHUB_EXPECT_GENRE_NAMES_MAPPED", rep.Notes.GenreNamesMapped)

	var genres int
	if err := d.Read.QueryRow(`SELECT count(*) FROM genres`).Scan(&genres); err != nil {
		t.Fatal(err)
	}
	t.Logf("genres=%d", genres)
	assertEnvCount(t, "FLIBUSTAHUB_EXPECT_GENRES", genres)
	logGenreAndAuthorHygiene(t, d)

	logReadableSample(t, d)

	if before.works > 0 {
		assertReimportIdempotent(t, d, before, probes, rep, editions)
	}
	verifySearchIndex(t, d)
	verifyIntegrity(t, d)
	verifyListingStatsAndPlans(t, d)

	filled := filepath.Join(filepath.Dir(d.Path()), "filled-catalog.sqlite")
	_ = os.Remove(filled)
	slash := filepath.ToSlash(filled)
	if _, err := d.Write.ExecContext(context.Background(), "VACUUM INTO ?", slash); err != nil {
		quoted := strings.ReplaceAll(slash, "'", "''")
		if _, err2 := d.Write.ExecContext(context.Background(), "VACUUM INTO '"+quoted+"'"); err2 != nil {
			t.Fatalf("snapshot: %v / %v", err, err2)
		}
	}
	t.Logf("filled snapshot %s", filled)

	if elapsed > 10*time.Minute {
		t.Errorf("import exceeded 10m budget: %s (driver decision is the owner's)", elapsed)
	}
}

func logGenreAndAuthorHygiene(t *testing.T, d *db.DB) {
	t.Helper()
	for _, code := range []string{"det_espionage", "nonf_biography"} {
		var works, editions int
		_ = d.Read.QueryRow(`SELECT work_count FROM genres WHERE code = ?`, code).Scan(&works)
		_ = d.Read.QueryRow(`SELECT count(*) FROM edition_genres eg JOIN genres g ON g.id = eg.genre_id WHERE g.code = ?`, code).Scan(&editions)
		t.Logf("genre %s work_count=%d editions=%d", code, works, editions)
	}
	var rawLabels int
	_ = d.Read.QueryRow(`SELECT count(*) FROM genres WHERE code IN ('Биографии и мемуары','Шпионский Детектив')`).Scan(&rawLabels)
	t.Logf("raw_genre_labels_left=%d", rawLabels)
	if rawLabels != 0 {
		t.Errorf("dictionary names still stored as genre codes: %d", rawLabels)
	}

	var emptyLast, namelessSort int
	_ = d.Read.QueryRow(`SELECT count(*) FROM authors WHERE trim(last_name) = ''`).Scan(&emptyLast)
	_ = d.Read.QueryRow(`SELECT count(*) FROM authors
 WHERE trim(last_name) = ''
   AND (trim(first_name) != '' OR trim(middle_name) != '')
   AND trim(sort_name) = ''`).Scan(&namelessSort)
	t.Logf("authors empty last_name=%d sortable_dirty_without_sort_name=%d", emptyLast, namelessSort)
	if namelessSort != 0 {
		t.Errorf("%d authors with empty last_name still sort as nameless", namelessSort)
	}
}

func logReadableSample(t *testing.T, d *db.DB) {
	t.Helper()
	var gromovTitle, gromovAuthors string
	err := d.Read.QueryRow(`SELECT w.title, w.authors_text FROM works w JOIN editions e ON e.work_id = w.id WHERE e.libid = '110119'`).
		Scan(&gromovTitle, &gromovAuthors)
	if err != nil {
		t.Errorf("libid 110119: %v", err)
	} else {
		t.Logf("libid 110119 title=%s authors=%s", gromovTitle, gromovAuthors)
		if gromovTitle != "Первый из могикан" || !strings.Contains(gromovAuthors, "Громов") {
			t.Errorf("libid 110119 want Первый из могикан / Громов, got title=%q authors=%q", gromovTitle, gromovAuthors)
		}
	}
	rows, err := d.Read.Query(`SELECT title, authors_text FROM works ORDER BY id LIMIT 10`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		var title, authors string
		if err := rows.Scan(&title, &authors); err != nil {
			t.Fatal(err)
		}
		i++
		t.Logf("readable[%d] title=%s authors=%s", i, title, authors)
	}
}

type catalogCounts struct {
	works, editions, authors int
}

type personalProbe struct {
	id      int64
	rating  int
	comment string
}

func prepareReimport(t *testing.T, d *db.DB) (catalogCounts, []personalProbe) {
	t.Helper()
	var c catalogCounts
	_ = d.Read.QueryRow(`SELECT count(*) FROM works`).Scan(&c.works)
	_ = d.Read.QueryRow(`SELECT count(*) FROM editions`).Scan(&c.editions)
	_ = d.Read.QueryRow(`SELECT count(*) FROM authors`).Scan(&c.authors)
	if c.works == 0 {
		return c, nil
	}
	t.Logf("reimport baseline works=%d editions=%d authors=%d", c.works, c.editions, c.authors)
	rows, err := d.Write.Query(`SELECT id FROM works ORDER BY id LIMIT 2`)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	_ = rows.Close()
	probes := make([]personalProbe, 0, len(ids))
	for i, id := range ids {
		p := personalProbe{id: id, rating: 7 + i, comment: personalProbeComment}
		if _, err := d.Write.Exec(`UPDATE works SET rating=?, rating_updated_at='probe', comment=?, comment_updated_at='probe' WHERE id=?`,
			p.rating, p.comment, p.id); err != nil {
			t.Fatal(err)
		}
		probes = append(probes, p)
	}
	t.Logf("stamped personal data on %d works", len(probes))
	return c, probes
}

func assertReimportIdempotent(t *testing.T, d *db.DB, before catalogCounts, probes []personalProbe, rep Report, editions int) {
	t.Helper()
	if rep.WorksAdded != 0 {
		t.Errorf("reimport works_added=%d, want 0", rep.WorksAdded)
	}
	if rep.EditionsAdded != 0 {
		t.Errorf("reimport editions_added=%d, want 0", rep.EditionsAdded)
	}
	var works, authors int
	_ = d.Read.QueryRow(`SELECT count(*) FROM works`).Scan(&works)
	_ = d.Read.QueryRow(`SELECT count(*) FROM authors`).Scan(&authors)
	if works != before.works || editions != before.editions || authors != before.authors {
		t.Errorf("counts changed works %d→%d editions %d→%d authors %d→%d",
			before.works, works, before.editions, editions, before.authors, authors)
	}
	for _, p := range probes {
		var rating int
		var comment string
		if err := d.Read.QueryRow(`SELECT rating, comment FROM works WHERE id=?`, p.id).Scan(&rating, &comment); err != nil {
			t.Fatal(err)
		}
		if rating != p.rating || comment != p.comment {
			t.Errorf("personal data lost on work %d: rating=%d comment=%q", p.id, rating, comment)
		}
	}
}

func verifySearchIndex(t *testing.T, d *db.DB) {
	t.Helper()
	var works, worksFTS, authorsFTS, seriesFTS int
	_ = d.Read.QueryRow(`SELECT count(*) FROM works`).Scan(&works)
	_ = d.Read.QueryRow(`SELECT count(*) FROM works_fts`).Scan(&worksFTS)
	_ = d.Read.QueryRow(`SELECT count(*) FROM authors_fts`).Scan(&authorsFTS)
	_ = d.Read.QueryRow(`SELECT count(*) FROM series_fts`).Scan(&seriesFTS)
	t.Logf("fts rows works=%d works_fts=%d authors_fts=%d series_fts=%d", works, worksFTS, authorsFTS, seriesFTS)
	if worksFTS != works {
		t.Errorf("works_fts=%d works=%d", worksFTS, works)
	}
	if authorsFTS == 0 || seriesFTS == 0 {
		t.Errorf("authors_fts=%d series_fts=%d, want both non-empty", authorsFTS, seriesFTS)
	}
	var dirty string
	if err := d.Read.QueryRow(`SELECT value FROM app_meta WHERE key=?`, db.MetaFTSDirty).Scan(&dirty); err != nil {
		t.Fatal(err)
	}
	if dirty != "0" {
		t.Errorf("fts_dirty=%s, want 0", dirty)
	}
	var trig int
	_ = d.Read.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name IN ('works_fts_au','works_fts_ad')`).Scan(&trig)
	if trig != 2 {
		t.Errorf("works_fts triggers=%d, want 2", trig)
	}

	var storedHex string
	_ = d.Read.QueryRow(`SELECT hex(substr(w.title,1,16)) FROM works w JOIN editions e ON e.work_id = w.id WHERE e.libid = '110119'`).Scan(&storedHex)
	t.Logf("libid 110119 title prefix hex=%s (UTF-8 Первый = D09FD0B5D180D0B2)", storedHex)

	yo := textnorm.Normalize("ёлка")
	logFTSHits(t, d, "works title елка", `SELECT title FROM works_fts WHERE works_fts MATCH ? LIMIT 8`, yo)
	yoHits := queryStrings(t, d, `SELECT w.title FROM works w
 WHERE w.id IN (SELECT rowid FROM works_fts WHERE works_fts MATCH ?)
   AND (instr(w.title, char(0x0451)) > 0 OR instr(w.title, char(0x0401)) > 0)
 LIMIT 5`, yo)
	t.Logf("елка matches with ё in title: %q", yoHits)
	if len(yoHits) == 0 {
		t.Error("MATCH елка found no title containing ё/Ё (index should store normalize)")
	}
	logFTSHits(t, d, "authors громов", `SELECT display_name FROM authors_fts WHERE authors_fts MATCH ? LIMIT 5`, textnorm.Normalize("Громов"))
	logFTSHits(t, d, "series матриархата", `SELECT name FROM series_fts WHERE series_fts MATCH ? LIMIT 5`, textnorm.Normalize("матриархата"))
}

func queryStrings(t *testing.T, d *db.DB, q string, args ...any) []string {
	t.Helper()
	rows, err := d.Read.Query(q, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

func logFTSHits(t *testing.T, d *db.DB, label, q, token string) {
	t.Helper()
	rows, err := d.Read.Query(q, token)
	if err != nil {
		t.Errorf("%s: %v", label, err)
		return
	}
	defer rows.Close()
	var hits []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		hits = append(hits, s)
	}
	t.Logf("%s token=%q hits=%d sample=%q", label, token, len(hits), hits)
	if len(hits) == 0 {
		t.Errorf("%s: no hits for %q", label, token)
	}
}

func verifyIntegrity(t *testing.T, d *db.DB) {
	t.Helper()
	t0 := time.Now()
	rows, err := d.Write.Query(`PRAGMA integrity_check`)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, s)
	}
	_ = rows.Close()
	t.Logf("integrity_check elapsed=%s result=%q", time.Since(t0).Round(time.Millisecond), lines)
	if len(lines) != 1 || lines[0] != "ok" {
		t.Errorf("integrity_check: %v", lines)
	}

	t0 = time.Now()
	fkRows, err := d.Write.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	var n int
	var sample []string
	for fkRows.Next() {
		var table, parent string
		var rowid, fkid int64
		if err := fkRows.Scan(&table, &rowid, &parent, &fkid); err != nil {
			t.Fatal(err)
		}
		n++
		if len(sample) < 5 {
			sample = append(sample, fmt.Sprintf("%s rowid=%d parent=%s fkid=%d", table, rowid, parent, fkid))
		}
	}
	_ = fkRows.Close()
	t.Logf("foreign_key_check elapsed=%s violations=%d sample=%v", time.Since(t0).Round(time.Millisecond), n, sample)
	if n != 0 {
		t.Errorf("foreign_key_check violations=%d", n)
	}
}

func verifyListingStatsAndPlans(t *testing.T, d *db.DB) {
	t.Helper()
	rows, err := d.Read.Query(`SELECT tbl, count(*) FROM sqlite_stat1 GROUP BY tbl ORDER BY tbl`)
	if err != nil {
		t.Fatal(err)
	}
	stats := map[string]int{}
	for rows.Next() {
		var tbl string
		var n int
		if err := rows.Scan(&tbl, &n); err != nil {
			t.Fatal(err)
		}
		stats[tbl] = n
	}
	_ = rows.Close()
	t.Logf("sqlite_stat1 tables=%v", stats)
	for _, tbl := range []string{"work_genres", "series", "authors"} {
		if stats[tbl] == 0 {
			t.Errorf("sqlite_stat1 has no rows for %s (listings in the next slice will pick a bad plan)", tbl)
		}
	}

	keysetSQL := `SELECT w.id, w.sort_title
  FROM works w
 WHERE ` + visibility.ListableWorkSQL + `
   AND (w.sort_title, w.id) > ('', 0)
 ORDER BY w.sort_title, w.id
 LIMIT 50`
	t.Logf("plan books keyset sort_title:\n%s", explainPlan(t, d, keysetSQL))

	var genreID int64
	if err := d.Read.QueryRow(`SELECT id FROM genres ORDER BY work_count DESC, id LIMIT 1`).Scan(&genreID); err != nil {
		t.Fatal(err)
	}
	genreSQL := fmt.Sprintf(`SELECT w.id, w.sort_title
  FROM work_genres wg
  JOIN works w ON w.id = wg.work_id
 WHERE wg.genre_id = %d
   AND EXISTS (
     SELECT 1 FROM editions e
      WHERE e.work_id = w.id AND e.is_active = 1 AND e.is_deleted = 0
   )
   AND (w.sort_title, w.id) > ('', 0)
 ORDER BY w.sort_title, w.id
 LIMIT 50`, genreID)
	t.Logf("plan books of genre_id=%d:\n%s", genreID, explainPlan(t, d, genreSQL))
}

func explainPlan(t *testing.T, d *db.DB, query string) string {
	t.Helper()
	rows, err := d.Read.Query("EXPLAIN QUERY PLAN " + query)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var b strings.Builder
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatal(err)
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func readNamedPragma(tx *sql.Tx, name string) (string, error) {
	var v any
	if err := tx.QueryRow("PRAGMA " + name).Scan(&v); err != nil {
		return "", err
	}
	switch t := v.(type) {
	case []byte:
		return string(t), nil
	default:
		return fmt.Sprint(t), nil
	}
}

func assertEnvCount(t *testing.T, key string, got int) {
	t.Helper()
	raw := os.Getenv(key)
	if raw == "" {
		return
	}
	want, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("%s=%q: %v", key, raw, err)
	}
	if got != want {
		t.Errorf("%s: got %d, want %d", key, got, want)
	}
}
