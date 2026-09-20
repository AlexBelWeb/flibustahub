package db

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/platform"
)

func TestSnapshotVacuumTempStore(t *testing.T) {
	src := os.Getenv("FLIBUSTAHUB_CATALOG_SNAPSHOT")
	if src == "" {
		t.Skip("FLIBUSTAHUB_CATALOG_SNAPSHOT is not set")
	}
	t.Run("dsn_memory", func(t *testing.T) {
		observeVacuum(t, src, false)
	})
	t.Run("pinned_file", func(t *testing.T) {
		observeVacuum(t, src, true)
	})
}

func observeVacuum(t *testing.T, src string, pinTempToDisk bool) {
	t.Helper()
	dir := t.TempDir()
	dst := filepath.Join(dir, "catalog.sqlite")
	if err := copySnapshot(src, dst); err != nil {
		t.Fatal(err)
	}
	d, err := Open(context.Background(), Options{
		Path:       dst,
		BackupsDir: filepath.Join(dir, "backups"),
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := d.WaitSearchIndex(ctx); err != nil {
		t.Fatal(err)
	}

	before, err := d.FileSize()
	if err != nil {
		t.Fatal(err)
	}
	tempDir := os.TempDir()
	baselineDB := listNames(t, dir)
	baselineTemp := listNames(t, tempDir)
	basePrivate, _ := platform.ProcessPrivateBytes()
	var heap runtime.MemStats
	runtime.ReadMemStats(&heap)
	baseHeap := heap.HeapAlloc

	stop := make(chan struct{})
	var mu sync.Mutex
	seenDB := map[string]int64{}
	seenTemp := map[string]int64{}
	var peakPrivate, peakHeap uint64
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				noteNew(dir, baselineDB, seenDB, &mu)
				noteNew(tempDir, baselineTemp, seenTemp, &mu)
				if priv, err := platform.ProcessPrivateBytes(); err == nil {
					mu.Lock()
					if priv > peakPrivate {
						peakPrivate = priv
					}
					mu.Unlock()
				}
				runtime.ReadMemStats(&heap)
				mu.Lock()
				if heap.HeapAlloc > peakHeap {
					peakHeap = heap.HeapAlloc
				}
				mu.Unlock()
			}
		}
	}()

	start := time.Now()
	if err := d.optimize(context.Background(), pinTempToDisk); err != nil {
		close(stop)
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	close(stop)
	time.Sleep(250 * time.Millisecond)
	noteNew(dir, baselineDB, seenDB, &mu)
	noteNew(tempDir, baselineTemp, seenTemp, &mu)

	after, err := d.FileSize()
	if err != nil {
		t.Fatal(err)
	}
	endPrivate, _ := platform.ProcessPrivateBytes()
	runtime.ReadMemStats(&heap)

	mu.Lock()
	dbFiles := formatSeen(seenDB)
	tempFiles := formatSeen(seenTemp)
	peakP := peakPrivate
	peakH := peakHeap
	mu.Unlock()

	mode := "DSN temp_store=MEMORY"
	if pinTempToDisk {
		mode = "PRAGMA temp_store=FILE next to catalog"
	}
	t.Logf("VACUUM %s", mode)
	t.Logf("file before=%d after=%d freed=%d wall=%s", before, after, before-after, elapsed)
	t.Logf("private bytes baseline=%d peak=%d after=%d", basePrivate, peakP, endPrivate)
	t.Logf("heapAlloc baseline=%d peak=%d after=%d", baseHeap, peakH, heap.HeapAlloc)
	t.Logf("new files in catalog dir: %s", dbFiles)
	t.Logf("new files in os.TempDir matching sqlite/etilqs: %s", tempFiles)
}

func copySnapshot(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func listNames(t *testing.T, dir string) map[string]struct{} {
	t.Helper()
	out := map[string]struct{}{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		out[e.Name()] = struct{}{}
	}
	return out
}

func noteNew(dir string, baseline map[string]struct{}, seen map[string]int64, mu *sync.Mutex) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	sysTemp := os.TempDir()
	for _, e := range entries {
		if _, ok := baseline[e.Name()]; ok {
			continue
		}
		if filepath.Clean(dir) == filepath.Clean(sysTemp) && !looksLikeSQLiteTemp(e.Name()) {
			continue
		}
		info, err := e.Info()
		size := int64(0)
		if err == nil {
			size = info.Size()
		}
		mu.Lock()
		if prev, ok := seen[e.Name()]; !ok || size > prev {
			seen[e.Name()] = size
		}
		mu.Unlock()
	}
}

func looksLikeSQLiteTemp(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "etilqs") ||
		strings.Contains(lower, "sqlite") ||
		strings.Contains(lower, "vacuum")
}

func formatSeen(seen map[string]int64) string {
	if len(seen) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(seen))
	for name, size := range seen {
		parts = append(parts, fmt.Sprintf("%s=%d", name, size))
	}
	return strings.Join(parts, ", ")
}
