//go:build !windows && !unix

package platform

import "time"

func ListenFocus(string, func()) (func(), error) {
	return func() {}, nil
}

func SignalFocus(string, time.Duration) bool {
	return false
}
