package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Handler handles AI-related HTTP endpoints
type Handler struct {
	storage *Storage
	loader  *Loader
	config  *config.AIConfig
}

// NewHandler creates a new AI handler
func NewHandler(storage *Storage, loader *Loader, cfg *config.AIConfig) *Handler {
	h := &Handler{
		storage: storage,
		loader:  loader,
		config:  cfg,
	}

	// Validate config at startup
	h.ValidateConfig()

	return h
}

// ValidateConfig checks AI configuration and logs any issues at startup
func (h *Handler) ValidateConfig() {
	if h.config == nil || !h.config.ProviderEnabled {
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
func (h *Handler) ListChatbots(c *fiber.Ctx) error {
	ctx := c.Context()

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
func (h *Handler) GetChatbot(c *fiber.Ctx) error {
	ctx := c.Context()
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

// SyncChatbotsRequest represents the request body for syncing chatbots
type SyncChatbotsRequest struct {
	Namespace string `json:"namespace"`
	Chatbots  []struct {
		Name string `json:"name"`
		Code string `json:"code"`
	} `json:"chatbots"`
	Options struct {
		DeleteMissing bool `json:"delete_missing"`
		DryRun        bool `json:"dry_run"`
	} `json:"options"`
}

// SyncChatbots syncs chatbots from filesystem or SDK payload
// POST /api/v1/admin/ai/chatbots/sync
// If chatbots array is empty, syncs from filesystem. Otherwise syncs provided chatbots.
func (h *Handler) SyncChatbots(c *fiber.Ctx) error {
	var req SyncChatbotsRequest
	_ = c.BodyParser(&req) // Body is optional, continue with defaults

	// Default namespace to "default" if not specified
	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// If no chatbots provided, fall back to filesystem sync
	if len(req.Chatbots) == 0 {
		return h.syncFromFilesystem(c, namespace)
	}

	// Sync from SDK payload
	return h.syncFromPayload(c, namespace, req.Chatbots, req.Options.DeleteMissing, req.Options.DryRun)
}

// syncFromFilesystem syncs chatbots from the filesystem
// All chatbots are synced to the specified namespace (default: "default")
// Any existing chatbot in that namespace not found in the filesystem will be deleted
func (h *Handler) syncFromFilesystem(c *fiber.Ctx, namespace string) error {
	ctx := c.Context()

	// Load chatbots from filesystem
	fsChatbots, err := h.loader.LoadAll()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load chatbots from filesystem")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load chatbots from filesystem",
		})
	}

	// Override namespace for all loaded chatbots with the requested namespace
	for _, cb := range fsChatbots {
		cb.Namespace = namespace
	}

	// Get existing chatbots in this namespace only
	dbChatbots, err := h.storage.ListChatbotsByNamespace(ctx, namespace)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list chatbots from database")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list chatbots from database",
		})
	}

	// Build map of existing chatbots by name (within this namespace)
	existingMap := make(map[string]*Chatbot)
	for _, cb := range dbChatbots {
		existingMap[cb.Name] = cb
	}

	// Track sync results
	createdCount := 0
	updatedCount := 0
	deletedCount := 0
	unchangedCount := 0
	syncErrors := []string{}

	// Track created/updated/deleted names for response
	createdNames := []string{}
	updatedNames := []string{}
	deletedNames := []string{}
	unchangedNames := []string{}

	// Track which chatbots we've processed
	processedNames := make(map[string]bool)

	// Create/update chatbots from filesystem
	for _, fsChatbot := range fsChatbots {
		processedNames[fsChatbot.Name] = true

		existing, exists := existingMap[fsChatbot.Name]
		if exists {
			// Check if update is needed
			if existing.Code == fsChatbot.Code {
				// No change, skip
				unchangedCount++
				unchangedNames = append(unchangedNames, fsChatbot.Name)
				continue
			}

			// Update existing chatbot
			fsChatbot.ID = existing.ID
			fsChatbot.CreatedAt = existing.CreatedAt
			fsChatbot.CreatedBy = existing.CreatedBy
			fsChatbot.Version = existing.Version

			if err := h.storage.UpdateChatbot(ctx, fsChatbot); err != nil {
				log.Error().Err(err).Str("name", fsChatbot.Name).Msg("Failed to update chatbot")
				syncErrors = append(syncErrors, "Failed to update "+fsChatbot.Name+": "+err.Error())
				continue
			}
			updatedCount++
			updatedNames = append(updatedNames, fsChatbot.Name)
		} else {
			// Create new chatbot
			if err := h.storage.CreateChatbot(ctx, fsChatbot); err != nil {
				log.Error().Err(err).Str("name", fsChatbot.Name).Msg("Failed to create chatbot")
				syncErrors = append(syncErrors, "Failed to create "+fsChatbot.Name+": "+err.Error())
				continue
			}
			createdCount++
			createdNames = append(createdNames, fsChatbot.Name)
		}
	}

	// Delete chatbots in this namespace that are no longer in the filesystem
	for name, dbChatbot := range existingMap {
		if !processedNames[name] {
			if err := h.storage.DeleteChatbot(ctx, dbChatbot.ID); err != nil {
				log.Error().Err(err).Str("name", dbChatbot.Name).Msg("Failed to delete chatbot")
				syncErrors = append(syncErrors, "Failed to delete "+name+": "+err.Error())
				continue
			}
			deletedCount++
			deletedNames = append(deletedNames, name)
		}
	}

	log.Info().
		Int("created", createdCount).
		Int("updated", updatedCount).
		Int("deleted", deletedCount).
		Int("unchanged", unchangedCount).
		Int("errors", len(syncErrors)).
		Str("namespace", namespace).
		Msg("Synced chatbots from filesystem")

	return c.JSON(fiber.Map{
		"message":   "Chatbots synced from filesystem",
		"namespace": namespace,
		"summary": fiber.Map{
			"created":   createdCount,
			"updated":   updatedCount,
			"deleted":   deletedCount,
			"unchanged": unchangedCount,
			"errors":    len(syncErrors),
		},
		"details": fiber.Map{
			"created":   createdNames,
			"updated":   updatedNames,
			"deleted":   deletedNames,
			"unchanged": unchangedNames,
		},
		"errors":  syncErrors,
		"dry_run": false,
	})
}

