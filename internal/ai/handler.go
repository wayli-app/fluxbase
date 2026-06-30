package ai

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// VectorManagerInterface defines the interface for hot-reloading embedding service
// This interface avoids circular dependency with internal/api package
type VectorManagerInterface interface {
	RefreshFromDatabase(ctx context.Context) error
}

// Handler handles AI-related HTTP endpoints
type Handler struct {
	storage              *Storage
	loader               *Loader
	config               *config.AIConfig
	vectorManager        VectorManagerInterface
	knowledgeBaseStorage *KnowledgeBaseStorage // Optional: for syncing KB links
	chatHandler          *ChatHandler          // Optional: for provider cache invalidation
}

// NewHandler creates a new AI handler
func NewHandler(storage *Storage, loader *Loader, cfg *config.AIConfig, vectorManager VectorManagerInterface) *Handler {
	h := &Handler{
		storage:       storage,
		loader:        loader,
		config:        cfg,
		vectorManager: vectorManager,
	}

	// Validate config at startup
	h.ValidateConfig()

	return h
}

// SetChatHandler wires the chat handler so provider mutations can invalidate
// the chat path's provider cache. Called after both handlers are constructed.
func (h *Handler) SetChatHandler(ch *ChatHandler) {
	h.chatHandler = ch
}

// invalidateProvider drops the provider from the chat handler's cache (if wired).
// Call this whenever a provider's config, default status, or existence changes.
func (h *Handler) invalidateProvider(providerID string) {
	if h.chatHandler != nil {
		h.chatHandler.InvalidateProvider(providerID)
	}
}

// SetKnowledgeBaseStorage sets the knowledge base storage for syncing KB links
func (h *Handler) SetKnowledgeBaseStorage(kbStorage *KnowledgeBaseStorage) {
	h.knowledgeBaseStorage = kbStorage
}

// ValidateConfig checks AI configuration and logs any issues at startup
func (h *Handler) ValidateConfig() {
	if h.config == nil || h.config.ProviderType == "" {
		return
	}

	switch h.config.ProviderType {
	case "ollama":
		if h.config.OllamaModel == "" {
			log.Warn().
				Str("issue", "missing_ollama_model").
				Str("provider_type", "ollama").
				Msg("AI provider configured as Ollama but FLUXBASE_AI_OLLAMA_MODEL is not set. The Ollama provider will NOT appear in the provider list until a model is configured.")
		}
	case "openai":
		if h.config.OpenAIAPIKey == "" {
			log.Warn().
				Str("issue", "missing_openai_api_key").
				Str("provider_type", "openai").
				Msg("AI provider configured as OpenAI but FLUXBASE_AI_OPENAI_API_KEY is not set. The OpenAI provider will NOT appear in the provider list.")
		}
	case "azure":
		var missing []string
		if h.config.AzureAPIKey == "" {
			missing = append(missing, "FLUXBASE_AI_AZURE_API_KEY")
		}
		if h.config.AzureEndpoint == "" {
			missing = append(missing, "FLUXBASE_AI_AZURE_ENDPOINT")
		}
		if h.config.AzureDeploymentName == "" {
			missing = append(missing, "FLUXBASE_AI_AZURE_DEPLOYMENT_NAME")
		}
		if len(missing) > 0 {
			log.Warn().
				Strs("missing_vars", missing).
				Str("provider_type", "azure").
				Msg("AI provider configured as Azure but some required environment variables are not set. The Azure provider will NOT appear in the provider list.")
		}
	}
}

// ============================================================================
// CHATBOT ENDPOINTS
// ============================================================================

// ListChatbots returns all chatbots (admin view)
// GET /api/v1/admin/ai/chatbots
func (h *Handler) ListChatbots(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	chatbots, err := h.storage.ListChatbots(ctx, false)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list chatbots")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list chatbots",
		})
	}

	// Convert to summaries for API response
	summaries := make([]ChatbotSummary, len(chatbots))
	for i, cb := range chatbots {
		summaries[i] = cb.ToSummary()
	}

	return c.JSON(fiber.Map{
		"chatbots": summaries,
		"count":    len(summaries),
	})
}

