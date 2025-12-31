package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/adminui"
	"github.com/fluxbase-eu/fluxbase/internal/ai"
	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/branching"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/email"
	"github.com/fluxbase-eu/fluxbase/internal/extensions"
	"github.com/fluxbase-eu/fluxbase/internal/functions"
	"github.com/fluxbase-eu/fluxbase/internal/jobs"
	"github.com/fluxbase-eu/fluxbase/internal/logging"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	mcpresources "github.com/fluxbase-eu/fluxbase/internal/mcp/resources"
	mcptools "github.com/fluxbase-eu/fluxbase/internal/mcp/tools"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/fluxbase-eu/fluxbase/internal/migrations"
	"github.com/fluxbase-eu/fluxbase/internal/observability"
	"github.com/fluxbase-eu/fluxbase/internal/pubsub"
	"github.com/fluxbase-eu/fluxbase/internal/ratelimit"
	"github.com/fluxbase-eu/fluxbase/internal/realtime"
	"github.com/fluxbase-eu/fluxbase/internal/rpc"
	"github.com/fluxbase-eu/fluxbase/internal/scaling"
	"github.com/fluxbase-eu/fluxbase/internal/secrets"
	"github.com/fluxbase-eu/fluxbase/internal/settings"
	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/fluxbase-eu/fluxbase/internal/webhook"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/rs/zerolog/log"
)

