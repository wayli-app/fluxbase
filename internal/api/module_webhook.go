package api

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/webhook"
)

type WebhookModule struct {
	Handlers *WebhookHandlers
}

func (m *WebhookModule) Name() string { return "webhook" }

func (m *WebhookModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	webhookService := webhook.NewWebhookService(db)
	if cfg.Debug {
		webhookService.SetAllowPrivateIPs(true)
		log.Warn().Msg("SECURITY: Debug mode enabled - webhook SSRF protection is DISABLED. Do NOT use in production!")
	}
	webhookTriggerService := webhook.NewTriggerService(db, webhookService, 4)

	m.Handlers.Trigger = webhookTriggerService
	m.Handlers.Handler = NewWebhookHandler(webhookService)
	registry.Register(webhookTriggerService)
	return nil
}

func (m *WebhookModule) Shutdown(ctx context.Context) error {
	if m.Handlers.Trigger != nil {
		m.Handlers.Trigger.Stop()
	}
	return nil
}
