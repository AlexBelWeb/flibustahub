package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/alexbelweb/flibustahub/internal/apperr"
)

var fileMagic = []byte{'F', 'H', 'S', 1}

func (s *Service) filePath() string {
	return filepath.Join(s.dir(), fileName)
}

func (s *Service) keyPath() string {
	return filepath.Join(s.dir(), keyFileName)
}

func deriveKey(material []byte) ([]byte, error) {
	return hkdf.Key(sha256.New, material, []byte("flibustahub"), "secrets-v1", 32)
}

func (s *Service) cryptoMaterial() ([]byte, error) {
	id, err := s.machineID()
	if err == nil && id != "" {
		return []byte(id), nil
	}
	return s.fileKeyMaterial()
}

func (s *Service) fileKeyMaterial() ([]byte, error) {
	path := s.keyPath()
	raw, err := os.ReadFile(path)
	if err == nil && len(raw) == 32 {
		return raw, nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	if err := atomicWrite(path, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func (s *Service) loadMap() (map[string]string, error) {
	path := s.filePath()
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	plain, err := s.decrypt(raw)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	if unmarshalErr := json.Unmarshal(plain, &out); unmarshalErr != nil {
		return nil, apperr.Wrap(apperr.CodeSecretStoreUnreadable, unmarshalErr, nil)
	}
	return out, nil
}

func (s *Service) saveMap(m map[string]string) error {
	path := s.filePath()
	if len(m) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
		}
		return nil
	}
	plain, err := json.Marshal(m)
	if err != nil {
		return apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	raw, err := s.encrypt(plain)
	if err != nil {
		return err
	}
	return atomicWrite(path, raw)
}

// lookupFile reports the file record. ok is true when the id is present,
// including a tombstone (empty value). A tombstone means "OS is stale".
func (s *Service) lookupFile(id string) (value string, ok bool, err error) {
	m, err := s.loadMap()
	if err != nil {
		return "", false, err
	}
	val, ok := m[id]
	return val, ok, nil
}

func (s *Service) putFile(id, secret string) error {
	m, err := s.loadMap()
	if err != nil {
		return err
	}
	m[id] = secret
	return s.saveMap(m)
}

func (s *Service) tombstoneFileID(id string) error {
	m, err := s.loadMap()
	if err != nil {
		return err
	}
	m[id] = ""
	return s.saveMap(m)
}

func (s *Service) deleteFileID(id string) error {
	m, err := s.loadMap()
	if err != nil {
		return err
	}
	if _, ok := m[id]; !ok {
		return nil
	}
	delete(m, id)
	return s.saveMap(m)
}

func (s *Service) encrypt(plain []byte) ([]byte, error) {
	material, err := s.cryptoMaterial()
	if err != nil {
		return nil, err
	}
	key, err := deriveKey(material)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	out := make([]byte, 0, len(fileMagic)+len(nonce)+len(plain)+gcm.Overhead())
	out = append(out, fileMagic...)
	out = append(out, nonce...)
	return append(out, gcm.Seal(nil, nonce, plain, fileMagic)...), nil
}

func (s *Service) decrypt(raw []byte) ([]byte, error) {
	if len(raw) < len(fileMagic)+12+16 {
		return nil, apperr.New(apperr.CodeSecretStoreUnreadable, nil)
	}
	if string(raw[:len(fileMagic)]) != string(fileMagic) {
		return nil, apperr.New(apperr.CodeSecretStoreUnreadable, nil)
	}
	material, err := s.cryptoMaterial()
	if err != nil {
		return nil, err
	}
	key, err := deriveKey(material)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	nonceSize := gcm.NonceSize()
	rest := raw[len(fileMagic):]
	if len(rest) < nonceSize {
		return nil, apperr.New(apperr.CodeSecretStoreUnreadable, nil)
	}
	nonce, cipherText := rest[:nonceSize], rest[nonceSize:]
	plain, err := gcm.Open(nil, nonce, cipherText, fileMagic)
	if err != nil {
		return nil, apperr.Wrap(apperr.CodeSecretStoreUnreadable, err, nil)
	}
	return plain, nil
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	tmp, err := os.CreateTemp(dir, "secrets-*.tmp")
	if err != nil {
		return apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	if err := tmp.Chmod(0o600); err != nil {
		cleanup()
		return apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return apperr.Wrap(apperr.CodeSecretStoreFailed, err, nil)
	}
	return nil
}
