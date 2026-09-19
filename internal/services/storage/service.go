// Package storage tracks whether the library folder can be read.
package storage

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
	"github.com/alexbelweb/flibustahub/internal/db"
	"github.com/alexbelweb/flibustahub/internal/inpx"
	"github.com/alexbelweb/flibustahub/internal/platform"
	"github.com/alexbelweb/flibustahub/internal/services/downloads"
)

const defaultTTL = 3 * time.Second

// DumpOffer is a newer on-disk dump the UI may present; import is never auto-started.
type DumpOffer struct {
	Path           string `json:"path"`
	Name           string `json:"name"`
	FileVersion    string `json:"fileVersion,omitempty"`
	CatalogVersion string `json:"catalogVersion,omitempty"`
}

// Snapshot is the last known storage state. Empty LibraryRoot is onboarding, not offline.
type Snapshot struct {
	Configured  bool       `json:"configured"`
	Available   bool       `json:"available"`
	Unreachable bool       `json:"unreachable"`
	LibraryRoot string     `json:"libraryRoot,omitempty"`
	Remapped    bool       `json:"remapped,omitempty"`
	DumpOffer   *DumpOffer `json:"dumpOffer,omitempty"`
}

// Service probes libraryRoot with a short result cache and single-flight.
type Service struct {
	cfg       *config.Store
	catalog   func() *db.DB
	log       *slog.Logger
	now       func() time.Time
	onChange  func(Snapshot)
	ttl       time.Duration
	identify  func(string) (string, string)
	findVol   func(string, string) (string, bool, bool)
	stat      func(string) (os.FileInfo, error)
	openRoot  func(string) error
	mu        sync.Mutex
	busy      bool
	busyWait  sync.Cond
	have      bool
	checkedAt time.Time
	snap      Snapshot
	lastEmit  Snapshot
	emitted   bool
	dumpSeen  bool
}

func New(cfg *config.Store, catalog func() *db.DB, log *slog.Logger, onChange func(Snapshot)) *Service {
	if log == nil {
		log = slog.Default()
	}
	if catalog == nil {
		catalog = func() *db.DB { return nil }
	}
	s := &Service{
		cfg:      cfg,
		catalog:  catalog,
		log:      log,
		now:      time.Now,
		onChange: onChange,
		ttl:      defaultTTL,
		identify: platform.IdentifyVolume,
		findVol:  platform.FindVolumePath,
		stat:     os.Stat,
		openRoot: func(path string) error {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			return f.Close()
		},
	}
	s.busyWait.L = &s.mu
	return s
}

func (s *Service) SetOnChange(fn func(Snapshot)) {
	s.mu.Lock()
	s.onChange = fn
	s.mu.Unlock()
}

// Snapshot returns the last computed state without probing.
func (s *Service) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snap
}

// Check probes libraryRoot. force always talks to the OS; other callers reuse a few seconds of cache.
func (s *Service) Check(ctx context.Context, force bool) Snapshot {
	start := s.now()
	s.mu.Lock()
	for s.busy {
		if !force && s.freshLocked(start) {
			snap := s.snap
			s.mu.Unlock()
			return snap
		}
		s.busyWait.Wait()
		start = s.now()
	}
	if !force && s.freshLocked(start) {
		snap := s.snap
		s.mu.Unlock()
		return snap
	}
	s.busy = true
	s.mu.Unlock()

	snap := s.probe(ctx)

	s.mu.Lock()
	s.snap = snap
	s.have = true
	s.checkedAt = s.now()
	s.busy = false
	s.busyWait.Broadcast()
	s.mu.Unlock()

	elapsed := s.now().Sub(start)
	if elapsed >= time.Second {
		s.log.Info("storage check waited", "ms", elapsed.Milliseconds(), "available", snap.Available)
	}
	s.emitIfChanged(snap)
	if snap.Remapped {
		s.mu.Lock()
		s.snap.Remapped = false
		s.mu.Unlock()
	}
	return snap
}

