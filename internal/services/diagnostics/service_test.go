package diagnostics

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/alexbelweb/flibustahub/internal/config"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/services/secrets"
)

func TestSnapshotFillsPathsAndLogTail(t *testing.T) {
	svc, _, logsDir := openDiag(t)
	writeLog(t, logsDir, []string{
		`{"level":"INFO","msg":"start"}`,
		`{"level":"WARN","msg":"one"}`,
		`{"level":"ERROR","msg":"two"}`,
		`{"level":"INFO","msg":"skip"}`,
	})
	snap, err := svc.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.OS == "" || snap.Arch == "" {
		t.Fatalf("os/arch empty: %+v", snap)
	}
	if snap.WebView == "" {
		t.Fatal("webview must be a value or unknown, not empty")
	}
	if snap.Paths.LogsDir != logsDir {
		t.Fatalf("logs dir = %q", snap.Paths.LogsDir)
	}
	if len(snap.RecentLogLines) != 2 {
		t.Fatalf("log lines = %v", snap.RecentLogLines)
	}
	if !strings.Contains(snap.RecentLogLines[0], `"one"`) || !strings.Contains(snap.RecentLogLines[1], `"two"`) {
		t.Fatalf("order = %v", snap.RecentLogLines)
	}
}

func TestLastWarnErrorsKeepsLinesAfterLongLine(t *testing.T) {
	dir := t.TempDir()
	longMsg := strings.Repeat("q", 200_000)
	longLine := `{"level":"ERROR","msg":"` + longMsg + `"}`
	writeLog(t, dir, []string{
		`{"level":"WARN","msg":"before"}`,
		longLine,
		`{"level":"WARN","msg":"after"}`,
	})
	got := lastWarnErrors(filepath.Join(dir, "app.log"), 50)
	if len(got) != 3 {
		t.Fatalf("lines = %d, want 3 (long line must not drop the rest): %v", len(got), brief(got))
	}
	if !strings.Contains(got[0], `"before"`) || !strings.Contains(got[2], `"after"`) {
		t.Fatalf("surrounding lines lost: %v", brief(got))
	}
	if utf8.RuneCountInString(got[1]) > logLineMaxRunes {
		t.Fatalf("long line was not truncated: %d runes", utf8.RuneCountInString(got[1]))
	}
	if !strings.Contains(strings.ToLower(got[1]), `"level":"error"`) {
		t.Fatalf("truncated error line dropped its level: %s", got[1][:min(80, len(got[1]))])
	}
}

func brief(lines []string) []string {
	out := make([]string, len(lines))
	for i, s := range lines {
		if len(s) > 60 {
			out[i] = s[:60] + "…"
		} else {
			out[i] = s
		}
	}
	return out
}

func TestArchiveOmitsSecretsAndPersonalData(t *testing.T) {
	svc, catalog, logsDir := openDiag(t)
	secret := "sk-diag-probe-key-value-not-for-archive"
	store := secrets.New(secrets.Options{
		DataDir: func() string { return filepath.Dir(logsDir) },
		Log:     slog.New(slog.DiscardHandler),
	})
	if err := store.Set("openai", secret); err != nil {
		t.Fatal(err)
	}
	comment := "unique-diag-comment-must-not-leak"
	if _, err := catalog.Write.Exec(`UPDATE works SET comment = ?, comment_updated_at = ? WHERE id = (SELECT id FROM works LIMIT 1)`, comment, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Write.Exec(`UPDATE works SET rating = 9, rating_updated_at = ? WHERE id = (SELECT id FROM works LIMIT 1)`, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	writeLog(t, logsDir, []string{`{"level":"WARN","msg":"archive probe"}`})

	dir := t.TempDir()
	got, err := svc.WriteArchive(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(got.Path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(secret)) {
		t.Fatal("secret value leaked into the archive")
	}
	if bytes.Contains(raw, []byte(comment)) {
		t.Fatal("comment leaked into the archive")
	}
	names := zipNames(t, got.Path)
	joined := strings.Join(names, " ")
	if !strings.Contains(joined, "versions.json") || !strings.Contains(joined, "config.json") || !strings.Contains(joined, "db-stats.json") {
		t.Fatalf("archive files = %v", names)
	}
	if strings.Contains(joined, "catalog.sqlite") || strings.Contains(joined, "secrets") {
		t.Fatalf("forbidden file in archive: %v", names)
	}
	if hit, err := zipContains(got.Path, `"rating"`); err != nil {
		t.Fatal(err)
	} else if hit {
		t.Fatal("archive must not mention rating")
	}
	if hit, err := zipContains(got.Path, `"comment"`); err != nil {
		t.Fatal(err)
	} else if hit {
		t.Fatal("archive must not mention comment")
	}
}

func TestIssueURLDoesNotTruncateBody(t *testing.T) {
	desc := strings.Repeat("x", 9000)
	rep := BuildReport(IssueKindBug, desc, Snapshot{Version: "dev", OS: "windows", Arch: "amd64", WebView: "unknown"})
	if rep.URLFits {
		t.Fatal("expected URL overflow")
	}
	if !strings.Contains(rep.Body, desc) {
		t.Fatal("body must stay complete when the URL does not fit")
	}
	if strings.Contains(rep.GitHubURL, "body=") {
		t.Fatal("overflow URL must not silently carry a truncated body")
	}
}

func TestIssueURLFitsShortBody(t *testing.T) {
	rep := BuildReport(IssueKindIdea, "add a filter", Snapshot{Version: "dev", WebView: "unknown"})
	if !rep.URLFits {
		t.Fatal("short body must fit")
	}
	if !strings.Contains(rep.GitHubURL, "body=") {
		t.Fatal("short URL must include the body")
	}
	if rep.Kind != IssueKindIdea {
		t.Fatalf("kind = %q", rep.Kind)
	}
}

func openDiag(t *testing.T) (*Service, *db.DB, string) {
	t.Helper()
	root := t.TempDir()
	store, err := config.Load(root, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	d, err := db.Open(context.Background(), db.Options{
		Path:       filepath.Join(root, "catalog.sqlite"),
		BackupsDir: filepath.Join(root, "backups"),
		Log:        slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	logsDir := store.Paths().LogsDir
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	svc := New(Options{
		Catalog: func() *db.DB { return d },
		Config:  func() *config.Store { return store },
		Version: "dev",
		Commit:  "abc",
		Built:   "now",
		Log:     slog.New(slog.DiscardHandler),
	})
	return svc, d, logsDir
}

func writeLog(t *testing.T, dir string, lines []string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "app.log"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func zipNames(t *testing.T, path string) []string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	names := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		names = append(names, f.Name)
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, rc)
		_ = rc.Close()
	}
	return names
}
