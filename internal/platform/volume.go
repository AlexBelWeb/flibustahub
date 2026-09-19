package platform

import (
	"path/filepath"
	"strings"
)

const (
	VolumeGUID  = "guid:"
	VolumeUUID  = "uuid:"
	VolumeLabel = "label:"
)

// IdentifyVolume returns a prefixed volume id and the path relative to the volume root.
func IdentifyVolume(path string) (id, rel string) {
	return identifyVolume(path)
}

// FindVolumePath locates a volume by stored id and rebuilds path using rel.
// ok is false when nothing matched. ambiguous is true when several volumes
// share the same label — the caller must not pick one.
func FindVolumePath(id, rel string) (path string, ok, ambiguous bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", false, false
	}
	return findVolumePath(id, rel)
}

func splitRel(path, root string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	if rel == "." {
		return ""
	}
	return rel
}
