package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
)

func (s *Server) registerRoutesViaRegistry() error {
	deps := &routes.AllDeps{
		Health:            s.buildHealthRouteDeps(),
		Realtime:          s.buildRealtimeRouteDeps(),
		Storage:           s.buildStorageRouteDeps(),
		REST:              s.buildRESTRouteDeps(),
		GraphQL:           s.buildGraphQLRouteDeps(),
		Vector:            s.buildVectorRouteDeps(),
		RPC:               s.buildRPCRouteDeps(),
		AI:                s.buildAIRouteDeps(),
		Settings:          s.buildSettingsRouteDeps(),
		UserSettings:      s.buildUserSettingsRouteDeps(),
		Dashboard:         s.buildDashboardAuthRouteDeps(),
		OpenAPI:           s.buildOpenAPIRouteDeps(),
		Auth:              s.buildAuthRouteDeps(),
		InternalAI:        s.buildInternalAIRouteDeps(),
		GitHubWebhook:     s.buildGitHubWebhookRouteDeps(),
		Invitation:        s.buildInvitationRouteDeps(),
		Webhook:           s.buildWebhookRouteDeps(),
		Monitoring:        s.buildMonitoringRouteDeps(),
		Functions:         s.buildFunctionsRouteDeps(),
		Jobs:              s.buildJobsRouteDeps(),
		ClientKeys:        s.buildClientKeysRouteDeps(),
		Secrets:           s.buildSecretsRouteDeps(),
		Sync:              s.buildSyncRouteDeps(),
		Admin:             s.buildAdminRouteDeps(),
		DashboardUserAuth: s.buildDashboardUserAuthRouteDeps(),
		CustomMCP:         s.buildCustomMCPRouteDeps(),
		MCP:               s.buildMCPRouteDeps(),
		MCPOAuth:          s.buildMCPOAuthRouteDeps(),
		Migrations:        s.buildMigrationsRouteDeps(),
		KnowledgeBase:     s.buildKnowledgeBaseRouteDeps(),
		Root:              s.handleHealth,
	}

	return routes.RegisterAllRoutes(s.app, deps)
}

func (s *Server) auditRegisteredRoutes() []routes.RouteAuditEntry {
	deps := &routes.AllDeps{
		Health:            s.buildHealthRouteDeps(),
		Realtime:          s.buildRealtimeRouteDeps(),
		Storage:           s.buildStorageRouteDeps(),
		REST:              s.buildRESTRouteDeps(),
		GraphQL:           s.buildGraphQLRouteDeps(),
		Vector:            s.buildVectorRouteDeps(),
		RPC:               s.buildRPCRouteDeps(),
		AI:                s.buildAIRouteDeps(),
		Settings:          s.buildSettingsRouteDeps(),
		UserSettings:      s.buildUserSettingsRouteDeps(),
		Dashboard:         s.buildDashboardAuthRouteDeps(),
		OpenAPI:           s.buildOpenAPIRouteDeps(),
		Auth:              s.buildAuthRouteDeps(),
		InternalAI:        s.buildInternalAIRouteDeps(),
		GitHubWebhook:     s.buildGitHubWebhookRouteDeps(),
		Invitation:        s.buildInvitationRouteDeps(),
		Webhook:           s.buildWebhookRouteDeps(),
		Monitoring:        s.buildMonitoringRouteDeps(),
		Functions:         s.buildFunctionsRouteDeps(),
		Jobs:              s.buildJobsRouteDeps(),
		ClientKeys:        s.buildClientKeysRouteDeps(),
		Secrets:           s.buildSecretsRouteDeps(),
		Sync:              s.buildSyncRouteDeps(),
		Admin:             s.buildAdminRouteDeps(),
		DashboardUserAuth: s.buildDashboardUserAuthRouteDeps(),
		CustomMCP:         s.buildCustomMCPRouteDeps(),
		MCP:               s.buildMCPRouteDeps(),
		MCPOAuth:          s.buildMCPOAuthRouteDeps(),
		Migrations:        s.buildMigrationsRouteDeps(),
		KnowledgeBase:     s.buildKnowledgeBaseRouteDeps(),
		Root:              s.handleHealth,
	}

	return routes.AuditRoutes(deps)
}