// syncFromPayload syncs chatbots from SDK payload
func (h *Handler) syncFromPayload(c *fiber.Ctx, namespace string, chatbots []struct {
	Name string `json:"name"`
	Code string `json:"code"`
}, deleteMissing bool, dryRun bool) error {
	ctx := c.Context()

	// Get existing chatbots in this namespace
	dbChatbots, err := h.storage.ListChatbotsByNamespace(ctx, namespace)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list existing chatbots",
		})
	}

	// Build map of existing chatbots by name
	existingMap := make(map[string]*Chatbot)
	for _, cb := range dbChatbots {
		existingMap[cb.Name] = cb
	}

	// Build set of payload chatbot names
	payloadNames := make(map[string]bool)
	for _, spec := range chatbots {
		payloadNames[spec.Name] = true
	}

	// Track results
	created := []string{}
	updated := []string{}
	deleted := []string{}
	errorList := []fiber.Map{}

	// If dry run, calculate what would be done
	if dryRun {
		for _, spec := range chatbots {
			if _, exists := existingMap[spec.Name]; exists {
				updated = append(updated, spec.Name)
			} else {
				created = append(created, spec.Name)
			}
		}

		if deleteMissing {
			for name := range existingMap {
				if !payloadNames[name] {
					deleted = append(deleted, name)
				}
			}
		}

		return c.JSON(fiber.Map{
			"summary": fiber.Map{
				"created": len(created),
				"updated": len(updated),
				"deleted": len(deleted),
				"errors":  0,
			},
			"details": fiber.Map{
				"created": created,
				"updated": updated,
				"deleted": deleted,
			},
			"dry_run": true,
		})
	}

	// Process chatbots
	for _, spec := range chatbots {
		existing, exists := existingMap[spec.Name]

		// Parse and compile the chatbot code
		chatbot, err := h.loader.ParseChatbotFromCode(spec.Code, namespace)
		if err != nil {
			log.Error().Err(err).Str("name", spec.Name).Msg("Failed to parse chatbot")
			errorList = append(errorList, fiber.Map{
				"name":  spec.Name,
				"error": "Failed to parse chatbot: " + err.Error(),
			})
			continue
		}

		// Set the name and code
		chatbot.Name = spec.Name
		chatbot.Code = spec.Code
		chatbot.Source = "sdk"

		if exists {
			// Update existing chatbot
			chatbot.ID = existing.ID
			chatbot.CreatedAt = existing.CreatedAt
			chatbot.CreatedBy = existing.CreatedBy
			chatbot.Version = existing.Version

			if err := h.storage.UpdateChatbot(ctx, chatbot); err != nil {
				log.Error().Err(err).Str("name", spec.Name).Msg("Failed to update chatbot")
				errorList = append(errorList, fiber.Map{
					"name":  spec.Name,
					"error": "Failed to update: " + err.Error(),
				})
				continue
			}
			updated = append(updated, spec.Name)
		} else {
			// Create new chatbot
			if err := h.storage.CreateChatbot(ctx, chatbot); err != nil {
				log.Error().Err(err).Str("name", spec.Name).Msg("Failed to create chatbot")
				errorList = append(errorList, fiber.Map{
					"name":  spec.Name,
					"error": "Failed to create: " + err.Error(),
				})
				continue
			}
			created = append(created, spec.Name)
		}
	}

	// Delete missing chatbots if requested
	if deleteMissing {
		for name, chatbot := range existingMap {
			if !payloadNames[name] && chatbot.Source == "sdk" {
				if err := h.storage.DeleteChatbot(ctx, chatbot.ID); err != nil {
					log.Error().Err(err).Str("name", name).Msg("Failed to delete chatbot")
					errorList = append(errorList, fiber.Map{
						"name":  name,
						"error": "Failed to delete: " + err.Error(),
					})
					continue
				}
				deleted = append(deleted, name)
			}
		}
	}

	log.Info().
		Int("created", len(created)).
		Int("updated", len(updated)).
		Int("deleted", len(deleted)).
		Int("errors", len(errorList)).
		Str("namespace", namespace).
		Msg("Synced chatbots from SDK payload")

	return c.JSON(fiber.Map{
		"summary": fiber.Map{
			"created": len(created),
			"updated": len(updated),
			"deleted": len(deleted),
			"errors":  len(errorList),
		},
		"details": fiber.Map{
			"created": created,
			"updated": updated,
			"deleted": deleted,
		},
		"errors":  errorList,
		"dry_run": false,
	})
}

