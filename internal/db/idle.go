package db

import (
	"context"
	"database/sql"
	"reflect"
	"sync"
	"time"
)

const idleCacheDelay = 60 * time.Second

// cacheGate releases the SQLite page cache once per idle period and once
// after a heavy catalog operation finishes. Connections are visited one at
// a time so the read pool can still serve a query.
type cacheGate struct {
	delay time.Duration
	owner *DB

	mu       sync.Mutex
	timer    *time.Timer
	away     bool
	released bool
	heavy    int
	releases int
}

func newCacheGate(delay time.Duration) *cacheGate {
	if delay <= 0 {
		delay = idleCacheDelay
	}
	return &cacheGate{delay: delay}
}

func (g *cacheGate) stop() {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.stopTimerLocked()
	g.mu.Unlock()
}

func (g *cacheGate) stopTimerLocked() {
	if g.timer != nil {
		g.timer.Stop()
		g.timer = nil
	}
}

// WindowAway starts the idle wait. A second call in the same away period does nothing.
func (d *DB) WindowAway() {
	if d == nil || d.idle == nil {
		return
	}
	d.idle.mu.Lock()
	defer d.idle.mu.Unlock()
	if d.idle.away {
		return
	}
	d.idle.away = true
	d.idle.released = false
	d.idle.armLocked()
}

// WindowBack cancels a pending release. The next away period may release again.
func (d *DB) WindowBack() {
	if d == nil || d.idle == nil {
		return
	}
	d.idle.mu.Lock()
	defer d.idle.mu.Unlock()
	d.idle.away = false
	d.idle.released = false
	d.idle.stopTimerLocked()
}

// BeginHeavy blocks cache release for the duration of import, warmup, or maintenance.
func (d *DB) BeginHeavy() {
	if d == nil || d.idle == nil {
		return
	}
	d.idle.mu.Lock()
	defer d.idle.mu.Unlock()
	d.idle.heavy++
	d.idle.stopTimerLocked()
}

// EndHeavy releases the cache once when the last heavy operation finishes.
func (d *DB) EndHeavy() {
	if d == nil || d.idle == nil {
		return
	}
	d.idle.mu.Lock()
	if d.idle.heavy == 0 {
		d.idle.mu.Unlock()
		return
	}
	d.idle.heavy--
	if d.idle.heavy > 0 {
		d.idle.mu.Unlock()
		return
	}
	if d.idle.away {
		d.idle.released = true
	}
	d.idle.mu.Unlock()
	d.releaseCache()
}

func (g *cacheGate) armLocked() {
	g.stopTimerLocked()
	g.timer = time.AfterFunc(g.delay, g.fire)
}

func (g *cacheGate) fire() {
	g.mu.Lock()
	owner := g.owner
	if !g.away || g.released || g.heavy > 0 || owner == nil {
		g.mu.Unlock()
		return
	}
	g.released = true
	g.mu.Unlock()
	owner.releaseCache()
}

func (d *DB) releaseCache() {
	if d == nil {
		return
	}
	readN, writeN := 0, 0
	if d.Read != nil {
		readN = d.Read.Stats().OpenConnections
	}
	if d.Write != nil {
		writeN = d.Write.Stats().OpenConnections
	}
	if d.log != nil {
		d.log.Info("page cache released", "readConns", readN, "writeConns", writeN)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	shrinkPoolSequential(ctx, d.Read)
	shrinkPoolSequential(ctx, d.Write)
	if d.idle != nil {
		d.idle.mu.Lock()
		d.idle.releases++
		d.idle.mu.Unlock()
	}
}

func shrinkPoolSequential(ctx context.Context, pool *sql.DB) {
	if pool == nil {
		return
	}
	target := pool.Stats().OpenConnections
	if target < 1 {
		return
	}
	seen := make(map[uintptr]struct{}, target)
	attempts := 0
	for len(seen) < target && attempts < target*4 {
		if ctx.Err() != nil {
			return
		}
		attempts++
		conn, err := pool.Conn(ctx)
		if err != nil {
			return
		}
		var id uintptr
		_ = conn.Raw(func(dc any) error {
			if dc != nil {
				id = reflect.ValueOf(dc).Pointer()
			}
			return nil
		})
		if _, ok := seen[id]; ok && id != 0 {
			_ = conn.Close()
			continue
		}
		_, _ = conn.ExecContext(ctx, "PRAGMA shrink_memory")
		if id == 0 {
			id = uintptr(attempts)
		}
		seen[id] = struct{}{}
		_ = conn.Close()
	}
}
