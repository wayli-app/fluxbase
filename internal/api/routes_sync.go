package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildSyncRouteDeps() *routes.SyncDeps {
	deps := &routes.SyncDeps{
		RequireSyncAuth: UnifiedAuthMiddleware(s.Auth.Handler.authService, s.Auth.DashboardHandler.jwtManager, s.db),
		RequireRole:     RequireRole("admin", "instance_admin", "service_role"),
		TenantContext:   s.Middleware.Tenant,
	}

	if s.Functions.Handler != nil {
		deps.RequireFunctionsSyncIPAllowlist = middleware.RequireSyncIPAllowlist(s.config.Functions.SyncAllowedIPRanges, "functions", &s.config.Server)
		deps.SyncFunctions = s.Functions.Handler.SyncFunctions
	}

	if s.Jobs.Handler != nil {
		deps.RequireJobsSyncIPAllowlist = middleware.RequireSyncIPAllowlist(s.config.Jobs.SyncAllowedIPRanges, "jobs", &s.config.Server)
		deps.SyncJobs = s.Jobs.Handler.SyncJobs
	}

	if s.AI.Handler != nil {
		deps.RequireAIEnabled = middleware.RequireAIEnabled(s.Auth.SettingsCache)
		deps.RequireAISyncIPAllowlist = middleware.RequireSyncIPAllowlist(s.config.AI.SyncAllowedIPRanges, "ai", &s.config.Server)
		deps.SyncChatbots = s.AI.Handler.SyncChatbots
	}

	if s.RPC.Handler != nil {
		deps.RequireRPCEnabled = middleware.RequireRPCEnabled(s.Auth.SettingsCache)
		deps.RequireRPCSyncIPAllowlist = middleware.RequireSyncIPAllowlist(s.config.RPC.SyncAllowedIPRanges, "rpc", &s.config.Server)
		deps.SyncProcedures = s.RPC.Handler.SyncProcedures
	}

	return deps
}
