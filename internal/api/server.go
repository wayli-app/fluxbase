package api

import (
	"context"
	"time"

	"github.com/wayli-app/fluxbase/internal/adminui"
	"github.com/wayli-app/fluxbase/internal/auth"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/wayli-app/fluxbase/internal/email"
	"github.com/wayli-app/fluxbase/internal/realtime"
	"github.com/wayli-app/fluxbase/internal/storage"
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
	app              *fiber.App
	config           *config.Config
	db               *database.Connection
	rest             *RESTHandler
	authHandler      *AuthHandler
	storageHandler   *StorageHandler
	realtimeManager  *realtime.Manager
	realtimeHandler  *realtime.RealtimeHandler
	realtimeListener *realtime.Listener
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

	// Initialize storage service
	storageService, err := storage.NewService(&cfg.Storage)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize storage service")
	}

	// Create handlers
	authHandler := NewAuthHandler(authService)
	storageHandler := NewStorageHandler(storageService)

	// Create realtime components
	realtimeManager := realtime.NewManager(context.Background())
	realtimeAuthAdapter := realtime.NewAuthServiceAdapter(authService)
	realtimeHandler := realtime.NewRealtimeHandler(realtimeManager, realtimeAuthAdapter)
	realtimeListener := realtime.NewListener(db.Pool(), realtimeHandler)

	// Create server instance
	server := &Server{
		app:              app,
		config:           cfg,
		db:               db,
		rest:             NewRESTHandler(db, NewQueryParser()),
		authHandler:      authHandler,
		storageHandler:   storageHandler,
		realtimeManager:  realtimeManager,
		realtimeHandler:  realtimeHandler,
		realtimeListener: realtimeListener,
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

	// API routes
	api := s.app.Group("/api")

	// REST API routes (auto-generated from database schema)
	rest := api.Group("/tables")
	s.setupRESTRoutes(rest)

	// RPC routes (auto-generated from database functions)
	rpc := api.Group("/rpc")
	s.setupRPCRoutes(rpc)

	// Auth routes
	auth := api.Group("/auth")
	s.setupAuthRoutes(auth)

	// Storage routes
	storage := api.Group("/storage")
	s.setupStorageRoutes(storage)

	// Realtime WebSocket endpoint
	s.app.Get("/realtime", s.realtimeHandler.HandleWebSocket)

	// Realtime stats endpoint
	s.app.Get("/api/realtime/stats", s.handleRealtimeStats)

	// Functions routes
	functions := api.Group("/functions")
	s.setupFunctionsRoutes(functions)

	// Admin routes (optional)
	admin := api.Group("/admin")
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

	// Get all tables from public schema (and optionally other schemas)
	tables, err := s.db.Inspector().GetAllTables(ctx, "public")
	if err != nil {
		log.Error().Err(err).Msg("Failed to get tables for REST API")
		return
	}

	// Register dynamic routes for each table
	for _, table := range tables {
		s.rest.RegisterTableRoutes(router, table)
	}

	// Get all views and register as read-only endpoints
	views, err := s.db.Inspector().GetAllViews(ctx, "public")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get views for REST API")
	} else {
		for _, view := range views {
			s.rest.RegisterViewRoutes(router, view)
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
	// Public routes (no authentication required)
	router.Post("/signup", s.authHandler.SignUp)
	router.Post("/signin", s.authHandler.SignIn)
	router.Post("/refresh", s.authHandler.RefreshToken)
	router.Post("/magiclink", s.authHandler.SendMagicLink)
	router.Post("/magiclink/verify", s.authHandler.VerifyMagicLink)

	// Protected routes (authentication required)
	router.Post("/signout", s.authHandler.SignOut)
	router.Get("/user", s.authHandler.GetUser)
	router.Patch("/user", s.authHandler.UpdateUser)
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
	router.Get("/tables", s.handleGetTables)
	router.Get("/schemas", s.handleGetSchemas)
	router.Post("/query", s.handleExecuteQuery)
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
	tables, err := s.db.Inspector().GetAllTables(ctx, "public")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(tables)
}

func (s *Server) handleGetSchemas(c *fiber.Ctx) error {
	ctx := c.Context()
	schemas, err := s.db.Inspector().GetSchemas(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(schemas)
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
	stats := s.realtimeHandler.GetStats()
	return c.JSON(stats)
}