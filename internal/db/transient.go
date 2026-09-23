package db

import (
	"context"
	"errors"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// transientOpenBudget is how long a startup open retries a lock left by a
// process that was killed. A persistent disk error is not in this set and
// returns on the first attempt.
const transientOpenBudget = 2 * time.Second

// transientLockIO reports SQLite results that show up for a moment after a
// process is killed: BUSY (and its extended codes), IOERR_LOCK, IOERR_UNLOCK,
// and IOERR_TRUNCATE. The last one is result 1546 in this SQLite build; that
// is the code a killed process left on the next open.
func transientLockIO(err error) bool {
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	return transientSQLiteCode(se.Code())
}

func transientSQLiteCode(code int) bool {
	switch code & 0xff {
	case sqlite3.SQLITE_BUSY:
		return true
	case sqlite3.SQLITE_IOERR:
		switch code {
		case sqlite3.SQLITE_IOERR_LOCK, sqlite3.SQLITE_IOERR_UNLOCK, sqlite3.SQLITE_IOERR_TRUNCATE:
			return true
		}
	}
	return false
}

func retryTransient(ctx context.Context, budget time.Duration, again func(error) bool, op func() error) error {
	if budget <= 0 {
		return op()
	}
	deadline := time.Now().Add(budget)
	var err error
	for {
		err = op()
		if err == nil || !again(err) || !time.Now().Before(deadline) {
			return err
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return err
		case <-timer.C:
		}
	}
}
