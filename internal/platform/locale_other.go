//go:build !windows

package platform

func detectOSLocale() string {
	return "en"
}
