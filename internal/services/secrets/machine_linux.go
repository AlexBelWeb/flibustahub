//go:build linux

package secrets

import (
	"fmt"
	"os"
	"strings"
)

func nativeMachineID() (string, error) {
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		id := strings.TrimSpace(string(raw))
		if id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("machine-id not found")
}
