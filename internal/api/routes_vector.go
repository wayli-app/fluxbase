package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
)

func (s *Server) buildVectorRouteDeps() *routes.VectorDeps {
	if s.AI.VectorHandler == nil {
		return nil
	}
	return &routes.VectorDeps{
		RequireAuth:        s.requireAuth,
		TenantMiddleware:   s.Middleware.Tenant,
		HandleCapabilities: s.AI.VectorHandler.HandleGetCapabilities,
		HandleEmbed:        s.AI.VectorHandler.HandleEmbed,
		HandleSearch:       s.AI.VectorHandler.HandleSearch,
	}
}
