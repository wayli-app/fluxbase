package api

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/storage/memory/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/adminui"
	"github.com/nimbleflux/fluxbase/internal/ai"
	"github.com/nimbleflux/fluxbase/internal/branching"
	"github.com/nimbleflux/fluxbase/internal/functions"
	"github.com/nimbleflux/fluxbase/internal/jobs"
	"github.com/nimbleflux/fluxbase/internal/mcp"
	"github.com/nimbleflux/fluxbase/internal/mcp/custom"
	mcpresources "github.com/nimbleflux/fluxbase/internal/mcp/resources"
	mcptools "github.com/nimbleflux/fluxbase/internal/mcp/tools"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/migrations"
	"github.com/nimbleflux/fluxbase/internal/observability"
	"github.com/nimbleflux/fluxbase/internal/pubsub"
	"github.com/nimbleflux/fluxbase/internal/ratelimit"
	"github.com/nimbleflux/fluxbase/internal/rpc"
	"github.com/nimbleflux/fluxbase/internal/scaling"
	"github.com/nimbleflux/fluxbase/internal/tenantdb"
)

func (s *Server) initCore() {
	cfg := s.config
	db := s.db

	app := fiber.New(fiber.Config{
		ServerHeader:      "Fluxbase",
		AppName:           fmt.Sprintf("Fluxbase v%s", s.version),
		BodyLimit:         cfg.Server.BodyLimit,
		StreamRequestBody: true,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		ErrorHandler:      customErrorHandler,
	})

	if cfg.Debug {
		app.Use(func(c fiber.Ctx) error {
			c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			c.Set("Pragma", "no-cache")
			c.Set("Expires", "0")
			return c.Next()
		})
	}

	tracer, err := observability.NewTracer(context.Background(), cfg.Tracing)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize OpenTelemetry tracer, tracing will be disabled")
	}

	rateLimitStore, err := ratelimit.NewStore(&cfg.Scaling, db.Pool())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize rate limit store, falling back to memory")
		rateLimitStore = nil
	} else {
		log.Info().Str("backend", cfg.Scaling.Backend).Msg("Rate limit store initialized")
	}

	ps, err := pubsub.NewPubSub(&cfg.Scaling, db.Pool())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize pub/sub, cross-instance broadcasting disabled")
		ps = nil
	} else {
		log.Info().Str("backend", cfg.Scaling.Backend).Msg("Pub/sub initialized for cross-instance broadcasting")
	}

	gcInterval := 10 * time.Minute
	if os.Getenv("FLUXBASE_TEST_MODE") == "1" {
		gcInterval = 24 * time.Hour
	}
	sharedMiddlewareStorage := memory.New(memory.Config{
		GCInterval: gcInterval,
	})

	s.app = app
	s.tracer = tracer
	s.rateLimiter = rateLimitStore
	s.pubSub = ps
	s.sharedMiddlewareStorage = sharedMiddlewareStorage
}

