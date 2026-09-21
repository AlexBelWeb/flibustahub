//go:build windows

package secrets

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

func nativeMachineID() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return "", err
	}
	defer func() { _ = k.Close() }()
	guid, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return "", err
	}
	if guid == "" {
		return "", fmt.Errorf("empty MachineGuid")
	}
	return guid, nil
}
