package secrets

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/zalando/go-keyring"
)

type errRing struct{ err error }

func (e errRing) Set(_, _, _ string) error        { return e.err }
func (e errRing) Get(_, _ string) (string, error) { return "", e.err }
func (e errRing) Delete(_, _ string) error        { return e.err }

type probeRing struct {
	sets int
}

func (p *probeRing) Get(_, _ string) (string, error) { return "", keyring.ErrNotFound }
func (p *probeRing) Set(_, _, _ string) error        { p.sets++; return nil }
func (p *probeRing) Delete(_, _ string) error        { return keyring.ErrNotFound }

func fileSvc(t *testing.T, machine string) *Service {
	t.Helper()
	dir := t.TempDir()
	return New(Options{
		DataDir: func() string { return dir },
		Log:     slog.New(slog.DiscardHandler),
		MachineID: func() (string, error) {
			if machine == "" {
				return "", errors.New("no machine id")
			}
			return machine, nil
		},
		ring: errRing{err: errors.New("no secret service")},
	})
}

func TestFileRoundTrip(t *testing.T) {
	s := fileSvc(t, "machine-a")
	const secret = "sk-test-secret-value-abcdef"
	if err := s.Set("gemini", secret); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("gemini")
	if err != nil {
		t.Fatal(err)
	}
	if got != secret {
		t.Fatalf("got %q", got)
	}
	st, err := s.Status("gemini")
	if err != nil {
		t.Fatal(err)
	}
	if st.Kind != KindFile || !st.HasSecret || st.MachineIDMissing {
		t.Fatalf("status = %+v", st)
	}
	if err := s.Delete("gemini"); err != nil {
		t.Fatal(err)
	}
	got, err = s.Get("gemini")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatal("secret still present after delete")
	}
	m, err := s.loadMap()
	if err != nil {
		t.Fatal(err)
	}
	val, ok := m["gemini"]
	if !ok {
		t.Fatal("file-mode delete must leave a tombstone")
	}
	if val != "" {
		t.Fatalf("tombstone value = %q", val)
	}
}

func TestNewDoesNotFailOnCorruptFile(t *testing.T) {
	s := fileSvc(t, "machine-a")
	if err := os.WriteFile(s.filePath(), []byte("not-a-secrets-file"), 0o600); err != nil {
		t.Fatal(err)
	}
	s2 := New(Options{
		DataDir:   s.dataDir,
		Log:       slog.New(slog.DiscardHandler),
		MachineID: s.machineID,
		ring:      errRing{err: errors.New("no secret service")},
	})
	if s2 == nil {
		t.Fatal("New must succeed")
	}
	_, err := s2.Get("openai")
	if apperr.As(err).Code != apperr.CodeSecretStoreUnreadable {
		t.Fatalf("Get code = %v", err)
	}
}

func TestWrongMachineKeyDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	mk := func(id string) *Service {
		return New(Options{
			DataDir:   func() string { return dir },
			Log:       slog.New(slog.DiscardHandler),
			MachineID: func() (string, error) { return id, nil },
			ring:      errRing{err: errors.New("no secret service")},
		})
	}
	if err := mk("machine-a").Set("openai", "sk-one"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("panic: %v", rec)
		}
	}()
	_, err := mk("machine-b").Get("openai")
	if apperr.As(err).Code != apperr.CodeSecretStoreUnreadable {
		t.Fatalf("Get code = %v", err)
	}
}

func TestFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX 0600 is not a Windows file mode")
	}
	s := fileSvc(t, "machine-a")
	if err := s.Set("ollama", "local-key"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(s.filePath())
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("perm = %o, want 0600", perm)
	}
}

func TestSecretNotLogged(t *testing.T) {
	dir := t.TempDir()
	const secret = "sk-this-is-a-test-key-must-not-appear"
	var buf bytes.Buffer
	s := New(Options{
		DataDir:   func() string { return dir },
		Log:       slog.New(slog.NewJSONHandler(&buf, nil)),
		MachineID: func() (string, error) { return "machine-a", nil },
		ring:      errRing{err: errors.New("no secret service")},
	})
	if err := s.Set("gemini", secret); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("gemini"); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("gemini"); err != nil {
		t.Fatal(err)
	}
	logged := buf.String()
	if strings.Contains(logged, secret) {
		t.Fatalf("secret leaked into log: %s", logged)
	}
	var sawBackend bool
	for _, line := range bytes.Split(buf.Bytes(), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(line, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["backend"] == KindFile {
			sawBackend = true
		}
	}
	if !sawBackend {
		t.Fatalf("expected backend=%s in log, got %s", KindFile, logged)
	}
}

func TestGeneratedFileKeyWhenMachineIDMissing(t *testing.T) {
	s := fileSvc(t, "")
	if err := s.Set("gemini", "sk-generated-backend"); err != nil {
		t.Fatal(err)
	}
	st, err := s.Status("gemini")
	if err != nil {
		t.Fatal(err)
	}
	if st.Kind != KindFile || !st.MachineIDMissing || !st.HasSecret {
		t.Fatalf("status = %+v", st)
	}
	info, err := os.Stat(s.keyPath())
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 32 {
		t.Fatalf("key file size = %d", info.Size())
	}
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Fatalf("key perm = %o, want 0600", perm)
		}
	}
	got, err := s.Get("gemini")
	if err != nil || got != "sk-generated-backend" {
		t.Fatalf("get = %q err=%v", got, err)
	}
}

