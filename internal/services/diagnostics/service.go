// Package diagnostics collects a support snapshot, archive, and issue report.
package diagnostics

import (
	"context"
	"log/slog"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/platform"
)

const recentLogLimit = 50

// Snapshot is the diagnostics panel payload. Paths stay; secrets do not.
type Snapshot struct {
	Version        string       `json:"version"`
	Commit         string       `json:"commit"`
	BuildDate      string       `json:"buildDate"`
	OS             string       `json:"os"`
	Arch           string       `json:"arch"`
	WebView        string       `json:"webView"`
	INPXVersion    string       `json:"inpxVersion"`
	Works          int          `json:"works"`
	Authors        int          `json:"authors"`
	ImportedAt     string       `json:"importedAt"`
	UnnamedGenres  int          `json:"unnamedGenres"`
	AIProvider     string       `json:"aiProvider"`
	OPDSEnabled    bool         `json:"opdsEnabled"`
	Paths          config.Paths `json:"paths"`
	RecentLogLines []string     `json:"recentLogLines"`
}

// Options wire catalog, config and log access.
type Options struct {
	Catalog func() *db.DB
	Config  func() *config.Store
	Version string
	Commit  string
	Built   string
	Log     *slog.Logger
}

// Service builds snapshots, zip archives and issue reports.
type Service struct {
	catalog func() *db.DB
	config  func() *config.Store
	version string
	commit  string
	built   string
	log     *slog.Logger
}

func New(opt Options) *Service {
	s := &Service{
		catalog: opt.Catalog,
		config:  opt.Config,
		version: opt.Version,
		commit:  opt.Commit,
		built:   opt.Built,
		log:     opt.Log,
	}
	if s.catalog == nil {
		s.catalog = func() *db.DB { return nil }
	}
	if s.config == nil {
		s.config = func() *config.Store { return nil }
	}
	if s.log == nil {
		s.log = slog.Default()
	}
	return s
}

// Snapshot collects the diagnostics panel. Missing catalog fields stay empty.
func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	out := Snapshot{
		Version:        s.version,
		Commit:         s.commit,
		BuildDate:      s.built,
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		WebView:        platform.WebViewVersion(),
		RecentLogLines: []string{},
	}
	if cfg := s.config(); cfg != nil {
		live := cfg.Live()
		out.AIProvider = live.AIProvider
		out.OPDSEnabled = live.OPDSEnabled
		out.Paths = cfg.Paths()
		out.RecentLogLines = lastWarnErrors(filepath.Join(out.Paths.LogsDir, "app.log"), recentLogLimit)
	}
	d := s.catalog()
	if d == nil || d.Read == nil {
		return out, nil
	}
	if v, err := db.INPXVersion(ctx, d.Read); err != nil {
		return Snapshot{}, apperr.Wrap(apperr.CodeDiagFailed, err, nil)
	} else {
		out.INPXVersion = v
	}
	if v, err := db.Meta(ctx, d.Read, db.MetaINPXImportedAt); err != nil {
		return Snapshot{}, apperr.Wrap(apperr.CodeDiagFailed, err, nil)
	} else {
		out.ImportedAt = v
	}
	if n, err := metaInt(ctx, d.Read, db.MetaWorksListable); err != nil {
		return Snapshot{}, err
	} else {
		out.Works = n
	}
	if n, err := metaInt(ctx, d.Read, db.MetaAuthorsTotal); err != nil {
		return Snapshot{}, err
	} else {
		out.Authors = n
	}
	if n, err := unnamedGenreCount(ctx, d.Read); err != nil {
		return Snapshot{}, err
	} else {
		out.UnnamedGenres = n
	}
	return out, nil
}

func metaInt(ctx context.Context, e db.Execer, key string) (int, error) {
	raw, err := db.Meta(ctx, e, key)
	if err != nil {
		return 0, apperr.Wrap(apperr.CodeDiagFailed, err, nil)
	}
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, nil
	}
	return n, nil
}

func unnamedGenreCount(ctx context.Context, e db.Execer) (int, error) {
	var n int
	err := e.QueryRowContext(ctx, `SELECT count(*) FROM genres WHERE name_ru = code`).Scan(&n)
	if err != nil {
		return 0, apperr.Wrap(apperr.CodeDiagFailed, err, nil)
	}
	return n, nil
}
