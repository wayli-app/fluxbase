package api

import (
	"context"

	"github.com/nimbleflux/fluxbase/internal/secrets"
)

type SecretsModule struct {
	Storage *secrets.Storage
	Handler *secrets.Handler
}

func (m *SecretsModule) Name() string { return "secrets" }

func (m *SecretsModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	m.Storage = secrets.NewStorage(registry.DB, registry.Config.EncryptionKeyBytes)
	m.Handler = secrets.NewHandler(m.Storage)
	registry.Register(m.Storage)
	return nil
}
