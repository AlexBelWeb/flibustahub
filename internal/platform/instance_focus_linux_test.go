//go:build linux

package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFocusSignalLongDataDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), strings.Repeat("portable-library", 8))
	if len(dir) <= 108 {
		t.Fatalf("dataDir length %d, want more than 108", len(dir))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("runtime dir", func(t *testing.T) {
		t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
		addr, file := focusAddr(dir)
		if !file {
			t.Fatal("expected a filesystem socket")
		}
		if len(addr) > 108 {
			t.Fatalf("socket path is %d bytes", len(addr))
		}
		if err := os.WriteFile(addr, []byte("stale"), 0o600); err != nil {
			t.Fatal(err)
		}
		focusRoundtrip(t, dir)
		if _, err := os.Stat(addr); !os.IsNotExist(err) {
			t.Fatal("stopped holder should remove the socket file")
		}
	})

	t.Run("abstract", func(t *testing.T) {
		// Setenv records the original value for cleanup; Unsetenv is the
		// case GitHub runners actually have.
		t.Setenv("XDG_RUNTIME_DIR", "restore-after-test")
		if err := os.Unsetenv("XDG_RUNTIME_DIR"); err != nil {
			t.Fatal(err)
		}
		if _, ok := os.LookupEnv("XDG_RUNTIME_DIR"); ok {
			t.Fatal("XDG_RUNTIME_DIR is still set")
		}
		addr, file := focusAddr(dir)
		if file || len(addr) == 0 || addr[0] != '@' {
			t.Fatalf("addr %q file %v", addr, file)
		}
		focusRoundtrip(t, dir)
	})
}

func focusRoundtrip(t *testing.T, dir string) {
	t.Helper()
	got := make(chan struct{}, 1)
	stop, err := ListenFocus(dir, func() {
		select {
		case got <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !SignalFocus(dir, time.Second) {
		stop()
		t.Fatal("holder should acknowledge")
	}
	select {
	case <-got:
	case <-time.After(time.Second):
		stop()
		t.Fatal("focus callback was not called")
	}
	stop()
	if SignalFocus(dir, 200*time.Millisecond) {
		t.Fatal("stopped holder must not acknowledge")
	}
}
