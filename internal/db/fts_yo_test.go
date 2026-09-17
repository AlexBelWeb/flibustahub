package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// TestFTS5YoVsYe records whether unicode61 remove_diacritics 2 folds
// Cyrillic yo into ye. The assertion is the observed behaviour, not a
// wished-for outcome: do not "fix" the test to match an implementation.
func TestFTS5YoVsYe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "probe.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE VIRTUAL TABLE probe USING fts5(x, tokenize = 'unicode61 remove_diacritics 2')`); err != nil {
		t.Fatalf("FTS5 unicode61 is unavailable: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO probe(rowid, x) VALUES (1, 'ёлка')`); err != nil {
		t.Fatal(err)
	}

	yeHits := countFTS(t, db, `SELECT count(*) FROM probe WHERE probe MATCH 'елка'`)
	yoHits := countFTS(t, db, `SELECT count(*) FROM probe WHERE probe MATCH 'ёлка'`)
	cafeHits := 0

	if _, err := db.Exec(`INSERT INTO probe(rowid, x) VALUES (2, 'café')`); err != nil {
		t.Fatal(err)
	}
	cafeHits = countFTS(t, db, `SELECT count(*) FROM probe WHERE probe MATCH 'cafe'`)

	t.Logf("FTS probe: MATCH елка against indexed ёлка = %d; MATCH ёлка = %d; MATCH cafe against café = %d", yeHits, yoHits, cafeHits)

	if yoHits != 1 {
		t.Fatalf("expected the exact indexed token ёлка to match, got %d", yoHits)
	}
	if cafeHits != 1 {
		t.Fatalf("expected latin remove_diacritics 2 to fold café→cafe, got %d", cafeHits)
	}

	// Documented SQLite behaviour: remove_diacritics applies to Latin, not Cyrillic.
	// If this fails, the tokenizer DID fold yo/ye and 4.1.3 stays as originally written.
	if yeHits != 0 {
		t.Fatalf("Cyrillic yo folded onto ye (hits=%d); that contradicts SQLite docs and must be reported", yeHits)
	}
}

func countFTS(t *testing.T, db *sql.DB, q string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q).Scan(&n); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}
