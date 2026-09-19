//go:build linux

package platform

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func identifyVolume(path string) (id, rel string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", ""
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		resolved = abs
	}
	mounts := mountTable()
	if uuid, mount := matchDisk("by-uuid", mounts, resolved); uuid != "" {
		return VolumeUUID + uuid, splitRel(resolved, mount)
	}
	if label, mount := matchDisk("by-label", mounts, resolved); label != "" {
		return VolumeLabel + label, splitRel(resolved, mount)
	}
	return "", ""
}

func findVolumePath(id, rel string) (string, bool, bool) {
	mounts := mountTable()
	switch {
	case strings.HasPrefix(id, VolumeUUID):
		name := strings.TrimPrefix(id, VolumeUUID)
		mount, ok := mountForDisk("by-uuid", name, mounts)
		if !ok {
			return "", false, false
		}
		return joinMount(mount, rel), true, false
	case strings.HasPrefix(id, VolumeLabel):
		name := strings.TrimPrefix(id, VolumeLabel)
		hits := mountsForLabel(name, mounts)
		if len(hits) == 0 {
			return "", false, false
		}
		if len(hits) > 1 {
			return "", false, true
		}
		return joinMount(hits[0], rel), true, false
	default:
		return "", false, false
	}
}

func joinMount(mount, rel string) string {
	if rel == "" {
		return mount
	}
	return filepath.Join(mount, rel)
}

type mnt struct {
	dev   string
	mount string
}

func mountTable() []mnt {
	f, err := os.Open("/proc/self/mounts")
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	var out []mnt
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		out = append(out, mnt{dev: fields[0], mount: fields[1]})
	}
	return out
}

func matchDisk(kind string, mounts []mnt, resolved string) (name, mount string) {
	dir := filepath.Join("/dev/disk", kind)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", ""
	}
	for _, e := range entries {
		dev, err := filepath.EvalSymlinks(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		m := mountForDev(dev, mounts)
		if m == "" {
			continue
		}
		rel, err := filepath.Rel(m, resolved)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		return e.Name(), m
	}
	return "", ""
}

func mountForDisk(kind, name string, mounts []mnt) (string, bool) {
	dev, err := filepath.EvalSymlinks(filepath.Join("/dev/disk", kind, name))
	if err != nil {
		return "", false
	}
	m := mountForDev(dev, mounts)
	return m, m != ""
}

func mountsForLabel(label string, mounts []mnt) []string {
	dir := filepath.Join("/dev/disk/by-label")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var hits []string
	seen := map[string]struct{}{}
	for _, e := range entries {
		if e.Name() != label {
			continue
		}
		dev, err := filepath.EvalSymlinks(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		m := mountForDev(dev, mounts)
		if m == "" {
			continue
		}
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		hits = append(hits, m)
	}
	return hits
}

func mountForDev(dev string, mounts []mnt) string {
	best := ""
	bestLen := -1
	for _, m := range mounts {
		src, err := filepath.EvalSymlinks(m.dev)
		if err != nil {
			src = m.dev
		}
		if src != dev && m.dev != dev {
			continue
		}
		if len(m.mount) > bestLen {
			best = m.mount
			bestLen = len(m.mount)
		}
	}
	return best
}