// GetChatbot returns a single chatbot by ID (admin view)
// GET /api/v1/admin/ai/chatbots/:id
func (h *Handler) GetChatbot(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	chatbot, err := h.storage.GetChatbot(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get chatbot")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get chatbot",
		})
	}

	if chatbot == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Chatbot not found",
		})
	}

	return c.JSON(chatbot)
}

// ToggleChatbotRequest represents the request to enable/disable a chatbot
type ToggleChatbotRequest struct {
	Enabled bool `json:"enabled"`
}

// ToggleChatbot enables or disables a chatbot
// PUT /api/v1/admin/ai/chatbots/:id/toggle
func (h *Handler) ToggleChatbot(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	var req ToggleChatbotRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	chatbot, err := h.storage.GetChatbot(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get chatbot")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get chatbot",
		})
	}

	if chatbot == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Chatbot not found",
		})
	}

	chatbot.Enabled = req.Enabled
	if err := h.storage.UpdateChatbot(ctx, chatbot); err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to update chatbot")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update chatbot",
		})
	}

	return c.JSON(fiber.Map{
		"id":      id,
		"enabled": req.Enabled,
	})
}

// DeleteChatbot deletes a chatbot
// DELETE /api/v1/admin/ai/chatbots/:id
func (h *Handler) DeleteChatbot(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	if err := h.storage.DeleteChatbot(ctx, id); err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to delete chatbot")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete chatbot",
		})
	}

	return c.JSON(fiber.Map{
		"deleted": true,
		"id":      id,
	})
}

// UpdateChatbotRequest represents the request to update chatbot configuration
type UpdateChatbotRequest struct {
	Description          *string  `json:"description"`
	Enabled              *bool    `json:"enabled"`
	MaxTokens            *int     `json:"max_tokens"`
	Temperature          *float64 `json:"temperature"`
	ProviderID           *string  `json:"provider_id"`
	PersistConversations *bool    `json:"persist_conversations"`
	ConversationTTLHours *int     `json:"conversation_ttl_hours"`
	MaxConversationTurns *int     `json:"max_conversation_turns"`
	RateLimitPerMinute   *int     `json:"rate_limit_per_minute"`
	DailyRequestLimit    *int     `json:"daily_request_limit"`
	DailyTokenBudget     *int     `json:"daily_token_budget"`
	AllowUnauthenticated *bool    `json:"allow_unauthenticated"`
	IsPublic             *bool    `json:"is_public"`
}