func (s *Service) freshLocked(now time.Time) bool {
	return s.have && now.Sub(s.checkedAt) < s.ttl
}

// Probe returns a typed error when a library file cannot be read right now.
func (s *Service) Probe(ctx context.Context) error {
	return errorOf(s.Check(ctx, false))
}

// ProbeForce is Probe after a live OS check.
func (s *Service) ProbeForce(ctx context.Context) error {
	return errorOf(s.Check(ctx, true))
}

func errorOf(snap Snapshot) error {
	if !snap.Configured {
		return apperr.New(apperr.CodeLibraryOffline, nil)
	}
	if snap.Unreachable {
		return apperr.New(apperr.CodeLibraryUnreachable, nil)
	}
	if !snap.Available {
		return apperr.New(apperr.CodeLibraryOffline, nil)
	}
	return nil
}

func (s *Service) probe(ctx context.Context) Snapshot {
	root := ""
	if s.cfg != nil {
		root = strings.TrimSpace(s.cfg.Live().LibraryRoot)
	}
	out := Snapshot{LibraryRoot: root, Configured: root != ""}
	if !out.Configured {
		return out
	}
	state := s.inspect(root)
	switch state {
	case rootOK:
		out.Available = true
	case rootUnreachable:
		out.Unreachable = true
	default:
		if mapped, remapped, ok := s.tryRemap(ctx, root); ok {
			out.LibraryRoot = mapped
			out.Available = true
			out.Remapped = remapped
		}
	}
	if out.Available {
		out.DumpOffer = s.maybeDump(ctx, out.LibraryRoot)
	}
	return out
}

type rootState int

const (
	rootMissing rootState = iota
	rootUnreachable
	rootOK
)

func (s *Service) inspect(root string) rootState {
	st, err := s.stat(root)
	if err != nil {
		if isNotExist(err) {
			return rootMissing
		}
		return rootUnreachable
	}
	if !st.IsDir() {
		return rootMissing
	}
	if err := s.openRoot(root); err != nil {
		return rootUnreachable
	}
	return rootOK
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err)
}

func (s *Service) tryRemap(ctx context.Context, oldRoot string) (string, bool, bool) {
	catalog := s.catalog()
	if catalog == nil {
		return "", false, false
	}
	id, err := db.Meta(ctx, catalog.Read, db.MetaLibraryVolume)
	if err != nil || strings.TrimSpace(id) == "" {
		return "", false, false
	}
	rel, _ := db.Meta(ctx, catalog.Read, db.MetaLibraryRel)
	path, ok, ambiguous := s.findVol(id, rel)
	if ambiguous {
		s.log.Warn("library volume label is ambiguous; not remapping", "id", id)
		return "", false, false
	}
	if !ok {
		return "", false, false
	}
	path = filepath.Clean(path)
	if s.inspect(path) != rootOK {
		return "", false, false
	}
	if samePath(path, oldRoot) {
		return path, false, true
	}
	if s.cfg != nil {
		if err := s.cfg.Update(func(f *config.File) { f.LibraryRoot = path }); err != nil {
			s.log.Warn("library root remap not saved", "err", err)
		}
	}
	s.log.Info("library root remapped by volume id", "from", oldRoot, "to", path, "id", id)
	s.persistVolume(ctx, path)
	return path, true, true
}

func samePath(a, b string) bool {
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return filepath.Clean(aa) == filepath.Clean(bb)
}

// PersistVolume stores the removable-volume identity for the current library root.
func (s *Service) PersistVolume(ctx context.Context, root string) {
	s.persistVolume(ctx, root)
}

