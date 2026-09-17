package platform

import (
	"os/exec"
	"runtime"
)

// OpenDir opens path in the OS file manager. It does not wait for the manager to exit.
func OpenDir(path string) error {
	if runtime.GOOS == "windows" {
		return exec.Command("explorer.exe", path).Start()
	}
	return exec.Command("xdg-open", path).Start()
}
