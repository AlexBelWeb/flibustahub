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
	if got.SearchIndexReady {
		t.Fatal("search index must not be ready without a catalog")
	}
}

func TestBootstrapDatabaseUpdating(t *testing.T) {
	svc := newTestService(t)
	svc.SetDatabaseUpdating(true)
	got := svc.Bootstrap()
	if !got.DatabaseUpdating {
		t.Fatal("expected databaseUpdating")
	}
	if got.CatalogReady {
		t.Fatal("catalog must stay closed while the database is updating")
	}
	if got.SearchIndexReady {
		t.Fatal("search index must not be ready while the catalog is still closed")
	}
	if got.CatalogOpening {
		t.Fatal("pre-open splash must not look like an in-flight open")
	}
}

func TestBootstrapSurfacesMigrateFailureWithoutCatalog(t *testing.T) {
	svc := newTestService(t)
	svc.AttachCatalog(nil, apperr.New(apperr.CodeDBMigrateFailed, map[string]string{"name": "002_works_added_date"}))
	got := svc.Bootstrap()
	if got.CatalogReady {
		t.Fatal("failed open must not report catalogReady")
	}
	if got.DatabaseUpdating {
		t.Fatal("updating flag must be clear on a failed open")
	}
	if got.StartupError == nil || got.StartupError.Code != apperr.CodeDBMigrateFailed {
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
	if !got.CatalogReady {
		t.Fatal("retry must report catalogReady")
	}
	if got.CatalogOpening {
		t.Fatal("finished retry must not report catalogOpening")
	}
	if got.DatabaseUpdating {
		t.Fatal("finished retry must not report databaseUpdating")
	}
}

func TestSetCatalogViewAndSidebar(t *testing.T) {
	svc := newTestService(t)
	if err := svc.SetCatalogView("mosaic"); apperr.As(err).Code != apperr.CodeInvalidCatalogView {
		t.Fatalf("got %v", err)
	}
	if err := svc.SetCatalogView(config.CatalogViewTile); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetSidebarCollapsed(true); err != nil {
		t.Fatal(err)
	}
	got := svc.Bootstrap()
	if got.CatalogView != config.CatalogViewTile || !got.SidebarCollapsed {
		t.Fatalf("%+v", got)
	}
}
