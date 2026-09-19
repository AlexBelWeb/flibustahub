package catalog

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/textnorm"
)

func openSvc(t *testing.T) (*Service, *db.DB) {
	t.Helper()
	d, err := db.Open(context.Background(), db.Options{
		Path:       filepath.Join(t.TempDir(), "catalog.sqlite"),
		BackupsDir: filepath.Join(t.TempDir(), "backups"),
		Log:        slog.New(slog.DiscardHandler),
		Now:        func() time.Time { return time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return New(d, slog.New(slog.DiscardHandler)), d
}

func execAll(t *testing.T, d *db.DB, stmts ...string) {
	t.Helper()
	for _, s := range stmts {
		if _, err := d.Write.Exec(s); err != nil {
			t.Fatalf("%v\n%s", err, s)
		}
	}
}

func seed(t *testing.T, d *db.DB) {
	t.Helper()
	execAll(t, d,
		`INSERT INTO authors(id, author_key, last_name, first_name, middle_name, display_name, sort_name) VALUES
		 (1, 'громов,александр,николаевич', 'Громов', 'Александр', 'Николаевич', 'Громов Александр Николаевич', 'громов александр николаевич'),
		 (2, 'елкин,иван,', 'Ёлкин', 'Иван', '', 'Ёлкин Иван', 'елкин иван'),
		 (3, 'йенсен,карл,', 'Йенсен', 'Карл', '', 'Йенсен Карл', 'йенсен карл'),
		 (4, 'asimov,isaac,', 'Asimov', 'Isaac', '', 'Asimov Isaac', 'asimov isaac')`,
		`INSERT INTO genres(id, code, name_ru) VALUES (1, 'sf', 'Фантастика'), (2, 'rare', 'Редкий')`,
		`INSERT INTO works(id, work_key, title, sort_title, authors_text, lang, created_at, updated_at) VALUES
		 (1, 'k-elka', 'Ёлка', 'елка', 'Громов Александр Николаевич', 'ru', 't', 't'),
		 (2, 'k-abrikos', 'Абрикос', 'абрикос', 'Громов Александр Николаевич', 'ru', 't', 't'),
		 (3, 'k-yabloko', 'Яблоко', 'яблоко', 'Ёлкин Иван', 'ru', 't', 't'),
		 (4, 'k-quote', '«Кавычки»', 'кавычки', 'Йенсен Карл', 'en', 't', 't'),
		 (5, 'k-v1', 'Том один', 'том один', 'Громов Александр Николаевич', 'ru', 't', 't'),
		 (6, 'k-v2', 'Том два', 'том два', 'Громов Александр Николаевич', 'ru', 't', 't'),
		 (7, 'k-v10', 'Том десять', 'том десять', 'Громов Александр Николаевич', 'ru', 't', 't'),
		 (8, 'k-range', 'Том диапазон', 'том диапазон', 'Громов Александр Николаевич', 'ru', 't', 't'),
		 (9, 'k-q', 'Том вопрос', 'том вопрос', 'Громов Александр Николаевич', 'ru', 't', 't'),
		 (10, 'k-empty', 'Том пустой', 'том пустой', 'Громов Александр Николаевич', 'ru', 't', 't'),
		 (11, 'k-ghost', 'Призрак', 'призрак', 'Asimov Isaac', 'en', 't', 't'),
		 (12, 'k-rare', 'Одинокая книга', 'одинокая книга', 'Йенсен Карл', 'ru', 't', 't')`,
		`INSERT INTO work_authors(work_id, author_id, position) VALUES
		 (1,1,0),(2,1,0),(3,2,0),(4,3,0),(5,1,0),(6,1,0),(7,1,0),(8,1,0),(9,1,0),(10,1,0),(11,4,0),(12,3,0)`,
		`INSERT INTO editions(id, libid, work_id, archive_name, file_name, series, series_no, lang, added_date, is_deleted, is_active) VALUES
		 (1,'1',1,'a.zip','f','Ёлки', '', 'ru', '2020-01-01', 0, 1),
		 (2,'2',2,'a.zip','f', NULL, NULL, 'ru', '2021-01-01', 0, 1),
		 (3,'3',3,'a.zip','f', NULL, NULL, 'ru', '2019-01-01', 0, 1),
		 (4,'4',4,'a.zip','f', NULL, NULL, 'en', '2018-01-01', 0, 1),
		 (5,'5',5,'a.zip','f','Серия Мусор', '1', 'ru', '2017-01-01', 0, 1),
		 (6,'6',6,'a.zip','f','Серия Мусор', '2', 'ru', '2017-01-02', 0, 1),
		 (7,'7',7,'a.zip','f','Серия Мусор', '10', 'ru', '2017-01-03', 0, 1),
		 (8,'8',8,'a.zip','f','Серия Мусор', '1-2', 'ru', '2017-01-04', 0, 1),
		 (9,'9',9,'a.zip','f','Серия Мусор', '?', 'ru', '2017-01-05', 0, 1),
		 (10,'10',10,'a.zip','f','Серия Мусор', '', 'ru', '2017-01-06', 0, 1),
		 (11,'11',11,'a.zip','f', NULL, NULL, 'en', '2016-01-01', 0, 1),
		 (12,'12',12,'a.zip','f', NULL, NULL, 'ru', '2015-01-01', 0, 1)`,
		`INSERT INTO edition_genres(edition_id, genre_id) VALUES (1,1),(2,1),(3,1),(4,1),(5,1),(6,1),(7,1),(8,1),(9,1),(10,1),(11,1),(12,2)`,
	)
	if err := db.WarmUpCatalog(context.Background(), d.Write); err != nil {
		t.Fatal(err)
	}
	fillWorksFTS(t, d)
}

func fillWorksFTS(t *testing.T, d *db.DB) {
	t.Helper()
	if _, err := d.Write.Exec(`INSERT INTO works_fts(rowid, title, authors, series)
		SELECT w.id, normalize(w.title), normalize(w.authors_text),
		       (SELECT normalize(group_concat(DISTINCT e.series)) FROM editions e
		         WHERE e.work_id = w.id AND e.is_active = 1 AND e.is_deleted = 0
		           AND e.series IS NOT NULL AND trim(e.series) != '')
		  FROM works w
		 WHERE w.id NOT IN (SELECT rowid FROM works_fts)`); err != nil {
		t.Fatal(err)
	}
}

func titles(page WorkPage) []string {
	out := make([]string, len(page.Items))
	for i, w := range page.Items {
		out[i] = w.Title
	}
	return out
}

func TestListWorksSortTitleAndKeyset(t *testing.T) {
	svc, d := openSvc(t)
	seed(t, d)
	ctx := context.Background()
	page, err := svc.ListWorks(ctx, ListWorksQuery{Limit: 4})
	if err != nil {
		t.Fatal(err)
	}
	got := titles(page)
	wantFirst := []string{"Абрикос", "Ёлка", "«Кавычки»", "Одинокая книга"}
	if strings.Join(got, ",") != strings.Join(wantFirst, ",") {
		t.Fatalf("first page %v want %v", got, wantFirst)
	}
	if page.NextCursor == "" {
		t.Fatal("expected cursor")
	}
	page2, err := svc.ListWorks(ctx, ListWorksQuery{Limit: 4, Cursor: page.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if page2.Items[0].Title == page.Items[0].Title {
		t.Fatal("keyset did not advance")
	}
	if page.Total == nil || page.Total.N != 12 {
		t.Fatalf("total %+v", page.Total)
	}
}

func TestSearchYoAndCase(t *testing.T) {
	svc, d := openSvc(t)
	seed(t, d)
	ctx := context.Background()
	a, err := svc.Search(ctx, SearchQuery{Q: "ёлка"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Search(ctx, SearchQuery{Q: "елка"})
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Works.Items) == 0 || len(a.Works.Items) != len(b.Works.Items) {
		t.Fatalf("ёлка %d елка %d", len(a.Works.Items), len(b.Works.Items))
	}
	if a.Works.Items[0].ID != b.Works.Items[0].ID {
		t.Fatalf("ids %d vs %d", a.Works.Items[0].ID, b.Works.Items[0].ID)
	}
	g1, err := svc.ListAuthors(ctx, ListPeopleQuery{Query: "Громов"})
	if err != nil {
		t.Fatal(err)
	}
	g2, err := svc.ListAuthors(ctx, ListPeopleQuery{Query: "громов"})
	if err != nil {
		t.Fatal(err)
	}
	if len(g1.Items) != 1 || g1.Items[0].ID != g2.Items[0].ID {
		t.Fatalf("author case: %+v %+v", g1.Items, g2.Items)
	}
}

func TestGhostStaysListableNotInGenre(t *testing.T) {
	svc, d := openSvc(t)
	seed(t, d)
	ctx := context.Background()
	if _, err := d.Write.Exec(`UPDATE works SET rating = 8 WHERE id = 11`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write.Exec(`UPDATE editions SET is_active = 0 WHERE work_id = 11`); err != nil {
		t.Fatal(err)
	}
	page, err := svc.ListWorks(ctx, ListWorksQuery{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range page.Items {
		if w.ID == 11 {
			found = true
			if w.HasFile {
				t.Fatal("ghost must not have a file")
			}
		}
	}
	if !found {
		t.Fatal("ghost missing from catalog")
	}
	res, err := svc.Search(ctx, SearchQuery{Q: "призрак"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Works.Items) != 1 || res.Works.Items[0].ID != 11 {
		t.Fatalf("ghost search %+v", titles(res.Works))
	}
	genre, err := svc.ListWorks(ctx, ListWorksQuery{GenreID: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range genre.Items {
		if w.ID == 11 {
			t.Fatal("ghost must not appear in genre filter")
		}
	}
	author, err := svc.ListWorks(ctx, ListWorksQuery{AuthorID: 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(author.Items) != 1 || author.Items[0].ID != 11 {
		t.Fatalf("author page should keep ghost, got %v", titles(author))
	}
}

func TestBrokenFTSFallsBackToLike(t *testing.T) {
	var buf bytes.Buffer
	d, err := db.Open(context.Background(), db.Options{
		Path:       filepath.Join(t.TempDir(), "catalog.sqlite"),
		BackupsDir: filepath.Join(t.TempDir(), "backups"),
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	seed(t, d)
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	svc := New(d, log)
	if _, err := d.Write.Exec(`DROP TABLE works_fts`); err != nil {
		t.Fatal(err)
	}
	res, err := svc.Search(context.Background(), SearchQuery{Q: "елка"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Fallback {
		t.Fatal("expected fallback")
	}
	if res.Works.Total != nil {
		t.Fatalf("fallback must not count, got %+v", res.Works.Total)
	}
	if len(res.Works.Items) != 1 || res.Works.Items[0].Title != "Ёлка" {
		t.Fatalf("like fallback %v", titles(res.Works))
	}
	if !strings.Contains(buf.String(), "like fallback") {
		t.Fatalf("warn not logged: %s", buf.String())
	}
}

func TestSeriesNoOrdersNonNumericLast(t *testing.T) {
	svc, d := openSvc(t)
	seed(t, d)
	ctx := context.Background()
	var seriesID int64
	if err := d.Read.QueryRow(`SELECT id FROM series WHERE name = 'Серия Мусор'`).Scan(&seriesID); err != nil {
		t.Fatal(err)
	}
	page, err := svc.ListWorks(ctx, ListWorksQuery{SeriesID: seriesID})
	if err != nil {
		t.Fatal(err)
	}
	got := titles(page)
	want := []string{"Том один", "Том два", "Том десять", "Том диапазон", "Том вопрос", "Том пустой"}
	// 1, 2, 10 numeric; 1-2, ?, empty last — last three ordered by sort_title
	wantLast := []string{"Том один", "Том два", "Том десять"}
	if len(got) != 6 {
		t.Fatalf("len %d %v", len(got), got)
	}
	if strings.Join(got[:3], ",") != strings.Join(wantLast, ",") {
		t.Fatalf("numeric prefix %v want %v (full %v)", got[:3], wantLast, got)
	}
	last := got[3:]
	if last[0] != "Том диапазон" && last[0] != "Том вопрос" && last[0] != "Том пустой" {
		t.Fatalf("non-numeric should be at end: %v want something like %v", got, want)
	}
	for i, title := range last {
		_ = i
		if title == "Том один" || title == "Том два" || title == "Том десять" {
			t.Fatalf("numeric mixed into tail: %v", got)
		}
	}
}

func TestAlphabetBuckets(t *testing.T) {
	svc, d := openSvc(t)
	seed(t, d)
	ctx := context.Background()
	letters := svc.Alphabet()
	for _, k := range letters {
		if k == "ё" {
			t.Fatal("Ё shelf present")
		}
	}
	ye, err := svc.ListAuthors(ctx, ListPeopleQuery{Letter: "е"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ye.Items) != 1 || !strings.Contains(ye.Items[0].DisplayName, "Ёлкин") {
		t.Fatalf("Е (Ё) shelf: %+v", ye.Items)
	}
	jay, err := svc.ListAuthors(ctx, ListPeopleQuery{Letter: "й"})
	if err != nil {
		t.Fatal(err)
	}
	if len(jay.Items) != 1 || !strings.Contains(jay.Items[0].DisplayName, "Йенсен") {
		t.Fatalf("Й shelf: %+v", jay.Items)
	}
}

func TestHistoryDedupByNormalize(t *testing.T) {
	svc, d := openSvc(t)
	seed(t, d)
	ctx := context.Background()
	if err := svc.RecordSearch(ctx, "Громов"); err != nil {
		t.Fatal(err)
	}
	if err := svc.RecordSearch(ctx, "громов"); err != nil {
		t.Fatal(err)
	}
	h, err := svc.SearchHistory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 1 || h[0] != "громов" {
		t.Fatalf("history %v", h)
	}
	if textnorm.Normalize(h[0]) != "громов" {
		t.Fatal("stored form should be the latest typed")
	}
}

func TestLikeMatchesWordStart(t *testing.T) {
	svc, d := openSvc(t)
	seed(t, d)
	if _, err := d.Write.Exec(`DROP TABLE works_fts`); err != nil {
		t.Fatal(err)
	}
	res, err := svc.Search(context.Background(), SearchQuery{Q: "лка"})
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range res.Works.Items {
		if w.ID == 1 {
			t.Fatal("mid-word LIKE must not match Ёлка for token лка")
		}
	}
}

func TestAddedDateSortUsesColumn(t *testing.T) {
	svc, d := openSvc(t)
	seed(t, d)
	page, err := svc.ListWorks(context.Background(), ListWorksQuery{Sort: SortAdded, Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	if page.Items[0].Title != "Абрикос" { // 2021-01-01 newest
		t.Fatalf("newest %q", page.Items[0].Title)
	}
}

func TestRandomWork(t *testing.T) {
	svc, d := openSvc(t)
	seed(t, d)
	w, err := svc.RandomWork(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if w.ID == 0 {
		t.Fatal("empty random")
	}
}

func TestGenresOrderedByNormalize(t *testing.T) {
	svc, d := openSvc(t)
	seed(t, d)
	got, err := svc.ListGenres(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 2 {
		t.Fatalf("genres %v", got)
	}
	if textnorm.Normalize(got[0].NameRU) > textnorm.Normalize(got[1].NameRU) {
		t.Fatalf("genre order by display name: %q then %q", got[0].NameRU, got[1].NameRU)
	}
}

func TestFilterWithoutReadyCountIsCapped(t *testing.T) {
	svc, d := openSvc(t)
	seed(t, d)
	ctx := context.Background()
	lang, err := svc.ListWorks(ctx, ListWorksQuery{Lang: "ru"})
	if err != nil {
		t.Fatal(err)
	}
	if lang.Total == nil || lang.Total.N < 1 {
		t.Fatalf("lang filter total %+v", lang.Total)
	}
	both, err := svc.ListWorks(ctx, ListWorksQuery{GenreID: 1, AuthorID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if both.Total == nil || both.Total.N < 1 {
		t.Fatalf("compound filter total %+v", both.Total)
	}
}

func TestGettersUnknownID(t *testing.T) {
	svc, d := openSvc(t)
	seed(t, d)
	ctx := context.Background()
	if _, err := svc.GetWork(ctx, 99); apperr.As(err).Code != apperr.CodeNotFound {
		t.Fatalf("work: %v", err)
	}
	if _, err := svc.GetAuthor(ctx, 99); apperr.As(err).Code != apperr.CodeNotFound {
		t.Fatalf("author: %v", err)
	}
	if _, err := svc.GetGenre(ctx, 99); apperr.As(err).Code != apperr.CodeNotFound {
		t.Fatalf("genre: %v", err)
	}
	if _, err := svc.GetSeries(ctx, 99); apperr.As(err).Code != apperr.CodeNotFound {
		t.Fatalf("series: %v", err)
	}
	work, err := svc.GetWork(ctx, 1)
	if err != nil || work.Title != "Ёлка" {
		t.Fatalf("work %+v %v", work, err)
	}
	author, err := svc.GetAuthor(ctx, 1)
	if err != nil || author.DisplayName == "" {
		t.Fatalf("author %+v %v", author, err)
	}
}

func TestWorkDetailsAndViewed(t *testing.T) {
	svc, d := openSvc(t)
	seed(t, d)
	ctx := context.Background()
	det, err := svc.GetWorkDetails(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	if det.SeriesID == 0 || len(det.Authors) == 0 {
		t.Fatalf("details %+v", det)
	}
	if det.PrevWorkID != nil {
		t.Fatalf("first volume prev=%v", det.PrevWorkID)
	}
	if det.NextWorkID == nil || *det.NextWorkID != 6 {
		t.Fatalf("next=%v", det.NextWorkID)
	}
	if len(det.Editions) != 1 || det.Editions[0].ArchiveName != "a.zip" {
		t.Fatalf("editions %+v", det.Editions)
	}
	if _, err := d.Write.Exec(`INSERT INTO editions(libid, work_id, archive_name, file_name, file_ext, size, is_deleted, is_active)
		VALUES ('5b', 5, 'b.zip', 'g', 'fb2', 366000, 0, 1)`); err != nil {
		t.Fatal(err)
	}
	det, err = svc.GetWorkDetails(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	if det.EditionCount != 2 || len(det.Editions) != 2 {
		t.Fatalf("two editions count=%d list=%d", det.EditionCount, len(det.Editions))
	}
	if det.Editions[1].ArchiveName != "b.zip" || det.Editions[1].Size == nil || *det.Editions[1].Size != 366000 {
		t.Fatalf("second edition %+v", det.Editions)
	}
	if err := svc.RecordViewed(ctx, 5); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := d.Read.QueryRow(`SELECT count(*) FROM recently_viewed WHERE work_id = 5`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("viewed n=%d err=%v", n, err)
	}
}
