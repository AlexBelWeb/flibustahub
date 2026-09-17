//go:build windows

package platform

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGetUserDefaultLocale = kernel32.NewProc("GetUserDefaultLocaleName")
)

func detectOSLocale() string {
	buf := make([]uint16, 85)
	r, _, _ := procGetUserDefaultLocale.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r == 0 {
		return "en"
	}
	return MatchLocale(syscall.UTF16ToString(buf))
}