// Server represents the HTTP server
type Server struct {
	app                   *fiber.App
	config                *config.Config
	db                    *database.Connection
	tracer                *observability.Tracer
	rest                  *RESTHandler
	authHandler           *AuthHandler
	adminAuthHandler      *AdminAuthHandler
	dashboardAuthHandler  *DashboardAuthHandler
	clientKeyService      *auth.ClientKeyService // Added for service-wide access
	clientKeyHandler      *ClientKeyHandler
	storageHandler        *StorageHandler
	webhookHandler        *WebhookHandler
	monitoringHandler     *MonitoringHandler
	userManagementHandler *UserManagementHandler
	invitationHandler     *InvitationHandler
	ddlHandler            *DDLHandler
	oauthProviderHandler  *OAuthProviderHandler
	oauthHandler          *OAuthHandler
	samlProviderHandler   *SAMLProviderHandler
	samlService           *auth.SAMLService
	adminSessionHandler   *AdminSessionHandler
	systemSettingsHandler *SystemSettingsHandler
	customSettingsHandler *CustomSettingsHandler
	userSettingsHandler   *UserSettingsHandler
	appSettingsHandler    *AppSettingsHandler
	settingsHandler       *SettingsHandler
	secretsService        *settings.SecretsService
	emailTemplateHandler  *EmailTemplateHandler
	emailSettingsHandler  *EmailSettingsHandler
	sqlHandler            *SQLHandler
	functionsHandler      *functions.Handler
	functionsScheduler    *functions.Scheduler
	jobsHandler           *jobs.Handler
	jobsManager           *jobs.Manager
	jobsScheduler         *jobs.Scheduler
	migrationsHandler     *migrations.Handler
	realtimeManager       *realtime.Manager
	realtimeHandler       *realtime.RealtimeHandler
	realtimeListener      *realtime.Listener
	webhookTriggerService *webhook.TriggerService
	aiHandler             *ai.Handler
	aiChatHandler         *ai.ChatHandler
	aiConversations       *ai.ConversationManager
	aiMetrics             *observability.Metrics
	knowledgeBaseHandler  *ai.KnowledgeBaseHandler
	rpcHandler            *rpc.Handler
	rpcScheduler          *rpc.Scheduler
	graphqlHandler        *GraphQLHandler
	extensionsHandler     *extensions.Handler
	vectorHandler         *VectorHandler
	loggingService        *logging.Service
	loggingHandler        *LoggingHandler
	retentionService      *logging.RetentionService
	schemaCache           *database.SchemaCache
	secretsHandler        *secrets.Handler
	secretsStorage        *secrets.Storage
	serviceKeyHandler     *ServiceKeyHandler
	mcpHandler            *mcp.Handler

	// Database branching components
	branchManager   *branching.Manager
	branchRouter    *branching.Router
	branchHandler   *BranchHandler
	githubWebhook   *GitHubWebhookHandler
	branchScheduler *branching.CleanupScheduler

	// Leader election for schedulers (used in multi-instance deployments)
	jobsSchedulerLeader      *scaling.LeaderElector
	functionsSchedulerLeader *scaling.LeaderElector
	rpcSchedulerLeader       *scaling.LeaderElector
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, db *database.Connection, version string) *Server {
	// Create Fiber app with config
	app := fiber.New(fiber.Config{
		ServerHeader:          "Fluxbase",
		AppName:               fmt.Sprintf("Fluxbase v%s", version),
		BodyLimit:             cfg.Server.BodyLimit,
		StreamRequestBody:     true, // Required for chunked upload streaming
		ReadTimeout:           cfg.Server.ReadTimeout,
		WriteTimeout:          cfg.Server.WriteTimeout,
		IdleTimeout:           cfg.Server.IdleTimeout,
		DisableStartupMessage: !cfg.Debug,
		ErrorHandler:          customErrorHandler,
		Prefork:               false,
	})

	// In debug mode, add no-cache headers to prevent browser from caching
	// connection failures during server restarts
	if cfg.Debug {
		app.Use(func(c *fiber.Ctx) error {
			c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			c.Set("Pragma", "no-cache")
			c.Set("Expires", "0")
			return c.Next()
		})
	}

	// Initialize OpenTelemetry tracer
	tracerCfg := observability.TracerConfig{
		Enabled:     cfg.Tracing.Enabled,
		Endpoint:    cfg.Tracing.Endpoint,
		ServiceName: cfg.Tracing.ServiceName,
		Environment: cfg.Tracing.Environment,
		SampleRate:  cfg.Tracing.SampleRate,
		Insecure:    cfg.Tracing.Insecure,
	}
	tracer, err := observability.NewTracer(context.Background(), tracerCfg)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize OpenTelemetry tracer, tracing will be disabled")
	}

	// Initialize rate limit store based on scaling configuration
	rateLimitStore, err := ratelimit.NewStore(&cfg.Scaling, db.Pool())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize rate limit store, falling back to memory")
	} else {
		ratelimit.SetGlobalStore(rateLimitStore)
		log.Info().Str("backend", cfg.Scaling.Backend).Msg("Rate limit store initialized")
	}

	// Initialize pub/sub for cross-instance communication
	ps, err := pubsub.NewPubSub(&cfg.Scaling, db.Pool())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize pub/sub, cross-instance broadcasting disabled")
	} else {
		pubsub.SetGlobalPubSub(ps)
		log.Info().Str("backend", cfg.Scaling.Backend).Msg("Pub/sub initialized for cross-instance broadcasting")
	}

	// Initialize email manager (handles dynamic refresh from settings)
	// The settings cache and secrets service will be injected later once they're initialized
	emailManager := email.NewManager(&cfg.Email, nil, nil)
	// Get a service wrapper that delegates to the manager's current service
	emailService := emailManager.WrapAsService()

	// Initialize auth service (use public URL for user-facing links like magic links, password resets)
	authService := auth.NewService(db, &cfg.Auth, emailService, cfg.GetPublicBaseURL())

	// Initialize API key service
	clientKeyService := auth.NewClientKeyService(db.Pool())

	// Initialize storage service (use public URL for signed URLs that users will access)
	storageService, err := storage.NewService(&cfg.Storage, cfg.GetPublicBaseURL(), cfg.Auth.JWTSecret)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize storage service")
	}

	// Ensure default buckets exist
	if err := storageService.EnsureDefaultBuckets(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed to ensure default buckets")
	}

	// Initialize central logging service
	var loggingService *logging.Service
	var loggingHandler *LoggingHandler
	var retentionService *logging.RetentionService
	if cfg.Logging.ConsoleEnabled || cfg.Logging.Backend != "" {
		loggingService, err = logging.New(&cfg.Logging, db, storageService.Provider, ps)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to initialize central logging service, continuing with default logging")
		} else {
			// Replace zerolog writer with the central logging writer
			log.Logger = log.Output(loggingService.Writer())
			log.Info().
				Str("backend", cfg.Logging.Backend).
				Bool("pubsub_enabled", cfg.Logging.PubSubEnabled).
				Int("batch_size", cfg.Logging.BatchSize).
				Msg("Central logging service initialized")

			// Create logging handler for API routes
			loggingHandler = NewLoggingHandler(loggingService)

			// Create retention cleanup service
			if cfg.Logging.RetentionEnabled {
				retentionService = logging.NewRetentionService(&cfg.Logging, loggingService.Storage())
			}
		}
	}

	// Initialize webhook service
	webhookService := webhook.NewWebhookService(db)
	// Allow private IPs in debug mode (for local testing with localhost webhooks)
	// SECURITY WARNING: This bypasses SSRF protection - NEVER enable debug mode in production!
	webhookService.AllowPrivateIPs = cfg.Debug
	if cfg.Debug {
		log.Warn().Msg("SECURITY: Debug mode enabled - webhook SSRF protection is DISABLED. Do NOT use in production!")
	}

	// Initialize webhook trigger service (4 workers)
	webhookTriggerService := webhook.NewTriggerService(db, webhookService, 4)

	// Initialize user management service (use public URL for password reset links, etc.)
	userMgmtService := auth.NewUserManagementService(
		auth.NewUserRepository(db),
		auth.NewSessionRepository(db),
		auth.NewPasswordHasherWithConfig(auth.PasswordHasherConfig{MinLength: cfg.Auth.PasswordMinLen, Cost: cfg.Auth.BcryptCost}),
		emailService,
		cfg.GetPublicBaseURL(),
	)

	// Create CAPTCHA service
	captchaService, err := auth.NewCaptchaService(&cfg.Security.Captcha)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize CAPTCHA service - CAPTCHA protection disabled")
		captchaService = nil
	}

	// Create handlers
	authHandler := NewAuthHandler(authService, captchaService)
	// Create dashboard JWT manager first (shared between auth service and handler)
	dashboardJWTManager := auth.NewJWTManager(cfg.Auth.JWTSecret, 24*time.Hour, 168*time.Hour)
	dashboardAuthService := auth.NewDashboardAuthService(db, dashboardJWTManager, cfg.Auth.TOTPIssuer)
	systemSettingsService := auth.NewSystemSettingsService(db)
	adminAuthHandler := NewAdminAuthHandler(authService, auth.NewUserRepository(db), dashboardAuthService, systemSettingsService, cfg)
	// Note: dashboardAuthHandler is initialized later after samlService is created
	clientKeyHandler := NewClientKeyHandler(clientKeyService)
	storageHandler := NewStorageHandler(storageService, db, &cfg.Storage.Transforms)
	webhookHandler := NewWebhookHandler(webhookService)

	// Initialize secrets storage and handler
	secretsStorage := secrets.NewStorage(db, cfg.EncryptionKey)
	secretsHandler := secrets.NewHandler(secretsStorage)

	userMgmtHandler := NewUserManagementHandler(userMgmtService, authService)
	invitationService := auth.NewInvitationService(db)
	invitationHandler := NewInvitationHandler(invitationService, dashboardAuthService, emailService, cfg.GetPublicBaseURL())
	ddlHandler := NewDDLHandler(db)
	serviceKeyHandler := NewServiceKeyHandler(db.Pool())
	oauthProviderHandler := NewOAuthProviderHandler(db.Pool(), authService.GetSettingsCache())
	jwtManager := auth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry, cfg.Auth.RefreshExpiry)
	// Use public URL for OAuth callbacks (these are redirects from external OAuth providers)
	oauthHandler := NewOAuthHandler(db.Pool(), authService, jwtManager, cfg.GetPublicBaseURL(), cfg.EncryptionKey)

	// Initialize SAML service and handler
	var samlService *auth.SAMLService
	var samlProviderHandler *SAMLProviderHandler
	var samlErr error
	samlService, samlErr = auth.NewSAMLService(db.Pool(), cfg.GetPublicBaseURL(), cfg.Auth.SAMLProviders)
	if samlErr != nil {
		log.Warn().Err(samlErr).Msg("Failed to initialize SAML service from config")
	}
	// Load SAML providers from database
	if samlService != nil {
		if err := samlService.LoadProvidersFromDB(context.Background()); err != nil {
			log.Warn().Err(err).Msg("Failed to load SAML providers from database")
		}
	}
	samlProviderHandler = NewSAMLProviderHandler(db.Pool(), samlService)
	// Initialize dashboard auth handler now that samlService is available
	dashboardAuthHandler := NewDashboardAuthHandler(dashboardAuthService, dashboardJWTManager, db, samlService, cfg.GetPublicBaseURL())
	adminSessionHandler := NewAdminSessionHandler(auth.NewSessionRepository(db))
	systemSettingsHandler := NewSystemSettingsHandler(systemSettingsService, authService.GetSettingsCache())
	customSettingsService := settings.NewCustomSettingsService(db, cfg.EncryptionKey)
	customSettingsHandler := NewCustomSettingsHandler(customSettingsService)
	userSettingsHandler := NewUserSettingsHandler(customSettingsService)
	secretsService := settings.NewSecretsService(db, cfg.EncryptionKey)
	userSettingsHandler.SetSecretsService(secretsService)
	appSettingsHandler := NewAppSettingsHandler(systemSettingsService, authService.GetSettingsCache(), cfg)
	settingsHandler := NewSettingsHandler(db)
	emailTemplateHandler := NewEmailTemplateHandler(db, emailService)

	// Initialize email settings handler with settings cache for dynamic configuration
	emailSettingsHandler := NewEmailSettingsHandler(
		systemSettingsService,
		authService.GetSettingsCache(),
		emailManager,
		secretsService,
		&cfg.Email,
	)

	// Refresh email manager with settings cache and secrets service now that they're available
	emailManager.SetSettingsCache(authService.GetSettingsCache())
	emailManager.SetSecretsService(secretsService)
	if err := emailManager.RefreshFromSettings(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed to refresh email service from settings on startup")
	}
	sqlHandler := NewSQLHandler(db.Pool(), authService)

	// Determine public URL for functions SDK client
	// For edge functions running inside the container, they should use the internal BaseURL
	// to communicate with the API server (faster, avoids external network hops)
	functionsInternalURL := cfg.BaseURL
	if functionsInternalURL == "" {
		functionsInternalURL = "http://localhost" + cfg.Server.Address
	}
	functionsHandler := functions.NewHandler(db, cfg.Functions.FunctionsDir, cfg.CORS, cfg.Auth.JWTSecret, functionsInternalURL, authService, loggingService, secretsStorage)
	functionsHandler.SetSettingsSecretsService(secretsService)
	functionsScheduler := functions.NewScheduler(db, cfg.Auth.JWTSecret, functionsInternalURL, secretsStorage)
	functionsHandler.SetScheduler(functionsScheduler)

	// Only create jobs components if jobs are enabled
	var jobsManager *jobs.Manager
	var jobsHandler *jobs.Handler
	var jobsScheduler *jobs.Scheduler
	if cfg.Jobs.Enabled {
		// Determine internal URL for jobs SDK client
		// Jobs run inside the container and should use the internal URL
		jobsInternalURL := cfg.BaseURL
		if jobsInternalURL == "" {
			// Fallback to server address
			jobsInternalURL = "http://localhost" + cfg.Server.Address
		}
		log.Info().
			Str("jobs_internal_url", jobsInternalURL).
			Bool("jwt_secret_set", cfg.Auth.JWTSecret != "").
			Msg("Initializing jobs manager with SDK credentials")
		jobsManager = jobs.NewManager(&cfg.Jobs, db, cfg.Auth.JWTSecret, jobsInternalURL, secretsStorage)
		jobsManager.SetSettingsSecretsService(secretsService)
		var err error
		jobsHandler, err = jobs.NewHandler(db, &cfg.Jobs, jobsManager, authService, loggingService)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize jobs handler")
		}
		// Create jobs scheduler for cron-based job execution
		jobsScheduler = jobs.NewScheduler(db)
		jobsHandler.SetScheduler(jobsScheduler)
	}

	// Create schema cache for dynamic REST API routing (5 minute TTL)
	schemaCache := database.NewSchemaCache(db.Inspector(), 5*time.Minute)
	// Configure PubSub for cross-instance cache invalidation
	if ps != nil {
		schemaCache.SetPubSub(ps)
		log.Info().Msg("Schema cache configured for cross-instance invalidation via pub/sub")
	}
	// Populate cache on startup
	if err := schemaCache.Refresh(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed to populate schema cache on startup")
	} else {
		log.Info().Int("tables", schemaCache.TableCount()).Int("views", schemaCache.ViewCount()).Msg("Schema cache populated")
	}

	migrationsHandler := migrations.NewHandler(db, schemaCache)

	// Create vector search handler (for pgvector support) - create early for embedding service sharing
	// Embedding can be enabled explicitly (EmbeddingEnabled=true) or via fallback from AI provider
	var vectorHandler *VectorHandler
	vectorHandler, err = NewVectorHandler(&cfg.AI, db.Inspector(), db)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize vector handler")
	} else if vectorHandler.IsEmbeddingConfigured() {
		// Embedding is available (either explicitly configured or via AI provider fallback)
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

	// Create AI components (only if AI is enabled)
	var aiHandler *ai.Handler
	var aiChatHandler *ai.ChatHandler
	var aiConversations *ai.ConversationManager
	var aiMetrics *observability.Metrics
	if cfg.AI.Enabled {
		// Create AI metrics
		aiMetrics = observability.NewMetrics()

		// Create AI storage
		aiStorage := ai.NewStorage(db)
		aiStorage.SetConfig(&cfg.AI)

		// Create AI loader
		aiLoader := ai.NewLoader(cfg.AI.ChatbotsDir)

		// Create conversation manager
		aiConversations = ai.NewConversationManager(db, cfg.AI.ConversationCacheTTL, cfg.AI.MaxConversationTurns)

		// Create AI handler for admin endpoints
		aiHandler = ai.NewHandler(aiStorage, aiLoader, &cfg.AI)

		// Get embedding service from vector handler (if available) for RAG support
		var embeddingService *ai.EmbeddingService
		if vectorHandler != nil {
			embeddingService = vectorHandler.GetEmbeddingService()
		}

		// Create AI chat handler for WebSocket with RAG support
		aiChatHandler = ai.NewChatHandler(db, aiStorage, aiConversations, aiMetrics, &cfg.AI, embeddingService, loggingService)

		log.Info().
			Str("chatbots_dir", cfg.AI.ChatbotsDir).
			Bool("auto_load", cfg.AI.AutoLoadOnBoot).
			Str("provider_type", cfg.AI.ProviderType).
			Str("provider_name", cfg.AI.ProviderName).
			Str("provider_model", cfg.AI.ProviderModel).
			Bool("rag_enabled", embeddingService != nil).
			Msg("AI components initialized")
	}

	// Create knowledge base handler for RAG management
	var knowledgeBaseHandler *ai.KnowledgeBaseHandler
	var ocrService *ai.OCRService
	if cfg.AI.Enabled {
		// Initialize OCR service for image-based PDF extraction
		if cfg.AI.OCREnabled {
			var err error
			ocrService, err = ai.NewOCRService(ai.OCRServiceConfig{
				Enabled:          cfg.AI.OCREnabled,
				ProviderType:     ai.OCRProviderType(cfg.AI.OCRProvider),
				DefaultLanguages: cfg.AI.OCRLanguages,
			})
			if err != nil {
				log.Warn().Err(err).Msg("Failed to initialize OCR service, OCR will be disabled")
			} else if ocrService.IsEnabled() {
				log.Info().
					Str("provider", cfg.AI.OCRProvider).
					Strs("languages", cfg.AI.OCRLanguages).
					Msg("OCR service initialized")
			}
		}

		kbStorage := ai.NewKnowledgeBaseStorage(db)
		var docProcessor *ai.DocumentProcessor
		if vectorHandler != nil && vectorHandler.GetEmbeddingService() != nil {
			docProcessor = ai.NewDocumentProcessor(kbStorage, vectorHandler.GetEmbeddingService())
		}

		// Use OCR-enabled handler if OCR service is available
		if ocrService != nil && ocrService.IsEnabled() {
			knowledgeBaseHandler = ai.NewKnowledgeBaseHandlerWithOCR(kbStorage, docProcessor, ocrService)
		} else {
			knowledgeBaseHandler = ai.NewKnowledgeBaseHandler(kbStorage, docProcessor)
		}
		knowledgeBaseHandler.SetStorageService(storageService)
		log.Info().
			Bool("processing_enabled", docProcessor != nil).
			Bool("ocr_enabled", ocrService != nil && ocrService.IsEnabled()).
			Msg("Knowledge base handler initialized")
	}

	// Create RPC components (only if RPC is enabled)
	var rpcHandler *rpc.Handler
	var rpcScheduler *rpc.Scheduler
	if cfg.RPC.Enabled {
		rpcStorage := rpc.NewStorage(db)
		rpcLoader := rpc.NewLoader(cfg.RPC.ProceduresDir)
		rpcMetrics := observability.NewMetrics()
		rpcHandler = rpc.NewHandler(db, rpcStorage, rpcLoader, rpcMetrics, &cfg.RPC, authService, loggingService)

		// Create RPC scheduler and wire it to handler
		rpcScheduler = rpc.NewScheduler(rpcStorage, rpcHandler.GetExecutor())
		rpcHandler.SetScheduler(rpcScheduler)

		log.Info().
			Str("procedures_dir", cfg.RPC.ProceduresDir).
			Bool("auto_load", cfg.RPC.AutoLoadOnBoot).
			Msg("RPC components initialized")
	}

	// Create realtime components
	realtimeManager := realtime.NewManager(context.Background())

	// Set up cross-instance broadcasting via pub/sub (if configured)
	if ps != nil {
		realtimeManager.SetPubSub(ps)
	}

	realtimeAuthAdapter := realtime.NewAuthServiceAdapter(authService)
	realtimeSubManager := realtime.NewSubscriptionManager(db.Pool())
	realtimeHandler := realtime.NewRealtimeHandler(realtimeManager, realtimeAuthAdapter, realtimeSubManager)
	realtimeListener := realtime.NewListener(db.Pool(), realtimeHandler, realtimeSubManager, ps)

	// Create monitoring handler
	monitoringHandler := NewMonitoringHandler(db.Pool(), realtimeHandler, storageService.Provider)

	// Create server instance
	server := &Server{
		app:                   app,
		config:                cfg,
		db:                    db,
		tracer:                tracer,
		rest:                  NewRESTHandler(db, NewQueryParser(cfg), schemaCache),
		authHandler:           authHandler,
		adminAuthHandler:      adminAuthHandler,
		dashboardAuthHandler:  dashboardAuthHandler,
		clientKeyService:      clientKeyService, // Added for service-wide access
		clientKeyHandler:      clientKeyHandler,
		storageHandler:        storageHandler,
		webhookHandler:        webhookHandler,
		monitoringHandler:     monitoringHandler,
		userManagementHandler: userMgmtHandler,
		invitationHandler:     invitationHandler,
		ddlHandler:            ddlHandler,
		oauthProviderHandler:  oauthProviderHandler,
		oauthHandler:          oauthHandler,
		samlProviderHandler:   samlProviderHandler,
		samlService:           samlService,
		adminSessionHandler:   adminSessionHandler,
		systemSettingsHandler: systemSettingsHandler,
		customSettingsHandler: customSettingsHandler,
		userSettingsHandler:   userSettingsHandler,
		appSettingsHandler:    appSettingsHandler,
		settingsHandler:       settingsHandler,
		secretsService:        secretsService,
		emailTemplateHandler:  emailTemplateHandler,
		emailSettingsHandler:  emailSettingsHandler,
		sqlHandler:            sqlHandler,
		functionsHandler:      functionsHandler,
		functionsScheduler:    functionsScheduler,
		jobsHandler:           jobsHandler,
		jobsManager:           jobsManager,
		jobsScheduler:         jobsScheduler,
		migrationsHandler:     migrationsHandler,
		realtimeManager:       realtimeManager,
		realtimeHandler:       realtimeHandler,
		realtimeListener:      realtimeListener,
		webhookTriggerService: webhookTriggerService,
		aiHandler:             aiHandler,
		aiChatHandler:         aiChatHandler,
		aiConversations:       aiConversations,
		aiMetrics:             aiMetrics,
		knowledgeBaseHandler:  knowledgeBaseHandler,
		rpcHandler:            rpcHandler,
		rpcScheduler:          rpcScheduler,
		extensionsHandler:     extensions.NewHandler(extensions.NewService(db)),
		vectorHandler:         vectorHandler,
		loggingService:        loggingService,
		loggingHandler:        loggingHandler,
		retentionService:      retentionService,
		schemaCache:           schemaCache,
		secretsHandler:        secretsHandler,
		secretsStorage:        secretsStorage,
		serviceKeyHandler:     serviceKeyHandler,
		mcpHandler:            mcp.NewHandler(&cfg.MCP, db),
	}

	// Initialize MCP Server if enabled
	if cfg.MCP.Enabled {
		server.setupMCPServer(schemaCache, storageService, functionsHandler, rpcHandler)
		log.Info().
			Str("base_path", cfg.MCP.BasePath).
			Dur("session_timeout", cfg.MCP.SessionTimeout).
			Msg("MCP Server enabled")
	}

	// Initialize Database Branching if enabled
	if cfg.Branching.Enabled {
		branchStorage := branching.NewStorage(db.Pool())
		dbURL := cfg.Database.RuntimeConnectionString()
		branchManager, err := branching.NewManager(branchStorage, cfg.Branching, db.Pool(), dbURL)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize branch manager")
		}
		branchRouter := branching.NewRouter(branchStorage, cfg.Branching, db.Pool(), dbURL)

		server.branchManager = branchManager
		server.branchRouter = branchRouter
		server.branchHandler = NewBranchHandler(branchManager, branchRouter, cfg.Branching)
		server.githubWebhook = NewGitHubWebhookHandler(branchManager, branchRouter, cfg.Branching)

		// Initialize cleanup scheduler if auto_delete_after is set
		if cfg.Branching.AutoDeleteAfter > 0 {
			// Use auto_delete_after as the interval, or default to hourly if it's very short
			cleanupInterval := cfg.Branching.AutoDeleteAfter
			if cleanupInterval < time.Hour {
				cleanupInterval = time.Hour
			}
			server.branchScheduler = branching.NewCleanupScheduler(branchManager, branchRouter, cleanupInterval)
			log.Info().
				Dur("interval", cleanupInterval).
				Dur("auto_delete_after", cfg.Branching.AutoDeleteAfter).
				Msg("Branch cleanup scheduler initialized")
		}

		log.Info().
			Int("max_branches", cfg.Branching.MaxTotalBranches).
			Int("max_per_user", cfg.Branching.MaxBranchesPerUser).
			Str("default_clone_mode", cfg.Branching.DefaultDataCloneMode).
			Msg("Database Branching enabled")
	}

	// Create GraphQL handler (if enabled)
	if cfg.GraphQL.Enabled {
		server.graphqlHandler = NewGraphQLHandler(db, schemaCache, &cfg.GraphQL)
		log.Info().
			Int("max_depth", cfg.GraphQL.MaxDepth).
			Int("max_complexity", cfg.GraphQL.MaxComplexity).
			Bool("introspection", cfg.GraphQL.Introspection).
			Msg("GraphQL API enabled")
	}

	// Start realtime listener (unless disabled or in worker-only mode)
	if !cfg.Scaling.DisableRealtime && !cfg.Scaling.WorkerOnly {
		if err := realtimeListener.Start(); err != nil {
			log.Error().Err(err).Msg("Failed to start realtime listener")
		}
	} else {
		log.Info().
			Bool("disable_realtime", cfg.Scaling.DisableRealtime).
			Bool("worker_only", cfg.Scaling.WorkerOnly).
			Msg("Realtime listener disabled by scaling configuration")
	}

	// Start edge functions scheduler (respects scaling configuration)
	if !cfg.Scaling.DisableScheduler && !cfg.Scaling.WorkerOnly {
		if cfg.Scaling.EnableSchedulerLeaderElection {
			// Use leader election - only the leader will run the scheduler
			server.functionsSchedulerLeader = scaling.NewLeaderElector(
				db.Pool(),
				scaling.FunctionsSchedulerLockID,
				"functions-scheduler",
			)
			server.functionsSchedulerLeader.Start(
				func() {
					// Became leader - start the scheduler
					log.Info().Msg("This instance is now the functions scheduler leader")
					if err := functionsScheduler.Start(); err != nil {
						log.Error().Err(err).Msg("Failed to start edge functions scheduler")
					}
				},
				func() {
					// Lost leadership - stop the scheduler
					log.Warn().Msg("Lost functions scheduler leadership - stopping scheduler")
					functionsScheduler.Stop()
				},
			)
		} else {
			// No leader election - start scheduler directly
			if err := functionsScheduler.Start(); err != nil {
				log.Error().Err(err).Msg("Failed to start edge functions scheduler")
			}
		}
	} else {
		log.Info().
			Bool("disable_scheduler", cfg.Scaling.DisableScheduler).
			Bool("worker_only", cfg.Scaling.WorkerOnly).
			Msg("Edge functions scheduler disabled by scaling configuration")
	}

	// Start jobs manager and scheduler
	if cfg.Jobs.Enabled && jobsManager != nil {
		// Job workers can run on any instance (including worker-only mode)
		// The scheduler should respect the scaling configuration
		workerCount := cfg.Jobs.EmbeddedWorkerCount
		if workerCount <= 0 {
			workerCount = 4 // Default to 4 workers if not configured
		}
		if err := jobsManager.Start(context.Background(), workerCount); err != nil {
			log.Error().Err(err).Msg("Failed to start jobs manager")
		} else {
			log.Info().Int("workers", workerCount).Msg("Jobs manager started successfully")
		}

		// Start jobs scheduler for cron-based execution (respects scaling configuration)
		if jobsScheduler != nil {
			if !cfg.Scaling.DisableScheduler && !cfg.Scaling.WorkerOnly {
				if cfg.Scaling.EnableSchedulerLeaderElection {
					// Use leader election - only the leader will run the scheduler
					server.jobsSchedulerLeader = scaling.NewLeaderElector(
						db.Pool(),
						scaling.JobsSchedulerLockID,
						"jobs-scheduler",
					)
					server.jobsSchedulerLeader.Start(
						func() {
							// Became leader - start the scheduler
							log.Info().Msg("This instance is now the jobs scheduler leader")
							if err := jobsScheduler.Start(); err != nil {
								log.Error().Err(err).Msg("Failed to start jobs scheduler")
							}
						},
						func() {
							// Lost leadership - stop the scheduler
							log.Warn().Msg("Lost jobs scheduler leadership - stopping scheduler")
							jobsScheduler.Stop()
						},
					)
				} else {
					// No leader election - start scheduler directly
					if err := jobsScheduler.Start(); err != nil {
						log.Error().Err(err).Msg("Failed to start jobs scheduler")
					}
				}
			} else {
				log.Info().
					Bool("disable_scheduler", cfg.Scaling.DisableScheduler).
					Bool("worker_only", cfg.Scaling.WorkerOnly).
					Msg("Jobs scheduler disabled by scaling configuration (workers still active)")
			}
		}
	}

	// Start RPC scheduler for cron-based procedure execution (respects scaling configuration)
	if cfg.RPC.Enabled && rpcScheduler != nil {
		if !cfg.Scaling.DisableScheduler && !cfg.Scaling.WorkerOnly {
			if cfg.Scaling.EnableSchedulerLeaderElection {
				// Use leader election - only the leader will run the scheduler
				server.rpcSchedulerLeader = scaling.NewLeaderElector(
					db.Pool(),
					scaling.RPCSchedulerLockID,
					"rpc-scheduler",
				)
				server.rpcSchedulerLeader.Start(
					func() {
						// Became leader - start the scheduler
						log.Info().Msg("This instance is now the RPC scheduler leader")
						if err := rpcScheduler.Start(); err != nil {
							log.Error().Err(err).Msg("Failed to start RPC scheduler")
						}
					},
					func() {
						// Lost leadership - stop the scheduler
						log.Warn().Msg("Lost RPC scheduler leadership - stopping scheduler")
						rpcScheduler.Stop()
					},
				)
			} else {
				// No leader election - start scheduler directly
				if err := rpcScheduler.Start(); err != nil {
					log.Error().Err(err).Msg("Failed to start RPC scheduler")
				}
			}
		} else {
			log.Info().
				Bool("disable_scheduler", cfg.Scaling.DisableScheduler).
				Bool("worker_only", cfg.Scaling.WorkerOnly).
				Msg("RPC scheduler disabled by scaling configuration")
		}
	}

	// Start webhook trigger service
	if err := webhookTriggerService.Start(context.Background()); err != nil {
		log.Error().Err(err).Msg("Failed to start webhook trigger service")
	}

	// Start retention cleanup service (for central logging)
	if retentionService != nil {
		retentionService.Start()
		log.Info().
			Dur("interval", cfg.Logging.RetentionCheckInterval).
			Msg("Log retention cleanup service started")
	}

	// Start branch cleanup scheduler
	if server.branchScheduler != nil {
		server.branchScheduler.Start()
	}

	// Auto-load AI chatbots if enabled
	if cfg.AI.Enabled && cfg.AI.AutoLoadOnBoot && aiHandler != nil {
		if err := aiHandler.AutoLoadChatbots(context.Background()); err != nil {
			log.Error().Err(err).Msg("Failed to auto-load AI chatbots")
		} else {
			log.Info().Msg("AI chatbots auto-loaded successfully")
		}
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

// setupMCPServer initializes the MCP server with tools and resources
func (s *Server) setupMCPServer(schemaCache *database.SchemaCache, storageService *storage.Service, functionsHandler *functions.Handler, rpcHandler *rpc.Handler) {
	mcpServer := s.mcpHandler.Server()

	// Register MCP Tools
	toolRegistry := mcpServer.ToolRegistry()

	// Database tools
	toolRegistry.Register(mcptools.NewQueryTableTool(s.db, schemaCache))
	toolRegistry.Register(mcptools.NewInsertRecordTool(s.db, schemaCache))
	toolRegistry.Register(mcptools.NewUpdateRecordTool(s.db, schemaCache))
	toolRegistry.Register(mcptools.NewDeleteRecordTool(s.db, schemaCache))

	// Storage tools
	if storageService != nil {
		toolRegistry.Register(mcptools.NewListObjectsTool(storageService))
		toolRegistry.Register(mcptools.NewUploadObjectTool(storageService))
		toolRegistry.Register(mcptools.NewDownloadObjectTool(storageService))
		toolRegistry.Register(mcptools.NewDeleteObjectTool(storageService))
	}

	// Functions invocation tools
	if functionsHandler != nil && s.config.Functions.Enabled {
		toolRegistry.Register(mcptools.NewInvokeFunctionTool(
			s.db,
			functionsHandler.GetRuntime(),
			functionsHandler.GetPublicURL(),
			functionsHandler.GetFunctionsDir(),
		))
	}

	// RPC invocation tools
	if rpcHandler != nil && s.config.RPC.Enabled {
		rpcStorage := rpc.NewStorage(s.db)
		toolRegistry.Register(mcptools.NewInvokeRPCTool(
			rpcHandler.GetExecutor(),
			rpcStorage,
		))
	}

	// Jobs tools
	if s.jobsManager != nil && s.config.Jobs.Enabled {
		jobsStorage := jobs.NewStorage(s.db)
		toolRegistry.Register(mcptools.NewSubmitJobTool(jobsStorage))
		toolRegistry.Register(mcptools.NewGetJobStatusTool(jobsStorage))
	}

	// Vector search tools
	if s.config.AI.Enabled && s.aiHandler != nil {
		// SearchVectorsTool requires RAGService - skip if not available
		log.Debug().Msg("MCP: Vector search tool registration deferred - requires RAG service")
	}

	// Register MCP Resources
	resourceRegistry := mcpServer.ResourceRegistry()

	// Schema resources
	resourceRegistry.Register(mcpresources.NewSchemaResource(schemaCache))
	resourceRegistry.Register(mcpresources.NewTableResource(schemaCache))

	// Functions resources
	if s.config.Functions.Enabled {
		resourceRegistry.Register(mcpresources.NewFunctionsResource(functions.NewStorage(s.db)))
	}

	// RPC resources
	if s.config.RPC.Enabled {
		resourceRegistry.Register(mcpresources.NewRPCResource(rpc.NewStorage(s.db)))
	}

	// Storage resources
	if storageService != nil {
		resourceRegistry.Register(mcpresources.NewBucketsResource(storageService))
	}

	log.Debug().
		Int("tools", len(toolRegistry.ListTools(&mcp.AuthContext{IsServiceRole: true}))).
		Int("resources", len(resourceRegistry.ListResources(&mcp.AuthContext{IsServiceRole: true}))).
		Msg("MCP Server initialized with tools and resources")
}

// setupMiddlewares sets up global middlewares
func (s *Server) setupMiddlewares() {
	// Request ID middleware - must be first for tracing
	log.Debug().Msg("Adding requestid middleware")
	s.app.Use(requestid.New())

	// OpenTelemetry tracing middleware - adds distributed tracing to all requests
	if s.config.Tracing.Enabled && s.tracer != nil && s.tracer.IsEnabled() {
		log.Debug().Msg("Adding OpenTelemetry tracing middleware")
		s.app.Use(middleware.TracingMiddleware(middleware.TracingConfig{
			Enabled:            true,
			ServiceName:        s.config.Tracing.ServiceName,
			SkipPaths:          []string{"/health", "/ready", "/metrics"},
			RecordRequestBody:  false, // Don't record bodies for security
			RecordResponseBody: false,
		}))
	}

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

	// Structured logger middleware - logs HTTP requests through zerolog
	// This allows HTTP logs to be captured by the central logging system
	log.Debug().Msg("Adding structured logger middleware")
	s.app.Use(middleware.StructuredLogger(middleware.StructuredLoggerConfig{
		SkipPaths: []string{"/health", "/ready", "/metrics"},
		// In debug mode, log all requests; in production, skip successful requests to reduce noise
		SkipSuccessfulRequests: !s.config.Debug,
	}))

	// Recover middleware - catch panics
	log.Debug().Msg("Adding recover middleware")
	s.app.Use(recover.New(recover.Config{
		EnableStackTrace: s.config.Debug,
	}))

	// CORS middleware
	// Note: AllowCredentials cannot be used with AllowOrigins="*" per CORS spec
	// If AllowOrigins is "*", we must disable credentials
	corsCredentials := s.config.CORS.AllowCredentials
	corsOrigins := s.config.CORS.AllowedOrigins
	if corsOrigins == "*" && corsCredentials {
		log.Warn().Msg("CORS: AllowCredentials disabled because AllowOrigins is '*' (not allowed per CORS spec)")
		corsCredentials = false
	}
	// Automatically add the public base URL to CORS origins if it's not already included
	// This ensures the dashboard can make API calls when deployed on a public URL
	if corsOrigins != "*" && s.config.PublicBaseURL != "" {
		if !strings.Contains(corsOrigins, s.config.PublicBaseURL) {
			corsOrigins = corsOrigins + "," + s.config.PublicBaseURL
			log.Debug().Str("public_url", s.config.PublicBaseURL).Msg("Added public base URL to CORS origins")
		}
	}
	log.Debug().
		Str("origins", corsOrigins).
		Bool("credentials", corsCredentials).
		Msg("Adding CORS middleware")

	// Build CORS config
	corsConfig := cors.Config{
		AllowMethods:     s.config.CORS.AllowedMethods,
		AllowHeaders:     s.config.CORS.AllowedHeaders,
		ExposeHeaders:    s.config.CORS.ExposedHeaders,
		AllowCredentials: corsCredentials,
		MaxAge:           s.config.CORS.MaxAge,
	}

	// When AllowOrigins is "*", use AllowOriginsFunc to dynamically allow all origins
	// This is required because Fiber's CORS middleware doesn't properly handle "*"
	// with the AllowOrigins string field in newer versions
	if corsOrigins == "*" {
		corsConfig.AllowOriginsFunc = func(origin string) bool {
			return true // Allow all origins
		}
	} else {
		corsConfig.AllowOrigins = corsOrigins
	}

	s.app.Use(cors.New(corsConfig))
	log.Debug().Msg("CORS middleware added")

	// Global rate limiting - 100 requests per minute per IP
	// Uses dynamic limiter that checks settings cache on each request
	// This allows toggling rate limiting via admin UI without server restart
	s.app.Use(middleware.DynamicGlobalAPILimiter(s.authHandler.authService.GetSettingsCache()))

	// Compression middleware
	s.app.Use(compress.New(compress.Config{
		Level: compress.LevelDefault,
	}))
}

// setupRoutes sets up all routes
func (s *Server) setupRoutes() {
	// Root path - simple health response
	s.app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	// Health check endpoint
	s.app.Get("/health", s.handleHealth)

	// API v1 routes - versioned for future compatibility
	v1 := s.app.Group("/api/v1")

	// Setup RLS middleware (before REST API routes)
	rlsConfig := middleware.RLSConfig{
		DB: s.db,
	}

	// REST API routes (auto-generated from database schema)
	// Required authentication (JWT, API key, or service key) - rejects unauthenticated requests
	// Metadata listing (GET /) requires admin role; data operations use RLS filtering
	// Pass jwtManager to support dashboard admin tokens (maps to service_role for full access)
	// BranchContext middleware enables queries against non-main branches via X-Fluxbase-Branch header
	restMiddlewares := []fiber.Handler{
		middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
		middleware.RLSMiddleware(rlsConfig),
	}
	// Add branch context middleware if branching is enabled
	if s.branchRouter != nil {
		restMiddlewares = append(restMiddlewares, middleware.BranchContextSimple(s.branchRouter))
	}
	rest := v1.Group("/tables", restMiddlewares...)
	s.setupRESTRoutes(rest)

	// Auth routes with CSRF protection
	// CSRF middleware protects against cross-site request forgery attacks
	csrfMiddleware := middleware.CSRF(middleware.CSRFConfig{
		TokenLength:    32,
		TokenLookup:    "header:X-CSRF-Token",
		CookieName:     "csrf_token",
		CookiePath:     "/",
		CookieSecure:   s.config.Tracing.Environment == "production",
		CookieHTTPOnly: true,
		CookieSameSite: "Strict",
		Expiration:     24 * time.Hour,
	})
	authRoutes := v1.Group("/auth", csrfMiddleware)
	s.setupAuthRoutes(authRoutes)

	// Public settings routes - optional authentication with RLS support
	// These routes respect app.settings RLS policies based on is_public and is_secret flags
	settings := v1.Group("/settings", OptionalAuthMiddleware(s.authHandler.authService))
	settings.Get("/:key", s.settingsHandler.GetSetting)
	settings.Post("/batch", s.settingsHandler.GetSettings)

	// User secret settings routes - require authentication
	// Users can only access their own secrets (encrypted, server-side decryption only)
	userSecrets := v1.Group("/settings/secret",
		middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
	)
	userSecrets.Post("/", s.userSettingsHandler.CreateSecret)
	userSecrets.Get("/", s.userSettingsHandler.ListSecrets)
	userSecrets.Get("/*", s.userSettingsHandler.GetSecret)
	userSecrets.Put("/*", s.userSettingsHandler.UpdateSecret)
	userSecrets.Delete("/*", s.userSettingsHandler.DeleteSecret)

	// Dashboard auth routes (separate from application auth)
	s.dashboardAuthHandler.RegisterRoutes(s.app)

	// client keys routes - require authentication
	s.clientKeyHandler.RegisterRoutes(s.app, s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)

	// Secrets routes - require authentication
	s.secretsHandler.RegisterRoutes(s.app, s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)

	// Webhook routes - require authentication
	s.webhookHandler.RegisterRoutes(s.app, s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)

	// Monitoring routes - require authentication
	s.monitoringHandler.RegisterRoutes(s.app, s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)

	// Edge functions routes - require authentication by default, but per-function config can override
	// Protected by feature flag middleware
	s.functionsHandler.RegisterRoutes(s.app, s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)

	// Jobs routes - require authentication
	// Protected by feature flag middleware
	// Note: Admin routes are registered in setupAdminRoutes with proper auth middleware
	if s.jobsHandler != nil {
		s.jobsHandler.RegisterRoutes(s.app, s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager)
	}

	// Storage routes - optional authentication (allows unauthenticated access to public buckets)
	// Protected by feature flag middleware
	// BranchContext middleware enables storage operations against non-main branches
	storageMiddlewares := []fiber.Handler{
		middleware.RequireStorageEnabled(s.authHandler.authService.GetSettingsCache()),
		middleware.OptionalAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool()),
	}
	if s.branchRouter != nil {
		storageMiddlewares = append(storageMiddlewares, middleware.BranchContextSimple(s.branchRouter))
	}
	storage := v1.Group("/storage", storageMiddlewares...)
	s.setupStorageRoutes(storage)

	// MCP routes - Model Context Protocol for AI assistants
	// Requires authentication via client key or service key
	if s.config.MCP.Enabled && s.mcpHandler != nil {
		mcpGroup := s.app.Group(s.config.MCP.BasePath,
			middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
		)
		s.mcpHandler.RegisterRoutes(mcpGroup)
		log.Debug().Str("base_path", s.config.MCP.BasePath).Msg("MCP routes registered")
	}

	// Database Branching routes
	// Admin endpoints require service key or dashboard admin
	if s.config.Branching.Enabled && s.branchHandler != nil {
		// Create API group with auth for branch management
		branchAPI := s.app.Group("/api/v1",
			middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
		)
		s.branchHandler.RegisterRoutes(branchAPI)

		// GitHub webhook endpoint (no auth, uses signature verification)
		// Rate limited to prevent abuse
		webhookAPI := s.app.Group("/api/v1", middleware.GitHubWebhookLimiter())
		s.githubWebhook.RegisterRoutes(webhookAPI)

		log.Debug().Msg("Database Branching routes registered")
	}

	// Realtime WebSocket endpoint (not versioned as it's WebSocket)
	// WebSocket validates auth internally, but make it required
	// Protected by feature flag middleware and realtime:connect scope
	s.app.Get("/realtime",
		middleware.RequireRealtimeEnabled(s.authHandler.authService.GetSettingsCache()),
		middleware.OptionalAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
		middleware.RequireScope(auth.ScopeRealtimeConnect),
		s.realtimeHandler.HandleWebSocket,
	)

	// Realtime stats endpoint - require authentication and realtime:connect scope
	// Protected by feature flag middleware
	s.app.Get("/api/v1/realtime/stats",
		middleware.RequireRealtimeEnabled(s.authHandler.authService.GetSettingsCache()),
		middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
		middleware.RequireScope(auth.ScopeRealtimeConnect),
		s.handleRealtimeStats,
	)

	// Realtime broadcast endpoint - require authentication and realtime:broadcast scope
	// Protected by feature flag middleware
	s.app.Post("/api/v1/realtime/broadcast",
		middleware.RequireRealtimeEnabled(s.authHandler.authService.GetSettingsCache()),
		middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
		middleware.RequireScope(auth.ScopeRealtimeBroadcast),
		s.handleRealtimeBroadcast,
	)

	// AI WebSocket endpoint (require AI enabled and authentication)
	if s.aiChatHandler != nil {
		s.app.Get("/ai/ws",
			middleware.RequireAIEnabled(s.authHandler.authService.GetSettingsCache()),
			middleware.OptionalAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			s.aiChatHandler.HandleWebSocket,
		)

		// Public AI chatbot list endpoint
		s.app.Get("/api/v1/ai/chatbots",
			middleware.RequireAIEnabled(s.authHandler.authService.GetSettingsCache()),
			middleware.OptionalAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			s.aiHandler.ListPublicChatbots,
		)

		s.app.Get("/api/v1/ai/chatbots/:id",
			middleware.RequireAIEnabled(s.authHandler.authService.GetSettingsCache()),
			middleware.OptionalAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			s.aiHandler.GetPublicChatbot,
		)

		// User conversation history endpoints (require authentication)
		s.app.Get("/api/v1/ai/conversations",
			middleware.RequireAIEnabled(s.authHandler.authService.GetSettingsCache()),
			middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			s.aiHandler.ListUserConversations,
		)

		s.app.Get("/api/v1/ai/conversations/:id",
			middleware.RequireAIEnabled(s.authHandler.authService.GetSettingsCache()),
			middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			s.aiHandler.GetUserConversation,
		)

		s.app.Delete("/api/v1/ai/conversations/:id",
			middleware.RequireAIEnabled(s.authHandler.authService.GetSettingsCache()),
			middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			s.aiHandler.DeleteUserConversation,
		)

		s.app.Patch("/api/v1/ai/conversations/:id",
			middleware.RequireAIEnabled(s.authHandler.authService.GetSettingsCache()),
			middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			s.aiHandler.UpdateUserConversation,
		)
	}

	// Public RPC endpoints (only if RPC is enabled) with scope enforcement
	if s.rpcHandler != nil {
		// List public procedures - requires read:rpc scope
		s.app.Get("/api/v1/rpc/procedures",
			middleware.RequireRPCEnabled(s.authHandler.authService.GetSettingsCache()),
			middleware.OptionalAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			middleware.RequireScope(auth.ScopeRPCRead),
			s.rpcHandler.ListPublicProcedures,
		)

		// Invoke RPC procedure - requires execute:rpc scope
		s.app.Post("/api/v1/rpc/:namespace/:name",
			middleware.RequireRPCEnabled(s.authHandler.authService.GetSettingsCache()),
			middleware.OptionalAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			middleware.RequireScope(auth.ScopeRPCExecute),
			s.rpcHandler.Invoke,
		)

		// Get execution status (public - users can see their own) - requires read:rpc scope
		s.app.Get("/api/v1/rpc/executions/:id",
			middleware.RequireRPCEnabled(s.authHandler.authService.GetSettingsCache()),
			middleware.OptionalAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			middleware.RequireScope(auth.ScopeRPCRead),
			s.rpcHandler.GetPublicExecution,
		)

		// Get execution logs (public - users can see their own) - requires read:rpc scope
		s.app.Get("/api/v1/rpc/executions/:id/logs",
			middleware.RequireRPCEnabled(s.authHandler.authService.GetSettingsCache()),
			middleware.OptionalAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			middleware.RequireScope(auth.ScopeRPCRead),
			s.rpcHandler.GetPublicExecutionLogs,
		)
	}

	// Vector search endpoints (only if vector handler is initialized)
	if s.vectorHandler != nil {
		// Capabilities endpoint (public - no auth required)
		// Returns information about pgvector installation status and embedding configuration
		s.app.Get("/api/v1/capabilities/vector", s.vectorHandler.HandleGetCapabilities)

		// Embedding endpoint (requires authentication)
		s.app.Post("/api/v1/vector/embed",
			middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			s.vectorHandler.HandleEmbed,
		)

		// Vector search endpoint (requires authentication)
		s.app.Post("/api/v1/vector/search",
			middleware.RequireAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			s.vectorHandler.HandleSearch,
		)

		log.Info().Msg("Vector search routes registered")
	}

	// GraphQL endpoint (if enabled)
	if s.graphqlHandler != nil {
		// GraphQL uses its own auth handling to set up RLS context
		s.app.Post("/api/v1/graphql",
			middleware.OptionalAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			s.graphqlHandler.HandleGraphQL,
		)
		// Introspection endpoint (GET)
		s.app.Get("/api/v1/graphql",
			middleware.OptionalAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
			s.graphqlHandler.HandleIntrospection,
		)
		log.Info().Msg("GraphQL endpoint registered at /api/v1/graphql")
	}

	// Admin API routes - always available (protected by their own auth middleware)
	admin := v1.Group("/admin")
	s.setupAdminRoutes(admin)

	// Public invitation routes (no auth required)
	invitations := v1.Group("/invitations")
	s.setupPublicInvitationRoutes(invitations)

	log.Info().Msg("Admin API routes registered")

	// Admin UI and dashboard auth routes - only enabled when admin.enabled=true
	// Requires setup_token for the dashboard authentication system
	if s.config.Admin.Enabled {
		if s.config.Security.SetupToken == "" {
			log.Error().Msg("Admin UI is enabled but FLUXBASE_SECURITY_SETUP_TOKEN is not set. Admin UI will not be registered for security reasons.")
		} else {
			// Dashboard auth routes (setup, login, etc.)
			s.setupDashboardAuthRoutes(admin)

			// Admin UI (embedded React app)
			// Pass the public base URL so it can be injected into the frontend at runtime
			adminUI := adminui.New(s.config.GetPublicBaseURL())
			adminUI.RegisterRoutes(s.app)
			log.Info().Msg("Admin UI enabled")
		}
	} else {
		log.Info().Msg("Admin UI is disabled. Set FLUXBASE_ADMIN_ENABLED=true to enable the admin dashboard UI.")
	}

	// Sync endpoints - always available (independent of admin dashboard)
	// Each is protected by IP allowlist + auth + role requirements + feature flag middleware
	syncAuth := UnifiedAuthMiddleware(s.authHandler.authService, s.dashboardAuthHandler.jwtManager, s.db.Pool())

	// Functions sync
	funcSync := v1.Group("/admin/functions")
	funcSync.Post("/sync",
		middleware.RequireSyncIPAllowlist(s.config.Functions.SyncAllowedIPRanges, "functions"),
		syncAuth,
		RequireRole("admin", "dashboard_admin", "service_role"),
		s.functionsHandler.SyncFunctions,
	)

	// Jobs sync (requires jobsHandler)
	if s.jobsHandler != nil {
		jobsSync := v1.Group("/admin/jobs")
		jobsSync.Post("/sync",
			middleware.RequireSyncIPAllowlist(s.config.Jobs.SyncAllowedIPRanges, "jobs"),
			syncAuth,
			RequireRole("admin", "dashboard_admin", "service_role"),
			s.jobsHandler.SyncJobs,
		)
	}

	// AI chatbots sync (requires aiHandler)
	if s.aiHandler != nil {
		requireAI := middleware.RequireAIEnabled(s.authHandler.authService.GetSettingsCache())
		aiSync := v1.Group("/admin/ai/chatbots")
		aiSync.Post("/sync",
			requireAI,
			middleware.RequireSyncIPAllowlist(s.config.AI.SyncAllowedIPRanges, "ai"),
			syncAuth,
			RequireRole("admin", "dashboard_admin", "service_role"),
			s.aiHandler.SyncChatbots,
		)
	}

	// RPC sync (requires rpcHandler)
	if s.rpcHandler != nil {
		requireRPC := middleware.RequireRPCEnabled(s.authHandler.authService.GetSettingsCache())
		rpcSync := v1.Group("/admin/rpc")
		rpcSync.Post("/sync",
			requireRPC,
			middleware.RequireSyncIPAllowlist(s.config.RPC.SyncAllowedIPRanges, "rpc"),
			syncAuth,
			RequireRole("admin", "dashboard_admin", "service_role"),
			s.rpcHandler.SyncProcedures,
		)
	}

	// OpenAPI specification
	// Uses optional auth middleware to detect admin users and provide full spec with database schema
	// Non-admin users get minimal spec with only auth endpoints
	openAPIHandler := NewOpenAPIHandler(s.db)
	s.app.Get("/openapi.json",
		middleware.OptionalAuthOrServiceKey(s.authHandler.authService, s.clientKeyService, s.db.Pool(), s.dashboardAuthHandler.jwtManager),
		openAPIHandler.GetOpenAPISpec,
	)

	// 404 handler
	s.app.Use(func(c *fiber.Ctx) error {
		return c.Status(404).JSON(fiber.Map{
			"error": "Not Found",
			"path":  c.Path(),
		})
	})
}

