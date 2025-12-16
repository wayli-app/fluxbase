package api

import (
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/ai"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// VectorHandler handles vector search endpoints
type VectorHandler struct {
	embeddingService *ai.EmbeddingService
	config           *config.AIConfig
	schemaInspector  *database.SchemaInspector
}

// NewVectorHandler creates a new vector handler
func NewVectorHandler(cfg *config.AIConfig, schemaInspector *database.SchemaInspector) (*VectorHandler, error) {
	handler := &VectorHandler{
		config:          cfg,
		schemaInspector: schemaInspector,
	}

	// Initialize embedding service if enabled
	if cfg.EmbeddingEnabled {
		embeddingCfg, err := buildEmbeddingConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to build embedding config: %w", err)
		}

		service, err := ai.NewEmbeddingService(embeddingCfg)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to initialize embedding service")
		} else {
			handler.embeddingService = service
		}
	}

	return handler, nil
}

// buildEmbeddingConfig builds the embedding service config from AI config
func buildEmbeddingConfig(cfg *config.AIConfig) (ai.EmbeddingServiceConfig, error) {
	// Determine provider type (fallback to main provider if embedding provider not specified)
	providerType := cfg.EmbeddingProvider
	if providerType == "" {
		providerType = cfg.ProviderType
	}

	// Build provider config using the map-based configuration
	providerCfg := ai.ProviderConfig{
		Type:   ai.ProviderType(providerType),
		Model:  cfg.EmbeddingModel,
		Config: make(map[string]string),
	}

	switch providerType {
	case "openai":
		providerCfg.Config["api_key"] = cfg.OpenAIAPIKey
		if cfg.OpenAIOrganizationID != "" {
			providerCfg.Config["organization_id"] = cfg.OpenAIOrganizationID
		}
		if cfg.OpenAIBaseURL != "" {
			providerCfg.Config["base_url"] = cfg.OpenAIBaseURL
		}
	case "azure":
		providerCfg.Config["api_key"] = cfg.AzureAPIKey
		providerCfg.Config["endpoint"] = cfg.AzureEndpoint
		deploymentName := cfg.AzureEmbeddingDeploymentName
		if deploymentName == "" {
			deploymentName = cfg.AzureDeploymentName
		}
		providerCfg.Config["deployment_name"] = deploymentName
		if cfg.AzureAPIVersion != "" {
			providerCfg.Config["api_version"] = cfg.AzureAPIVersion
		}
	case "ollama":
		providerCfg.Config["endpoint"] = cfg.OllamaEndpoint
	default:
		return ai.EmbeddingServiceConfig{}, fmt.Errorf("unsupported embedding provider: %s", providerType)
	}

	return ai.EmbeddingServiceConfig{
		Provider:     providerCfg,
		DefaultModel: cfg.EmbeddingModel,
		CacheEnabled: true,
	}, nil
}

// EmbedRequest represents a request to generate embeddings
type EmbedRequest struct {
	Text  string   `json:"text,omitempty"`  // Single text to embed
	Texts []string `json:"texts,omitempty"` // Multiple texts to embed
	Model string   `json:"model,omitempty"` // Optional model override
}

// EmbedResponse represents the response from embedding generation
type EmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	Model      string      `json:"model"`
	Dimensions int         `json:"dimensions"`
	Usage      *EmbedUsage `json:"usage,omitempty"`
}

// EmbedUsage represents token usage for embedding
type EmbedUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// VectorSearchRequest represents a request for vector search
type VectorSearchRequest struct {
	Table          string              `json:"table"`
	Column         string              `json:"column"`
	Query          string              `json:"query,omitempty"`  // Text to search (will be auto-embedded)
	Vector         []float64           `json:"vector,omitempty"` // Direct vector input
	Metric         string              `json:"metric,omitempty"` // Distance metric: l2, cosine, inner_product
	MatchThreshold *float64            `json:"match_threshold,omitempty"`
	MatchCount     *int                `json:"match_count,omitempty"`
	Select         string              `json:"select,omitempty"`
	Filters        []VectorQueryFilter `json:"filters,omitempty"`
}

