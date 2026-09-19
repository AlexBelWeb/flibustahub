package downloads

import (
	"path/filepath"
	"strings"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/fb2"
)

func zipPath(root, name string) (string, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	name = strings.TrimSpace(name)
	if root == "" {
		return "", apperr.New(apperr.CodeLibraryOffline, nil)
	}
	if name == "" || !fb2.ValidEntryName(name) {
		return "", apperr.New(apperr.CodeArchiveMissing, map[string]string{"archive": name})
	}
	p := filepath.Join(root, filepath.FromSlash(name))
	rel, err := filepath.Rel(root, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", apperr.New(apperr.CodeArchiveMissing, map[string]string{"archive": name})
	}
	return p, nil
}