// UpdateChatbot updates a chatbot's configuration
// PUT /api/v1/admin/ai/chatbots/:id
func (h *Handler) UpdateChatbot(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	var req UpdateChatbotRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate inputs
	if req.Temperature != nil && (*req.Temperature < 0 || *req.Temperature > 2) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Temperature must be between 0 and 2",
		})
	}
	if req.MaxTokens != nil && *req.MaxTokens <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Max tokens must be positive",
		})
	}
	if req.ConversationTTLHours != nil && *req.ConversationTTLHours <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Conversation TTL hours must be positive",
		})
	}
	if req.MaxConversationTurns != nil && *req.MaxConversationTurns <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Max conversation turns must be positive",
		})
	}
	if req.RateLimitPerMinute != nil && *req.RateLimitPerMinute <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Rate limit per minute must be positive",
		})
	}
	if req.DailyRequestLimit != nil && *req.DailyRequestLimit <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Daily request limit must be positive",
		})
	}
	if req.DailyTokenBudget != nil && *req.DailyTokenBudget <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Daily token budget must be positive",
		})
	}

	// Get existing chatbot
	chatbot, err := h.storage.GetChatbot(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get chatbot")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get chatbot",
		})
	}

	if chatbot == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Chatbot not found",
		})
	}

	// Apply partial updates (only non-nil fields)
	if req.Description != nil {
		chatbot.Description = *req.Description
	}
	if req.Enabled != nil {
		chatbot.Enabled = *req.Enabled
	}
	if req.MaxTokens != nil {
		chatbot.MaxTokens = *req.MaxTokens
	}
	if req.Temperature != nil {
		chatbot.Temperature = *req.Temperature
	}
	if req.ProviderID != nil {
		if *req.ProviderID == "" {
			chatbot.ProviderID = nil
		} else {
			chatbot.ProviderID = req.ProviderID
		}
	}
	if req.PersistConversations != nil {
		chatbot.PersistConversations = *req.PersistConversations
	}
	if req.ConversationTTLHours != nil {
		chatbot.ConversationTTLHours = *req.ConversationTTLHours
	}
	if req.MaxConversationTurns != nil {
		chatbot.MaxConversationTurns = *req.MaxConversationTurns
	}
	if req.RateLimitPerMinute != nil {
		chatbot.RateLimitPerMinute = *req.RateLimitPerMinute
	}
	if req.DailyRequestLimit != nil {
		chatbot.DailyRequestLimit = *req.DailyRequestLimit
	}
	if req.DailyTokenBudget != nil {
		chatbot.DailyTokenBudget = *req.DailyTokenBudget
	}
	if req.AllowUnauthenticated != nil {
		chatbot.AllowUnauthenticated = *req.AllowUnauthenticated
	}
	if req.IsPublic != nil {
		chatbot.IsPublic = *req.IsPublic
	}

	// Update in database
	if err := h.storage.UpdateChatbot(ctx, chatbot); err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to update chatbot")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update chatbot",
		})
	}

	return c.JSON(chatbot)
}

// ============================================================================
// PROVIDER ENDPOINTS
// ============================================================================

// ListProviders returns all AI providers
// GET /api/v1/admin/ai/providers
func (h *Handler) ListProviders(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	providers, err := h.storage.ListProviders(ctx, false)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list providers")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list providers",
		})
	}

	// Filter out config-based (FROM_CONFIG) providers for non-default tenants.
	// Instance-level YAML/env config must not leak to non-default tenants.
	isDefaultTenant, _ := c.Locals("is_default_tenant").(bool)
	if !isDefaultTenant {
		filtered := make([]*ProviderRecord, 0, len(providers))
		for _, p := range providers {
			if p.ID != "FROM_CONFIG" {
				filtered = append(filtered, p)
			}
		}
		providers = filtered
	}

	// Remove sensitive config for API response
	for _, p := range providers {
		if p.Config != nil {
			// Mask API key
			if _, ok := p.Config["api_key"]; ok {
				p.Config["api_key"] = "***masked***"
			}
		}
	}

	return c.JSON(fiber.Map{
		"providers": providers,
		"count":     len(providers),
	})
}

// GetProvider returns a single provider by ID
// GET /api/v1/admin/ai/providers/:id
func (h *Handler) GetProvider(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	provider, err := h.storage.GetProvider(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get provider")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get provider",
		})
	}

	if provider == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}

	// Mask API key
	if provider.Config != nil {
		if _, ok := provider.Config["api_key"]; ok {
			provider.Config["api_key"] = "***masked***"
		}
	}

	return c.JSON(provider)
}

// CreateProviderRequest represents the request to create a provider
type CreateProviderRequest struct {
	Name           string         `json:"name"`
	DisplayName    string         `json:"display_name"`
	ProviderType   string         `json:"provider_type"`
	IsDefault      bool           `json:"is_default"`
	EmbeddingModel *string        `json:"embedding_model"`
	Config         map[string]any `json:"config"`
	Enabled        bool           `json:"enabled"`
}

