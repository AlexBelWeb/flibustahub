package platform

import (
	"os"
	"path/filepath"
)

// DownloadsDir returns the OS downloads folder when it exists.
// An empty result means the first download must ask the user.
func DownloadsDir() string {
	dir := osDownloadsDir()
	if dir == "" {
		return ""
	}
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return ""
	}
	return dir
}

func homeDownloads() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, "Downloads")
}
