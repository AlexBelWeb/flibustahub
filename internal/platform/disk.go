package platform

import (
	"os"
	"path/filepath"
)

// DiskFree returns free bytes on the volume that contains path.
func DiskFree(path string) (uint64, error) {
	return diskFree(volumePath(path))
}

func volumePath(path string) string {
	if path == "" {
		return ""
	}
	st, err := os.Stat(path)
	if err == nil && st.IsDir() {
		return path
	}
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return path
	}
	return dir
}
