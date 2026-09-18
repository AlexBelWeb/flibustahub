//go:build unix

package platform

import (
	"os"
	"path/filepath"
	"syscall"
)

func acquireInstance(dataDir string) (func(), bool, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return func() {}, false, err
	}
	f, err := os.OpenFile(filepath.Join(dataDir, "instance.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return func() {}, false, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return func() {}, false, nil
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, true, nil
}
