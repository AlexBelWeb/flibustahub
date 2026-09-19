package repositories

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/inpx"
)

func TestWorkListSQLPlanUsesSortTitleIndex(t *testing.T) {
	d, err := db.Open(context.Background(), db.Options{
		Path:       filepath.Join(t.TempDir(), "catalog.sqlite"),
		BackupsDir: filepath.Join(t.TempDir(), "backups"),
		Log:        slog.New(slog.DiscardHandler),
		Now:        func() time.Time { return time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	_, err = d.Write.Exec(`INSERT INTO works(id, work_key, title, sort_title, authors_text, created_at, updated_at)
		VALUES (1, 'k1', '_', ?, 'A', 't', 't'),
		       (2, 'k2', 'Абрикос', 'абрикос', 'A', 't', 't'),
		       (3, 'k3', 'Яблоко', 'яблоко', 'A', 't', 't')`, inpx.SortTitle("_"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.Write.Exec(`INSERT INTO editions(libid, work_id, archive_name, file_name, is_deleted, is_active)
		VALUES ('1', 1, 'a.zip', 'f', 0, 1),
		       ('2', 2, 'a.zip', 'f', 0, 1),
		       ('3', 3, 'a.zip', 'f', 0, 1)`)
	if err != nil {
		t.Fatal(err)
	}

	first, args := WorkListSQL(WorkListParams{Limit: 50, Visible: true})
	firstPlan := explainSQL(t, d, first, args...)
	t.Logf("title list plan:\n%s", firstPlan)
	if !strings.Contains(firstPlan, "idx_works_sort_title") {
		t.Fatalf("first page does not use idx_works_sort_title:\n%s", firstPlan)
	}
	if strings.Contains(firstPlan, "sort_title =") || strings.Contains(strings.ToLower(firstPlan), "ifnull") {
		t.Fatalf("first page plan is not a range on sort_title:\n%s", firstPlan)
	}

	next, nextArgs := WorkListSQL(WorkListParams{HasCursor: true, AfterTitle: "абрикос", AfterID: 2, Limit: 50, Visible: true})
	nextPlan := explainSQL(t, d, next, nextArgs...)
	t.Logf("title keyset plan:\n%s", nextPlan)
	if !strings.Contains(nextPlan, "idx_works_sort_title") {
		t.Fatalf("keyset does not use idx_works_sort_title:\n%s", nextPlan)
	}
	if !strings.Contains(nextPlan, "sort_title>?") && !strings.Contains(nextPlan, "sort_title>=?") {
		t.Fatalf("keyset plan is not a range seek:\n%s", nextPlan)
	}

	var firstTitle string
	if err := d.Read.QueryRow(`SELECT title FROM works ORDER BY sort_title, id LIMIT 1`).Scan(&firstTitle); err != nil {
		t.Fatal(err)
	}
	if firstTitle != "Абрикос" {
		t.Fatalf("sentinel must sort last, first=%q", firstTitle)
	}
}

func explainSQL(t *testing.T, d *db.DB, query string, args ...any) string {
	t.Helper()
	rows, err := d.Read.Query("EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
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