// normalizeConfig converts any config values to strings and removes empty/invalid values
// This allows the API to accept numbers, booleans, etc. while storing as strings
func normalizeConfig(config map[string]any) map[string]string {
	if config == nil {
		return make(map[string]string)
	}
	normalized := make(map[string]string, len(config))
	for k, v := range config {
		if v == nil {
			continue
		}
		str := fmt.Sprintf("%v", v)
		// Skip empty values and string representations of undefined/null
		if str == "" || str == "undefined" || str == "null" {
			continue
		}
		normalized[k] = str
	}
	return normalized
}

// CreateProvider creates a new AI provider
// POST /api/v1/admin/ai/providers
func (h *Handler) CreateProvider(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	var req CreateProviderRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Normalize config to convert values to strings and remove empty/invalid values
	normalizedConfig := normalizeConfig(req.Config)

	// Validate provider type
	if req.ProviderType != "openai" && req.ProviderType != "azure" && req.ProviderType != "ollama" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid provider type. Must be 'openai', 'azure', or 'ollama'",
		})
	}

	// Check if there's an existing default provider
	existingDefault, err := h.storage.GetDefaultProvider(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to check for existing default provider")
	}

	// Auto-set as default if no default provider exists
	isDefault := req.IsDefault
	if existingDefault == nil {
		isDefault = true
		log.Info().Msg("No default AI provider exists, setting new provider as default")
	}

	provider := &ProviderRecord{
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		ProviderType:   req.ProviderType,
		IsDefault:      isDefault,
		EmbeddingModel: req.EmbeddingModel,
		Config:         normalizedConfig,
		Enabled:        true, // Always enable new providers
	}

	if err := h.storage.CreateProvider(ctx, provider); err != nil {
		log.Error().Err(err).Str("name", req.Name).Msg("Failed to create provider")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create provider",
		})
	}

	// Reload embedding service from database providers
	if h.vectorManager != nil {
		if err := h.vectorManager.RefreshFromDatabase(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to reload embedding service after provider creation")
		}
	}

	// Mask API key in response
	if provider.Config != nil {
		if _, ok := provider.Config["api_key"]; ok {
			provider.Config["api_key"] = "***masked***"
		}
	}

	return c.Status(fiber.StatusCreated).JSON(provider)
}

// SetDefaultProvider sets a provider as the default
// PUT /api/v1/admin/ai/providers/:id/default
func (h *Handler) SetDefaultProvider(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	// Prevent modifying config-based provider
	if id == "FROM_CONFIG" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot modify config-based provider. This provider is configured via environment variables or fluxbase.yaml and is read-only.",
		})
	}

	if err := h.storage.SetDefaultProvider(ctx, id); err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to set default provider")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to set default provider",
		})
	}

	// Invalidate chat-side cache: chatbots without an explicit ProviderID fall
	// back to the default and need to pick up the new default on next request.
	h.invalidateProvider(id)

	// Reload embedding service from database providers
	if h.vectorManager != nil {
		if err := h.vectorManager.RefreshFromDatabase(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to reload embedding service after setting default provider")
		}
	}

	return c.JSON(fiber.Map{
		"id":        id,
		"isDefault": true,
	})
}

// DeleteProvider deletes a provider
// DELETE /api/v1/admin/ai/providers/:id
func (h *Handler) DeleteProvider(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	// Prevent deleting config-based provider
	if id == "FROM_CONFIG" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot delete config-based provider. This provider is configured via environment variables or fluxbase.yaml and is read-only.",
		})
	}

	if err := h.storage.DeleteProvider(ctx, id); err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to delete provider")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete provider",
		})
	}

	// Drop the deleted provider from the chat-side cache.
	h.invalidateProvider(id)

	// Reload embedding service from database providers
	if h.vectorManager != nil {
		if err := h.vectorManager.RefreshFromDatabase(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to reload embedding service after provider deletion")
		}
	}

	return c.JSON(fiber.Map{
		"deleted": true,
		"id":      id,
	})
}

