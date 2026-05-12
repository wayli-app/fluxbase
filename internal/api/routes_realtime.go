package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildRealtimeRouteDeps() *routes.RealtimeDeps {
	return &routes.RealtimeDeps{
		RequireRealtimeEnabled: middleware.RequireRealtimeEnabled(s.Auth.Handler.authService.GetSettingsCache()),
		OptionalAuth:           s.optionalAuth,
		RequireAuth:            s.requireAuth,
		RequireScope:           middleware.RequireScope,
		TenantMiddleware:       s.Middleware.Tenant,
		HandleWebSocket:        s.Realtime.Handler.HandleWebSocket,
		HandleStats:            s.handleRealtimeStats,
	}
}
