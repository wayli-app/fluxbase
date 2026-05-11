package api

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/ai"
	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/branching"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/functions"
	"github.com/nimbleflux/fluxbase/internal/jobs"
	"github.com/nimbleflux/fluxbase/internal/mcp"
	"github.com/nimbleflux/fluxbase/internal/mcp/custom"
	mcpresources "github.com/nimbleflux/fluxbase/internal/mcp/resources"
	mcptools "github.com/nimbleflux/fluxbase/internal/mcp/tools"
	"github.com/nimbleflux/fluxbase/internal/migrations"
	"github.com/nimbleflux/fluxbase/internal/rpc"
	"github.com/nimbleflux/fluxbase/internal/storage"
)

type MCPModule struct {
	Handlers *MCPHandlers
}

func (m *MCPModule) Name() string { return "mcp" }

func (m *MCPModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	authService := GetService[*auth.Service](registry)

	m.Handlers = &MCPHandlers{}
	m.Handlers.Handler = mcp.NewHandler(&cfg.MCP, db)
	m.Handlers.OAuth = NewMCPOAuthHandler(db, &cfg.MCP, authService, cfg.BaseURL, cfg.GetPublicBaseURL())

	if !cfg.MCP.Enabled {
		return nil
	}

	schemaCache := GetService[*database.SchemaCache](registry)
	storageService := GetService[*storage.Service](registry)
	functionsHandler := GetService[*functions.Handler](registry)
	rpcHandler := GetService[*rpc.Handler](registry)
	jobsManager := GetService[*jobs.Manager](registry)

	var vectorHandler *VectorHandler
	if vh := GetService[*VectorHandler](registry); vh != nil {
		vectorHandler = vh
	}

	mcpServer := m.Handlers.Handler.Server()
	toolRegistry := mcpServer.ToolRegistry()

	toolRegistry.Register(mcptools.NewThinkTool())

	queryTableTool := mcptools.NewQueryTableTool(db, schemaCache)
	if vectorHandler != nil && vectorHandler.GetEmbeddingService() != nil {
		queryTableTool.SetEmbeddingGenerator(vectorHandler.GetEmbeddingService())
		log.Debug().Msg("MCP: QueryTableTool configured with embedding generator for vector search")
	}
	toolRegistry.Register(queryTableTool)
	toolRegistry.Register(mcptools.NewInsertRecordTool(db, schemaCache))
	toolRegistry.Register(mcptools.NewUpdateRecordTool(db, schemaCache))
	toolRegistry.Register(mcptools.NewDeleteRecordTool(db, schemaCache))
	toolRegistry.Register(mcptools.NewExecuteSQLTool(db))

	if storageService != nil {
		toolRegistry.Register(mcptools.NewListObjectsTool(storageService))
		toolRegistry.Register(mcptools.NewUploadObjectTool(storageService))
		toolRegistry.Register(mcptools.NewDownloadObjectTool(storageService))
		toolRegistry.Register(mcptools.NewDeleteObjectTool(storageService))
	}

	if functionsHandler != nil && cfg.Functions.Enabled {
		toolRegistry.Register(mcptools.NewInvokeFunctionTool(
			db,
			functionsHandler.GetRuntime(),
			functionsHandler.GetPublicURL(),
			functionsHandler.GetFunctionsDir(),
		))
	}

	if rpcHandler != nil && cfg.RPC.Enabled {
		rpcStorage := rpc.NewStorage(db)
		toolRegistry.Register(mcptools.NewInvokeRPCTool(
			rpcHandler.GetExecutor(),
			rpcStorage,
		))
	}

	if jobsManager != nil && cfg.Jobs.Enabled {
		jobsStorage := jobs.NewStorage(db)
		toolRegistry.Register(mcptools.NewSubmitJobTool(jobsStorage))
		toolRegistry.Register(mcptools.NewGetJobStatusTool(jobsStorage))
	}

	aiChatHandler := GetService[*ai.ChatHandler](registry)
	if aiChatHandler != nil {
		if ragService := aiChatHandler.GetRAGService(); ragService != nil {
			toolRegistry.Register(mcptools.NewSearchVectorsTool(ragService))
			log.Debug().Msg("MCP: Registered search_vectors tool")
		} else {
			log.Debug().Msg("MCP: Vector search tool not registered - RAG service not available")
		}
	}

	kbStorage := GetService[*ai.KnowledgeBaseStorage](registry)
	if kbStorage != nil {
		knowledgeGraph := ai.NewKnowledgeGraph(kbStorage)
		toolRegistry.Register(mcptools.NewQueryKnowledgeGraphTool(knowledgeGraph))
		toolRegistry.Register(mcptools.NewFindRelatedEntitiesTool(knowledgeGraph))
		toolRegistry.Register(mcptools.NewBrowseKnowledgeGraphTool(knowledgeGraph))
		log.Debug().Msg("MCP: Registered knowledge graph tools")
	}

	toolRegistry.Register(mcptools.NewListSchemasTool(db))
	toolRegistry.Register(mcptools.NewCreateSchemaTool(db))
	toolRegistry.Register(mcptools.NewCreateTableTool(db))
	toolRegistry.Register(mcptools.NewDropTableTool(db))
	toolRegistry.Register(mcptools.NewAddColumnTool(db))
	toolRegistry.Register(mcptools.NewDropColumnTool(db))
	toolRegistry.Register(mcptools.NewRenameTableTool(db))

	toolRegistry.Register(mcptools.NewHttpRequestTool())

	if cfg.Functions.Enabled {
		functionsStorage := functions.NewStorage(db)
		toolRegistry.Register(mcptools.NewSyncFunctionTool(functionsStorage))
	}

	if cfg.Jobs.Enabled {
		jobsStorage := jobs.NewStorage(db)
		toolRegistry.Register(mcptools.NewSyncJobTool(jobsStorage))
	}

	if cfg.RPC.Enabled {
		rpcStorage := rpc.NewStorage(db)
		toolRegistry.Register(mcptools.NewSyncRPCTool(rpcStorage))
	}

	migrationsStorage := migrations.NewStorage(db)
	migrationsExecutor := migrations.NewExecutor(db)
	toolRegistry.Register(mcptools.NewSyncMigrationTool(migrationsStorage, migrationsExecutor))

	if cfg.AI.Enabled {
		aiStorage := ai.NewStorage(db)
		toolRegistry.Register(mcptools.NewSyncChatbotTool(aiStorage))
	}

	branchManager := GetService[*branching.Manager](registry)
	branchRouter := GetService[*branching.Router](registry)
	if branchManager != nil && cfg.Branching.Enabled {
		branchStorage := branching.NewStorage(db, cfg.EncryptionKeyBytes)
		toolRegistry.Register(mcptools.NewListBranchesTool(branchStorage))
		toolRegistry.Register(mcptools.NewGetBranchTool(branchStorage))
		toolRegistry.Register(mcptools.NewCreateBranchTool(branchManager))
		toolRegistry.Register(mcptools.NewDeleteBranchTool(branchManager, branchStorage))
		toolRegistry.Register(mcptools.NewResetBranchTool(branchManager, branchStorage))
		toolRegistry.Register(mcptools.NewGrantBranchAccessTool(branchStorage))
		toolRegistry.Register(mcptools.NewRevokeBranchAccessTool(branchStorage))
		toolRegistry.Register(mcptools.NewGetActiveBranchTool(branchRouter))
		toolRegistry.Register(mcptools.NewSetActiveBranchTool(branchRouter, branchStorage))
	}

	resourceRegistry := mcpServer.ResourceRegistry()

	resourceRegistry.Register(mcpresources.NewSchemaResource(schemaCache))
	resourceRegistry.Register(mcpresources.NewTableResource(schemaCache))

	if cfg.Functions.Enabled {
		resourceRegistry.Register(mcpresources.NewFunctionsResource(functions.NewStorage(db)))
	}

	if cfg.RPC.Enabled {
		resourceRegistry.Register(mcpresources.NewRPCResource(rpc.NewStorage(db)))
	}

	resourceRegistry.Register(mcpresources.NewBucketsResource(db))

	if aiChatHandler != nil {
		aiChatHandler.SetMCPToolRegistry(toolRegistry)
		aiChatHandler.SetMCPResources(resourceRegistry)
		log.Debug().Msg("MCP registries wired to AI chat handler")
	}

	customStorage := custom.NewStorage(db)
	mcpInternalURL := cfg.BaseURL
	if mcpInternalURL == "" {
		mcpInternalURL = "http://localhost" + cfg.Server.Address
	}
	customExecutor := custom.NewExecutor(cfg.Auth.JWTSecret, mcpInternalURL, nil)
	m.Handlers.CustomManager = custom.NewManager(customStorage, customExecutor, toolRegistry, resourceRegistry)
	m.Handlers.CustomHandler = NewCustomMCPHandler(customStorage, m.Handlers.CustomManager, &cfg.MCP)

	loadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := m.Handlers.CustomManager.LoadAndRegisterAll(loadCtx); err != nil {
		log.Warn().Err(err).Msg("Failed to load some custom MCP tools/resources")
	}

	log.Debug().
		Int("tools", len(toolRegistry.ListTools(&mcp.AuthContext{IsServiceRole: true}))).
		Int("resources", len(resourceRegistry.ListResources(&mcp.AuthContext{IsServiceRole: true}))).
		Msg("MCP Server initialized with tools and resources")

	if cfg.MCP.Enabled && cfg.MCP.AutoLoadOnBoot && m.Handlers.CustomManager != nil {
		if err := m.Handlers.CustomManager.AutoLoadFromDir(context.Background(), cfg.MCP.ToolsDir); err != nil {
			log.Error().Err(err).Msg("Failed to auto-load custom MCP tools")
		} else {
			log.Info().Msg("Custom MCP tools auto-loaded successfully")
		}
	}

	log.Info().
		Str("base_path", cfg.MCP.BasePath).
		Dur("session_timeout", cfg.MCP.SessionTimeout).
		Msg("MCP Server enabled")

	return nil
}
