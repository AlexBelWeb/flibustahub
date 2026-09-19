package app

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/migrations"
)

func TestBootstrapOpenCatalogRace(t *testing.T) {
	svc := newTestService(t)
	t.Cleanup(svc.CloseCatalog)

	var done atomic.Bool
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for !done.Load() {
				_ = svc.Bootstrap()
			}
		}()
	}
	if err := svc.OpenCatalog(); err != nil {
		done.Store(true)
		wg.Wait()
		t.Fatal(err)
	}
	done.Store(true)
	wg.Wait()
	got := svc.Bootstrap()
	if !got.CatalogReady {
		t.Fatal("catalog should be ready after OpenCatalog")
	}
}

func TestRetryStartupSingleFlight(t *testing.T) {
	svc := newTestService(t)
	t.Cleanup(svc.CloseCatalog)

	pause := make(chan struct{})
	svc.openPause = pause

	done := make(chan error, 1)
	go func() {
		done <- svc.OpenCatalog()
	}()

	deadline := time.Now().Add(3 * time.Second)
	for !svc.Bootstrap().CatalogOpening {
		if time.Now().After(deadline) {
			t.Fatal("open never claimed the slot")
		}
		time.Sleep(time.Millisecond)
	}

	start := time.Now()
	got := svc.RetryStartup()
	if time.Since(start) > 200*time.Millisecond {
		t.Fatalf("RetryStartup waited %s for the in-flight open", time.Since(start))
	}
	if !got.CatalogOpening {
		t.Fatal("in-flight retry must report catalogOpening")
	}
	if got.DatabaseUpdating {
		t.Fatal("empty catalog open must not report databaseUpdating")
	}
	if got.CatalogReady {
		t.Fatal("second caller must not see a catalog from a nested open")
	}

	close(pause)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	ready := svc.Bootstrap()
	if !ready.CatalogReady {
		t.Fatal("first open should finish after the pause")
	}
	if ready.CatalogOpening {
		t.Fatal("opening must clear when the open finishes")
	}
	if ready.DatabaseUpdating {
		t.Fatal("updating must clear when the open finishes")
	}
}

func TestOpenCatalogDoesNotReportUpdatingWithoutPending(t *testing.T) {
	svc := newTestService(t)
	t.Cleanup(svc.CloseCatalog)

	pause := make(chan struct{})
	svc.openPause = pause
	errc := make(chan error, 1)
	go func() {
		errc <- svc.OpenCatalog()
	}()

	deadline := time.Now().Add(3 * time.Second)
	for !svc.Bootstrap().CatalogOpening {
		if time.Now().After(deadline) {
			t.Fatal("open never claimed the slot")
		}
		time.Sleep(time.Millisecond)
	}
	for i := 0; i < 50; i++ {
		got := svc.Bootstrap()
		if got.DatabaseUpdating {
			t.Fatal("ordinary open must not report databaseUpdating")
		}
		if !got.CatalogOpening {
			t.Fatal("ordinary open must report catalogOpening until it finishes")
		}
	}

	close(pause)
	if err := <-errc; err != nil {
		t.Fatal(err)
	}
	ready := svc.Bootstrap()
	if ready.DatabaseUpdating || ready.CatalogOpening {
		t.Fatalf("flags after open: updating=%v opening=%v", ready.DatabaseUpdating, ready.CatalogOpening)
	}
	if !ready.CatalogReady {
		t.Fatal("catalog should be ready")
	}
}

func TestRetryStartupReportsUpdatingWhenPending(t *testing.T) {
	svc := newTestService(t)
	t.Cleanup(svc.CloseCatalog)
	seedPendingCatalog(t, svc)
	svc.AttachCatalog(nil, apperr.New(apperr.CodeDBMigrateFailed, map[string]string{"name": "002_works_added_date"}))

	pause := make(chan struct{})
	svc.openPause = pause
	done := make(chan Bootstrap, 1)
	go func() {
		done <- svc.RetryStartup()
	}()

	deadline := time.Now().Add(5 * time.Second)
	var got Bootstrap
	for {
		got = svc.Bootstrap()
		if got.DatabaseUpdating {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("pending retry never reported databaseUpdating: opening=%v ready=%v", got.CatalogOpening, got.CatalogReady)
		}
		time.Sleep(time.Millisecond)
	}
	if !got.CatalogOpening {
		t.Fatal("pending retry must report catalogOpening")
	}

	close(pause)
	finished := <-done
	if finished.DatabaseUpdating || finished.CatalogOpening {
		t.Fatalf("flags after retry: updating=%v opening=%v", finished.DatabaseUpdating, finished.CatalogOpening)
	}
	if !finished.CatalogReady {
		t.Fatal("pending retry should finish with a catalog")
	}
}

func seedPendingCatalog(t *testing.T, svc *Service) {
	t.Helper()
	initial, err := migrations.FS.ReadFile("001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	paths := svc.config().Paths()
	d, err := db.Open(context.Background(), db.Options{
		Path:       paths.DBPath,
		BackupsDir: paths.BackupsDir,
		Log:        slog.New(slog.DiscardHandler),
		Migrations: fstest.MapFS{"001_initial.sql": &fstest.MapFile{Data: initial}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestShutdownDuringOpenClosesHandle(t *testing.T) {
	svc := newTestService(t)
	hold := make(chan struct{})
	svc.openHold = hold

	errc := make(chan error, 1)
	go func() {
		errc <- svc.OpenCatalog()
	}()

	deadline := time.Now().Add(15 * time.Second)
	for !svc.Bootstrap().CatalogOpening {
		if time.Now().After(deadline) {
			t.Fatal("open never claimed the slot")
		}
		time.Sleep(time.Millisecond)
	}

	shutDone := make(chan struct{})
	go func() {
		svc.Shutdown(nil)
		close(shutDone)
	}()
	close(hold)

	if err := <-errc; err != nil {
		t.Fatal(err)
	}
	<-shutDone
	if svc.Catalog() != nil {
		t.Fatal("catalog leaked after shutdown during open")
	}
}

func TestShutdownPreventsNewOpen(t *testing.T) {
	svc := newTestService(t)
	svc.Shutdown(nil)
	if err := svc.OpenCatalog(); err != nil {
		t.Fatal(err)
	}
	if svc.Catalog() != nil {
		t.Fatal("must not open a catalog after shutdown")
	}
}

func TestShutdownIsIdempotent(t *testing.T) {
	svc := newTestService(t)
	t.Cleanup(svc.CloseCatalog)
	if err := svc.OpenCatalog(); err != nil {
		t.Fatal(err)
	}
	svc.Shutdown(nil)
	svc.Shutdown(nil)
	if svc.Catalog() != nil {
		t.Fatal("catalog still open after shutdown")
	}
}
