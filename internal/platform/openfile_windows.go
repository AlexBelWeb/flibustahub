//go:build windows

package platform

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

// OpenFile opens path with the default associated application.
func OpenFile(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	verb, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := syscall.UTF16PtrFromString(abs)
	if err != nil {
		return err
	}
	err = windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL)
	if err != nil {
		if errors.Is(err, windows.ERROR_NO_ASSOCIATION) || isShellNoAssoc(err) {
			return ErrNoAssociation
		}
		return err
	}
	return nil
}

func isShellNoAssoc(err error) bool {
	var errno syscall.Errno
	if errors.As(err, &errno) && (errno == 31 || errno == windows.ERROR_NO_ASSOCIATION) {
		return true
	}
	return false
}

// OpenFileWith launches appPath with filePath as its argument.
func OpenFileWith(appPath, filePath string) error {
	if _, err := os.Stat(appPath); err != nil {
		return err
	}
	return exec.Command(appPath, filePath).Start()
}
