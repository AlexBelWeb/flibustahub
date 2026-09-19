package covers

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/alexbelweb/flibustahub/internal/fb2"
)

var coverKinds = []fb2.ImageKind{fb2.ImageJPEG, fb2.ImagePNG, fb2.ImageGIF, fb2.ImageWebP}

// Hit is a cover-cache lookup result. Found with None means a verified absence.
type Hit struct {
	Found   bool   `json:"found"`
	None    bool   `json:"none,omitempty"`
	Missing bool   `json:"missing,omitempty"`
	Path    string `json:"-"`
	MIME    string `json:"-"`
}

func coverBase(dir string, workID int64) string {
	return filepath.Join(dir, strconv.FormatInt(workID, 10))
}

// Lookup reports a cached image or a .none marker. Missing files are a miss.
func Lookup(dir string, workID int64) Hit {
	if dir == "" || workID <= 0 {
		return Hit{}
	}
	base := coverBase(dir, workID)
	for _, k := range coverKinds {
		p := base + k.Ext()
		if fileExists(p) {
			return Hit{Found: true, Path: p, MIME: k.MIME()}
		}
	}
	none := base + ".none"
	if fileExists(none) {
		return Hit{Found: true, None: true, Path: none}
	}
	return Hit{}
}

func WriteCover(dir string, workID int64, data []byte, kind fb2.ImageKind) error {
	if kind == fb2.ImageNone || len(data) == 0 {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	base := coverBase(dir, workID)
	removeCoverFiles(base)
	return os.WriteFile(base+kind.Ext(), data, 0o644)
}

func WriteNone(dir string, workID int64) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	base := coverBase(dir, workID)
	removeCoverFiles(base)
	f, err := os.OpenFile(base+".none", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}

func ClearDir(dir string) error {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var first error
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// ClearNoneMarkers removes {workId}.none files after a successful import.
// Image files are left in place.
func ClearNoneMarkers(dir string) error {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var first error
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".none" {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func removeCoverFiles(base string) {
	for _, k := range coverKinds {
		_ = os.Remove(base + k.Ext())
	}
	_ = os.Remove(base + ".none")
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
