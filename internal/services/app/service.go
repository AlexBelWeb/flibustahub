// Package app implements application-level services shared by Wails and HTTP.
package app

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
	"github.com/alexbelweb/flibustahub/internal/data"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/platform"
	"github.com/alexbelweb/flibustahub/internal/services/covers"
	"github.com/alexbelweb/flibustahub/internal/services/inpximport"
)

// Service owns bootstrap state that is not tied to a UI toolkit.
type Service struct {
	cfg       *config.Store
	log       *slog.Logger
	version   string
	commit    string
	built     string
	catalog   *db.DB
	importer  *inpximport.Service
	covers    *covers.Service
	coverEmit func(covers.Progress)
	httpAddr  string
	startErr  error

	importMu     sync.Mutex
	importing    bool
	importCancel context.CancelFunc
	lastProgress inpximport.Progress

	updating     atomic.Bool
	openMu       sync.Mutex
	opening      bool
	stopping     bool
	shutdownOnce sync.Once
	openPause    <-chan struct{}
	openHold     <-chan struct{}
}

func New(cfg *config.Store, log *slog.Logger, version, commit, built string) *Service {
	return &Service{cfg: cfg, log: log, version: version, commit: commit, built: built}
}

// AttachCatalog stores the catalog handle and a startup error from config or DB.
func (s *Service) AttachCatalog(catalog *db.DB, startup error) {
	s.openMu.Lock()
	defer s.openMu.Unlock()
	s.catalog = catalog
	s.startErr = startup
	if catalog != nil {
		s.importer = inpximport.New(catalog, s.log)
	} else {
		s.importer = nil
	}
}

func (s *Service) SetStartupError(err error) {
	s.openMu.Lock()
	defer s.openMu.Unlock()
	s.startErr = err
}

func (s *Service) Catalog() *db.DB {
	s.openMu.Lock()
	defer s.openMu.Unlock()
	return s.catalog
}

func (s *Service) CloseCatalog() {
	s.openMu.Lock()
	defer s.openMu.Unlock()
	s.closeCatalogLocked(shutdownBudget)
}

func (s *Service) closeCatalogLocked(budget time.Duration) {
	if s.covers != nil {
		s.covers.Stop()
		s.covers.Wait(budget)
		s.covers = nil
	}
	if s.catalog != nil {
		_ = s.catalog.CloseWithin(budget)
		s.catalog = nil
	}
	s.importer = nil
}

// Bootstrap is the payload the UI needs on first paint.
type Bootstrap struct {
	Version           string                `json:"version"`
	Commit            string                `json:"commit"`
	BuildDate         string                `json:"buildDate"`
	Locale            string                `json:"locale"`
	Theme             string                `json:"theme"`
	VisualEffectsPref string                `json:"visualEffectsPref"`
	SidebarCollapsed  bool                  `json:"sidebarCollapsed"`
	CatalogView       string                `json:"catalogView"`
	Capabilities      platform.Capabilities `json:"capabilities"`
	LibraryRoot       string                `json:"libraryRoot"`
	Paths             config.Paths          `json:"paths"`
	SearchIndexReady  bool                  `json:"searchIndexReady"`
	DatabaseUpdating  bool                  `json:"databaseUpdating"`
	CatalogOpening    bool                  `json:"catalogOpening"`
	CatalogReady      bool                  `json:"catalogReady"`
	MediaBase         string                `json:"mediaBase,omitempty"`
	StartupError      *apperr.Public        `json:"startupError,omitempty"`
}

func (s *Service) config() *config.Store {
	s.openMu.Lock()
	defer s.openMu.Unlock()
	return s.cfg
}

type startSnap struct {
	cfg      *config.Store
	catalog  *db.DB
	importer *inpximport.Service
	startErr error
	updating bool
	opening  bool
	httpAddr string
}

func (s *Service) snap() startSnap {
	s.openMu.Lock()
	defer s.openMu.Unlock()
	return startSnap{
		cfg:      s.cfg,
		catalog:  s.catalog,
		importer: s.importer,
		startErr: s.startErr,
		updating: s.updating.Load(),
		opening:  s.opening,
		httpAddr: s.httpAddr,
	}
}

func (s *Service) Bootstrap() Bootstrap {
	st := s.snap()
	live := st.cfg.Live()
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
		SidebarCollapsed:  live.SidebarCollapsed,
		CatalogView:       live.CatalogView,
		Capabilities:      caps,
		LibraryRoot:       live.LibraryRoot,
		Paths:             st.cfg.Paths(),
		SearchIndexReady:  st.catalog != nil && st.catalog.SearchIndexReady(),
		DatabaseUpdating:  st.updating,
		CatalogOpening:    st.opening,
		CatalogReady:      st.catalog != nil,
		MediaBase:         st.httpAddr,
	}
	if st.startErr != nil {
		p := apperr.As(st.startErr).Public()
		out.StartupError = &p
	}
	return out
}

func (s *Service) SetLocale(code string) error {
	if code != "ru" && code != "en" {
		return apperr.New(apperr.CodeInvalidLocale, map[string]string{"locale": code})
	}
	return s.config().Update(func(f *config.File) { f.Locale = code })
}

func (s *Service) SetTheme(theme string) error {
	switch theme {
	case config.ThemeSystem, config.ThemeDark, config.ThemeLight:
	default:
		return apperr.New(apperr.CodeInvalidTheme, map[string]string{"theme": theme})
	}
	return s.config().Update(func(f *config.File) { f.Theme = theme })
}

