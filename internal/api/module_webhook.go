package api

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/webhook"
)

type WebhookModule struct {
	Handler *WebhookHandler
	Trigger *webhook.TriggerService
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

	m.Trigger = webhookTriggerService
	m.Handler = NewWebhookHandler(webhookService)
	registry.Register(webhookTriggerService)
	registry.Register(m.Handler)
	return nil
}