// ToggleChatbotRequest represents the request to enable/disable a chatbot
type ToggleChatbotRequest struct {
	Enabled bool `json:"enabled"`
}

// ToggleChatbot enables or disables a chatbot
// PUT /api/v1/admin/ai/chatbots/:id/toggle
func (h *Handler) ToggleChatbot(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	var req ToggleChatbotRequest
	if err := c.BodyParser(&req); err != nil {
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
func (h *Handler) DeleteChatbot(c *fiber.Ctx) error {
	ctx := c.Context()
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
func (h *Handler) UpdateChatbot(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	var req UpdateChatbotRequest
	if err := c.BodyParser(&req); err != nil {
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
func (h *Handler) ListProviders(c *fiber.Ctx) error {
	ctx := c.Context()

	providers, err := h.storage.ListProviders(ctx, false)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list providers")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list providers",
		})
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
func (h *Handler) GetProvider(c *fiber.Ctx) error {
	ctx := c.Context()
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
	Name         string            `json:"name"`
	DisplayName  string            `json:"display_name"`
	ProviderType string            `json:"provider_type"`
	IsDefault    bool              `json:"is_default"`
	Config       map[string]string `json:"config"`
	Enabled      bool              `json:"enabled"`
}

// sanitizeConfig removes empty, "undefined", and "null" string values from config
func sanitizeConfig(config map[string]string) map[string]string {
	if config == nil {
		return make(map[string]string)
	}
	sanitized := make(map[string]string, len(config))
	for k, v := range config {
		// Skip empty values and string representations of undefined/null
		if v == "" || v == "undefined" || v == "null" {
			continue
		}
		sanitized[k] = v
	}
	return sanitized
}

// CreateProvider creates a new AI provider
// POST /api/v1/admin/ai/providers
func (h *Handler) CreateProvider(c *fiber.Ctx) error {
	ctx := c.Context()

	var req CreateProviderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Sanitize config to remove empty/invalid values
	req.Config = sanitizeConfig(req.Config)

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
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		ProviderType: req.ProviderType,
		IsDefault:    isDefault,
		Config:       req.Config,
		Enabled:      true, // Always enable new providers
	}

	if err := h.storage.CreateProvider(ctx, provider); err != nil {
		log.Error().Err(err).Str("name", req.Name).Msg("Failed to create provider")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create provider",
		})
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
func (h *Handler) SetDefaultProvider(c *fiber.Ctx) error {
	ctx := c.Context()
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

	return c.JSON(fiber.Map{
		"id":        id,
		"isDefault": true,
	})
}

// DeleteProvider deletes a provider
// DELETE /api/v1/admin/ai/providers/:id
func (h *Handler) DeleteProvider(c *fiber.Ctx) error {
	ctx := c.Context()
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

	return c.JSON(fiber.Map{
		"deleted": true,
		"id":      id,
	})
}

// UpdateProviderRequest represents the request to update a provider
type UpdateProviderRequest struct {
	DisplayName *string           `json:"display_name"`
	Config      map[string]string `json:"config"`
	Enabled     *bool             `json:"enabled"`
}

// UpdateProvider updates an AI provider
// PUT /api/v1/admin/ai/providers/:id
func (h *Handler) UpdateProvider(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	// Prevent modifying config-based provider
	if id == "FROM_CONFIG" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot modify config-based provider. This provider is configured via environment variables or fluxbase.yaml and is read-only.",
		})
	}

	var req UpdateProviderRequest
	if err := c.BodyParser(&req); err != nil {
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
		// Sanitize and merge config - only update fields that are provided
		sanitizedConfig := sanitizeConfig(req.Config)
		if provider.Config == nil {
			provider.Config = make(map[string]string)
		}
		for k, v := range sanitizedConfig {
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

	if err := h.storage.UpdateProvider(ctx, provider); err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to update provider")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update provider",
		})
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
func (h *Handler) ListPublicChatbots(c *fiber.Ctx) error {
	ctx := c.Context()

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

// GetPublicChatbot returns a single public chatbot by name
// GET /api/v1/ai/chatbots/:namespace/:name
func (h *Handler) GetPublicChatbot(c *fiber.Ctx) error {
	ctx := c.Context()
	namespace := c.Params("namespace")
	name := c.Params("name")

	chatbot, err := h.storage.GetChatbotByName(ctx, namespace, name)
	if err != nil {
		log.Error().Err(err).Str("namespace", namespace).Str("name", name).Msg("Failed to get chatbot")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get chatbot",
		})
	}

	if chatbot == nil || !chatbot.Enabled || !chatbot.IsPublic {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Chatbot not found",
		})
	}

	// Return only public information
	return c.JSON(chatbot.ToSummary())
}

