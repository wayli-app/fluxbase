package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildFunctionsRouteDeps() *routes.FunctionsDeps {
	if s.Functions.Handler == nil {
		return nil
	}
	return &routes.FunctionsDeps{
		RequireFunctionsEnabled: middleware.RequireFunctionsEnabled(s.Auth.Handler.authService.GetSettingsCache()),
		RequireAuth:             s.requireAuth,
		OptionalAuth:            s.optionalAuth,
		RequireScope:            middleware.RequireScope,
		TenantMiddleware:        s.Middleware.Tenant,
		ListFunctions:           s.Functions.Handler.ListFunctions,
		GetFunction:             s.Functions.Handler.GetFunction,
		CreateFunction:          s.Functions.Handler.CreateFunction,
		UpdateFunction:          s.Functions.Handler.UpdateFunction,
		DeleteFunction:          s.Functions.Handler.DeleteFunction,
		InvokeFunction:          s.Functions.Handler.InvokeFunction,
		GetExecutions:           s.Functions.Handler.GetExecutions,
		ListSharedModules:       s.Functions.Handler.ListSharedModules,
		GetSharedModule:         s.Functions.Handler.GetSharedModule,
		CreateSharedModule:      s.Functions.Handler.CreateSharedModule,
		UpdateSharedModule:      s.Functions.Handler.UpdateSharedModule,
		DeleteSharedModule:      s.Functions.Handler.DeleteSharedModule,
	}
}
