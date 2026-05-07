package ai

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/storage"
)

// KnowledgeBaseHandler handles knowledge base management endpoints
type KnowledgeBaseHandler struct {
	storage        *KnowledgeBaseStorage
	processor      *DocumentProcessor
	storageService *storage.Service
	textExtractor  *TextExtractor
	ocrService     *OCRService
	tableExporter  *TableExporter
	knowledgeGraph *KnowledgeGraph
	syncService    *TableExportSyncService
}

// NewKnowledgeBaseHandler creates a new knowledge base handler
func NewKnowledgeBaseHandler(storage *KnowledgeBaseStorage, processor *DocumentProcessor) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{
		storage:       storage,
		processor:     processor,
		textExtractor: NewTextExtractor(),
	}
}

// NewKnowledgeBaseHandlerWithOCR creates a new knowledge base handler with OCR support
func NewKnowledgeBaseHandlerWithOCR(storage *KnowledgeBaseStorage, processor *DocumentProcessor, ocrService *OCRService) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{
		storage:       storage,
		processor:     processor,
		textExtractor: NewTextExtractorWithOCR(ocrService),
		ocrService:    ocrService,
	}
}

// SetStorageService sets the storage service for file uploads
func (h *KnowledgeBaseHandler) SetStorageService(svc *storage.Service) {
	h.storageService = svc
}

// SetKnowledgeGraph sets the knowledge graph service for entity operations
func (h *KnowledgeBaseHandler) SetKnowledgeGraph(kg *KnowledgeGraph) {
	h.knowledgeGraph = kg
}

// ============================================================================
// KNOWLEDGE BASE ENDPOINTS
// ============================================================================

// ListKnowledgeBases returns all knowledge bases
// GET /api/v1/admin/ai/knowledge-bases
func (h *KnowledgeBaseHandler) ListKnowledgeBases(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	// Parse optional namespace filter
	namespace := c.Query("namespace", "") // Empty = all namespaces
	if namespace == "default" {
		namespace = ""
	}

	kbs, err := h.storage.ListKnowledgeBases(ctx, namespace, false)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list knowledge bases")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list knowledge bases",
		})
	}

	// Convert to summaries
	summaries := make([]KnowledgeBaseSummary, len(kbs))
	for i, kb := range kbs {
		summaries[i] = kb.ToSummary()
	}

	return c.JSON(fiber.Map{
		"knowledge_bases": summaries,
		"count":           len(summaries),
	})
}

// GetKnowledgeBase returns a specific knowledge base
// GET /api/v1/admin/ai/knowledge-bases/:id
func (h *KnowledgeBaseHandler) GetKnowledgeBase(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	kb, err := h.storage.GetKnowledgeBase(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get knowledge base",
		})
	}
	if kb == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Knowledge base not found",
		})
	}

	return c.JSON(kb)
}

// CreateKnowledgeBase creates a new knowledge base
// POST /api/v1/admin/ai/knowledge-bases
func (h *KnowledgeBaseHandler) CreateKnowledgeBase(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	var req CreateKnowledgeBaseRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}

	kb, err := h.storage.CreateKnowledgeBaseFromRequest(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create knowledge base",
		})
	}

	// Set created_by and owner_id to current user if available
	// For service role operations without a user context, these remain NULL
	if uid := middleware.GetUserID(c); uid != "" {
		kb.CreatedBy = &uid
		kb.OwnerID = &uid
	}
	if kb.CreatedBy != nil {
		if err := h.storage.UpdateKnowledgeBase(ctx, kb); err != nil {
			log.Warn().Err(err).Msg("Failed to set KB owner")
		}
	}

	return c.Status(fiber.StatusCreated).JSON(kb)
}

// UpdateKnowledgeBase updates an existing knowledge base
// PUT /api/v1/admin/ai/knowledge-bases/:id
func (h *KnowledgeBaseHandler) UpdateKnowledgeBase(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	var req UpdateKnowledgeBaseRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	kb, err := h.storage.UpdateKnowledgeBaseByID(ctx, id, req)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to update knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update knowledge base",
		})
	}
	if kb == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Knowledge base not found",
		})
	}

	return c.JSON(kb)
}

