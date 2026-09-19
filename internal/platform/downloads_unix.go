//go:build !windows

package platform

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func osDownloadsDir() string {
	if dir := strings.TrimSpace(os.Getenv("XDG_DOWNLOAD_DIR")); dir != "" {
		return expandHome(dir)
	}
	if dir := xdgUserDir("XDG_DOWNLOAD_DIR"); dir != "" {
		return dir
	}
	return homeDownloads()
}

func xdgUserDir(key string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		cfg = filepath.Join(home, ".config")
	}
	f, err := os.Open(filepath.Join(cfg, "user-dirs.dirs"))
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	prefix := key + "="
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "#") || !strings.HasPrefix(line, prefix) {
			continue
		}
		val := strings.Trim(strings.TrimPrefix(line, prefix), "\"")
		return expandHome(val)
	}
	return ""
}

func expandHome(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "$HOME") {
		home, _ := os.UserHomeDir()
		return home + strings.TrimPrefix(p, "$HOME")
	}
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}
