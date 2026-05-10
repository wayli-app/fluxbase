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
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/extensions"
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
	"github.com/nimbleflux/fluxbase/internal/realtime"
	"github.com/nimbleflux/fluxbase/internal/rpc"
	"github.com/nimbleflux/fluxbase/internal/scaling"
	"github.com/nimbleflux/fluxbase/internal/settings"
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

func (s *Server) initTenancy() {
	cfg := s.config
	db := s.db

	s.Tenancy.ServiceKey = NewServiceKeyHandler(db)

	var tenantManager *tenantdb.Manager
	var tenantStorage *tenantdb.Repository
	if cfg.Tenants.Enabled {
		tenantStorage = tenantdb.NewRepository(db.Pool())
		dbURL := cfg.Database.RuntimeConnectionString()
		tenantCfg := tenantdb.Config{
			Enabled:        cfg.Tenants.Enabled,
			DatabasePrefix: cfg.Tenants.DatabasePrefix,
			MaxTenants:     cfg.Tenants.MaxTenants,
			Pool: tenantdb.PoolConfig{
				MaxTotalConnections: cfg.Tenants.Pool.MaxTotalConnections,
				EvictionAge:         cfg.Tenants.Pool.EvictionAge,
			},
			Migrations: tenantdb.MigrationsConfig{
				CheckInterval: cfg.Tenants.Migrations.CheckInterval,
				OnCreate:      cfg.Tenants.Migrations.OnCreate,
				OnAccess:      cfg.Tenants.Migrations.OnAccess,
				Background:    cfg.Tenants.Migrations.Background,
			},
		}
		tenantManager = tenantdb.NewManager(tenantStorage, tenantCfg, db.Pool(), dbURL)
		tenantManager.SetAdminDBURL(cfg.Database.AdminConnectionString())

		if adminDBURL := cfg.Database.AdminConnectionString(); adminDBURL != "" {
			fdwCfg, fdwErr := tenantdb.ParseFDWConfig(adminDBURL)
			if fdwErr != nil {
				log.Warn().Err(fdwErr).Msg("Failed to parse FDW config, FDW disabled for tenant databases")
			} else {
				tenantManager.SetFDWConfig(fdwCfg)
				log.Info().Msg("FDW enabled for tenant databases")

				go func() {
					time.Sleep(5 * time.Second)
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
					defer cancel()
					tenantManager.UpgradeAllTenantsFDW(ctx)
				}()
			}
		}

		tenantRouter := tenantdb.NewRouter(tenantStorage, tenantCfg, db.Pool(), db.Pool(), dbURL)
		tenantRouter.SetManager(tenantManager)
		tenantManager.SetRouter(tenantRouter)

		log.Info().Msg("Multi-tenancy enabled")

		if cfg.Tenants.Declarative.Enabled && cfg.Tenants.Declarative.SchemaDir != "" {
			declarativeCfg := tenantdb.DeclarativeConfig{
				Enabled:          cfg.Tenants.Declarative.Enabled,
				SchemaDir:        cfg.Tenants.Declarative.SchemaDir,
				OnCreate:         cfg.Tenants.Declarative.OnCreate,
				OnStartup:        cfg.Tenants.Declarative.OnStartup,
				AllowDestructive: cfg.Tenants.Declarative.AllowDestructive,
			}
			declarativeSvc := tenantdb.NewDeclarativeService(
				declarativeCfg,
				"pgschema",
				cfg.Database.Host,
				cfg.Database.Port,
				cfg.Database.AdminUser,
				cfg.Database.AdminPassword,
				db.Pool(),
			)
			tenantManager.SetDeclarativeService(declarativeSvc)
			tenantManager.SetDeclarativeConfig(declarativeCfg)
			log.Info().
				Str("schema_dir", cfg.Tenants.Declarative.SchemaDir).
				Bool("on_create", cfg.Tenants.Declarative.OnCreate).
				Bool("on_startup", cfg.Tenants.Declarative.OnStartup).
				Msg("Tenant declarative schema service initialized")

			if cfg.Tenants.Declarative.OnStartup {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
					defer cancel()
					if err := tenantManager.ApplyDeclarativeSchemas(ctx); err != nil {
						log.Error().Err(err).Msg("Failed to apply tenant declarative schemas on startup")
					}
				}()
			}
		}
	}

	s.Tenancy.Manager = tenantManager
	s.Tenancy.Storage = tenantStorage
	s.Tenancy.Tenant = NewTenantHandler(db, tenantManager, tenantStorage, s.invitationService, s.emailService, cfg)

	if tenantManager != nil && tenantManager.GetRouter() != nil {
		s.Middleware.TenantDB = middleware.TenantDBMiddleware(middleware.TenantDBConfig{
			Router:  tenantManager.GetRouter(),
			Repository: tenantStorage,
		})
	}
}

