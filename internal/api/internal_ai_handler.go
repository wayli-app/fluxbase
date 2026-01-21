package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/ai"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// InternalAIHandler handles AI requests from custom MCP tools, edge functions, and jobs.
type InternalAIHandler struct {
	aiStorage        *ai.Storage
	embeddingService *ai.EmbeddingService
	defaultProvider  string
}

// NewInternalAIHandler creates a new InternalAIHandler.
func NewInternalAIHandler(aiStorage *ai.Storage, embeddingService *ai.EmbeddingService, defaultProvider string) *InternalAIHandler {
	return &InternalAIHandler{
		aiStorage:        aiStorage,
		embeddingService: embeddingService,
		defaultProvider:  defaultProvider,
	}
}

// InternalChatRequest represents a chat completion request.
type InternalChatRequest struct {
	Messages    []InternalChatMessage `json:"messages"`
	Model       string                `json:"model,omitempty"`
	Provider    string                `json:"provider,omitempty"`
	MaxTokens   int                   `json:"max_tokens,omitempty"`
	Temperature *float64              `json:"temperature,omitempty"`
}

// InternalChatMessage represents a message in the chat.
type InternalChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// InternalChatResponse represents a chat completion response.
type InternalChatResponse struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	FinishReason string `json:"finish_reason,omitempty"`
	Usage        *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// InternalEmbedRequest represents an embedding request.
type InternalEmbedRequest struct {
	Text     string `json:"text"`
	Provider string `json:"provider,omitempty"`
}

// InternalEmbedResponse represents an embedding response.
type InternalEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
	Model     string    `json:"model"`
}

// HandleChat handles POST /api/v1/internal/ai/chat
// This endpoint allows custom MCP tools, edge functions, and jobs to make AI completions.
func (h *InternalAIHandler) HandleChat(c *fiber.Ctx) error {
	if h.aiStorage == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "AI service not configured",
		})
	}

	var req InternalChatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid request body: %v", err),
		})
	}

	if len(req.Messages) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "messages array is required",
		})
	}

	// Get provider - use specified or default
	providerName := req.Provider
	if providerName == "" {
		providerName = h.defaultProvider
	}
	if providerName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No AI provider configured. Set 'provider' in request or configure default provider.",
		})
	}

	// Get the provider from storage
	provider, err := h.aiStorage.GetProviderByName(c.Context(), providerName)
	if err != nil {
		log.Warn().Err(err).Str("provider", providerName).Msg("Failed to get AI provider")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("AI provider '%s' not found", providerName),
		})
	}

	// Build provider config
	providerConfig := ai.ProviderConfig{
		Name:        provider.Name,
		DisplayName: provider.DisplayName,
		Type:        ai.ProviderType(provider.Type),
		Model:       provider.Model,
		Config:      provider.Config,
	}

	// Override model if specified
	if req.Model != "" {
		providerConfig.Model = req.Model
	}

	// Create the provider instance
	aiProvider, err := ai.NewProvider(providerConfig)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to create AI provider")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to initialize AI provider: %v", err),
		})
	}
	defer aiProvider.Close()

	// Convert messages
	messages := make([]ai.Message, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = ai.Message{
			Role:    ai.Role(strings.ToLower(m.Role)),
			Content: m.Content,
		}
	}

	// Set defaults
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	temperature := 0.7
	if req.Temperature != nil {
		temperature = *req.Temperature
	}

	// Make the request
	chatReq := &ai.ChatRequest{
		Model:       providerConfig.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	resp, err := aiProvider.Chat(c.Context(), chatReq)
	if err != nil {
		log.Error().Err(err).Msg("AI chat request failed")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("AI request failed: %v", err),
		})
	}

	if len(resp.Choices) == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "AI returned no response",
		})
	}

	// Build response
	response := InternalChatResponse{
		Content:      resp.Choices[0].Message.Content,
		Model:        resp.Model,
		FinishReason: resp.Choices[0].FinishReason,
	}

	if resp.Usage != nil {
		response.Usage = &struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return c.JSON(response)
}

// HandleEmbed handles POST /api/v1/internal/ai/embed
// This endpoint allows custom MCP tools, edge functions, and jobs to generate embeddings.
func (h *InternalAIHandler) HandleEmbed(c *fiber.Ctx) error {
	if h.embeddingService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Embedding service not configured",
		})
	}

	var req InternalEmbedRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Invalid request body: %v", err),
		})
	}

	if req.Text == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "text is required",
		})
	}

	// Generate embedding
	embedding, err := h.embeddingService.GenerateEmbedding(c.Context(), req.Text)
	if err != nil {
		log.Error().Err(err).Msg("Embedding generation failed")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Embedding generation failed: %v", err),
		})
	}

	// Get model info
	modelInfo := h.embeddingService.GetModelInfo()

	return c.JSON(InternalEmbedResponse{
		Embedding: embedding,
		Model:     modelInfo,
	})
}

// HandleListProviders handles GET /api/v1/internal/ai/providers
// This endpoint lists available AI providers.
func (h *InternalAIHandler) HandleListProviders(c *fiber.Ctx) error {
	if h.aiStorage == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "AI service not configured",
		})
	}

	providers, err := h.aiStorage.ListProviders(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to list providers: %v", err),
		})
	}

	// Return simplified provider info (hide config/API keys)
	result := make([]map[string]any, len(providers))
	for i, p := range providers {
		result[i] = map[string]any{
			"name":         p.Name,
			"display_name": p.DisplayName,
			"type":         p.Type,
			"model":        p.Model,
			"enabled":      p.Enabled,
		}
	}

	return c.JSON(fiber.Map{
		"providers": result,
		"default":   h.defaultProvider,
	})
}

// Helper to marshal embedding to JSON (handles float32 slice)
func marshalEmbedding(embedding []float32) (string, error) {
	data, err := json.Marshal(embedding)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
