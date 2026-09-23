package db

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlite3 "modernc.org/sqlite/lib"
)

func TestTransientSQLiteCode(t *testing.T) {
	if !transientSQLiteCode(sqlite3.SQLITE_BUSY) {
		t.Fatal("BUSY")
	}
	if !transientSQLiteCode(sqlite3.SQLITE_BUSY | (1 << 8)) {
		t.Fatal("extended BUSY")
	}
	if !transientSQLiteCode(sqlite3.SQLITE_IOERR_LOCK) {
		t.Fatal("IOERR_LOCK")
	}
	if !transientSQLiteCode(sqlite3.SQLITE_IOERR_UNLOCK) {
		t.Fatal("IOERR_UNLOCK")
	}
	if sqlite3.SQLITE_IOERR_TRUNCATE != 1546 || !transientSQLiteCode(sqlite3.SQLITE_IOERR_TRUNCATE) {
		t.Fatal("1546 must be retried")
	}
	if transientSQLiteCode(sqlite3.SQLITE_IOERR) {
		t.Fatal("a generic disk error must surface immediately")
	}
	if transientSQLiteCode(sqlite3.SQLITE_CORRUPT) {
		t.Fatal("corrupt")
	}
}

func TestRetryTransientSucceedsWithinBudget(t *testing.T) {
	n := 0
	err := retryTransient(context.Background(), 300*time.Millisecond, func(error) bool { return true }, func() error {
		n++
		if n >= 2 {
			return nil
		}
		return errors.New("busy")
	})
	if err != nil || n != 2 {
		t.Fatalf("err=%v n=%d", err, n)
	}
}

func TestRetryTransientGivesUp(t *testing.T) {
	start := time.Now()
	n := 0
	err := retryTransient(context.Background(), 120*time.Millisecond, func(error) bool { return true }, func() error {
		n++
		return errors.New("still locked")
	})
	if err == nil {
		t.Fatal("expected the last error")
	}
	if n < 2 {
		t.Fatalf("retries = %d", n)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("retry ran for %s", time.Since(start))
	}
}

func TestRetryTransientStopsOnPermanentError(t *testing.T) {
	n := 0
	err := retryTransient(context.Background(), time.Second, func(error) bool { return false }, func() error {
		n++
		return errors.New("disk")
	})
	if err == nil || n != 1 {
		t.Fatalf("err=%v n=%d", err, n)
	}
}
