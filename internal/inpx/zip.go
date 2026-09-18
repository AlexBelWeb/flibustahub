package inpx

import (
	"archive/zip"
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const utf8BOM = "\uFEFF"

// DumpMeta is version, encodings and archive names discovered while reading an INPX.
type DumpMeta struct {
	Version          string
	Archives         []string
	SkippedMalformed int
	SkippedNoLibID   int
	RecordsSeen      int
	Encodings        EncodingStats
	InpBytesTotal    int64
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

type recordHandler func(rec Record, bytesDone, bytesTotal int64) error

// EncodingStats counts detected encodings of dump members.
type EncodingStats struct {
	UTF8           int    `json:"utf-8"`
	CP1251         int    `json:"cp1251"`
	VersionInfo    string `json:"version_info,omitempty"`
	CollectionInfo string `json:"collection_info,omitempty"`
}

// WalkRecords opens an .inpx zip, reads version.info / collection.info, and
// yields records from .inp members in filename order.
func WalkRecords(r io.ReaderAt, size int64, fn func(Record) error) (DumpMeta, error) {
	return WalkRecordsProgress(r, size, func(rec Record, _, _ int64) error {
		return fn(rec)
	})
}

// WalkRecordsProgress is WalkRecords plus a running uncompressed-byte cursor
// against the ZIP TOC total of all .inp members.
func WalkRecordsProgress(r io.ReaderAt, size int64, fn func(Record, int64, int64) error) (DumpMeta, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return DumpMeta{}, err
	}
	meta, inps := zipIndex(zr)
	var started int64
	for _, f := range inps {
		if err := walkInp(f, fn, &meta, &started); err != nil {
			return meta, err
		}
	}
	return meta, nil
}

// PeekMeta reads version and archive names without parsing records.
func PeekMeta(r io.ReaderAt, size int64) (DumpMeta, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return DumpMeta{}, err
	}
	meta, _ := zipIndex(zr)
	return meta, nil
}

func zipIndex(zr *zip.Reader) (DumpMeta, []*zip.File) {
	ver, enc := versionFromZip(zr)
	meta := DumpMeta{Version: ver, Encodings: enc}
	var inps []*zip.File
	seen := map[string]struct{}{}
	for _, f := range zr.File {
		base := filepath.Base(f.Name)
		if !strings.HasSuffix(strings.ToLower(base), ".inp") {
			continue
		}
		inps = append(inps, f)
		meta.InpBytesTotal += int64(f.UncompressedSize64)
		arch := ArchiveNameFromInp(base)
		if _, ok := seen[arch]; !ok {
			seen[arch] = struct{}{}
			meta.Archives = append(meta.Archives, arch)
		}
	}
	sort.Slice(inps, func(i, j int) bool {
		return strings.ToLower(filepath.Base(inps[i].Name)) < strings.ToLower(filepath.Base(inps[j].Name))
	})
	sort.Strings(meta.Archives)
	return meta, inps
}

func walkInp(f *zip.File, fn recordHandler, meta *DumpMeta, started *int64) error {
	uncompressed := int64(f.UncompressedSize64)
	rc, err := f.Open()
	if err != nil {
		return err
	}
	body, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		return err
	}
	decoded, enc := DecodeFile(body)
	switch enc {
	case EncodingCP1251:
		meta.Encodings.CP1251++
	default:
		meta.Encodings.UTF8++
	}
	base := filepath.Base(f.Name)
	archive := ArchiveNameFromInp(base)
	decodedLen := int64(len(decoded))
	cr := &countingReader{r: bytes.NewReader(decoded)}
	sc := bufio.NewScanner(cr)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		rec, skip := ParseLine(sc.Bytes(), archive)
		switch skip {
		case SkipMalformed:
			meta.SkippedMalformed++
			continue
		case SkipNoLibID:
			meta.SkippedNoLibID++
			continue
		case SkipEmpty:
			continue
		}
		meta.RecordsSeen++
		done := *started
		if decodedLen > 0 {
			done += uncompressed * cr.n / decodedLen
		} else {
			done += uncompressed
		}
		if cap := *started + uncompressed; done > cap {
			done = cap
		}
		if err := fn(rec, done, meta.InpBytesTotal); err != nil {
			return err
		}
	}
	*started += uncompressed
	return sc.Err()
}

