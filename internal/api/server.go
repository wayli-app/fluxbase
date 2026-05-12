package api

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/observability"
	"github.com/nimbleflux/fluxbase/internal/pubsub"
	"github.com/nimbleflux/fluxbase/internal/ratelimit"
	"github.com/nimbleflux/fluxbase/internal/storage"
)

// Server represents the HTTP server
type Server struct {
	// Core infrastructure
	app     *fiber.App
	config  *config.Config
	db      *database.Connection
	tracer  *observability.Tracer
	rest    *RESTHandler
	version string

	// Handler groups (organized by domain)
	Auth       *AuthHandlers
	Storage    *StorageHandlers
	AI         *AIHandlers
	Functions  *FunctionsHandlers
	Jobs       *JobsHandlers
	Realtime   *RealtimeHandlers
	MCP        *MCPHandlers
	Tenancy    *TenancyHandlers
	Branching  *BranchingHandlers
	Settings   *SettingsHandlers
	Webhook    *WebhookHandlers
	Logging    *LoggingHandlers
	Schema     *SchemaHandlers
	RPC        *RPCHandlers
	GraphQL    *GraphQLHandlers
	Extensions *ExtensionsHandlers
	Secrets    *SecretsHandlers
	Scaling    *ScalingHandlers
	Metrics    *MetricsComponents
	Email      *EmailHandlers
	Captcha    *CaptchaHandlers
	Monitoring *MonitoringHandlers
	Quota      *QuotaHandlers
	Middleware *MiddlewareComponents

	// SQL handler (standalone, used by SQL editor)
	sqlHandler *SQLHandler

	// Server-owned dependencies (instead of global singletons)
	rateLimiter ratelimit.Store
	pubSub      pubsub.PubSub

	// Shared storage for middleware (rate limiter, CSRF, etc.)
	// This prevents creating multiple GC goroutines from Fiber's memory.New()
	sharedMiddlewareStorage fiber.Storage

	// Tenant configuration loader for multi-tenant config overrides
	tenantConfigLoader *config.TenantConfigLoader

	// Schema graph cache (per-tenant, TTL-based)
	graphCache *schemaGraphCache

	// Cached middleware instances (created once during init)
	requireAuth  fiber.Handler
	optionalAuth fiber.Handler

	// Service registry for module-based initialization
	registry *ServiceRegistry
	modules  []Module
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, db *database.Connection, version string) *Server {
	s := &Server{
		config:     cfg,
		db:         db,
		version:    version,
		graphCache: newSchemaGraphCache(2 * time.Minute),
		Auth:       &AuthHandlers{},
		Storage:    &StorageHandlers{},
		AI:         &AIHandlers{},
		Functions:  &FunctionsHandlers{},
		Jobs:       &JobsHandlers{},
		Realtime:   &RealtimeHandlers{},
		MCP:        &MCPHandlers{},
		Tenancy:    &TenancyHandlers{},
		Branching:  &BranchingHandlers{},
		Settings:   &SettingsHandlers{},
		Webhook:    &WebhookHandlers{},
		Logging:    &LoggingHandlers{},
		Schema:     &SchemaHandlers{},
		RPC:        &RPCHandlers{},
		GraphQL:    &GraphQLHandlers{},
		Extensions: &ExtensionsHandlers{},
		Secrets:    &SecretsHandlers{},
		Scaling:    &ScalingHandlers{},
		Email:      &EmailHandlers{},
		Captcha:    &CaptchaHandlers{},
		Monitoring: &MonitoringHandlers{},
		Quota:      &QuotaHandlers{},
		Middleware: &MiddlewareComponents{
			Tenant: middleware.TenantMiddleware(middleware.TenantConfig{
				DB: db,
			}),
		},
	}

	metricsObj := observability.NewMetrics()
	metricsStart := time.Now()

	s.initCore()

	s.registry = NewServiceRegistry(cfg, db)
	s.registry.PubSub = s.pubSub
	s.registry.RateLimiter = s.rateLimiter
	s.registry.Metrics = metricsObj

	db.SetMetrics(metricsObj)

	emailMod := &EmailModule{}
	secretsMod := &SecretsModule{}
	storageMod := &StorageModule{}
	loggingMod := &LoggingModule{}
	authMod := &AuthModule{}
	webhookMod := &WebhookModule{}
	extensionsMod := &ExtensionsModule{}
	tenancyMod := &TenancyModule{}
	settingsMod := &SettingsModule{}
	schemaMod := &SchemaModule{GraphCache: s.graphCache}
	functionsMod := &FunctionsModule{}
	jobsMod := &JobsModule{}
	rpcMod := &RPCModule{}
	aiMod := &AIModule{}
	realtimeMod := &RealtimeModule{}
	branchingMod := &BranchingModule{}
	graphqlMod := &GraphQLModule{}
	mcpMod := &MCPModule{}
	metricsMod := &MetricsModule{Handlers: &MetricsComponents{Metrics: metricsObj, StartTime: metricsStart}}
	bgMod := &BackgroundServicesModule{}

	mods := []Module{
		emailMod,
		secretsMod,
		storageMod,
		loggingMod,
		authMod,
		webhookMod,
		extensionsMod,
		tenancyMod,
		settingsMod,
		schemaMod,
		functionsMod,
		jobsMod,
		rpcMod,
		aiMod,
		realtimeMod,
		branchingMod,
		graphqlMod,
		mcpMod,
		metricsMod,
		bgMod,
	}

	if err := InitModules(context.Background(), s.registry, mods); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize modules")
	}
	s.modules = mods

	s.Secrets = secretsMod.Handlers
	s.Storage = storageMod.Handlers
	s.Logging = loggingMod.Handlers
	s.Auth = authMod.Handlers
	s.sqlHandler = authMod.SQLHandler
	s.requireAuth = authMod.RequireAuth
	s.optionalAuth = authMod.OptionalAuth
	s.Webhook = webhookMod.Handlers
	s.Extensions = extensionsMod.Handlers

	s.Tenancy = tenancyMod.Handlers
	if tenancyMod.Middleware != nil && tenancyMod.Middleware.TenantDB != nil {
		s.Middleware.TenantDB = tenancyMod.Middleware.TenantDB
	}

	s.Settings = settingsMod.Handlers
	s.Email = settingsMod.Email
	s.Captcha = settingsMod.Captcha

	s.Schema = schemaMod.Handlers
	s.rest = schemaMod.RESTHandler

	s.Functions = functionsMod.Handlers

	s.Jobs = jobsMod.Handlers

	s.RPC = rpcMod.Handlers

	s.AI = aiMod.Handlers
	s.Quota = aiMod.Quota

	s.Realtime = realtimeMod.Handlers
	s.Monitoring = realtimeMod.Monitoring

	s.Branching = branchingMod.Handlers

	s.GraphQL = graphqlMod.Handlers

	s.MCP = mcpMod.Handlers

	s.Metrics = metricsMod.Handlers

	s.Scaling = bgMod.Handlers

	if s.Webhook != nil && s.Webhook.Trigger != nil {
		if err := s.Webhook.Trigger.Start(context.Background()); err != nil {
			log.Error().Err(err).Msg("Failed to start webhook trigger service")
		}
	}
	if s.Logging != nil && s.Logging.Retention != nil {
		s.Logging.Retention.Start()
		log.Info().
			Dur("interval", s.config.Logging.RetentionCheckInterval).
			Msg("Log retention cleanup service started")
	}
	if s.Branching != nil && s.Branching.Scheduler != nil {
		s.Branching.Scheduler.Start()
	}

	s.setupMiddlewares()
	s.setupRoutes()

	log.Debug().Msg("Server initialization complete")
	return s
}

