package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactSecretsAndHomePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	dataDir := filepath.Join(home, "AppData", "Local", "FlibustaHub")
	lib := filepath.Join("D:", "Books")
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: redact(Options{DataDir: dataDir, LibraryRoot: lib, HomeDir: home}),
	})
	log := slog.New(handler)
	log.Info("test",
		"apiKey", "secret-value",
		"token", "abc",
		"personal", filepath.Join(home, "Documents", "notes.txt"),
		"data", filepath.Join(dataDir, "logs", "app.log"),
	)
	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["apiKey"] != "[redacted]" || payload["token"] != "[redacted]" {
		t.Fatalf("secrets not redacted: %s", buf.String())
	}
	personal, _ := payload["personal"].(string)
	if !strings.HasPrefix(personal, "~") {
		t.Fatalf("home path should be shortened, got %q", personal)
	}
	data, _ := payload["data"].(string)
	if !strings.Contains(data, "app.log") || strings.HasPrefix(data, "~") {
		t.Fatalf("dataDir paths must stay, got %q", data)
	}
}

func TestSetupCreatesLogFile(t *testing.T) {
	dir := t.TempDir()
	logger, rotator, err := Setup(Options{LogsDir: dir, ToStdout: false, Level: slog.LevelInfo})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rotator.Close() })
	logger.Info("hello")
	if _, err := os.Stat(filepath.Join(dir, "app.log")); err != nil {
		t.Fatal(err)
	}
}
