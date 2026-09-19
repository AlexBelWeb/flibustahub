package storage

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
)

func testCfg(t *testing.T, root string) *config.Store {
	t.Helper()
	store, err := config.Load(t.TempDir(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(f *config.File) { f.LibraryRoot = root }); err != nil {
		t.Fatal(err)
	}
	return store
}

func TestCheckEmptyRootIsNotConfigured(t *testing.T) {
	store, err := config.Load(t.TempDir(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	s := New(store, nil, slog.New(slog.DiscardHandler), nil)
	snap := s.Check(context.Background(), true)
	if snap.Configured || snap.Available || snap.Unreachable {
		t.Fatalf("%+v", snap)
	}
	if err := s.Probe(context.Background()); apperr.As(err).Code != apperr.CodeLibraryOffline {
		t.Fatalf("probe %v", err)
	}
}

func TestCheckAvailableAndMissing(t *testing.T) {
	root := t.TempDir()
	s := New(testCfg(t, root), nil, slog.New(slog.DiscardHandler), nil)
	snap := s.Check(context.Background(), true)
	if !snap.Available || !snap.Configured {
		t.Fatalf("%+v", snap)
	}
	s.cfg = testCfg(t, filepath.Join(root, "nope"))
	s.have = false
	snap = s.Check(context.Background(), true)
	if snap.Available || snap.Unreachable {
		t.Fatalf("missing %+v", snap)
	}
}

func TestCheckUnreachable(t *testing.T) {
	root := t.TempDir()
	s := New(testCfg(t, root), nil, slog.New(slog.DiscardHandler), nil)
	s.openRoot = func(string) error { return fs.ErrPermission }
	snap := s.Check(context.Background(), true)
	if !snap.Unreachable || snap.Available {
		t.Fatalf("%+v", snap)
	}
	if err := errorOf(snap); apperr.As(err).Code != apperr.CodeLibraryUnreachable {
		t.Fatalf("err %v", err)
	}
}

func TestCheckCachesUntilForce(t *testing.T) {
	root := t.TempDir()
	s := New(testCfg(t, root), nil, slog.New(slog.DiscardHandler), nil)
	s.ttl = time.Hour
	var stats atomic.Int32
	real := s.stat
	s.stat = func(path string) (os.FileInfo, error) {
		stats.Add(1)
		return real(path)
	}
	s.Check(context.Background(), true)
	s.Check(context.Background(), false)
	s.Check(context.Background(), false)
	if stats.Load() != 1 {
		t.Fatalf("cached checks still probed: %d", stats.Load())
	}
	s.Check(context.Background(), true)
	if stats.Load() != 2 {
		t.Fatalf("force did not probe: %d", stats.Load())
	}
}

func TestEmitOnFirstAndOnChange(t *testing.T) {
	root := t.TempDir()
	var n atomic.Int32
	s := New(testCfg(t, root), nil, slog.New(slog.DiscardHandler), func(Snapshot) { n.Add(1) })
	s.Check(context.Background(), true)
	s.Check(context.Background(), true)
	if n.Load() != 1 {
		t.Fatalf("first emit count %d", n.Load())
	}
	s.cfg = testCfg(t, filepath.Join(root, "gone"))
	s.have = false
	s.Check(context.Background(), true)
	if n.Load() != 2 {
		t.Fatalf("change emit count %d", n.Load())
	}
}