// ============================================================================
// METRICS ENDPOINT
// ============================================================================

// ChatbotMetric represents metrics for a single chatbot
type ChatbotMetric struct {
	ChatbotID   string `json:"chatbot_id"`
	ChatbotName string `json:"chatbot_name"`
	Requests    int64  `json:"requests"`
	Tokens      int64  `json:"tokens"`
	ErrorCount  int64  `json:"error_count"`
}

// ProviderMetric represents metrics for a single provider
type ProviderMetric struct {
	ProviderID   string  `json:"provider_id"`
	ProviderName string  `json:"provider_name"`
	Requests     int64   `json:"requests"`
	AvgLatencyMS float64 `json:"avg_latency_ms"`
}

// AIMetrics represents aggregated AI metrics
type AIMetrics struct {
	TotalRequests         int64            `json:"total_requests"`
	TotalTokens           int64            `json:"total_tokens"`
	TotalPromptTokens     int64            `json:"total_prompt_tokens"`
	TotalCompletionTokens int64            `json:"total_completion_tokens"`
	ActiveConversations   int              `json:"active_conversations"`
	TotalConversations    int              `json:"total_conversations"`
	ChatbotStats          []ChatbotMetric  `json:"chatbot_stats"`
	ProviderStats         []ProviderMetric `json:"provider_stats"`
	ErrorRate             float64          `json:"error_rate"`
	AvgResponseTimeMS     float64          `json:"avg_response_time_ms"`
}

// GetAIMetrics returns aggregated AI metrics
// GET /api/v1/admin/ai/metrics
func (h *Handler) GetAIMetrics(c *fiber.Ctx) error {
	ctx := c.Context()

	metrics := AIMetrics{
		ChatbotStats:  make([]ChatbotMetric, 0),
		ProviderStats: make([]ProviderMetric, 0),
	}

	// Query conversation metrics
	convQuery := `
		SELECT
			COUNT(*) as total_conversations,
			COUNT(*) FILTER (WHERE status = 'active') as active_conversations,
			COALESCE(SUM(total_prompt_tokens), 0) as total_prompt_tokens,
			COALESCE(SUM(total_completion_tokens), 0) as total_completion_tokens
		FROM ai.conversations
	`
	err := h.storage.db.QueryRow(ctx, convQuery).Scan(
		&metrics.TotalConversations,
		&metrics.ActiveConversations,
		&metrics.TotalPromptTokens,
		&metrics.TotalCompletionTokens,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query conversation metrics")
	}

	metrics.TotalTokens = metrics.TotalPromptTokens + metrics.TotalCompletionTokens

	// Query audit log for request counts and error rates
	auditQuery := `
		SELECT
			COUNT(*) as total_requests,
			COUNT(*) FILTER (WHERE success = false) as error_count,
			COALESCE(AVG(execution_duration_ms), 0) as avg_duration
		FROM ai.query_audit_log
		WHERE executed = true
	`
	var errorCount int64
	err = h.storage.db.QueryRow(ctx, auditQuery).Scan(
		&metrics.TotalRequests,
		&errorCount,
		&metrics.AvgResponseTimeMS,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query audit log metrics")
	}

	if metrics.TotalRequests > 0 {
		metrics.ErrorRate = float64(errorCount) / float64(metrics.TotalRequests) * 100
	}

	// Query per-chatbot metrics
	chatbotQuery := `
		SELECT
			c.id,
			c.name,
			COUNT(a.id) as requests,
			COALESCE(SUM(conv.total_prompt_tokens + conv.total_completion_tokens), 0) as tokens,
			COUNT(a.id) FILTER (WHERE a.success = false) as error_count
		FROM ai.chatbots c
		LEFT JOIN ai.query_audit_log a ON a.chatbot_id = c.id
		LEFT JOIN ai.conversations conv ON conv.chatbot_id = c.id
		GROUP BY c.id, c.name
		HAVING COUNT(a.id) > 0
		ORDER BY requests DESC
		LIMIT 20
	`
	rows, err := h.storage.db.Query(ctx, chatbotQuery)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query chatbot metrics")
	} else {
		defer rows.Close()
		for rows.Next() {
			var metric ChatbotMetric
			err := rows.Scan(
				&metric.ChatbotID,
				&metric.ChatbotName,
				&metric.Requests,
				&metric.Tokens,
				&metric.ErrorCount,
			)
			if err != nil {
				log.Error().Err(err).Msg("Failed to scan chatbot metric")
				continue
			}
			metrics.ChatbotStats = append(metrics.ChatbotStats, metric)
		}
	}

	return c.JSON(metrics)
}

