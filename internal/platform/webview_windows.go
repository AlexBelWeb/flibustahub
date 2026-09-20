//go:build windows

package platform

import (
	"golang.org/x/sys/windows/registry"
)

const webview2Client = `{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`

func webViewVersion() string {
	keys := []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE}
	paths := []string{
		`SOFTWARE\Microsoft\EdgeUpdate\Clients\` + webview2Client,
		`SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\` + webview2Client,
	}
	for _, root := range keys {
		for _, path := range paths {
			k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			pv, _, err := k.GetStringValue("pv")
			_ = k.Close()
			if err == nil && pv != "" && pv != "0.0.0.0" {
				return pv
			}
		}
	}
	return "unknown"
}
