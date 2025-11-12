package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/adminui"
	"github.com/wayli-app/fluxbase/internal/auth"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/wayli-app/fluxbase/internal/email"
	"github.com/wayli-app/fluxbase/internal/functions"
	"github.com/wayli-app/fluxbase/internal/middleware"
	"github.com/wayli-app/fluxbase/internal/realtime"
	"github.com/wayli-app/fluxbase/internal/settings"
	"github.com/wayli-app/fluxbase/internal/storage"
	"github.com/wayli-app/fluxbase/internal/webhook"
)

// Server represents the HTTP server
type Server struct {
	app                   *fiber.App
	config                *config.Config
	db                    *database.Connection
	rest                  *RESTHandler
	authHandler           *AuthHandler
	adminAuthHandler      *AdminAuthHandler
	dashboardAuthHandler  *DashboardAuthHandler
	apiKeyService         *auth.APIKeyService // Added for service-wide access
	apiKeyHandler         *APIKeyHandler
	storageHandler        *StorageHandler
	webhookHandler        *WebhookHandler
	monitoringHandler     *MonitoringHandler
	userManagementHandler *UserManagementHandler
	invitationHandler     *InvitationHandler
	ddlHandler            *DDLHandler
	oauthProviderHandler  *OAuthProviderHandler
	oauthHandler          *OAuthHandler
	systemSettingsHandler *SystemSettingsHandler
	customSettingsHandler *CustomSettingsHandler
	appSettingsHandler    *AppSettingsHandler
	emailTemplateHandler  *EmailTemplateHandler
	sqlHandler            *SQLHandler
	functionsHandler      *functions.Handler
	functionsScheduler    *functions.Scheduler
	realtimeManager       *realtime.Manager
	realtimeHandler       *realtime.RealtimeHandler
	realtimeListener      *realtime.Listener
	webhookTriggerService *webhook.TriggerService
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, db *database.Connection) *Server {
	// Create Fiber app with config
	app := fiber.New(fiber.Config{
		ServerHeader:          "Fluxbase",
		AppName:               "Fluxbase v1.0.0",
		BodyLimit:             cfg.Server.BodyLimit,
		ReadTimeout:           cfg.Server.ReadTimeout,
		WriteTimeout:          cfg.Server.WriteTimeout,
		IdleTimeout:           cfg.Server.IdleTimeout,
		DisableStartupMessage: !cfg.Debug,
		ErrorHandler:          customErrorHandler,
		Prefork:               false,
	})

	// Initialize email service
	emailService, err := email.NewService(&cfg.Email)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize email service, some features may be disabled")
		emailService = &email.NoOpService{}
	}

	// Initialize auth service
	authService := auth.NewService(db, &cfg.Auth, emailService, cfg.BaseURL)

	// Initialize API key service
	apiKeyService := auth.NewAPIKeyService(db.Pool())

	// Initialize storage service
	storageService, err := storage.NewService(&cfg.Storage)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize storage service")
	}

	// Initialize webhook service
	webhookService := webhook.NewWebhookService(db.Pool())

	// Initialize webhook trigger service (4 workers)
	webhookTriggerService := webhook.NewTriggerService(db.Pool(), webhookService, 4)

	// Initialize user management service
	userMgmtService := auth.NewUserManagementService(
		auth.NewUserRepository(db),
		auth.NewSessionRepository(db),
		auth.NewPasswordHasherWithConfig(auth.PasswordHasherConfig{MinLength: cfg.Auth.PasswordMinLen, Cost: cfg.Auth.BcryptCost}),
		emailService,
		cfg.BaseURL,
	)

	// Create handlers
	authHandler := NewAuthHandler(authService)
	// Create dashboard JWT manager first (shared between auth service and handler)
	dashboardJWTManager := auth.NewJWTManager(cfg.Auth.JWTSecret, 24*time.Hour, 168*time.Hour)
	dashboardAuthService := auth.NewDashboardAuthService(db.Pool(), dashboardJWTManager)
	systemSettingsService := auth.NewSystemSettingsService(db)
	adminAuthHandler := NewAdminAuthHandler(authService, auth.NewUserRepository(db), dashboardAuthService, systemSettingsService)
	dashboardAuthHandler := NewDashboardAuthHandler(dashboardAuthService, dashboardJWTManager)
	apiKeyHandler := NewAPIKeyHandler(apiKeyService)
	storageHandler := NewStorageHandler(storageService)
	webhookHandler := NewWebhookHandler(webhookService)
	userMgmtHandler := NewUserManagementHandler(userMgmtService, authService)
	invitationService := auth.NewInvitationService(db)
	invitationHandler := NewInvitationHandler(invitationService, dashboardAuthService)
	ddlHandler := NewDDLHandler(db)
	oauthProviderHandler := NewOAuthProviderHandler(db.Pool())
	jwtManager := auth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry, cfg.Auth.RefreshExpiry)
	baseURL := fmt.Sprintf("http://%s", cfg.Server.Address)
	oauthHandler := NewOAuthHandler(db.Pool(), authService, jwtManager, baseURL)
	systemSettingsHandler := NewSystemSettingsHandler(systemSettingsService)
	customSettingsService := settings.NewCustomSettingsService(db)
	customSettingsHandler := NewCustomSettingsHandler(customSettingsService)
	appSettingsHandler := NewAppSettingsHandler(systemSettingsService)
	emailTemplateHandler := NewEmailTemplateHandler(db)
	sqlHandler := NewSQLHandler(db.Pool())
	functionsHandler := functions.NewHandler(db.Pool(), cfg.Functions.FunctionsDir)
	functionsScheduler := functions.NewScheduler(db.Pool())
	functionsHandler.SetScheduler(functionsScheduler)

	// Create realtime components
	realtimeManager := realtime.NewManager(context.Background())
	realtimeAuthAdapter := realtime.NewAuthServiceAdapter(authService)
	realtimeSubManager := realtime.NewSubscriptionManager(db.Pool())
	realtimeHandler := realtime.NewRealtimeHandler(realtimeManager, realtimeAuthAdapter, realtimeSubManager)
	realtimeListener := realtime.NewListener(db.Pool(), realtimeHandler, realtimeSubManager)

	// Create monitoring handler
	monitoringHandler := NewMonitoringHandler(db.Pool(), realtimeHandler, storageService.Provider)

	// Create server instance
	server := &Server{
		app:                   app,
		config:                cfg,
		db:                    db,
		rest:                  NewRESTHandler(db, NewQueryParser()),
		authHandler:           authHandler,
		adminAuthHandler:      adminAuthHandler,
		dashboardAuthHandler:  dashboardAuthHandler,
		apiKeyService:         apiKeyService, // Added for service-wide access
		apiKeyHandler:         apiKeyHandler,
		storageHandler:        storageHandler,
		webhookHandler:        webhookHandler,
		monitoringHandler:     monitoringHandler,
		userManagementHandler: userMgmtHandler,
		invitationHandler:     invitationHandler,
		ddlHandler:            ddlHandler,
		oauthProviderHandler:  oauthProviderHandler,
		oauthHandler:          oauthHandler,
		systemSettingsHandler: systemSettingsHandler,
		customSettingsHandler: customSettingsHandler,
		appSettingsHandler:    appSettingsHandler,
		emailTemplateHandler:  emailTemplateHandler,
		sqlHandler:            sqlHandler,
		functionsHandler:      functionsHandler,
		functionsScheduler:    functionsScheduler,
		realtimeManager:       realtimeManager,
		realtimeHandler:       realtimeHandler,
		realtimeListener:      realtimeListener,
		webhookTriggerService: webhookTriggerService,
	}

	// Start realtime listener
	if err := realtimeListener.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start realtime listener")
	}

	// Start edge functions scheduler
	if err := functionsScheduler.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start edge functions scheduler")
	}

	// Start webhook trigger service
	if err := webhookTriggerService.Start(context.Background()); err != nil {
		log.Error().Err(err).Msg("Failed to start webhook trigger service")
	}

	// Setup middlewares
	log.Debug().Msg("Setting up middlewares")
	server.setupMiddlewares()

	// Setup routes
	log.Debug().Msg("Setting up routes")
	server.setupRoutes()

	log.Debug().Msg("Server initialization complete")
	return server
}

