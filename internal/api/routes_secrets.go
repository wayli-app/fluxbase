package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildSecretsRouteDeps() *routes.SecretsDeps {
	if s.Secrets.Handler == nil {
		return nil
	}
	return &routes.SecretsDeps{
		RequireAuth:        s.requireAuth,
		RequireScope:       middleware.RequireScope,
		ListSecrets:        s.Secrets.Handler.ListSecrets,
		GetStats:           s.Secrets.Handler.GetStats,
		GetSecretByName:    s.Secrets.Handler.GetSecretByName,
		GetVersionsByName:  s.Secrets.Handler.GetVersionsByName,
		UpdateSecretByName: s.Secrets.Handler.UpdateSecretByName,
		DeleteSecretByName: s.Secrets.Handler.DeleteSecretByName,
		RollbackByName:     s.Secrets.Handler.RollbackByName,
		GetSecret:          s.Secrets.Handler.GetSecret,
		GetVersions:        s.Secrets.Handler.GetVersions,
		CreateSecret:       s.Secrets.Handler.CreateSecret,
		UpdateSecret:       s.Secrets.Handler.UpdateSecret,
		DeleteSecret:       s.Secrets.Handler.DeleteSecret,
		RollbackToVersion:  s.Secrets.Handler.RollbackToVersion,

		TenantMiddleware:   s.Middleware.Tenant,
		TenantDBMiddleware: s.Middleware.TenantDB,
	}
}
