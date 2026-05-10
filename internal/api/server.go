package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/email"
	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/observability"
	"github.com/nimbleflux/fluxbase/internal/pubsub"
	"github.com/nimbleflux/fluxbase/internal/ratelimit"
	"github.com/nimbleflux/fluxbase/internal/realtime"
	"github.com/nimbleflux/fluxbase/internal/secrets"
	"github.com/nimbleflux/fluxbase/internal/storage"
	"github.com/nimbleflux/fluxbase/internal/webhook"
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

	// Cross-init method dependencies
	authService           *auth.Service
	emailManager          *email.Manager
	emailService          email.Service
	storageManager        *storage.Manager
	storageService        *storage.Service
	loggingService        *logging.Service
	captchaService        *auth.CaptchaService
	systemSettingsService *auth.SystemSettingsService
	userMgmtService       *auth.UserManagementService
	invitationService     *auth.InvitationService
	secretsStorage        *secrets.Storage

	// Cached middleware instances (created once during init)
	requireAuth    fiber.Handler
	optionalAuth   fiber.Handler
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
		Metrics: &MetricsComponents{
			Metrics:   observability.NewMetrics(),
			StartTime: time.Now(),
		},
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

	s.initCore()
	s.initEmail()
	s.initAuth()
	s.initStorage()
	s.initLogging()
	s.initWebhook()
	s.initSecrets()
	s.initTenancy()
	s.initSettings()
	s.initSchema()
	s.initFunctions()
	s.initJobs()
	s.initRPC()
	s.initAI()
	s.initRealtime()
	s.setupMCPServer()
	s.initBranching()
	s.initGraphQL()
	s.initExtensions()
	s.initMetrics()
	s.initBackgroundServices()
	s.setupMiddlewares()
	s.setupRoutes()

	if s.rateLimiter != nil {
		ratelimit.SetGlobalStore(s.rateLimiter)
	}
	if s.pubSub != nil {
		pubsub.SetGlobalPubSub(s.pubSub)
	}

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

func (s *Server) handleGetTables(c fiber.Ctx) error {
	ctx := context.Context(c.RequestCtx())

	if userID := middleware.GetUserID(c); userID != "" {
		if userRole, ok := GetUserRole(c); ok {
			ctx = database.ContextWithAuth(ctx, userID, userRole, userRole == "admin" || userRole == "service_role" || userRole == "tenant_service")
		}
	}

	inspector := s.db.Inspector()
	tenantPool := middleware.GetTenantPool(c)

	var schemasToQuery []string
	schemaParam := c.Query("schema")

	if schemaParam != "" {
		schemasToQuery = []string{schemaParam}
	} else {
		var schemas []string
		var err error
		if tenantPool != nil {
			schemas, err = inspector.GetSchemasFromPool(ctx, tenantPool)
		} else {
			schemas, err = inspector.GetSchemas(ctx)
		}
		if err != nil {
			return SendOperationFailed(c, "list schemas")
		}

		for _, schema := range schemas {
			if schema == "information_schema" || schema == "pg_catalog" || schema == "pg_toast" {
				continue
			}
			schemasToQuery = append(schemasToQuery, schema)
		}
	}

	var allItems []database.TableInfo
	for _, schema := range schemasToQuery {
		var tables, views, matviews []database.TableInfo
		var err error

		if tenantPool != nil {
			tables, err = inspector.GetAllTablesFromPool(ctx, tenantPool, schema)
		} else {
			tables, err = inspector.GetAllTables(ctx, schema)
		}
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get tables from schema")
		} else {
			allItems = append(allItems, tables...)
		}

		if tenantPool != nil {
			views, err = inspector.GetAllViewsFromPool(ctx, tenantPool, schema)
		} else {
			views, err = inspector.GetAllViews(ctx, schema)
		}
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get views from schema")
		} else {
			allItems = append(allItems, views...)
		}

		if tenantPool != nil {
			matviews, err = inspector.GetAllMaterializedViewsFromPool(ctx, tenantPool, schema)
		} else {
			matviews, err = inspector.GetAllMaterializedViews(ctx, schema)
		}
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get materialized views from schema")
		} else {
			allItems = append(allItems, matviews...)
		}
	}

	return c.JSON(allItems)
}

