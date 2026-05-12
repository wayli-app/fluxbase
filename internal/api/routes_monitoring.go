package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildMonitoringRouteDeps() *routes.MonitoringDeps {
	return &routes.MonitoringDeps{
		RequireAuth:        s.requireAuth,
		RequireScope:       middleware.RequireScope,
		TenantMiddleware:   s.Middleware.Tenant,
		TenantDBMiddleware: s.Middleware.TenantDB,
		GetMetrics:         s.Monitoring.Handler.GetMetrics,
		GetHealth:          s.Monitoring.Handler.GetHealth,
		GetLogs:            s.Monitoring.Handler.GetLogs,
	}
}
