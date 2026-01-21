package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// Storage handles database operations for AI entities
type Storage struct {
	db     *database.Connection
	config *config.AIConfig
}

// NewStorage creates a new AI storage instance
func NewStorage(db *database.Connection) *Storage {
	return &Storage{
		db:     db,
		config: nil, // Will be set via SetConfig if needed
	}
}

// SetConfig sets the AI configuration for the storage
func (s *Storage) SetConfig(cfg *config.AIConfig) {
	s.config = cfg
}

// UserExists checks if a user exists in auth.users
func (s *Storage) UserExists(ctx context.Context, userID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM auth.users WHERE id = $1)`
	err := s.db.Pool().QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return exists, nil
}

// ============================================================================
// CHATBOT OPERATIONS
// ============================================================================

// CreateChatbot creates a new chatbot in the database
func (s *Storage) CreateChatbot(ctx context.Context, chatbot *Chatbot) error {
	query := `
		INSERT INTO ai.chatbots (
			id, name, namespace, description, code, original_code, is_bundled, bundle_error,
			allowed_tables, allowed_operations, allowed_schemas, http_allowed_domains,
			intent_rules, required_columns, default_table,
			enabled, max_tokens, temperature, provider_id,
			persist_conversations, conversation_ttl_hours, max_conversation_turns,
			rate_limit_per_minute, daily_request_limit, daily_token_budget,
			allow_unauthenticated, is_public, require_roles, response_language, disable_execution_logs,
			mcp_tools, use_mcp_schema,
			version, source, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15,
			$16, $17, $18, $19,
			$20, $21, $22,
			$23, $24, $25,
			$26, $27, $28, $29, $30,
			$31, $32,
			$33, $34, $35, $36, $37
		)
	`

	if chatbot.ID == "" {
		chatbot.ID = uuid.New().String()
	}
	if chatbot.CreatedAt.IsZero() {
		chatbot.CreatedAt = time.Now()
	}
	chatbot.UpdatedAt = time.Now()

	// Serialize intent rules and required columns to JSON
	var intentRulesJSON, requiredColumnsJSON []byte
	var err error
	if len(chatbot.IntentRules) > 0 {
		intentRulesJSON, err = json.Marshal(chatbot.IntentRules)
		if err != nil {
			return fmt.Errorf("failed to marshal intent_rules: %w", err)
		}
	}
	if len(chatbot.RequiredColumns) > 0 {
		requiredColumnsJSON, err = json.Marshal(chatbot.RequiredColumns)
		if err != nil {
			return fmt.Errorf("failed to marshal required_columns: %w", err)
		}
	}

	_, err = s.db.Exec(ctx, query,
		chatbot.ID, chatbot.Name, chatbot.Namespace, chatbot.Description,
		chatbot.Code, chatbot.OriginalCode, chatbot.IsBundled, chatbot.BundleError,
		chatbot.AllowedTables, chatbot.AllowedOperations, chatbot.AllowedSchemas, chatbot.HTTPAllowedDomains,
		intentRulesJSON, requiredColumnsJSON, chatbot.DefaultTable,
		chatbot.Enabled, chatbot.MaxTokens, chatbot.Temperature, chatbot.ProviderID,
		chatbot.PersistConversations, chatbot.ConversationTTLHours, chatbot.MaxConversationTurns,
		chatbot.RateLimitPerMinute, chatbot.DailyRequestLimit, chatbot.DailyTokenBudget,
		chatbot.AllowUnauthenticated, chatbot.IsPublic, chatbot.RequireRoles, chatbot.ResponseLanguage, chatbot.DisableExecutionLogs,
		chatbot.MCPTools, chatbot.UseMCPSchema,
		chatbot.Version, chatbot.Source,
		chatbot.CreatedBy, chatbot.CreatedAt, chatbot.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create chatbot: %w", err)
	}

	log.Info().
		Str("id", chatbot.ID).
		Str("name", chatbot.Name).
		Str("namespace", chatbot.Namespace).
		Msg("Created chatbot")

	return nil
}

// UpdateChatbot updates an existing chatbot in the database
func (s *Storage) UpdateChatbot(ctx context.Context, chatbot *Chatbot) error {
	query := `
		UPDATE ai.chatbots SET
			description = $2,
			code = $3,
			original_code = $4,
			is_bundled = $5,
			bundle_error = $6,
			allowed_tables = $7,
			allowed_operations = $8,
			allowed_schemas = $9,
			http_allowed_domains = $10,
			intent_rules = $11,
			required_columns = $12,
			default_table = $13,
			enabled = $14,
			max_tokens = $15,
			temperature = $16,
			provider_id = $17,
			persist_conversations = $18,
			conversation_ttl_hours = $19,
			max_conversation_turns = $20,
			rate_limit_per_minute = $21,
			daily_request_limit = $22,
			daily_token_budget = $23,
			allow_unauthenticated = $24,
			is_public = $25,
			require_roles = $26,
			response_language = $27,
			disable_execution_logs = $28,
			mcp_tools = $29,
			use_mcp_schema = $30,
			version = version + 1,
			updated_at = $31
		WHERE id = $1
	`

	chatbot.UpdatedAt = time.Now()

	// Serialize intent rules and required columns to JSON
	var intentRulesJSON, requiredColumnsJSON []byte
	var err error
	if len(chatbot.IntentRules) > 0 {
		intentRulesJSON, err = json.Marshal(chatbot.IntentRules)
		if err != nil {
			return fmt.Errorf("failed to marshal intent_rules: %w", err)
		}
	}
	if len(chatbot.RequiredColumns) > 0 {
		requiredColumnsJSON, err = json.Marshal(chatbot.RequiredColumns)
		if err != nil {
			return fmt.Errorf("failed to marshal required_columns: %w", err)
		}
	}

	result, err := s.db.Exec(ctx, query,
		chatbot.ID,
		chatbot.Description,
		chatbot.Code,
		chatbot.OriginalCode,
		chatbot.IsBundled,
		chatbot.BundleError,
		chatbot.AllowedTables,
		chatbot.AllowedOperations,
		chatbot.AllowedSchemas,
		chatbot.HTTPAllowedDomains,
		intentRulesJSON,
		requiredColumnsJSON,
		chatbot.DefaultTable,
		chatbot.Enabled,
		chatbot.MaxTokens,
		chatbot.Temperature,
		chatbot.ProviderID,
		chatbot.PersistConversations,
		chatbot.ConversationTTLHours,
		chatbot.MaxConversationTurns,
		chatbot.RateLimitPerMinute,
		chatbot.DailyRequestLimit,
		chatbot.DailyTokenBudget,
		chatbot.AllowUnauthenticated,
		chatbot.IsPublic,
		chatbot.RequireRoles,
		chatbot.ResponseLanguage,
		chatbot.DisableExecutionLogs,
		chatbot.MCPTools,
		chatbot.UseMCPSchema,
		chatbot.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update chatbot: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("chatbot not found: %s", chatbot.ID)
	}

	log.Info().
		Str("id", chatbot.ID).
		Str("name", chatbot.Name).
		Msg("Updated chatbot")

	return nil
}

// GetChatbot retrieves a chatbot by ID
func (s *Storage) GetChatbot(ctx context.Context, id string) (*Chatbot, error) {
	query := `
		SELECT
			id, name, namespace, description, code, original_code, is_bundled, bundle_error,
			allowed_tables, allowed_operations, allowed_schemas, http_allowed_domains,
			intent_rules, required_columns, default_table,
			enabled, max_tokens, temperature, provider_id,
			persist_conversations, conversation_ttl_hours, max_conversation_turns,
			rate_limit_per_minute, daily_request_limit, daily_token_budget,
			allow_unauthenticated, is_public, require_roles, response_language, disable_execution_logs,
			mcp_tools, use_mcp_schema,
			version, source, created_by, created_at, updated_at
		FROM ai.chatbots
		WHERE id = $1
	`

	chatbot := &Chatbot{}
	var intentRulesJSON, requiredColumnsJSON []byte
	var defaultTable *string
	var responseLanguage *string
	err := s.db.QueryRow(ctx, query, id).Scan(
		&chatbot.ID, &chatbot.Name, &chatbot.Namespace, &chatbot.Description,
		&chatbot.Code, &chatbot.OriginalCode, &chatbot.IsBundled, &chatbot.BundleError,
		&chatbot.AllowedTables, &chatbot.AllowedOperations, &chatbot.AllowedSchemas, &chatbot.HTTPAllowedDomains,
		&intentRulesJSON, &requiredColumnsJSON, &defaultTable,
		&chatbot.Enabled, &chatbot.MaxTokens, &chatbot.Temperature, &chatbot.ProviderID,
		&chatbot.PersistConversations, &chatbot.ConversationTTLHours, &chatbot.MaxConversationTurns,
		&chatbot.RateLimitPerMinute, &chatbot.DailyRequestLimit, &chatbot.DailyTokenBudget,
		&chatbot.AllowUnauthenticated, &chatbot.IsPublic, &chatbot.RequireRoles, &responseLanguage, &chatbot.DisableExecutionLogs,
		&chatbot.MCPTools, &chatbot.UseMCPSchema,
		&chatbot.Version, &chatbot.Source,
		&chatbot.CreatedBy, &chatbot.CreatedAt, &chatbot.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get chatbot: %w", err)
	}

	// Deserialize JSON fields
	if len(intentRulesJSON) > 0 {
		if err := json.Unmarshal(intentRulesJSON, &chatbot.IntentRules); err != nil {
			log.Warn().Err(err).Str("chatbot_id", id).Msg("Failed to unmarshal intent_rules")
		}
	}
	if len(requiredColumnsJSON) > 0 {
		if err := json.Unmarshal(requiredColumnsJSON, &chatbot.RequiredColumns); err != nil {
			log.Warn().Err(err).Str("chatbot_id", id).Msg("Failed to unmarshal required_columns")
		}
	}
	if defaultTable != nil {
		chatbot.DefaultTable = *defaultTable
	}
	if responseLanguage != nil {
		chatbot.ResponseLanguage = *responseLanguage
	}

	chatbot.PopulateDerivedFields()
	return chatbot, nil
}

// GetChatbotByName retrieves a chatbot by name and namespace
func (s *Storage) GetChatbotByName(ctx context.Context, namespace, name string) (*Chatbot, error) {
	query := `
		SELECT
			id, name, namespace, description, code, original_code, is_bundled, bundle_error,
			allowed_tables, allowed_operations, allowed_schemas, http_allowed_domains,
			intent_rules, required_columns, default_table,
			enabled, max_tokens, temperature, provider_id,
			persist_conversations, conversation_ttl_hours, max_conversation_turns,
			rate_limit_per_minute, daily_request_limit, daily_token_budget,
			allow_unauthenticated, is_public, require_roles, response_language, disable_execution_logs,
			mcp_tools, use_mcp_schema,
			version, source, created_by, created_at, updated_at
		FROM ai.chatbots
		WHERE namespace = $1 AND name = $2
	`

	chatbot := &Chatbot{}
	var intentRulesJSON, requiredColumnsJSON []byte
	var defaultTable *string
	var responseLanguage *string
	err := s.db.QueryRow(ctx, query, namespace, name).Scan(
		&chatbot.ID, &chatbot.Name, &chatbot.Namespace, &chatbot.Description,
		&chatbot.Code, &chatbot.OriginalCode, &chatbot.IsBundled, &chatbot.BundleError,
		&chatbot.AllowedTables, &chatbot.AllowedOperations, &chatbot.AllowedSchemas, &chatbot.HTTPAllowedDomains,
		&intentRulesJSON, &requiredColumnsJSON, &defaultTable,
		&chatbot.Enabled, &chatbot.MaxTokens, &chatbot.Temperature, &chatbot.ProviderID,
		&chatbot.PersistConversations, &chatbot.ConversationTTLHours, &chatbot.MaxConversationTurns,
		&chatbot.RateLimitPerMinute, &chatbot.DailyRequestLimit, &chatbot.DailyTokenBudget,
		&chatbot.AllowUnauthenticated, &chatbot.IsPublic, &chatbot.RequireRoles, &responseLanguage, &chatbot.DisableExecutionLogs,
		&chatbot.MCPTools, &chatbot.UseMCPSchema,
		&chatbot.Version, &chatbot.Source,
		&chatbot.CreatedBy, &chatbot.CreatedAt, &chatbot.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get chatbot by name: %w", err)
	}

	// Deserialize JSON fields
	if len(intentRulesJSON) > 0 {
		if err := json.Unmarshal(intentRulesJSON, &chatbot.IntentRules); err != nil {
			log.Warn().Err(err).Str("chatbot_name", name).Msg("Failed to unmarshal intent_rules")
		}
	}
	if len(requiredColumnsJSON) > 0 {
		if err := json.Unmarshal(requiredColumnsJSON, &chatbot.RequiredColumns); err != nil {
			log.Warn().Err(err).Str("chatbot_name", name).Msg("Failed to unmarshal required_columns")
		}
	}
	if defaultTable != nil {
		chatbot.DefaultTable = *defaultTable
	}
	if responseLanguage != nil {
		chatbot.ResponseLanguage = *responseLanguage
	}

	chatbot.PopulateDerivedFields()
	return chatbot, nil
}

// ListChatbots lists all chatbots with optional filtering
func (s *Storage) ListChatbots(ctx context.Context, enabledOnly bool) ([]*Chatbot, error) {
	query := `
		SELECT
			id, name, namespace, description, code, original_code, is_bundled, bundle_error,
			allowed_tables, allowed_operations, allowed_schemas, http_allowed_domains,
			intent_rules, required_columns, default_table,
			enabled, max_tokens, temperature, provider_id,
			persist_conversations, conversation_ttl_hours, max_conversation_turns,
			rate_limit_per_minute, daily_request_limit, daily_token_budget,
			allow_unauthenticated, is_public, require_roles, response_language, disable_execution_logs,
			mcp_tools, use_mcp_schema,
			version, source, created_by, created_at, updated_at
		FROM ai.chatbots
	`

	if enabledOnly {
		query += " WHERE enabled = true"
	}

	query += " ORDER BY namespace, name"

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list chatbots: %w", err)
	}
	defer rows.Close()

	var chatbots []*Chatbot
	for rows.Next() {
		chatbot := &Chatbot{}
		var intentRulesJSON, requiredColumnsJSON []byte
		var defaultTable *string
		var responseLanguage *string
		err := rows.Scan(
			&chatbot.ID, &chatbot.Name, &chatbot.Namespace, &chatbot.Description,
			&chatbot.Code, &chatbot.OriginalCode, &chatbot.IsBundled, &chatbot.BundleError,
			&chatbot.AllowedTables, &chatbot.AllowedOperations, &chatbot.AllowedSchemas, &chatbot.HTTPAllowedDomains,
			&intentRulesJSON, &requiredColumnsJSON, &defaultTable,
			&chatbot.Enabled, &chatbot.MaxTokens, &chatbot.Temperature, &chatbot.ProviderID,
			&chatbot.PersistConversations, &chatbot.ConversationTTLHours, &chatbot.MaxConversationTurns,
			&chatbot.RateLimitPerMinute, &chatbot.DailyRequestLimit, &chatbot.DailyTokenBudget,
			&chatbot.AllowUnauthenticated, &chatbot.IsPublic, &chatbot.RequireRoles, &responseLanguage, &chatbot.DisableExecutionLogs,
			&chatbot.MCPTools, &chatbot.UseMCPSchema,
			&chatbot.Version, &chatbot.Source,
			&chatbot.CreatedBy, &chatbot.CreatedAt, &chatbot.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chatbot row: %w", err)
		}

		// Deserialize JSON fields
		if len(intentRulesJSON) > 0 {
			_ = json.Unmarshal(intentRulesJSON, &chatbot.IntentRules)
		}
		if len(requiredColumnsJSON) > 0 {
			_ = json.Unmarshal(requiredColumnsJSON, &chatbot.RequiredColumns)
		}
		if defaultTable != nil {
			chatbot.DefaultTable = *defaultTable
		}
		if responseLanguage != nil {
			chatbot.ResponseLanguage = *responseLanguage
		}

		chatbot.PopulateDerivedFields()
		chatbots = append(chatbots, chatbot)
	}

	return chatbots, nil
}

// ListChatbotsByNamespace lists chatbots filtered by namespace
func (s *Storage) ListChatbotsByNamespace(ctx context.Context, namespace string) ([]*Chatbot, error) {
	query := `
		SELECT
			id, name, namespace, description, code, original_code, is_bundled, bundle_error,
			allowed_tables, allowed_operations, allowed_schemas, http_allowed_domains,
			intent_rules, required_columns, default_table,
			enabled, max_tokens, temperature, provider_id,
			persist_conversations, conversation_ttl_hours, max_conversation_turns,
			rate_limit_per_minute, daily_request_limit, daily_token_budget,
			allow_unauthenticated, is_public, require_roles, response_language, disable_execution_logs,
			mcp_tools, use_mcp_schema,
			version, source, created_by, created_at, updated_at
		FROM ai.chatbots
		WHERE namespace = $1
		ORDER BY name
	`

	rows, err := s.db.Query(ctx, query, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list chatbots by namespace: %w", err)
	}
	defer rows.Close()

	var chatbots []*Chatbot
	for rows.Next() {
		chatbot := &Chatbot{}
		var intentRulesJSON, requiredColumnsJSON []byte
		var defaultTable *string
		var responseLanguage *string
		err := rows.Scan(
			&chatbot.ID, &chatbot.Name, &chatbot.Namespace, &chatbot.Description,
			&chatbot.Code, &chatbot.OriginalCode, &chatbot.IsBundled, &chatbot.BundleError,
			&chatbot.AllowedTables, &chatbot.AllowedOperations, &chatbot.AllowedSchemas, &chatbot.HTTPAllowedDomains,
			&intentRulesJSON, &requiredColumnsJSON, &defaultTable,
			&chatbot.Enabled, &chatbot.MaxTokens, &chatbot.Temperature, &chatbot.ProviderID,
			&chatbot.PersistConversations, &chatbot.ConversationTTLHours, &chatbot.MaxConversationTurns,
			&chatbot.RateLimitPerMinute, &chatbot.DailyRequestLimit, &chatbot.DailyTokenBudget,
			&chatbot.AllowUnauthenticated, &chatbot.IsPublic, &chatbot.RequireRoles, &responseLanguage, &chatbot.DisableExecutionLogs,
			&chatbot.MCPTools, &chatbot.UseMCPSchema,
			&chatbot.Version, &chatbot.Source,
			&chatbot.CreatedBy, &chatbot.CreatedAt, &chatbot.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chatbot row: %w", err)
		}

		// Deserialize JSON fields
		if len(intentRulesJSON) > 0 {
			_ = json.Unmarshal(intentRulesJSON, &chatbot.IntentRules)
		}
		if len(requiredColumnsJSON) > 0 {
			_ = json.Unmarshal(requiredColumnsJSON, &chatbot.RequiredColumns)
		}
		if defaultTable != nil {
			chatbot.DefaultTable = *defaultTable
		}
		if responseLanguage != nil {
			chatbot.ResponseLanguage = *responseLanguage
		}

		chatbot.PopulateDerivedFields()
		chatbots = append(chatbots, chatbot)
	}

	return chatbots, nil
}

// DeleteChatbot deletes a chatbot by ID
func (s *Storage) DeleteChatbot(ctx context.Context, id string) error {
	query := `DELETE FROM ai.chatbots WHERE id = $1`

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete chatbot: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("chatbot not found: %s", id)
	}

	log.Info().Str("id", id).Msg("Deleted chatbot")

	return nil
}

// UpsertChatbot creates or updates a chatbot based on namespace and name
func (s *Storage) UpsertChatbot(ctx context.Context, chatbot *Chatbot) error {
	// Check if chatbot exists
	existing, err := s.GetChatbotByName(ctx, chatbot.Namespace, chatbot.Name)
	if err != nil {
		return err
	}

	if existing != nil {
		// Update existing
		chatbot.ID = existing.ID
		chatbot.CreatedAt = existing.CreatedAt
		chatbot.CreatedBy = existing.CreatedBy
		return s.UpdateChatbot(ctx, chatbot)
	}

	// Create new
	return s.CreateChatbot(ctx, chatbot)
}

// ============================================================================
// PROVIDER OPERATIONS
// ============================================================================

// ProviderRecord represents a provider in the database
type ProviderRecord struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	DisplayName      string            `json:"display_name"`
	ProviderType     string            `json:"provider_type"`
	IsDefault        bool              `json:"is_default"`
	UseForEmbeddings *bool             `json:"use_for_embeddings"` // Pointer to distinguish null (auto) from false
	EmbeddingModel   *string           `json:"embedding_model"`    // Embedding model for this provider (null = provider default)
	Config           map[string]string `json:"config"`
	Enabled          bool              `json:"enabled"`
	ReadOnly         bool              `json:"read_only"` // True if configured via environment/YAML (cannot be modified)
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	CreatedBy        *string           `json:"created_by,omitempty"`
}

// CreateProvider creates a new AI provider
func (s *Storage) CreateProvider(ctx context.Context, provider *ProviderRecord) error {
	query := `
		INSERT INTO ai.providers (
			id, name, display_name, provider_type, is_default, use_for_embeddings, embedding_model, config, enabled, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`

	if provider.ID == "" {
		provider.ID = uuid.New().String()
	}
	if provider.CreatedAt.IsZero() {
		provider.CreatedAt = time.Now()
	}
	provider.UpdatedAt = time.Now()

	_, err := s.db.Exec(ctx, query,
		provider.ID, provider.Name, provider.DisplayName, provider.ProviderType,
		provider.IsDefault, provider.UseForEmbeddings, provider.EmbeddingModel, provider.Config, provider.Enabled, provider.CreatedBy,
		provider.CreatedAt, provider.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	log.Info().
		Str("id", provider.ID).
		Str("name", provider.Name).
		Str("type", provider.ProviderType).
		Msg("Created AI provider")

	return nil
}

// UpdateProvider updates an existing AI provider
func (s *Storage) UpdateProvider(ctx context.Context, provider *ProviderRecord) error {
	query := `
		UPDATE ai.providers SET
			display_name = $2,
			config = $3,
			enabled = $4,
			use_for_embeddings = $5,
			embedding_model = $6,
			updated_at = $7
		WHERE id = $1
	`

	provider.UpdatedAt = time.Now()

	result, err := s.db.Exec(ctx, query,
		provider.ID,
		provider.DisplayName,
		provider.Config,
		provider.Enabled,
		provider.UseForEmbeddings,
		provider.EmbeddingModel,
		provider.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update provider: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("provider not found: %s", provider.ID)
	}

	log.Info().
		Str("id", provider.ID).
		Str("display_name", provider.DisplayName).
		Msg("Updated AI provider")

	return nil
}

// GetProvider retrieves a provider by ID
func (s *Storage) GetProvider(ctx context.Context, id string) (*ProviderRecord, error) {
	query := `
		SELECT id, name, display_name, provider_type, is_default, use_for_embeddings, embedding_model, config, enabled, created_by, created_at, updated_at
		FROM ai.providers
		WHERE id = $1
	`

	provider := &ProviderRecord{}
	err := s.db.QueryRow(ctx, query, id).Scan(
		&provider.ID, &provider.Name, &provider.DisplayName, &provider.ProviderType,
		&provider.IsDefault, &provider.UseForEmbeddings, &provider.EmbeddingModel, &provider.Config, &provider.Enabled, &provider.CreatedBy,
		&provider.CreatedAt, &provider.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return provider, nil
}

// GetProviderByName retrieves a provider by name
func (s *Storage) GetProviderByName(ctx context.Context, name string) (*ProviderRecord, error) {
	// First check if it's a config-based provider
	if s.config != nil && s.config.ProviderType != "" {
		configProvider := s.buildConfigBasedProvider()
		if configProvider != nil && configProvider.Name == name {
			return configProvider, nil
		}
	}

	query := `
		SELECT id, name, display_name, provider_type, is_default, use_for_embeddings, embedding_model, config, enabled, created_by, created_at, updated_at
		FROM ai.providers
		WHERE name = $1 AND enabled = true
	`

	provider := &ProviderRecord{}
	err := s.db.QueryRow(ctx, query, name).Scan(
		&provider.ID, &provider.Name, &provider.DisplayName, &provider.ProviderType,
		&provider.IsDefault, &provider.UseForEmbeddings, &provider.EmbeddingModel, &provider.Config, &provider.Enabled, &provider.CreatedBy,
		&provider.CreatedAt, &provider.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get provider by name: %w", err)
	}

	return provider, nil
}

// GetDefaultProvider retrieves the default provider
func (s *Storage) GetDefaultProvider(ctx context.Context) (*ProviderRecord, error) {
	query := `
		SELECT id, name, display_name, provider_type, is_default, use_for_embeddings, embedding_model, config, enabled, created_by, created_at, updated_at
		FROM ai.providers
		WHERE is_default = true AND enabled = true
		LIMIT 1
	`

	provider := &ProviderRecord{}
	err := s.db.QueryRow(ctx, query).Scan(
		&provider.ID, &provider.Name, &provider.DisplayName, &provider.ProviderType,
		&provider.IsDefault, &provider.UseForEmbeddings, &provider.EmbeddingModel, &provider.Config, &provider.Enabled, &provider.CreatedBy,
		&provider.CreatedAt, &provider.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get default provider: %w", err)
	}

	return provider, nil
}

// GetEffectiveDefaultProvider retrieves the effective default provider
// Checks config-based provider first, then falls back to database
func (s *Storage) GetEffectiveDefaultProvider(ctx context.Context) (*ProviderRecord, error) {
	// Check if config-based provider is set (enabled is inferred from ProviderType being set)
	if s.config != nil && s.config.ProviderType != "" {
		provider := s.buildConfigBasedProvider()
		if provider != nil {
			return provider, nil
		}
	}

	// Fallback to database provider
	return s.GetDefaultProvider(ctx)
}

// buildConfigBasedProvider constructs a ProviderRecord from config
// A config-based provider is enabled if ProviderType is set
func (s *Storage) buildConfigBasedProvider() *ProviderRecord {
	if s.config == nil {
		log.Debug().Msg("buildConfigBasedProvider: config is nil")
		return nil
	}

	providerType := s.config.ProviderType
	if providerType == "" {
		log.Debug().Msg("buildConfigBasedProvider: provider type is empty")
		return nil
	}

	log.Debug().
		Str("provider_type", providerType).
		Str("provider_name", s.config.ProviderName).
		Str("provider_model", s.config.ProviderModel).
		Msg("buildConfigBasedProvider: building config-based provider")

	// Build config map based on provider type
	configMap := make(map[string]string)

	switch providerType {
	case "openai":
		if s.config.OpenAIAPIKey == "" {
			log.Error().
				Str("provider_type", "openai").
				Str("required_env_var", "FLUXBASE_AI_OPENAI_API_KEY").
				Msg("OpenAI provider enabled but FLUXBASE_AI_OPENAI_API_KEY is not set. Provider will NOT appear in the list")
			return nil
		}
		log.Debug().Msg("buildConfigBasedProvider: OpenAI provider configured")
		configMap["api_key"] = s.config.OpenAIAPIKey
		if s.config.OpenAIOrganizationID != "" {
			configMap["organization_id"] = s.config.OpenAIOrganizationID
		}
		if s.config.OpenAIBaseURL != "" {
			configMap["base_url"] = s.config.OpenAIBaseURL
		}

	case "azure":
		if s.config.AzureAPIKey == "" || s.config.AzureEndpoint == "" || s.config.AzureDeploymentName == "" {
			var missing []string
			if s.config.AzureAPIKey == "" {
				missing = append(missing, "FLUXBASE_AI_AZURE_API_KEY")
			}
			if s.config.AzureEndpoint == "" {
				missing = append(missing, "FLUXBASE_AI_AZURE_ENDPOINT")
			}
			if s.config.AzureDeploymentName == "" {
				missing = append(missing, "FLUXBASE_AI_AZURE_DEPLOYMENT_NAME")
			}
			log.Error().
				Str("provider_type", "azure").
				Strs("missing_env_vars", missing).
				Msg("Azure provider enabled but required environment variables are not set. Provider will NOT appear in the list")
			return nil
		}
		configMap["api_key"] = s.config.AzureAPIKey
		configMap["endpoint"] = s.config.AzureEndpoint
		configMap["deployment_name"] = s.config.AzureDeploymentName
		if s.config.AzureAPIVersion != "" {
			configMap["api_version"] = s.config.AzureAPIVersion
		} else {
			configMap["api_version"] = "2024-02-15-preview"
		}

	case "ollama":
		if s.config.OllamaModel == "" {
			log.Error().
				Str("provider_type", "ollama").
				Str("required_env_var", "FLUXBASE_AI_OLLAMA_MODEL").
				Msg("Ollama provider enabled but FLUXBASE_AI_OLLAMA_MODEL is not set. Provider will NOT appear in the list. Set this env var (e.g., llama2, mistral, codellama)")
			return nil
		}
		if s.config.OllamaEndpoint != "" {
			configMap["endpoint"] = s.config.OllamaEndpoint
		} else {
			configMap["endpoint"] = "http://localhost:11434"
		}

	default:
		log.Warn().Str("provider_type", providerType).Msg("Unknown provider type in config")
		return nil
	}

	// Determine display name
	displayName := s.config.ProviderName
	if displayName == "" {
		displayName = "Config Provider (" + providerType + ")"
	}

	// Determine model
	model := s.config.ProviderModel
	if model == "" {
		switch providerType {
		case "openai":
			model = "gpt-4-turbo"
		case "ollama":
			model = s.config.OllamaModel
		}
	}

	// Add model to config map if set
	if model != "" {
		configMap["model"] = model
	}

	provider := &ProviderRecord{
		ID:           "FROM_CONFIG",
		Name:         "config",
		DisplayName:  displayName,
		ProviderType: providerType,
		IsDefault:    true,
		Config:       configMap,
		Enabled:      true,
		ReadOnly:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatedBy:    nil,
	}

	log.Info().
		Str("id", provider.ID).
		Str("display_name", provider.DisplayName).
		Str("provider_type", provider.ProviderType).
		Str("model", model).
		Bool("is_default", provider.IsDefault).
		Bool("read_only", provider.ReadOnly).
		Msg("Config-based AI provider created successfully")

	return provider
}

// ListProviders lists all AI providers
func (s *Storage) ListProviders(ctx context.Context, enabledOnly bool) ([]*ProviderRecord, error) {
	var providers []*ProviderRecord

	// Check if config-based provider exists and should be included
	configProvider := s.buildConfigBasedProvider()
	if configProvider != nil && (!enabledOnly || configProvider.Enabled) {
		providers = append(providers, configProvider)
	}

	// Query database providers
	query := `
		SELECT id, name, display_name, provider_type, is_default, use_for_embeddings, embedding_model, config, enabled, created_by, created_at, updated_at
		FROM ai.providers
	`

	if enabledOnly {
		query += " WHERE enabled = true"
	}

	query += " ORDER BY is_default DESC, name"

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		provider := &ProviderRecord{}
		err := rows.Scan(
			&provider.ID, &provider.Name, &provider.DisplayName, &provider.ProviderType,
			&provider.IsDefault, &provider.UseForEmbeddings, &provider.EmbeddingModel, &provider.Config, &provider.Enabled, &provider.CreatedBy,
			&provider.CreatedAt, &provider.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan provider row: %w", err)
		}
		// Set ReadOnly to false for database providers
		provider.ReadOnly = false
		providers = append(providers, provider)
	}

	return providers, nil
}

// SetDefaultProvider sets a provider as the default
func (s *Storage) SetDefaultProvider(ctx context.Context, id string) error {
	// Use transaction to ensure atomicity
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Clear existing default
	_, err = tx.Exec(ctx, "UPDATE ai.providers SET is_default = false WHERE is_default = true")
	if err != nil {
		return fmt.Errorf("failed to clear default: %w", err)
	}

	// Set new default
	result, err := tx.Exec(ctx, "UPDATE ai.providers SET is_default = true WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to set default: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("provider not found: %s", id)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info().Str("id", id).Msg("Set default AI provider")

	return nil
}

// DeleteProvider deletes a provider by ID
func (s *Storage) DeleteProvider(ctx context.Context, id string) error {
	query := `DELETE FROM ai.providers WHERE id = $1`

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("provider not found: %s", id)
	}

	log.Info().Str("id", id).Msg("Deleted AI provider")

	return nil
}

// GetEmbeddingProviderPreference returns the provider explicitly set for embeddings (if any)
// Returns nil if no explicit preference is set (use default provider in auto mode)
func (s *Storage) GetEmbeddingProviderPreference(ctx context.Context) (*ProviderRecord, error) {
	query := `
		SELECT id, name, display_name, provider_type, is_default, use_for_embeddings, embedding_model, config, enabled, created_by, created_at, updated_at
		FROM ai.providers
		WHERE use_for_embeddings = true AND enabled = true
		LIMIT 1
	`

	provider := &ProviderRecord{}
	err := s.db.QueryRow(ctx, query).Scan(
		&provider.ID, &provider.Name, &provider.DisplayName, &provider.ProviderType,
		&provider.IsDefault, &provider.UseForEmbeddings, &provider.EmbeddingModel, &provider.Config, &provider.Enabled, &provider.CreatedBy,
		&provider.CreatedAt, &provider.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		// No explicit preference set - return nil without error (auto mode)
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding provider preference: %w", err)
	}

	// Mark as read-only=false since it's from database
	provider.ReadOnly = false

	return provider, nil
}

// SetEmbeddingProviderPreference sets a provider as the embedding provider
// Pass empty id to clear preference (revert to auto/default mode)
func (s *Storage) SetEmbeddingProviderPreference(ctx context.Context, id string) error {
	// Use transaction to ensure atomicity
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Clear any existing embedding preference (set to NULL for auto mode)
	_, err = tx.Exec(ctx, "UPDATE ai.providers SET use_for_embeddings = NULL WHERE use_for_embeddings = true")
	if err != nil {
		return fmt.Errorf("failed to clear embedding preference: %w", err)
	}

	// If id provided, set it as embedding provider (cannot set read-only providers)
	if id != "" {
		result, err := tx.Exec(ctx, `
			UPDATE ai.providers
			SET use_for_embeddings = true
			WHERE id = $1 AND read_only = false
		`, id)
		if err != nil {
			return fmt.Errorf("failed to set embedding provider: %w", err)
		}

		if result.RowsAffected() == 0 {
			return fmt.Errorf("provider not found or is read-only: %s", id)
		}

		log.Info().Str("id", id).Msg("Set embedding provider preference")
	} else {
		log.Info().Msg("Cleared embedding provider preference (reverted to auto mode)")
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ============================================================================
// USER CONVERSATION OPERATIONS
// ============================================================================

// UserConversationSummary represents a conversation for the user API (not admin)
type UserConversationSummary struct {
	ID           string    `json:"id"`
	ChatbotName  string    `json:"chatbot"`
	Namespace    string    `json:"namespace"`
	Title        *string   `json:"title"`
	Preview      string    `json:"preview"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserMessageDetail represents a message in user API response
type UserMessageDetail struct {
	ID           string            `json:"id"`
	Role         string            `json:"role"`
	Content      string            `json:"content"`
	Timestamp    time.Time         `json:"timestamp"`
	QueryResults []UserQueryResult `json:"query_results,omitempty"` // Array of query results for assistant messages
	Usage        *UserUsageStats   `json:"usage,omitempty"`
}

// UserQueryResult represents SQL query results for user API
type UserQueryResult struct {
	Query    string                   `json:"query"`
	Summary  string                   `json:"summary"`
	RowCount int                      `json:"row_count"`
	Data     []map[string]interface{} `json:"data,omitempty"`
}

// UserUsageStats represents token usage for user API
type UserUsageStats struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// UserConversationDetail represents a full conversation with messages for user API
type UserConversationDetail struct {
	ID          string              `json:"id"`
	ChatbotName string              `json:"chatbot"`
	Namespace   string              `json:"namespace"`
	Title       *string             `json:"title"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	Messages    []UserMessageDetail `json:"messages"`
}

// ListUserConversationsOptions contains options for listing user conversations
type ListUserConversationsOptions struct {
	UserID      string
	ChatbotName *string
	Namespace   *string
	Limit       int
	Offset      int
}

// ListUserConversationsResult contains the result of listing user conversations
type ListUserConversationsResult struct {
	Conversations []UserConversationSummary `json:"conversations"`
	Total         int                       `json:"total"`
	HasMore       bool                      `json:"has_more"`
}

// ListUserConversations lists conversations for a specific user with pagination
func (s *Storage) ListUserConversations(ctx context.Context, opts ListUserConversationsOptions) (*ListUserConversationsResult, error) {
	// Build the main query with CTEs for preview and message count
	query := `
		WITH conv_preview AS (
			SELECT DISTINCT ON (m.conversation_id)
				m.conversation_id,
				LEFT(m.content, 50) AS preview
			FROM ai.messages m
			WHERE m.role = 'user'
			ORDER BY m.conversation_id, m.sequence_number ASC
		),
		conv_count AS (
			SELECT conversation_id, COUNT(*) AS message_count
			FROM ai.messages
			GROUP BY conversation_id
		)
		SELECT
			c.id,
			cb.name AS chatbot_name,
			cb.namespace,
			c.title,
			COALESCE(cp.preview, '') AS preview,
			COALESCE(cc.message_count, 0) AS message_count,
			c.created_at,
			c.updated_at
		FROM ai.conversations c
		LEFT JOIN ai.chatbots cb ON cb.id = c.chatbot_id
		LEFT JOIN conv_preview cp ON cp.conversation_id = c.id
		LEFT JOIN conv_count cc ON cc.conversation_id = c.id
		WHERE c.user_id = $1 AND c.status = 'active'
	`

	args := []interface{}{opts.UserID}
	argIndex := 2

	if opts.ChatbotName != nil {
		query += fmt.Sprintf(" AND cb.name = $%d", argIndex)
		args = append(args, *opts.ChatbotName)
		argIndex++
	}

	if opts.Namespace != nil {
		query += fmt.Sprintf(" AND cb.namespace = $%d", argIndex)
		args = append(args, *opts.Namespace)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY c.updated_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, opts.Limit, opts.Offset)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list user conversations: %w", err)
	}
	defer rows.Close()

	var conversations []UserConversationSummary
	for rows.Next() {
		var conv UserConversationSummary
		err := rows.Scan(
			&conv.ID,
			&conv.ChatbotName,
			&conv.Namespace,
			&conv.Title,
			&conv.Preview,
			&conv.MessageCount,
			&conv.CreatedAt,
			&conv.UpdatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan conversation")
			continue
		}
		conversations = append(conversations, conv)
	}

	// Get total count
	countQuery := `
		SELECT COUNT(*)
		FROM ai.conversations c
		LEFT JOIN ai.chatbots cb ON cb.id = c.chatbot_id
		WHERE c.user_id = $1 AND c.status = 'active'
	`
	countArgs := []interface{}{opts.UserID}
	countArgIndex := 2

	if opts.ChatbotName != nil {
		countQuery += fmt.Sprintf(" AND cb.name = $%d", countArgIndex)
		countArgs = append(countArgs, *opts.ChatbotName)
		countArgIndex++
	}

	if opts.Namespace != nil {
		countQuery += fmt.Sprintf(" AND cb.namespace = $%d", countArgIndex)
		countArgs = append(countArgs, *opts.Namespace)
	}

	var total int
	err = s.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get conversation count")
		total = len(conversations)
	}

	// Ensure conversations is not nil
	if conversations == nil {
		conversations = []UserConversationSummary{}
	}

	return &ListUserConversationsResult{
		Conversations: conversations,
		Total:         total,
		HasMore:       opts.Offset+len(conversations) < total,
	}, nil
}

// GetUserConversation retrieves a single conversation with messages for a user
func (s *Storage) GetUserConversation(ctx context.Context, userID, conversationID string) (*UserConversationDetail, error) {
	// Get conversation details
	query := `
		SELECT
			c.id,
			cb.name AS chatbot_name,
			cb.namespace,
			c.title,
			c.created_at,
			c.updated_at
		FROM ai.conversations c
		LEFT JOIN ai.chatbots cb ON cb.id = c.chatbot_id
		WHERE c.id = $1 AND c.user_id = $2 AND c.status = 'active'
	`

	var conv UserConversationDetail
	err := s.db.QueryRow(ctx, query, conversationID, userID).Scan(
		&conv.ID,
		&conv.ChatbotName,
		&conv.Namespace,
		&conv.Title,
		&conv.CreatedAt,
		&conv.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// Get messages
	msgQuery := `
		SELECT
			id,
			role,
			content,
			query_results,
			executed_sql,
			sql_result_summary,
			sql_row_count,
			prompt_tokens,
			completion_tokens,
			created_at
		FROM ai.messages
		WHERE conversation_id = $1
		ORDER BY sequence_number ASC
	`

	rows, err := s.db.Query(ctx, msgQuery, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []UserMessageDetail
	for rows.Next() {
		var msg UserMessageDetail
		var queryResultsJSON []byte
		var executedSQL *string
		var sqlSummary *string
		var sqlRowCount *int
		var promptTokens *int
		var completionTokens *int

		err := rows.Scan(
			&msg.ID,
			&msg.Role,
			&msg.Content,
			&queryResultsJSON,
			&executedSQL,
			&sqlSummary,
			&sqlRowCount,
			&promptTokens,
			&completionTokens,
			&msg.Timestamp,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan message")
			continue
		}

		// Parse query_results JSONB if present (new format with full data)
		if len(queryResultsJSON) > 0 {
			var queryResults []UserQueryResult
			if err := json.Unmarshal(queryResultsJSON, &queryResults); err != nil {
				log.Warn().Err(err).Msg("Failed to parse query_results JSON")
			} else if len(queryResults) > 0 {
				msg.QueryResults = queryResults
			}
		}

		// Fallback to legacy fields if no query_results (for backward compatibility)
		if msg.QueryResults == nil && (sqlSummary != nil && *sqlSummary != "") {
			legacyResult := UserQueryResult{
				Summary:  *sqlSummary,
				RowCount: 0,
			}
			if executedSQL != nil {
				legacyResult.Query = *executedSQL
			}
			if sqlRowCount != nil {
				legacyResult.RowCount = *sqlRowCount
			}
			msg.QueryResults = []UserQueryResult{legacyResult}
		}

		// Add usage stats if present
		if promptTokens != nil || completionTokens != nil {
			msg.Usage = &UserUsageStats{}
			if promptTokens != nil {
				msg.Usage.PromptTokens = *promptTokens
			}
			if completionTokens != nil {
				msg.Usage.CompletionTokens = *completionTokens
			}
			msg.Usage.TotalTokens = msg.Usage.PromptTokens + msg.Usage.CompletionTokens
		}

		messages = append(messages, msg)
	}

	// Ensure messages is not nil
	if messages == nil {
		messages = []UserMessageDetail{}
	}

	conv.Messages = messages
	return &conv, nil
}

// DeleteUserConversation soft-deletes a conversation owned by the user
func (s *Storage) DeleteUserConversation(ctx context.Context, userID, conversationID string) error {
	query := `
		UPDATE ai.conversations
		SET status = 'deleted', updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status = 'active'
	`

	result, err := s.db.Exec(ctx, query, conversationID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found")
	}

	log.Info().
		Str("conversation_id", conversationID).
		Str("user_id", userID).
		Msg("Deleted user conversation")

	return nil
}

// UpdateConversationTitle updates the title of a conversation owned by the user
func (s *Storage) UpdateConversationTitle(ctx context.Context, userID, conversationID, title string) error {
	query := `
		UPDATE ai.conversations
		SET title = $3, updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status = 'active'
	`

	result, err := s.db.Exec(ctx, query, conversationID, userID, title)
	if err != nil {
		return fmt.Errorf("failed to update conversation title: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found")
	}

	log.Info().
		Str("conversation_id", conversationID).
		Str("title", title).
		Msg("Updated conversation title")

	return nil
}

// SetConversationTitle sets the title of a conversation (internal use, no ownership check)
func (s *Storage) SetConversationTitle(ctx context.Context, conversationID, title string) error {
	query := `
		UPDATE ai.conversations
		SET title = $2, updated_at = NOW()
		WHERE id = $1 AND title IS NULL
	`

	_, err := s.db.Exec(ctx, query, conversationID, title)
	if err != nil {
		return fmt.Errorf("failed to set conversation title: %w", err)
	}

	return nil
}