func (s *Service) persistVolume(ctx context.Context, root string) {
	root = strings.TrimSpace(root)
	if root == "" {
		return
	}
	catalog := s.catalog()
	if catalog == nil || catalog.Write == nil {
		return
	}
	id, rel := s.identify(root)
	if id == "" {
		return
	}
	if err := db.SetMeta(ctx, catalog.Write, db.MetaLibraryVolume, id); err != nil {
		s.log.Warn("library volume id not saved", "err", err)
		return
	}
	if err := db.SetMeta(ctx, catalog.Write, db.MetaLibraryRel, rel); err != nil {
		s.log.Warn("library volume rel not saved", "err", err)
	}
}

func (s *Service) maybeDump(ctx context.Context, root string) *DumpOffer {
	if s.catalog() == nil {
		return nil
	}
	s.mu.Lock()
	first := !s.dumpSeen
	became := !s.have || !s.snap.Available
	if !first && !became {
		offer := s.snap.DumpOffer
		s.mu.Unlock()
		return offer
	}
	s.dumpSeen = true
	s.mu.Unlock()
	return s.scanDump(ctx, root)
}

func (s *Service) scanDump(ctx context.Context, root string) *DumpOffer {
	catalog := s.catalog()
	if catalog == nil {
		return nil
	}
	path, err := inpx.FindINPX(root, "")
	if err != nil {
		return nil
	}
	st, err := s.stat(path)
	if err != nil || st.IsDir() {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	meta, peekErr := inpx.PeekMeta(f, st.Size())
	_ = f.Close()
	if peekErr != nil {
		s.log.Warn("inpx peek failed", "path", path, "err", peekErr)
		return nil
	}
	catalogVer, err := db.INPXVersion(ctx, catalog.Read)
	if err != nil {
		return nil
	}
	importedAt := time.Time{}
	if raw, metaErr := db.Meta(ctx, catalog.Read, db.MetaINPXImportedAt); metaErr == nil && raw != "" {
		importedAt, _ = time.Parse(time.RFC3339, raw)
	}
	dismissed := ""
	if s.cfg != nil {
		dismissed = s.cfg.Live().DismissedINPXVersion
	}
	if !downloads.DumpNewer(meta.Version, catalogVer, dismissed, st.ModTime(), importedAt) {
		return nil
	}
	return &DumpOffer{
		Path:           path,
		Name:           filepath.Base(path),
		FileVersion:    meta.Version,
		CatalogVersion: catalogVer,
	}
}

// DismissDumpOffer remembers the offered version so the banner stays gone until a newer dump appears.
func (s *Service) DismissDumpOffer() error {
	s.mu.Lock()
	offer := s.snap.DumpOffer
	s.mu.Unlock()
	if offer == nil || s.cfg == nil {
		return nil
	}
	ver := strings.TrimSpace(offer.FileVersion)
	if ver == "" {
		ver = offer.Name
	}
	if err := s.cfg.Update(func(f *config.File) { f.DismissedINPXVersion = ver }); err != nil {
		return err
	}
	s.mu.Lock()
	s.snap.DumpOffer = nil
	snap := s.snap
	s.mu.Unlock()
	s.emitIfChanged(snap)
	return nil
}

func (s *Service) emitIfChanged(snap Snapshot) {
	s.mu.Lock()
	changed := !s.emitted || !sameSnap(s.lastEmit, snap)
	if changed {
		s.lastEmit = snap
		s.emitted = true
	}
	fn := s.onChange
	s.mu.Unlock()
	if changed && fn != nil {
		fn(snap)
	}
}

func sameSnap(a, b Snapshot) bool {
	if a.Configured != b.Configured || a.Available != b.Available || a.Unreachable != b.Unreachable {
		return false
	}
	if a.LibraryRoot != b.LibraryRoot || a.Remapped != b.Remapped {
		return false
	}
	if (a.DumpOffer == nil) != (b.DumpOffer == nil) {
		return false
	}
	if a.DumpOffer != nil && (a.DumpOffer.Path != b.DumpOffer.Path || a.DumpOffer.FileVersion != b.DumpOffer.FileVersion) {
		return false
	}
	return true
}