// setupRESTRoutes sets up dynamic REST routes using wildcard patterns
// This allows new tables created via migrations to be immediately accessible
// without requiring a server restart.
func (s *Server) setupRESTRoutes(router fiber.Router) {
	log.Info().Msg("Setting up dynamic REST API routes with wildcard patterns")

	// Metadata endpoint - list all tables (admin only)
	// Regular users should not be able to discover all tables; they access tables by name with RLS
	router.Get("/", RequireRole("admin", "dashboard_admin", "service_role"), s.rest.HandleGetTables)

	// Dynamic routes using wildcard patterns
	// Order matters: more specific routes first

	// POST query endpoint for complex filters (avoids URL length limits)
	// Routes: /tables/:schema/:table/query and /tables/:table/query
	router.Post("/:schema/:table/query",
		middleware.RequireScope(auth.ScopeTablesRead),
		s.rest.HandleDynamicQuery)
	router.Post("/:schema/query",
		middleware.RequireScope(auth.ScopeTablesRead),
		s.rest.HandleDynamicQuery)

	// Routes with ID parameter: /tables/:schema/:table/:id and /tables/:table/:id
	// These handle GET (fetch one), PUT (replace), PATCH (update), DELETE (remove)
	router.Get("/:schema/:table/:id",
		middleware.RequireScope(auth.ScopeTablesRead),
		s.rest.HandleDynamicTableById)
	router.Put("/:schema/:table/:id",
		middleware.RequireScope(auth.ScopeTablesWrite),
		s.rest.HandleDynamicTableById)
	router.Patch("/:schema/:table/:id",
		middleware.RequireScope(auth.ScopeTablesWrite),
		s.rest.HandleDynamicTableById)
	router.Delete("/:schema/:table/:id",
		middleware.RequireScope(auth.ScopeTablesWrite),
		s.rest.HandleDynamicTableById)

	// Collection routes: /tables/:schema/:table and /tables/:table
	// These handle GET (list), POST (create), PATCH (batch update), DELETE (batch delete)
	router.Get("/:schema/:table",
		middleware.RequireScope(auth.ScopeTablesRead),
		s.rest.HandleDynamicTable)
	router.Post("/:schema/:table",
		middleware.RequireScope(auth.ScopeTablesWrite),
		s.rest.HandleDynamicTable)
	router.Patch("/:schema/:table",
		middleware.RequireScope(auth.ScopeTablesWrite),
		s.rest.HandleDynamicTable)
	router.Delete("/:schema/:table",
		middleware.RequireScope(auth.ScopeTablesWrite),
		s.rest.HandleDynamicTable)

	// Single-segment routes for public schema: /tables/:table
	// Note: Fiber parses /tables/posts as schema="posts", table=""
	// The handler detects this and treats it as public.posts
	router.Get("/:schema",
		middleware.RequireScope(auth.ScopeTablesRead),
		s.rest.HandleDynamicTable)
	router.Post("/:schema",
		middleware.RequireScope(auth.ScopeTablesWrite),
		s.rest.HandleDynamicTable)
	router.Patch("/:schema",
		middleware.RequireScope(auth.ScopeTablesWrite),
		s.rest.HandleDynamicTable)
	router.Delete("/:schema",
		middleware.RequireScope(auth.ScopeTablesWrite),
		s.rest.HandleDynamicTable)

	log.Info().Msg("Dynamic REST API routes configured")
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
		"2fa":            middleware.Auth2FALimiter(),       // Strict rate limit for 2FA verification
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
	// Signed URL download (PUBLIC - no auth required, token provides authorization)
	router.Get("/object", s.storageHandler.DownloadSignedObject)

	// Transform config (PUBLIC - no auth required, just returns config info)
	router.Get("/config/transforms", s.storageHandler.GetTransformConfig)

	// Bucket management with scope enforcement
	router.Get("/buckets", middleware.RequireScope(auth.ScopeStorageRead), s.storageHandler.ListBuckets)
	router.Post("/buckets/:bucket", middleware.RequireScope(auth.ScopeStorageWrite), s.storageHandler.CreateBucket)
	router.Put("/buckets/:bucket", middleware.RequireScope(auth.ScopeStorageWrite), s.storageHandler.UpdateBucketSettings)
	router.Delete("/buckets/:bucket", middleware.RequireScope(auth.ScopeStorageWrite), s.storageHandler.DeleteBucket)

	// List files in bucket (must come before /:bucket/*)
	router.Get("/:bucket", middleware.RequireScope(auth.ScopeStorageRead), s.storageHandler.ListFiles)

	// Multipart upload (must come before /:bucket/*)
	router.Post("/:bucket/multipart", middleware.RequireScope(auth.ScopeStorageWrite), s.storageHandler.MultipartUpload)

	// File sharing (must come before /:bucket/* to avoid matching generic routes)
	router.Post("/:bucket/*/share", middleware.RequireScope(auth.ScopeStorageWrite), s.storageHandler.ShareObject)            // Share file with user
	router.Delete("/:bucket/*/share/:user_id", middleware.RequireScope(auth.ScopeStorageWrite), s.storageHandler.RevokeShare) // Revoke share
	router.Get("/:bucket/*/shares", middleware.RequireScope(auth.ScopeStorageRead), s.storageHandler.ListShares)              // List shares

	// Signed URLs (for S3-compatible storage, must come before /:bucket/*)
	router.Post("/:bucket/sign/*", middleware.RequireScope(auth.ScopeStorageWrite), s.storageHandler.GenerateSignedURL)

	// Streaming upload (must come before /:bucket/*)
	router.Post("/:bucket/stream/*", middleware.RequireScope(auth.ScopeStorageWrite), s.storageHandler.StreamUpload)

	// Chunked upload routes (for resumable large file uploads, must come before /:bucket/*)
	router.Post("/:bucket/chunked/init", middleware.RequireScope(auth.ScopeStorageWrite), s.storageHandler.InitChunkedUpload)
	router.Put("/:bucket/chunked/:uploadId/:chunkIndex", middleware.RequireScope(auth.ScopeStorageWrite), s.storageHandler.UploadChunk)
	router.Post("/:bucket/chunked/:uploadId/complete", middleware.RequireScope(auth.ScopeStorageWrite), s.storageHandler.CompleteChunkedUpload)
	router.Get("/:bucket/chunked/:uploadId/status", middleware.RequireScope(auth.ScopeStorageRead), s.storageHandler.GetChunkedUploadStatus)
	router.Delete("/:bucket/chunked/:uploadId", middleware.RequireScope(auth.ScopeStorageWrite), s.storageHandler.AbortChunkedUpload)

	// File operations (generic wildcard routes - must come LAST)
	router.Post("/:bucket/*", middleware.RequireScope(auth.ScopeStorageWrite), s.storageHandler.UploadFile)   // Upload file
	router.Get("/:bucket/*", middleware.RequireScope(auth.ScopeStorageRead), s.storageHandler.DownloadFile)   // Download file
	router.Head("/:bucket/*", middleware.RequireScope(auth.ScopeStorageRead), s.storageHandler.DownloadFile)  // HEAD delegates to GetFileInfo for Content-Length
	router.Delete("/:bucket/*", middleware.RequireScope(auth.ScopeStorageWrite), s.storageHandler.DeleteFile) // Delete file
}

