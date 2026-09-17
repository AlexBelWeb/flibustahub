// Package logging configures slog with rotating files and optional stdout.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Options control the logger destination.
type Options struct {
	LogsDir     string
	ToStdout    bool
	Level       slog.Level
	LibraryRoot string
	DataDir     string
	HomeDir     string
}

// Setup installs the default slog logger and returns the rotating file for tests to close.
func Setup(opt Options) (*slog.Logger, *lumberjack.Logger, error) {
	if opt.LogsDir == "" {
		return slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil
	}
	if err := os.MkdirAll(opt.LogsDir, 0o755); err != nil {
		return nil, nil, err
	}
	rotator := &lumberjack.Logger{
		Filename:   filepath.Join(opt.LogsDir, "app.log"),
		MaxSize:    5,
		MaxBackups: 4,
		MaxAge:     0,
		Compress:   false,
	}
	var w io.Writer = rotator
	if opt.ToStdout {
		w = io.MultiWriter(rotator, os.Stdout)
	}
	level := opt.Level
	if level == 0 {
		level = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: redact(opt),
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger, rotator, nil
}

func redact(opt Options) func([]string, slog.Attr) slog.Attr {
	home := opt.HomeDir
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return func(_ []string, attr slog.Attr) slog.Attr {
		key := strings.ToLower(attr.Key)
		if key == "apikey" || key == "api_key" || key == "password" || key == "token" || key == "authorization" {
			return slog.String(attr.Key, "[redacted]")
		}
		if attr.Value.Kind() != slog.KindString {
			return attr
		}
		val := attr.Value.String()
		if looksLikeSecret(val) {
			return slog.String(attr.Key, "[redacted]")
		}
		if home != "" && strings.HasPrefix(val, home) {
			if opt.LibraryRoot != "" && strings.HasPrefix(val, opt.LibraryRoot) {
				return attr
			}
			if opt.DataDir != "" && strings.HasPrefix(val, opt.DataDir) {
				return attr
			}
			return slog.String(attr.Key, "~"+strings.TrimPrefix(val, home))
		}
		return attr
	}
}

func looksLikeSecret(v string) bool {
	l := strings.ToLower(v)
	return strings.HasPrefix(l, "sk-") || strings.HasPrefix(l, "ya29.") || strings.Contains(l, "bearer ")
}
