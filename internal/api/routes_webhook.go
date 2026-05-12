package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildWebhookRouteDeps() *routes.WebhookDeps {
	return &routes.WebhookDeps{
		RequireAuth:    s.requireAuth,
		RequireScope:   middleware.RequireScope,
		ListWebhooks:   s.Webhook.Handler.ListWebhooks,
		GetWebhook:     s.Webhook.Handler.GetWebhook,
		ListDeliveries: s.Webhook.Handler.ListDeliveries,
		CreateWebhook:  s.Webhook.Handler.CreateWebhook,
		UpdateWebhook:  s.Webhook.Handler.UpdateWebhook,
		DeleteWebhook:  s.Webhook.Handler.DeleteWebhook,
		TestWebhook:    s.Webhook.Handler.TestWebhook,

		TenantMiddleware:   s.Middleware.Tenant,
		TenantDBMiddleware: s.Middleware.TenantDB,
	}
}
