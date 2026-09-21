// Package secrets stores API keys in the OS credential store, with an
// encrypted file in the data directory when that store is unavailable.
package secrets

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/zalando/go-keyring"
)

const (
	// KindOS means the secret lives in the OS credential store.
	KindOS = "os"
	// KindFile means the OS store is unavailable and the secret is in an encrypted file.
	KindFile = "file"

	serviceName        = "FlibustaHub"
	fileName           = "secrets.enc"
	keyFileName        = "secrets.key"
	probeAccount       = "__probe__"
	windowsCredBlobMax = 2560 // CRED_MAX_CREDENTIAL_BLOB_SIZE; go-keyring v0.2.8 checks this before CredWrite
)

// Status is the honest view of where secrets are stored. Values are never included.
type Status struct {
	Kind             string `json:"kind"`
	MachineIDMissing bool   `json:"machineIdMissing"`
	HasSecret        bool   `json:"hasSecret"`
}

type keyRing interface {
	Set(service, user, password string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

type osRing struct{}

func (osRing) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (osRing) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (osRing) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

// Options construct a Service. Ring and MachineID are test hooks.
type Options struct {
	DataDir   func() string
	Log       *slog.Logger
	MachineID func() (string, error)
	ring      keyRing
}

// Service stores secrets by provider id. It does not write config.json.
type Service struct {
	dataDir   func() string
	log       *slog.Logger
	machineID func() (string, error)
	ring      keyRing

	mu     sync.Mutex
	probed bool
	useOS  bool
}

func New(opt Options) *Service {
	s := &Service{
		dataDir:   opt.DataDir,
		log:       opt.Log,
		machineID: opt.MachineID,
		ring:      opt.ring,
	}
	if s.dataDir == nil {
		s.dataDir = func() string { return "" }
	}
	if s.log == nil {
		s.log = slog.Default()
	}
	if s.machineID == nil {
		s.machineID = nativeMachineID
	}
	if s.ring == nil {
		s.ring = osRing{}
	}
	return s
}

func knownID(id string) bool {
	switch id {
	case "gemini", "openai", "ollama":
		return true
	default:
		return false
	}
}

// ValidProvider is empty (cleared) or a known provider code.
func ValidProvider(id string) bool {
	return id == "" || knownID(id)
}

func (s *Service) dir() string {
	if s.dataDir == nil {
		return ""
	}
	return s.dataDir()
}

func (s *Service) kindLocked() string {
	if s.probed {
		if s.useOS {
			return KindOS
		}
		return KindFile
	}
	s.probed = true
	_, err := s.ring.Get(serviceName, probeAccount)
	if err == nil || errors.Is(err, keyring.ErrNotFound) {
		s.useOS = true
		s.log.Info("secret store ready", "backend", KindOS)
		return KindOS
	}
	s.useOS = false
	s.log.Info("secret store ready", "backend", KindFile)
	return KindFile
}

func (s *Service) machineIDMissing() bool {
	_, err := s.machineID()
	return err != nil
}

// Status reports the active backend and whether id has a stored secret.
// An empty id reports the backend only. A damaged file is an error, not a crash.
func (s *Service) Status(id string) (Status, error) {
	if id != "" && !knownID(id) {
		return Status{}, apperr.New(apperr.CodeSecretInvalidID, map[string]string{"id": id})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	kind := s.kindLocked()
	st := Status{Kind: kind}
	if kind == KindFile {
		st.MachineIDMissing = s.machineIDMissing()
	}
	if id == "" {
		return st, nil
	}
	_, ok, err := s.getLocked(id)
	if err != nil {
		return st, err
	}
	st.HasSecret = ok
	return st, nil
}

// Set stores secret under id. The value is never written to logs or config.json.
func (s *Service) Set(id, secret string) error {
	if !knownID(id) {
		return apperr.New(apperr.CodeSecretInvalidID, map[string]string{"id": id})
	}
	if secret == "" {
		return apperr.New(apperr.CodeSecretEmpty, nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	kind := s.kindLocked()
	if kind == KindOS {
		if len(secret) > windowsCredBlobMax {
			return apperr.New(apperr.CodeSecretTooLarge, map[string]string{"max": "2560"})
		}
		if err := s.ring.Set(serviceName, id, secret); err != nil {
			return mapRingErr(err)
		}
		_ = s.deleteFileID(id)
		s.log.Info("secret stored", "backend", KindOS)
		return nil
	}
	if err := s.putFile(id, secret); err != nil {
		return err
	}
	s.log.Info("secret stored", "backend", KindFile)
	return nil
}

// Get returns the secret for id. Missing is ("", nil).
func (s *Service) Get(id string) (string, error) {
	if !knownID(id) {
		return "", apperr.New(apperr.CodeSecretInvalidID, map[string]string{"id": id})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kindLocked()
	val, _, err := s.getLocked(id)
	return val, err
}

// Delete removes the secret for id from both backends. Missing is success.
// A damaged file is removed so the user can store a new key.
func (s *Service) Delete(id string) error {
	if !knownID(id) {
		return apperr.New(apperr.CodeSecretInvalidID, map[string]string{"id": id})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	kind := s.kindLocked()
	if kind == KindOS {
		if err := s.ring.Delete(serviceName, id); err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return mapRingErr(err)
		}
		if err := s.clearFileRecord(id); err != nil {
			return err
		}
		s.log.Info("secret deleted", "backend", KindOS)
		return nil
	}
	if err := s.tombstoneFileID(id); err != nil {
		if apperr.As(err).Code == apperr.CodeSecretStoreUnreadable {
			if rmErr := os.Remove(filepath.Join(s.dir(), fileName)); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
				return apperr.Wrap(apperr.CodeSecretStoreFailed, rmErr, nil)
			}
		} else {
			return err
		}
	}
	s.log.Info("secret deleted", "backend", KindFile)
	return nil
}

func (s *Service) clearFileRecord(id string) error {
	if err := s.deleteFileID(id); err != nil {
		if apperr.As(err).Code == apperr.CodeSecretStoreUnreadable {
			if rmErr := os.Remove(filepath.Join(s.dir(), fileName)); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
				return apperr.Wrap(apperr.CodeSecretStoreFailed, rmErr, nil)
			}
			return nil
		}
		return err
	}
	return nil
}

func (s *Service) getLocked(id string) (string, bool, error) {
	fileVal, fileOK, err := s.lookupFile(id)
	if err != nil {
		return "", false, err
	}
	if fileOK {
		if !s.useOS {
			return fileVal, fileVal != "", nil
		}
		if fileVal == "" {
			if delErr := s.ring.Delete(serviceName, id); delErr != nil && !errors.Is(delErr, keyring.ErrNotFound) {
				return "", false, mapRingErr(delErr)
			}
			if err := s.deleteFileID(id); err != nil {
				return "", false, err
			}
			s.log.Info("secret deleted", "backend", KindOS)
			return "", false, nil
		}
		if promoErr := s.promoteToOS(id, fileVal); promoErr != nil {
			return fileVal, true, nil
		}
		return fileVal, true, nil
	}
	if !s.useOS {
		return "", false, nil
	}
	val, err := s.ring.Get(serviceName, id)
	if err == nil {
		return val, true, nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return "", false, nil
	}
	return "", false, mapRingErr(err)
}

func (s *Service) promoteToOS(id, secret string) error {
	if len(secret) > windowsCredBlobMax {
		return apperr.New(apperr.CodeSecretTooLarge, map[string]string{"max": "2560"})
	}
	if err := s.ring.Set(serviceName, id, secret); err != nil {
		return mapRingErr(err)
	}
	if err := s.deleteFileID(id); err != nil {
		return err
	}
	s.log.Info("secret stored", "backend", KindOS)
	return nil
}

func mapRingErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, keyring.ErrSetDataTooBig) {
		return apperr.New(apperr.CodeSecretTooLarge, map[string]string{"max": "2560"})
	}
	return apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
}