// SetEmbeddingProvider sets a provider as the embedding provider
// PUT /api/v1/admin/ai/providers/:id/embedding
func (h *Handler) SetEmbeddingProvider(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	// Prevent modifying config-based provider
	if id == "FROM_CONFIG" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot modify config-based provider. This provider is configured via environment variables or fluxbase.yaml and is read-only.",
		})
	}

	// Set embedding provider preference
	if err := h.storage.SetEmbeddingProviderPreference(ctx, id); err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to set embedding provider")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to set embedding provider",
		})
	}

	// Reload embedding service from database providers
	if h.vectorManager != nil {
		if err := h.vectorManager.RefreshFromDatabase(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to reload embedding service after setting embedding provider")
		}
	}

	return c.JSON(fiber.Map{
		"id":                 id,
		"use_for_embeddings": true,
	})
}

// ClearEmbeddingProvider clears the explicit embedding provider preference
// DELETE /api/v1/admin/ai/providers/:id/embedding
func (h *Handler) ClearEmbeddingProvider(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	// Clear embedding preference (revert to auto/default)
	if err := h.storage.SetEmbeddingProviderPreference(ctx, ""); err != nil {
		log.Error().Err(err).Msg("Failed to clear embedding provider preference")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to clear embedding provider preference",
		})
	}

	// Reload embedding service to use default provider
	if h.vectorManager != nil {
		if err := h.vectorManager.RefreshFromDatabase(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to reload embedding service after clearing embedding provider")
		}
	}

	return c.JSON(fiber.Map{
		"use_for_embeddings": false,
	})
}

// UpdateProviderRequest represents the request to update a provider
type UpdateProviderRequest struct {
	DisplayName    *string        `json:"display_name"`
	Config         map[string]any `json:"config"`
	Enabled        *bool          `json:"enabled"`
	EmbeddingModel *string        `json:"embedding_model"`
}

// UpdateProvider updates an AI provider
// PUT /api/v1/admin/ai/providers/:id
func (h *Handler) UpdateProvider(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	// Prevent modifying config-based provider
	if id == "FROM_CONFIG" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot modify config-based provider. This provider is configured via environment variables or fluxbase.yaml and is read-only.",
		})
	}

	var req UpdateProviderRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get existing provider
	provider, err := h.storage.GetProvider(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get provider")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get provider",
		})
	}

	if provider == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Provider not found",
		})
	}

	// Apply updates
	if req.DisplayName != nil {
		provider.DisplayName = *req.DisplayName
	}
	if req.Config != nil {
		// Normalize and merge config - only update fields that are provided
		normalizedConfig := normalizeConfig(req.Config)
		if provider.Config == nil {
			provider.Config = make(map[string]string)
		}
		for k, v := range normalizedConfig {
			// Skip masked api_key - keep existing value
			if k == "api_key" && v == "***masked***" {
				continue
			}
			provider.Config[k] = v
		}
	}
	if req.Enabled != nil {
		provider.Enabled = *req.Enabled
	}
	if req.EmbeddingModel != nil {
		// Allow setting to empty string to reset to default
		if *req.EmbeddingModel == "" {
			provider.EmbeddingModel = nil
		} else {
			provider.EmbeddingModel = req.EmbeddingModel
		}
	}

	if err := h.storage.UpdateProvider(ctx, provider); err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to update provider")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update provider",
		})
	}

	// Invalidate chat-side cache so the new config (API key, model, etc.) takes
	// effect immediately rather than on next process restart.
	h.invalidateProvider(id)

	// Reload embedding service from database providers
	if h.vectorManager != nil {
		if err := h.vectorManager.RefreshFromDatabase(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to reload embedding service after provider update")
		}
	}

	// Mask API key in response
	if provider.Config != nil {
		if _, ok := provider.Config["api_key"]; ok {
			provider.Config["api_key"] = "***masked***"
		}
	}

	return c.JSON(provider)
}

// ============================================================================
// PUBLIC CHATBOT ENDPOINTS
// ============================================================================

