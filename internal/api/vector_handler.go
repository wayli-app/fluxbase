package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/ai"
	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// VectorHandler handles vector search endpoints
type VectorHandler struct {
	embeddingService *ai.EmbeddingService
	config           *config.AIConfig
	schemaInspector  *database.SchemaInspector
	db               *database.Connection
}

// NewVectorHandler creates a new vector handler
func NewVectorHandler(cfg *config.AIConfig, schemaInspector *database.SchemaInspector, db *database.Connection) (*VectorHandler, error) {
	handler := &VectorHandler{
		config:          cfg,
		schemaInspector: schemaInspector,
		db:              db,
	}

	// Initialize embedding service
	// Priority:
	// 1. If EmbeddingEnabled is true, use explicit embedding configuration
	// 2. If EmbeddingEnabled is false but AI provider is configured (ProviderEnabled=true),
	//    try to use AI provider credentials as fallback for embeddings
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
			log.Info().Msg("Embedding service initialized from explicit configuration")
		}
	} else if cfg.ProviderEnabled && cfg.ProviderType != "" {
		// Fallback: Try to use AI provider settings for embeddings
		embeddingCfg, err := buildEmbeddingConfigFromAIProvider(cfg)
		if err != nil {
			log.Debug().Err(err).Msg("Could not initialize embedding from AI provider (fallback)")
		} else {
			service, err := ai.NewEmbeddingService(embeddingCfg)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to initialize embedding service from AI provider fallback")
			} else {
				handler.embeddingService = service
				log.Info().
					Str("provider", cfg.ProviderType).
					Msg("Embedding service initialized from AI provider settings (fallback)")
			}
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

// buildEmbeddingConfigFromAIProvider builds embedding config using AI provider settings as fallback
// This allows embeddings to work when only the main AI provider is configured (e.g., OpenAI for chatbots)
func buildEmbeddingConfigFromAIProvider(cfg *config.AIConfig) (ai.EmbeddingServiceConfig, error) {
	providerType := cfg.ProviderType

	// Build provider config using the main AI provider credentials
	providerCfg := ai.ProviderConfig{
		Type:   ai.ProviderType(providerType),
		Config: make(map[string]string),
	}

	// Determine default embedding model based on provider
	var defaultModel string

	switch providerType {
	case "openai":
		if cfg.OpenAIAPIKey == "" {
			return ai.EmbeddingServiceConfig{}, fmt.Errorf("openai: api_key not configured")
		}
		providerCfg.Config["api_key"] = cfg.OpenAIAPIKey
		if cfg.OpenAIOrganizationID != "" {
			providerCfg.Config["organization_id"] = cfg.OpenAIOrganizationID
		}
		if cfg.OpenAIBaseURL != "" {
			providerCfg.Config["base_url"] = cfg.OpenAIBaseURL
		}
		defaultModel = "text-embedding-3-small"

	case "azure":
		if cfg.AzureAPIKey == "" || cfg.AzureEndpoint == "" {
			return ai.EmbeddingServiceConfig{}, fmt.Errorf("azure: api_key or endpoint not configured")
		}
		providerCfg.Config["api_key"] = cfg.AzureAPIKey
		providerCfg.Config["endpoint"] = cfg.AzureEndpoint
		// For Azure, we need a deployment name - try embedding-specific first, then fall back to main
		deploymentName := cfg.AzureEmbeddingDeploymentName
		if deploymentName == "" {
			deploymentName = cfg.AzureDeploymentName
		}
		if deploymentName == "" {
			return ai.EmbeddingServiceConfig{}, fmt.Errorf("azure: no deployment name configured for embeddings")
		}
		providerCfg.Config["deployment_name"] = deploymentName
		if cfg.AzureAPIVersion != "" {
			providerCfg.Config["api_version"] = cfg.AzureAPIVersion
		}
		defaultModel = "text-embedding-ada-002"

	case "ollama":
		endpoint := cfg.OllamaEndpoint
		if endpoint == "" {
			endpoint = "http://localhost:11434"
		}
		providerCfg.Config["endpoint"] = endpoint
		defaultModel = "nomic-embed-text"

	default:
		return ai.EmbeddingServiceConfig{}, fmt.Errorf("unsupported provider type for embedding fallback: %s", providerType)
	}

	// Use explicit embedding model if configured, otherwise use provider default
	if cfg.EmbeddingModel != "" {
		defaultModel = cfg.EmbeddingModel
	}
	providerCfg.Model = defaultModel

	return ai.EmbeddingServiceConfig{
		Provider:     providerCfg,
		DefaultModel: defaultModel,
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
	if h.db == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Database not configured",
		})
	}

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

	// Validate table and column names (prevent SQL injection)
	if !isValidIdentifier(req.Table) || !isValidIdentifier(req.Column) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid table or column name",
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

	// Validate and normalize metric
	metric := strings.ToLower(req.Metric)
	if metric == "" {
		metric = "cosine"
	}

	var distanceOp string
	switch metric {
	case "l2", "euclidean":
		distanceOp = "<->"
	case "cosine":
		distanceOp = "<=>"
	case "inner_product", "ip":
		distanceOp = "<#>"
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid metric; use l2, cosine, or inner_product",
		})
	}

	// Build select columns
	selectCols := "*"
	if req.Select != "" {
		// Validate select columns
		cols := strings.Split(req.Select, ",")
		validCols := make([]string, 0, len(cols))
		for _, col := range cols {
			col = strings.TrimSpace(col)
			if isValidIdentifier(col) {
				validCols = append(validCols, col)
			}
		}
		if len(validCols) > 0 {
			selectCols = strings.Join(validCols, ", ")
		}
	}

	// Set defaults
	matchCount := 10
	if req.MatchCount != nil && *req.MatchCount > 0 {
		matchCount = *req.MatchCount
		if matchCount > 1000 {
			matchCount = 1000 // Cap at 1000
		}
	}

	// Get user context for RLS
	userID := ""
	userRole := "anon"
	var claims *auth.TokenClaims
	if user, ok := c.Locals("user").(*auth.TokenClaims); ok && user != nil {
		userID = user.Subject
		userRole = user.Role
		claims = user
	}

	// Execute vector search with RLS context
	data, distances, err := h.executeVectorSearch(c.Context(), vectorSearchParams{
		table:          req.Table,
		column:         req.Column,
		selectCols:     selectCols,
		queryVector:    queryVector,
		distanceOp:     distanceOp,
		matchThreshold: req.MatchThreshold,
		matchCount:     matchCount,
		filters:        req.Filters,
		userID:         userID,
		userRole:       userRole,
		claims:         claims,
	})
	if err != nil {
		log.Error().Err(err).
			Str("table", req.Table).
			Str("column", req.Column).
			Msg("Vector search failed")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Vector search failed: " + err.Error(),
		})
	}

	result := VectorSearchResponse{
		Data:      data,
		Distances: distances,
		Model:     embeddingModel,
	}

	log.Debug().
		Str("table", req.Table).
		Str("column", req.Column).
		Str("metric", metric).
		Int("results", len(data)).
		Msg("Vector search completed")

	return c.JSON(result)
}

