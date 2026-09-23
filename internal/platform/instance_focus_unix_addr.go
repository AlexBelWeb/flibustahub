//go:build unix && !linux

package platform

import (
	"os"
	"path/filepath"
)

func focusAddr(dataDir string) (addr string, file bool) {
	name := "flibustahub-" + instanceKey(dataDir) + ".sock"
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, name), true
	}
	return filepath.Join(os.TempDir(), name), true
}
