package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func absExisting(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", os.ErrNotExist
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err != nil {
		return "", err
	}
	return abs, nil
}

func absExistingDir(path string) (string, error) {
	abs, err := absExisting(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", abs)
	}
	return abs, nil
}

var (
	revealPath = revealPathOS
	openDir    = openDirOS
)

// ShowInFolder opens the OS file manager with path selected.
// The target is resolved to an absolute path and must exist before the
// manager is launched: explorer.exe otherwise opens Documents and still
// reports success.
func ShowInFolder(path string) error {
	abs, err := absExisting(path)
	if err != nil {
		return err
	}
	return revealPath(abs)
}

// OpenDir opens path in the OS file manager. It does not wait for the manager to exit.
func OpenDir(path string) error {
	abs, err := absExistingDir(path)
	if err != nil {
		return err
	}
	return openDir(abs)
}
