package main

import (
	"context"
	"embed"
	"log/slog"
	"os"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
	"github.com/alexbelweb/flibustahub/internal/handlers"
	"github.com/alexbelweb/flibustahub/internal/httpapi"
	"github.com/alexbelweb/flibustahub/internal/logging"
	"github.com/alexbelweb/flibustahub/internal/platform"
	appsvc "github.com/alexbelweb/flibustahub/internal/services/app"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
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
	store, err := config.Load(dataDir, bootLog)
	if err != nil {
		bootLog.Error("config load failed", "err", err)
		os.Exit(1)
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
		os.Exit(1)
	}

	logger.Info("starting", "version", version, "commit", commit, "buildDate", buildDate)

	svc := appsvc.New(store, logger, version, commit, buildDate)
	ui := handlers.NewApp(svc)
	win := handlers.NewRuntime(svc)
	httpServer := httpapi.New(logger)
	if startErr := httpServer.Start("127.0.0.1", store.Live().OPDSPort); startErr != nil {
		logger.Warn("loopback http did not start", "err", startErr)
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
		},
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			win.SetContext(ctx)
			win.PersistWindow()
			shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = httpServer.Shutdown(shutCtx)
			return false
		},
		Bind: []interface{}{
			ui,
		},
		ErrorFormatter: apperr.FormatWails,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "flibustahub-single-instance",
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				win.FocusExistingWindow()
			},
		},
		Linux: &linux.Options{
			ProgramName:      "FlibustaHub",
			WebviewGpuPolicy: linux.WebviewGpuPolicyOnDemand,
		},
	})
	if err != nil {
		logger.Error("wails exited", "err", err)
		os.Exit(1)
	}
}
