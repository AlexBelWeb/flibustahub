// Package app implements application-level services shared by Wails and HTTP.
package app

import (
	"context"
	"errors"
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
	"github.com/alexbelweb/flibustahub/internal/services/diagnostics"
	"github.com/alexbelweb/flibustahub/internal/services/downloads"
	"github.com/alexbelweb/flibustahub/internal/services/inpximport"
	"github.com/alexbelweb/flibustahub/internal/services/maintenance"
	"github.com/alexbelweb/flibustahub/internal/services/secrets"
	"github.com/alexbelweb/flibustahub/internal/services/storage"
)

// Service owns bootstrap state that is not tied to a UI toolkit.
type Service struct {
	cfg            *config.Store
	log            *slog.Logger
	version        string
	commit         string
	built          string
	catalog        *db.DB
	importer       *inpximport.Service
	covers         *covers.Service
	downloads      *downloads.Service
	storage        *storage.Service
	secrets        *secrets.Service
	maint          *maintenance.Service
	diag           *diagnostics.Service
	coverEmit      func(covers.Progress)
	storeEmit      func(storage.Snapshot)
	httpAddr       string
	startErr       error
	instanceStop   func()
	instanceUnlock func()
	instanceFocus  func()

	importMu     sync.Mutex
	importing    bool
	importCancel context.CancelFunc
	lastProgress inpximport.Progress

	updating     atomic.Bool
	openMu       sync.Mutex
	opening      bool
	stopping     bool
	shutdownOnce sync.Once
	// stopBudget is the shared shutdown wait. New sets the production value.
	// A test may set zero so a busy pool fails the close immediately.
	stopBudget time.Duration
	openPause  <-chan struct{}
	openHold   <-chan struct{}
}

func New(cfg *config.Store, log *slog.Logger, version, commit, built string) *Service {
	s := &Service{
		cfg: cfg, log: log, version: version, commit: commit, built: built,
		stopBudget: shutdownBudget,
	}
	s.storage = storage.New(cfg, s.Catalog, s.Logger(), s.emitStorage)
	s.secrets = secrets.New(secrets.Options{
		DataDir: func() string { return s.config().Paths().DataDir },
		Log:     s.Logger(),
	})
	s.maint = maintenance.New(maintenance.Options{
		Catalog:   s.Catalog,
		Importing: s.IsImporting,
		Warming:   func() bool { return s.CoverWarmupProgress().Running },
		Log:       s.Logger(),
	})
	s.diag = diagnostics.New(diagnostics.Options{
		Catalog: s.Catalog,
		Config:  s.config,
		Version: version,
		Commit:  commit,
		Built:   built,
		Log:     s.Logger(),
	})
	return s
}

// SetInstanceFocus is called when a second launch asks this process to come forward.
func (s *Service) SetInstanceFocus(fn func()) {
	s.openMu.Lock()
	s.instanceFocus = fn
	s.openMu.Unlock()
}

// BindInstance keeps the process lock and focus channel. Shutdown stops the
// channel first and drops the lock only after the catalog is closed.
func (s *Service) BindInstance(hold platform.InstanceHold) {
	if hold.StopFocus == nil && hold.Unlock == nil {
		return
	}
	s.openMu.Lock()
	s.instanceStop = hold.StopFocus
	s.instanceUnlock = hold.Unlock
	s.openMu.Unlock()
}

// claimInstance takes the process lock if this process does not hold it yet.
func (s *Service) claimInstance(wait time.Duration) bool {
	s.openMu.Lock()
	if s.instanceUnlock != nil {
		s.openMu.Unlock()
		return true
	}
	dataDir := ""
	focus := s.instanceFocus
	if s.cfg != nil {
		dataDir = s.cfg.Live().DataDir
	}
	s.openMu.Unlock()
	kind, hold, err := platform.DecideStart(
		func() (func(), bool, error) { return platform.AcquireInstance(dataDir) },
		func(timeout time.Duration) bool { return platform.SignalFocus(dataDir, timeout) },
		func(cb func()) (func(), error) { return platform.ListenFocus(dataDir, cb) },
		focus,
		platform.FocusSignalTimeout,
		wait,
	)
	if err != nil {
		logInstanceErr(s.Logger(), err)
	}
	if kind != platform.StartPrimary {
		return false
	}
	s.BindInstance(hold)
	return true
}

