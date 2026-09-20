package repositories

import (
	"strings"
	"testing"
)

func TestWorkListSQLKeysetIsTuple(t *testing.T) {
	title, titleArgs := WorkListSQL(WorkListParams{HasCursor: true, AfterTitle: "яблоко", AfterID: 9, Limit: 50})
	if !strings.Contains(title, "(w.sort_title, w.id) > (?, ?)") {
		t.Fatalf("title keyset is not a row-value predicate:\n%s", title)
	}
	if !strings.Contains(title, "w.sort_title >= ?") {
		t.Fatalf("title keyset missing leftmost seek:\n%s", title)
	}
	if strings.Contains(title, "sort_title > ? OR") {
		t.Fatalf("title keyset fell back to OR form:\n%s", title)
	}
	if len(titleArgs) != 4 {
		t.Fatalf("title args %d = %v", len(titleArgs), titleArgs)
	}

	added, addedArgs := WorkListSQL(WorkListParams{Sort: "added", HasCursor: true, AfterAdded: "2020-01-01", AfterID: 9, Limit: 50})
	if !strings.Contains(added, "(w.added_date, w.id) < (?, ?)") {
		t.Fatalf("added keyset is not a row-value predicate:\n%s", added)
	}
	if !strings.Contains(added, "w.added_date <= ?") {
		t.Fatalf("added keyset missing leftmost seek:\n%s", added)
	}
	if strings.Contains(added, "added_date < ? OR") {
		t.Fatalf("added keyset fell back to OR form:\n%s", added)
	}
	if len(addedArgs) != 4 {
		t.Fatalf("added args %d = %v", len(addedArgs), addedArgs)
	}

	rating, ratingArgs := WorkListSQL(WorkListParams{Sort: "rating", HasCursor: true, AfterRating: 8, AfterRateAt: "t", AfterID: 9, Limit: 50})
	if !strings.Contains(rating, "(w.rating, w.rating_updated_at, w.id) < (?, ?, ?)") {
		t.Fatalf("rating keyset is not a row-value predicate:\n%s", rating)
	}
	if !strings.Contains(rating, "ORDER BY w.rating DESC, w.rating_updated_at DESC, w.id DESC") {
		t.Fatalf("rating order mixed directions:\n%s", rating)
	}
	if strings.Contains(rating, "w.sort_title, w.id") || strings.Contains(rating, "w.sort_title, w.id DESC") {
		t.Fatalf("rating keyset still uses title:\n%s", rating)
	}
	if len(ratingArgs) != 5 {
		t.Fatalf("rating args %d = %v", len(ratingArgs), ratingArgs)
	}
}

func TestSeriesListSQLKeyset(t *testing.T) {
	q, args, ok := SeriesListSQL(NameListParams{HasCursor: true, AfterName: "я", AfterID: 9, Limit: 50})
	if !ok {
		t.Fatal("ok")
	}
	if !strings.Contains(q, "(sort_name, id) > (?, ?)") {
		t.Fatalf("series keyset is not a row-value predicate:\n%s", q)
	}
	if len(args) != 4 {
		t.Fatalf("args %d = %v", len(args), args)
	}
	letter, letterArgs, ok := SeriesListSQL(NameListParams{Letter: "а", Limit: 50})
	if !ok {
		t.Fatal("letter")
	}
	if !strings.Contains(letter, "sort_name >= ?") || !strings.Contains(letter, "sort_name < ?") {
		t.Fatalf("letter range missing:\n%s", letter)
	}
	if len(letterArgs) != 3 {
		t.Fatalf("letter args %d = %v", len(letterArgs), letterArgs)
	}
}
