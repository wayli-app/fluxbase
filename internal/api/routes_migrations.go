package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildMigrationsRouteDeps() *routes.MigrationsDeps {
	if s.Schema.Migrations == nil || !s.config.Migrations.Enabled {
		return nil
	}

	var tenantPoolProvider middleware.MigrationsTenantPoolProvider
	if s.Tenancy.Manager != nil && s.Tenancy.Manager.GetRouter() != nil {
		tenantPoolProvider = s.Tenancy.Manager.GetRouter()
	}

	return &routes.MigrationsDeps{
		SecurityMiddleware: middleware.RequireMigrationsFullSecurityWithTenantProvider(
			&s.config.Migrations,
			&s.config.Server,
			s.db.Pool(),
			s.Auth.Handler.authService,
			s.config.Security.ServiceRoleRateLimit,
			s.config.Security.ServiceRoleRateWindow,
			s.sharedMiddlewareStorage,
			tenantPoolProvider,
		),
		TenantContext:     s.Middleware.Tenant,
		RequireRole:       RequireRole,
		CreateMigration:   s.Schema.Migrations.CreateMigration,
		ListMigrations:    s.Schema.Migrations.ListMigrations,
		GetMigration:      s.Schema.Migrations.GetMigration,
		UpdateMigration:   s.Schema.Migrations.UpdateMigration,
		DeleteMigration:   s.Schema.Migrations.DeleteMigration,
		ApplyMigration:    s.Schema.Migrations.ApplyMigration,
		RollbackMigration: s.Schema.Migrations.RollbackMigration,
		ApplyPending:      s.Schema.Migrations.ApplyPending,
		SyncMigrations:    s.Schema.Migrations.SyncMigrations,
		GetExecutions:     s.Schema.Migrations.GetExecutions,
	}
}