func (s *Server) setupMCPServer() {
	cfg := s.config
	db := s.db

	s.MCP.Handler = mcp.NewHandler(&cfg.MCP, db)
	s.MCP.OAuth = NewMCPOAuthHandler(db, &cfg.MCP, s.authService, cfg.BaseURL, cfg.GetPublicBaseURL())

	if !cfg.MCP.Enabled {
		return
	}

	schemaCache := s.Schema.Cache
	storageService := s.storageService
	functionsHandler := s.Functions.Handler
	rpcHandler := s.RPC.Handler
	vectorHandler := s.AI.VectorHandler

	mcpServer := s.MCP.Handler.Server()

	toolRegistry := mcpServer.ToolRegistry()

	toolRegistry.Register(mcptools.NewThinkTool())

	queryTableTool := mcptools.NewQueryTableTool(s.db, schemaCache)
	if vectorHandler != nil && vectorHandler.GetEmbeddingService() != nil {
		queryTableTool.SetEmbeddingGenerator(vectorHandler.GetEmbeddingService())
		log.Debug().Msg("MCP: QueryTableTool configured with embedding generator for vector search")
	}
	toolRegistry.Register(queryTableTool)
	toolRegistry.Register(mcptools.NewInsertRecordTool(s.db, schemaCache))
	toolRegistry.Register(mcptools.NewUpdateRecordTool(s.db, schemaCache))
	toolRegistry.Register(mcptools.NewDeleteRecordTool(s.db, schemaCache))
	toolRegistry.Register(mcptools.NewExecuteSQLTool(s.db))

	if storageService != nil {
		toolRegistry.Register(mcptools.NewListObjectsTool(storageService))
		toolRegistry.Register(mcptools.NewUploadObjectTool(storageService))
		toolRegistry.Register(mcptools.NewDownloadObjectTool(storageService))
		toolRegistry.Register(mcptools.NewDeleteObjectTool(storageService))
	}

	if functionsHandler != nil && s.config.Functions.Enabled {
		toolRegistry.Register(mcptools.NewInvokeFunctionTool(
			s.db,
			functionsHandler.GetRuntime(),
			functionsHandler.GetPublicURL(),
			functionsHandler.GetFunctionsDir(),
		))
	}

	if rpcHandler != nil && s.config.RPC.Enabled {
		rpcStorage := rpc.NewStorage(s.db)
		toolRegistry.Register(mcptools.NewInvokeRPCTool(
			rpcHandler.GetExecutor(),
			rpcStorage,
		))
	}

	if s.Jobs.Manager != nil && s.config.Jobs.Enabled {
		jobsStorage := jobs.NewStorage(s.db)
		toolRegistry.Register(mcptools.NewSubmitJobTool(jobsStorage))
		toolRegistry.Register(mcptools.NewGetJobStatusTool(jobsStorage))
	}

	if s.AI.Chat != nil {
		if ragService := s.AI.Chat.GetRAGService(); ragService != nil {
			toolRegistry.Register(mcptools.NewSearchVectorsTool(ragService))
			log.Debug().Msg("MCP: Registered search_vectors tool")
		} else {
			log.Debug().Msg("MCP: Vector search tool not registered - RAG service not available")
		}
	}

	if s.AI.KBStorage != nil {
		knowledgeGraph := ai.NewKnowledgeGraph(s.AI.KBStorage)
		toolRegistry.Register(mcptools.NewQueryKnowledgeGraphTool(knowledgeGraph))
		toolRegistry.Register(mcptools.NewFindRelatedEntitiesTool(knowledgeGraph))
		toolRegistry.Register(mcptools.NewBrowseKnowledgeGraphTool(knowledgeGraph))
		log.Debug().Msg("MCP: Registered knowledge graph tools")
	}

	toolRegistry.Register(mcptools.NewListSchemasTool(s.db))
	toolRegistry.Register(mcptools.NewCreateSchemaTool(s.db))
	toolRegistry.Register(mcptools.NewCreateTableTool(s.db))
	toolRegistry.Register(mcptools.NewDropTableTool(s.db))
	toolRegistry.Register(mcptools.NewAddColumnTool(s.db))
	toolRegistry.Register(mcptools.NewDropColumnTool(s.db))
	toolRegistry.Register(mcptools.NewRenameTableTool(s.db))

	toolRegistry.Register(mcptools.NewHttpRequestTool())

	if s.config.Functions.Enabled {
		functionsStorage := functions.NewStorage(s.db)
		toolRegistry.Register(mcptools.NewSyncFunctionTool(functionsStorage))
	}

	if s.config.Jobs.Enabled {
		jobsStorage := jobs.NewStorage(s.db)
		toolRegistry.Register(mcptools.NewSyncJobTool(jobsStorage))
	}

	if s.config.RPC.Enabled {
		rpcStorage := rpc.NewStorage(s.db)
		toolRegistry.Register(mcptools.NewSyncRPCTool(rpcStorage))
	}

	migrationsStorage := migrations.NewStorage(s.db)
	migrationsExecutor := migrations.NewExecutor(s.db)
	toolRegistry.Register(mcptools.NewSyncMigrationTool(migrationsStorage, migrationsExecutor))

	if s.config.AI.Enabled {
		aiStorage := ai.NewStorage(s.db)
		toolRegistry.Register(mcptools.NewSyncChatbotTool(aiStorage))
	}

	if s.Branching.Manager != nil && s.config.Branching.Enabled {
		branchStorage := branching.NewStorage(s.db, s.config.EncryptionKey)
		toolRegistry.Register(mcptools.NewListBranchesTool(branchStorage))
		toolRegistry.Register(mcptools.NewGetBranchTool(branchStorage))
		toolRegistry.Register(mcptools.NewCreateBranchTool(s.Branching.Manager))
		toolRegistry.Register(mcptools.NewDeleteBranchTool(s.Branching.Manager, branchStorage))
		toolRegistry.Register(mcptools.NewResetBranchTool(s.Branching.Manager, branchStorage))
		toolRegistry.Register(mcptools.NewGrantBranchAccessTool(branchStorage))
		toolRegistry.Register(mcptools.NewRevokeBranchAccessTool(branchStorage))
		toolRegistry.Register(mcptools.NewGetActiveBranchTool(s.Branching.Router))
		toolRegistry.Register(mcptools.NewSetActiveBranchTool(s.Branching.Router, branchStorage))
	}

	resourceRegistry := mcpServer.ResourceRegistry()

	resourceRegistry.Register(mcpresources.NewSchemaResource(schemaCache))
	resourceRegistry.Register(mcpresources.NewTableResource(schemaCache))

	if s.config.Functions.Enabled {
		resourceRegistry.Register(mcpresources.NewFunctionsResource(functions.NewStorage(s.db)))
	}

	if s.config.RPC.Enabled {
		resourceRegistry.Register(mcpresources.NewRPCResource(rpc.NewStorage(s.db)))
	}

	resourceRegistry.Register(mcpresources.NewBucketsResource(s.db))

	if s.AI.Chat != nil {
		s.AI.Chat.SetMCPToolRegistry(toolRegistry)
		s.AI.Chat.SetMCPResources(resourceRegistry)
		log.Debug().Msg("MCP registries wired to AI chat handler")
	}

	customStorage := custom.NewStorage(s.db)
	mcpInternalURL := s.config.BaseURL
	if mcpInternalURL == "" {
		mcpInternalURL = "http://localhost" + s.config.Server.Address
	}
	customExecutor := custom.NewExecutor(s.config.Auth.JWTSecret, mcpInternalURL, nil)
	s.MCP.CustomManager = custom.NewManager(customStorage, customExecutor, toolRegistry, resourceRegistry)
	s.MCP.CustomHandler = NewCustomMCPHandler(customStorage, s.MCP.CustomManager, &s.config.MCP)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.MCP.CustomManager.LoadAndRegisterAll(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to load some custom MCP tools/resources")
	}

	log.Debug().
		Int("tools", len(toolRegistry.ListTools(&mcp.AuthContext{IsServiceRole: true}))).
		Int("resources", len(resourceRegistry.ListResources(&mcp.AuthContext{IsServiceRole: true}))).
		Msg("MCP Server initialized with tools and resources")

	if s.config.MCP.Enabled && s.config.MCP.AutoLoadOnBoot && s.MCP.CustomManager != nil {
		if err := s.MCP.CustomManager.AutoLoadFromDir(context.Background(), s.config.MCP.ToolsDir); err != nil {
			log.Error().Err(err).Msg("Failed to auto-load custom MCP tools")
		} else {
			log.Info().Msg("Custom MCP tools auto-loaded successfully")
		}
	}

	log.Info().
		Str("base_path", cfg.MCP.BasePath).
		Dur("session_timeout", cfg.MCP.SessionTimeout).
		Msg("MCP Server enabled")
}

