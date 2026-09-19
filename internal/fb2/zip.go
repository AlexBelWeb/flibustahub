package fb2

import (
	"archive/zip"
	"errors"
	"io"
)

// ErrMissingEntry means the named book was not found among valid ZIP entries.
var ErrMissingEntry = errors.New("fb2 entry not in archive")

// ReadEntry streams one ZIP member into memory. The archive is not written to disk.
func ReadEntry(f *zip.File) ([]byte, error) {
	if f == nil {
		return nil, ErrMissingEntry
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	body, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// ParseZip finds the book entry and parses cover plus annotation from it.
func ParseZip(zr *zip.Reader, fileName, fileExt string) (Book, error) {
	if zr == nil {
		return Book{}, ErrMissingEntry
	}
	entry := FindEntry(zr.File, fileName, fileExt)
	if entry == nil {
		return Book{}, ErrMissingEntry
	}
	raw, err := ReadEntry(entry)
	if err != nil {
		return Book{}, err
	}
	return Parse(raw)
}
