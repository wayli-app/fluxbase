package api

import (
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
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
	"github.com/wayli-app/fluxbase/internal/jobs"
	"github.com/wayli-app/fluxbase/internal/middleware"
	"github.com/wayli-app/fluxbase/internal/migrations"
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
	settingsHandler       *SettingsHandler
	emailTemplateHandler  *EmailTemplateHandler
	sqlHandler            *SQLHandler
	functionsHandler      *functions.Handler
	functionsScheduler    *functions.Scheduler
	jobsHandler           *jobs.Handler
	jobsManager           *jobs.Manager
	migrationsHandler     *migrations.Handler
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

	// Ensure default buckets exist
	if err := storageService.EnsureDefaultBuckets(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed to ensure default buckets")
	}

	// Initialize webhook service
	webhookService := webhook.NewWebhookService(db)

	// Initialize webhook trigger service (4 workers)
	webhookTriggerService := webhook.NewTriggerService(db, webhookService, 4)

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
	dashboardAuthService := auth.NewDashboardAuthService(db, dashboardJWTManager, cfg.Auth.TOTPIssuer)
	systemSettingsService := auth.NewSystemSettingsService(db)
	adminAuthHandler := NewAdminAuthHandler(authService, auth.NewUserRepository(db), dashboardAuthService, systemSettingsService, cfg)
	dashboardAuthHandler := NewDashboardAuthHandler(dashboardAuthService, dashboardJWTManager)
	apiKeyHandler := NewAPIKeyHandler(apiKeyService)
	storageHandler := NewStorageHandler(storageService, db)
	webhookHandler := NewWebhookHandler(webhookService)
	userMgmtHandler := NewUserManagementHandler(userMgmtService, authService)
	invitationService := auth.NewInvitationService(db)
	invitationHandler := NewInvitationHandler(invitationService, dashboardAuthService, cfg.BaseURL)
	ddlHandler := NewDDLHandler(db)
	oauthProviderHandler := NewOAuthProviderHandler(db.Pool(), authService.GetSettingsCache())
	jwtManager := auth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry, cfg.Auth.RefreshExpiry)
	baseURL := fmt.Sprintf("http://%s", cfg.Server.Address)
	oauthHandler := NewOAuthHandler(db.Pool(), authService, jwtManager, baseURL)
	systemSettingsHandler := NewSystemSettingsHandler(systemSettingsService, authService.GetSettingsCache())
	customSettingsService := settings.NewCustomSettingsService(db)
	customSettingsHandler := NewCustomSettingsHandler(customSettingsService)
	appSettingsHandler := NewAppSettingsHandler(systemSettingsService, authService.GetSettingsCache())
	settingsHandler := NewSettingsHandler(db)
	emailTemplateHandler := NewEmailTemplateHandler(db)
	sqlHandler := NewSQLHandler(db.Pool())
	functionsHandler := functions.NewHandler(db, cfg.Functions.FunctionsDir, cfg.CORS)
	functionsScheduler := functions.NewScheduler(db)
	functionsHandler.SetScheduler(functionsScheduler)

	// Only create jobs components if jobs are enabled
	var jobsManager *jobs.Manager
	var jobsHandler *jobs.Handler
	if cfg.Jobs.Enabled {
		// Determine public URL for jobs SDK client
		jobsPublicURL := cfg.BaseURL
		if jobsPublicURL == "" {
			// Fallback to server address
			jobsPublicURL = "http://localhost" + cfg.Server.Address
		}
		log.Info().
			Str("jobs_public_url", jobsPublicURL).
			Bool("jwt_secret_set", cfg.Auth.JWTSecret != "").
			Msg("Initializing jobs manager with SDK credentials")
		jobsManager = jobs.NewManager(&cfg.Jobs, db, cfg.Auth.JWTSecret, jobsPublicURL)
		var err error
		jobsHandler, err = jobs.NewHandler(db, &cfg.Jobs, jobsManager)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize jobs handler")
		}
	}

	migrationsHandler := migrations.NewHandler(db)

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
		rest:                  NewRESTHandler(db, NewQueryParser(cfg)),
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
		settingsHandler:       settingsHandler,
		emailTemplateHandler:  emailTemplateHandler,
		sqlHandler:            sqlHandler,
		functionsHandler:      functionsHandler,
		functionsScheduler:    functionsScheduler,
		jobsHandler:           jobsHandler,
		jobsManager:           jobsManager,
		migrationsHandler:     migrationsHandler,
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

	// Start jobs manager
	if cfg.Jobs.Enabled && jobsManager != nil {
		workerCount := cfg.Jobs.EmbeddedWorkerCount
		if workerCount <= 0 {
			workerCount = 4 // Default to 4 workers if not configured
		}
		if err := jobsManager.Start(context.Background(), workerCount); err != nil {
			log.Error().Err(err).Msg("Failed to start jobs manager")
		} else {
			log.Info().Int("workers", workerCount).Msg("Jobs manager started successfully")
		}
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
	// Note: AllowCredentials cannot be used with AllowOrigins="*" per CORS spec
	// If AllowOrigins is "*", we must disable credentials
	corsCredentials := s.config.CORS.AllowCredentials
	if s.config.CORS.AllowedOrigins == "*" && corsCredentials {
		log.Warn().Msg("CORS: AllowCredentials disabled because AllowOrigins is '*' (not allowed per CORS spec)")
		corsCredentials = false
	}
	log.Debug().
		Str("origins", s.config.CORS.AllowedOrigins).
		Bool("credentials", corsCredentials).
		Msg("Adding CORS middleware")
	s.app.Use(cors.New(cors.Config{
		AllowOrigins:     s.config.CORS.AllowedOrigins,
		AllowMethods:     s.config.CORS.AllowedMethods,
		AllowHeaders:     s.config.CORS.AllowedHeaders,
		ExposeHeaders:    s.config.CORS.ExposedHeaders,
		AllowCredentials: corsCredentials,
		MaxAge:           s.config.CORS.MaxAge,
	}))
	log.Debug().Msg("CORS middleware added")

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

	// Setup RLS middleware (before REST API routes)
	rlsConfig := middleware.RLSConfig{
		DB: s.db,
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

	// Public settings routes - optional authentication with RLS support
	// These routes respect app.settings RLS policies based on is_public and is_secret flags
	settings := v1.Group("/settings", OptionalAuthMiddleware(s.authHandler.authService))
	settings.Get("/:key", s.settingsHandler.GetSetting)
	settings.Post("/batch", s.settingsHandler.GetSettings)

	// Dashboard auth routes (separate from application auth)
	s.dashboardAuthHandler.RegisterRoutes(s.app)

	// API Keys routes - require authentication
	s.apiKeyHandler.RegisterRoutes(s.app, s.authHandler.authService, s.apiKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)

	// Webhook routes - require authentication
	s.webhookHandler.RegisterRoutes(s.app, s.authHandler.authService, s.apiKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)

	// Monitoring routes - require authentication
	s.monitoringHandler.RegisterRoutes(s.app, s.authHandler.authService, s.apiKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)

	// Edge functions routes - require authentication by default, but per-function config can override
	// Protected by feature flag middleware
	s.functionsHandler.RegisterRoutes(s.app, s.authHandler.authService, s.apiKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)

	// Jobs routes - require authentication
	// Protected by feature flag middleware
	if s.jobsHandler != nil {
		s.jobsHandler.RegisterRoutes(s.app, s.authHandler.authService, s.apiKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)
		s.jobsHandler.RegisterAdminRoutes(s.app) // Admin routes for job management
	}

	// Storage routes - optional authentication (allows unauthenticated access to public buckets)
	// Protected by feature flag middleware
	storage := v1.Group("/storage",
		middleware.RequireStorageEnabled(s.authHandler.authService.GetSettingsCache()),
		middleware.OptionalAuthOrServiceKey(s.authHandler.authService, s.apiKeyService, s.db.Pool()),
	)
	s.setupStorageRoutes(storage)

	// Realtime WebSocket endpoint (not versioned as it's WebSocket)
	// WebSocket validates auth internally, but make it required
	// Protected by feature flag middleware
	s.app.Get("/realtime",
		middleware.RequireRealtimeEnabled(s.authHandler.authService.GetSettingsCache()),
		s.realtimeHandler.HandleWebSocket,
	)

	// Realtime stats endpoint - require authentication
	// Protected by feature flag middleware
	s.app.Get("/api/v1/realtime/stats",
		middleware.RequireRealtimeEnabled(s.authHandler.authService.GetSettingsCache()),
		middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.apiKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
		s.handleRealtimeStats,
	)

	// Realtime broadcast endpoint - require authentication
	// Protected by feature flag middleware
	s.app.Post("/api/v1/realtime/broadcast",
		middleware.RequireRealtimeEnabled(s.authHandler.authService.GetSettingsCache()),
		middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.apiKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
		s.handleRealtimeBroadcast,
	)

	// Admin routes and UI (only enabled if setup token is configured)
	if s.config.Security.SetupToken != "" {
		admin := v1.Group("/admin")
		s.setupAdminRoutes(admin)

		// Public invitation routes (no auth required)
		invitations := v1.Group("/invitations")
		s.setupPublicInvitationRoutes(invitations)

		// Admin UI (embedded React app)
		adminUI := adminui.New()
		adminUI.RegisterRoutes(s.app)
	} else {
		log.Warn().Msg("Admin dashboard is disabled. Set FLUXBASE_SECURITY_SETUP_TOKEN to enable admin functionality.")
	}

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

	// Register tables from all schemas (excluding system schemas only)
	for _, schema := range schemas {
		// Skip system schemas and internal migration tracking schema
		// Note: auth and dashboard schemas are included and protected by RLS/authentication
		if schema == "information_schema" || schema == "pg_catalog" || schema == "pg_toast" ||
			schema == "_fluxbase" {
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
		"otp":            middleware.AuthMagicLinkLimiter(), // Use same rate limit as magic link
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
	router.Put("/buckets/:bucket", s.storageHandler.UpdateBucketSettings)
	router.Delete("/buckets/:bucket", s.storageHandler.DeleteBucket)

	// List files in bucket (must come before /:bucket/*)
	router.Get("/:bucket", s.storageHandler.ListFiles)

	// Multipart upload (must come before /:bucket/*)
	router.Post("/:bucket/multipart", s.storageHandler.MultipartUpload)

	// File sharing (must come before /:bucket/* to avoid matching generic routes)
	router.Post("/:bucket/*/share", s.storageHandler.ShareObject)            // Share file with user
	router.Delete("/:bucket/*/share/:user_id", s.storageHandler.RevokeShare) // Revoke share
	router.Get("/:bucket/*/shares", s.storageHandler.ListShares)             // List shares

	// Signed URLs (for S3-compatible storage, must come before /:bucket/*)
	router.Post("/:bucket/*/signed-url", s.storageHandler.GenerateSignedURL)

	// File operations (generic wildcard routes - must come LAST)
	router.Post("/:bucket/*", s.storageHandler.UploadFile)   // Upload file
	router.Get("/:bucket/*", s.storageHandler.DownloadFile)  // Download file
	router.Head("/:bucket/*", s.storageHandler.DownloadFile) // HEAD delegates to GetFileInfo for Content-Length
	router.Delete("/:bucket/*", s.storageHandler.DeleteFile) // Delete file
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
	router.Get("/tables/:schema/:table", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.handleGetTableSchema)
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
	router.Get("/system/settings/*", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.systemSettingsHandler.GetSetting)
	router.Put("/system/settings/*", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.systemSettingsHandler.UpdateSetting)
	router.Delete("/system/settings/*", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.systemSettingsHandler.DeleteSetting)

	// Custom settings routes (require admin or dashboard_admin role)
	router.Post("/settings/custom", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.customSettingsHandler.CreateSetting)
	router.Get("/settings/custom", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.customSettingsHandler.ListSettings)
	router.Get("/settings/custom/*", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.customSettingsHandler.GetSetting)
	router.Put("/settings/custom/*", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.customSettingsHandler.UpdateSetting)
	router.Delete("/settings/custom/*", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.customSettingsHandler.DeleteSetting)

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

	// Functions management routes (require admin, dashboard_admin, or service_role)
	router.Post("/functions/reload", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.functionsHandler.ReloadFunctions)
	router.Get("/functions/namespaces", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.functionsHandler.ListNamespaces)
	// Functions sync - with IP allowlist protection (similar to migrations)
	router.Post("/functions/sync",
		middleware.RequireSyncIPAllowlist(s.config.Functions.SyncAllowedIPRanges, "functions"),
		unifiedAuth,
		RequireRole("admin", "dashboard_admin", "service_role"),
		s.functionsHandler.SyncFunctions,
	)

	// Jobs management routes (require admin, dashboard_admin, or service_role)
	// Only register if jobs are enabled
	if s.jobsHandler != nil {
		// Jobs sync - with IP allowlist protection (similar to migrations)
		router.Post("/jobs/sync",
			middleware.RequireSyncIPAllowlist(s.config.Jobs.SyncAllowedIPRanges, "jobs"),
			unifiedAuth,
			RequireRole("admin", "dashboard_admin", "service_role"),
			s.jobsHandler.SyncJobs,
		)
		router.Get("/jobs/namespaces", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.jobsHandler.ListNamespaces)
		router.Get("/jobs/functions", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.jobsHandler.ListJobFunctions)
		router.Get("/jobs/functions/:namespace/:name", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.jobsHandler.GetJobFunction)
		router.Delete("/jobs/functions/:namespace/:name", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.jobsHandler.DeleteJobFunction)
		router.Get("/jobs/stats", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.jobsHandler.GetJobStats)
		router.Get("/jobs/workers", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.jobsHandler.ListWorkers)

		// Queue operations - list and individual job management
		router.Get("/jobs/queue", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.jobsHandler.ListAllJobs)
		router.Get("/jobs/queue/:id", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.jobsHandler.GetJobAdmin)
		router.Post("/jobs/queue/:id/terminate", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.jobsHandler.TerminateJob)
		router.Post("/jobs/queue/:id/cancel", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.jobsHandler.CancelJobAdmin)
		router.Post("/jobs/queue/:id/retry", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.jobsHandler.RetryJobAdmin)
		router.Post("/jobs/queue/:id/resubmit", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.jobsHandler.ResubmitJobAdmin)
	}

	// Migrations routes (require service key authentication with enhanced security)
	// Only registered if migrations API is enabled in config
	if s.config.Migrations.Enabled {
		// Build secure middleware stack for migrations API
		// Layer 1: Feature flag check
		// Layer 2: IP allowlist (only allow app container)
		// Layer 3: Service key authentication (no JWT/API keys)
		// Layer 4: Scope validation (migrations:execute)
		// Layer 5: Rate limiting (10 req/hour)
		// Layer 6: Audit logging
		migrationsAuth := []fiber.Handler{
			middleware.RequireMigrationsEnabled(&s.config.Migrations),
			middleware.RequireMigrationsIPAllowlist(&s.config.Migrations),
			middleware.RequireServiceKeyOnly(s.db.Pool(), s.authHandler.authService),
			middleware.RequireMigrationScope(),
			middleware.MigrationAPILimiter(),
			middleware.MigrationsAuditLog(),
		}

		s.migrationsHandler.RegisterRoutes(s.app, migrationsAuth...)

		log.Info().
			Bool("enabled", s.config.Migrations.Enabled).
			Strs("allowed_ips", s.config.Migrations.AllowedIPRanges).
			Bool("require_service_key", s.config.Migrations.RequireServiceKey).
			Msg("Migrations API registered with enhanced security controls")
	} else {
		log.Info().Msg("Migrations API disabled")
	}

	// Schema refresh endpoint (require admin, dashboard_admin, or service_role)
	router.Post("/schema/refresh", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.handleRefreshSchema)
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

func (s *Server) handleGetTableSchema(c *fiber.Ctx) error {
	ctx := c.Context()
	schema := c.Params("schema")
	table := c.Params("table")

	if schema == "" || table == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Schema and table parameters are required",
		})
	}

	// Get table information including column details
	tableInfo, err := s.db.Inspector().GetTableInfo(ctx, schema, table)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fmt.Sprintf("Table not found: %s.%s", schema, table),
		})
	}

	return c.JSON(tableInfo)
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

