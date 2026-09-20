//go:build !windows && !linux

package platform

import (
	"fmt"
	"runtime"
)

func processPrivateBytes() (uint64, error) {
	return 0, fmt.Errorf("process private bytes are not available on %s", runtime.GOOS)
}