func (s *Server) initMetrics() {
	cfg := s.config

	if !cfg.Metrics.Enabled {
		return
	}

	s.Metrics.Server = observability.NewMetricsServer(cfg.Metrics.Port, cfg.Metrics.Path)
	if err := s.Metrics.Server.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start metrics server")
	}

	s.db.SetMetrics(s.Metrics.Metrics)

	if s.storageService != nil {
		s.storageService.SetMetrics(s.Metrics.Metrics)
	}

	s.authService.SetMetrics(s.Metrics.Metrics)

	if s.Realtime.Manager != nil {
		s.Realtime.Manager.SetMetrics(s.Metrics.Metrics)
	}

	middleware.SetRateLimiterMetrics(s.Metrics.Metrics)

	s.Metrics.StopChan = make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.Metrics.Metrics.UpdateUptime(s.Metrics.StartTime)
			case <-s.Metrics.StopChan:
				return
			}
		}
	}()
}

func (s *Server) initBackgroundServices() {
	cfg := s.config

	if !cfg.Scaling.DisableRealtime && !cfg.Scaling.WorkerOnly {
		if err := s.Realtime.Listener.Start(); err != nil {
			log.Error().Err(err).Msg("Failed to start realtime listener")
		}
	} else {
		log.Info().
			Bool("disable_realtime", cfg.Scaling.DisableRealtime).
			Bool("worker_only", cfg.Scaling.WorkerOnly).
			Msg("Realtime listener disabled by scaling configuration")
	}

	s.Scaling.FunctionsLeader = s.startSchedulerWithLeaderElection(
		"functions-scheduler", scaling.FunctionsSchedulerLockID,
		func() {
			log.Info().Msg("This instance is now the functions scheduler leader")
			if err := s.Functions.Scheduler.Start(); err != nil {
				log.Error().Err(err).Msg("Failed to start edge functions scheduler")
			}
		},
		func() {
			log.Warn().Msg("Lost functions scheduler leadership - stopping scheduler")
			s.Functions.Scheduler.Stop()
		},
	)

	if cfg.Jobs.Enabled && s.Jobs.Manager != nil {
		workerCount := cfg.Jobs.EmbeddedWorkerCount
		if workerCount <= 0 {
			workerCount = 4
		}
		if err := s.Jobs.Manager.Start(context.Background(), workerCount); err != nil {
			log.Error().Err(err).Msg("Failed to start jobs manager")
		} else {
			log.Info().Int("workers", workerCount).Msg("Jobs manager started successfully")
		}

		if s.Jobs.Scheduler != nil {
			s.Scaling.JobsLeader = s.startSchedulerWithLeaderElection(
				"jobs-scheduler", scaling.JobsSchedulerLockID,
				func() {
					log.Info().Msg("This instance is now the jobs scheduler leader")
					if err := s.Jobs.Scheduler.Start(); err != nil {
						log.Error().Err(err).Msg("Failed to start jobs scheduler")
					}
				},
				func() {
					log.Warn().Msg("Lost jobs scheduler leadership - stopping scheduler")
					s.Jobs.Scheduler.Stop()
				},
			)
		}
	}

	if cfg.RPC.Enabled && s.RPC.Scheduler != nil {
		s.Scaling.RPCLeader = s.startSchedulerWithLeaderElection(
			"rpc-scheduler", scaling.RPCSchedulerLockID,
			func() {
				log.Info().Msg("This instance is now the RPC scheduler leader")
				if err := s.RPC.Scheduler.Start(); err != nil {
					log.Error().Err(err).Msg("Failed to start RPC scheduler")
				}
			},
			func() {
				log.Warn().Msg("Lost RPC scheduler leadership - stopping scheduler")
				s.RPC.Scheduler.Stop()
			},
		)
	}

	if err := s.Webhook.Trigger.Start(context.Background()); err != nil {
		log.Error().Err(err).Msg("Failed to start webhook trigger service")
	}

	if s.Logging.Retention != nil {
		s.Logging.Retention.Start()
		log.Info().
			Dur("interval", cfg.Logging.RetentionCheckInterval).
			Msg("Log retention cleanup service started")
	}

	if s.Branching.Scheduler != nil {
		s.Branching.Scheduler.Start()
	}
}