// ListPublicChatbots returns all public, enabled chatbots for users
// GET /api/v1/ai/chatbots
func (h *Handler) ListPublicChatbots(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	chatbots, err := h.storage.ListChatbots(ctx, true)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list chatbots")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list chatbots",
		})
	}

	// Filter to only public chatbots
	var publicChatbots []ChatbotSummary
	for _, cb := range chatbots {
		if cb.IsPublic {
			publicChatbots = append(publicChatbots, cb.ToSummary())
		}
	}

	return c.JSON(fiber.Map{
		"chatbots": publicChatbots,
		"count":    len(publicChatbots),
	})
}

// GetPublicChatbot returns a single public chatbot by ID
// GET /api/v1/ai/chatbots/:id
func (h *Handler) GetPublicChatbot(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")

	chatbot, err := h.storage.GetChatbot(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get chatbot")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get chatbot",
		})
	}

	if chatbot == nil || !chatbot.Enabled || !chatbot.IsPublic {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Chatbot not found",
		})
	}

	return c.JSON(chatbot.ToSummary())
}

// LookupChatbotByNameResponse represents the response for chatbot lookup by name
type LookupChatbotByNameResponse struct {
	Chatbot    *ChatbotSummary `json:"chatbot,omitempty"`
	Ambiguous  bool            `json:"ambiguous"`
	Namespaces []string        `json:"namespaces,omitempty"`
	Error      string          `json:"error,omitempty"`
}

// LookupChatbotByName finds a chatbot by name with smart namespace resolution
// GET /api/v1/ai/chatbots/by-name/:name
//
// Resolution logic:
// 1. Find all enabled, public chatbots with the given name
// 2. If exactly one match -> return it
// 3. If multiple matches -> try "default" namespace first
// 4. If multiple matches and none in "default" -> return 409 Conflict with namespace list
func (h *Handler) LookupChatbotByName(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	name := c.Params("name")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(LookupChatbotByNameResponse{
			Ambiguous: false,
			Error:     "Chatbot name is required",
		})
	}

	// Find all chatbots with this name (enabled only)
	chatbots, err := h.storage.FindChatbotsByName(ctx, name, true)
	if err != nil {
		log.Error().Err(err).Str("name", name).Msg("Failed to lookup chatbot by name")
		return c.Status(fiber.StatusInternalServerError).JSON(LookupChatbotByNameResponse{
			Ambiguous: false,
			Error:     "Failed to lookup chatbot",
		})
	}

	// Filter to only public chatbots
	var publicChatbots []*Chatbot
	for _, cb := range chatbots {
		if cb.IsPublic {
			publicChatbots = append(publicChatbots, cb)
		}
	}

	// No matches
	if len(publicChatbots) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(LookupChatbotByNameResponse{
			Ambiguous: false,
			Error:     "Chatbot not found",
		})
	}

	// Exactly one match - return it
	if len(publicChatbots) == 1 {
		summary := publicChatbots[0].ToSummary()
		return c.JSON(LookupChatbotByNameResponse{
			Chatbot:   &summary,
			Ambiguous: false,
		})
	}

	// Multiple matches - check if one is in "default" namespace
	for _, cb := range publicChatbots {
		if cb.Namespace == "default" {
			summary := cb.ToSummary()
			return c.JSON(LookupChatbotByNameResponse{
				Chatbot:   &summary,
				Ambiguous: false,
			})
		}
	}

	// Multiple matches, none in default - return ambiguous
	namespaces := make([]string, len(publicChatbots))
	for i, cb := range publicChatbots {
		namespaces[i] = cb.Namespace
	}

	return c.Status(fiber.StatusConflict).JSON(LookupChatbotByNameResponse{
		Ambiguous:  true,
		Namespaces: namespaces,
		Error:      fmt.Sprintf("Chatbot '%s' exists in multiple namespaces: %v. Please specify the namespace explicitly.", name, namespaces),
	})
}
