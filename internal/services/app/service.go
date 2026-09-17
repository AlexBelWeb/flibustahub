// Package app implements application-level services shared by Wails and HTTP.
package app

import (
	"context"
	"log/slog"
	"os"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/platform"
)

// Service owns bootstrap state that is not tied to a UI toolkit.
type Service struct {
	cfg      *config.Store
	log      *slog.Logger
	version  string
	commit   string
	built    string
	catalog  *db.DB
	startErr error
}

func New(cfg *config.Store, log *slog.Logger, version, commit, built string) *Service {
	return &Service{cfg: cfg, log: log, version: version, commit: commit, built: built}
}

// AttachCatalog stores the catalog handle and a startup error from config or DB.
func (s *Service) AttachCatalog(catalog *db.DB, startup error) {
	s.catalog = catalog
	s.startErr = startup
}

func (s *Service) Catalog() *db.DB {
	return s.catalog
}

func (s *Service) CloseCatalog() {
	if s.catalog != nil {
		_ = s.catalog.Close()
		s.catalog = nil
	}
}

// Bootstrap is the payload the UI needs on first paint.
type Bootstrap struct {
	Version           string                `json:"version"`
	Commit            string                `json:"commit"`
	BuildDate         string                `json:"buildDate"`
	Locale            string                `json:"locale"`
	Theme             string                `json:"theme"`
	VisualEffectsPref string                `json:"visualEffectsPref"`
	Capabilities      platform.Capabilities `json:"capabilities"`
	LibraryRoot       string                `json:"libraryRoot"`
	Paths             config.Paths          `json:"paths"`
	StartupError      *apperr.Public        `json:"startupError,omitempty"`
}

func (s *Service) Bootstrap() Bootstrap {
	live := s.cfg.Live()
	locale := live.Locale
	if locale == "" {
		locale = platform.DetectLocale()
	} else {
		locale = platform.MatchLocale(locale)
	}
	caps := platform.Detect(live.VisualEffects)
	out := Bootstrap{
		Version:           s.version,
		Commit:            s.commit,
		BuildDate:         s.built,
		Locale:            locale,
		Theme:             live.Theme,
		VisualEffectsPref: live.VisualEffects,
		Capabilities:      caps,
		LibraryRoot:       live.LibraryRoot,
		Paths:             s.cfg.Paths(),
	}
	if s.startErr != nil {
		p := apperr.As(s.startErr).Public()
		out.StartupError = &p
	}
	return out
}

func (s *Service) SetLocale(code string) error {
	if code != "ru" && code != "en" {
		return apperr.New(apperr.CodeInvalidLocale, map[string]string{"locale": code})
	}
	return s.cfg.Update(func(f *config.File) { f.Locale = code })
}

func (s *Service) SetTheme(theme string) error {
	switch theme {
	case config.ThemeSystem, config.ThemeDark, config.ThemeLight:
	default:
		return apperr.New(apperr.CodeInvalidTheme, map[string]string{"theme": theme})
	}
	return s.cfg.Update(func(f *config.File) { f.Theme = theme })
}

func (s *Service) SetVisualEffects(mode string) error {
	switch mode {
	case config.EffectsAuto, config.EffectsFull, config.EffectsReduced:
	default:
		return apperr.New(apperr.CodeInvalidEffects, map[string]string{"mode": mode})
	}
	return s.cfg.Update(func(f *config.File) { f.VisualEffects = mode })
}

func (s *Service) SaveWindow(state config.WindowState) error {
	return s.cfg.Update(func(f *config.File) { f.Window = state })
}

func (s *Service) WindowState() config.WindowState {
	return s.cfg.Live().Window
}

func (s *Service) Logger() *slog.Logger {
	if s.log != nil {
		return s.log
	}
	return slog.Default()
}

// RetryStartup reloads config and reopens the catalog.
func (s *Service) RetryStartup() Bootstrap {
	dataDir := s.cfg.Live().DataDir
	store, cfgErr := config.Load(dataDir, s.Logger())
	s.cfg = store
	s.CloseCatalog()
	if cfgErr != nil {
		s.startErr = cfgErr
		return s.Bootstrap()
	}
	paths := store.Paths()
	catalog, dbErr := db.Open(context.Background(), db.Options{
		Path:       paths.DBPath,
		BackupsDir: paths.BackupsDir,
		Log:        s.Logger(),
	})
	s.catalog = catalog
	s.startErr = dbErr
	return s.Bootstrap()
}

func (s *Service) OpenLogsDir() error {
	dir := s.cfg.Paths().LogsDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.CodeOpenDirFailed, err, nil)
	}
	if err := platform.OpenDir(dir); err != nil {
		return apperr.Wrap(apperr.CodeOpenDirFailed, err, nil)
	}
	return nil
}

func (s *Service) OpenDataDir() error {
	dir := s.cfg.Paths().DataDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.CodeOpenDirFailed, err, nil)
	}
	if err := platform.OpenDir(dir); err != nil {
		return apperr.Wrap(apperr.CodeOpenDirFailed, err, nil)
	}
	return nil
}