// DeleteKnowledgeBase deletes a knowledge base
// DELETE /api/v1/admin/ai/knowledge-bases/:id
func (h *KnowledgeBaseHandler) DeleteKnowledgeBase(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	err := h.storage.DeleteKnowledgeBase(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to delete knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete knowledge base",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ============================================================================
// CHATBOT-KNOWLEDGE BASE LINKING ENDPOINTS
// ============================================================================

// ListChatbotKnowledgeBases returns knowledge bases linked to a chatbot
// GET /api/v1/admin/ai/chatbots/:id/knowledge-bases
func (h *KnowledgeBaseHandler) ListChatbotKnowledgeBases(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	chatbotID := c.Params("id")

	if chatbotID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Chatbot ID is required",
		})
	}

	links, err := h.storage.GetChatbotKnowledgeBases(ctx, chatbotID)
	if err != nil {
		log.Error().Err(err).Str("chatbot_id", chatbotID).Msg("Failed to get chatbot knowledge bases")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get chatbot knowledge bases",
		})
	}

	return c.JSON(fiber.Map{
		"knowledge_bases": links,
		"count":           len(links),
	})
}

// LinkKnowledgeBase links a knowledge base to a chatbot
// POST /api/v1/admin/ai/chatbots/:id/knowledge-bases
func (h *KnowledgeBaseHandler) LinkKnowledgeBase(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	chatbotID := c.Params("id")

	if chatbotID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Chatbot ID is required",
		})
	}

	var req LinkKnowledgeBaseRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.KnowledgeBaseID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	// Set defaults
	priority := 1
	maxChunks := 5
	similarityThreshold := 0.7

	if req.Priority != nil {
		priority = *req.Priority
	}
	if req.MaxChunks != nil {
		maxChunks = *req.MaxChunks
	}
	if req.SimilarityThreshold != nil {
		similarityThreshold = *req.SimilarityThreshold
	}

	link, err := h.storage.LinkChatbotKnowledgeBaseSimple(ctx, chatbotID, req.KnowledgeBaseID, priority, maxChunks, similarityThreshold)
	if err != nil {
		log.Error().Err(err).
			Str("chatbot_id", chatbotID).
			Str("kb_id", req.KnowledgeBaseID).
			Msg("Failed to link knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to link knowledge base",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(link)
}

// UpdateChatbotKnowledgeBaseRequest represents a request to update a link
type UpdateChatbotKnowledgeBaseRequest struct {
	Priority            *int     `json:"priority,omitempty"`
	MaxChunks           *int     `json:"max_chunks,omitempty"`
	SimilarityThreshold *float64 `json:"similarity_threshold,omitempty"`
	Enabled             *bool    `json:"enabled,omitempty"`
}

// UpdateChatbotKnowledgeBase updates a chatbot-knowledge base link
// PUT /api/v1/admin/ai/chatbots/:id/knowledge-bases/:kb_id
func (h *KnowledgeBaseHandler) UpdateChatbotKnowledgeBase(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	chatbotID := c.Params("id")
	kbID := c.Params("kb_id")

	if chatbotID == "" || kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Chatbot ID and knowledge base ID are required",
		})
	}

	var req UpdateChatbotKnowledgeBaseRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	opts := UpdateChatbotKnowledgeBaseOptions{
		Priority:            req.Priority,
		MaxChunks:           req.MaxChunks,
		SimilarityThreshold: req.SimilarityThreshold,
		Enabled:             req.Enabled,
	}

	link, err := h.storage.UpdateChatbotKnowledgeBaseLink(ctx, chatbotID, kbID, opts)
	if err != nil {
		log.Error().Err(err).
			Str("chatbot_id", chatbotID).
			Str("kb_id", kbID).
			Msg("Failed to update chatbot knowledge base link")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update chatbot knowledge base link",
		})
	}
	if link == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Chatbot-knowledge base link not found",
		})
	}

	return c.JSON(link)
}

