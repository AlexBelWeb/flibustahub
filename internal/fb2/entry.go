package fb2

import (
	"archive/zip"
	"path"
	"strings"
)

// ValidEntryName rejects ZIP names that look like path traversal or absolute paths.
// A matching TOC entry is still required before Open; the name is never used as a disk path.
func ValidEntryName(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	n := strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(n, "/") || strings.HasPrefix(n, "//") {
		return false
	}
	if i := strings.IndexByte(n, ':'); i >= 0 && !strings.Contains(n[:i], "/") {
		return false
	}
	for _, seg := range strings.Split(n, "/") {
		if seg == ".." {
			return false
		}
	}
	return true
}

// FindEntry locates a book inside a ZIP: name.ext (case-insensitive), then name.fb2,
// then a linear scan by base name. Invalid TOC names are skipped, not opened.
func FindEntry(files []*zip.File, fileName, fileExt string) *zip.File {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return nil
	}
	ext := strings.TrimPrefix(strings.TrimSpace(fileExt), ".")
	wantExact := strings.ToLower(fileName + "." + ext)
	wantFB2 := strings.ToLower(fileName + ".fb2")

	if f := matchBase(files, wantExact); f != nil {
		return f
	}
	if !strings.EqualFold(ext, "fb2") {
		if f := matchBase(files, wantFB2); f != nil {
			return f
		}
	}
	wantBase := strings.ToLower(fileName)
	for _, f := range files {
		if f == nil || !ValidEntryName(f.Name) {
			continue
		}
		base := path.Base(strings.ReplaceAll(f.Name, "\\", "/"))
		stem := strings.TrimSuffix(base, path.Ext(base))
		if strings.EqualFold(stem, wantBase) {
			return f
		}
	}
	return nil
}

func matchBase(files []*zip.File, wantLower string) *zip.File {
	for _, f := range files {
		if f == nil || !ValidEntryName(f.Name) {
			continue
		}
		base := path.Base(strings.ReplaceAll(f.Name, "\\", "/"))
		if strings.ToLower(base) == wantLower {
			return f
		}
	}
	return nil
}
