package api

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/ai"
	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/logging"
	"github.com/nimbleflux/fluxbase/internal/observability"
	"github.com/nimbleflux/fluxbase/internal/settings"
	"github.com/nimbleflux/fluxbase/internal/storage"
)

type AIModule struct {
	Handlers *AIHandlers
	Quota    *QuotaHandlers
}

func (m *AIModule) Name() string { return "ai" }

func (m *AIModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB
	storageSvc := GetService[*storage.Service](registry)
	loggingSvc := GetService[*logging.Service](registry)
	secretsSvc := GetService[*settings.SecretsService](registry)
	userMgmtSvc := GetService[*auth.UserManagementService](registry)

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
	var knowledgeBaseHandler *ai.KnowledgeBaseHandler
	var kbStorage *ai.KnowledgeBaseStorage
	var knowledgeGraph *ai.KnowledgeGraph
	var docProcessor *ai.DocumentProcessor
	var tableExportSyncService *ai.TableExportSyncService
	var ocrService *ai.OCRService
	var quotaHandler *QuotaHandler
	var internalAIHandler *InternalAIHandler

	if cfg.AI.Enabled {
		aiMetrics = observability.NewMetrics()

		aiLoader := ai.NewLoader(cfg.AI.ChatbotsDir)
		aiConversations = ai.NewConversationManager(db, cfg.AI.ConversationCacheTTL, cfg.AI.MaxConversationTurns)
		aiConversations.SetMetrics(aiMetrics)
		aiHandler = ai.NewHandler(aiStorage, aiLoader, &cfg.AI, vectorManager)

		var embeddingService *ai.EmbeddingService
		if vectorHandler != nil {
			embeddingService = vectorHandler.GetEmbeddingService()
		}

		aiChatHandler = ai.NewChatHandler(db, aiStorage, aiConversations, aiMetrics, &cfg.AI, embeddingService, loggingSvc)

		// Wire chat handler back into the admin handler so provider mutations
		// invalidate the chat path's provider cache.
		aiHandler.SetChatHandler(aiChatHandler)

		if secretsSvc != nil {
			settingsResolver := ai.NewSettingsResolver(secretsSvc, 5*time.Minute)
			aiChatHandler.SetSettingsResolver(settingsResolver)
		}

		log.Info().
			Str("chatbots_dir", cfg.AI.ChatbotsDir).
			Bool("auto_load", cfg.AI.AutoLoadOnBoot).
			Str("provider_type", cfg.AI.ProviderType).
			Str("provider_name", cfg.AI.ProviderName).
			Str("provider_model", cfg.AI.ProviderModel).
			Bool("rag_enabled", embeddingService != nil).
			Msg("AI components initialized")

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
		knowledgeGraph = ai.NewKnowledgeGraph(kbStorage)
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
		if storageSvc != nil {
			knowledgeBaseHandler.SetStorageService(storageSvc)
		}

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
		quotaHandler = NewQuotaHandler(quotaService, userMgmtSvc)
		log.Info().Msg("Quota service and handler initialized")

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

	m.Handlers = &AIHandlers{
		Handler:         aiHandler,
		Chat:            aiChatHandler,
		Conversations:   aiConversations,
		Metrics:         aiMetrics,
		KnowledgeBase:   knowledgeBaseHandler,
		KBStorage:       kbStorage,
		KnowledgeGraph:  knowledgeGraph,
		DocProcessor:    docProcessor,
		TableExportSync: tableExportSyncService,
		VectorManager:   vectorManager,
		VectorHandler:   vectorHandler,
		Internal:        internalAIHandler,
	}
	m.Quota = &QuotaHandlers{Handler: quotaHandler}

	if cfg.AI.Enabled && cfg.AI.AutoLoadOnBoot && aiHandler != nil {
		if err := aiHandler.AutoLoadChatbots(ctx); err != nil {
			log.Error().Err(err).Msg("Failed to auto-load AI chatbots")
		} else {
			log.Info().Msg("AI chatbots auto-loaded successfully")
		}
	}

	if vectorHandler != nil {
		registry.Register(vectorHandler)
	}
	if aiChatHandler != nil {
		registry.Register(aiChatHandler)
	}
	if kbStorage != nil {
		registry.Register(kbStorage)
	}
	return nil
}

func (m *AIModule) Shutdown(ctx context.Context) error {
	if m.Handlers != nil && m.Handlers.Conversations != nil {
		m.Handlers.Conversations.Close()
	}
	return nil
}