func (s *Server) setupMiddlewares() {
	log.Debug().Msg("Adding requestid middleware")
	s.app.Use(requestid.New())

	if s.config.Tracing.Enabled && s.tracer != nil && s.tracer.IsEnabled() {
		log.Debug().Msg("Adding OpenTelemetry tracing middleware")
		s.app.Use(middleware.TracingMiddleware(middleware.TracingConfig{
			Enabled:            true,
			ServiceName:        s.config.Tracing.ServiceName,
			SkipPaths:          []string{"/health", "/ready", "/metrics"},
			RecordRequestBody:  false,
			RecordResponseBody: false,
		}))
	}

	if s.config.Metrics.Enabled && s.Metrics.Metrics != nil {
		log.Debug().Msg("Adding Prometheus metrics middleware")
		s.app.Use(s.Metrics.Metrics.MetricsMiddleware())
	}

	log.Debug().Msg("Adding security headers middleware")
	s.app.Use(func(c fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/admin") {
			return middleware.AdminUISecurityHeaders()(c)
		}
		return middleware.SecurityHeaders()(c)
	})

	log.Debug().Msg("Adding structured logger middleware")
	s.app.Use(middleware.StructuredLogger(middleware.StructuredLoggerConfig{
		SkipPaths:              []string{"/health", "/ready", "/metrics"},
		SkipSuccessfulRequests: !s.config.Debug,
	}))

	log.Debug().Msg("Adding recover middleware")
	s.app.Use(recover.New(recover.Config{
		EnableStackTrace: s.config.Debug,
	}))

	corsCredentials := s.config.CORS.AllowCredentials
	corsOrigins := s.config.CORS.AllowedOrigins

	hasWildcard := false
	for _, origin := range corsOrigins {
		if origin == "*" {
			hasWildcard = true
			break
		}
	}

	if hasWildcard && corsCredentials {
		log.Warn().Msg("CORS: AllowCredentials disabled because AllowOrigins contains '*' (not allowed per CORS spec)")
		corsCredentials = false
	}
	if !hasWildcard && s.config.PublicBaseURL != "" {
		found := false
		for _, origin := range corsOrigins {
			if origin == s.config.PublicBaseURL {
				found = true
				break
			}
		}
		if !found {
			corsOrigins = append(corsOrigins, s.config.PublicBaseURL)
			log.Debug().Str("public_url", s.config.PublicBaseURL).Msg("Added public base URL to CORS origins")
		}
	}
	log.Debug().
		Strs("origins", corsOrigins).
		Bool("credentials", corsCredentials).
		Msg("Adding CORS middleware")

	corsConfig := cors.Config{
		AllowMethods:     s.config.CORS.AllowedMethods,
		AllowHeaders:     s.config.CORS.AllowedHeaders,
		ExposeHeaders:    s.config.CORS.ExposedHeaders,
		AllowCredentials: corsCredentials,
		MaxAge:           s.config.CORS.MaxAge,
	}

	if hasWildcard {
		corsConfig.AllowOriginsFunc = func(origin string) bool {
			return true
		}
	} else {
		corsConfig.AllowOrigins = corsOrigins
	}

	s.app.Use(cors.New(corsConfig))
	log.Debug().Msg("CORS middleware added")

	if len(s.config.Server.AllowedIPRanges) > 0 {
		log.Info().
			Int("ranges", len(s.config.Server.AllowedIPRanges)).
			Strs("ranges", s.config.Server.AllowedIPRanges).
			Msg("Adding global IP allowlist middleware")
		s.app.Use(middleware.RequireGlobalIPAllowlist(&s.config.Server))
	} else {
		log.Debug().Msg("Global IP allowlist disabled (no ranges configured)")
	}

	s.app.Use(middleware.DynamicGlobalAPILimiter(s.Auth.Handler.authService.GetSettingsCache(), s.sharedMiddlewareStorage))

	if s.config.Server.BodyLimits.Enabled {
		bodyLimitConfig := middleware.BodyLimitsFromConfig(
			s.config.Server.BodyLimits.DefaultLimit,
			s.config.Server.BodyLimits.RESTLimit,
			s.config.Server.BodyLimits.AuthLimit,
			s.config.Server.BodyLimits.StorageLimit,
			s.config.Server.BodyLimits.BulkLimit,
			s.config.Server.BodyLimits.AdminLimit,
			s.config.Server.BodyLimits.MaxJSONDepth,
		)
		s.app.Use(middleware.BodyLimitMiddleware(bodyLimitConfig))
		log.Info().
			Int64("default", s.config.Server.BodyLimits.DefaultLimit).
			Int64("rest", s.config.Server.BodyLimits.RESTLimit).
			Int64("auth", s.config.Server.BodyLimits.AuthLimit).
			Int64("storage", s.config.Server.BodyLimits.StorageLimit).
			Int("max_json_depth", s.config.Server.BodyLimits.MaxJSONDepth).
			Msg("Per-endpoint body limits enabled")
	}

	idempotencyConfig := middleware.DefaultIdempotencyConfig()
	idempotencyConfig.DB = s.db.Pool()
	s.Middleware.Idempotency = middleware.NewIdempotencyMiddleware(idempotencyConfig)
	s.app.Use(s.Middleware.Idempotency.Middleware())
	log.Info().
		Str("header", idempotencyConfig.HeaderName).
		Dur("ttl", idempotencyConfig.TTL).
		Msg("Idempotency key support enabled")

	s.app.Use(compress.New(compress.Config{
		Level: compress.LevelDefault,
	}))
}

