package ai

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// ListMyEntities lists entities in a knowledge base
// GET /api/v1/ai/knowledge-bases/:id/entities
func (h *UserKnowledgeBaseHandler) ListMyEntities(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")

	// Check viewer permission
	hasPermission, err := h.storage.CheckKBPermission(ctx, kbID, userID, string(KBPermissionViewer))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check permission",
		})
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Viewer permission required",
		})
	}

	// Check if knowledge graph is available
	if h.knowledgeGraph == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Knowledge graph features are not available",
		})
	}

	// Parse optional entity_type filter
	entityTypeStr := c.Query("entity_type")
	var entityType *EntityType
	if entityTypeStr != "" {
		et := EntityType(entityTypeStr)
		entityType = &et
	}

	// Get entities
	entities, err := h.knowledgeGraph.ListEntities(ctx, kbID, entityType)
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

// SearchMyEntities searches entities in a knowledge base
// GET /api/v1/ai/knowledge-bases/:id/entities/search
func (h *UserKnowledgeBaseHandler) SearchMyEntities(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")

	// Check viewer permission
	hasPermission, err := h.storage.CheckKBPermission(ctx, kbID, userID, string(KBPermissionViewer))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check permission",
		})
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Viewer permission required",
		})
	}

	// Check if knowledge graph is available
	if h.knowledgeGraph == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Knowledge graph features are not available",
		})
	}

	// Get query from URL param
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query parameter 'q' is required",
		})
	}

	// Parse optional entity types filter
	var entityTypes []EntityType
	if typeStr := c.Query("entity_types"); typeStr != "" {
		for _, t := range splitCommaSeparated(typeStr) {
			entityTypes = append(entityTypes, EntityType(t))
		}
	}

	// Parse limit
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := parseIntParam(limitStr, 1, 100); err == nil {
			limit = l
		}
	}

	// Search entities
	entities, err := h.knowledgeGraph.SearchEntities(ctx, kbID, query, entityTypes, limit)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Str("query", query).Msg("Failed to search entities")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search entities",
		})
	}

	return c.JSON(fiber.Map{
		"entities": entities,
		"query":    query,
		"count":    len(entities),
	})
}

// GetMyEntityRelationships gets relationships for an entity
// GET /api/v1/ai/knowledge-bases/:id/entities/:entity_id/relationships
func (h *UserKnowledgeBaseHandler) GetMyEntityRelationships(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")
	entityID := c.Params("entity_id")

	// Check viewer permission
	hasPermission, err := h.storage.CheckKBPermission(ctx, kbID, userID, string(KBPermissionViewer))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check permission",
		})
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Viewer permission required",
		})
	}

	// Check if knowledge graph is available
	if h.knowledgeGraph == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Knowledge graph features are not available",
		})
	}

	// Get relationships for the entity
	relationships, err := h.knowledgeGraph.GetRelationships(ctx, kbID, entityID)
	if err != nil {
		log.Error().Err(err).Str("kb_id", kbID).Str("entity_id", entityID).Msg("Failed to get entity relationships")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get entity relationships",
		})
	}

	return c.JSON(fiber.Map{
		"relationships": relationships,
		"entity_id":     entityID,
		"count":         len(relationships),
	})
}

// GetMyKnowledgeGraph gets the full knowledge graph
// GET /api/v1/ai/knowledge-bases/:id/graph
func (h *UserKnowledgeBaseHandler) GetMyKnowledgeGraph(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")

	// Check viewer permission
	hasPermission, err := h.storage.CheckKBPermission(ctx, kbID, userID, string(KBPermissionViewer))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check permission",
		})
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Viewer permission required",
		})
	}

	// Check if knowledge graph is available
	if h.knowledgeGraph == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Knowledge graph features are not available",
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

	// Get relationships for each entity and collect unique ones
	allRelationships := make(map[string]EntityRelationship)
	for _, entity := range entities {
		relationships, err := h.knowledgeGraph.GetRelationships(ctx, kbID, entity.ID)
		if err != nil {
			log.Warn().Err(err).Str("entity_id", entity.ID).Msg("Failed to get relationships for entity")
			continue
		}
		for _, rel := range relationships {
			allRelationships[rel.ID] = rel
		}
	}

	// Convert map to slice
	relationships := make([]EntityRelationship, 0, len(allRelationships))
	for _, rel := range allRelationships {
		relationships = append(relationships, rel)
	}

	return c.JSON(fiber.Map{
		"knowledge_base_id":  kbID,
		"entities":           entities,
		"relationships":      relationships,
		"entity_count":       len(entities),
		"relationship_count": len(relationships),
	})
}

// ListMyLinkedChatbots lists chatbots linked to a knowledge base
// GET /api/v1/ai/knowledge-bases/:id/chatbots
func (h *UserKnowledgeBaseHandler) ListMyLinkedChatbots(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	userID := middleware.GetUserID(c)
	kbID := c.Params("id")

	// Check viewer permission
	hasPermission, err := h.storage.CheckKBPermission(ctx, kbID, userID, string(KBPermissionViewer))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check permission",
		})
	}
	if !hasPermission {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Viewer permission required",
		})
	}

	// Get linked chatbots
	links, err := h.storage.GetKnowledgeBaseChatbots(ctx, kbID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get linked chatbots",
		})
	}

	return c.JSON(fiber.Map{
		"chatbots": links,
		"count":    len(links),
	})
}

// splitCommaSeparated splits a comma-separated string into trimmed parts
func splitCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseIntParam parses an integer parameter with min/max bounds
func parseIntParam(s string, min, max int) (int, error) {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if val < min {
		return min, nil
	}
	if val > max {
		return max, nil
	}
	return val, nil
}