func versionFromZip(zr *zip.Reader) (version string, enc EncodingStats) {
	if raw := readZipFile(zr, "version.info"); len(raw) > 0 {
		decoded, e := DecodeFile(raw)
		enc.VersionInfo = string(e)
		if line := firstLine(decoded); line != "" {
			version = line
		}
	}
	if raw := readZipFile(zr, "collection.info"); len(raw) > 0 {
		decoded, e := DecodeFile(raw)
		enc.CollectionInfo = string(e)
		if version == "" {
			version = secondLine(decoded)
		}
	}
	return version, enc
}

func readZipFile(zr *zip.Reader, want string) []byte {
	want = strings.ToLower(want)
	for _, f := range zr.File {
		base := strings.ToLower(filepath.Base(f.Name))
		if base != want {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil
		}
		return b
	}
	return nil
}

func firstLine(b []byte) string {
	b = stripBOM(b)
	line, _, _ := bytes.Cut(b, []byte{'\n'})
	return strings.TrimSpace(strings.TrimSuffix(string(line), "\r"))
}

func secondLine(b []byte) string {
	b = stripBOM(b)
	_, rest, found := bytes.Cut(b, []byte{'\n'})
	if !found {
		return ""
	}
	line, _, _ := bytes.Cut(rest, []byte{'\n'})
	return strings.TrimSpace(strings.TrimSuffix(string(line), "\r"))
}

func stripBOM(b []byte) []byte {
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
	if r, size := utf8.DecodeRune(b); r == '\uFEFF' {
		return b[size:]
	}
	s := string(b)
	return []byte(strings.TrimPrefix(s, utf8BOM))
}

// OpenPath walks an .inpx file on disk.
func OpenPath(path string, fn func(Record) error) (DumpMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return DumpMeta{}, err
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return DumpMeta{}, err
	}
	return WalkRecords(f, st.Size(), fn)
}

// FindINPX returns explicitPath, or the newest .inpx in the library root (non-recursive).
func FindINPX(libraryRoot, explicitPath string) (string, error) {
	if strings.TrimSpace(explicitPath) != "" {
		return explicitPath, nil
	}
	if strings.TrimSpace(libraryRoot) == "" {
		return "", os.ErrNotExist
	}
	entries, err := os.ReadDir(libraryRoot)
	if err != nil {
		return "", err
	}
	var best string
	var bestMod int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.EqualFold(filepath.Ext(name), ".inpx") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		mt := info.ModTime().UnixNano()
		if best == "" || mt > bestMod || (mt == bestMod && name < best) {
			best = name
			bestMod = mt
		}
	}
	if best == "" {
		return "", os.ErrNotExist
	}
	return filepath.Join(libraryRoot, best), nil
}

// DumpFile is an .inpx found in the library root. Archives are never opened.
type DumpFile struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

// InspectLibraryRoot lists .inpx files and counts .zip names in the folder itself.
// A missing or unreadable folder is an error, not an empty listing.
func InspectLibraryRoot(root string) (zipCount int, dumps []DumpFile, err error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return 0, nil, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, nil, err
	}
	dumps = make([]DumpFile, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		switch ext {
		case ".zip":
			zipCount++
		case ".inpx":
			dumps = append(dumps, DumpFile{
				Path: filepath.Join(root, e.Name()),
				Name: e.Name(),
			})
		}
	}
	return zipCount, dumps, nil
}

// MissingArchives reports zip names listed in the dump that are not in libraryRoot.
func MissingArchives(libraryRoot string, archives []string) []string {
	var missing []string
	for _, a := range archives {
		if _, err := os.Stat(filepath.Join(libraryRoot, a)); err != nil {
			missing = append(missing, a)
		}
	}
	return missing
}