func (s *Server) initSettings() {
	cfg := s.config
	db := s.db
	authService := s.authService

	unifiedSettingsService := settings.NewUnifiedService(db, cfg, cfg.EncryptionKey)
	s.Settings.Unified = unifiedSettingsService
	s.Settings.Instance = NewInstanceSettingsHandler(unifiedSettingsService)
	s.Settings.Tenant = NewTenantSettingsHandler(unifiedSettingsService, s.Tenancy.Storage)

	tenantConfigResolver := NewTenantConfigResolver(db, cfg, unifiedSettingsService)
	SetGlobalResolver(tenantConfigResolver)
	log.Info().Msg("Tenant config resolver initialized for dynamic settings")

	s.Settings.System = NewSystemSettingsHandler(s.systemSettingsService, authService.GetSettingsCache())

	customSettingsService := settings.NewCustomSettingsService(db, cfg.EncryptionKey)
	s.Settings.Custom = NewCustomSettingsHandler(customSettingsService)

	secretsService := settings.NewSecretsService(db, cfg.EncryptionKey)
	s.Settings.Service = secretsService

	userSettingsHandler := NewUserSettingsHandler(db, customSettingsService)
	userSettingsHandler.SetSecretsService(secretsService)
	s.Settings.User = userSettingsHandler

	s.Settings.App = NewAppSettingsHandler(s.systemSettingsService, authService.GetSettingsCache(), cfg)
	s.Settings.Handler = NewSettingsHandler(db)

	s.Email.Template = NewEmailTemplateHandler(db, s.emailService)

	s.Email.Settings = NewEmailSettingsHandler(
		s.systemSettingsService,
		authService.GetSettingsCache(),
		s.emailManager,
		secretsService,
		cfg,
		unifiedSettingsService,
	)

	s.emailManager.SetSettingsCache(authService.GetSettingsCache())
	s.emailManager.SetSecretsService(secretsService)
	if err := s.emailManager.RefreshFromSettings(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed to refresh email service from settings on startup")
	}

	s.Captcha.Settings = NewCaptchaSettingsHandler(
		s.systemSettingsService,
		authService.GetSettingsCache(),
		secretsService,
		&cfg.Security,
		s.captchaService,
	)

	if s.captchaService != nil {
		if err := s.captchaService.ReloadFromSettings(context.Background(), authService.GetSettingsCache(), &cfg.Security); err != nil {
			log.Warn().Err(err).Msg("Failed to refresh captcha service from settings on startup")
		}
	}
}

func (s *Server) initSchema() {
	cfg := s.config
	db := s.db

	ddlHandler := NewDDLHandler(db, nil)
	s.Schema.DDL = ddlHandler

	schemaCache := database.NewSchemaCache(db.Inspector(), 5*time.Minute)
	if s.pubSub != nil {
		schemaCache.SetPubSub(s.pubSub)
		log.Info().Msg("Schema cache configured for cross-instance invalidation via pub/sub")
	}
	if err := schemaCache.Refresh(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed to populate schema cache on startup")
	} else {
		log.Info().Int("tables", schemaCache.TableCount()).Int("views", schemaCache.ViewCount()).Msg("Schema cache populated")
	}
	s.Schema.Cache = schemaCache

	ddlHandler.SetSchemaCache(schemaCache)

	migrationsHandler := migrations.NewHandler(db, schemaCache)
	if s.Tenancy.Manager != nil && s.Tenancy.Manager.GetRouter() != nil {
		migrationsHandler.SetTenantPoolProvider(s.Tenancy.Manager.GetRouter())
	}
	s.Schema.Migrations = migrationsHandler

	s.Schema.Export = NewSchemaExportHandler(schemaCache, db.Inspector())

	internalSchemaHandler := NewInternalSchemaHandler()
	internalSchemaHandler.Initialize(cfg, db)
	s.Schema.InternalSchema = internalSchemaHandler
	log.Info().Msg("Internal schema handler initialized")

	s.rest = NewRESTHandler(db, NewQueryParser(cfg), schemaCache, cfg)
}

