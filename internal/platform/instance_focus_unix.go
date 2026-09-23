//go:build unix

package platform

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ListenFocus serves a unix socket in dataDir. The socket is removed before the
// caller drops the lock, and a new holder binds its own. A file left by a killed
// process is unlinked here: this function runs only while the lock is held.
func ListenFocus(dataDir string, onFocus func()) (func(), error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return func() {}, err
	}
	path := focusSocketPath(dataDir)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return func() {}, err
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return func() {}, err
	}
	_ = os.Chmod(path, 0o600)
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
			_ = os.Remove(path)
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
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("unix", focusSocketPath(dataDir))
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

func focusSocketPath(dataDir string) string {
	return filepath.Join(dataDir, "instance.focus.sock")
}