// ReleaseInstance stops the focus channel and drops the process lock.
// Safe to call more than once. Shutdown uses the two halves separately.
func (s *Service) ReleaseInstance() {
	s.stopInstanceFocus()
	s.unlockInstance()
}

func (s *Service) stopInstanceFocus() {
	s.openMu.Lock()
	stop := s.instanceStop
	s.instanceStop = nil
	s.openMu.Unlock()
	if stop != nil {
		stop()
	}
}

func (s *Service) unlockInstance() {
	s.openMu.Lock()
	unlock := s.instanceUnlock
	s.instanceUnlock = nil
	s.openMu.Unlock()
	if unlock != nil {
		unlock()
	}
}

func logInstanceErr(log *slog.Logger, err error) {
	var focus *platform.FocusListenError
	if errors.As(err, &focus) {
		log.Warn("focus channel did not start", "err", err)
		return
	}
	log.Error("instance lock failed", "err", err)
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
	if err := s.closeCatalogLocked(shutdownBudget); err != nil {
		s.Logger().Warn("catalog close did not finish", "err", err)
	}
}

func (s *Service) closeCatalogLocked(budget time.Duration) error {
	if s.downloads != nil {
		s.downloads.Stop()
		s.downloads = nil
	}
	if s.covers != nil {
		s.covers.Stop()
		s.covers.Wait(budget)
		s.covers = nil
	}
	if s.catalog != nil {
		if err := s.catalog.CloseWithin(budget); err != nil {
			return err
		}
		s.catalog = nil
	}
	s.importer = nil
	return nil
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
	AIProvider        string                `json:"aiProvider"`
	Capabilities      platform.Capabilities `json:"capabilities"`
	LibraryRoot       string                `json:"libraryRoot"`
	Paths             config.Paths          `json:"paths"`
	SearchIndexReady  bool                  `json:"searchIndexReady"`
	DatabaseUpdating  bool                  `json:"databaseUpdating"`
	CatalogOpening    bool                  `json:"catalogOpening"`
	CatalogReady      bool                  `json:"catalogReady"`
	MediaBase         string                `json:"mediaBase,omitempty"`
	Storage           storage.Snapshot      `json:"storage"`
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
		AIProvider:        live.AIProvider,
		Capabilities:      caps,
		LibraryRoot:       live.LibraryRoot,
		Paths:             st.cfg.Paths(),
		SearchIndexReady:  st.catalog != nil && st.catalog.SearchIndexReady(),
		DatabaseUpdating:  st.updating,
		CatalogOpening:    st.opening,
		CatalogReady:      st.catalog != nil,
		MediaBase:         st.httpAddr,
		Storage:           s.storageSnapshot(),
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
	closeErr := s.closeCatalogLocked(shutdownBudget)
	stopping := s.stopping
	s.openMu.Unlock()
	if closeErr != nil {
		return closeErr
	}
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
		s.downloads = s.newDownloadsLocked()
	}
	s.openMu.Unlock()
	if catalog != nil && s.storage != nil {
		root := cfg.Live().LibraryRoot
		s.storage.PersistVolume(context.Background(), root)
		go s.CheckStorage(context.Background(), true)
	}
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
		closeErr := s.closeCatalogLocked(shutdownBudget)
		s.startErr = cfgErr
		s.openMu.Unlock()
		if closeErr != nil {
			s.Logger().Warn("catalog close did not finish", "err", closeErr)
		}
		s.endOpening()
		return s.Bootstrap()
	}
	if s.Catalog() == nil && !s.claimInstance(2*time.Second) {
		s.SetStartupError(apperr.New(apperr.CodeInstanceRunning, nil))
		s.endOpening()
		return s.Bootstrap()
	}
	_ = s.openCatalogWork()
	s.endOpening()
	return s.Bootstrap()
}

func (s *Service) OpenDownloadsDir() error {
	dir := s.config().Paths().DownloadsDir
	if dir == "" {
		return apperr.New(apperr.CodeDownloadsDirUnusable, nil)
	}
	if err := platform.OpenDir(dir); err != nil {
		return apperr.Wrap(apperr.CodeOpenDirFailed, err, nil)
	}
	return nil
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