func (s *Server) initFunctions() {
	cfg := s.config
	db := s.db

	functionsInternalURL := cfg.BaseURL
	if functionsInternalURL == "" {
		functionsInternalURL = "http://localhost" + cfg.Server.Address
	}
	functionsHandler := functions.NewHandler(db, cfg.Functions.FunctionsDir, cfg.CORS, cfg.Auth.JWTSecret, functionsInternalURL, cfg.Deno.NpmRegistry, cfg.Deno.JsrRegistry, s.authService, s.loggingService, s.secretsStorage, cfg)
	functionsHandler.SetSettingsSecretsService(s.Settings.Service)
	functionsScheduler := functions.NewScheduler(db, cfg.Auth.JWTSecret, functionsInternalURL, s.secretsStorage, cfg)
	functionsHandler.SetScheduler(functionsScheduler)

	s.Functions.Handler = functionsHandler
	s.Functions.Scheduler = functionsScheduler
}

func (s *Server) initJobs() {
	cfg := s.config
	db := s.db

	if !cfg.Jobs.Enabled {
		return
	}

	jobsInternalURL := cfg.BaseURL
	if jobsInternalURL == "" {
		jobsInternalURL = "http://localhost" + cfg.Server.Address
	}
	log.Info().
		Str("jobs_internal_url", jobsInternalURL).
		Bool("jwt_secret_set", cfg.Auth.JWTSecret != "").
		Msg("Initializing jobs manager with SDK credentials")

	jobsManager := jobs.NewManager(&cfg.Jobs, db, cfg.Auth.JWTSecret, jobsInternalURL, s.secretsStorage, cfg)
	jobsManager.SetSettingsSecretsService(s.Settings.Service)
	s.Jobs.Manager = jobsManager

	jobsHandler, err := jobs.NewHandler(db, &cfg.Jobs, jobsManager, s.authService, s.loggingService, cfg.Deno.NpmRegistry, cfg.Deno.JsrRegistry)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize jobs handler")
	}
	s.Jobs.Handler = jobsHandler

	jobsScheduler := jobs.NewScheduler(db)
	jobsHandler.SetScheduler(jobsScheduler)
	s.Jobs.Scheduler = jobsScheduler
}

func (s *Server) initRPC() {
	cfg := s.config
	db := s.db

	if !cfg.RPC.Enabled {
		return
	}

	rpcStorage := rpc.NewStorage(db)
	rpcLoader := rpc.NewLoader(cfg.RPC.ProceduresDir)
	rpcMetrics := observability.NewMetrics()
	rpcHandler := rpc.NewHandler(db, rpcStorage, rpcLoader, rpcMetrics, &cfg.RPC, s.authService, s.loggingService, cfg)

	rpcScheduler := rpc.NewScheduler(rpcStorage, rpcHandler.GetExecutor())
	rpcHandler.SetScheduler(rpcScheduler)

	s.RPC.Handler = rpcHandler
	s.RPC.Scheduler = rpcScheduler

	log.Info().
		Str("procedures_dir", cfg.RPC.ProceduresDir).
		Bool("auto_load", cfg.RPC.AutoLoadOnBoot).
		Msg("RPC components initialized")
}