// ConversationSummary represents a conversation with basic info
type ConversationSummary struct {
	ID                    string     `json:"id"`
	ChatbotID             string     `json:"chatbot_id"`
	ChatbotName           string     `json:"chatbot_name"`
	UserID                *string    `json:"user_id"`
	UserEmail             *string    `json:"user_email"`
	SessionID             *string    `json:"session_id"`
	Title                 *string    `json:"title"`
	Status                string     `json:"status"`
	TurnCount             int        `json:"turn_count"`
	TotalPromptTokens     int        `json:"total_prompt_tokens"`
	TotalCompletionTokens int        `json:"total_completion_tokens"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	LastMessageAt         *time.Time `json:"last_message_at"`
}

// GetConversations returns a list of AI conversations with optional filters
// GET /api/v1/admin/ai/conversations?chatbot_id=X&user_id=Y&status=active&limit=50
func (h *Handler) GetConversations(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse query parameters
	chatbotID := c.Query("chatbot_id")
	userID := c.Query("user_id")
	status := c.Query("status")
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	// Build query
	query := `
		SELECT
			c.id,
			c.chatbot_id,
			cb.name as chatbot_name,
			c.user_id,
			u.email as user_email,
			c.session_id,
			c.title,
			c.status,
			c.turn_count,
			c.total_prompt_tokens,
			c.total_completion_tokens,
			c.created_at,
			c.updated_at,
			c.last_message_at
		FROM ai.conversations c
		LEFT JOIN ai.chatbots cb ON cb.id = c.chatbot_id
		LEFT JOIN auth.users u ON u.id = c.user_id
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	if chatbotID != "" {
		query += fmt.Sprintf(" AND c.chatbot_id = $%d", argIndex)
		args = append(args, chatbotID)
		argIndex++
	}

	if userID != "" {
		query += fmt.Sprintf(" AND c.user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	if status != "" {
		query += fmt.Sprintf(" AND c.status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	// Build count query with same filters (without LIMIT/OFFSET)
	countQuery := `
		SELECT COUNT(*)
		FROM ai.conversations c
		WHERE 1=1
	`
	countArgs := []interface{}{}
	countArgIndex := 1

	if chatbotID != "" {
		countQuery += fmt.Sprintf(" AND c.chatbot_id = $%d", countArgIndex)
		countArgs = append(countArgs, chatbotID)
		countArgIndex++
	}
	if userID != "" {
		countQuery += fmt.Sprintf(" AND c.user_id = $%d", countArgIndex)
		countArgs = append(countArgs, userID)
		countArgIndex++
	}
	if status != "" {
		countQuery += fmt.Sprintf(" AND c.status = $%d", countArgIndex)
		countArgs = append(countArgs, status)
	}

	var totalCount int
	if err := h.storage.db.QueryRow(ctx, countQuery, countArgs...).Scan(&totalCount); err != nil {
		log.Error().Err(err).Msg("Failed to count conversations")
		totalCount = 0
	}

	query += fmt.Sprintf(" ORDER BY c.last_message_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := h.storage.db.Query(ctx, query, args...)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query conversations")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to query conversations",
		})
	}
	defer rows.Close()

	conversations := make([]ConversationSummary, 0)
	for rows.Next() {
		var conv ConversationSummary
		err := rows.Scan(
			&conv.ID,
			&conv.ChatbotID,
			&conv.ChatbotName,
			&conv.UserID,
			&conv.UserEmail,
			&conv.SessionID,
			&conv.Title,
			&conv.Status,
			&conv.TurnCount,
			&conv.TotalPromptTokens,
			&conv.TotalCompletionTokens,
			&conv.CreatedAt,
			&conv.UpdatedAt,
			&conv.LastMessageAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan conversation")
			continue
		}
		conversations = append(conversations, conv)
	}

	return c.JSON(fiber.Map{
		"conversations": conversations,
		"total":         len(conversations),
		"total_count":   totalCount,
	})
}

