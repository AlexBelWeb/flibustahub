package db

import (
	"fmt"
	"net/url"
	"path/filepath"
)

const (
	pragmaBusyTimeout = 30000
	pragmaCacheSize   = -64000
	pragmaMmapSize    = 268435456
)

func fileDSN(path string) string {
	slashed := filepath.ToSlash(path)
	q := url.Values{}
	q.Add("_pragma", fmt.Sprintf("busy_timeout=%d", pragmaBusyTimeout))
	q.Add("_pragma", "foreign_keys=ON")
	q.Add("_pragma", "temp_store=MEMORY")
	q.Add("_pragma", fmt.Sprintf("cache_size=%d", pragmaCacheSize))
	q.Add("_pragma", fmt.Sprintf("mmap_size=%d", pragmaMmapSize))
	q.Add("_pragma", "journal_mode=WAL")
	return "file:" + slashed + "?" + q.Encode()
}
