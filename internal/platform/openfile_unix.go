//go:build !windows

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
)

func revealPathOS(abs string) error {
	return exec.Command("xdg-open", filepath.Dir(abs)).Start()
}

func openDirOS(abs string) error {
	return exec.Command("xdg-open", abs).Start()
}

// OpenFile opens path with the default associated application.
func OpenFile(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	return exec.Command("xdg-open", abs).Start()
}

// OpenFileWith launches appPath with filePath as its argument.
func OpenFileWith(appPath, filePath string) error {
	if _, err := os.Stat(appPath); err != nil {
		return err
	}
	return exec.Command(appPath, filePath).Start()
}
