package platform

import (
	"testing"
	"time"
)

func TestDecideStartExitsWhenHolderAcks(t *testing.T) {
	listened := 0
	kind, hold, err := DecideStart(
		func() (func(), bool, error) { return func() {}, false, nil },
		func(time.Duration) bool { return true },
		func(func()) (func(), error) {
			listened++
			return func() {}, nil
		},
		func() {},
		time.Second,
		time.Second,
	)
	if err != nil {
		t.Fatal(err)
	}
	if kind != StartExit {
		t.Fatalf("kind = %v, want exit without a window", kind)
	}
	if listened != 0 {
		t.Fatal("an acknowledging holder must not start a focus channel here")
	}
	hold.Release()
}

func TestDecideStartBecomesPrimaryWithChannelWhenLockFrees(t *testing.T) {
	calls := 0
	var order []string
	kind, hold, err := DecideStart(
		func() (func(), bool, error) {
			calls++
			if calls < 3 {
				return func() {}, false, nil
			}
			return func() { order = append(order, "lock") }, true, nil
		},
		func(time.Duration) bool { return false },
		func(func()) (func(), error) {
			return func() { order = append(order, "channel") }, nil
		},
		func() {},
		time.Millisecond,
		time.Second,
	)
	if err != nil {
		t.Fatal(err)
	}
	if kind != StartPrimary {
		t.Fatalf("kind = %v, want primary", kind)
	}
	hold.Release()
	if len(order) != 2 || order[0] != "channel" || order[1] != "lock" {
		t.Fatalf("release order = %v, want channel then lock", order)
	}
}

func TestDecideStartBlocksWhenHolderStaysSilent(t *testing.T) {
	listened := 0
	kind, _, err := DecideStart(
		func() (func(), bool, error) { return func() {}, false, nil },
		func(time.Duration) bool { return false },
		func(func()) (func(), error) {
			listened++
			return func() {}, nil
		},
		func() {},
		time.Millisecond,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if kind != StartBlocked {
		t.Fatalf("kind = %v, want blocked", kind)
	}
	if listened != 0 {
		t.Fatal("a process that never takes the lock must not listen")
	}
}