func (s *Server) handleGetTableSchema(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	schema := c.Params("schema")
	table := c.Params("table")

	if schema == "" || table == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Schema and table parameters are required",
		})
	}

	var tableInfo *database.TableInfo
	var err error
	if pool := middleware.GetTenantPool(c); pool != nil {
		tableInfo, err = s.db.Inspector().GetTableInfoFromPool(ctx, pool, schema, table)
	} else {
		tableInfo, err = s.db.Inspector().GetTableInfo(ctx, schema, table)
	}
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fmt.Sprintf("Table not found: %s.%s", schema, table),
		})
	}

	return c.JSON(tableInfo)
}

func (s *Server) handleGetSchemas(c fiber.Ctx) error {
	ctx := context.Context(c.RequestCtx())

	if userID := middleware.GetUserID(c); userID != "" {
		if userRole, ok := GetUserRole(c); ok {
			ctx = database.ContextWithAuth(ctx, userID, userRole, userRole == "admin" || userRole == "service_role" || userRole == "tenant_service")
		}
	}

	var schemas []string
	var err error
	if pool := middleware.GetTenantPool(c); pool != nil {
		schemas, err = s.db.Inspector().GetSchemasFromPool(ctx, pool)
	} else {
		schemas, err = s.db.Inspector().GetSchemas(ctx)
	}
	if err != nil {
		return SendOperationFailed(c, "list schemas")
	}

	var userSchemas []string
	for _, schema := range schemas {
		if schema != "information_schema" && schema != "pg_catalog" && schema != "pg_toast" {
			userSchemas = append(userSchemas, schema)
		}
	}

	if userRole, ok := GetUserRole(c); ok {
		isInstanceAdmin := userRole == "admin" || userRole == "instance_admin" || userRole == "service_role" || userRole == "tenant_service"
		if !isInstanceAdmin {
			tenantVisible := map[string]bool{
				"public": true, "auth": true, "storage": true, "functions": true,
				"jobs": true, "ai": true, "rpc": true, "mcp": true,
				"realtime": true, "branching": true, "logging": true, "platform": true,
			}
			var filtered []string
			for _, schema := range userSchemas {
				if tenantVisible[schema] {
					filtered = append(filtered, schema)
				}
			}
			userSchemas = filtered
		}
	}

	return c.JSON(userSchemas)
}

func (s *Server) handleExecuteQuery(c fiber.Ctx) error {
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "Not implemented"})
}

// InvalidateSchemaCache invalidates the REST API schema cache.
func (s *Server) InvalidateSchemaCache(ctx context.Context) error {
	schemaCache := s.rest.SchemaCache()
	if schemaCache == nil {
		return fmt.Errorf("schema cache not initialized")
	}

	schemaCache.InvalidateAll(ctx)
	s.graphCache.Invalidate()
	log.Debug().Msg("Schema cache invalidated and refresh triggered")

	return nil
}