// MessageDetail represents a message within a conversation
type MessageDetail struct {
	ID               string    `json:"id"`
	ConversationID   string    `json:"conversation_id"`
	Role             string    `json:"role"`
	Content          string    `json:"content"`
	ToolCallID       *string   `json:"tool_call_id"`
	ToolName         *string   `json:"tool_name"`
	ExecutedSQL      *string   `json:"executed_sql"`
	SQLResultSummary *string   `json:"sql_result_summary"`
	SQLRowCount      *int      `json:"sql_row_count"`
	SQLError         *string   `json:"sql_error"`
	SQLDurationMS    *int      `json:"sql_duration_ms"`
	PromptTokens     *int      `json:"prompt_tokens"`
	CompletionTokens *int      `json:"completion_tokens"`
	CreatedAt        time.Time `json:"created_at"`
	SequenceNumber   int       `json:"sequence_number"`
}

// GetConversationMessages returns all messages for a specific conversation
// GET /api/v1/admin/ai/conversations/:id/messages
func (h *Handler) GetConversationMessages(c *fiber.Ctx) error {
	ctx := c.Context()
	conversationID := c.Params("id")

	query := `
		SELECT
			id,
			conversation_id,
			role,
			content,
			tool_call_id,
			tool_name,
			executed_sql,
			sql_result_summary,
			sql_row_count,
			sql_error,
			sql_duration_ms,
			prompt_tokens,
			completion_tokens,
			created_at,
			sequence_number
		FROM ai.messages
		WHERE conversation_id = $1
		ORDER BY sequence_number ASC
	`

	rows, err := h.storage.db.Query(ctx, query, conversationID)
	if err != nil {
		log.Error().Err(err).Str("conversation_id", conversationID).Msg("Failed to query messages")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to query messages",
		})
	}
	defer rows.Close()

	messages := make([]MessageDetail, 0)
	for rows.Next() {
		var msg MessageDetail
		err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.Role,
			&msg.Content,
			&msg.ToolCallID,
			&msg.ToolName,
			&msg.ExecutedSQL,
			&msg.SQLResultSummary,
			&msg.SQLRowCount,
			&msg.SQLError,
			&msg.SQLDurationMS,
			&msg.PromptTokens,
			&msg.CompletionTokens,
			&msg.CreatedAt,
			&msg.SequenceNumber,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan message")
			continue
		}
		messages = append(messages, msg)
	}

	return c.JSON(fiber.Map{
		"messages": messages,
		"total":    len(messages),
	})
}

// AuditLogEntry represents an audit log entry
type AuditLogEntry struct {
	ID                  string    `json:"id"`
	ChatbotID           *string   `json:"chatbot_id"`
	ChatbotName         *string   `json:"chatbot_name"`
	ConversationID      *string   `json:"conversation_id"`
	MessageID           *string   `json:"message_id"`
	UserID              *string   `json:"user_id"`
	UserEmail           *string   `json:"user_email"`
	GeneratedSQL        string    `json:"generated_sql"`
	SanitizedSQL        *string   `json:"sanitized_sql"`
	Executed            bool      `json:"executed"`
	ValidationPassed    *bool     `json:"validation_passed"`
	ValidationErrors    []string  `json:"validation_errors"`
	Success             *bool     `json:"success"`
	ErrorMessage        *string   `json:"error_message"`
	RowsReturned        *int      `json:"rows_returned"`
	ExecutionDurationMS *int      `json:"execution_duration_ms"`
	TablesAccessed      []string  `json:"tables_accessed"`
	OperationsUsed      []string  `json:"operations_used"`
	IPAddress           *string   `json:"ip_address"`
	UserAgent           *string   `json:"user_agent"`
	CreatedAt           time.Time `json:"created_at"`
}