func (s *Server) initAI() {
	cfg := s.config
	db := s.db
	storageService := s.storageService
	loggingService := s.loggingService
	secretsService := s.Settings.Service
	userMgmtService := s.userMgmtService

	aiStorage := ai.NewStorage(db)
	aiStorage.SetConfig(&cfg.AI)

	vectorManager := NewVectorManager(&cfg.AI, aiStorage, db.Inspector(), db)

	var vectorHandler *VectorHandler
	vectorHandler, err := NewVectorHandler(vectorManager, db.Inspector(), db, cfg)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize vector handler")
	} else if vectorHandler.IsEmbeddingConfigured() {
		provider := cfg.AI.EmbeddingProvider
		if provider == "" {
			provider = cfg.AI.ProviderType
		}
		model := ""
		if vectorHandler.GetEmbeddingService() != nil {
			model = vectorHandler.GetEmbeddingService().DefaultModel()
		}
		log.Info().
			Str("provider", provider).
			Str("model", model).
			Bool("explicit_config", cfg.AI.EmbeddingEnabled).
			Msg("Vector handler initialized with embedding support")
	} else {
		log.Info().Msg("Vector handler initialized (embedding not available)")
	}

	var aiHandler *ai.Handler
	var aiChatHandler *ai.ChatHandler
	var aiConversations *ai.ConversationManager
	var aiMetrics *observability.Metrics
	if cfg.AI.Enabled {
		aiMetrics = observability.NewMetrics()

		aiLoader := ai.NewLoader(cfg.AI.ChatbotsDir)

		aiConversations = ai.NewConversationManager(db, cfg.AI.ConversationCacheTTL, cfg.AI.MaxConversationTurns)

		aiHandler = ai.NewHandler(aiStorage, aiLoader, &cfg.AI, vectorManager)

		var embeddingService *ai.EmbeddingService
		if vectorHandler != nil {
			embeddingService = vectorHandler.GetEmbeddingService()
		}

		aiChatHandler = ai.NewChatHandler(db, aiStorage, aiConversations, aiMetrics, &cfg.AI, embeddingService, loggingService)

		settingsResolver := ai.NewSettingsResolver(secretsService, 5*time.Minute)
		aiChatHandler.SetSettingsResolver(settingsResolver)

		log.Info().
			Str("chatbots_dir", cfg.AI.ChatbotsDir).
			Bool("auto_load", cfg.AI.AutoLoadOnBoot).
			Str("provider_type", cfg.AI.ProviderType).
			Str("provider_name", cfg.AI.ProviderName).
			Str("provider_model", cfg.AI.ProviderModel).
			Bool("rag_enabled", embeddingService != nil).
			Msg("AI components initialized")
	}

	var knowledgeBaseHandler *ai.KnowledgeBaseHandler
	var kbStorage *ai.KnowledgeBaseStorage
	var docProcessor *ai.DocumentProcessor
	var tableExportSyncService *ai.TableExportSyncService
	var ocrService *ai.OCRService
	var quotaHandler *QuotaHandler
	if cfg.AI.Enabled {
		if cfg.AI.OCREnabled {
			var ocrErr error
			ocrService, ocrErr = ai.NewOCRService(ai.OCRServiceConfig{
				Enabled:          cfg.AI.OCREnabled,
				ProviderType:     ai.OCRProviderType(cfg.AI.OCRProvider),
				DefaultLanguages: cfg.AI.OCRLanguages,
			})
			if ocrErr != nil {
				log.Warn().Err(ocrErr).Msg("Failed to initialize OCR service, OCR will be disabled")
			} else if ocrService.IsEnabled() {
				log.Info().
					Str("provider", cfg.AI.OCRProvider).
					Strs("languages", cfg.AI.OCRLanguages).
					Msg("OCR service initialized")
			}
		}

		kbStorage = ai.NewKnowledgeBaseStorage(db)

		knowledgeGraph := ai.NewKnowledgeGraph(kbStorage)
		log.Info().Msg("Knowledge graph initialized")

		entityExtractor := ai.NewRuleBasedExtractor()
		log.Info().Msg("Entity extractor initialized")

		if vectorHandler != nil && vectorHandler.GetEmbeddingService() != nil {
			docProcessor = ai.NewDocumentProcessor(kbStorage, vectorHandler.GetEmbeddingService(), entityExtractor, knowledgeGraph)
		}

		if ocrService != nil && ocrService.IsEnabled() {
			knowledgeBaseHandler = ai.NewKnowledgeBaseHandlerWithOCR(kbStorage, docProcessor, ocrService)
		} else {
			knowledgeBaseHandler = ai.NewKnowledgeBaseHandler(kbStorage, docProcessor)
		}
		knowledgeBaseHandler.SetStorageService(storageService)

		tableExporter := ai.NewTableExporter(db, docProcessor, knowledgeGraph, kbStorage)
		knowledgeBaseHandler.SetTableExporter(tableExporter)
		knowledgeBaseHandler.SetKnowledgeGraph(knowledgeGraph)
		log.Info().Msg("Table exporter initialized")

		tableExportSyncService = ai.NewTableExportSyncService(db, tableExporter, kbStorage)
		knowledgeBaseHandler.SetSyncService(tableExportSyncService)
		log.Info().Msg("Table export sync service initialized")

		aiHandler.SetKnowledgeBaseStorage(kbStorage)
		log.Info().Msg("AI handler configured with knowledge base storage")

		log.Info().
			Bool("processing_enabled", docProcessor != nil).
			Bool("ocr_enabled", ocrService != nil && ocrService.IsEnabled()).
			Bool("entity_extraction_enabled", true).
			Bool("table_export_enabled", true).
			Bool("sync_enabled", true).
			Msg("Knowledge base handler initialized")

		quotaService := ai.NewQuotaService(kbStorage)
		quotaHandler = NewQuotaHandler(quotaService, userMgmtService)
		log.Info().Msg("Quota service and handler initialized")
	}

	var internalAIHandler *InternalAIHandler
	if cfg.AI.Enabled {
		var embeddingSvc *ai.EmbeddingService
		if vectorHandler != nil {
			embeddingSvc = vectorHandler.GetEmbeddingService()
		}
		internalAIHandler = NewInternalAIHandler(aiStorage, embeddingSvc, cfg.AI.ProviderName)
		log.Info().
			Str("default_provider", cfg.AI.ProviderName).
			Bool("embedding_enabled", embeddingSvc != nil).
			Msg("Internal AI handler initialized for MCP tools/functions/jobs")
	}

	s.AI = &AIHandlers{
		Handler:         aiHandler,
		Chat:            aiChatHandler,
		Conversations:   aiConversations,
		Metrics:         aiMetrics,
		KnowledgeBase:   knowledgeBaseHandler,
		KBStorage:       kbStorage,
		DocProcessor:    docProcessor,
		TableExportSync: tableExportSyncService,
		VectorManager:   vectorManager,
		VectorHandler:   vectorHandler,
		Internal:        internalAIHandler,
	}

	s.Quota.Handler = quotaHandler

	if cfg.AI.Enabled && cfg.AI.AutoLoadOnBoot && s.AI.Handler != nil {
		if err := s.AI.Handler.AutoLoadChatbots(context.Background()); err != nil {
			log.Error().Err(err).Msg("Failed to auto-load AI chatbots")
		} else {
			log.Info().Msg("AI chatbots auto-loaded successfully")
		}
	}
}

