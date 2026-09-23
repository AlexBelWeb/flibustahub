//go:build windows

package platform

import (
	"syscall"
	"unsafe"
)

var (
	procCreateMutexW = kernel32.NewProc("CreateMutexW")
	procCloseHandle  = kernel32.NewProc("CloseHandle")
)

const errorAlreadyExists syscall.Errno = 183

func acquireInstance(dataDir string) (func(), bool, error) {
	name, err := syscall.UTF16PtrFromString(instanceMutexName(dataDir))
	if err != nil {
		return func() {}, false, err
	}
	handle, _, callErr := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		return func() {}, false, callErr
	}
	if callErr == errorAlreadyExists {
		_, _, _ = procCloseHandle.Call(handle)
		return func() {}, false, nil
	}
	return func() {
		_, _, _ = procCloseHandle.Call(handle)
	}, true, nil
}

func instanceMutexName(dataDir string) string {
	return `Local\FlibustaHub-` + instanceKey(dataDir)
}