// GetAuditLog returns audit log entries with optional filters
// GET /api/v1/admin/ai/audit?chatbot_id=X&user_id=Y&success=true&limit=100
func (h *Handler) GetAuditLog(c *fiber.Ctx) error {
	ctx := c.Context()

	// Parse query parameters
	chatbotID := c.Query("chatbot_id")
	userID := c.Query("user_id")
	successStr := c.Query("success")
	limit := c.QueryInt("limit", 100)
	offset := c.QueryInt("offset", 0)

	// Build query
	query := `
		SELECT
			a.id,
			a.chatbot_id,
			cb.name as chatbot_name,
			a.conversation_id,
			a.message_id,
			a.user_id,
			u.email as user_email,
			a.generated_sql,
			a.sanitized_sql,
			a.executed,
			a.validation_passed,
			a.validation_errors,
			a.success,
			a.error_message,
			a.rows_returned,
			a.execution_duration_ms,
			a.tables_accessed,
			a.operations_used,
			a.ip_address,
			a.user_agent,
			a.created_at
		FROM ai.query_audit_log a
		LEFT JOIN ai.chatbots cb ON cb.id = a.chatbot_id
		LEFT JOIN auth.users u ON u.id = a.user_id
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	if chatbotID != "" {
		query += fmt.Sprintf(" AND a.chatbot_id = $%d", argIndex)
		args = append(args, chatbotID)
		argIndex++
	}

	if userID != "" {
		query += fmt.Sprintf(" AND a.user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	if successStr != "" {
		success := successStr == "true"
		query += fmt.Sprintf(" AND a.success = $%d", argIndex)
		args = append(args, success)
		argIndex++
	}

	// Build count query with same filters (without LIMIT/OFFSET)
	countQuery := `
		SELECT COUNT(*)
		FROM ai.query_audit_log a
		WHERE 1=1
	`
	countArgs := []interface{}{}
	countArgIndex := 1

	if chatbotID != "" {
		countQuery += fmt.Sprintf(" AND a.chatbot_id = $%d", countArgIndex)
		countArgs = append(countArgs, chatbotID)
		countArgIndex++
	}
	if userID != "" {
		countQuery += fmt.Sprintf(" AND a.user_id = $%d", countArgIndex)
		countArgs = append(countArgs, userID)
		countArgIndex++
	}
	if successStr != "" {
		success := successStr == "true"
		countQuery += fmt.Sprintf(" AND a.success = $%d", countArgIndex)
		countArgs = append(countArgs, success)
	}

	var totalCount int
	if err := h.storage.db.QueryRow(ctx, countQuery, countArgs...).Scan(&totalCount); err != nil {
		log.Error().Err(err).Msg("Failed to count audit log entries")
		totalCount = 0
	}

	query += fmt.Sprintf(" ORDER BY a.created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := h.storage.db.Query(ctx, query, args...)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query audit log")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to query audit log",
		})
	}
	defer rows.Close()

	entries := make([]AuditLogEntry, 0)
	for rows.Next() {
		var entry AuditLogEntry
		err := rows.Scan(
			&entry.ID,
			&entry.ChatbotID,
			&entry.ChatbotName,
			&entry.ConversationID,
			&entry.MessageID,
			&entry.UserID,
			&entry.UserEmail,
			&entry.GeneratedSQL,
			&entry.SanitizedSQL,
			&entry.Executed,
			&entry.ValidationPassed,
			&entry.ValidationErrors,
			&entry.Success,
			&entry.ErrorMessage,
			&entry.RowsReturned,
			&entry.ExecutionDurationMS,
			&entry.TablesAccessed,
			&entry.OperationsUsed,
			&entry.IPAddress,
			&entry.UserAgent,
			&entry.CreatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan audit log entry")
			continue
		}
		entries = append(entries, entry)
	}

	return c.JSON(fiber.Map{
		"entries":     entries,
		"total":       len(entries),
		"total_count": totalCount,
	})
}

// ============================================================================
// AUTO-LOAD HELPER
// ============================================================================

// AutoLoadChatbots loads chatbots from filesystem to database on startup
func (h *Handler) AutoLoadChatbots(ctx context.Context) error {
	if !h.config.AutoLoadOnBoot {
		log.Info().Msg("Auto-load chatbots disabled, skipping")
		return nil
	}

	log.Info().Str("dir", h.config.ChatbotsDir).Msg("Auto-loading chatbots from filesystem")

	// Load from filesystem
	chatbots, err := h.loader.LoadAll()
	if err != nil {
		return err
	}

	if len(chatbots) == 0 {
		log.Info().Msg("No chatbots found in filesystem")
		return nil
	}

	// Upsert each chatbot
	created, updated := 0, 0
	for _, cb := range chatbots {
		existing, err := h.storage.GetChatbotByName(ctx, cb.Namespace, cb.Name)
		if err != nil {
			log.Error().Err(err).Str("name", cb.Name).Msg("Failed to check existing chatbot")
			continue
		}

		if existing != nil {
			cb.ID = existing.ID
			cb.CreatedAt = existing.CreatedAt
			cb.CreatedBy = existing.CreatedBy
			if err := h.storage.UpdateChatbot(ctx, cb); err != nil {
				log.Error().Err(err).Str("name", cb.Name).Msg("Failed to update chatbot")
				continue
			}
			updated++
		} else {
			if err := h.storage.CreateChatbot(ctx, cb); err != nil {
				log.Error().Err(err).Str("name", cb.Name).Msg("Failed to create chatbot")
				continue
			}
			created++
		}
	}

	log.Info().
		Int("created", created).
		Int("updated", updated).
		Int("total", len(chatbots)).
		Msg("Auto-loaded chatbots from filesystem")

	return nil
}

// ============================================================================
// USER CONVERSATION ENDPOINTS
// ============================================================================

// UpdateConversationTitleRequest represents the request body for updating title
type UpdateConversationTitleRequest struct {
	Title string `json:"title"`
}

// ListUserConversations lists the authenticated user's conversations
// GET /api/v1/ai/conversations
func (h *Handler) ListUserConversations(c *fiber.Ctx) error {
	ctx := c.Context()

	// Get authenticated user ID from context
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Parse query params
	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100 // Cap at 100
	}
	if limit < 1 {
		limit = 50
	}
	offset := c.QueryInt("offset", 0)
	if offset < 0 {
		offset = 0
	}

	// Build options
	opts := ListUserConversationsOptions{
		UserID: userIDStr,
		Limit:  limit,
		Offset: offset,
	}

	if chatbot := c.Query("chatbot"); chatbot != "" {
		opts.ChatbotName = &chatbot
	}
	if namespace := c.Query("namespace"); namespace != "" {
		opts.Namespace = &namespace
	}

	// Query conversations
	result, err := h.storage.ListUserConversations(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list user conversations")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list conversations",
		})
	}

	return c.JSON(result)
}

// GetUserConversation retrieves a single conversation with messages
// GET /api/v1/ai/conversations/:id
func (h *Handler) GetUserConversation(c *fiber.Ctx) error {
	ctx := c.Context()
	conversationID := c.Params("id")

	// Get authenticated user ID from context
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	conversation, err := h.storage.GetUserConversation(ctx, userIDStr, conversationID)
	if err != nil {
		log.Error().Err(err).Str("id", conversationID).Msg("Failed to get conversation")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get conversation",
		})
	}

	if conversation == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Conversation not found",
		})
	}

	return c.JSON(conversation)
}

// DeleteUserConversation deletes a user's conversation
// DELETE /api/v1/ai/conversations/:id
func (h *Handler) DeleteUserConversation(c *fiber.Ctx) error {
	ctx := c.Context()
	conversationID := c.Params("id")

	// Get authenticated user ID from context
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	err := h.storage.DeleteUserConversation(ctx, userIDStr, conversationID)
	if err != nil {
		if err.Error() == "conversation not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Conversation not found",
			})
		}
		log.Error().Err(err).Str("id", conversationID).Msg("Failed to delete conversation")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete conversation",
		})
	}

	return c.JSON(fiber.Map{
		"deleted": true,
		"id":      conversationID,
	})
}

// UpdateUserConversation updates a conversation (title only for now)
// PATCH /api/v1/ai/conversations/:id
func (h *Handler) UpdateUserConversation(c *fiber.Ctx) error {
	ctx := c.Context()
	conversationID := c.Params("id")

	// Get authenticated user ID from context
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var req UpdateConversationTitleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate title
	if req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Title cannot be empty",
		})
	}
	if len(req.Title) > 200 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Title must be 200 characters or less",
		})
	}

	err := h.storage.UpdateConversationTitle(ctx, userIDStr, conversationID, req.Title)
	if err != nil {
		if err.Error() == "conversation not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Conversation not found",
			})
		}
		log.Error().Err(err).Str("id", conversationID).Msg("Failed to update conversation")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update conversation",
		})
	}

	// Return updated conversation
	conversation, err := h.storage.GetUserConversation(ctx, userIDStr, conversationID)
	if err != nil {
		log.Error().Err(err).Str("id", conversationID).Msg("Failed to get updated conversation")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Conversation updated but failed to retrieve",
		})
	}

	return c.JSON(conversation)
}