// setupMiddlewares sets up global middlewares
func (s *Server) setupMiddlewares() {
	// Request ID middleware - must be first for tracing
	log.Debug().Msg("Adding requestid middleware")
	s.app.Use(requestid.New())

	// Security headers middleware - protect against common attacks
	// Apply different CSP for admin UI (needs Google Fonts) vs API routes
	log.Debug().Msg("Adding security headers middleware")
	s.app.Use(func(c *fiber.Ctx) error {
		// Apply relaxed CSP for admin UI
		if strings.HasPrefix(c.Path(), "/admin") {
			return middleware.AdminUISecurityHeaders()(c)
		}
		// Apply strict CSP for all other routes
		return middleware.SecurityHeaders()(c)
	})

	// Logger middleware
	log.Debug().Msg("Adding logger middleware")
	if s.config.Debug {
		s.app.Use(logger.New(logger.Config{
			Format: "[${time}] ${status} - ${latency} ${method} ${path} ${error}\n",
		}))
	}

	// Recover middleware - catch panics
	log.Debug().Msg("Adding recover middleware")
	s.app.Use(recover.New(recover.Config{
		EnableStackTrace: s.config.Debug,
	}))

	// CORS middleware
	log.Debug().Msg("Adding CORS middleware")
	s.app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173,http://localhost:8080", // Can't use "*" with AllowCredentials
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Request-ID, X-CSRF-Token, Prefer",
		ExposeHeaders:    "Content-Range, Content-Encoding, Content-Length, X-Request-ID, X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset",
		AllowCredentials: true, // Required for CSRF tokens
		MaxAge:           300,
	}))

	// Global rate limiting - 100 requests per minute per IP
	// Note: Global rate limiting is disabled by default. Enable via config if needed.
	// To enable: Set ENABLE_GLOBAL_RATE_LIMIT=true in environment
	if s.config.Security.EnableGlobalRateLimit {
		log.Info().Msg("Enabling global rate limiter (100 req/min per IP)")
		s.app.Use(middleware.GlobalAPILimiter())
	}

	// Compression middleware
	s.app.Use(compress.New(compress.Config{
		Level: compress.LevelDefault,
	}))
}

