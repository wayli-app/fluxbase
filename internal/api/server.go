package api

import (
	"context"
	"time"

	"github.com/wayli-app/fluxbase/internal/adminui"
	"github.com/wayli-app/fluxbase/internal/auth"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/wayli-app/fluxbase/internal/email"
	"github.com/wayli-app/fluxbase/internal/middleware"
	"github.com/wayli-app/fluxbase/internal/realtime"
	"github.com/wayli-app/fluxbase/internal/storage"
	"github.com/wayli-app/fluxbase/internal/webhook"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/rs/zerolog/log"
)

// Server represents the HTTP server
type Server struct {
	app                   *fiber.App
	config                *config.Config
	db                    *database.Connection
	rest                  *RESTHandler
	authHandler           *AuthHandler
	adminAuthHandler      *AdminAuthHandler
	apiKeyHandler         *APIKeyHandler
	storageHandler        *StorageHandler
	webhookHandler        *WebhookHandler
	monitoringHandler     *MonitoringHandler
	userManagementHandler *UserManagementHandler
	realtimeManager       *realtime.Manager
	realtimeHandler       *realtime.RealtimeHandler
	realtimeListener      *realtime.Listener
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
	adminAuthHandler := NewAdminAuthHandler(authService, auth.NewUserRepository(db))
	apiKeyHandler := NewAPIKeyHandler(apiKeyService)
	storageHandler := NewStorageHandler(storageService)
	webhookHandler := NewWebhookHandler(webhookService)
	userMgmtHandler := NewUserManagementHandler(userMgmtService)

	// Create realtime components
	realtimeManager := realtime.NewManager(context.Background())
	realtimeAuthAdapter := realtime.NewAuthServiceAdapter(authService)
	realtimeHandler := realtime.NewRealtimeHandler(realtimeManager, realtimeAuthAdapter)
	realtimeListener := realtime.NewListener(db.Pool(), realtimeHandler)

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
		apiKeyHandler:         apiKeyHandler,
		storageHandler:        storageHandler,
		webhookHandler:        webhookHandler,
		monitoringHandler:     monitoringHandler,
		userManagementHandler: userMgmtHandler,
		realtimeManager:       realtimeManager,
		realtimeHandler:       realtimeHandler,
		realtimeListener:      realtimeListener,
	}

	// Start realtime listener
	if err := realtimeListener.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start realtime listener")
	}

	// Setup middlewares
	server.setupMiddlewares()

	// Setup routes
	server.setupRoutes()

	return server
}

