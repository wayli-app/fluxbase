package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
)

func (s *Server) buildGraphQLRouteDeps() *routes.GraphQLDeps {
	if s.GraphQL.Handler == nil {
		return nil
	}
	return &routes.GraphQLDeps{
		OptionalAuth:     s.optionalAuth,
		HandleGraphQL:    s.GraphQL.Handler.HandleGraphQL,
		HandleIntrospect: s.GraphQL.Handler.HandleIntrospection,

		TenantMiddleware:   s.Middleware.Tenant,
		TenantDBMiddleware: s.Middleware.TenantDB,
	}
}
