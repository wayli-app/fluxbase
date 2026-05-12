package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildRESTRouteDeps() *routes.RESTDeps {
	return &routes.RESTDeps{
		RequireAuth:        s.requireAuth,
		RequireScope:       middleware.RequireScope,
		HandleTables:       s.rest.HandleDynamicTable,
		HandleQuery:        s.rest.HandleDynamicQuery,
		HandleById:         s.rest.HandleDynamicTableById,
		TenantMiddleware:   s.Middleware.Tenant,
		TenantDBMiddleware: s.Middleware.TenantDB,
	}
}
