package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/ai"
	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

type VectorHandler struct {
	vectorManager   *VectorManager
	config          *config.AIConfig
	baseConfig      *config.Config
	schemaInspector *database.SchemaInspector
	db              *database.Connection
}

func NewVectorHandler(vectorManager *VectorManager, schemaInspector *database.SchemaInspector, db *database.Connection, baseConfig *config.Config) (*VectorHandler, error) {
	handler := &VectorHandler{
		vectorManager:   vectorManager,
		config:          vectorManager.envConfig,
		baseConfig:      baseConfig,
		schemaInspector: schemaInspector,
		db:              db,
	}

	return handler, nil
}

func (h *VectorHandler) getConfig(c fiber.Ctx) *config.AIConfig {
	if tc, ok := c.Locals("tenant_config").(*config.Config); ok && tc != nil {
		return &tc.AI
	}
	if h.config != nil {
		return h.config
	}
	return &h.baseConfig.AI
}

func inferProviderType(cfg *config.AIConfig) string {
	if cfg.EmbeddingProvider != "" {
		return cfg.EmbeddingProvider
	}
	if cfg.ProviderType != "" {
		return cfg.ProviderType
	}
	if cfg.OpenAIAPIKey != "" {
		return "openai"
	}
	if cfg.AzureAPIKey != "" && cfg.AzureEndpoint != "" {
		return "azure"
	}
	if cfg.OllamaEndpoint != "" {
		return "ollama"
	}
	return ""
}

func buildEmbeddingConfig(cfg *config.AIConfig) (ai.EmbeddingServiceConfig, error) {
	providerType := inferProviderType(cfg)

	defaultModel := cfg.EmbeddingModel
	if defaultModel == "" {
		switch providerType {
		case "openai":
			defaultModel = "text-embedding-3-small"
		case "azure":
			defaultModel = "text-embedding-ada-002"
		case "ollama":
			defaultModel = "nomic-embed-text"
		}
	}

	providerCfg := ai.ProviderConfig{
		Type:   ai.ProviderType(providerType),
		Model:  defaultModel,
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
		DefaultModel: defaultModel,
		CacheEnabled: true,
	}, nil
}

func buildEmbeddingConfigFromAIProvider(cfg *config.AIConfig) (ai.EmbeddingServiceConfig, error) {
	providerType := inferProviderType(cfg)

	providerCfg := ai.ProviderConfig{
		Type:   ai.ProviderType(providerType),
		Config: make(map[string]string),
	}

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

type EmbedRequest struct {
	Text     string   `json:"text,omitempty"`
	Texts    []string `json:"texts,omitempty"`
	Model    string   `json:"model,omitempty"`
	Provider string   `json:"provider,omitempty"`
}

type EmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
	Model      string      `json:"model"`
	Dimensions int         `json:"dimensions"`
	Usage      *EmbedUsage `json:"usage,omitempty"`
}

type EmbedUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type VectorSearchRequest struct {
	Table          string              `json:"table"`
	Column         string              `json:"column"`
	Query          string              `json:"query,omitempty"`
	Vector         []float64           `json:"vector,omitempty"`
	Metric         string              `json:"metric,omitempty"`
	MatchThreshold *float64            `json:"match_threshold,omitempty"`
	MatchCount     *int                `json:"match_count,omitempty"`
	Select         string              `json:"select,omitempty"`
	Filters        []VectorQueryFilter `json:"filters,omitempty"`
}

