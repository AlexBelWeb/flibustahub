package db

import (
	"context"
	"sync"
	"testing"
	"time"
)

func (d *DB) setIdleDelay(delay time.Duration) {
	d.idle.mu.Lock()
	d.idle.delay = delay
	d.idle.mu.Unlock()
}

func (d *DB) releaseCount() int {
	d.idle.mu.Lock()
	defer d.idle.mu.Unlock()
	return d.idle.releases
}

func TestIdleReleaseFiresOncePerAwayPeriod(t *testing.T) {
	d := openTest(t)
	d.setIdleDelay(30 * time.Millisecond)

	d.WindowAway()
	time.Sleep(10 * time.Millisecond)
	if got := d.releaseCount(); got != 0 {
		t.Fatalf("released before the idle delay: %d", got)
	}
	time.Sleep(50 * time.Millisecond)
	if got := d.releaseCount(); got != 1 {
		t.Fatalf("releases = %d, want 1", got)
	}
	d.WindowAway()
	time.Sleep(50 * time.Millisecond)
	if got := d.releaseCount(); got != 1 {
		t.Fatalf("second away in the same period released again: %d", got)
	}

	d.WindowBack()
	d.WindowAway()
	time.Sleep(50 * time.Millisecond)
	if got := d.releaseCount(); got != 2 {
		t.Fatalf("new away period releases = %d, want 2", got)
	}
}

func TestIdleReleaseCancelledByReturn(t *testing.T) {
	d := openTest(t)
	d.setIdleDelay(40 * time.Millisecond)
	d.WindowAway()
	time.Sleep(10 * time.Millisecond)
	d.WindowBack()
	time.Sleep(50 * time.Millisecond)
	if got := d.releaseCount(); got != 0 {
		t.Fatalf("released after focus returned: %d", got)
	}
}

func TestIdleReleaseSkippedDuringHeavy(t *testing.T) {
	d := openTest(t)
	d.setIdleDelay(20 * time.Millisecond)
	d.BeginHeavy()
	d.WindowAway()
	time.Sleep(50 * time.Millisecond)
	if got := d.releaseCount(); got != 0 {
		t.Fatalf("released during a heavy operation: %d", got)
	}
	d.EndHeavy()
	if got := d.releaseCount(); got != 1 {
		t.Fatalf("releases after heavy = %d, want 1", got)
	}
	time.Sleep(40 * time.Millisecond)
	if got := d.releaseCount(); got != 1 {
		t.Fatalf("idle fired again after the heavy release: %d", got)
	}
}

func TestQueryDuringCacheRelease(t *testing.T) {
	d := openTest(t)
	var wg sync.WaitGroup
	errCh := make(chan error, 8)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			var n int
			err := d.Read.QueryRowContext(ctx, `SELECT count(*) FROM works`).Scan(&n)
			if err != nil {
				errCh <- err
			}
		}()
	}
	d.releaseCache()
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}
