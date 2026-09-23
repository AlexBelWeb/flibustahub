package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"sync"
	"time"
)

const (
	// FocusSignalTimeout is how long a second launch waits for the holder to
	// acknowledge a focus request before it treats the holder as gone or hung.
	FocusSignalTimeout = 500 * time.Millisecond
	// InstanceClaimWait is how long to wait for the lock after that acknowledgement
	// does not arrive. A live holder is not waited on: the signal returns first.
	InstanceClaimWait = 2 * time.Second
)

// StartKind is the startup decision for this process.
type StartKind int

const (
	// StartPrimary holds the dataDir lock and its focus channel.
	StartPrimary StartKind = iota
	// StartExit means a live holder acknowledged the focus signal. The process
	// must leave before it creates a window.
	StartExit
	// StartBlocked means the lock is still held and the holder did not answer.
	StartBlocked
)

// AcquireInstance claims the single-process lock for dataDir.
// release is always non-nil and safe to defer. primary is true only for the
// first live process; a second process must not open the catalog or bind HTTP.
func AcquireInstance(dataDir string) (release func(), primary bool, err error) {
	return acquireInstance(dataDir)
}

// DecideStart chooses the single primary process for dataDir.
// try is one non-blocking lock attempt. signal asks the holder to focus its
// window and reports whether it answered within the timeout. listen is started
// whenever this process becomes primary, including after claimWait, so the next
// launch can focus it. onFocus runs on the holder when a signal arrives.
// A lock error fails open: the process starts, and the error is returned for logging.
func DecideStart(
	try func() (release func(), primary bool, err error),
	signal func(timeout time.Duration) bool,
	listen func(onFocus func()) (stop func(), err error),
	onFocus func(),
	signalTimeout, claimWait time.Duration,
) (kind StartKind, release func(), err error) {
	unlock, primary, err := try()
	if err != nil {
		if unlock != nil {
			unlock()
		}
		return StartPrimary, func() {}, err
	}
	if primary {
		release, err = becomePrimary(unlock, listen, onFocus)
		return StartPrimary, release, err
	}
	if signal != nil && signal(signalTimeout) {
		return StartExit, func() {}, nil
	}
	deadline := time.Now().Add(claimWait)
	for {
		unlock, primary, err = try()
		if err != nil {
			if unlock != nil {
				unlock()
			}
			return StartPrimary, func() {}, err
		}
		if primary {
			release, err = becomePrimary(unlock, listen, onFocus)
			return StartPrimary, release, err
		}
		if !time.Now().Before(deadline) {
			return StartBlocked, func() {}, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// becomePrimary starts the focus channel and returns a release that stops the
// channel before it drops the lock. The next process must not observe a live
// channel belonging to a process that no longer holds the lock.
func becomePrimary(unlock func(), listen func(func()) (func(), error), onFocus func()) (func(), error) {
	if listen == nil {
		return unlock, nil
	}
	stop, err := listen(onFocus)
	if err != nil {
		return unlock, err
	}
	return composeRelease(stop, unlock), nil
}

func composeRelease(stop, unlock func()) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			if stop != nil {
				stop()
			}
			if unlock != nil {
				unlock()
			}
		})
	}
}

func instanceKey(dataDir string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(dataDir)))
	return hex.EncodeToString(sum[:8])
}

// ClaimInstance retries AcquireInstance until wait elapses. A process that is
// still exiting keeps the lock for a moment; a restart in that window should
// become primary instead of opening a window that can never load the catalog.
func ClaimInstance(dataDir string, wait time.Duration) (func(), bool, error) {
	deadline := time.Now().Add(wait)
	for {
		release, primary, err := AcquireInstance(dataDir)
		if err != nil || primary || !time.Now().Before(deadline) {
			return release, primary, err
		}
		time.Sleep(50 * time.Millisecond)
	}
}