type VectorQueryFilter struct {
	Column   string      `json:"column"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type VectorSearchResponse struct {
	Data      []map[string]interface{} `json:"data"`
	Distances []float64                `json:"distances,omitempty"`
	Model     string                   `json:"model,omitempty"`
}

func (h *VectorHandler) HandleEmbed(c fiber.Ctx) error {
	var req EmbedRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	var embeddingService *ai.EmbeddingService
	var err error

	if req.Provider != "" {
		role, _ := c.Locals("user_role").(string)
		isAdmin := role == "admin" || role == "instance_admin" || role == "service_role" || role == "tenant_service"

		if !isAdmin {
			return SendForbidden(c, "Provider selection requires admin privileges", ErrCodeAccessDenied)
		}

		embeddingService, err = h.vectorManager.GetEmbeddingServiceForProvider(c.RequestCtx(), req.Provider)
		if err != nil {
			return SendBadRequest(c, "Invalid embedding provider", ErrCodeInvalidInput)
		}
	} else {
		embeddingService = h.vectorManager.GetEmbeddingService()
	}

	if embeddingService == nil {
		return SendFeatureDisabled(c, "Vector search")
	}

	var texts []string
	if req.Text != "" {
		texts = append(texts, req.Text)
	}
	texts = append(texts, req.Texts...)

	if len(texts) == 0 {
		return SendBadRequest(c, "No text provided for embedding", ErrCodeInvalidInput)
	}

	resp, err := embeddingService.Embed(c.RequestCtx(), texts, req.Model)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate embeddings")
		return SendInternalError(c, "Failed to generate embeddings")
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

func (h *VectorHandler) HandleSearch(c fiber.Ctx) error {
	if h.db == nil {
		return SendNotInitialized(c, "Database")
	}

	var req VectorSearchRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Table == "" || req.Column == "" {
		return SendBadRequest(c, "table and column are required", ErrCodeMissingField)
	}

	if !isValidIdentifier(req.Table) || !isValidIdentifier(req.Column) {
		return SendBadRequest(c, "Invalid table or column name", ErrCodeInvalidInput)
	}

	var queryVector []float64
	var embeddingModel string

	//nolint:gocritic // Conditions check different request fields, not switch-compatible
	if req.Query != "" {
		embeddingService := h.vectorManager.GetEmbeddingService()
		if embeddingService == nil {
			return SendFeatureDisabled(c, "Vector search")
		}

		embedding, err := embeddingService.EmbedSingle(c.RequestCtx(), req.Query, "")
		if err != nil {
			log.Error().Err(err).Msg("Failed to embed query")
			return SendInternalError(c, "Failed to embed query")
		}

		queryVector = make([]float64, len(embedding))
		for i, v := range embedding {
			queryVector[i] = float64(v)
		}
		embeddingModel = embeddingService.DefaultModel()
	} else if len(req.Vector) > 0 {
		queryVector = req.Vector
	} else {
		return SendBadRequest(c, "Either query or vector must be provided", ErrCodeMissingField)
	}

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
		return SendBadRequest(c, "Invalid metric; use l2, cosine, or inner_product", ErrCodeInvalidInput)
	}

	selectCols := "*"
	if req.Select != "" {
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

	matchCount := 10
	if req.MatchCount != nil && *req.MatchCount > 0 {
		matchCount = *req.MatchCount
		if matchCount > 1000 {
			matchCount = 1000
		}
	}

	userID := ""
	userRole := "anon"
	var claims *auth.TokenClaims
	if user, ok := c.Locals("user").(*auth.TokenClaims); ok && user != nil {
		userID = user.Subject
		userRole = user.Role
		claims = user
	} else {
		if id := middleware.GetUserID(c); id != "" {
			userID = id
		}
		if role, ok := c.Locals("user_role").(string); ok && role != "" {
			userRole = role
		}
		if jwtClaims, ok := c.Locals("jwt_claims").(*auth.TokenClaims); ok {
			claims = jwtClaims
		}
	}

	data, distances, err := h.executeVectorSearch(middleware.CtxWithTenant(c), vectorSearchParams{
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
		return SendInternalError(c, "Vector search failed")
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

func (h *VectorHandler) executeVectorSearch(ctx context.Context, params vectorSearchParams) ([]map[string]interface{}, []float64, error) {
	vectorStr := formatVectorLiteral(params.queryVector)

	query := fmt.Sprintf(`
		SELECT %s, (%s %s '%s'::vector) as _distance
		FROM %s
		WHERE 1=1
	`, params.selectCols, params.column, params.distanceOp, vectorStr, params.table)

	if params.matchThreshold != nil {
		query += fmt.Sprintf(" AND (%s %s '%s'::vector) < %f",
			params.column, params.distanceOp, vectorStr, *params.matchThreshold)
	}

	for i, filter := range params.filters {
		if !isValidIdentifier(filter.Column) {
			continue
		}
		op := normalizeOperator(filter.Operator)
		if op == "" {
			continue
		}
		query += fmt.Sprintf(" AND %s %s $%d", filter.Column, op, i+1)
	}

	query += fmt.Sprintf(" ORDER BY _distance LIMIT %d", params.matchCount)

	filterValues := make([]interface{}, len(params.filters))
	for i, filter := range params.filters {
		filterValues[i] = filter.Value
	}

	tx, err := h.db.Pool().Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := middleware.SetRLSContext(ctx, tx, params.userID, params.userRole, params.claims); err != nil {
		return nil, nil, fmt.Errorf("failed to set RLS context: %w", err)
	}

	rows, err := tx.Query(ctx, query, filterValues...)
	if err != nil {
		return nil, nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

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

func formatVectorLiteral(v []float64) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

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

func (h *VectorHandler) IsEmbeddingConfigured() bool {
	return h.vectorManager.GetEmbeddingService() != nil
}

func (h *VectorHandler) GetEmbeddingService() *ai.EmbeddingService {
	return h.vectorManager.GetEmbeddingService()
}

type VectorCapabilities struct {
	Enabled           bool   `json:"enabled"`
	PgVectorInstalled bool   `json:"pgvector_installed"`
	PgVectorVersion   string `json:"pgvector_version,omitempty"`
	EmbeddingEnabled  bool   `json:"embedding_enabled"`
	EmbeddingProvider string `json:"embedding_provider,omitempty"`
	EmbeddingModel    string `json:"embedding_model,omitempty"`
}

func (h *VectorHandler) HandleGetCapabilities(c fiber.Ctx) error {
	embeddingAvailable := h.vectorManager.GetEmbeddingService() != nil

	pgVectorInstalled := false
	var pgVectorVersion string
	if h.schemaInspector != nil {
		installed, version, err := h.schemaInspector.IsPgVectorInstalled(c.RequestCtx())
		if err != nil {
			log.Warn().Err(err).Msg("Failed to check pgvector status")
		} else {
			pgVectorInstalled = installed
			pgVectorVersion = version
		}
	}

	role, _ := c.Locals("user_role").(string)
	isAdmin := role == "admin" || role == "instance_admin" || role == "service_role" || role == "tenant_service"

	if !isAdmin {
		return c.JSON(fiber.Map{
			"enabled": pgVectorInstalled && embeddingAvailable,
		})
	}

	caps := VectorCapabilities{
		Enabled:           pgVectorInstalled && embeddingAvailable,
		PgVectorInstalled: pgVectorInstalled,
		PgVectorVersion:   pgVectorVersion,
		EmbeddingEnabled:  embeddingAvailable,
	}

	if embeddingAvailable {
		cfg := h.getConfig(c)

		provider := cfg.EmbeddingProvider
		if provider == "" {
			provider = cfg.ProviderType
		}
		caps.EmbeddingProvider = provider

		embeddingService := h.vectorManager.GetEmbeddingService()
		if embeddingService != nil {
			caps.EmbeddingModel = embeddingService.DefaultModel()
		} else if cfg.EmbeddingModel != "" {
			caps.EmbeddingModel = cfg.EmbeddingModel
		}
	}

	return c.JSON(caps)
}

func (h *VectorHandler) IsPgVectorInstalled(c fiber.Ctx) bool {
	if h.schemaInspector == nil {
		return false
	}
	installed, _, err := h.schemaInspector.IsPgVectorInstalled(c.RequestCtx())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to check pgvector status")
		return false
	}
	return installed
}

// fiber:context-methods migrated
