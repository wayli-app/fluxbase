package api

import (
	"context"

	"github.com/nimbleflux/fluxbase/internal/email"
)

type EmailModule struct {
	Manager *email.Manager
	Service email.Service
}

func (m *EmailModule) Name() string { return "email" }

func (m *EmailModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	m.Manager = email.NewManager(&registry.Config.Email, nil, nil, registry.Config)
	m.Service = m.Manager.WrapAsService()
	registry.Register(m.Manager)
	return nil
}
