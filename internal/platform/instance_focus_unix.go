//go:build unix

package platform

import (
	"context"
	"net"
	"os"
	"sync"
	"time"
)

// ListenFocus serves the focus socket for dataDir. A filesystem socket left by
// a killed holder is unlinked here: this runs only while the lock is held.
// An abstract-namespace socket has no file to remove.
func ListenFocus(dataDir string, onFocus func()) (func(), error) {
	addr, file := focusAddr(dataDir)
	if file {
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			return func() {}, err
		}
	}
	ln, err := net.Listen("unix", addr)
	if err != nil {
		return func() {}, err
	}
	if file {
		_ = os.Chmod(addr, 0o600)
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveFocusConn(ctx, conn, onFocus)
		}
	}()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			cancel()
			_ = ln.Close()
			wg.Wait()
			if file {
				_ = os.Remove(addr)
			}
		})
	}
	return stop, nil
}

func serveFocusConn(ctx context.Context, conn net.Conn, onFocus func()) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(FocusSignalTimeout))
	buf := make([]byte, 1)
	if _, err := conn.Read(buf); err != nil {
		return
	}
	if _, err := conn.Write([]byte{1}); err != nil {
		return
	}
	if onFocus != nil && ctx.Err() == nil {
		onFocus()
	}
}

// SignalFocus dials the holder's socket and waits up to timeout for the ack byte.
func SignalFocus(dataDir string, timeout time.Duration) bool {
	if timeout <= 0 {
		timeout = FocusSignalTimeout
	}
	addr, _ := focusAddr(dataDir)
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("unix", addr)
	if err != nil {
		return false
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if _, err := conn.Write([]byte{1}); err != nil {
		return false
	}
	buf := make([]byte, 1)
	if _, err := conn.Read(buf); err != nil {
		return false
	}
	return buf[0] == 1
}
