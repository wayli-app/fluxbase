package api

import (
	"context"

	"github.com/nimbleflux/fluxbase/internal/extensions"
)

type ExtensionsModule struct {
	Handler *extensions.Handler
}

func (m *ExtensionsModule) Name() string { return "extensions" }

func (m *ExtensionsModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	m.Handler = extensions.NewHandler(extensions.NewService(registry.DB))
	return nil
}
