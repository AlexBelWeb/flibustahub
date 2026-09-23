package main

import (
	"context"
	"embed"
	"log/slog"
	"os"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
	catalogdb "github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/events"
	"github.com/alexbelweb/flibustahub/internal/handlers"
	"github.com/alexbelweb/flibustahub/internal/httpapi"
	"github.com/alexbelweb/flibustahub/internal/logging"
	"github.com/alexbelweb/flibustahub/internal/platform"
	appsvc "github.com/alexbelweb/flibustahub/internal/services/app"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Populated via -ldflags at release time. Git tags are the source of version.
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	dataDir := os.Getenv("FLIBUSTAHUB_DATADIR")
	bootLog := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	store, cfgErr := config.Load(dataDir, bootLog)
	if cfgErr != nil {
		bootLog.Error("config load failed", "err", cfgErr)
	}
	paths := store.Paths()
	toStdout := version == "dev" || os.Getenv("FLIBUSTAHUB_LOG_STDOUT") == "1"
	logger, _, err := logging.Setup(logging.Options{
		LogsDir:     paths.LogsDir,
		ToStdout:    toStdout,
		Level:       slog.LevelInfo,
		LibraryRoot: store.Live().LibraryRoot,
		DataDir:     paths.DataDir,
	})
	if err != nil {
		bootLog.Error("logger setup failed", "err", err)
		logger = bootLog
	}

	svc := appsvc.New(store, logger, version, commit, buildDate)
	win := handlers.NewRuntime(svc)
	ui := handlers.NewApp(svc, win)
	httpServer := httpapi.New(logger)
	httpServer.SetCovers(svc)

	svc.SetInstanceFocus(win.FocusExistingWindow)
	kind, releaseInstance, lockErr := platform.DecideStart(
		func() (func(), bool, error) { return platform.AcquireInstance(paths.DataDir) },
		func(timeout time.Duration) bool { return platform.SignalFocus(paths.DataDir, timeout) },
		func(cb func()) (func(), error) { return platform.ListenFocus(paths.DataDir, cb) },
		win.FocusExistingWindow,
		platform.FocusSignalTimeout,
		platform.InstanceClaimWait,
	)
	if lockErr != nil {
		logger.Error("instance lock failed", "err", lockErr)
	}
	defer svc.ReleaseInstance()
	if kind == platform.StartExit {
		logger.Info("focusing existing instance")
		os.Exit(0)
	}

	pending := false
	var startCatalog func() error
	if kind == platform.StartBlocked {
		logger.Info("another instance is running")
		svc.AttachCatalog(nil, apperr.New(apperr.CodeInstanceRunning, nil))
	} else {
		svc.BindInstance(releaseInstance)
		logger.Info("starting", "version", version, "commit", commit, "buildDate", buildDate)
		// First paint can show the migration splash before OnStartup calls Open.
		// OpenCatalog re-checks the same path, including on RetryStartup.
		var pendErr error
		pending, pendErr = catalogdb.HasPendingMigrations(context.Background(), paths.DBPath)
		if pendErr != nil {
			logger.Error("pending migrations check failed", "err", pendErr)
		}
		if pending {
			svc.SetDatabaseUpdating(true)
		}
		if cfgErr != nil {
			svc.SetStartupError(cfgErr)
		}
		startCatalog = func() error {
			dbErr := svc.OpenCatalog()
			if dbErr != nil {
				logger.Error("catalog open failed", "err", dbErr)
			} else {
				logger.Info("catalog ready")
			}
			shown := cfgErr
			if shown == nil {
				shown = dbErr
			} else {
				svc.SetStartupError(cfgErr)
			}
			if startErr := httpServer.Start("127.0.0.1", store.Live().OPDSPort); startErr != nil {
				logger.Warn("loopback http did not start", "err", startErr)
			} else if addr := httpServer.Addr(); addr != "" {
				svc.SetHTTPAddr("http://" + addr)
			}
			return shown
		}
	}

	saved := store.Live().Window
	width, height := platform.DefaultWidth, platform.DefaultHeight
	if saved.Width >= platform.MinWidth && saved.Height >= platform.MinHeight {
		width, height = saved.Width, saved.Height
	}

	err = wails.Run(&options.App{
		Title:            "FlibustaHub",
		Width:            width,
		Height:           height,
		MinWidth:         platform.MinWidth,
		MinHeight:        platform.MinHeight,
		Frameless:        false,
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 20, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			win.SetContext(ctx)
			win.RestoreWindow()
			win.ApplyWindowTheme(store.Live().Theme)
			svc.AfterWindow()
			if startCatalog != nil {
				go func() {
					err := startCatalog()
					payload := handlers.DBUpdated{}
					if err != nil {
						p := apperr.As(err).Public()
						payload.Error = &p
					}
					runtime.EventsEmit(ctx, events.DBUpdated, payload)
					if err == nil {
						_ = svc.WaitSearchIndex(ctx)
						runtime.EventsEmit(ctx, events.SearchIndexReady)
					}
				}()
			}
		},
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			win.SetContext(ctx)
			if win.BeforeClose() {
				return true
			}
			win.PersistWindow()
			svc.Shutdown(httpServer.Shutdown)
			return false
		},
		OnShutdown: func(_ context.Context) {
			svc.Shutdown(httpServer.Shutdown)
		},
		Bind: []interface{}{
			ui,
		},
		ErrorFormatter: apperr.FormatWails,
		Linux: &linux.Options{
			ProgramName:      "FlibustaHub",
			WebviewGpuPolicy: linux.WebviewGpuPolicyOnDemand,
		},
	})
	svc.Shutdown(httpServer.Shutdown)
	if err != nil {
		logger.Error("wails exited", "err", err)
		os.Exit(1)
	}
}
