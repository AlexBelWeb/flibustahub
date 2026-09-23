package app

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/platform"
)

func TestShutdownStopsFocusWhileLockHeld(t *testing.T) {
	svc := newTestService(t)
	dir := svc.cfg.Live().DataDir
	unlock, primary, err := platform.AcquireInstance(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !primary {
		t.Fatal("test must hold the instance lock")
	}
	stop, err := platform.ListenFocus(dir, func() {})
	if err != nil {
		unlock()
		t.Fatal(err)
	}
	svc.BindInstance(platform.InstanceHold{StopFocus: stop, Unlock: unlock})

	svc.Shutdown(func(context.Context) error {
		if platform.SignalFocus(dir, 300*time.Millisecond) {
			t.Error("focus channel must be silent once shutdown has started")
		}
		_, again, acqErr := platform.AcquireInstance(dir)
		if acqErr != nil {
			t.Error(acqErr)
		}
		if again {
			t.Error("catalog lock must still be held during shutdown")
		}
		return nil
	})

	rel, primary, err := platform.AcquireInstance(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !primary {
		t.Fatal("lock must be released after the catalog is closed")
	}
	rel()
}

func TestShutdownKeepsLockWhenCatalogCloseTimesOut(t *testing.T) {
	svc := newTestService(t)
	svc.stopBudget = 0
	dir := svc.cfg.Live().DataDir
	catalog, err := db.Open(context.Background(), db.Options{
		Path:       filepath.Join(dir, "catalog.sqlite"),
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := catalog.Read.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	svc.AttachCatalog(catalog, nil)

	unlock, primary, err := platform.AcquireInstance(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !primary {
		t.Fatal("test must hold the instance lock")
	}
	svc.BindInstance(platform.InstanceHold{Unlock: unlock})

	svc.Shutdown(nil)

	_, again, err := platform.AcquireInstance(dir)
	if err != nil {
		t.Fatal(err)
	}
	if again {
		t.Fatal("lock must stay held when the catalog did not close")
	}
}
