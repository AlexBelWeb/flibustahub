// Package config loads and stores the application JSON config.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/alexbelweb/flibustahub/internal/apperr"
)

const (
	ThemeSystem = "system"
	ThemeDark   = "dark"
	ThemeLight  = "light"

	EffectsAuto     = "auto"
	EffectsFull     = "full"
	EffectsReduced  = "reduced"
	DefaultOPDSPort = 8787
	DefaultOPDSBind = "127.0.0.1"
)

// File is the on-disk config.json document.
type File struct {
	DataDir         string      `json:"dataDir,omitempty"`
	DBPath          string      `json:"dbPath,omitempty"`
	CoversDir       string      `json:"coversDir,omitempty"`
	LogsDir         string      `json:"logsDir,omitempty"`
	BackupsDir      string      `json:"backupsDir,omitempty"`
	DownloadsDir    string      `json:"downloadsDir,omitempty"`
	LibraryRoot     string      `json:"libraryRoot"`
	INPXPath        string      `json:"inpxPath"`
	OPDSEnabled     bool        `json:"opdsEnabled"`
	OPDSPort        int         `json:"opdsPort"`
	OPDSBindAddress string      `json:"opdsBindAddress"`
	Theme           string      `json:"theme"`
	Locale          string      `json:"locale"`
	VisualEffects   string      `json:"visualEffects"`
	Window          WindowState `json:"window"`
	AIProvider      string      `json:"aiProvider"`
}

// WindowState is restored on startup after checking it still fits a monitor.
type WindowState struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximised bool `json:"maximised"`
}

// Store holds the last saved document and the live view after env overlays.
type Store struct {
	mu   sync.Mutex
	path string
	disk File
	live File
	log  *slog.Logger
}

// Paths are resolved locations used by the rest of the app.
type Paths struct {
	DataDir      string `json:"dataDir"`
	DBPath       string `json:"dbPath"`
	CoversDir    string `json:"coversDir"`
	LogsDir      string `json:"logsDir"`
	BackupsDir   string `json:"backupsDir"`
	DownloadsDir string `json:"downloadsDir"`
	ConfigPath   string `json:"configPath"`
}

func defaultDataDir() string {
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
		return filepath.Join(base, "FlibustaHub")
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "flibustahub")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "flibustahub")
}

func defaultDownloadsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Downloads")
}

// Defaults returns a config document with spec defaults filled in.
func Defaults(dataDir string) File {
	if dataDir == "" {
		dataDir = defaultDataDir()
	}
	return File{
		DataDir:         dataDir,
		DBPath:          filepath.Join(dataDir, "catalog.sqlite"),
		CoversDir:       filepath.Join(dataDir, "covers"),
		LogsDir:         filepath.Join(dataDir, "logs"),
		BackupsDir:      filepath.Join(dataDir, "backups"),
		DownloadsDir:    defaultDownloadsDir(),
		LibraryRoot:     "",
		INPXPath:        "",
		OPDSEnabled:     false,
		OPDSPort:        DefaultOPDSPort,
		OPDSBindAddress: DefaultOPDSBind,
		Theme:           ThemeSystem,
		Locale:          "",
		VisualEffects:   EffectsAuto,
		AIProvider:      "",
	}
}

func (f File) withDerivedPaths() File {
	dataDir := f.DataDir
	if dataDir == "" {
		dataDir = defaultDataDir()
	}
	if f.DBPath == "" {
		f.DBPath = filepath.Join(dataDir, "catalog.sqlite")
	}
	if f.CoversDir == "" {
		f.CoversDir = filepath.Join(dataDir, "covers")
	}
	if f.LogsDir == "" {
		f.LogsDir = filepath.Join(dataDir, "logs")
	}
	if f.BackupsDir == "" {
		f.BackupsDir = filepath.Join(dataDir, "backups")
	}
	if f.DownloadsDir == "" {
		f.DownloadsDir = defaultDownloadsDir()
	}
	if f.OPDSPort == 0 {
		f.OPDSPort = DefaultOPDSPort
	}
	if f.OPDSBindAddress == "" {
		f.OPDSBindAddress = DefaultOPDSBind
	}
	if f.Theme == "" {
		f.Theme = ThemeSystem
	}
	if f.VisualEffects == "" {
		f.VisualEffects = EffectsAuto
	}
	f.DataDir = dataDir
	return f
}

func applyEnv(f File) File {
	if v := os.Getenv("FLIBUSTAHUB_DATADIR"); v != "" {
		f.DataDir = v
	}
	if v := os.Getenv("FLIBUSTAHUB_DBPATH"); v != "" {
		f.DBPath = v
	}
	if v := os.Getenv("FLIBUSTAHUB_COVERSDIR"); v != "" {
		f.CoversDir = v
	}
	if v := os.Getenv("FLIBUSTAHUB_LOGSDIR"); v != "" {
		f.LogsDir = v
	}
	if v := os.Getenv("FLIBUSTAHUB_BACKUPSDIR"); v != "" {
		f.BackupsDir = v
	}
	if v := os.Getenv("FLIBUSTAHUB_DOWNLOADSDIR"); v != "" {
		f.DownloadsDir = v
	}
	if v := os.Getenv("FLIBUSTAHUB_LIBRARYROOT"); v != "" {
		f.LibraryRoot = v
	}
	if v := os.Getenv("FLIBUSTAHUB_INPXPATH"); v != "" {
		f.INPXPath = v
	}
	if v := os.Getenv("FLIBUSTAHUB_THEME"); v != "" {
		f.Theme = v
	}
	if v := os.Getenv("FLIBUSTAHUB_LOCALE"); v != "" {
		f.Locale = v
	}
	if v := os.Getenv("FLIBUSTAHUB_VISUALEFFECTS"); v != "" {
		f.VisualEffects = v
	}
	if v := os.Getenv("FLIBUSTAHUB_OPDSPORT"); v != "" {
		var port int
		if _, err := parsePort(v); err == nil {
			port, _ = parsePort(v)
			f.OPDSPort = port
		}
	}
	if v := os.Getenv("FLIBUSTAHUB_OPDSBINDADDRESS"); v != "" {
		f.OPDSBindAddress = v
	}
	return f.withDerivedPaths()
}

