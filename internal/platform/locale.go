package platform

import (
	"os"
	"strings"
)

// MatchLocale maps a BCP-47 / POSIX locale to a supported UI code.
// Unknown values fall back to English.
func MatchLocale(tag string) string {
	norm := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(tag), "_", "-"))
	if norm == "" {
		return "en"
	}
	if strings.HasPrefix(norm, "ru") {
		return "ru"
	}
	return "en"
}

// DetectLocale reads the OS locale. An empty config value should call this.
func DetectLocale() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(key); v != "" {
			v = strings.SplitN(v, ".", 2)[0]
			return MatchLocale(v)
		}
	}
	return detectOSLocale()
}
