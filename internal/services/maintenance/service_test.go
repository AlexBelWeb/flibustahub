package maintenance

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/db"
)

func openMaint(t *testing.T, extra ...func(*Options)) (*Service, *db.DB) {
	t.Helper()
	d, err := db.Open(context.Background(), db.Options{
		Path:       filepath.Join(t.TempDir(), "catalog.sqlite"),
		BackupsDir: filepath.Join(t.TempDir(), "backups"),
		Log:        slog.New(slog.DiscardHandler),
		Now:        func() time.Time { return time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	opt := Options{
		Catalog:  func() *db.DB { return d },
		DiskFree: func(string) (uint64, error) { return 1 << 40, nil },
		Log:      slog.New(slog.DiscardHandler),
	}
	for _, fn := range extra {
		fn(&opt)
	}
	return New(opt), d
}

func TestOptimizeRejectsWhenDiskFull(t *testing.T) {
	svc, d := openMaint(t)
	before, err := d.FileSize()
	if err != nil {
		t.Fatal(err)
	}
	need := spaceNeed(before)
	if need == 0 {
		t.Fatal("expected a non-empty catalog")
	}
	svc.diskFree = func(string) (uint64, error) { return need - 1, nil }
	_, err = svc.Optimize(context.Background())
	got := apperr.As(err)
	if got.Code != apperr.CodeDBNoSpace {
		t.Fatalf("code = %v", err)
	}
	if got.Params["need"] != strconv.FormatUint(need, 10) {
		t.Fatalf("need = %q, want %d (2× file size %d)", got.Params["need"], need, before)
	}
	if svc.Running() {
		t.Fatal("busy flag leaked")
	}
}

func TestSpaceNeedIsTwiceTheFile(t *testing.T) {
	if spaceNeed(100) != 200 {
		t.Fatalf("spaceNeed(100) = %d", spaceNeed(100))
	}
	if spaceNeed(0) != 0 || spaceNeed(-5) != 0 {
		t.Fatal("empty or negative size must not demand space")
	}
}

func TestOptimizeRejectsDuringImport(t *testing.T) {
	svc, _ := openMaint(t)
	svc.importing = func() bool { return true }
	_, err := svc.Optimize(context.Background())
	if apperr.As(err).Code != apperr.CodeImportInProgress {
		t.Fatalf("optimize code = %v", err)
	}
	_, err = svc.Backup(context.Background())
	if apperr.As(err).Code != apperr.CodeImportInProgress {
		t.Fatalf("backup code = %v", err)
	}
}

func TestOptimizeRejectsDuringCoverWarmup(t *testing.T) {
	svc, _ := openMaint(t)
	svc.warming = func() bool { return true }
	_, err := svc.Optimize(context.Background())
	if apperr.As(err).Code != apperr.CodeCoverWarmupInProgress {
		t.Fatalf("code = %v", err)
	}
	_, err = svc.Backup(context.Background())
	if apperr.As(err).Code != apperr.CodeCoverWarmupInProgress {
		t.Fatalf("backup code = %v", err)
	}
}

func TestOptimizeRejectsParallelStart(t *testing.T) {
	hold := make(chan struct{})
	svc, _ := openMaint(t, func(o *Options) { o.Hold = hold })
	errCh := make(chan error, 1)
	go func() {
		_, err := svc.Optimize(context.Background())
		errCh <- err
	}()
	deadline := time.Now().Add(2 * time.Second)
	for !svc.Running() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !svc.Running() {
		t.Fatal("first optimize did not take the lock")
	}
	_, err := svc.Optimize(context.Background())
	if apperr.As(err).Code != apperr.CodeDBMaintenanceBusy {
		t.Fatalf("second optimize = %v", err)
	}
	_, err = svc.Backup(context.Background())
	if apperr.As(err).Code != apperr.CodeDBMaintenanceBusy {
		t.Fatalf("backup during optimize = %v", err)
	}
	close(hold)
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestOptimizeReportsFreedBytes(t *testing.T) {
	svc, d := openMaint(t)
	blob := bytes.Repeat([]byte("x"), 100_000)
	if _, err := d.Write.Exec(`CREATE TABLE waste(b BLOB)`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		if _, err := d.Write.Exec(`INSERT INTO waste(b) VALUES (?)`, blob); err != nil {
			t.Fatal(err)
		}
	}
	var busy, logFrames, checkpointed int
	if err := d.Write.QueryRow(`PRAGMA wal_checkpoint(TRUNCATE)`).Scan(&busy, &logFrames, &checkpointed); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write.Exec(`DROP TABLE waste`); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Optimize(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.BytesFreed != got.BytesBefore-got.BytesAfter {
		t.Fatalf("freed %d != before %d - after %d", got.BytesFreed, got.BytesBefore, got.BytesAfter)
	}
	if got.BytesAfter >= got.BytesBefore {
		t.Fatalf("expected the file to shrink, before=%d after=%d", got.BytesBefore, got.BytesAfter)
	}
	if got.BytesFreed <= 0 {
		t.Fatalf("expected positive freed bytes, got %+v", got)
	}
}

func TestOptimizeClampsNegativeFreedBytes(t *testing.T) {
	svc, _ := openMaint(t)
	first, err := svc.Optimize(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Optimize(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if second.BytesFreed < 0 {
		t.Fatalf("freed must not go negative: first=%+v second=%+v", first, second)
	}
	if second.BytesFreed != clampFreed(second.BytesBefore, second.BytesAfter) {
		t.Fatalf("clamp mismatch: %+v", second)
	}
}

func TestBackupUsesManualReason(t *testing.T) {
	svc, _ := openMaint(t)
	got, err := svc.Backup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Base(got.Path)
	if !strings.HasPrefix(base, "catalog-manual-") || !strings.HasSuffix(base, ".sqlite") {
		t.Fatalf("path = %q", got.Path)
	}
	if _, err := os.Stat(got.Path); err != nil {
		t.Fatal(err)
	}
}

func TestBackupRejectsParallelWithOptimize(t *testing.T) {
	hold := make(chan struct{})
	svc, _ := openMaint(t, func(o *Options) { o.Hold = hold })
	var started sync.WaitGroup
	started.Add(1)
	errCh := make(chan error, 1)
	go func() {
		started.Done()
		_, err := svc.Backup(context.Background())
		errCh <- err
	}()
	started.Wait()
	deadline := time.Now().Add(2 * time.Second)
	for !svc.Running() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !svc.Running() {
		t.Fatal("backup did not take the lock")
	}
	_, err := svc.Optimize(context.Background())
	if apperr.As(err).Code != apperr.CodeDBMaintenanceBusy {
		t.Fatalf("optimize during backup = %v", err)
	}
	close(hold)
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}
