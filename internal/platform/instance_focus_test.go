//go:build windows || unix

package platform

import (
	"testing"
	"time"
)

func TestFocusSignalRoundtrip(t *testing.T) {
	dir := t.TempDir()
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
		t.Fatal("holder should acknowledge the focus signal")
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
	stop, err = ListenFocus(dir, func() {
		select {
		case got <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	if !SignalFocus(dir, time.Second) {
		t.Fatal("a new holder in the same process must be focusable")
	}
}