// handleRefreshSchema refreshes the REST API schema cache without requiring a server restart
func (s *Server) handleRefreshSchema(c fiber.Ctx) error {
	log.Info().Msg("Schema refresh requested")

	schemaCache := s.rest.SchemaCache()
	if schemaCache == nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Schema cache not initialized",
		})
	}

	if err := schemaCache.Refresh(c.RequestCtx()); err != nil {
		log.Error().Err(err).Msg("Failed to refresh schema cache")
		return c.Status(500).JSON(fiber.Map{
			"error":   "Failed to refresh schema cache",
			"details": err.Error(),
		})
	}

	s.graphCache.Invalidate()

	log.Info().
		Int("tables", schemaCache.TableCount()).
		Int("views", schemaCache.ViewCount()).
		Msg("Schema cache refreshed successfully")

	return c.JSON(fiber.Map{
		"message": "Schema cache refreshed successfully",
		"tables":  schemaCache.TableCount(),
		"views":   schemaCache.ViewCount(),
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.app.Listen(s.config.Server.Address, fiber.ListenConfig{EnablePrefork: false, DisableStartupMessage: !s.config.Debug})
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.Scaling.FunctionsLeader != nil {
		log.Info().Msg("Stopping functions scheduler leader election")
		s.Scaling.FunctionsLeader.Stop()
	}
	if s.Scaling.JobsLeader != nil {
		log.Info().Msg("Stopping jobs scheduler leader election")
		s.Scaling.JobsLeader.Stop()
	}
	if s.Scaling.RPCLeader != nil {
		log.Info().Msg("Stopping RPC scheduler leader election")
		s.Scaling.RPCLeader.Stop()
	}

	if s.Realtime.Listener != nil {
		log.Info().Msg("Stopping realtime listener")
		s.Realtime.Listener.Stop()
	}

	if s.Realtime.Manager != nil {
		log.Info().Msg("Closing WebSocket connections")
		s.Realtime.Manager.Shutdown()
	}

	if s.Functions.Scheduler != nil {
		s.Functions.Scheduler.Stop()
	}

	if s.Jobs.Scheduler != nil {
		s.Jobs.Scheduler.Stop()
	}
	if s.Jobs.Manager != nil {
		s.Jobs.Manager.Stop()
	}

	if s.RPC.Scheduler != nil {
		s.RPC.Scheduler.Stop()
	}

	if s.RPC.Handler != nil {
		s.RPC.Handler.GetExecutor().Stop()
	}

	if s.Webhook.Trigger != nil {
		s.Webhook.Trigger.Stop()
	}

	if s.AI.Conversations != nil {
		s.AI.Conversations.Close()
	}

	if s.Middleware.Idempotency != nil {
		s.Middleware.Idempotency.Stop()
	}

	if s.Auth.OAuth != nil {
		s.Auth.OAuth.Stop()
	}

	if s.Branching.Scheduler != nil {
		s.Branching.Scheduler.Stop()
	}

	if s.Branching.Router != nil {
		log.Info().Msg("Closing branch connection pools")
		s.Branching.Router.CloseAllPools()
	}
	if s.Branching.Manager != nil {
		log.Info().Msg("Closing branch manager")
		s.Branching.Manager.Close()
	}

	if s.Metrics.StopChan != nil {
		close(s.Metrics.StopChan)
	}

	if s.Metrics.Server != nil {
		if err := s.Metrics.Server.Shutdown(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to shutdown metrics server")
		}
	}

	if s.tracer != nil {
		if err := s.tracer.Shutdown(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to shutdown OpenTelemetry tracer")
		}
	}

	if s.Logging.Retention != nil {
		log.Info().Msg("Stopping log retention cleanup service")
		s.Logging.Retention.Stop()
	}

	if s.Logging.Service != nil {
		log.Info().Msg("Closing central logging service")
		if err := s.Logging.Service.Close(); err != nil {
			log.Warn().Err(err).Msg("Failed to close logging service")
		}
	}

	if s.Schema.Cache != nil {
		s.Schema.Cache.Close()
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

// GetWebhookTriggerService returns the webhook trigger service for testing
func (s *Server) GetWebhookTriggerService() *webhook.TriggerService {
	return s.Webhook.Trigger
}

// GetAuthService returns the auth service from the auth handler
func (s *Server) GetAuthService() *auth.Service {
	if s.Auth.Handler == nil {
		return nil
	}
	return s.Auth.Handler.authService
}

// GetLoggingService returns the central logging service
func (s *Server) GetLoggingService() *logging.Service {
	return s.Logging.Service
}

// SetTenantConfigLoader sets the tenant configuration loader
func (s *Server) SetTenantConfigLoader(loader *config.TenantConfigLoader) {
	s.tenantConfigLoader = loader
	if s.Settings != nil && s.Settings.Unified != nil {
		s.Settings.Unified.SetTenantConfigLoader(loader)
	}
}

// GetTenantConfigLoader returns the tenant configuration loader
func (s *Server) GetTenantConfigLoader() *config.TenantConfigLoader {
	return s.tenantConfigLoader
}

// SchemaCache returns the REST API schema cache
func (s *Server) SchemaCache() *database.SchemaCache {
	return s.Schema.Cache
}

// LoadFunctionsFromFilesystem loads edge functions from the filesystem
func (s *Server) LoadFunctionsFromFilesystem(ctx context.Context) error {
	if s.Functions.Handler == nil {
		return fmt.Errorf("functions handler not initialized")
	}
	return s.Functions.Handler.LoadFromFilesystem(ctx)
}

// LoadJobsFromFilesystem loads job functions from the filesystem
func (s *Server) LoadJobsFromFilesystem(ctx context.Context) error {
	if s.Jobs.Handler == nil {
		return fmt.Errorf("jobs handler not initialized")
	}
	return s.Jobs.Handler.LoadFromFilesystem(ctx, "default")
}

// LoadAIChatbotsFromFilesystem loads AI chatbots from the filesystem
func (s *Server) LoadAIChatbotsFromFilesystem(ctx context.Context) error {
	if s.AI.Handler == nil {
		return fmt.Errorf("AI handler not initialized")
	}
	return s.AI.Handler.AutoLoadChatbots(ctx)
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

// handleRealtimeStats returns realtime statistics
func (s *Server) handleRealtimeStats(c fiber.Ctx) error {
	role, _ := c.Locals("user_role").(string)
	if role != "admin" && role != "instance_admin" && role != "tenant_admin" && role != "service_role" && role != "tenant_service" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required to view realtime stats",
		})
	}

	const defaultLimit = 25
	const maxLimit = 100
	limit := fiber.Query[int](c, "limit", defaultLimit)
	offset := fiber.Query[int](c, "offset", 0)
	search := strings.ToLower(c.Query("search", ""))

	limit, offset = NormalizePaginationParams(limit, offset, defaultLimit, maxLimit)

	manager := s.Realtime.Handler.GetManager()
	allConnections := manager.GetConnectionsForStats()

	userIDs := make([]string, 0)
	for _, conn := range allConnections {
		if conn.UserID != nil {
			userIDs = append(userIDs, *conn.UserID)
		}
	}

	type userInfo struct {
		email       string
		displayName *string
	}
	userInfoMap := make(map[string]userInfo)
	if len(userIDs) > 0 {
		query := `SELECT id, email, raw_user_meta_data->>'display_name' as display_name FROM auth.users WHERE id = ANY($1)`
		rows, err := s.db.Query(c.RequestCtx(), query, userIDs)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, email string
				var displayName *string
				if err := rows.Scan(&id, &email, &displayName); err == nil {
					userInfoMap[id] = userInfo{
						email:       email,
						displayName: displayName,
					}
				}
			}
		}
	}

	enrichedConnections := make([]realtime.ConnectionInfo, 0, len(allConnections))
	for _, conn := range allConnections {
		if conn.UserID != nil {
			if info, ok := userInfoMap[*conn.UserID]; ok {
				conn.Email = &info.email
				conn.DisplayName = info.displayName
			}
		}
		enrichedConnections = append(enrichedConnections, conn)
	}

	var filteredConnections []realtime.ConnectionInfo
	if search != "" {
		for _, conn := range enrichedConnections {
			if strings.Contains(strings.ToLower(conn.ID), search) ||
				strings.Contains(strings.ToLower(conn.RemoteAddr), search) ||
				(conn.UserID != nil && strings.Contains(strings.ToLower(*conn.UserID), search)) ||
				(conn.Email != nil && strings.Contains(strings.ToLower(*conn.Email), search)) ||
				(conn.DisplayName != nil && strings.Contains(strings.ToLower(*conn.DisplayName), search)) {
				filteredConnections = append(filteredConnections, conn)
			}
		}
	} else {
		filteredConnections = enrichedConnections
	}

	total := len(filteredConnections)

	if offset >= len(filteredConnections) {
		filteredConnections = []realtime.ConnectionInfo{}
	} else {
		filteredConnections = filteredConnections[offset:]
	}
	if len(filteredConnections) > limit {
		filteredConnections = filteredConnections[:limit]
	}

	return c.JSON(fiber.Map{
		"total_connections": total,
		"connections":       filteredConnections,
		"limit":             limit,
		"offset":            offset,
	})
}
