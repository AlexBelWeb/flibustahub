//go:build !windows && !linux

package secrets

import (
	"fmt"
	"runtime"
)

func nativeMachineID() (string, error) {
	return "", fmt.Errorf("machine-id not available on %s", runtime.GOOS)
}
