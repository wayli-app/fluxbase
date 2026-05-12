package api

import (
	"context"

	"github.com/nimbleflux/fluxbase/internal/email"
)

type EmailModule struct {
	LazyService *email.LazyService
}

func (m *EmailModule) Name() string { return "email" }

func (m *EmailModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	m.LazyService = email.NewLazyService()
	registry.Register(m.LazyService)
	return nil
}