// createMCPAuthMiddleware creates authentication middleware for MCP that supports
// JWT, client key, service key, AND MCP OAuth tokens
func (s *Server) createMCPAuthMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer mcp_at_") {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			if s.MCP.OAuth != nil {
				clientID, userID, scopes, err := s.MCP.OAuth.ValidateAccessToken(c, token)
				if err == nil {
					c.Locals("auth_type", "mcp_oauth")
					c.Locals("client_key_id", clientID)
					c.Locals("client_key_scopes", scopes)
					if userID != nil {
						c.Locals("user_id", *userID)
					}
					return c.Next()
				}
			}
		}

		return middleware.RequireAuthOrServiceKey(
			s.Auth.Handler.authService,
			s.Auth.ClientKeyService,
			s.db.Pool(),
			nil,
			s.Auth.DashboardHandler.jwtManager,
		)(c)
	}
}

// handleHealth handles health check requests
func (s *Server) handleHealth(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.RequestCtx(), 5*time.Second)
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

	response := fiber.Map{
		"status":    status,
		"timestamp": time.Now().UTC(),
	}

	role, hasRole := GetUserRole(c)
	if hasRole && (role == "admin" || role == "instance_admin" || role == "service_role" || role == "tenant_admin" || role == "tenant_service") {
		services := fiber.Map{
			"database": dbHealthy,
			"realtime": true,
		}

		if dbHealthy {
			var dbSizeStr string
			err := s.db.Pool().QueryRow(ctx, "SELECT pg_size_pretty(pg_database_size(current_database()))").Scan(&dbSizeStr)
			if err == nil {
				services["database_size"] = dbSizeStr
			}
		}

		response["services"] = services
	}

	return c.Status(httpStatus).JSON(response)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.app.Listen(s.config.Server.Address, fiber.ListenConfig{EnablePrefork: false, DisableStartupMessage: !s.config.Debug})
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	ShutdownModules(ctx, s.modules)

	if s.Middleware.Idempotency != nil {
		s.Middleware.Idempotency.Stop()
	}

	if s.tracer != nil {
		if err := s.tracer.Shutdown(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to shutdown OpenTelemetry tracer")
		}
	}

	if s.pubSub != nil {
		log.Info().Msg("Closing pub/sub")
		if err := s.pubSub.Close(); err != nil {
			log.Warn().Err(err).Msg("Failed to close pub/sub")
		}
	}

	if s.rateLimiter != nil {
		log.Info().Msg("Closing rate limit store")
		if err := s.rateLimiter.Close(); err != nil {
			log.Warn().Err(err).Msg("Failed to close rate limit store")
		}
	}

	log.Info().Msg("Shutting down HTTP server")
	return s.app.ShutdownWithContext(ctx)
}