// UnlinkKnowledgeBase removes a knowledge base from a chatbot
// DELETE /api/v1/admin/ai/chatbots/:id/knowledge-bases/:kb_id
func (h *KnowledgeBaseHandler) UnlinkKnowledgeBase(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	chatbotID := c.Params("id")
	kbID := c.Params("kb_id")

	if chatbotID == "" || kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Chatbot ID and knowledge base ID are required",
		})
	}

	err := h.storage.UnlinkChatbotKnowledgeBase(ctx, chatbotID, kbID)
	if err != nil {
		log.Error().Err(err).
			Str("chatbot_id", chatbotID).
			Str("kb_id", kbID).
			Msg("Failed to unlink knowledge base")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to unlink knowledge base",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ============================================================================
// TABLE EXPORT ENDPOINTS
// ============================================================================

// ExportTableToKnowledgeBase exports a database table as a knowledge base document
// POST /api/v1/admin/ai/knowledge-bases/:id/tables/export
func (h *KnowledgeBaseHandler) ExportTableToKnowledgeBase(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	var req ExportTableRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.KnowledgeBaseID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	kbID := req.KnowledgeBaseID

	// Validate required fields
	if req.Schema == "" || req.Table == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "schema and table are required",
		})
	}

	// Determine owner_id for the document
	// Priority: 1) authenticated user, 2) KB's owner_id, 3) KB's created_by, 4) nil for system documents
	if uid := middleware.GetUserID(c); uid != "" {
		req.OwnerID = &uid
	} else if kb, err := h.storage.GetKnowledgeBase(ctx, kbID); err == nil && kb != nil {
		if kb.OwnerID != nil {
			req.OwnerID = kb.OwnerID
		} else if kb.CreatedBy != nil {
			req.OwnerID = kb.CreatedBy
		}
	}

	// If we still don't have an owner_id, leave it as nil (will be NULL in database)
	// This is acceptable for system-generated documents created via service role
	// The database migration 096 allows NULL owner_id for such cases

	// Set defaults
	if req.SampleRowCount == 0 {
		req.SampleRowCount = 5
	}

	if h.tableExporter == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Table export service not configured",
		})
	}

	result, err := h.tableExporter.ExportTable(ctx, req)
	if err != nil {
		log.Error().Err(err).Str("table", req.Table).Msg("Failed to export table")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to export table: %v", err),
		})
	}

	return c.JSON(result)
}

// ListExportableTables lists all tables that can be exported to knowledge bases
// GET /api/v1/admin/ai/tables?schema=public&knowledge_base_id=xxx
func (h *KnowledgeBaseHandler) ListExportableTables(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	schema := c.Query("schema", "public")
	kbID := c.Query("knowledge_base_id", "")

	if h.tableExporter == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Table export service not configured",
		})
	}

	tables, err := h.tableExporter.ListExportableTables(ctx, []string{schema})
	if err != nil {
		log.Error().Err(err).Str("schema", schema).Msg("Failed to list tables")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list tables",
		})
	}

	// Return simplified table info
	type TableSummary struct {
		Schema      string `json:"schema"`
		Name        string `json:"name"`
		Columns     int    `json:"columns"`
		ForeignKeys int    `json:"foreign_keys"`
		// Optional: last export time if knowledge_base_id is provided
		LastExport *string `json:"last_export,omitempty"`
	}

	summaries := make([]TableSummary, len(tables))

	// If KB ID is provided, fetch all exported table documents at once
	var exportedTableDocs map[string]*Document // key: "schema.table"
	if kbID != "" {
		// Get all documents with source=database_export
		docs, err := h.storage.ListDocuments(ctx, kbID)
		if err == nil {
			exportedTableDocs = make(map[string]*Document)
			for i := range docs {
				// Parse metadata to check if this is a table export
				var metadata map[string]interface{}
				if docs[i].Metadata != nil {
					if err := json.Unmarshal(docs[i].Metadata, &metadata); err == nil {
						// Check if this is a table export document
						if source, ok := metadata["source"].(string); ok && source == "database_export" {
							if schema, ok := metadata["schema"].(string); ok && schema != "" {
								if table, ok := metadata["table"].(string); ok && table != "" {
									key := schema + "." + table
									exportedTableDocs[key] = &docs[i]
								}
							}
						}
					}
				}
			}
		}
	}

	for i, t := range tables {
		summary := TableSummary{
			Schema:      t.Schema,
			Name:        t.Name,
			Columns:     len(t.Columns),
			ForeignKeys: len(t.ForeignKeys),
		}

		// If we have export info for this table, add the last export time
		if exportedTableDocs != nil {
			key := t.Schema + "." + t.Name
			if doc := exportedTableDocs[key]; doc != nil {
				// Format the updated_at time as ISO string
				lastExport := doc.UpdatedAt.Format(time.RFC3339)
				summary.LastExport = &lastExport
			}
		}

		summaries[i] = summary
	}

	return c.JSON(fiber.Map{
		"tables": summaries,
		"count":  len(summaries),
	})
}

// GetTableDetails returns detailed info about a table for column selection
// GET /api/v1/admin/ai/tables/:schema/:table
func (h *KnowledgeBaseHandler) GetTableDetails(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	schema := c.Params("schema")
	table := c.Params("table")

	if schema == "" || table == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Schema and table parameters are required",
		})
	}

	if h.tableExporter == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Table export service not configured",
		})
	}

	inspector := database.NewSchemaInspector(h.tableExporter.conn)
	tableInfo, err := inspector.GetTableInfo(ctx, schema, table)
	if err != nil {
		log.Error().Err(err).Str("schema", schema).Str("table", table).Msg("Failed to get table info")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get table info",
		})
	}
	if tableInfo == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Table not found",
		})
	}

	return c.JSON(tableInfo)
}

