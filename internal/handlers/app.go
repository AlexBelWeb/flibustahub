// Package handlers exposes Wails bindings. No SQL, no business rules.
package handlers

import (
	"context"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
	"github.com/alexbelweb/flibustahub/internal/events"
	"github.com/alexbelweb/flibustahub/internal/platform"
	appsvc "github.com/alexbelweb/flibustahub/internal/services/app"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DBUpdated is the payload of db:updated. Error is set when open or migrate failed.
type DBUpdated struct {
	Error *apperr.Public `json:"error,omitempty"`
}

// Runtime holds the Wails context and window helpers. It is not bound to JS.
type Runtime struct {
	svc   *appsvc.Service
	ctx   context.Context
	guard closeGuard
}

func NewRuntime(svc *appsvc.Service) *Runtime {
	r := &Runtime{svc: svc}
	r.guard.onTimeout = func() {
		if r.ctx != nil {
			runtime.Quit(r.ctx)
		}
	}
	return r
}

func (r *Runtime) SetContext(ctx context.Context) {
	r.ctx = ctx
}

func (r *Runtime) DismissClose() {
	r.guard.dismiss()
}

// BeforeClose is called from OnBeforeClose. true means keep the window open.
func (r *Runtime) BeforeClose() bool {
	if r.guard.shouldForce() || !r.svc.IsImporting() {
		return false
	}
	prevent := r.guard.prevent()
	if prevent && r.ctx != nil {
		runtime.EventsEmit(r.ctx, events.ImportCloseRequested, CloseRequested{
			Committed: r.svc.ImportCommitted(),
		})
	}
	return prevent
}

func (r *Runtime) FocusExistingWindow() {
	if r.ctx == nil {
		return
	}
	runtime.WindowUnminimise(r.ctx)
	runtime.WindowShow(r.ctx)
	runtime.WindowSetAlwaysOnTop(r.ctx, true)
	runtime.WindowSetAlwaysOnTop(r.ctx, false)
}

func (r *Runtime) RestoreWindow() {
	if r.ctx == nil {
		return
	}
	st := r.svc.WindowState()
	fitted := platform.FitWindow(
		platform.Rect{X: st.X, Y: st.Y, W: st.Width, H: st.Height},
		screensAsRects(r.ctx),
	)
	runtime.WindowSetSize(r.ctx, fitted.W, fitted.H)
	runtime.WindowSetPosition(r.ctx, fitted.X, fitted.Y)
	if st.Maximised {
		runtime.WindowMaximise(r.ctx)
	}
}

func (r *Runtime) PersistWindow() {
	if r.ctx == nil {
		return
	}
	w, h := runtime.WindowGetSize(r.ctx)
	x, y := runtime.WindowGetPosition(r.ctx)
	_ = r.svc.SaveWindow(config.WindowState{
		X:         x,
		Y:         y,
		Width:     w,
		Height:    h,
		Maximised: runtime.WindowIsMaximised(r.ctx),
	})
}

func (r *Runtime) ApplyWindowTheme(theme string) {
	if r.ctx == nil {
		return
	}
	switch theme {
	case config.ThemeLight:
		runtime.WindowSetLightTheme(r.ctx)
	case config.ThemeDark:
		runtime.WindowSetDarkTheme(r.ctx)
	default:
		runtime.WindowSetSystemDefaultTheme(r.ctx)
	}
}

func screensAsRects(ctx context.Context) []platform.Rect {
	screens, err := runtime.ScreenGetAll(ctx)
	if err != nil || len(screens) == 0 {
		return nil
	}
	out := make([]platform.Rect, 0, len(screens))
	for _, s := range screens {
		w, h := s.Size.Width, s.Size.Height
		out = append(out, platform.Rect{W: w, H: h})
	}
	return out
}

// App is the Wails-bound facade.
type App struct {
	svc *appsvc.Service
	rt  *Runtime
}

func NewApp(svc *appsvc.Service, rt *Runtime) *App {
	return &App{svc: svc, rt: rt}
}

func (a *App) Bootstrap() appsvc.Bootstrap {
	return a.svc.Bootstrap()
}

func (a *App) RetryStartup() appsvc.Bootstrap {
	return a.svc.RetryStartup()
}

func (a *App) OpenLogsDir() error {
	return a.svc.OpenLogsDir()
}

func (a *App) OpenDataDir() error {
	return a.svc.OpenDataDir()
}

func (a *App) SetLocale(code string) error {
	return a.svc.SetLocale(code)
}

func (a *App) SetTheme(theme string) error {
	return a.svc.SetTheme(theme)
}

func (a *App) SetVisualEffects(mode string) error {
	return a.svc.SetVisualEffects(mode)
}