func (s *Server) initRealtime() {
	cfg := s.config
	db := s.db
	ps := s.pubSub
	authService := s.authService
	storageService := s.storageService
	loggingService := s.loggingService

	s.Realtime.Admin = NewRealtimeAdminHandler(db)

	realtimeManager := realtime.NewManagerWithConfig(context.Background(), realtime.ManagerConfig{
		MaxConnections:         cfg.Realtime.MaxConnections,
		MaxConnectionsPerUser:  cfg.Realtime.MaxConnectionsPerUser,
		MaxConnectionsPerIP:    cfg.Realtime.MaxConnectionsPerIP,
		ClientMessageQueueSize: cfg.Realtime.ClientMessageQueueSize,
		SlowClientThreshold:    cfg.Realtime.SlowClientThreshold,
		SlowClientTimeout:      cfg.Realtime.SlowClientTimeout,
	})
	realtimeManager.SetBaseConfig(cfg)

	if ps != nil {
		realtimeManager.SetPubSub(ps)
	}

	realtimeAuthAdapter := realtime.NewAuthServiceAdapter(authService)
	realtimeSubManager := realtime.NewSubscriptionManagerWithConfig(
		realtime.NewPgxSubscriptionDB(db.Pool()),
		realtime.RLSCacheConfig{
			MaxSize: cfg.Realtime.RLSCacheSize,
			TTL:     cfg.Realtime.RLSCacheTTL,
		},
	)
	realtimeHandler := realtime.NewRealtimeHandler(realtimeManager, realtimeAuthAdapter, realtimeSubManager)
	realtimeListener := realtime.NewListenerPool(
		db.Pool(),
		realtimeHandler,
		realtimeSubManager,
		ps,
		realtime.ListenerPoolConfig{
			PoolSize:    cfg.Realtime.ListenerPoolSize,
			WorkerCount: cfg.Realtime.NotificationWorkers,
			QueueSize:   1000,
		},
	)

	s.Realtime.Manager = realtimeManager
	s.Realtime.Handler = realtimeHandler
	s.Realtime.Listener = realtimeListener

	monitoringHandler := NewMonitoringHandler(db, realtimeHandler, storageService.Provider)
	if loggingService != nil {
		monitoringHandler.SetLoggingService(loggingService)
	}
	s.Monitoring.Handler = monitoringHandler
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

func (s *Server) initBranching() {
	cfg := s.config
	db := s.db

	if !cfg.Branching.Enabled {
		return
	}

	branchStorage := branching.NewStorage(db, cfg.EncryptionKey)
	dbURL := cfg.Database.RuntimeConnectionString()
	branchManager, err := branching.NewManager(branchStorage, cfg.Branching, db.Pool(), dbURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize branch manager")
	}
	branchRouter := branching.NewRouter(branchStorage, cfg.Branching, db.Pool(), dbURL)

	s.Branching.Manager = branchManager
	s.Branching.Router = branchRouter
	s.Branching.Handler = NewBranchHandler(branchManager, branchRouter, cfg.Branching)
	s.Branching.GitHub = NewGitHubWebhookHandler(branchManager, branchRouter, cfg.Branching)

	if s.Tenancy.Manager != nil {
		branchManager.SetTenantResolver(&branchTenantResolver{manager: s.Tenancy.Manager})
		branchManager.SetFDWRepairer(&branchFDWRepairer{manager: s.Tenancy.Manager})
	}

	if cfg.Branching.AutoDeleteAfter > 0 {
		cleanupInterval := cfg.Branching.AutoDeleteAfter
		if cleanupInterval < time.Hour {
			cleanupInterval = time.Hour
		}
		s.Branching.Scheduler = branching.NewCleanupScheduler(branchManager, branchRouter, cleanupInterval)
		log.Info().
			Dur("interval", cleanupInterval).
			Dur("auto_delete_after", cfg.Branching.AutoDeleteAfter).
			Msg("Branch cleanup scheduler initialized")
	}

	log.Info().
		Int("max_branches", cfg.Branching.MaxTotalBranches).
		Str("default_clone_mode", cfg.Branching.DefaultDataCloneMode).
		Msg("Database Branching enabled")
}

func (s *Server) initGraphQL() {
	cfg := s.config
	db := s.db

	if !cfg.GraphQL.Enabled {
		return
	}

	s.GraphQL.Handler = NewGraphQLHandler(db, s.Schema.Cache, &cfg.GraphQL, cfg)

	if s.Schema.DDL != nil && s.GraphQL.Handler != nil {
		s.Schema.DDL.SetGraphQLInvalidator(s.GraphQL.Handler.InvalidateSchema)
	}

	log.Info().
		Int("max_depth", cfg.GraphQL.MaxDepth).
		Int("max_complexity", cfg.GraphQL.MaxComplexity).
		Bool("introspection", cfg.GraphQL.Introspection).
		Msg("GraphQL API enabled")
	if cfg.GraphQL.Introspection {
		log.Warn().Msg("GraphQL introspection is enabled — consider setting graphql.introspection to false in production")
	}
}

func (s *Server) initExtensions() {
	s.Extensions.Handler = extensions.NewHandler(extensions.NewService(s.db))
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
