package handlers

import (
	"sync"
	"time"
)

const closeAckTimeout = 2 * time.Second

// closeGuard decides whether a window-close during import should be blocked.
// First close: prevent and wait for the UI (or timeout). Second close: allow.
type closeGuard struct {
	mu        sync.Mutex
	attempts  int
	gen       int
	timer     *time.Timer
	force     bool
	timeout   time.Duration
	onTimeout func()
}

func (g *closeGuard) ackTimeout() time.Duration {
	if g.timeout > 0 {
		return g.timeout
	}
	return closeAckTimeout
}

func (g *closeGuard) prevent() bool {
	g.mu.Lock()
	if g.force {
		g.mu.Unlock()
		return false
	}
	g.attempts++
	n := g.attempts
	gen := g.gen
	if n >= 2 {
		g.mu.Unlock()
		return false
	}
	if g.timer != nil {
		g.timer.Stop()
	}
	timeout := g.ackTimeout()
	g.timer = time.AfterFunc(timeout, func() {
		g.mu.Lock()
		fire := g.gen == gen && g.attempts == 1 && !g.force
		if fire {
			g.force = true
		}
		g.mu.Unlock()
		if fire && g.onTimeout != nil {
			g.onTimeout()
		}
	})
	g.mu.Unlock()
	return true
}

func (g *closeGuard) dismiss() {
	g.mu.Lock()
	g.gen++
	g.attempts = 0
	if g.timer != nil {
		g.timer.Stop()
		g.timer = nil
	}
	g.mu.Unlock()
}

func (g *closeGuard) shouldForce() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.force
}