// VectorQueryFilter represents a filter for the search
type VectorQueryFilter struct {
	Column   string      `json:"column"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// VectorSearchResponse represents the response from vector search
type VectorSearchResponse struct {
	Data      []map[string]interface{} `json:"data"`
	Distances []float64                `json:"distances,omitempty"`
	Model     string                   `json:"model,omitempty"`
}

// HandleEmbed handles POST /api/v1/vector/embed
func (h *VectorHandler) HandleEmbed(c *fiber.Ctx) error {
	if h.embeddingService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Embedding service not configured",
		})
	}

	var req EmbedRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Collect texts to embed
	var texts []string
	if req.Text != "" {
		texts = append(texts, req.Text)
	}
	texts = append(texts, req.Texts...)

	if len(texts) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No text provided for embedding",
		})
	}

	// Generate embeddings
	resp, err := h.embeddingService.Embed(c.Context(), texts, req.Model)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate embeddings")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate embeddings: " + err.Error(),
		})
	}

	result := EmbedResponse{
		Embeddings: resp.Embeddings,
		Model:      resp.Model,
		Dimensions: resp.Dimensions,
	}

	if resp.Usage != nil {
		result.Usage = &EmbedUsage{
			PromptTokens: resp.Usage.PromptTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		}
	}

	return c.JSON(result)
}

// HandleSearch handles POST /api/v1/vector/search
// This is a convenience endpoint that auto-embeds query text and performs vector similarity search
func (h *VectorHandler) HandleSearch(c *fiber.Ctx) error {
	var req VectorSearchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Table == "" || req.Column == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "table and column are required",
		})
	}

	// Determine the query vector
	var queryVector []float64
	var embeddingModel string

	if req.Query != "" {
		// Auto-embed the query text
		if h.embeddingService == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Embedding service not configured; provide vector directly",
			})
		}

		embedding, err := h.embeddingService.EmbedSingle(c.Context(), req.Query, "")
		if err != nil {
			log.Error().Err(err).Msg("Failed to embed query")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to embed query: " + err.Error(),
			})
		}

		// Convert float32 to float64
		queryVector = make([]float64, len(embedding))
		for i, v := range embedding {
			queryVector[i] = float64(v)
		}
		embeddingModel = h.embeddingService.DefaultModel()
	} else if len(req.Vector) > 0 {
		queryVector = req.Vector
	} else {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Either query or vector must be provided",
		})
	}

	// Validate metric
	metric := req.Metric
	if metric == "" {
		metric = "cosine"
	}

	switch strings.ToLower(metric) {
	case "l2", "euclidean", "cosine", "inner_product", "ip":
		// Valid
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid metric; use l2, cosine, or inner_product",
		})
	}

	// For now, return an informational response indicating how to use vector search
	// A full implementation would execute the query using the tables API internally
	result := VectorSearchResponse{
		Data:  []map[string]interface{}{},
		Model: embeddingModel,
	}

	// Log the request for debugging
	log.Debug().
		Str("table", req.Table).
		Str("column", req.Column).
		Str("metric", metric).
		Int("vector_dimensions", len(queryVector)).
		Msg("Vector search request")

	return c.JSON(result)
}

// IsEmbeddingConfigured returns whether the embedding service is available
func (h *VectorHandler) IsEmbeddingConfigured() bool {
	return h.embeddingService != nil
}

// VectorCapabilities represents the vector search capabilities of the system
type VectorCapabilities struct {
	Enabled           bool   `json:"enabled"`
	PgVectorInstalled bool   `json:"pgvector_installed"`
	PgVectorVersion   string `json:"pgvector_version,omitempty"`
	EmbeddingEnabled  bool   `json:"embedding_enabled"`
	EmbeddingProvider string `json:"embedding_provider,omitempty"`
	EmbeddingModel    string `json:"embedding_model,omitempty"`
}

// HandleGetCapabilities handles GET /api/v1/capabilities/vector
// Returns information about vector search capabilities
func (h *VectorHandler) HandleGetCapabilities(c *fiber.Ctx) error {
	caps := VectorCapabilities{
		EmbeddingEnabled: h.config.EmbeddingEnabled,
	}

	// Check pgvector installation status
	if h.schemaInspector != nil {
		installed, version, err := h.schemaInspector.IsPgVectorInstalled(c.Context())
		if err != nil {
			log.Warn().Err(err).Msg("Failed to check pgvector status")
		} else {
			caps.PgVectorInstalled = installed
			caps.PgVectorVersion = version
		}
	}

	// Set enabled if both pgvector is installed and embedding is available
	caps.Enabled = caps.PgVectorInstalled && caps.EmbeddingEnabled

	// Add embedding provider info if enabled
	if caps.EmbeddingEnabled {
		provider := h.config.EmbeddingProvider
		if provider == "" {
			provider = h.config.ProviderType
		}
		caps.EmbeddingProvider = provider
		caps.EmbeddingModel = h.config.EmbeddingModel
	}

	return c.JSON(caps)
}

// IsPgVectorInstalled checks whether pgvector is installed on the database
func (h *VectorHandler) IsPgVectorInstalled(c *fiber.Ctx) bool {
	if h.schemaInspector == nil {
		return false
	}
	installed, _, err := h.schemaInspector.IsPgVectorInstalled(c.Context())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to check pgvector status")
		return false
	}
	return installed
}
