//go:build windows

package platform

import (
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func identifyVolume(path string) (id, rel string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", ""
	}
	vol := filepath.VolumeName(abs)
	if vol == "" {
		return "", ""
	}
	root := vol + `\`
	rel = splitRel(abs, root)
	if guid := volumeGUID(root); guid != "" {
		return VolumeGUID + guid, rel
	}
	if label := volumeLabel(root); label != "" {
		return VolumeLabel + label, rel
	}
	return "", ""
}

func findVolumePath(id, rel string) (string, bool, bool) {
	roots := mountedRoots()
	switch {
	case strings.HasPrefix(id, VolumeGUID):
		want := strings.TrimPrefix(id, VolumeGUID)
		for _, root := range roots {
			if volumeGUID(root) == want {
				return joinVol(root, rel), true, false
			}
		}
		return "", false, false
	case strings.HasPrefix(id, VolumeLabel):
		want := strings.TrimPrefix(id, VolumeLabel)
		var hits []string
		for _, root := range roots {
			if volumeLabel(root) == want {
				hits = append(hits, root)
			}
		}
		if len(hits) == 0 {
			return "", false, false
		}
		if len(hits) > 1 {
			return "", false, true
		}
		return joinVol(hits[0], rel), true, false
	default:
		return "", false, false
	}
}

func joinVol(root, rel string) string {
	if rel == "" {
		return root
	}
	return filepath.Join(root, rel)
}

func volumeGUID(root string) string {
	if !strings.HasSuffix(root, `\`) {
		root += `\`
	}
	buf := make([]uint16, 64)
	p, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return ""
	}
	if err := windows.GetVolumeNameForVolumeMountPoint(p, &buf[0], uint32(len(buf))); err != nil {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func volumeLabel(root string) string {
	if !strings.HasSuffix(root, `\`) {
		root += `\`
	}
	p, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return ""
	}
	label := make([]uint16, 261)
	if err := windows.GetVolumeInformation(p, &label[0], uint32(len(label)), nil, nil, nil, nil, 0); err != nil {
		return ""
	}
	return strings.TrimSpace(syscall.UTF16ToString(label))
}

func mountedRoots() []string {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}
	var out []string
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		out = append(out, root)
	}
	return out
}
