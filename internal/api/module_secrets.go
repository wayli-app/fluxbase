package api

import (
	"context"

	"github.com/nimbleflux/fluxbase/internal/secrets"
)

type SecretsModule struct {
	Handlers *SecretsHandlers
}

func (m *SecretsModule) Name() string { return "secrets" }

func (m *SecretsModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	m.Handlers = &SecretsHandlers{}
	m.Handlers.Storage = secrets.NewStorage(registry.DB, registry.Config.EncryptionKeyBytes)
	m.Handlers.Handler = secrets.NewHandler(m.Handlers.Storage)
	registry.Register(m.Handlers.Storage)
	return nil
}
