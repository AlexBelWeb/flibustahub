package handlers

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestCloseGuardSecondAttemptAllows(t *testing.T) {
	g := closeGuard{timeout: time.Hour}
	if !g.prevent() {
		t.Fatal("first close must prevent")
	}
	if g.prevent() {
		t.Fatal("second close must allow")
	}
}

func TestCloseGuardDismissResets(t *testing.T) {
	g := closeGuard{timeout: time.Hour}
	if !g.prevent() {
		t.Fatal("prevent")
	}
	g.dismiss()
	if !g.prevent() {
		t.Fatal("after dismiss, next close is first again")
	}
}

func TestCloseGuardTimeoutForces(t *testing.T) {
	var fired atomic.Bool
	g := closeGuard{
		timeout: 20 * time.Millisecond,
		onTimeout: func() {
			fired.Store(true)
		},
	}
	if !g.prevent() {
		t.Fatal("prevent")
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if fired.Load() && g.shouldForce() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timeout did not force quit")
}

func TestCloseGuardDismissCancelsTimeout(t *testing.T) {
	var fired atomic.Bool
	g := closeGuard{
		timeout: 30 * time.Millisecond,
		onTimeout: func() {
			fired.Store(true)
		},
	}
	if !g.prevent() {
		t.Fatal("prevent")
	}
	g.dismiss()
	time.Sleep(80 * time.Millisecond)
	if fired.Load() || g.shouldForce() {
		t.Fatal("dismissed close must not quit")
	}
}

func TestLongOpCloseKind(t *testing.T) {
	if got := longOpCloseKind(true, false); got != CloseKindImport {
		t.Fatalf("import: %q", got)
	}
	if got := longOpCloseKind(false, true); got != CloseKindMaintenance {
		t.Fatalf("maintenance: %q", got)
	}
	if got := longOpCloseKind(true, true); got != CloseKindImport {
		t.Fatalf("import wins when both: %q", got)
	}
	if got := longOpCloseKind(false, false); got != "" {
		t.Fatalf("idle: %q", got)
	}
}