// setupRoutes sets up all routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.app.Get("/health", s.handleHealth)

	// API v1 routes - versioned for future compatibility
	v1 := s.app.Group("/api/v1")

	// Setup RLS middleware if enabled (before REST API routes)
	rlsConfig := middleware.RLSConfig{
		DB:      s.db,
		Enabled: s.config.Auth.EnableRLS,
	}

	// REST API routes (auto-generated from database schema)
	// Optional authentication (JWT, API key, or service key) - allows anonymous access with RLS
	// Followed by RLS middleware to set PostgreSQL session variables (role 'anon' if unauthenticated)
	rest := v1.Group("/tables",
		middleware.OptionalAuthOrServiceKey(s.authHandler.authService, s.apiKeyService, s.db.Pool()),
		middleware.RLSMiddleware(rlsConfig),
	)
	s.setupRESTRoutes(rest)

	// RPC routes (auto-generated from database functions)
	// Require authentication (JWT, API key, or service key)
	// followed by RLS middleware to set PostgreSQL session variables
	rpc := v1.Group("/rpc",
		middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.apiKeyService, s.db.Pool()),
		middleware.RLSMiddleware(rlsConfig),
	)
	s.setupRPCRoutes(rpc)

	// Auth routes
	auth := v1.Group("/auth")
	s.setupAuthRoutes(auth)

	// Dashboard auth routes (separate from application auth)
	s.dashboardAuthHandler.RegisterRoutes(s.app)

	// API Keys routes - require authentication
	s.apiKeyHandler.RegisterRoutes(s.app, s.authHandler.authService, s.apiKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)

	// Webhook routes - require authentication
	s.webhookHandler.RegisterRoutes(s.app, s.authHandler.authService, s.apiKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)

	// Monitoring routes - require authentication
	s.monitoringHandler.RegisterRoutes(s.app, s.authHandler.authService, s.apiKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)

	// Edge functions routes - require authentication by default, but per-function config can override
	s.functionsHandler.RegisterRoutes(s.app, s.authHandler.authService, s.apiKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)

	// Storage routes - require authentication
	storage := v1.Group("/storage",
		middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.apiKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
	)
	s.setupStorageRoutes(storage)

	// Realtime WebSocket endpoint (not versioned as it's WebSocket)
	// WebSocket validates auth internally, but make it required
	s.app.Get("/realtime", s.realtimeHandler.HandleWebSocket)

	// Realtime stats endpoint - require authentication
	s.app.Get("/api/v1/realtime/stats",
		middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.apiKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
		s.handleRealtimeStats,
	)

	// Realtime broadcast endpoint - require authentication
	s.app.Post("/api/v1/realtime/broadcast",
		middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.apiKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
		s.handleRealtimeBroadcast,
	)

	// Admin routes (optional)
	admin := v1.Group("/admin")
	s.setupAdminRoutes(admin)

	// Public invitation routes (no auth required)
	invitations := v1.Group("/invitations")
	s.setupPublicInvitationRoutes(invitations)

	// Admin UI (embedded React app)
	adminUI := adminui.New()
	adminUI.RegisterRoutes(s.app)

	// OpenAPI specification
	openAPIHandler := NewOpenAPIHandler(s.db)
	openAPIHandler.RegisterRoutes(s.app)

	// 404 handler
	s.app.Use(func(c *fiber.Ctx) error {
		return c.Status(404).JSON(fiber.Map{
			"error": "Not Found",
			"path":  c.Path(),
		})
	})
}