// ============================================================================
// TABLE EXPORT SYNC CONFIG ENDPOINTS
// ============================================================================

// CreateTableExportSync creates a sync config and optionally triggers initial export
// POST /api/v1/admin/ai/knowledge-bases/:id/sync-configs
func (h *KnowledgeBaseHandler) CreateTableExportSync(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	kbID := c.Params("id")

	if kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	if h.syncService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Sync service not configured",
		})
	}

	var req CreateTableExportSyncConfig
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	req.KnowledgeBaseID = kbID

	config, err := h.syncService.CreateSyncConfig(ctx, &req)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to create sync config")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create sync config: %v", err),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(config)
}

// ListTableExportSyncs lists sync configs for a knowledge base
// GET /api/v1/admin/ai/knowledge-bases/:id/sync-configs
func (h *KnowledgeBaseHandler) ListTableExportSyncs(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	kbID := c.Params("id")

	if kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	if h.syncService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Sync service not configured",
		})
	}

	configs, err := h.syncService.GetSyncConfigsByKnowledgeBase(ctx, kbID)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to list sync configs")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list sync configs",
		})
	}

	return c.JSON(fiber.Map{
		"sync_configs": configs,
		"count":        len(configs),
	})
}

// UpdateTableExportSync updates a sync config
// PATCH /api/v1/admin/ai/knowledge-bases/:id/sync-configs/:syncId
func (h *KnowledgeBaseHandler) UpdateTableExportSync(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	syncID := c.Params("syncId")

	if syncID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Sync config ID is required",
		})
	}

	if h.syncService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Sync service not configured",
		})
	}

	var req UpdateTableExportSyncConfig
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	config, err := h.syncService.UpdateSyncConfig(ctx, syncID, req)
	if err != nil {
		log.Error().Err(err).Str("sync_id", syncID).Msg("Failed to update sync config")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to update sync config: %v", err),
		})
	}

	return c.JSON(config)
}

// DeleteTableExportSync deletes a sync config
// DELETE /api/v1/admin/ai/knowledge-bases/:id/sync-configs/:syncId
func (h *KnowledgeBaseHandler) DeleteTableExportSync(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	syncID := c.Params("syncId")

	if syncID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Sync config ID is required",
		})
	}

	if h.syncService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Sync service not configured",
		})
	}

	err := h.syncService.DeleteSyncConfig(ctx, syncID)
	if err != nil {
		log.Error().Err(err).Str("sync_id", syncID).Msg("Failed to delete sync config")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to delete sync config: %v", err),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// TriggerTableExportSync manually triggers a sync
// POST /api/v1/admin/ai/knowledge-bases/:id/sync-configs/:syncId/trigger
func (h *KnowledgeBaseHandler) TriggerTableExportSync(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	syncID := c.Params("syncId")

	if syncID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Sync config ID is required",
		})
	}

	if h.syncService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Sync service not configured",
		})
	}

	result, err := h.syncService.TriggerSync(ctx, syncID)
	if err != nil {
		log.Error().Err(err).Str("sync_id", syncID).Msg("Failed to trigger sync")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to trigger sync: %v", err),
		})
	}

	return c.JSON(result)
}

// ============================================================================
// KNOWLEDGE BASE CHATBOTS (Reverse Lookup)
// ============================================================================

// ListKnowledgeBaseChatbots returns all chatbots linked to a knowledge base
// GET /api/v1/admin/ai/knowledge-bases/:id/chatbots
func (h *KnowledgeBaseHandler) ListKnowledgeBaseChatbots(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	kbID := c.Params("id")

	if kbID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Knowledge base ID is required",
		})
	}

	links, err := h.storage.GetKnowledgeBaseChatbots(ctx, kbID)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to get knowledge base chatbots")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get linked chatbots",
		})
	}

	return c.JSON(fiber.Map{
		"chatbots": links,
		"count":    len(links),
	})
}

// ============================================================================
// KNOWLEDGE GRAPH ENDPOINTS
// ============================================================================

