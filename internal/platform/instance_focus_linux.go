//go:build linux

package platform

import (
	"os"
	"path/filepath"
)

// focusAddr places the socket where the path length does not depend on dataDir.
// A portable catalog on /media/<user>/<volume-label>/... exceeds the 108-byte
// unix socket limit. XDG_RUNTIME_DIR is short and mode 0700; without it the
// Linux abstract namespace keeps the same key and leaves no file behind.
func focusAddr(dataDir string) (addr string, file bool) {
	key := instanceKey(dataDir)
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "flibustahub-"+key+".sock"), true
	}
	return "@flibustahub-" + key, false
}