func parsePort(v string) (int, error) {
	var n int
	for _, r := range v {
		if r < '0' || r > '9' {
			return 0, errors.New("not a port")
		}
		n = n*10 + int(r-'0')
	}
	if n <= 0 || n > 65535 {
		return 0, errors.New("port out of range")
	}
	return n, nil
}

// Load reads config.json from dataDir, replacing a broken file with defaults.
// A Store is always returned so the UI can start; err is set when the file
// could not be used as-is.
func Load(dataDir string, log *slog.Logger) (*Store, error) {
	if dataDir == "" {
		dataDir = defaultDataDir()
	}
	if log == nil {
		log = slog.Default()
	}
	path := filepath.Join(dataDir, "config.json")
	disk := Defaults(dataDir)
	makeStore := func() *Store {
		return &Store{path: path, disk: disk, live: applyEnv(disk), log: log}
	}

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return makeStore(), apperr.Wrap(apperr.CodeConfigWriteFailed, err, nil)
	}

	raw, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return makeStore(), nil
	case err != nil:
		log.Error("config: cannot read file, using defaults", "err", err)
		return makeStore(), apperr.Wrap(apperr.CodeConfigUnreadable, err, nil)
	default:
		var parsed File
		if unmarshalErr := json.Unmarshal(raw, &parsed); unmarshalErr != nil {
			broken := path + ".broken"
			if renameErr := os.Rename(path, broken); renameErr != nil {
				log.Error("config: damaged file, rename failed", "err", renameErr)
			} else {
				log.Error("config: damaged file moved aside, using defaults", "broken", broken)
			}
		} else {
			if parsed.DataDir == "" {
				parsed.DataDir = dataDir
			}
			disk = mergeDefaults(parsed, dataDir)
		}
	}

	return makeStore(), nil
}

func mergeDefaults(parsed File, dataDir string) File {
	base := Defaults(dataDir)
	if parsed.DataDir != "" {
		base.DataDir = parsed.DataDir
	}
	if parsed.DBPath != "" {
		base.DBPath = parsed.DBPath
	}
	if parsed.CoversDir != "" {
		base.CoversDir = parsed.CoversDir
	}
	if parsed.LogsDir != "" {
		base.LogsDir = parsed.LogsDir
	}
	if parsed.BackupsDir != "" {
		base.BackupsDir = parsed.BackupsDir
	}
	if parsed.DownloadsDir != "" {
		base.DownloadsDir = parsed.DownloadsDir
	}
	base.LibraryRoot = parsed.LibraryRoot
	base.INPXPath = parsed.INPXPath
	base.OPDSEnabled = parsed.OPDSEnabled
	if parsed.OPDSPort != 0 {
		base.OPDSPort = parsed.OPDSPort
	}
	if parsed.OPDSBindAddress != "" {
		base.OPDSBindAddress = parsed.OPDSBindAddress
	}
	if parsed.Theme != "" {
		base.Theme = parsed.Theme
	}
	base.Locale = parsed.Locale
	if parsed.VisualEffects != "" {
		base.VisualEffects = parsed.VisualEffects
	}
	base.Window = parsed.Window
	base.AIProvider = parsed.AIProvider
	return base.withDerivedPaths()
}

func (s *Store) Path() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.path
}

func (s *Store) Live() File {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.live
}

func (s *Store) Paths() Paths {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Paths{
		DataDir:      s.live.DataDir,
		DBPath:       s.live.DBPath,
		CoversDir:    s.live.CoversDir,
		LogsDir:      s.live.LogsDir,
		BackupsDir:   s.live.BackupsDir,
		DownloadsDir: s.live.DownloadsDir,
		ConfigPath:   s.path,
	}
}

func (s *Store) Update(mutator func(*File)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	mutator(&s.disk)
	s.disk = s.disk.withDerivedPaths()
	s.live = applyEnv(s.disk)
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.CodeConfigWriteFailed, err, nil)
	}
	raw, err := json.MarshalIndent(s.disk, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.CodeConfigWriteFailed, err, nil)
	}
	tmp, err := os.CreateTemp(dir, "config-*.tmp")
	if err != nil {
		return apperr.Wrap(apperr.CodeConfigWriteFailed, err, nil)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return apperr.Wrap(apperr.CodeConfigWriteFailed, err, nil)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return apperr.Wrap(apperr.CodeConfigWriteFailed, err, nil)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		_ = os.Remove(tmpName)
		return apperr.Wrap(apperr.CodeConfigWriteFailed, err, nil)
	}
	return nil
}

// EnvOverrides reports keys present in the process environment.
func EnvOverrides() []string {
	var keys []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "FLIBUSTAHUB_") {
			keys = append(keys, strings.SplitN(kv, "=", 2)[0])
		}
	}
	return keys
}