// vectorSearchParams holds parameters for vector search execution
type vectorSearchParams struct {
	table          string
	column         string
	selectCols     string
	queryVector    []float64
	distanceOp     string
	matchThreshold *float64
	matchCount     int
	filters        []VectorQueryFilter
	userID         string
	userRole       string
	claims         *auth.TokenClaims
}

// executeVectorSearch executes the vector similarity search with RLS context
func (h *VectorHandler) executeVectorSearch(ctx context.Context, params vectorSearchParams) ([]map[string]interface{}, []float64, error) {
	// Format the vector as PostgreSQL array literal
	vectorStr := formatVectorLiteral(params.queryVector)

	// Build the base query
	// Using subquery to calculate distance once
	query := fmt.Sprintf(`
		SELECT %s, (%s %s '%s'::vector) as _distance
		FROM %s
		WHERE 1=1
	`, params.selectCols, params.column, params.distanceOp, vectorStr, params.table)

	// Add threshold filter if specified
	if params.matchThreshold != nil {
		query += fmt.Sprintf(" AND (%s %s '%s'::vector) < %f",
			params.column, params.distanceOp, vectorStr, *params.matchThreshold)
	}

	// Add custom filters
	for i, filter := range params.filters {
		if !isValidIdentifier(filter.Column) {
			continue
		}
		op := normalizeOperator(filter.Operator)
		if op == "" {
			continue
		}
		// Use parameter placeholder to prevent injection
		query += fmt.Sprintf(" AND %s %s $%d", filter.Column, op, i+1)
	}

	// Add ordering and limit
	query += fmt.Sprintf(" ORDER BY _distance LIMIT %d", params.matchCount)

	// Collect filter values for parameterized query
	filterValues := make([]interface{}, len(params.filters))
	for i, filter := range params.filters {
		filterValues[i] = filter.Value
	}

	// Execute with RLS context
	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Set RLS context
	if err := middleware.SetRLSContext(ctx, tx, params.userID, params.userRole, params.claims); err != nil {
		return nil, nil, fmt.Errorf("failed to set RLS context: %w", err)
	}

	// Execute query
	rows, err := tx.Query(ctx, query, filterValues...)
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Collect results
	var data []map[string]interface{}
	var distances []float64

	fieldDescs := rows.FieldDescriptions()
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			continue
		}

		row := make(map[string]interface{})
		var distance float64

		for i, fd := range fieldDescs {
			colName := string(fd.Name)
			if colName == "_distance" {
				if d, ok := values[i].(float64); ok {
					distance = d
				}
			} else {
				row[colName] = values[i]
			}
		}

		data = append(data, row)
		distances = append(distances, distance)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit: %w", err)
	}

	return data, distances, nil
}

