//go:build windows

package platform

import "golang.org/x/sys/windows"

func osDownloadsDir() string {
	dir, err := windows.KnownFolderPath(windows.FOLDERID_Downloads, windows.KF_FLAG_DEFAULT)
	if err == nil && dir != "" {
		return dir
	}
	return homeDownloads()
}