func (s *Service) SetVisualEffects(mode string) error {
	switch mode {
	case config.EffectsAuto, config.EffectsFull, config.EffectsReduced:
	default:
		return apperr.New(apperr.CodeInvalidEffects, map[string]string{"mode": mode})
	}
	return s.config().Update(func(f *config.File) { f.VisualEffects = mode })
}

func (s *Service) SetSidebarCollapsed(collapsed bool) error {
	return s.config().Update(func(f *config.File) { f.SidebarCollapsed = collapsed })
}

func (s *Service) SetCatalogView(view string) error {
	switch view {
	case config.CatalogViewTile, config.CatalogViewTable:
	default:
		return apperr.New(apperr.CodeInvalidCatalogView, map[string]string{"view": view})
	}
	return s.config().Update(func(f *config.File) { f.CatalogView = view })
}

func (s *Service) WaitSearchIndex(ctx context.Context) error {
	st := s.snap()
	if st.catalog == nil {
		return nil
	}
	return st.catalog.WaitSearchIndex(ctx)
}

func (s *Service) SaveWindow(state config.WindowState) error {
	return s.config().Update(func(f *config.File) { f.Window = state })
}

func (s *Service) WindowState() config.WindowState {
	return s.config().Live().Window
}

func (s *Service) Logger() *slog.Logger {
	if s.log != nil {
		return s.log
	}
	return slog.Default()
}

// SetDatabaseUpdating marks a start that still has to apply a catalog migration
// so the window can show that state before Open runs. It is not the in-flight
// open flag; that is catalogOpening.
func (s *Service) SetDatabaseUpdating(v bool) {
	s.updating.Store(v)
}

func (s *Service) tryBeginOpening() bool {
	s.openMu.Lock()
	defer s.openMu.Unlock()
	if s.opening || s.stopping {
		return false
	}
	s.opening = true
	return true
}

func (s *Service) endOpening() {
	s.openMu.Lock()
	s.opening = false
	s.updating.Store(false)
	s.openMu.Unlock()
}

func (s *Service) peekStartErr() error {
	s.openMu.Lock()
	defer s.openMu.Unlock()
	return s.startErr
}

// OpenCatalog closes any previous handle and opens the file. A concurrent
// caller returns the current start error instead of starting a second open.
func (s *Service) OpenCatalog() error {
	if !s.tryBeginOpening() {
		return s.peekStartErr()
	}
	defer s.endOpening()
	return s.openCatalogWork()
}

func (s *Service) openCatalogWork() error {
	s.openMu.Lock()
	cfg := s.cfg
	s.openMu.Unlock()

	paths := cfg.Paths()
	pending, pendErr := db.HasPendingMigrations(context.Background(), paths.DBPath)
	if pendErr != nil {
		s.Logger().Warn("pending migrations check failed", "err", pendErr)
	}
	s.updating.Store(pending)

	if pause := s.openPause; pause != nil {
		<-pause
	}

	s.openMu.Lock()
	s.closeCatalogLocked(shutdownBudget)
	stopping := s.stopping
	s.openMu.Unlock()
	if stopping {
		return s.peekStartErr()
	}

	catalog, dbErr := db.Open(context.Background(), db.Options{
		Path:       paths.DBPath,
		BackupsDir: paths.BackupsDir,
		Log:        s.Logger(),
	})
	if hold := s.openHold; hold != nil {
		<-hold
	}

	importer := (*inpximport.Service)(nil)
	if catalog != nil {
		data.Load(paths.DataDir, s.Logger())
		if err := catalog.SyncGenreNames(context.Background()); err != nil {
			s.Logger().Warn("genre names not synced", "err", err)
		}
		importer = inpximport.New(catalog, s.Logger())
	}

	s.openMu.Lock()
	if s.stopping {
		s.openMu.Unlock()
		if catalog != nil {
			_ = catalog.Close()
		}
		return dbErr
	}
	s.catalog = catalog
	s.importer = importer
	s.startErr = dbErr
	if catalog != nil {
		s.covers = s.newCoversLocked()
	}
	s.openMu.Unlock()
	return dbErr
}

// RetryStartup reloads config and reopens the catalog without restarting the process.
// A call that arrives while an open is already running returns the current
// bootstrap snapshot and does not start another open.
func (s *Service) RetryStartup() Bootstrap {
	if !s.tryBeginOpening() {
		return s.Bootstrap()
	}
	// The returned snapshot is evaluated before deferred functions run, so
	// endOpening must run first; otherwise catalogOpening stays true.
	defer s.endOpening()

	s.openMu.Lock()
	dataDir := s.cfg.Live().DataDir
	s.openMu.Unlock()

	store, cfgErr := config.Load(dataDir, s.Logger())

	s.openMu.Lock()
	s.cfg = store
	s.openMu.Unlock()

	if cfgErr != nil {
		s.openMu.Lock()
		s.closeCatalogLocked(shutdownBudget)
		s.startErr = cfgErr
		s.openMu.Unlock()
		s.endOpening()
		return s.Bootstrap()
	}
	_ = s.openCatalogWork()
	s.endOpening()
	return s.Bootstrap()
}

func (s *Service) OpenLogsDir() error {
	dir := s.config().Paths().LogsDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.CodeOpenDirFailed, err, nil)
	}
	if err := platform.OpenDir(dir); err != nil {
		return apperr.Wrap(apperr.CodeOpenDirFailed, err, nil)
	}
	return nil
}

func (s *Service) OpenDataDir() error {
	dir := s.config().Paths().DataDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.CodeOpenDirFailed, err, nil)
	}
	if err := platform.OpenDir(dir); err != nil {
		return apperr.Wrap(apperr.CodeOpenDirFailed, err, nil)
	}
	return nil
}
