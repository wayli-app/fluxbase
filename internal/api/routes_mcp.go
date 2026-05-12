package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildCustomMCPRouteDeps() *routes.CustomMCPDeps {
	if s.MCP.CustomHandler == nil {
		return nil
	}
	return &routes.CustomMCPDeps{
		RequireAuth:      s.requireAuth,
		RequireAdmin:     middleware.RequireAdmin(),
		TenantMiddleware: s.Middleware.Tenant,
		GetConfig:        s.MCP.CustomHandler.GetConfig,
		ListTools:        s.MCP.CustomHandler.ListTools,
		CreateTool:       s.MCP.CustomHandler.CreateTool,
		SyncTool:         s.MCP.CustomHandler.SyncTool,
		GetTool:          s.MCP.CustomHandler.GetTool,
		UpdateTool:       s.MCP.CustomHandler.UpdateTool,
		DeleteTool:       s.MCP.CustomHandler.DeleteTool,
		TestTool:         s.MCP.CustomHandler.TestTool,
		ListResources:    s.MCP.CustomHandler.ListResources,
		CreateResource:   s.MCP.CustomHandler.CreateResource,
		SyncResource:     s.MCP.CustomHandler.SyncResource,
		GetResource:      s.MCP.CustomHandler.GetResource,
		UpdateResource:   s.MCP.CustomHandler.UpdateResource,
		DeleteResource:   s.MCP.CustomHandler.DeleteResource,
		TestResource:     s.MCP.CustomHandler.TestResource,
	}
}

func (s *Server) buildMCPRouteDeps() *routes.MCPDeps {
	if s.MCP.Handler == nil {
		return nil
	}
	return &routes.MCPDeps{
		BasePath:         s.config.MCP.BasePath,
		MCPAuth:          s.createMCPAuthMiddleware(),
		TenantMiddleware: s.Middleware.Tenant,
		HandlePost:       s.MCP.Handler.HandlePost,
		HandleGet:        s.MCP.Handler.HandleGet,
		HandleHealth:     s.MCP.Handler.HandleHealth,
	}
}

func (s *Server) buildMCPOAuthRouteDeps() *routes.MCPOAuthDeps {
	if s.MCP.OAuth == nil {
		return nil
	}
	return &routes.MCPOAuthDeps{
		BasePath:                          s.config.MCP.BasePath,
		HandleAuthorizationServerMetadata: s.MCP.OAuth.HandleAuthorizationServerMetadata,
		HandleProtectedResourceMetadata:   s.MCP.OAuth.HandleProtectedResourceMetadata,
		HandleClientRegistration:          s.MCP.OAuth.HandleClientRegistration,
		HandleAuthorize:                   s.MCP.OAuth.HandleAuthorize,
		HandleAuthorizeConsent:            s.MCP.OAuth.HandleAuthorizeConsent,
		HandleToken:                       s.MCP.OAuth.HandleToken,
		HandleRevoke:                      s.MCP.OAuth.HandleRevoke,
	}
}