// setupRESTRoutes sets up auto-generated REST routes
func (s *Server) setupRESTRoutes(router fiber.Router) {
	// Initialize REST routes on startup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get all schemas to register tables from
	schemas, err := s.db.Inspector().GetSchemas(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get schemas for REST API")
		return
	}

	// Register tables from all schemas (excluding system and sensitive schemas)
	for _, schema := range schemas {
		// Skip system schemas and sensitive schemas (dashboard, auth, _fluxbase)
		// Sensitive schemas contain user credentials and should not be exposed via REST API
		// They are only accessible through protected admin endpoints
		// _fluxbase is an internal schema for migration tracking
		if schema == "information_schema" || schema == "pg_catalog" || schema == "pg_toast" ||
			schema == "dashboard" || schema == "auth" || schema == "_fluxbase" {
			continue
		}

		// Get all tables from this schema
		tables, err := s.db.Inspector().GetAllTables(ctx, schema)
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get tables from schema")
			continue
		}

		// Register dynamic routes for each table
		for _, table := range tables {
			s.rest.RegisterTableRoutes(router, table)
		}

		// Get all views and register as read-only endpoints
		views, err := s.db.Inspector().GetAllViews(ctx, schema)
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get views from schema")
		} else {
			for _, view := range views {
				s.rest.RegisterViewRoutes(router, view)
			}
		}
	}

	// Metadata endpoint
	router.Get("/", s.rest.HandleGetTables)
}

// setupRPCRoutes sets up auto-generated RPC routes for database functions
func (s *Server) setupRPCRoutes(router fiber.Router) {
	// Initialize RPC handler
	rpcHandler := NewRPCHandler(s.db)

	// Register all function routes
	if err := rpcHandler.RegisterRoutes(router); err != nil {
		log.Error().Err(err).Msg("Failed to register RPC routes")
	}
}

