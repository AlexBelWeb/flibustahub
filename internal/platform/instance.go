package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
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
) (kind StartKind, hold InstanceHold, err error) {
	unlock, primary, err := try()
	if err != nil {
		if unlock != nil {
			unlock()
		}
		return StartPrimary, InstanceHold{}, err
	}
	if primary {
		hold, err = becomePrimary(unlock, listen, onFocus)
		return StartPrimary, hold, err
	}
	if signal != nil && signal(signalTimeout) {
		return StartExit, InstanceHold{}, nil
	}
	deadline := time.Now().Add(claimWait)
	for {
		unlock, primary, err = try()
		if err != nil {
			if unlock != nil {
				unlock()
			}
			return StartPrimary, InstanceHold{}, err
		}
		if primary {
			hold, err = becomePrimary(unlock, listen, onFocus)
			return StartPrimary, hold, err
		}
		if !time.Now().Before(deadline) {
			return StartBlocked, InstanceHold{}, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// InstanceHold is the primary process's lock and focus channel.
// StopFocus and Unlock answer different questions and are released at different
// times: the channel goes down when the process can no longer show a window,
// the lock stays until the catalog file is closed. Release does both, channel
// first, for a process-exit backstop.
type InstanceHold struct {
	StopFocus func()
	Unlock    func()
}

// Release stops the focus channel and then drops the lock. Both parts are
// safe to call on their own before this.
func (h InstanceHold) Release() {
	if h.StopFocus != nil {
		h.StopFocus()
	}
	if h.Unlock != nil {
		h.Unlock()
	}
}

// FocusListenError means this process holds the lock but could not start the
// focus channel. The process still starts; the next launch cannot focus it.
type FocusListenError struct{ Err error }

func (e *FocusListenError) Error() string {
	if e == nil || e.Err == nil {
		return "focus channel did not start"
	}
	return e.Err.Error()
}

func (e *FocusListenError) Unwrap() error { return e.Err }

// becomePrimary starts the focus channel. A listen failure still returns the
// lock so the caller remains the primary process.
func becomePrimary(unlock func(), listen func(func()) (func(), error), onFocus func()) (InstanceHold, error) {
	hold := InstanceHold{Unlock: unlock}
	if listen == nil {
		return hold, nil
	}
	stop, err := listen(onFocus)
	if err != nil {
		return hold, &FocusListenError{Err: err}
	}
	hold.StopFocus = stop
	return hold, nil
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