// setupMiddlewares sets up global middlewares
func (s *Server) setupMiddlewares() {
	// Request ID middleware
	s.app.Use(requestid.New())

	// Logger middleware
	if s.config.Debug {
		s.app.Use(logger.New(logger.Config{
			Format: "[${time}] ${status} - ${latency} ${method} ${path} ${error}\n",
		}))
	}

	// Recover middleware
	s.app.Use(recover.New(recover.Config{
		EnableStackTrace: s.config.Debug,
	}))

	// CORS middleware
	s.app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Request-ID, Prefer",
		ExposeHeaders:    "Content-Range, Content-Encoding, Content-Length, X-Request-ID",
		AllowCredentials: false,
		MaxAge:           300,
	}))

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
	// Apply optional auth middleware (allows both authenticated and anonymous)
	// followed by RLS middleware to set PostgreSQL session variables
	rest := v1.Group("/tables",
		OptionalAuthMiddleware(s.authHandler.authService),
		middleware.RLSMiddleware(rlsConfig),
	)
	s.setupRESTRoutes(rest)

	// RPC routes (auto-generated from database functions)
	rpc := v1.Group("/rpc")
	s.setupRPCRoutes(rpc)

	// Auth routes
	auth := v1.Group("/auth")
	s.setupAuthRoutes(auth)

	// API Keys routes
	s.apiKeyHandler.RegisterRoutes(s.app)

	// Webhook routes
	s.webhookHandler.RegisterRoutes(s.app)

	// Monitoring routes
	s.monitoringHandler.RegisterRoutes(s.app)

	// User management routes (admin only)
	s.userManagementHandler.RegisterRoutes(s.app)

	// Storage routes
	storage := v1.Group("/storage")
	s.setupStorageRoutes(storage)

	// Realtime WebSocket endpoint (not versioned as it's WebSocket)
	s.app.Get("/realtime", s.realtimeHandler.HandleWebSocket)

	// Realtime stats endpoint
	s.app.Get("/api/v1/realtime/stats", s.handleRealtimeStats)

	// Realtime broadcast endpoint
	s.app.Post("/api/v1/realtime/broadcast", s.handleRealtimeBroadcast)

	// Functions routes
	functions := v1.Group("/functions")
	s.setupFunctionsRoutes(functions)

	// Admin routes (optional)
	admin := v1.Group("/admin")
	s.setupAdminRoutes(admin)

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

	// Register tables from all schemas (including auth, public, etc.)
	for _, schema := range schemas {
		// Skip system schemas
		if schema == "information_schema" || schema == "pg_catalog" || schema == "pg_toast" {
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
	s.authHandler.RegisterRoutes(s.app, rateLimiters)
}

// setupStorageRoutes sets up storage routes
func (s *Server) setupStorageRoutes(router fiber.Router) {
	// Bucket management
	router.Get("/buckets", s.storageHandler.ListBuckets)
	router.Post("/buckets/:bucket", s.storageHandler.CreateBucket)
	router.Delete("/buckets/:bucket", s.storageHandler.DeleteBucket)

	// File operations
	router.Post("/:bucket/*", s.storageHandler.UploadFile)          // Upload file
	router.Get("/:bucket/*", s.storageHandler.DownloadFile)         // Download file
	router.Delete("/:bucket/*", s.storageHandler.DeleteFile)        // Delete file
	router.Head("/:bucket/*", s.storageHandler.GetFileInfo)         // Get file metadata

	// List files in bucket
	router.Get("/:bucket", s.storageHandler.ListFiles)

	// Multipart upload
	router.Post("/:bucket/multipart", s.storageHandler.MultipartUpload)

	// Signed URLs (for S3-compatible storage)
	router.Post("/:bucket/*/signed-url", s.storageHandler.GenerateSignedURL)
}

// setupFunctionsRoutes sets up edge functions routes
func (s *Server) setupFunctionsRoutes(router fiber.Router) {
	router.Get("/", s.handleListFunctions)
	router.Post("/", s.handleCreateFunction)
	router.Put("/:name", s.handleUpdateFunction)
	router.Delete("/:name", s.handleDeleteFunction)
	router.Post("/:name/invoke", s.handleInvokeFunction)
}

// setupAdminRoutes sets up admin routes
func (s *Server) setupAdminRoutes(router fiber.Router) {
	// Public admin auth routes (no authentication required)
	router.Get("/setup/status", s.adminAuthHandler.GetSetupStatus)
	router.Post("/setup", s.adminAuthHandler.InitialSetup)
	router.Post("/login", s.adminAuthHandler.AdminLogin)
	router.Post("/refresh", s.adminAuthHandler.AdminRefreshToken)

	// Protected admin routes (require admin authentication)
	router.Post("/logout", AuthMiddleware(s.authHandler.authService), s.adminAuthHandler.AdminLogout)
	router.Get("/me", AuthMiddleware(s.authHandler.authService), s.adminAuthHandler.GetCurrentAdmin)

	// Existing admin routes (protect with auth)
	router.Get("/tables", AuthMiddleware(s.authHandler.authService), s.handleGetTables)
	router.Get("/schemas", AuthMiddleware(s.authHandler.authService), s.handleGetSchemas)
	router.Post("/query", AuthMiddleware(s.authHandler.authService), s.handleExecuteQuery)
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

// Placeholder handlers (to be implemented)
func (s *Server) handleGetBuckets(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Get buckets endpoint - to be implemented"})
}

func (s *Server) handleCreateBucket(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Create bucket endpoint - to be implemented"})
}

func (s *Server) handleDeleteBucket(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Delete bucket endpoint - to be implemented"})
}

func (s *Server) handleUpload(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Upload endpoint - to be implemented"})
}

func (s *Server) handleDownload(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Download endpoint - to be implemented"})
}

func (s *Server) handleDeleteObject(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Delete object endpoint - to be implemented"})
}

func (s *Server) handleListObjects(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "List objects endpoint - to be implemented"})
}

func (s *Server) handleListFunctions(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "List functions endpoint - to be implemented"})
}

func (s *Server) handleCreateFunction(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Create function endpoint - to be implemented"})
}

func (s *Server) handleUpdateFunction(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Update function endpoint - to be implemented"})
}

func (s *Server) handleDeleteFunction(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Delete function endpoint - to be implemented"})
}

func (s *Server) handleInvokeFunction(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Invoke function endpoint - to be implemented"})
}

func (s *Server) handleRealtimeUpgrade(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Realtime WebSocket endpoint - to be implemented"})
}

func (s *Server) handleGetTables(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get all schemas
	schemas, err := s.db.Inspector().GetSchemas(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Collect tables from all schemas (except system schemas)
	var allTables []database.TableInfo
	for _, schema := range schemas {
		// Skip system schemas
		if schema == "information_schema" || schema == "pg_catalog" || schema == "pg_toast" {
			continue
		}

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
	return s.app.ShutdownWithContext(ctx)
}

// App returns the underlying Fiber app instance for testing
func (s *Server) App() *fiber.App {
	return s.app
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

	// Broadcast the message
	s.realtimeHandler.Broadcast(req.Channel, req.Message)

	return c.JSON(fiber.Map{
		"success": true,
		"channel": req.Channel,
	})
}