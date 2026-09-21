package app

import (
	"github.com/alexbelweb/flibustahub/internal/apperr"
	"github.com/alexbelweb/flibustahub/internal/config"
	"github.com/alexbelweb/flibustahub/internal/services/secrets"
)

func (s *Service) secretsSvc() *secrets.Service {
	if s == nil {
		return nil
	}
	return s.secrets
}

func (s *Service) SecretStatus(id string) (secrets.Status, error) {
	sec := s.secretsSvc()
	if sec == nil {
		return secrets.Status{}, apperr.New(apperr.CodeSecretStoreFailed, nil)
	}
	return sec.Status(id)
}

func (s *Service) SetSecret(id, secret string) error {
	sec := s.secretsSvc()
	if sec == nil {
		return apperr.New(apperr.CodeSecretStoreFailed, nil)
	}
	return sec.Set(id, secret)
}

func (s *Service) DeleteSecret(id string) error {
	sec := s.secretsSvc()
	if sec == nil {
		return apperr.New(apperr.CodeSecretStoreFailed, nil)
	}
	return sec.Delete(id)
}

func (s *Service) SetAIProvider(id string) error {
	if !secrets.ValidProvider(id) {
		return apperr.New(apperr.CodeInvalidAIProvider, map[string]string{"id": id})
	}
	return s.config().Update(func(f *config.File) { f.AIProvider = id })
}
