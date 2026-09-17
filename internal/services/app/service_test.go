package app

import (
	"log/slog"
	"testing"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	store, err := config.Load(t.TempDir(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	return New(store, slog.New(slog.DiscardHandler), "dev", "abc", "now")
}

func TestSetLocaleRejectsUnknown(t *testing.T) {
	svc := newTestService(t)
	err := svc.SetLocale("de")
	if apperr.As(err).Code != apperr.CodeInvalidLocale {
		t.Fatalf("got %v", err)
	}
}

func TestSetLocalePersists(t *testing.T) {
	svc := newTestService(t)
	if err := svc.SetLocale("en"); err != nil {
		t.Fatal(err)
	}
	if svc.cfg.Live().Locale != "en" {
		t.Fatal("locale not saved")
	}
}

func TestBootstrapUsesSystemLocaleWhenEmpty(t *testing.T) {
	t.Setenv("LC_ALL", "ru_RU.UTF-8")
	svc := newTestService(t)
	got := svc.Bootstrap()
	if got.Locale != "ru" {
		t.Fatalf("locale = %q", got.Locale)
	}
	if got.Version != "dev" {
		t.Fatalf("version = %q", got.Version)
	}
}

func TestBootstrapSurfacesStartupError(t *testing.T) {
	svc := newTestService(t)
	svc.AttachCatalog(nil, apperr.New(apperr.CodeDBOpenFailed, nil))
	got := svc.Bootstrap()
	if got.StartupError == nil || got.StartupError.Code != apperr.CodeDBOpenFailed {
		t.Fatalf("startup = %+v", got.StartupError)
	}
}

func TestRetryStartupOpensCatalog(t *testing.T) {
	svc := newTestService(t)
	svc.AttachCatalog(nil, apperr.New(apperr.CodeDBOpenFailed, nil))
	got := svc.RetryStartup()
	t.Cleanup(svc.CloseCatalog)
	if got.StartupError != nil {
		t.Fatalf("retry failed: %+v", got.StartupError)
	}
	if svc.Catalog() == nil {
		t.Fatal("catalog not opened")
	}
}