func (s *Server) setupRoutes() {
	s.auditRoutesAtStartup()

	if err := s.registerRoutesViaRegistry(); err != nil {
		log.Fatal().Err(err).Msg("Failed to setup routes via registry")
	}

	if s.config.Admin.Enabled {
		if s.config.Security.SetupToken == "" {
			log.Error().Msg("Admin UI is enabled but FLUXBASE_SECURITY_SETUP_TOKEN is not set. Admin UI will not be registered for security reasons.")
		} else {
			adminUI := adminui.New(s.config.GetPublicBaseURL())
			adminUI.RegisterRoutes(s.app)
		}
	}

	s.app.Use(func(c fiber.Ctx) error {
		return c.Status(404).JSON(fiber.Map{
			"error": "Not Found",
			"path":  c.Path(),
		})
	})
}

func (s *Server) auditRoutesAtStartup() {
	entries := s.auditRegisteredRoutes()
	publicCount := 0
	authRequiredCount := 0
	for _, e := range entries {
		if e.Public {
			publicCount++
		}
		if e.Auth == "required" || e.Auth == "service_key" || e.Auth == "dashboard" {
			authRequiredCount++
		}
	}
	log.Info().
		Int("total", len(entries)).
		Int("public", publicCount).
		Int("auth_required", authRequiredCount).
		Msg("Route audit completed")
}