// setupAuthRoutes sets up authentication routes
func (s *Server) setupAuthRoutes(router fiber.Router) {
	// Import rate limiters from middleware package
	rateLimiters := map[string]fiber.Handler{
		"signup":         middleware.AuthSignupLimiter(),
		"login":          middleware.AuthLoginLimiter(),
		"refresh":        middleware.AuthRefreshLimiter(),
		"magiclink":      middleware.AuthMagicLinkLimiter(),
		"password_reset": middleware.AuthPasswordResetLimiter(),
	}

	// Use the auth handler's RegisterRoutes method with rate limiters
	// Pass the router (which is /api/v1/auth) instead of the whole app
	s.authHandler.RegisterRoutes(router, rateLimiters)

	// OAuth routes
	router.Get("/oauth/providers", s.oauthHandler.ListEnabledProviders)
	router.Get("/oauth/:provider/authorize", s.oauthHandler.Authorize)
	router.Get("/oauth/:provider/callback", s.oauthHandler.Callback)
}

// setupStorageRoutes sets up storage routes
func (s *Server) setupStorageRoutes(router fiber.Router) {
	// Bucket management
	router.Get("/buckets", s.storageHandler.ListBuckets)
	router.Post("/buckets/:bucket", s.storageHandler.CreateBucket)
	router.Delete("/buckets/:bucket", s.storageHandler.DeleteBucket)

	// File operations
	router.Post("/:bucket/*", s.storageHandler.UploadFile)   // Upload file
	router.Get("/:bucket/*", s.storageHandler.DownloadFile)  // Download file
	router.Delete("/:bucket/*", s.storageHandler.DeleteFile) // Delete file
	router.Head("/:bucket/*", s.storageHandler.GetFileInfo)  // Get file metadata

	// List files in bucket
	router.Get("/:bucket", s.storageHandler.ListFiles)

	// Multipart upload
	router.Post("/:bucket/multipart", s.storageHandler.MultipartUpload)

	// Signed URLs (for S3-compatible storage)
	router.Post("/:bucket/*/signed-url", s.storageHandler.GenerateSignedURL)
}

