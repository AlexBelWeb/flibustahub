package testdata

import (
	"os"
	"path/filepath"
	"time"
)

// WriteLibraryRoot creates a synthetic library: two archives on disk, one
// missing, a fresh .inpx in the root and a stale copy under update/.
func WriteLibraryRoot(dir string) error {
	if err := os.MkdirAll(filepath.Join(dir, "update"), 0o755); err != nil {
		return err
	}
	files := map[string][]byte{
		DumpName:                          CatalogINPX(),
		ArchiveLow:                        EmptyArchiveZip(),
		ArchiveHigh:                       EmptyArchiveZip(),
		filepath.Join("update", DumpName): StaleINPX(),
	}
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return err
		}
	}
	fresh := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	stale := time.Date(2025, 12, 2, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(dir, DumpName), fresh, fresh); err != nil {
		return err
	}
	return os.Chtimes(filepath.Join(dir, "update", DumpName), stale, stale)
}

// WriteCollectionOnlyDump writes an .inpx that has no version.info.
func WriteCollectionOnlyDump(dir, name string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), CollectionOnlyINPX(), 0o644)
}