// handleRefreshSchema refreshes the REST API schema cache by re-registering all table routes
func (s *Server) handleRefreshSchema(c *fiber.Ctx) error {
	log.Info().Msg("Schema refresh requested - triggering graceful server restart")

	// Send response before shutting down
	// Client should retry after a few seconds
	c.Status(202).JSON(fiber.Map{
		"message": "Server restart initiated to refresh schema cache. Reconnect in 3-5 seconds.",
	})

	// Trigger graceful shutdown in a goroutine to allow response to be sent
	go func() {
		// Wait a moment to ensure the response is sent
		time.Sleep(500 * time.Millisecond)

		log.Info().Msg("Triggering graceful server shutdown for restart")

		// Send SIGTERM to trigger graceful shutdown via main's signal handler
		// This ensures proper cleanup through the main goroutine's shutdown path
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
			log.Error().Err(err).Msg("Failed to send shutdown signal, forcing exit")
			os.Exit(1)
		}
	}()

	return nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.app.Listen(s.config.Server.Address)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Stop realtime listener (PostgreSQL LISTEN/NOTIFY)
	if s.realtimeListener != nil {
		log.Info().Msg("Stopping realtime listener")
		s.realtimeListener.Stop()
	}

	// Shutdown realtime manager (close all WebSocket connections)
	if s.realtimeManager != nil {
		log.Info().Msg("Closing WebSocket connections")
		s.realtimeManager.Shutdown()
	}

	// Stop edge functions scheduler
	if s.functionsScheduler != nil {
		s.functionsScheduler.Stop()
	}

	// Stop jobs manager
	if s.jobsManager != nil {
		s.jobsManager.Stop()
	}

	// Stop webhook trigger service
	if s.webhookTriggerService != nil {
		s.webhookTriggerService.Stop()
	}

	log.Info().Msg("Shutting down HTTP server")
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