func (s *Server) startSchedulerWithLeaderElection(name string, lockID int64, startFn, stopFn func()) *scaling.LeaderElector {
	if s.config.Scaling.DisableScheduler || s.config.Scaling.WorkerOnly {
		log.Info().
			Bool("disable_scheduler", s.config.Scaling.DisableScheduler).
			Bool("worker_only", s.config.Scaling.WorkerOnly).
			Msgf("%s disabled by scaling configuration", name)
		return nil
	}
	if s.config.Scaling.EnableSchedulerLeaderElection {
		elector := scaling.NewLeaderElector(s.db.Pool(), lockID, name)
		elector.Start(startFn, stopFn)
		return elector
	}
	startFn()
	return nil
}

type branchTenantResolver struct {
	manager *tenantdb.Manager
}

func (r *branchTenantResolver) GetTenantDatabase(ctx context.Context, tenantID uuid.UUID) (*branching.TenantDatabaseInfo, error) {
	tenant, err := r.manager.GetRepository().GetTenant(ctx, tenantID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	info := &branching.TenantDatabaseInfo{
		Slug:      tenant.Slug,
		IsDefault: tenant.IsDefault,
	}
	if tenant.DBName != nil {
		info.DBName = *tenant.DBName
	}
	return info, nil
}

type branchFDWRepairer struct {
	manager *tenantdb.Manager
}

func (r *branchFDWRepairer) RepairFDWForBranch(ctx context.Context, branchDBURL string, tenantID uuid.UUID) error {
	return r.manager.RepairFDWForBranch(ctx, branchDBURL, tenantID)
}
