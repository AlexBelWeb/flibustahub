package catalog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/repositories"
	"github.com/alexbelweb/flibustahub/internal/visibility"
)

func openSnapshot(t *testing.T) (*Service, *db.DB) {
	t.Helper()
	src := os.Getenv("FLIBUSTAHUB_CATALOG_SNAPSHOT")
	if src == "" {
		t.Skip("FLIBUSTAHUB_CATALOG_SNAPSHOT is not set")
	}
	dir := t.TempDir()
	dst := filepath.Join(dir, "catalog.sqlite")
	start := time.Now()
	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	d, err := db.Open(context.Background(), db.Options{
		Path:       dst,
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("snapshot copy+open %s", time.Since(start))
	t.Cleanup(func() { _ = d.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := d.WaitSearchIndex(ctx); err != nil {
		t.Fatal(err)
	}
	return New(d, slog.New(slog.DiscardHandler)), d
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func warm(t *testing.T, name string, fn func()) time.Duration {
	t.Helper()
	fn()
	start := time.Now()
	fn()
	elapsed := time.Since(start)
	t.Logf("%s %s", name, elapsed)
	return elapsed
}

func budget(t *testing.T, name string, d, max time.Duration) {
	t.Helper()
	if d > max {
		t.Errorf("%s %s exceeds %s", name, d, max)
	}
}

func explainSQL(t *testing.T, d *db.DB, q string, args ...any) string {
	t.Helper()
	rows, err := d.Read.Query("EXPLAIN QUERY PLAN "+q, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var b strings.Builder
	for rows.Next() {
		var id, parent, nu int
		var detail string
		if err := rows.Scan(&id, &parent, &nu, &detail); err != nil {
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

func TestSnapshotBudgets(t *testing.T) {
	svc, d := openSnapshot(t)
	ctx := context.Background()
	repo := repositories.NewCatalog(d)

	fts := warm(t, "fts_elka", func() {
		res, err := svc.Search(ctx, SearchQuery{Q: "елка"})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Works.Items) == 0 {
			t.Fatal("елка: no hits")
		}
	})
	budget(t, "FTS first page", fts, 150*time.Millisecond)

	freq := pickFrequentToken(t, svc)
	ftsFreq := warm(t, "fts_frequent_"+freq, func() {
		res, err := svc.Search(ctx, SearchQuery{Q: freq})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Works.Items) == 0 {
			t.Fatal("frequent token: no hits")
		}
	})
	budget(t, "FTS frequent token", ftsFreq, 150*time.Millisecond)

	depth450 := warm(t, "fts_depth_450_"+freq, func() {
		res, err := svc.Search(ctx, SearchQuery{Q: freq, Offset: 450})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Works.Items) == 0 {
			t.Fatal("depth 450: no hits")
		}
	})
	t.Logf("fts depth 450 is a LIMIT/OFFSET page, not a 50ms keyset budget: %s", depth450)

	measureKeysetDepths(t, svc, d)
	measureAuthors(t, svc, d)
	measureSeries(t, svc, d)
	measureProlificAuthor(t, svc, d)
	measureThresholdCurve(t, repo, d)

	if err := db.SetFTSDirty(ctx, d.Write, true); err != nil {
		t.Fatal(err)
	}
	if svc.db.SearchIndexReady() {
		t.Fatal("expected LIKE fallback path")
	}
	for _, q := range []string{"елка", freq, freq + " елка"} {
		name := "like_" + strings.ReplaceAll(q, " ", "_")
		elapsed := warm(t, name, func() {
			res, err := svc.Search(ctx, SearchQuery{Q: q})
			if err != nil {
				t.Fatal(err)
			}
			if !res.Fallback {
				t.Fatal("expected fallback flag")
			}
			if res.Works.Total != nil {
				t.Fatalf("fallback counted %v", res.Works.Total)
			}
		})
		t.Logf("%s wall %s (not a 50ms keyset budget)", name, elapsed)
	}
}

func measureKeysetDepths(t *testing.T, svc *Service, d *db.DB) {
	t.Helper()
	ctx := context.Background()
	var listable int
	if err := d.Read.QueryRow(`SELECT CAST(value AS INTEGER) FROM app_meta WHERE key = 'works_listable'`).Scan(&listable); err != nil {
		t.Logf("works_listable missing: %v", err)
	}
	t.Logf("works_listable=%d", listable)

	type sortSpec struct {
		name string
		sort string
		kind cursorKind
		q    string
	}
	sorts := []sortSpec{
		{
			name: "title", sort: SortTitle, kind: curTitle,
			q: `SELECT w.sort_title, w.id FROM works w WHERE ` + visibility.ListableWorkSQL + ` ORDER BY w.sort_title, w.id LIMIT 1 OFFSET ?`,
		},
		{
			name: "added", sort: SortAdded, kind: curAdded,
			q: `SELECT w.added_date, w.id FROM works w WHERE ` + visibility.ListableWorkSQL + ` ORDER BY w.added_date DESC, w.id DESC LIMIT 1 OFFSET ?`,
		},
	}
	depths := []int{1000, 50000, 300000, 550000}
	for _, sp := range sorts {
		var times []time.Duration
		var lastV string
		var lastID int64
		var measured []int
		for _, depth := range depths {
			off := depth - 1
			if listable > 0 && off >= listable {
				t.Logf("%s skip depth %d: listable=%d", sp.name, depth, listable)
				continue
			}
			var v string
			var id int64
			setup := time.Now()
			if err := d.Read.QueryRow(sp.q, off).Scan(&v, &id); err != nil {
				t.Fatalf("%s cursor depth %d: %v", sp.name, depth, err)
			}
			t.Logf("%s depth %d setup %s v=%q id=%d (OFFSET is setup only, not the measured query)", sp.name, depth, time.Since(setup), v, id)
			cur := encodeCursor(pageCursor{K: sp.kind, V: v, ID: id})
			elapsed := warm(t, fmt.Sprintf("keyset_%s_depth_%d", sp.name, depth), func() {
				page, err := svc.ListWorks(ctx, ListWorksQuery{Sort: sp.sort, Limit: 50, Cursor: cur})
				if err != nil {
					t.Fatal(err)
				}
				if len(page.Items) == 0 {
					t.Fatalf("%s page empty at depth %d", sp.name, depth)
				}
			})
			budget(t, fmt.Sprintf("%s keyset depth %d", sp.name, depth), elapsed, 50*time.Millisecond)
			times = append(times, elapsed)
			measured = append(measured, depth)
			lastV, lastID = v, id
		}
		if len(times) >= 2 {
			first, last := times[0], times[len(times)-1]
			t.Logf("%s keyset depths %v times %v", sp.name, measured, times)
			if last > 15*time.Millisecond && last > 3*first {
				t.Errorf("%s keyset grows with depth: %v (predicate is not using the index)", sp.name, times)
			}
		}
		p := repositories.WorkListParams{HasCursor: true, Limit: 50, AfterID: lastID}
		if sp.name == "added" {
			p.Sort = "added"
			p.AfterAdded = lastV
		} else {
			p.AfterTitle = lastV
		}
		q, args := repositories.WorkListSQL(p)
		plan := explainSQL(t, d, q, args...)
		t.Logf("list %s plan (repository SQL):\n%s", sp.name, plan)
		if strings.Contains(strings.ToLower(plan), "ifnull") {
			t.Errorf("%s keyset plan uses ifnull", sp.name)
		}
		if sp.name == "added" && !strings.Contains(plan, "idx_works_added") {
			t.Errorf("added_date keyset plan does not use idx_works_added:\n%s", plan)
		}
		if sp.name == "title" && !strings.Contains(plan, "idx_works_sort_title") {
			t.Errorf("title keyset plan does not use idx_works_sort_title:\n%s", plan)
		}
		lower := strings.ToLower(plan)
		if strings.Contains(lower, "scan w") && !strings.Contains(plan, "sort_title>?") && !strings.Contains(plan, "sort_title>=?") && !strings.Contains(plan, "added_date<?") && !strings.Contains(plan, "added_date<=?") {
			t.Errorf("%s keyset plan looks like a scan from the index start (hidden OFFSET):\n%s", sp.name, plan)
		}
	}
}

func pickFrequentToken(t *testing.T, svc *Service) string {
	t.Helper()
	ctx := context.Background()
	for _, q := range []string{"история", "жизнь", "любовь", "война", "мир", "день"} {
		res, err := svc.Search(ctx, SearchQuery{Q: q, Limit: 1})
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		if res.Works.Total != nil {
			n = res.Works.Total.N
		}
		t.Logf("token %q hits=%d capped=%v", q, n, res.Works.Total != nil && res.Works.Total.Capped)
		if n >= 500 || (res.Works.Total != nil && res.Works.Total.Capped) {
			return q
		}
	}
	return "история"
}

func measureAuthors(t *testing.T, svc *Service, d *db.DB) {
	t.Helper()
	ctx := context.Background()
	var n int
	if err := d.Read.QueryRow(`SELECT count(*) FROM authors WHERE work_count > 0 AND sort_name >= 'а' AND sort_name < 'б'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	t.Logf("authors letter а count=%d", n)
	off := n / 2
	if off > 2000 {
		off = 2000
	}
	if off < 40 {
		off = 0
	}
	var sortName string
	var id int64
	if off > 0 {
		if err := d.Read.QueryRow(`SELECT sort_name, id FROM authors WHERE work_count > 0 AND sort_name >= 'а' AND sort_name < 'б'
			ORDER BY sort_name, id LIMIT 1 OFFSET ?`, off).Scan(&sortName, &id); err != nil {
			t.Fatal(err)
		}
	}
	cur := ""
	if id > 0 {
		cur = encodeCursor(pageCursor{K: curName, V: sortName, ID: id})
	}
	elapsed := warm(t, fmt.Sprintf("authors_letter_a_depth_%d", off), func() {
		page, err := svc.ListAuthors(ctx, ListPeopleQuery{Letter: "а", Cursor: cur, Limit: 40})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) == 0 {
			t.Fatal("author letter page empty")
		}
	})
	budget(t, "authors letter deep keyset", elapsed, 50*time.Millisecond)
}

func measureSeries(t *testing.T, svc *Service, d *db.DB) {
	t.Helper()
	ctx := context.Background()
	page, err := svc.ListSeries(ctx, ListPeopleQuery{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if page.NextCursor == "" {
		t.Fatal("series cursor empty")
	}
	cur, ok := decodeCursor(page.NextCursor, curName)
	if !ok {
		t.Fatal("series cursor decode")
	}
	elapsed := warm(t, "series_keyset", func() {
		p, err := svc.ListSeries(ctx, ListPeopleQuery{Limit: 50, Cursor: page.NextCursor})
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Items) == 0 {
			t.Fatal("series next page empty")
		}
	})
	budget(t, "series listing keyset", elapsed, 50*time.Millisecond)
	nextSQL, nextArgs, ok := repositories.SeriesListSQL(repositories.NameListParams{
		HasCursor: true, AfterName: cur.V, AfterID: cur.ID, Limit: 50,
	})
	if !ok {
		t.Fatal("series next SQL")
	}
	nextPlan := explainSQL(t, d, nextSQL, nextArgs...)
	t.Logf("series next page plan (repository SQL):\n%s", nextPlan)
	if !strings.Contains(nextPlan, "idx_series_sort") {
		t.Logf("series next page did not mention idx_series_sort")
	}

	letter := "а"
	letterPage, err := svc.ListSeries(ctx, ListPeopleQuery{Letter: letter, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(letterPage.Items) == 0 {
		t.Fatalf("series letter %s empty", letter)
	}
	letterElapsed := warm(t, "series_letter_"+letter, func() {
		p, err := svc.ListSeries(ctx, ListPeopleQuery{Letter: letter, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Items) == 0 {
			t.Fatal("series letter page empty")
		}
	})
	budget(t, "series letter "+letter, letterElapsed, 50*time.Millisecond)
	letterSQL, letterArgs, ok := repositories.SeriesListSQL(repositories.NameListParams{Letter: letter, Limit: 50})
	if !ok {
		t.Fatal("series letter SQL")
	}
	letterPlan := explainSQL(t, d, letterSQL, letterArgs...)
	t.Logf("series letter %s plan (repository SQL):\n%s", letter, letterPlan)
	if !strings.Contains(letterPlan, "idx_series_sort") {
		t.Logf("series letter plan did not mention idx_series_sort")
	}
}

func measureProlificAuthor(t *testing.T, svc *Service, d *db.DB) {
	t.Helper()
	ctx := context.Background()
	var id int64
	var n int
	if err := d.Read.QueryRow(`SELECT id, work_count FROM authors ORDER BY work_count DESC LIMIT 1`).Scan(&id, &n); err != nil {
		t.Fatal(err)
	}
	t.Logf("prolific author id=%d work_count=%d", id, n)
	first := warm(t, "prolific_author_page", func() {
		page, err := svc.ListWorks(ctx, ListWorksQuery{AuthorID: id, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) == 0 {
			t.Fatal("prolific author page empty")
		}
	})
	budget(t, "prolific author first page", first, 50*time.Millisecond)
	if n < 80 {
		return
	}
	page, err := svc.ListWorks(ctx, ListWorksQuery{AuthorID: id, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	elapsed := warm(t, "prolific_author_next", func() {
		p, err := svc.ListWorks(ctx, ListWorksQuery{AuthorID: id, Limit: 50, Cursor: page.NextCursor})
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Items) == 0 {
			t.Fatal("prolific author next page empty")
		}
	})
	budget(t, "prolific author next page", elapsed, 50*time.Millisecond)
}

func measureThresholdCurve(t *testing.T, repo *repositories.Catalog, d *db.DB) {
	t.Helper()
	ctx := context.Background()
	targets := []int{1, 10, 50, 100, 500, 1000, 2000, 5000, 8000, 10000, 20000, 50000}
	type pt struct {
		id, n int64
	}
	var points []pt
	for _, want := range targets {
		var id int64
		var n int
		err := d.Read.QueryRow(`SELECT id, work_count FROM genres WHERE work_count > 0
			ORDER BY abs(work_count - ?) ASC, id LIMIT 1`, want).Scan(&id, &n)
		if err != nil {
			t.Fatal(err)
		}
		dup := false
		for _, p := range points {
			if p.id == id {
				dup = true
				break
			}
		}
		if !dup {
			points = append(points, pt{id: id, n: int64(n)})
		}
	}
	var maxID int64
	var maxN int
	if err := d.Read.QueryRow(`SELECT id, work_count FROM genres WHERE work_count > 0 ORDER BY work_count DESC LIMIT 1`).Scan(&maxID, &maxN); err != nil {
		t.Fatal(err)
	}
	if maxID != 0 {
		dup := false
		for _, p := range points {
			if p.id == maxID {
				dup = true
				break
			}
		}
		if !dup {
			points = append(points, pt{id: maxID, n: int64(maxN)})
		}
	}
	for _, p := range points {
		wideP := repositories.WorkListParams{Sort: "title", GenreID: p.id, Visible: true, Narrow: false, Limit: 50}
		narrowP := wideP
		narrowP.Narrow = true
		wide := warm(t, fmt.Sprintf("curve_wide_n=%d_id=%d", p.n, p.id), func() {
			if _, err := repo.ListWorkIDs(ctx, wideP); err != nil {
				t.Fatal(err)
			}
		})
		narrow := warm(t, fmt.Sprintf("curve_narrow_n=%d_id=%d", p.n, p.id), func() {
			if _, err := repo.ListWorkIDs(ctx, narrowP); err != nil {
				t.Fatal(err)
			}
		})
		t.Logf("threshold curve count=%d wide=%s narrow=%s", p.n, wide, narrow)
	}
}
