//go:build !windows && !linux

package platform

import (
	"fmt"
	"runtime"
)

func diskFree(string) (uint64, error) {
	return 0, fmt.Errorf("disk free space is not available on %s", runtime.GOOS)
}