// formatVectorLiteral formats a float64 slice as PostgreSQL vector literal
func formatVectorLiteral(v []float64) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// normalizeOperator converts filter operators to SQL operators
func normalizeOperator(op string) string {
	switch strings.ToLower(op) {
	case "eq", "=":
		return "="
	case "neq", "!=", "<>":
		return "!="
	case "gt", ">":
		return ">"
	case "gte", ">=":
		return ">="
	case "lt", "<":
		return "<"
	case "lte", "<=":
		return "<="
	case "like":
		return "LIKE"
	case "ilike":
		return "ILIKE"
	case "is":
		return "IS"
	case "in":
		return "IN"
	default:
		return ""
	}
}

// IsEmbeddingConfigured returns whether the embedding service is available
func (h *VectorHandler) IsEmbeddingConfigured() bool {
	return h.embeddingService != nil
}

// GetEmbeddingService returns the embedding service (may be nil if not configured)
func (h *VectorHandler) GetEmbeddingService() *ai.EmbeddingService {
	return h.embeddingService
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
// Non-admin users only receive minimal info (enabled status)
func (h *VectorHandler) HandleGetCapabilities(c *fiber.Ctx) error {
	// EmbeddingEnabled reflects actual service availability (including fallback)
	embeddingAvailable := h.embeddingService != nil

	// Check pgvector installation status
	pgVectorInstalled := false
	var pgVectorVersion string
	if h.schemaInspector != nil {
		installed, version, err := h.schemaInspector.IsPgVectorInstalled(c.Context())
		if err != nil {
			log.Warn().Err(err).Msg("Failed to check pgvector status")
		} else {
			pgVectorInstalled = installed
			pgVectorVersion = version
		}
	}

	// Check if user has admin role
	role, _ := c.Locals("user_role").(string)
	isAdmin := role == "admin" || role == "dashboard_admin" || role == "service_role"

	// Non-admin users only get minimal info (enabled status)
	if !isAdmin {
		return c.JSON(fiber.Map{
			"enabled": pgVectorInstalled && embeddingAvailable,
		})
	}

	// Admin users get full details
	caps := VectorCapabilities{
		Enabled:           pgVectorInstalled && embeddingAvailable,
		PgVectorInstalled: pgVectorInstalled,
		PgVectorVersion:   pgVectorVersion,
		EmbeddingEnabled:  embeddingAvailable,
	}

	// Add embedding provider info if embedding is available
	if embeddingAvailable {
		// Determine actual provider being used
		provider := h.config.EmbeddingProvider
		if provider == "" {
			provider = h.config.ProviderType
		}
		caps.EmbeddingProvider = provider

		// Get model from service if available
		if h.embeddingService != nil {
			caps.EmbeddingModel = h.embeddingService.DefaultModel()
		} else if h.config.EmbeddingModel != "" {
			caps.EmbeddingModel = h.config.EmbeddingModel
		}
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
