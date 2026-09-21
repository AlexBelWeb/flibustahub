//go:build !windows

package platform

func webViewVersion() string {
	return "unknown"
}
