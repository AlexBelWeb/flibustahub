//go:build linux

package platform

import "golang.org/x/sys/unix"

func processPrivateBytes() (uint64, error) {
	var r unix.Rusage
	if err := unix.Getrusage(unix.RUSAGE_SELF, &r); err != nil {
		return 0, err
	}
	// Linux Maxrss is kilobytes; this is peak RSS, not a live private set.
	if r.Maxrss < 0 {
		return 0, nil
	}
	return uint64(r.Maxrss) * 1024, nil
}
