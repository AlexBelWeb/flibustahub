//go:build linux

package platform

import "golang.org/x/sys/unix"

func diskFree(path string) (uint64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, err
	}
	bsize := uint64(st.Bsize)
	if st.Frsize > 0 {
		bsize = uint64(st.Frsize)
	}
	return uint64(st.Bavail) * bsize, nil
}