// ListEntities lists all entities in a knowledge base
// GET /api/v1/admin/ai/knowledge-bases/:id/entities
func (h *KnowledgeBaseHandler) ListEntities(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	kbID := c.Params("id")
	entityType := c.Query("type")

	if h.knowledgeGraph == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Knowledge graph service not available",
		})
	}

	var entityTypeFilter *EntityType
	if entityType != "" {
		t := EntityType(entityType)
		entityTypeFilter = &t
	}

	entities, err := h.knowledgeGraph.ListEntities(ctx, kbID, entityTypeFilter)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to list entities")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list entities",
		})
	}

	return c.JSON(fiber.Map{
		"entities": entities,
		"count":    len(entities),
	})
}

// SearchEntities searches entities by name
// GET /api/v1/admin/ai/knowledge-bases/:id/entities/search
func (h *KnowledgeBaseHandler) SearchEntities(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	kbID := c.Params("id")
	query := c.Query("q")
	entityTypes := c.Query("types")
	limitStr := c.Query("limit", "50")

	limit := 50
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if h.knowledgeGraph == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Knowledge graph service not available",
		})
	}

	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query parameter 'q' is required",
		})
	}

	var types []EntityType
	if entityTypes != "" {
		for _, t := range splitCommaList(entityTypes) {
			types = append(types, EntityType(t))
		}
	}

	entities, err := h.knowledgeGraph.SearchEntities(ctx, kbID, query, types, limit)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Str("query", query).Msg("Failed to search entities")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search entities",
		})
	}

	return c.JSON(fiber.Map{
		"entities": entities,
		"count":    len(entities),
	})
}

// GetEntityRelationships gets relationships for an entity
// GET /api/v1/admin/ai/knowledge-bases/:id/entities/:entity_id/relationships
func (h *KnowledgeBaseHandler) GetEntityRelationships(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	kbID := c.Params("id")
	entityID := c.Params("entity_id")

	if h.knowledgeGraph == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Knowledge graph service not available",
		})
	}

	if entityID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Entity ID is required",
		})
	}

	relationships, err := h.knowledgeGraph.GetRelationships(ctx, kbID, entityID)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Str("entity_id", entityID).Msg("Failed to get entity relationships")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get entity relationships",
		})
	}

	return c.JSON(fiber.Map{
		"relationships": relationships,
		"count":         len(relationships),
	})
}

// GetKnowledgeGraph returns the full graph data for visualization
// GET /api/v1/admin/ai/knowledge-bases/:id/graph
func (h *KnowledgeBaseHandler) GetKnowledgeGraph(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	kbID := c.Params("id")

	if h.knowledgeGraph == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Knowledge graph service not available",
		})
	}

	// Get all entities
	entities, err := h.knowledgeGraph.ListEntities(ctx, kbID, nil)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Msg("Failed to list entities for graph")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get knowledge graph",
		})
	}

	// Get document counts for each entity (for display in the graph)
	entityDocCounts := make(map[string]int)
	if len(entities) > 0 {
		// Query all document-entity counts in one go
		entityIDs := make([]string, len(entities))
		for i, e := range entities {
			entityIDs[i] = e.ID
		}

		query := `
			SELECT entity_id, COUNT(*) as doc_count
			FROM ai.document_entities
			WHERE entity_id = ANY($1)
			GROUP BY entity_id
		`
		rows, err := h.storage.DB.Query(ctx, query, entityIDs)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var entityID string
				var count int
				if err := rows.Scan(&entityID, &count); err == nil {
					entityDocCounts[entityID] = count
				}
			}
		}
	}

	// Add document count to each entity
	for i := range entities {
		if count, ok := entityDocCounts[entities[i].ID]; ok {
			// Add document_count to metadata for display
			if entities[i].Metadata == nil {
				entities[i].Metadata = make(map[string]interface{})
			}
			entities[i].Metadata["document_count"] = count
		}
	}

	// Get relationships for each entity
	var allRelationships []EntityRelationship
	relationshipMap := make(map[string]bool) // Deduplicate relationships

	for _, entity := range entities {
		relationships, err := h.knowledgeGraph.GetRelationships(ctx, kbID, entity.ID)
		if err != nil {
			log.Warn().Err(err).Str("entity_id", entity.ID).Msg("Failed to get relationships for entity")
			continue
		}
		for _, rel := range relationships {
			key := rel.ID
			if !relationshipMap[key] {
				relationshipMap[key] = true
				allRelationships = append(allRelationships, rel)
			}
		}
	}

	return c.JSON(fiber.Map{
		"entities":           entities,
		"relationships":      allRelationships,
		"entity_count":       len(entities),
		"relationship_count": len(allRelationships),
	})
}

// splitCommaList splits a comma-separated string into a slice
func splitCommaList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// fiber:context-methods migrated
