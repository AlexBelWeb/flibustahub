package handlers

import (
	"github.com/alexbelweb/flibustahub/internal/services/secrets"
)

func (a *App) SecretStatus(id string) (secrets.Status, error) {
	return a.svc.SecretStatus(id)
}

func (a *App) SetSecret(id, secret string) error {
	return a.svc.SetSecret(id, secret)
}

func (a *App) DeleteSecret(id string) error {
	return a.svc.DeleteSecret(id)
}

func (a *App) SetAIProvider(id string) error {
	return a.svc.SetAIProvider(id)
}