// App returns the underlying Fiber app instance for testing
func (s *Server) App() *fiber.App {
	return s.app
}

// GetStorageService returns the base storage service from the storage handler
func (s *Server) GetStorageService() *storage.Service {
	if s.Storage.Handler == nil || s.Storage.Handler.storageManager == nil {
		return nil
	}
	return s.Storage.Handler.storageManager.GetBaseService()
}

// GetAuthService returns the auth service from the auth handler
func (s *Server) GetAuthService() *auth.Service {
	if s.Auth.Handler == nil {
		return nil
	}
	return s.Auth.Handler.authService
}

// SetTenantConfigLoader sets the tenant configuration loader
func (s *Server) SetTenantConfigLoader(loader *config.TenantConfigLoader) {
	s.tenantConfigLoader = loader
	if s.Settings != nil && s.Settings.Unified != nil {
		s.Settings.Unified.SetTenantConfigLoader(loader)
	}
}

// customErrorHandler handles errors globally with a consistent response shape
func customErrorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"
	errCode := ErrCodeInternalError

	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
		message = e.Message
		switch {
		case code == fiber.StatusBadRequest:
			errCode = ErrCodeInvalidBody
		case code == fiber.StatusUnauthorized:
			errCode = ErrCodeAuthRequired
		case code == fiber.StatusForbidden:
			errCode = ErrCodeAccessDenied
		case code == fiber.StatusNotFound:
			errCode = ErrCodeNotFound
		case code == fiber.StatusConflict:
			errCode = ErrCodeConflict
		case code < 500:
			errCode = ErrCodeInvalidInput
		}
	}

	if code >= 500 {
		log.Error().Err(err).Str("path", c.Path()).Msg("Server error")
	}

	return c.Status(code).JSON(ErrorResponse{
		Error:     message,
		Code:      errCode,
		RequestID: getRequestID(c),
	})
}
