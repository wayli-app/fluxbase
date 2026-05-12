package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
)

func (s *Server) buildOpenAPIRouteDeps() *routes.OpenAPIDeps {
	return &routes.OpenAPIDeps{
		OptionalAuth:    s.optionalAuth,
		TenantContext:   s.Middleware.Tenant,
		TenantDBContext: s.Middleware.TenantDB,
		GetOpenAPISpec:  NewOpenAPIHandler(s.db).GetOpenAPISpec,
	}
}