// setupAdminRoutes sets up admin routes
func (s *Server) setupAdminRoutes(router fiber.Router) {
	// Public admin auth routes (no authentication required)
	router.Get("/setup/status", s.adminAuthHandler.GetSetupStatus)
	router.Post("/setup", middleware.AdminSetupLimiter(), s.adminAuthHandler.InitialSetup)
	router.Post("/login", middleware.AdminLoginLimiter(), s.adminAuthHandler.AdminLogin)
	router.Post("/refresh", s.adminAuthHandler.AdminRefreshToken)

	// Protected admin routes (require authentication from either auth.users or dashboard.users)
	// UnifiedAuthMiddleware accepts tokens from both authentication systems
	unifiedAuth := UnifiedAuthMiddleware(s.authHandler.authService, s.dashboardAuthHandler.jwtManager)

	router.Post("/logout", unifiedAuth, s.adminAuthHandler.AdminLogout)
	router.Get("/me", unifiedAuth, s.adminAuthHandler.GetCurrentAdmin)

	// Admin panel routes (require admin or dashboard_admin role)
	router.Get("/tables", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.handleGetTables)
	router.Get("/schemas", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.handleGetSchemas)
	router.Post("/query", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.handleExecuteQuery)

	// DDL routes (schema and table management) - require admin or dashboard_admin role
	router.Post("/schemas", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.ddlHandler.CreateSchema)
	router.Post("/tables", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.ddlHandler.CreateTable)
	router.Delete("/tables/:schema/:table", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.ddlHandler.DeleteTable)

	// OAuth provider management routes (require admin or dashboard_admin role)
	router.Get("/oauth/providers", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.oauthProviderHandler.ListOAuthProviders)
	router.Get("/oauth/providers/:id", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.oauthProviderHandler.GetOAuthProvider)
	router.Post("/oauth/providers", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.oauthProviderHandler.CreateOAuthProvider)
	router.Put("/oauth/providers/:id", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.oauthProviderHandler.UpdateOAuthProvider)
	router.Delete("/oauth/providers/:id", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.oauthProviderHandler.DeleteOAuthProvider)

	// Auth settings routes (require admin or dashboard_admin role)
	router.Get("/auth/settings", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.oauthProviderHandler.GetAuthSettings)
	router.Put("/auth/settings", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.oauthProviderHandler.UpdateAuthSettings)

	// System settings routes (require admin or dashboard_admin role)
	router.Get("/system/settings", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.systemSettingsHandler.ListSettings)
	router.Get("/system/settings/:key", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.systemSettingsHandler.GetSetting)
	router.Put("/system/settings/:key", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.systemSettingsHandler.UpdateSetting)
	router.Delete("/system/settings/:key", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.systemSettingsHandler.DeleteSetting)

	// Custom settings routes (require admin or dashboard_admin role)
	router.Post("/settings/custom", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.customSettingsHandler.CreateSetting)
	router.Get("/settings/custom", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.customSettingsHandler.ListSettings)
	router.Get("/settings/custom/:key", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.customSettingsHandler.GetSetting)
	router.Put("/settings/custom/:key", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.customSettingsHandler.UpdateSetting)
	router.Delete("/settings/custom/:key", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.customSettingsHandler.DeleteSetting)

	// App settings routes (require admin or dashboard_admin role)
	router.Get("/app/settings", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.appSettingsHandler.GetAppSettings)
	router.Put("/app/settings", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.appSettingsHandler.UpdateAppSettings)

	// Email template routes (require admin or dashboard_admin role)
	router.Get("/email/templates", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.emailTemplateHandler.ListTemplates)
	router.Get("/email/templates/:type", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.emailTemplateHandler.GetTemplate)
	router.Put("/email/templates/:type", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.emailTemplateHandler.UpdateTemplate)
	router.Post("/email/templates/:type/reset", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.emailTemplateHandler.ResetTemplate)
	router.Post("/email/templates/:type/test", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.emailTemplateHandler.TestTemplate)

	// User management routes (require admin or dashboard_admin role)
	router.Get("/users", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.userManagementHandler.ListUsers)
	router.Post("/users/invite", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.userManagementHandler.InviteUser)
	router.Delete("/users/:id", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.userManagementHandler.DeleteUser)
	router.Patch("/users/:id/role", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.userManagementHandler.UpdateUserRole)
	router.Post("/users/:id/reset-password", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.userManagementHandler.ResetUserPassword)

	// Invitation management routes (require admin or dashboard_admin role)
	router.Post("/invitations", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.invitationHandler.CreateInvitation)
	router.Get("/invitations", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.invitationHandler.ListInvitations)
	router.Delete("/invitations/:token", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.invitationHandler.RevokeInvitation)

	// SQL Editor route (require dashboard_admin role only)
	router.Post("/sql/execute", unifiedAuth, RequireRole("dashboard_admin"), s.sqlHandler.ExecuteSQL)

	// Functions management routes (require admin or dashboard_admin role)
	router.Post("/functions/reload", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.functionsHandler.ReloadFunctions)
}

