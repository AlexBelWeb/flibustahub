//go:build windows

package platform

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	wmFocus       = 0x8000 + 1 // WM_APP + 1, delivered only to our class.
	wmQuit        = 0x0012
	hwndMessage   = ^uintptr(2) // HWND_MESSAGE == (HWND)-3
	smtoAbortHung = 0x0002
	smtoBlock     = 0x0001
	classExists   = syscall.Errno(1410) // ERROR_CLASS_ALREADY_EXISTS
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procRegisterClassExW    = user32.NewProc("RegisterClassExW")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDefWindowProcW      = user32.NewProc("DefWindowProcW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procPeekMessageW        = user32.NewProc("PeekMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPostThreadMessageW  = user32.NewProc("PostThreadMessageW")
	procFindWindowW         = user32.NewProc("FindWindowW")
	procSendMessageTimeoutW = user32.NewProc("SendMessageTimeoutW")
	procGetModuleHandleW    = kernel32.NewProc("GetModuleHandleW")
	procGetCurrentThreadId  = kernel32.NewProc("GetCurrentThreadId")
	focusWndProc            = syscall.NewCallback(focusWindowProc)
	focusHandlerMu          sync.Mutex
	focusHandlers           = map[uintptr]chan struct{}{}
)

// wndClassExW matches WNDCLASSEXW on amd64.
type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     uintptr
	hIcon         uintptr
	hCursor       uintptr
	hbrBackground uintptr
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       uintptr
}

// winMSG matches MSG on amd64. Go inserts the padding after message.
type winMSG struct {
	hwnd     uintptr
	message  uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	ptX      int32
	ptY      int32
	lPrivate uint32
}

func focusWindowProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	if msg == wmFocus {
		focusHandlerMu.Lock()
		ch := focusHandlers[hwnd]
		focusHandlerMu.Unlock()
		if ch != nil {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
		return 1
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return ret
}

// ListenFocus creates a message-only window keyed by dataDir and runs its
// message loop. The holder is created whenever this process owns the lock,
// including after a wait, which is the step the toolkit lock skips.
// stop destroys that window before the caller drops the lock.
func ListenFocus(dataDir string, onFocus func()) (func(), error) {
	className, err := syscall.UTF16PtrFromString(focusClassName(dataDir))
	if err != nil {
		return func() {}, err
	}
	windowName, err := syscall.UTF16PtrFromString(focusWindowName(dataDir))
	if err != nil {
		return func() {}, err
	}
	signals := make(chan struct{}, 1)
	go func() {
		for range signals {
			if onFocus != nil {
				onFocus()
			}
		}
	}()

	ready := make(chan error, 1)
	done := make(chan struct{})
	var (
		tid  uint32
		hwnd uintptr
	)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		id, _, _ := procGetCurrentThreadId.Call()
		tid = uint32(id)
		// PeekMessage creates this thread's queue before anyone posts WM_QUIT.
		var prime winMSG
		_, _, _ = procPeekMessageW.Call(uintptr(unsafe.Pointer(&prime)), 0, 0, 0, 0)
		handle, createErr := createFocusWindow(className, windowName)
		if createErr != nil {
			close(signals)
			ready <- createErr
			close(done)
			return
		}
		hwnd = handle
		focusHandlerMu.Lock()
		focusHandlers[hwnd] = signals
		focusHandlerMu.Unlock()
		ready <- nil
		var msg winMSG
		for {
			ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(ret) <= 0 {
				break
			}
			_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
		focusHandlerMu.Lock()
		delete(focusHandlers, hwnd)
		focusHandlerMu.Unlock()
		_, _, _ = procDestroyWindow.Call(hwnd)
		close(signals)
		close(done)
	}()
	if err := <-ready; err != nil {
		return func() {}, err
	}
	var once sync.Once
	stop := func() {
		once.Do(func() {
			_, _, _ = procPostThreadMessageW.Call(uintptr(tid), wmQuit, 0, 0)
			select {
			case <-done:
			case <-time.After(500 * time.Millisecond):
			}
		})
	}
	return stop, nil
}

func createFocusWindow(className, windowName *uint16) (uintptr, error) {
	inst, _, _ := procGetModuleHandleW.Call(0)
	class := wndClassExW{
		cbSize:        uint32(unsafe.Sizeof(wndClassExW{})),
		lpfnWndProc:   focusWndProc,
		hInstance:     inst,
		lpszClassName: className,
	}
	atom, _, callErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&class)))
	if atom == 0 && callErr != classExists {
		if callErr == syscall.Errno(0) {
			return 0, fmt.Errorf("register focus window class")
		}
		return 0, callErr
	}
	hwnd, _, callErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		0,
		0, 0, 0, 0,
		hwndMessage,
		0,
		inst,
		0,
	)
	if hwnd == 0 {
		if callErr == syscall.Errno(0) {
			return 0, fmt.Errorf("create focus window")
		}
		return 0, callErr
	}
	return hwnd, nil
}

// SignalFocus asks the holder to focus its window. No window, or no answer
// before timeout, is not an acknowledgement: the caller may then wait for the lock.
func SignalFocus(dataDir string, timeout time.Duration) bool {
	className, err := syscall.UTF16PtrFromString(focusClassName(dataDir))
	if err != nil {
		return false
	}
	windowName, err := syscall.UTF16PtrFromString(focusWindowName(dataDir))
	if err != nil {
		return false
	}
	hwnd, _, _ := procFindWindowW.Call(
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
	)
	if hwnd == 0 {
		return false
	}
	ms := timeout.Milliseconds()
	if ms < 1 {
		ms = 1
	}
	var result uintptr
	ret, _, _ := procSendMessageTimeoutW.Call(
		hwnd,
		wmFocus,
		0,
		0,
		smtoAbortHung|smtoBlock,
		uintptr(ms),
		uintptr(unsafe.Pointer(&result)),
	)
	return ret != 0 && result == 1
}

func focusClassName(dataDir string) string {
	return "FlibustaHubFocus-" + instanceKey(dataDir)
}

func focusWindowName(dataDir string) string {
	return "FlibustaHubFocusWnd-" + instanceKey(dataDir)
}