func TestTooLargeRejectedBeforeOSWrite(t *testing.T) {
	ring := &probeRing{}
	s := New(Options{
		DataDir:   func() string { return t.TempDir() },
		Log:       slog.New(slog.DiscardHandler),
		MachineID: func() (string, error) { return "machine-a", nil },
		ring:      ring,
	})
	err := s.Set("gemini", strings.Repeat("a", windowsCredBlobMax+1))
	if apperr.As(err).Code != apperr.CodeSecretTooLarge {
		t.Fatalf("code = %v", err)
	}
	if ring.sets != 0 {
		t.Fatalf("OS Set called %d times", ring.sets)
	}
}

func TestSetDoesNotWriteConfig(t *testing.T) {
	s := fileSvc(t, "machine-a")
	cfg := filepath.Join(s.dir(), "config.json")
	original := []byte("{\n  \"aiProvider\": \"gemini\"\n}\n")
	if err := os.WriteFile(cfg, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("gemini", "sk-not-for-config"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(original) {
		t.Fatalf("config.json changed: %s", raw)
	}
	if bytes.Contains(raw, []byte("sk-not-for-config")) {
		t.Fatal("secret written to config.json")
	}
}

func TestInvalidID(t *testing.T) {
	s := fileSvc(t, "machine-a")
	if apperr.As(s.Set("Gemini", "x")).Code != apperr.CodeSecretInvalidID {
		t.Fatal("expected invalid id")
	}
	if apperr.As(s.Set("gemini", "")).Code != apperr.CodeSecretEmpty {
		t.Fatal("expected empty secret")
	}
}

type memRing struct {
	data map[string]string
}

func (m *memRing) slot(service, user string) string { return service + "\x00" + user }

func (m *memRing) Get(service, user string) (string, error) {
	if m.data == nil {
		return "", keyring.ErrNotFound
	}
	v, ok := m.data[m.slot(service, user)]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return v, nil
}

func (m *memRing) Set(service, user, password string) error {
	if m.data == nil {
		m.data = map[string]string{}
	}
	m.data[m.slot(service, user)] = password
	return nil
}

func (m *memRing) Delete(service, user string) error {
	if m.data == nil {
		return keyring.ErrNotFound
	}
	k := m.slot(service, user)
	if _, ok := m.data[k]; !ok {
		return keyring.ErrNotFound
	}
	delete(m.data, k)
	return nil
}

func testSvc(dir, machine string, ring keyRing) *Service {
	return New(Options{
		DataDir: func() string { return dir },
		Log:     slog.New(slog.DiscardHandler),
		MachineID: func() (string, error) {
			return machine, nil
		},
		ring: ring,
	})
}

func TestFileUpdateWinsWhenOSReturns(t *testing.T) {
	dir := t.TempDir()
	osStore := &memRing{}
	if err := osStore.Set(serviceName, "gemini", "sk-old-from-os"); err != nil {
		t.Fatal(err)
	}
	file := testSvc(dir, "machine-a", errRing{err: errors.New("no secret service")})
	if err := file.Set("gemini", "sk-new-from-file"); err != nil {
		t.Fatal(err)
	}
	online := testSvc(dir, "machine-a", osStore)
	got, err := online.Get("gemini")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sk-new-from-file" {
		t.Fatalf("got %q, want the file value", got)
	}
	m, err := online.loadMap()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m["gemini"]; ok {
		t.Fatal("file record must be cleared after promotion")
	}
	stored, err := osStore.Get(serviceName, "gemini")
	if err != nil {
		t.Fatal(err)
	}
	if stored != "sk-new-from-file" {
		t.Fatalf("OS store = %q", stored)
	}
}

func TestFileTombstoneDeletesStaleOS(t *testing.T) {
	dir := t.TempDir()
	osStore := &memRing{}
	if err := testSvc(dir, "machine-a", osStore).Set("gemini", "sk-should-vanish"); err != nil {
		t.Fatal(err)
	}
	offline := testSvc(dir, "machine-a", errRing{err: errors.New("no secret service")})
	if err := offline.Delete("gemini"); err != nil {
		t.Fatal(err)
	}
	online := testSvc(dir, "machine-a", osStore)
	st, err := online.Status("gemini")
	if err != nil {
		t.Fatal(err)
	}
	if st.HasSecret {
		t.Fatalf("status = %+v, want no secret", st)
	}
	_, err = osStore.Get(serviceName, "gemini")
	if !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("OS record still present: %v", err)
	}
	m, err := online.loadMap()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m["gemini"]; ok {
		t.Fatal("tombstone must be removed after OS reconcile")
	}
}