// setupPublicInvitationRoutes sets up public invitation routes (no auth required)
func (s *Server) setupPublicInvitationRoutes(router fiber.Router) {
	// Public invitation routes (no authentication required)
	router.Get("/:token/validate", s.invitationHandler.ValidateInvitation)
	router.Post("/:token/accept", s.invitationHandler.AcceptInvitation)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(c *fiber.Ctx) error {
	// Check database health
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	dbHealthy := true
	if err := s.db.Health(ctx); err != nil {
		dbHealthy = false
		log.Error().Err(err).Msg("Database health check failed")
	}

	status := "ok"
	httpStatus := fiber.StatusOK
	if !dbHealthy {
		status = "degraded"
		httpStatus = fiber.StatusServiceUnavailable
	}

	return c.Status(httpStatus).JSON(fiber.Map{
		"status": status,
		"services": fiber.Map{
			"database": dbHealthy,
			"realtime": s.config.Realtime.Enabled,
		},
		"timestamp": time.Now().UTC(),
	})
}

func (s *Server) handleGetTables(c *fiber.Ctx) error {
	ctx := c.Context()

	// Check if schema query parameter is provided
	schemaParam := c.Query("schema")

	var schemasToQuery []string

	if schemaParam != "" {
		// If schema parameter provided, query only that schema
		schemasToQuery = []string{schemaParam}
	} else {
		// Otherwise, get all schemas (backward compatible behavior)
		schemas, err := s.db.Inspector().GetSchemas(ctx)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Filter out system schemas
		for _, schema := range schemas {
			// Skip system schemas only
			if schema == "information_schema" || schema == "pg_catalog" || schema == "pg_toast" {
				continue
			}
			schemasToQuery = append(schemasToQuery, schema)
		}
	}

	// Collect tables from requested schema(s)
	var allTables []database.TableInfo
	for _, schema := range schemasToQuery {
		tables, err := s.db.Inspector().GetAllTables(ctx, schema)
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get tables from schema")
			continue
		}

		allTables = append(allTables, tables...)
	}

	return c.JSON(allTables)
}

func (s *Server) handleGetSchemas(c *fiber.Ctx) error {
	ctx := c.Context()
	schemas, err := s.db.Inspector().GetSchemas(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Filter out system schemas
	var userSchemas []string
	for _, schema := range schemas {
		if schema != "information_schema" && schema != "pg_catalog" && schema != "pg_toast" {
			userSchemas = append(userSchemas, schema)
		}
	}

	return c.JSON(userSchemas)
}

func (s *Server) handleExecuteQuery(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Execute query endpoint - to be implemented"})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.app.Listen(s.config.Server.Address)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Stop realtime listener
	if s.realtimeListener != nil {
		s.realtimeListener.Stop()
	}

	// Stop edge functions scheduler
	if s.functionsScheduler != nil {
		s.functionsScheduler.Stop()
	}

	// Stop webhook trigger service
	if s.webhookTriggerService != nil {
		s.webhookTriggerService.Stop()
	}

	return s.app.ShutdownWithContext(ctx)
}

// App returns the underlying Fiber app instance for testing
func (s *Server) App() *fiber.App {
	return s.app
}

// GetStorageService returns the storage service from the storage handler
func (s *Server) GetStorageService() *storage.Service {
	if s.storageHandler == nil {
		return nil
	}
	return s.storageHandler.storage
}

// GetWebhookTriggerService returns the webhook trigger service for testing
func (s *Server) GetWebhookTriggerService() *webhook.TriggerService {
	return s.webhookTriggerService
}

// customErrorHandler handles errors globally
func customErrorHandler(c *fiber.Ctx, err error) error {
	// Default to 500 status code
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	// Check if it's a Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Log error
	if code >= 500 {
		log.Error().Err(err).Str("path", c.Path()).Msg("Server error")
	}

	// Return JSON error response
	return c.Status(code).JSON(fiber.Map{
		"error": message,
		"code":  code,
	})
}

// handleRealtimeStats returns realtime statistics
func (s *Server) handleRealtimeStats(c *fiber.Ctx) error {
	stats := s.realtimeHandler.GetDetailedStats()
	return c.JSON(stats)
}

// BroadcastRequest represents a broadcast request
type BroadcastRequest struct {
	Channel string      `json:"channel"`
	Message interface{} `json:"message"`
}

// handleRealtimeBroadcast broadcasts a message to a channel
func (s *Server) handleRealtimeBroadcast(c *fiber.Ctx) error {
	var req BroadcastRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Channel == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Channel is required",
		})
	}

	// TODO: Implement broadcast functionality
	// s.realtimeHandler.Broadcast(req.Channel, req.Message)

	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"error":   "Broadcast functionality not yet implemented",
		"channel": req.Channel,
	})
}
