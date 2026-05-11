package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/ai"
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

func (h *InternalAIHandler) requireAIStorage(c fiber.Ctx) error {
	if h.aiStorage == nil {
		return SendNotInitialized(c, "AI storage")
	}
	return nil
}

func (h *InternalAIHandler) requireEmbeddingService(c fiber.Ctx) error {
	if h.embeddingService == nil {
		return SendNotInitialized(c, "Embedding service")
	}
	return nil
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
func (h *InternalAIHandler) HandleChat(c fiber.Ctx) error {
	if err := h.requireAIStorage(c); err != nil {
		return err
	}

	var req InternalChatRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if len(req.Messages) == 0 {
		return SendBadRequest(c, "Messages array is required", ErrCodeMissingField)
	}

	// Get provider - use specified or default
	providerName := req.Provider
	if providerName == "" {
		providerName = h.defaultProvider
	}
	if providerName == "" {
		return SendBadRequest(c, "No AI provider configured. Set 'provider' in request or configure default provider.", ErrCodeInvalidInput)
	}

	// Get the provider from storage
	provider, err := h.aiStorage.GetProviderByName(c.RequestCtx(), providerName)
	if err != nil {
		log.Warn().Err(err).Str("provider", providerName).Msg("Failed to get AI provider")
		return SendNotFound(c, fmt.Sprintf("AI provider '%s' not found", providerName))
	}

	// Build provider config
	// Get model from config map or use provided model
	model := req.Model
	if model == "" {
		if m, ok := provider.Config["model"]; ok && m != "" {
			model = m
		}
	}

	providerConfig := ai.ProviderConfig{
		Name:        provider.Name,
		DisplayName: provider.DisplayName,
		Type:        ai.ProviderType(provider.ProviderType),
		Model:       model,
		Config:      provider.Config,
	}

	// Create the provider instance
	aiProvider, err := ai.NewProvider(providerConfig)
	if err != nil {
		log.Error().Err(err).Str("provider", providerName).Msg("Failed to create AI provider")
		return SendInternalError(c, "Failed to initialize AI provider")
	}
	defer func() { _ = aiProvider.Close() }()

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

	resp, err := aiProvider.Chat(c.RequestCtx(), chatReq)
	if err != nil {
		log.Error().Err(err).Msg("AI chat request failed")
		return SendInternalError(c, "AI request failed")
	}

	if len(resp.Choices) == 0 {
		return SendInternalError(c, "AI returned no response")
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
func (h *InternalAIHandler) HandleEmbed(c fiber.Ctx) error {
	if err := h.requireEmbeddingService(c); err != nil {
		return err
	}

	var req InternalEmbedRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Text == "" {
		return SendMissingField(c, "text")
	}

	// Generate embedding
	embedding, err := h.embeddingService.GenerateEmbedding(c.RequestCtx(), req.Text)
	if err != nil {
		log.Error().Err(err).Msg("Embedding generation failed")
		return SendInternalError(c, "Embedding generation failed")
	}

	// Get default model name
	modelName := h.embeddingService.DefaultModel()

	return c.JSON(InternalEmbedResponse{
		Embedding: embedding,
		Model:     modelName,
	})
}

// HandleListProviders handles GET /api/v1/internal/ai/providers
// This endpoint lists available AI providers.
func (h *InternalAIHandler) HandleListProviders(c fiber.Ctx) error {
	if err := h.requireAIStorage(c); err != nil {
		return err
	}

	providers, err := h.aiStorage.ListProviders(c.RequestCtx(), true) // Only enabled providers
	if err != nil {
		return SendInternalError(c, "Failed to list providers")
	}

	// Return simplified provider info (hide config/API keys)
	result := make([]map[string]any, len(providers))
	for i, p := range providers {
		// Get model from config if available
		model := ""
		if m, ok := p.Config["model"]; ok {
			model = m
		}
		result[i] = map[string]any{
			"name":         p.Name,
			"display_name": p.DisplayName,
			"type":         p.ProviderType,
			"model":        model,
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

// fiber:context-methods migrated
