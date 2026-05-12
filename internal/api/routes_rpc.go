package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildRPCRouteDeps() *routes.RPCDeps {
	if s.RPC.Handler == nil {
		return nil
	}
	return &routes.RPCDeps{
		RequireRPCEnabled: middleware.RequireRPCEnabled(s.Auth.Handler.authService.GetSettingsCache()),
		OptionalAuth:      s.optionalAuth,
		RequireScope:      middleware.RequireScope,
		ListProcedures:    s.RPC.Handler.ListPublicProcedures,
		Invoke:            s.RPC.Handler.Invoke,
		GetExecution:      s.RPC.Handler.GetPublicExecution,
		GetExecutionLogs:  s.RPC.Handler.GetPublicExecutionLogs,

		TenantMiddleware:   s.Middleware.Tenant,
		TenantDBMiddleware: s.Middleware.TenantDB,
	}
}