// GetAuthService returns the auth service from the auth handler
func (s *Server) GetAuthService() *auth.Service {
	if s.authHandler == nil {
		return nil
	}
	return s.authHandler.authService
}

// LoadFunctionsFromFilesystem loads edge functions from the filesystem
// This is called at boot time if auto_load_on_boot is enabled
func (s *Server) LoadFunctionsFromFilesystem(ctx context.Context) error {
	if s.functionsHandler == nil {
		return fmt.Errorf("functions handler not initialized")
	}
	return s.functionsHandler.LoadFromFilesystem(ctx)
}

// LoadJobsFromFilesystem loads job functions from the filesystem
// This is called at boot time if auto_load_on_boot is enabled
func (s *Server) LoadJobsFromFilesystem(ctx context.Context) error {
	if s.jobsHandler == nil {
		return fmt.Errorf("jobs handler not initialized")
	}
	// Use "default" as the namespace for jobs loaded at boot
	return s.jobsHandler.LoadFromFilesystem(ctx, "default")
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

	// Get the realtime manager and broadcast to the channel
	if s.realtimeHandler == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Realtime service not available",
		})
	}

	manager := s.realtimeHandler.GetManager()
	recipientCount := manager.BroadcastToChannel(req.Channel, realtime.ServerMessage{
		Type:    realtime.MessageTypeBroadcast,
		Channel: req.Channel,
		Payload: map[string]interface{}{
			"broadcast": map[string]interface{}{
				"event":   "broadcast",
				"payload": req.Message,
			},
		},
	})

	return c.JSON(fiber.Map{
		"success":    true,
		"channel":    req.Channel,
		"recipients": recipientCount,
	})
}