// setupDashboardAuthRoutes sets up dashboard authentication routes
// These are only available when admin UI is enabled
func (s *Server) setupDashboardAuthRoutes(router fiber.Router) {
	// Public dashboard auth routes (no authentication required)
	router.Get("/setup/status", s.adminAuthHandler.GetSetupStatus)
	router.Post("/setup", middleware.AdminSetupLimiter(), s.adminAuthHandler.InitialSetup)
	router.Post("/login", middleware.AdminLoginLimiter(), s.adminAuthHandler.AdminLogin)
	router.Post("/refresh", s.adminAuthHandler.AdminRefreshToken)

	// Protected dashboard auth routes
	unifiedAuth := UnifiedAuthMiddleware(s.authHandler.authService, s.dashboardAuthHandler.jwtManager, s.db.Pool())
	router.Post("/logout", unifiedAuth, s.adminAuthHandler.AdminLogout)
	router.Get("/me", unifiedAuth, s.adminAuthHandler.GetCurrentAdmin)
}

// setupAdminRoutes sets up admin API routes
func (s *Server) setupAdminRoutes(router fiber.Router) {
	// Protected admin routes (require authentication from either auth.users or dashboard.users)
	// UnifiedAuthMiddleware accepts tokens from both authentication systems
	// The db pool is passed to allow real-time role checking from auth.users
	unifiedAuth := UnifiedAuthMiddleware(s.authHandler.authService, s.dashboardAuthHandler.jwtManager, s.db.Pool())

	// Admin panel routes (require admin or dashboard_admin role)
	router.Get("/tables", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.handleGetTables)
	router.Get("/tables/:schema/:table", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.handleGetTableSchema)
	router.Get("/schemas", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.handleGetSchemas)
	router.Post("/query", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.handleExecuteQuery)

	// DDL routes (schema and table management) - require admin or dashboard_admin role
	router.Get("/ddl/schemas", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.ddlHandler.ListSchemas)
	router.Post("/ddl/schemas", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.ddlHandler.CreateSchema)
	router.Get("/ddl/tables", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.ddlHandler.ListTables)
	router.Post("/ddl/tables", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.ddlHandler.CreateTable)
	router.Delete("/ddl/tables/:schema/:table", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.ddlHandler.DeleteTable)

	// Legacy DDL routes (without /ddl/ prefix) - keep for backwards compatibility
	router.Post("/schemas", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.ddlHandler.CreateSchema)
	router.Post("/tables", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.ddlHandler.CreateTable)
	router.Delete("/tables/:schema/:table", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.ddlHandler.DeleteTable)
	router.Patch("/tables/:schema/:table", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.ddlHandler.RenameTable)
	router.Post("/tables/:schema/:table/columns", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.ddlHandler.AddColumn)
	router.Delete("/tables/:schema/:table/columns/:column", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.ddlHandler.DropColumn)

	// OAuth provider management routes (require admin or dashboard_admin role)
	router.Get("/oauth/providers", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.oauthProviderHandler.ListOAuthProviders)
	router.Get("/oauth/providers/:id", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.oauthProviderHandler.GetOAuthProvider)
	router.Post("/oauth/providers", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.oauthProviderHandler.CreateOAuthProvider)
	router.Put("/oauth/providers/:id", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.oauthProviderHandler.UpdateOAuthProvider)
	router.Delete("/oauth/providers/:id", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.oauthProviderHandler.DeleteOAuthProvider)

	// SAML provider management routes (require admin or dashboard_admin role)
	router.Get("/saml/providers", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.samlProviderHandler.ListSAMLProviders)
	router.Get("/saml/providers/:id", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.samlProviderHandler.GetSAMLProvider)
	router.Post("/saml/providers", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.samlProviderHandler.CreateSAMLProvider)
	router.Put("/saml/providers/:id", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.samlProviderHandler.UpdateSAMLProvider)
	router.Delete("/saml/providers/:id", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.samlProviderHandler.DeleteSAMLProvider)
	router.Post("/saml/validate-metadata", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.samlProviderHandler.ValidateMetadata)
	router.Post("/saml/upload-metadata", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.samlProviderHandler.UploadMetadata)

	// Auth settings routes (require admin or dashboard_admin role)
	router.Get("/auth/settings", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.oauthProviderHandler.GetAuthSettings)
	router.Put("/auth/settings", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.oauthProviderHandler.UpdateAuthSettings)

	// Session management routes (require admin or dashboard_admin role)
	router.Get("/auth/sessions", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.adminSessionHandler.ListSessions)
	router.Delete("/auth/sessions/:id", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.adminSessionHandler.RevokeSession)
	router.Delete("/auth/sessions/user/:user_id", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.adminSessionHandler.RevokeUserSessions)

	// System settings routes (require admin or dashboard_admin role)
	router.Get("/system/settings", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.systemSettingsHandler.ListSettings)
	router.Get("/system/settings/*", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.systemSettingsHandler.GetSetting)
	router.Put("/system/settings/*", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.systemSettingsHandler.UpdateSetting)
	router.Delete("/system/settings/*", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.systemSettingsHandler.DeleteSetting)

	// Custom settings routes (require admin, dashboard_admin, or service_role)
	router.Post("/settings/custom", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.customSettingsHandler.CreateSetting)
	router.Get("/settings/custom", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.customSettingsHandler.ListSettings)

	// System secret settings routes (must come before wildcard routes)
	router.Post("/settings/custom/secret", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.customSettingsHandler.CreateSecretSetting)
	router.Get("/settings/custom/secrets", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.customSettingsHandler.ListSecretSettings)
	router.Get("/settings/custom/secret/*", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.customSettingsHandler.GetSecretSetting)
	router.Put("/settings/custom/secret/*", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.customSettingsHandler.UpdateSecretSetting)
	router.Delete("/settings/custom/secret/*", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.customSettingsHandler.DeleteSecretSetting)

	// User secret decryption route (service_role only - used by edge functions to decrypt user secrets)
	router.Get("/settings/user/:user_id/secret/:key/decrypt", unifiedAuth, RequireRole("service_role"), s.userSettingsHandler.GetUserSecretValue)

	// Regular custom settings wildcard routes
	router.Get("/settings/custom/*", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.customSettingsHandler.GetSetting)
	router.Put("/settings/custom/*", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.customSettingsHandler.UpdateSetting)
	router.Delete("/settings/custom/*", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.customSettingsHandler.DeleteSetting)

	// App settings routes (require admin or dashboard_admin role)
	router.Get("/app/settings", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.appSettingsHandler.GetAppSettings)
	router.Put("/app/settings", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.appSettingsHandler.UpdateAppSettings)

	// Email settings routes (require admin or dashboard_admin role)
	router.Get("/email/settings", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.emailSettingsHandler.GetSettings)
	router.Put("/email/settings", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.emailSettingsHandler.UpdateSettings)
	router.Post("/email/settings/test", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.emailSettingsHandler.TestSettings)

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

	// Service key management routes (require admin, dashboard_admin, or service_role)
	router.Get("/service-keys", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.serviceKeyHandler.ListServiceKeys)
	router.Get("/service-keys/:id", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.serviceKeyHandler.GetServiceKey)
	router.Post("/service-keys", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.serviceKeyHandler.CreateServiceKey)
	router.Patch("/service-keys/:id", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.serviceKeyHandler.UpdateServiceKey)
	router.Delete("/service-keys/:id", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.serviceKeyHandler.DeleteServiceKey)
	router.Post("/service-keys/:id/disable", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.serviceKeyHandler.DisableServiceKey)
	router.Post("/service-keys/:id/enable", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.serviceKeyHandler.EnableServiceKey)

	// SQL Editor route (require admin or dashboard_admin role)
	router.Post("/sql/execute", unifiedAuth, RequireRole("admin", "dashboard_admin"), s.sqlHandler.ExecuteSQL)

	// Functions management routes (require admin, dashboard_admin, or service_role)
	router.Post("/functions/reload", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.functionsHandler.ReloadFunctions)
	router.Get("/functions/namespaces", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.functionsHandler.ListNamespaces)
	// Note: Functions sync is registered outside setupAdminRoutes so it works when admin dashboard is disabled
	// Functions executions - admin endpoint to list all executions
	router.Get("/functions/executions", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.functionsHandler.ListAllExecutions)
	// Functions execution logs - admin endpoint to get logs for a specific execution
	router.Get("/functions/executions/:executionId/logs", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.functionsHandler.GetExecutionLogs)

	// Jobs management routes (require admin, dashboard_admin, or service_role)
	// Only register if jobs are enabled
	// Note: Jobs sync is registered outside setupAdminRoutes so it works when admin dashboard is disabled
	if s.jobsHandler != nil {
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

	// AI management routes (require admin, dashboard_admin, or service_role)
	// Only register if AI is enabled
	// Note: AI chatbots sync is registered outside setupAdminRoutes so it works when admin dashboard is disabled
	if s.aiHandler != nil {
		// Feature flag check for all AI routes
		requireAI := middleware.RequireAIEnabled(s.authHandler.authService.GetSettingsCache())

		// Chatbot management
		router.Get("/ai/chatbots", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.ListChatbots)
		router.Get("/ai/chatbots/:id", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.GetChatbot)
		router.Put("/ai/chatbots/:id/toggle", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.ToggleChatbot)
		router.Put("/ai/chatbots/:id", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.UpdateChatbot)
		router.Delete("/ai/chatbots/:id", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.DeleteChatbot)

		// Metrics
		router.Get("/ai/metrics", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.GetAIMetrics)

		// Conversations & Audit
		router.Get("/ai/conversations", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.GetConversations)
		router.Get("/ai/conversations/:id/messages", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.GetConversationMessages)
		router.Get("/ai/audit", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.GetAuditLog)

		// Provider management
		router.Get("/ai/providers", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.ListProviders)
		router.Get("/ai/providers/:id", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.GetProvider)
		router.Post("/ai/providers", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.CreateProvider)
		router.Put("/ai/providers/:id/default", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.SetDefaultProvider)
		router.Delete("/ai/providers/:id", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.DeleteProvider)
		router.Put("/ai/providers/:id", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.aiHandler.UpdateProvider)

		// Knowledge base management (RAG)
		if s.knowledgeBaseHandler != nil {
			router.Get("/ai/knowledge-bases", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.ListKnowledgeBases)
			router.Get("/ai/knowledge-bases/capabilities", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.GetCapabilities)
			router.Get("/ai/knowledge-bases/:id", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.GetKnowledgeBase)
			router.Post("/ai/knowledge-bases", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.CreateKnowledgeBase)
			router.Put("/ai/knowledge-bases/:id", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.UpdateKnowledgeBase)
			router.Delete("/ai/knowledge-bases/:id", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.DeleteKnowledgeBase)

			// Documents within a knowledge base
			router.Get("/ai/knowledge-bases/:id/documents", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.ListDocuments)
			router.Post("/ai/knowledge-bases/:id/documents", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.AddDocument)
			router.Get("/ai/knowledge-bases/:id/documents/:doc_id", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.GetDocument)
			router.Delete("/ai/knowledge-bases/:id/documents/:doc_id", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.DeleteDocument)
			router.Patch("/ai/knowledge-bases/:id/documents/:doc_id", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.UpdateDocument)
			router.Post("/ai/knowledge-bases/:id/documents/upload", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.UploadDocument)

			// Search/test endpoint
			router.Post("/ai/knowledge-bases/:id/search", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.SearchKnowledgeBase)
			router.Post("/ai/knowledge-bases/:id/debug-search", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.DebugSearch)

			// Chatbot knowledge base linking
			router.Get("/ai/chatbots/:id/knowledge-bases", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.ListChatbotKnowledgeBases)
			router.Post("/ai/chatbots/:id/knowledge-bases", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.LinkKnowledgeBase)
			router.Put("/ai/chatbots/:id/knowledge-bases/:kb_id", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.UpdateChatbotKnowledgeBase)
			router.Delete("/ai/chatbots/:id/knowledge-bases/:kb_id", requireAI, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.knowledgeBaseHandler.UnlinkKnowledgeBase)
		}
	}

	// RPC management routes (require admin, dashboard_admin, or service_role)
	// Only register if RPC is enabled
	// Note: RPC sync is registered outside setupAdminRoutes so it works when admin dashboard is disabled
	if s.rpcHandler != nil {
		requireRPC := middleware.RequireRPCEnabled(s.authHandler.authService.GetSettingsCache())

		// Procedure management
		router.Get("/rpc/namespaces", requireRPC, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.rpcHandler.ListNamespaces)
		router.Get("/rpc/procedures", requireRPC, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.rpcHandler.ListProcedures)
		router.Get("/rpc/procedures/:namespace/:name", requireRPC, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.rpcHandler.GetProcedure)
		router.Put("/rpc/procedures/:namespace/:name", requireRPC, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.rpcHandler.UpdateProcedure)
		router.Delete("/rpc/procedures/:namespace/:name", requireRPC, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.rpcHandler.DeleteProcedure)

		// Execution management
		router.Get("/rpc/executions", requireRPC, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.rpcHandler.ListExecutions)
		router.Get("/rpc/executions/:id", requireRPC, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.rpcHandler.GetExecution)
		router.Get("/rpc/executions/:id/logs", requireRPC, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.rpcHandler.GetExecutionLogs)
		router.Post("/rpc/executions/:id/cancel", requireRPC, unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.rpcHandler.CancelExecution)
	}

	// Extensions management routes
	router.Get("/extensions", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.extensionsHandler.ListExtensions)
	router.Get("/extensions/:name/status", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.extensionsHandler.GetExtensionStatus)
	router.Post("/extensions/:name/enable", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.extensionsHandler.EnableExtension)
	router.Post("/extensions/:name/disable", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.extensionsHandler.DisableExtension)
	router.Post("/extensions/sync", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.extensionsHandler.SyncExtensions)

	// Migrations routes (require service key authentication with enhanced security)
	// Only registered if migrations API is enabled in config
	if s.config.Migrations.Enabled {
		// Build secure middleware stack for migrations API
		// Layer 1: Feature flag check
		// Layer 2: IP allowlist (only allow app container)
		// Layer 3: Service key authentication (no JWT/client keys)
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

	// Central logging routes (require admin, dashboard_admin, or service_role)
	if s.loggingHandler != nil {
		router.Get("/logs", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.loggingHandler.QueryLogs)
		router.Get("/logs/stats", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.loggingHandler.GetLogStats)
		router.Get("/logs/executions/:execution_id", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.loggingHandler.GetExecutionLogs)
		router.Post("/logs/flush", unifiedAuth, RequireRole("admin", "dashboard_admin", "service_role"), s.loggingHandler.FlushLogs)
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

	// Collect tables, views, and materialized views from requested schema(s)
	var allItems []database.TableInfo
	for _, schema := range schemasToQuery {
		// Get tables
		tables, err := s.db.Inspector().GetAllTables(ctx, schema)
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get tables from schema")
		} else {
			allItems = append(allItems, tables...)
		}

		// Get views
		views, err := s.db.Inspector().GetAllViews(ctx, schema)
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get views from schema")
		} else {
			allItems = append(allItems, views...)
		}

		// Get materialized views
		matviews, err := s.db.Inspector().GetAllMaterializedViews(ctx, schema)
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get materialized views from schema")
		} else {
			allItems = append(allItems, matviews...)
		}
	}

	return c.JSON(allItems)
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

// handleRefreshSchema refreshes the REST API schema cache without requiring a server restart
func (s *Server) handleRefreshSchema(c *fiber.Ctx) error {
	log.Info().Msg("Schema refresh requested")

	// Get the schema cache from the REST handler
	schemaCache := s.rest.SchemaCache()
	if schemaCache == nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Schema cache not initialized",
		})
	}

	// Force refresh the schema cache
	if err := schemaCache.Refresh(c.Context()); err != nil {
		log.Error().Err(err).Msg("Failed to refresh schema cache")
		return c.Status(500).JSON(fiber.Map{
			"error":   "Failed to refresh schema cache",
			"details": err.Error(),
		})
	}

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
	return s.app.Listen(s.config.Server.Address)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Stop leader electors first (releases advisory locks)
	if s.functionsSchedulerLeader != nil {
		log.Info().Msg("Stopping functions scheduler leader election")
		s.functionsSchedulerLeader.Stop()
	}
	if s.jobsSchedulerLeader != nil {
		log.Info().Msg("Stopping jobs scheduler leader election")
		s.jobsSchedulerLeader.Stop()
	}
	if s.rpcSchedulerLeader != nil {
		log.Info().Msg("Stopping RPC scheduler leader election")
		s.rpcSchedulerLeader.Stop()
	}

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

	// Stop jobs scheduler and manager
	if s.jobsScheduler != nil {
		s.jobsScheduler.Stop()
	}
	if s.jobsManager != nil {
		s.jobsManager.Stop()
	}

	// Stop RPC scheduler
	if s.rpcScheduler != nil {
		s.rpcScheduler.Stop()
	}

	// Stop webhook trigger service
	if s.webhookTriggerService != nil {
		s.webhookTriggerService.Stop()
	}

	// Close AI conversation manager
	if s.aiConversations != nil {
		s.aiConversations.Close()
	}

	// Stop branch cleanup scheduler
	if s.branchScheduler != nil {
		s.branchScheduler.Stop()
	}

	// Close database branching components
	if s.branchRouter != nil {
		log.Info().Msg("Closing branch connection pools")
		s.branchRouter.CloseAllPools()
	}
	if s.branchManager != nil {
		log.Info().Msg("Closing branch manager")
		s.branchManager.Close()
	}

	// Shutdown OpenTelemetry tracer (flush remaining spans)
	if s.tracer != nil {
		if err := s.tracer.Shutdown(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to shutdown OpenTelemetry tracer")
		}
	}

	// Stop retention cleanup service
	if s.retentionService != nil {
		log.Info().Msg("Stopping log retention cleanup service")
		s.retentionService.Stop()
	}

	// Close central logging service (flush remaining log entries)
	if s.loggingService != nil {
		log.Info().Msg("Closing central logging service")
		if err := s.loggingService.Close(); err != nil {
			log.Warn().Err(err).Msg("Failed to close logging service")
		}
	}

	// Close schema cache (stops invalidation listener)
	if s.schemaCache != nil {
		s.schemaCache.Close()
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

// GetLoggingService returns the central logging service
func (s *Server) GetLoggingService() *logging.Service {
	return s.loggingService
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

// LoadAIChatbotsFromFilesystem loads AI chatbots from the filesystem
// This is called at boot time if auto_load_on_boot is enabled
func (s *Server) LoadAIChatbotsFromFilesystem(ctx context.Context) error {
	if s.aiHandler == nil {
		return fmt.Errorf("AI handler not initialized")
	}
	return s.aiHandler.AutoLoadChatbots(ctx)
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
// Admin-only endpoint - non-admin users receive 403 Forbidden
func (s *Server) handleRealtimeStats(c *fiber.Ctx) error {
	// Check if user has admin role
	role, _ := c.Locals("user_role").(string)
	if role != "admin" && role != "dashboard_admin" && role != "service_role" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required to view realtime stats",
		})
	}

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
