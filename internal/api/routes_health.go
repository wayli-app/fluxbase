package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
)

func (s *Server) buildHealthRouteDeps() *routes.HealthDeps {
	return &routes.HealthDeps{
		Handler:      s.handleHealth,
		OptionalAuth: s.optionalAuth,
	}
}
