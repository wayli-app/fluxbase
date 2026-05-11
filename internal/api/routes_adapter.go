package api

import (
	"github.com/gofiber/fiber/v3"

	"github.com/nimbleflux/fluxbase/internal/ai"
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildHealthRouteDeps() *routes.HealthDeps {
	return &routes.HealthDeps{
		Handler:      s.handleHealth,
		OptionalAuth: s.optionalAuth,
	}
}

func (s *Server) buildRealtimeRouteDeps() *routes.RealtimeDeps {
	return &routes.RealtimeDeps{
		RequireRealtimeEnabled: middleware.RequireRealtimeEnabled(s.Auth.Handler.authService.GetSettingsCache()),
		OptionalAuth:           s.optionalAuth,
		RequireAuth:            s.requireAuth,
		RequireScope:           middleware.RequireScope,
		TenantMiddleware:       s.Middleware.Tenant,
		HandleWebSocket:        s.Realtime.Handler.HandleWebSocket,
		HandleStats:            s.handleRealtimeStats,
	}
}

func (s *Server) buildStorageRouteDeps() *routes.StorageDeps {
	return &routes.StorageDeps{
		RequireAuth:            s.requireAuth,
		OptionalAuth:           s.optionalAuth,
		RequireScope:           middleware.RequireScope,
		DownloadSignedObject:   s.Storage.Handler.DownloadSignedObject,
		GetTransformConfig:     s.Storage.Handler.GetTransformConfig,
		ListBuckets:            s.Storage.Handler.ListBuckets,
		CreateBucket:           s.Storage.Handler.CreateBucket,
		UpdateBucketSettings:   s.Storage.Handler.UpdateBucketSettings,
		DeleteBucket:           s.Storage.Handler.DeleteBucket,
		ListFiles:              s.Storage.Handler.ListFiles,
		MultipartUpload:        s.Storage.Handler.MultipartUpload,
		ShareObject:            s.Storage.Handler.ShareObject,
		RevokeShare:            s.Storage.Handler.RevokeShare,
		ListShares:             s.Storage.Handler.ListShares,
		GenerateSignedURL:      s.Storage.Handler.GenerateSignedURL,
		StreamUpload:           s.Storage.Handler.StreamUpload,
		StorageUploadLimiter:   middleware.StorageUploadLimiter(s.sharedMiddlewareStorage),
		InitChunkedUpload:      s.Storage.Handler.InitChunkedUpload,
		UploadChunk:            s.Storage.Handler.UploadChunk,
		CompleteChunkedUpload:  s.Storage.Handler.CompleteChunkedUpload,
		GetChunkedUploadStatus: s.Storage.Handler.GetChunkedUploadStatus,
		AbortChunkedUpload:     s.Storage.Handler.AbortChunkedUpload,
		UploadFile:             s.Storage.Handler.UploadFile,
		DownloadFile:           s.Storage.Handler.DownloadFile,
		DeleteFile:             s.Storage.Handler.DeleteFile,

		TenantMiddleware:   s.Middleware.Tenant,
		TenantDBMiddleware: s.Middleware.TenantDB,
	}
}

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

func (s *Server) buildRPCRouteDeps() *routes.RPCDeps {
	if s.RPC.Handler == nil {
		return nil
	}
	return &routes.RPCDeps{
		RequireRPCEnabled: middleware.RequireRPCEnabled(s.Auth.Handler.authService.GetSettingsCache()),
		OptionalAuth:      s.optionalAuth,
		RequireScope:      middleware.RequireScope,
		ListProcedures:    s.RPC.Handler.ListPublicProcedures,
		Invoke:            s.RPC.Handler.Invoke,
		GetExecution:      s.RPC.Handler.GetPublicExecution,
		GetExecutionLogs:  s.RPC.Handler.GetPublicExecutionLogs,

		TenantMiddleware:   s.Middleware.Tenant,
		TenantDBMiddleware: s.Middleware.TenantDB,
	}
}

func (s *Server) buildAIRouteDeps() *routes.AIDeps {
	if s.AI.Chat == nil || s.AI.Handler == nil {
		return nil
	}
	return &routes.AIDeps{
		RequireAIEnabled:       middleware.RequireAIEnabled(s.Auth.Handler.authService.GetSettingsCache()),
		OptionalAuth:           s.optionalAuth,
		RequireAuth:            s.requireAuth,
		TenantMiddleware:       s.Middleware.Tenant,
		HandleWebSocket:        s.AI.Chat.HandleWebSocket,
		ListPublicChatbots:     s.AI.Handler.ListPublicChatbots,
		LookupChatbotByName:    s.AI.Handler.LookupChatbotByName,
		GetPublicChatbot:       s.AI.Handler.GetPublicChatbot,
		ListUserConversations:  s.AI.Handler.ListUserConversations,
		GetUserConversation:    s.AI.Handler.GetUserConversation,
		DeleteUserConversation: s.AI.Handler.DeleteUserConversation,
		UpdateUserConversation: s.AI.Handler.UpdateUserConversation,
	}
}

func (s *Server) buildSettingsRouteDeps() *routes.SettingsDeps {
	return &routes.SettingsDeps{
		OptionalAuth: s.optionalAuth,
		GetSetting:   s.Settings.Handler.GetSetting,
		GetSettings:  s.Settings.Handler.GetSettings,
		BatchGet:     s.Settings.Handler.GetSettings,
	}
}

func (s *Server) buildUserSettingsRouteDeps() *routes.UserSettingsDeps {
	return &routes.UserSettingsDeps{
		RequireAuth:       s.requireAuth,
		TenantMiddleware:  s.Middleware.Tenant,
		ListSettings:      s.Settings.User.ListSettings,
		GetUserOwnSetting: s.Settings.User.GetUserOwnSetting,
		GetSystemSetting:  s.Settings.User.GetSystemSettingPublic,
		GetSetting:        s.Settings.User.GetSetting,
		SetSetting:        s.Settings.User.SetSetting,
		DeleteSetting:     s.Settings.User.DeleteSetting,
		CreateSecret:      s.Settings.User.CreateSecret,
		ListSecrets:       s.Settings.User.ListSecrets,
		GetSecret:         s.Settings.User.GetSecret,
		UpdateSecret:      s.Settings.User.UpdateSecret,
		DeleteSecret:      s.Settings.User.DeleteSecret,
	}
}

func (s *Server) buildDashboardAuthRouteDeps() *routes.DashboardAuthDeps {
	return &routes.DashboardAuthDeps{
		SetupLimiter:    middleware.AdminSetupLimiterWithConfig(s.config.Security.AdminSetupRateLimit, s.config.Security.AdminSetupRateWindow, s.sharedMiddlewareStorage),
		LoginLimiter:    middleware.AdminLoginLimiterWithConfig(s.config.Security.AdminLoginRateLimit, s.config.Security.AdminLoginRateWindow, s.sharedMiddlewareStorage),
		GetSetupStatus:  s.Auth.AdminHandler.GetSetupStatus,
		InitialSetup:    s.Auth.AdminHandler.InitialSetup,
		AdminLogin:      s.Auth.AdminHandler.AdminLogin,
		RefreshToken:    s.Auth.AdminHandler.AdminRefreshToken,
		UnifiedAuth:     UnifiedAuthMiddleware(s.Auth.Handler.authService, s.Auth.DashboardHandler.jwtManager, s.db),
		AdminLogout:     s.Auth.AdminHandler.AdminLogout,
		GetCurrentAdmin: s.Auth.AdminHandler.GetCurrentAdmin,
	}
}

func (s *Server) buildOpenAPIRouteDeps() *routes.OpenAPIDeps {
	return &routes.OpenAPIDeps{
		OptionalAuth:    s.optionalAuth,
		TenantContext:   s.Middleware.Tenant,
		TenantDBContext: s.Middleware.TenantDB,
		GetOpenAPISpec:  NewOpenAPIHandler(s.db).GetOpenAPISpec,
	}
}

func (s *Server) buildAuthRouteDeps() *routes.AuthDeps {
	rateLimiters := map[string]fiber.Handler{
		"signup":         middleware.AuthSignupLimiterWithConfig(s.config.Security.AuthSignupRateLimit, s.config.Security.AuthSignupRateWindow, s.sharedMiddlewareStorage),
		"login":          middleware.AuthLoginLimiterWithConfig(s.config.Security.AuthLoginRateLimit, s.config.Security.AuthLoginRateWindow, s.sharedMiddlewareStorage),
		"refresh":        middleware.AuthRefreshLimiterWithConfig(s.config.Security.AuthRefreshRateLimit, s.config.Security.AuthRefreshRateWindow, s.sharedMiddlewareStorage),
		"magiclink":      middleware.AuthMagicLinkLimiterWithConfig(s.config.Security.AuthMagicLinkRateLimit, s.config.Security.AuthMagicLinkRateWindow, s.sharedMiddlewareStorage),
		"password_reset": middleware.AuthPasswordResetLimiterWithConfig(s.config.Security.AuthPasswordResetRateLimit, s.config.Security.AuthPasswordResetRateWindow, s.sharedMiddlewareStorage),
		"otp":            middleware.AuthMagicLinkLimiterWithConfig(s.config.Security.AuthMagicLinkRateLimit, s.config.Security.AuthMagicLinkRateWindow, s.sharedMiddlewareStorage),
		"2fa":            middleware.Auth2FALimiterWithConfig(s.config.Security.Auth2FARateLimit, s.config.Security.Auth2FARateWindow, s.sharedMiddlewareStorage),
	}

	return &routes.AuthDeps{
		AuthMiddleware:            AuthMiddleware(s.Auth.Handler.authService),
		TenantMiddleware:          s.Middleware.Tenant,
		RequireRole:               RequireRole,
		RequireScope:              middleware.RequireScope,
		RateLimiters:              rateLimiters,
		GetCSRFToken:              s.Auth.Handler.GetCSRFToken,
		GetCaptchaConfig:          s.Auth.Handler.GetCaptchaConfig,
		CheckCaptcha:              s.Auth.Handler.CheckCaptcha,
		GetAuthConfig:             s.Auth.Handler.GetAuthConfig,
		SignUp:                    s.Auth.Handler.SignUp,
		SignIn:                    s.Auth.Handler.SignIn,
		RefreshToken:              s.Auth.Handler.RefreshToken,
		SendMagicLink:             s.Auth.Handler.SendMagicLink,
		VerifyMagicLink:           s.Auth.Handler.VerifyMagicLink,
		RequestPasswordReset:      s.Auth.Handler.RequestPasswordReset,
		ResetPassword:             s.Auth.Handler.ResetPassword,
		VerifyPasswordReset:       s.Auth.Handler.VerifyPasswordResetToken,
		VerifyEmail:               s.Auth.Handler.VerifyEmail,
		ResendVerification:        s.Auth.Handler.ResendVerificationEmail,
		VerifyTOTP:                s.Auth.Handler.VerifyTOTP,
		SendOTP:                   s.Auth.Handler.SendOTP,
		VerifyOTP:                 s.Auth.Handler.VerifyOTP,
		ResendOTP:                 s.Auth.Handler.ResendOTP,
		SignInWithIDToken:         s.Auth.Handler.SignInWithIDToken,
		SignOut:                   s.Auth.Handler.SignOut,
		GetUser:                   s.Auth.Handler.GetUser,
		UpdateUser:                s.Auth.Handler.UpdateUser,
		StartImpersonation:        s.Auth.Handler.StartImpersonation,
		StartAnonImpersonation:    s.Auth.Handler.StartAnonImpersonation,
		StopImpersonation:         s.Auth.Handler.StopImpersonation,
		GetActiveImpersonation:    s.Auth.Handler.GetActiveImpersonation,
		ListImpersonationSessions: s.Auth.Handler.ListImpersonationSessions,
		SetupTOTP:                 s.Auth.Handler.SetupTOTP,
		EnableTOTP:                s.Auth.Handler.EnableTOTP,
		DisableTOTP:               s.Auth.Handler.DisableTOTP,
		GetTOTPStatus:             s.Auth.Handler.GetTOTPStatus,
		GetUserIdentities:         s.Auth.Handler.GetUserIdentities,
		LinkIdentity:              s.Auth.Handler.LinkIdentity,
		UnlinkIdentity:            s.Auth.Handler.UnlinkIdentity,
		Reauthenticate:            s.Auth.Handler.Reauthenticate,
		ListOAuthProviders:        s.Auth.OAuth.ListEnabledProviders,
		OAuthAuthorize:            s.Auth.OAuth.Authorize,
		OAuthCallback:             s.Auth.OAuth.Callback,
		GetSPMetadata:             s.Auth.SAMLProvider.GetSPMetadata,
		StartServiceImpersonation: s.Auth.Handler.StartServiceImpersonation,
		GetProviderToken:          s.Auth.OAuth.GetProviderToken,
		OAuthLogout:               s.Auth.OAuth.Logout,
		OAuthLogoutCallback:       s.Auth.OAuth.LogoutCallback,
		ListSAMLProviders:         s.Auth.SAML.ListSAMLProviders,
		InitiateSAMLLogin:         s.Auth.SAML.InitiateSAMLLogin,
		HandleSAMLAssertion:       s.Auth.SAML.HandleSAMLAssertion,
		HandleSAMLLogout:          s.Auth.SAML.HandleSAMLLogout,
		InitiateSAMLLogout:        s.Auth.SAML.InitiateSAMLLogout,
	}
}

func (s *Server) buildInternalAIRouteDeps() *routes.InternalAIDeps {
	if s.AI.Internal == nil {
		return nil
	}
	return &routes.InternalAIDeps{
		RequireInternal:     middleware.RequireInternal(),
		RequireAuth:         s.requireAuth,
		HandleChat:          s.AI.Internal.HandleChat,
		HandleEmbed:         s.AI.Internal.HandleEmbed,
		HandleListProviders: s.AI.Internal.HandleListProviders,
	}
}

func (s *Server) buildGitHubWebhookRouteDeps() *routes.GitHubWebhookDeps {
	if s.Branching.GitHub == nil {
		return nil
	}
	return &routes.GitHubWebhookDeps{
		GitHubWebhookLimiter: middleware.GitHubWebhookLimiter(s.sharedMiddlewareStorage),
		HandleWebhook:        s.Branching.GitHub.HandleWebhook,
	}
}

func (s *Server) buildInvitationRouteDeps() *routes.InvitationDeps {
	return &routes.InvitationDeps{
		ValidateInvitation: s.Auth.Invitation.ValidateInvitation,
		AcceptInvitation:   s.Auth.Invitation.AcceptInvitation,
	}
}

func (s *Server) buildWebhookRouteDeps() *routes.WebhookDeps {
	return &routes.WebhookDeps{
		RequireAuth:    s.requireAuth,
		RequireScope:   middleware.RequireScope,
		ListWebhooks:   s.Webhook.Handler.ListWebhooks,
		GetWebhook:     s.Webhook.Handler.GetWebhook,
		ListDeliveries: s.Webhook.Handler.ListDeliveries,
		CreateWebhook:  s.Webhook.Handler.CreateWebhook,
		UpdateWebhook:  s.Webhook.Handler.UpdateWebhook,
		DeleteWebhook:  s.Webhook.Handler.DeleteWebhook,
		TestWebhook:    s.Webhook.Handler.TestWebhook,

		TenantMiddleware:   s.Middleware.Tenant,
		TenantDBMiddleware: s.Middleware.TenantDB,
	}
}

func (s *Server) buildMonitoringRouteDeps() *routes.MonitoringDeps {
	return &routes.MonitoringDeps{
		RequireAuth:        s.requireAuth,
		RequireScope:       middleware.RequireScope,
		TenantMiddleware:   s.Middleware.Tenant,
		TenantDBMiddleware: s.Middleware.TenantDB,
		GetMetrics:         s.Monitoring.Handler.GetMetrics,
		GetHealth:          s.Monitoring.Handler.GetHealth,
		GetLogs:            s.Monitoring.Handler.GetLogs,
	}
}

func (s *Server) buildFunctionsRouteDeps() *routes.FunctionsDeps {
	if s.Functions.Handler == nil {
		return nil
	}
	return &routes.FunctionsDeps{
		RequireFunctionsEnabled: middleware.RequireFunctionsEnabled(s.Auth.Handler.authService.GetSettingsCache()),
		RequireAuth:             s.requireAuth,
		OptionalAuth:            s.optionalAuth,
		RequireScope:            middleware.RequireScope,
		TenantMiddleware:        s.Middleware.Tenant,
		ListFunctions:           s.Functions.Handler.ListFunctions,
		GetFunction:             s.Functions.Handler.GetFunction,
		CreateFunction:          s.Functions.Handler.CreateFunction,
		UpdateFunction:          s.Functions.Handler.UpdateFunction,
		DeleteFunction:          s.Functions.Handler.DeleteFunction,
		InvokeFunction:          s.Functions.Handler.InvokeFunction,
		GetExecutions:           s.Functions.Handler.GetExecutions,
		ListSharedModules:       s.Functions.Handler.ListSharedModules,
		GetSharedModule:         s.Functions.Handler.GetSharedModule,
		CreateSharedModule:      s.Functions.Handler.CreateSharedModule,
		UpdateSharedModule:      s.Functions.Handler.UpdateSharedModule,
		DeleteSharedModule:      s.Functions.Handler.DeleteSharedModule,
	}
}

func (s *Server) buildJobsRouteDeps() *routes.JobsDeps {
	if s.Jobs.Handler == nil {
		return nil
	}
	return &routes.JobsDeps{
		RequireJobsEnabled: middleware.RequireJobsEnabled(s.Auth.Handler.authService.GetSettingsCache()),
		RequireAuth:        s.requireAuth,
		SubmitJob:          s.Jobs.Handler.SubmitJob,
		GetJob:             s.Jobs.Handler.GetJob,
		ListJobs:           s.Jobs.Handler.ListJobs,
		CancelJob:          s.Jobs.Handler.CancelJob,
		RetryJob:           s.Jobs.Handler.RetryJob,
		GetJobLogsUser:     s.Jobs.Handler.GetJobLogsUser,

		TenantMiddleware:   s.Middleware.Tenant,
		TenantDBMiddleware: s.Middleware.TenantDB,
	}
}

func (s *Server) buildBranchRouteDeps() *routes.BranchDeps {
	if s.Branching.Handler == nil || !s.config.Branching.Enabled {
		return nil
	}

	return &routes.BranchDeps{
		GetActiveBranch:    s.Branching.Handler.GetActiveBranch,
		SetActiveBranch:    s.Branching.Handler.SetActiveBranch,
		ResetActiveBranch:  s.Branching.Handler.ResetActiveBranch,
		GetPoolStats:       s.Branching.Handler.GetPoolStats,
		CreateBranch:       s.Branching.Handler.CreateBranch,
		ListBranches:       s.Branching.Handler.ListBranches,
		GetBranch:          s.Branching.Handler.GetBranch,
		DeleteBranch:       s.Branching.Handler.DeleteBranch,
		ResetBranch:        s.Branching.Handler.ResetBranch,
		GetBranchActivity:  s.Branching.Handler.GetBranchActivity,
		ListBranchAccess:   s.Branching.Handler.ListBranchAccess,
		GrantBranchAccess:  s.Branching.Handler.GrantBranchAccess,
		RevokeBranchAccess: s.Branching.Handler.RevokeBranchAccess,
		ListGitHubConfigs:  s.Branching.Handler.ListGitHubConfigs,
		UpsertGitHubConfig: s.Branching.Handler.UpsertGitHubConfig,
		DeleteGitHubConfig: s.Branching.Handler.DeleteGitHubConfig,
	}
}

func (s *Server) buildClientKeysRouteDeps() *routes.ClientKeysDeps {
	return &routes.ClientKeysDeps{
		RequireAuth:                      s.requireAuth,
		RequireAdminIfClientKeysDisabled: middleware.RequireAdminIfClientKeysDisabled(s.Auth.Handler.authService.GetSettingsCache()),
		RequireScope:                     middleware.RequireScope,
		TenantMiddleware:                 s.Middleware.Tenant,
		ListClientKeys:                   s.Auth.ClientKeyHandler.ListClientKeys,
		GetClientKey:                     s.Auth.ClientKeyHandler.GetClientKey,
		CreateClientKey:                  s.Auth.ClientKeyHandler.CreateClientKey,
		UpdateClientKey:                  s.Auth.ClientKeyHandler.UpdateClientKey,
		DeleteClientKey:                  s.Auth.ClientKeyHandler.DeleteClientKey,
		RevokeClientKey:                  s.Auth.ClientKeyHandler.RevokeClientKey,
	}
}

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

func (s *Server) buildSyncRouteDeps() *routes.SyncDeps {
	deps := &routes.SyncDeps{
		RequireSyncAuth: UnifiedAuthMiddleware(s.Auth.Handler.authService, s.Auth.DashboardHandler.jwtManager, s.db),
		RequireRole:     RequireRole("admin", "instance_admin", "service_role"),
		TenantContext:   s.Middleware.Tenant,
	}

	// Functions sync
	if s.Functions.Handler != nil {
		deps.RequireFunctionsSyncIPAllowlist = middleware.RequireSyncIPAllowlist(s.config.Functions.SyncAllowedIPRanges, "functions", &s.config.Server)
		deps.SyncFunctions = s.Functions.Handler.SyncFunctions
	}

	// Jobs sync
	if s.Jobs.Handler != nil {
		deps.RequireJobsSyncIPAllowlist = middleware.RequireSyncIPAllowlist(s.config.Jobs.SyncAllowedIPRanges, "jobs", &s.config.Server)
		deps.SyncJobs = s.Jobs.Handler.SyncJobs
	}

	// AI sync
	if s.AI.Handler != nil {
		deps.RequireAIEnabled = middleware.RequireAIEnabled(s.Auth.Handler.authService.GetSettingsCache())
		deps.RequireAISyncIPAllowlist = middleware.RequireSyncIPAllowlist(s.config.AI.SyncAllowedIPRanges, "ai", &s.config.Server)
		deps.SyncChatbots = s.AI.Handler.SyncChatbots
	}

	// RPC sync
	if s.RPC.Handler != nil {
		deps.RequireRPCEnabled = middleware.RequireRPCEnabled(s.Auth.Handler.authService.GetSettingsCache())
		deps.RequireRPCSyncIPAllowlist = middleware.RequireSyncIPAllowlist(s.config.RPC.SyncAllowedIPRanges, "rpc", &s.config.Server)
		deps.SyncProcedures = s.RPC.Handler.SyncProcedures
	}

	return deps
}

func (s *Server) buildDashboardUserAuthRouteDeps() *routes.DashboardUserAuthDeps {
	return &routes.DashboardUserAuthDeps{
		RequireDashboardAuth:     s.Auth.DashboardHandler.RequireDashboardAuth,
		TenantMiddleware:         s.Middleware.Tenant,
		Signup:                   s.Auth.DashboardHandler.Signup,
		Login:                    s.Auth.DashboardHandler.Login,
		RefreshToken:             s.Auth.DashboardHandler.RefreshToken,
		VerifyTOTP:               s.Auth.DashboardHandler.VerifyTOTP,
		RequestPasswordReset:     s.Auth.DashboardHandler.RequestPasswordReset,
		VerifyPasswordResetToken: s.Auth.DashboardHandler.VerifyPasswordResetToken,
		ConfirmPasswordReset:     s.Auth.DashboardHandler.ConfirmPasswordReset,
		GetSSOProviders:          s.Auth.DashboardHandler.GetSSOProviders,
		InitiateOAuthLogin:       s.Auth.DashboardHandler.InitiateOAuthLogin,
		OAuthCallback:            s.Auth.DashboardHandler.OAuthCallback,
		InitiateSAMLLogin:        s.Auth.DashboardHandler.InitiateSAMLLogin,
		SAMLACSCallback:          s.Auth.DashboardHandler.SAMLACSCallback,
		GetCurrentUser:           s.Auth.DashboardHandler.GetCurrentUser,
		UpdateProfile:            s.Auth.DashboardHandler.UpdateProfile,
		ChangePassword:           s.Auth.DashboardHandler.ChangePassword,
		DeleteAccount:            s.Auth.DashboardHandler.DeleteAccount,
		SetupTOTP:                s.Auth.DashboardHandler.SetupTOTP,
		EnableTOTP:               s.Auth.DashboardHandler.EnableTOTP,
		DisableTOTP:              s.Auth.DashboardHandler.DisableTOTP,
	}
}

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

// knowledgeBaseDisabledHandler returns a handler that responds with "AI not enabled" error
func knowledgeBaseDisabledHandler(c fiber.Ctx) error {
	return SendFeatureDisabled(c, "AI features")
}

func (s *Server) buildKnowledgeBaseRouteDeps() *routes.KnowledgeBaseDeps {
	deps := &routes.KnowledgeBaseDeps{
		RequireAIEnabled: middleware.RequireAIEnabled(s.Auth.Handler.authService.GetSettingsCache()),
		RequireAuth:      s.requireAuth,
		TenantMiddleware: s.Middleware.Tenant,
	}

	// If AI/knowledge base storage is not available, use stub handlers
	// This ensures routes return a proper error instead of 404
	if s.AI.KBStorage == nil {
		deps.ListKBs = knowledgeBaseDisabledHandler
		deps.CreateKB = knowledgeBaseDisabledHandler
		deps.GetKB = knowledgeBaseDisabledHandler
		deps.ShareKB = knowledgeBaseDisabledHandler
		deps.ListPermissions = knowledgeBaseDisabledHandler
		deps.RevokePermission = knowledgeBaseDisabledHandler
		deps.ListDocuments = knowledgeBaseDisabledHandler
		deps.GetDocument = knowledgeBaseDisabledHandler
		deps.AddDocument = knowledgeBaseDisabledHandler
		deps.UploadDocument = knowledgeBaseDisabledHandler
		deps.DeleteDocument = knowledgeBaseDisabledHandler
		deps.SearchKB = knowledgeBaseDisabledHandler
		return deps
	}

	handler := ai.NewUserKnowledgeBaseHandler(s.AI.KBStorage)
	if s.AI.DocProcessor != nil {
		handler = ai.NewUserKnowledgeBaseHandlerWithProcessor(s.AI.KBStorage, s.AI.DocProcessor)
	}

	deps.ListKBs = handler.ListMyKnowledgeBases
	deps.CreateKB = handler.CreateMyKnowledgeBase
	deps.GetKB = handler.GetMyKnowledgeBase
	deps.ShareKB = handler.ShareKnowledgeBase
	deps.ListPermissions = handler.ListPermissions
	deps.RevokePermission = handler.RevokePermission

	if s.AI.DocProcessor != nil {
		deps.ListDocuments = handler.ListMyDocuments
		deps.GetDocument = handler.GetMyDocument
		deps.AddDocument = handler.AddMyDocument
		deps.UploadDocument = handler.UploadMyDocument
		deps.DeleteDocument = handler.DeleteMyDocument
		deps.SearchKB = handler.SearchMyKB
	}

	return deps
}

func (s *Server) buildAdminRouteDeps() *routes.AdminDeps {
	unifiedAuth := UnifiedAuthMiddleware(s.Auth.Handler.authService, s.Auth.DashboardHandler.jwtManager, s.db)
	return &routes.AdminDeps{
		UnifiedAuth:        unifiedAuth,
		RequireRole:        RequireRole,
		TenantMiddleware:   s.Middleware.Tenant,
		TenantDBMiddleware: s.Middleware.TenantDB,

		// Subgroup dependencies
		Branch: s.buildBranchRouteDeps(),
		Schema: &routes.SchemaAdminDeps{
			GetTables:               s.handleGetTables,
			GetTableSchema:          s.handleGetTableSchema,
			GetSchemas:              s.handleGetSchemas,
			ExecuteQuery:            s.handleExecuteQuery,
			ListSchemasDDL:          s.Schema.DDL.ListSchemas,
			CreateSchemaDDL:         s.Schema.DDL.CreateSchema,
			ListTablesDDL:           s.Schema.DDL.ListTables,
			CreateTableDDL:          s.Schema.DDL.CreateTable,
			DeleteTableDDL:          s.Schema.DDL.DeleteTable,
			RenameTableDDL:          s.Schema.DDL.RenameTable,
			AddColumnDDL:            s.Schema.DDL.AddColumn,
			DropColumnDDL:           s.Schema.DDL.DropColumn,
			EnableRealtime:          s.Realtime.Admin.HandleEnableRealtime,
			ListRealtimeTables:      s.Realtime.Admin.HandleListRealtimeTables,
			GetRealtimeStatus:       s.Realtime.Admin.HandleGetRealtimeStatus,
			UpdateRealtimeConfig:    s.Realtime.Admin.HandleUpdateRealtimeConfig,
			DisableRealtime:         s.Realtime.Admin.HandleDisableRealtime,
			ExecuteSQL:              s.sqlHandler.ExecuteSQL,
			ExportTypeScript:        s.Schema.Export.HandleExportTypeScript,
			RefreshSchema:           s.handleRefreshSchema,
			GetSchemaGraph:          s.GetSchemaGraph,
			GetTableRelationships:   s.GetTableRelationships,
			GetTablesWithRLS:        s.GetTablesWithRLS,
			GetTableRLSStatus:       s.GetTableRLSStatus,
			ToggleTableRLS:          s.ToggleTableRLS,
			ListPolicies:            s.ListPolicies,
			CreatePolicy:            s.CreatePolicy,
			UpdatePolicy:            s.UpdatePolicy,
			DeletePolicy:            s.DeletePolicy,
			GetPolicyTemplates:      s.GetPolicyTemplates,
			GetSecurityWarnings:     s.GetSecurityWarnings,
			DumpInternalSchema:      s.Schema.InternalSchema.DumpSchema,
			PlanInternalSchema:      s.Schema.InternalSchema.PlanSchema,
			ApplyInternalSchema:     s.Schema.InternalSchema.ApplySchema,
			ValidateInternalSchema:  s.Schema.InternalSchema.ValidateSchema,
			GetInternalSchemaStatus: s.Schema.InternalSchema.GetSchemaStatus,
			MigrateInternalSchema:   s.Schema.InternalSchema.MigrateSchema,
		},
		AuthProviders: &routes.AuthProvidersAdminDeps{
			ListOAuthProviders:  s.Auth.OAuthProvider.ListOAuthProviders,
			GetOAuthProvider:    s.Auth.OAuthProvider.GetOAuthProvider,
			CreateOAuthProvider: s.Auth.OAuthProvider.CreateOAuthProvider,
			UpdateOAuthProvider: s.Auth.OAuthProvider.UpdateOAuthProvider,
			DeleteOAuthProvider: s.Auth.OAuthProvider.DeleteOAuthProvider,
			ListSAMLProviders:   s.Auth.SAML.ListSAMLProviders,
			GetSAMLProvider:     s.Auth.SAMLProvider.GetSAMLProvider,
			CreateSAMLProvider:  s.Auth.SAMLProvider.CreateSAMLProvider,
			UpdateSAMLProvider:  s.Auth.SAMLProvider.UpdateSAMLProvider,
			DeleteSAMLProvider:  s.Auth.SAMLProvider.DeleteSAMLProvider,
			ValidateSAML:        s.Auth.SAMLProvider.ValidateMetadata,
			UploadSAMLMetadata:  s.Auth.SAMLProvider.UploadMetadata,
			GetAuthSettings:     s.Auth.OAuthProvider.GetAuthSettings,
			UpdateAuthSettings:  s.Auth.OAuthProvider.UpdateAuthSettings,
			ListSessions:        s.Auth.AdminSession.ListSessions,
			RevokeSession:       s.Auth.AdminSession.RevokeSession,
			RevokeUserSessions:  s.Auth.AdminSession.RevokeUserSessions,
		},
		Users: &routes.UsersAdminDeps{
			ListUsers:           s.Auth.UserManagement.ListUsers,
			InviteUser:          s.Auth.UserManagement.InviteUser,
			DeleteUser:          s.Auth.UserManagement.DeleteUser,
			UpdateUser:          s.Auth.UserManagement.UpdateUser,
			UpdateUserRole:      s.Auth.UserManagement.UpdateUserRole,
			ResetUserPassword:   s.Auth.UserManagement.ResetUserPassword,
			ListUsersWithQuotas: s.Quota.Handler.ListUsersWithQuotas,
			GetUserQuota:        s.Quota.Handler.GetUserQuota,
			SetUserQuota:        s.Quota.Handler.SetUserQuota,
			CreateInvitation:    s.Auth.Invitation.CreateInvitation,
			ListInvitations:     s.Auth.Invitation.ListInvitations,
			RevokeInvitation:    s.Auth.Invitation.RevokeInvitation,
		},
		Tenants: &routes.TenantsAdminDeps{
			ListMyTenants:             s.Tenancy.Tenant.ListMyTenants,
			ListTenants:               s.Tenancy.Tenant.ListTenants,
			ListDeletedTenants:        s.Tenancy.Tenant.ListDeletedTenants,
			CreateTenant:              s.Tenancy.Tenant.CreateTenant,
			GetTenant:                 s.Tenancy.Tenant.GetTenant,
			UpdateTenant:              s.Tenancy.Tenant.UpdateTenant,
			DeleteTenant:              s.Tenancy.Tenant.DeleteTenant,
			RecoverTenant:             s.Tenancy.Tenant.RecoverTenant,
			MigrateTenant:             s.Tenancy.Tenant.MigrateTenant,
			RepairTenant:              s.Tenancy.Tenant.RepairTenant,
			ListAdmins:                s.Tenancy.Tenant.ListAdmins,
			AssignAdmin:               s.Tenancy.Tenant.AssignAdmin,
			RemoveAdmin:               s.Tenancy.Tenant.RemoveAdmin,
			GetTenantSettings:         s.Settings.Tenant.GetTenantSettings,
			UpdateTenantSettings:      s.Settings.Tenant.UpdateTenantSettings,
			DeleteTenantSetting:       s.Settings.Tenant.DeleteTenantSetting,
			GetTenantSetting:          s.Settings.Tenant.GetTenantSetting,
			GetTenantSchemaStatus:     s.Tenancy.Tenant.GetTenantSchemaStatus,
			ApplyTenantSchema:         s.Tenancy.Tenant.ApplyTenantSchema,
			GetStoredSchema:           s.Tenancy.Tenant.GetStoredSchema,
			UploadTenantSchema:        s.Tenancy.Tenant.UploadTenantSchema,
			ApplyUploadedTenantSchema: s.Tenancy.Tenant.ApplyUploadedTenantSchema,
			DeleteStoredSchema:        s.Tenancy.Tenant.DeleteStoredSchema,
		},
		ServiceKeys: &routes.ServiceKeysAdminDeps{
			ListServiceKeys:      s.Tenancy.ServiceKey.ListServiceKeys,
			GetServiceKey:        s.Tenancy.ServiceKey.GetServiceKey,
			CreateServiceKey:     s.Tenancy.ServiceKey.CreateServiceKey,
			UpdateServiceKey:     s.Tenancy.ServiceKey.UpdateServiceKey,
			DeleteServiceKey:     s.Tenancy.ServiceKey.DeleteServiceKey,
			DisableServiceKey:    s.Tenancy.ServiceKey.DisableServiceKey,
			EnableServiceKey:     s.Tenancy.ServiceKey.EnableServiceKey,
			RevokeServiceKey:     s.Tenancy.ServiceKey.RevokeServiceKey,
			DeprecateServiceKey:  s.Tenancy.ServiceKey.DeprecateServiceKey,
			RotateServiceKey:     s.Tenancy.ServiceKey.RotateServiceKey,
			GetRevocationHistory: s.Tenancy.ServiceKey.GetRevocationHistory,
		},
		Functions: &routes.FunctionsAdminDeps{
			ReloadFunctions:        s.Functions.Handler.ReloadFunctions,
			ListFunctionNamespaces: s.Functions.Handler.ListNamespaces,
			ListAllExecutions:      s.Functions.Handler.ListAllExecutions,
			GetExecutionLogs:       s.Functions.Handler.GetExecutionLogs,
			SyncFunctions:          s.Functions.Handler.SyncFunctions,
		},
		Jobs: &routes.JobsAdminDeps{
			ListJobNamespaces: s.Jobs.Handler.ListNamespaces,
			ListJobFunctions:  s.Jobs.Handler.ListJobFunctions,
			GetJobFunction:    s.Jobs.Handler.GetJobFunction,
			DeleteJobFunction: s.Jobs.Handler.DeleteJobFunction,
			GetJobStats:       s.Jobs.Handler.GetJobStats,
			ListWorkers:       s.Jobs.Handler.ListWorkers,
			ListAllJobs:       s.Jobs.Handler.ListAllJobs,
			GetJobAdmin:       s.Jobs.Handler.GetJobAdmin,
			TerminateJob:      s.Jobs.Handler.TerminateJob,
			CancelJobAdmin:    s.Jobs.Handler.CancelJobAdmin,
			RetryJobAdmin:     s.Jobs.Handler.RetryJobAdmin,
			ResubmitJobAdmin:  s.Jobs.Handler.ResubmitJobAdmin,
			SyncJobs:          s.Jobs.Handler.SyncJobs,
		},
		AI: &routes.AIAdminDeps{
			ListChatbots:               s.AI.Handler.ListChatbots,
			GetChatbot:                 s.AI.Handler.GetChatbot,
			ToggleChatbot:              s.AI.Handler.ToggleChatbot,
			UpdateChatbot:              s.AI.Handler.UpdateChatbot,
			DeleteChatbot:              s.AI.Handler.DeleteChatbot,
			SyncChatbots:               s.AI.Handler.SyncChatbots,
			GetAIMetrics:               s.AI.Handler.GetAIMetrics,
			ListAIProviders:            s.AI.Handler.ListProviders,
			CreateAIProvider:           s.AI.Handler.CreateProvider,
			UpdateAIProvider:           s.AI.Handler.UpdateProvider,
			DeleteAIProvider:           s.AI.Handler.DeleteProvider,
			SetDefaultAIProvider:       s.AI.Handler.SetDefaultProvider,
			SetEmbeddingAIProvider:     s.AI.Handler.SetEmbeddingProvider,
			ClearEmbeddingAIProvider:   s.AI.Handler.ClearEmbeddingProvider,
			ListAIConversations:        s.AI.Handler.GetConversations,
			GetAIConversationMessages:  s.AI.Handler.GetConversationMessages,
			GetAIAuditLog:              s.AI.Handler.GetAuditLog,
			ListExportableTables:       s.AI.KnowledgeBase.ListExportableTables,
			GetExportableTableDetails:  s.AI.KnowledgeBase.GetTableDetails,
			ExportTableToKnowledgeBase: s.AI.KnowledgeBase.ExportTableToKnowledgeBase,
			ListKnowledgeBases:         s.AI.KnowledgeBase.ListKnowledgeBases,
			GetKnowledgeBase:           s.AI.KnowledgeBase.GetKnowledgeBase,
			CreateKnowledgeBase:        s.AI.KnowledgeBase.CreateKnowledgeBase,
			UpdateKnowledgeBase:        s.AI.KnowledgeBase.UpdateKnowledgeBase,
			ListChatbotKnowledgeBases:  s.AI.KnowledgeBase.ListChatbotKnowledgeBases,
			LinkKnowledgeBase:          s.AI.KnowledgeBase.LinkKnowledgeBase,
			UpdateChatbotKnowledgeBase: s.AI.KnowledgeBase.UpdateChatbotKnowledgeBase,
			UnlinkKnowledgeBase:        s.AI.KnowledgeBase.UnlinkKnowledgeBase,
			DeleteKnowledgeBase:        s.AI.KnowledgeBase.DeleteKnowledgeBase,
			ListDocuments:              s.AI.KnowledgeBase.ListDocuments,
			GetDocument:                s.AI.KnowledgeBase.GetDocument,
			AddDocument:                s.AI.KnowledgeBase.AddDocument,
			UploadDocument:             s.AI.KnowledgeBase.UploadDocument,
			DeleteDocument:             s.AI.KnowledgeBase.DeleteDocument,
			UpdateDocument:             s.AI.KnowledgeBase.UpdateDocument,
			DeleteDocumentsByFilter:    s.AI.KnowledgeBase.DeleteDocumentsByFilter,
			SearchKnowledgeBase:        s.AI.KnowledgeBase.SearchKnowledgeBase,
			DebugSearch:                s.AI.KnowledgeBase.DebugSearch,
			ListEntities:               s.AI.KnowledgeBase.ListEntities,
			SearchEntities:             s.AI.KnowledgeBase.SearchEntities,
			GetEntityRelationships:     s.AI.KnowledgeBase.GetEntityRelationships,
			GetKnowledgeGraph:          s.AI.KnowledgeBase.GetKnowledgeGraph,
			ListKBChatbots:             s.AI.KnowledgeBase.ListKnowledgeBaseChatbots,
			CreateTableExportSync:      s.AI.KnowledgeBase.CreateTableExportSync,
			ListTableExportSyncs:       s.AI.KnowledgeBase.ListTableExportSyncs,
			UpdateTableExportSync:      s.AI.KnowledgeBase.UpdateTableExportSync,
			DeleteTableExportSync:      s.AI.KnowledgeBase.DeleteTableExportSync,
			TriggerTableExportSync:     s.AI.KnowledgeBase.TriggerTableExportSync,
			GrantDocumentPermission:    s.AI.KnowledgeBase.GrantDocumentPermission,
			ListDocumentPermissions:    s.AI.KnowledgeBase.ListDocumentPermissions,
			RevokeDocumentPermission:   s.AI.KnowledgeBase.RevokeDocumentPermission,
		},
		RPC: &routes.RPCAdminDeps{
			ListRPCNamespaces:   s.RPC.Handler.ListNamespaces,
			ListProcedures:      s.RPC.Handler.ListProcedures,
			GetProcedure:        s.RPC.Handler.GetProcedure,
			UpdateProcedure:     s.RPC.Handler.UpdateProcedure,
			DeleteProcedure:     s.RPC.Handler.DeleteProcedure,
			SyncProcedures:      s.RPC.Handler.SyncProcedures,
			ListRPCExecutions:   s.RPC.Handler.ListExecutions,
			GetRPCExecution:     s.RPC.Handler.GetExecution,
			GetRPCExecutionLogs: s.RPC.Handler.GetExecutionLogs,
			CancelRPCExecution:  s.RPC.Handler.CancelExecution,
		},
		Logs: &routes.LogsAdminDeps{
			ListLogs:              s.Logging.Handler.QueryLogs,
			GetLogStats:           s.Logging.Handler.GetLogStats,
			GetExecutionLogsAdmin: s.Logging.Handler.GetExecutionLogs,
			FlushLogs:             s.Logging.Handler.FlushLogs,
			GenerateTestLogs:      s.Logging.Handler.GenerateTestLogs,
		},
		Settings: &routes.SettingsAdminDeps{
			ListSystemSettings:        s.Settings.System.ListSettings,
			GetSystemSetting:          s.Settings.System.GetSetting,
			UpdateSystemSetting:       s.Settings.System.UpdateSetting,
			DeleteSystemSetting:       s.Settings.System.DeleteSetting,
			CreateCustomSetting:       s.Settings.Custom.CreateSetting,
			ListCustomSettings:        s.Settings.Custom.ListSettings,
			CreateSecretSetting:       s.Settings.Custom.CreateSecretSetting,
			ListSecretSettings:        s.Settings.Custom.ListSecretSettings,
			GetSecretSetting:          s.Settings.Custom.GetSecretSetting,
			UpdateSecretSetting:       s.Settings.Custom.UpdateSecretSetting,
			DeleteSecretSetting:       s.Settings.Custom.DeleteSecretSetting,
			GetUserSecretValue:        s.Settings.User.GetUserSecretValue,
			GetCustomSetting:          s.Settings.Custom.GetSetting,
			UpdateCustomSetting:       s.Settings.Custom.UpdateSetting,
			DeleteCustomSetting:       s.Settings.Custom.DeleteSetting,
			GetAppSettings:            s.Settings.App.GetAppSettings,
			UpdateAppSettings:         s.Settings.App.UpdateAppSettings,
			ListEmailSettings:         s.Email.Settings.GetSettings,
			GetEmailSetting:           s.Email.Settings.GetSettings,
			UpdateEmailSetting:        s.Email.Settings.UpdateSettings,
			TestEmailSettings:         s.Email.Settings.TestSettings,
			ListEmailTemplates:        s.Email.Template.ListTemplates,
			GetEmailTemplate:          s.Email.Template.GetTemplate,
			UpdateEmailTemplate:       s.Email.Template.UpdateTemplate,
			TestEmailTemplate:         s.Email.Template.TestTemplate,
			ResetEmailTemplate:        s.Email.Template.ResetTemplate,
			GetTenantEmailSettings:    s.Email.Settings.GetSettingsForTenant,
			UpdateTenantEmailSettings: s.Email.Settings.UpdateSettingsForTenant,
			DeleteTenantEmailSetting:  s.Email.Settings.DeleteSettingForTenant,
			TestTenantEmailSettings:   s.Email.Settings.TestSettingsForTenant,
			GetCaptchaSettings:        s.Captcha.Settings.GetSettings,
			UpdateCaptchaSettings:     s.Captcha.Settings.UpdateSettings,
			GetInstanceSettings:       s.Settings.Instance.GetInstanceSettings,
			UpdateInstanceSettings:    s.Settings.Instance.UpdateInstanceSettings,
			GetOverridableSettings:    s.Settings.Instance.GetOverridableSettings,
			UpdateOverridableSettings: s.Settings.Instance.UpdateOverridableSettings,
		},
		Extensions: &routes.ExtensionsAdminDeps{
			ListExtensions:   s.Extensions.Handler.ListExtensionsForTenant,
			GetExtension:     s.Extensions.Handler.GetExtensionStatusForTenant,
			EnableExtension:  s.Extensions.Handler.EnableExtensionForTenant,
			DisableExtension: s.Extensions.Handler.DisableExtensionForTenant,
			SyncExtensions:   s.Extensions.Handler.SyncExtensions,
		},
		ExtensionsTenant: &routes.ExtensionsTenantDeps{
			ListExtensions:   s.Extensions.Handler.ListExtensionsForTenant,
			GetExtension:     s.Extensions.Handler.GetExtensionStatusForTenant,
			EnableExtension:  s.Extensions.Handler.EnableExtensionForTenant,
			DisableExtension: s.Extensions.Handler.DisableExtensionForTenant,
		},
	}
}

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
