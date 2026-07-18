package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
)

// Storage handles database operations for AI entities
type Storage struct {
	database.TenantAware
	db     *database.Connection
	config *config.AIConfig
}

// NewStorage creates a new AI storage instance
func NewStorage(db *database.Connection) *Storage {
	return &Storage{
		TenantAware: database.TenantAware{DB: db},
		db:          db,
		config:      nil,
	}
}

// SetConfig sets the AI configuration for the storage
func (s *Storage) SetConfig(cfg *config.AIConfig) {
	s.config = cfg
}

// UserExists checks if a user exists in auth.users
func (s *Storage) UserExists(ctx context.Context, userID string) (bool, error) {
	var exists bool
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		query := `SELECT EXISTS(SELECT 1 FROM auth.users WHERE id = $1)`
		return tx.QueryRow(ctx, query, userID).Scan(&exists)
	})
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return exists, nil
}

// ============================================================================
// CHATBOT OPERATIONS
// ============================================================================

// CreateChatbot creates a new chatbot in the database
// Deprecated: Use CreateChatbotWithTenant for tenant-scoped operations
func (s *Storage) CreateChatbot(ctx context.Context, chatbot *Chatbot) error {
	// Resolve tenant from context (matches CreateProcedure's pattern) so the
	// INSERT writes tenant_id explicitly. Previously this hardcoded "" which
	// produced rows with NULL tenant_id when the role wrapper's GUC was not
	// set — see CreateChatbotWithTenant for the full explanation.
	tenantID := database.TenantFromContext(ctx)
	return s.CreateChatbotWithTenant(ctx, tenantID, chatbot)
}

// CreateChatbotWithTenant creates a new chatbot in the database with tenant context
func (s *Storage) CreateChatbotWithTenant(ctx context.Context, tenantID string, chatbot *Chatbot) error {
	// IMPORTANT: include tenant_id in the INSERT column list explicitly.
	// The chatbots table has a BEFORE INSERT trigger
	// (auth.set_tenant_id_from_context) that auto-populates tenant_id from
	// the app.current_tenant_id GUC when NEW.tenant_id IS NULL. But that
	// GUC is only set by WrapWithTenantAwareRole when tenantID != "". When
	// callers pass empty string (which the deprecated CreateChatbot wrapper
	// used to do unconditionally), the trigger leaves tenant_id NULL and
	// the row becomes invisible to tenant-scoped list queries, breaking
	// subsequent syncs (the procedure oscillation bug).
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
			version, source, created_by, created_at, updated_at, tenant_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15,
			$16, $17, $18, $19,
			$20, $21, $22,
			$23, $24, $25,
			$26, $27, $28, $29, $30,
			$31, $32,
			$33, $34, $35, $36, $37, $38
		)
	`

	if chatbot.ID == "" {
		chatbot.ID = uuid.New().String()
	}
	if chatbot.CreatedAt.IsZero() {
		chatbot.CreatedAt = time.Now()
	}
	chatbot.UpdatedAt = time.Now()

	// Resolve tenant ID: prefer the explicit parameter; fall back to context.
	resolvedTenant := tenantID
	if resolvedTenant == "" {
		resolvedTenant = database.TenantFromContext(ctx)
	}

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

	err = database.WrapWithTenantAwareRole(ctx, s.db, resolvedTenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(
			ctx, query,
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
			database.TenantOrNil(resolvedTenant),
		)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to create chatbot: %w", err)
	}

	log.Info().
		Str("id", chatbot.ID).
		Str("name", chatbot.Name).
		Str("namespace", chatbot.Namespace).
		Str("tenant_id", resolvedTenant).
		Msg("Created chatbot")

	return nil
}

// UpdateChatbot updates an existing chatbot in the database
func (s *Storage) UpdateChatbot(ctx context.Context, chatbot *Chatbot) error {
	// Get tenant ID from context for backward compatibility
	tenantID := database.TenantFromContext(ctx)
	return s.UpdateChatbotWithTenant(ctx, tenantID, chatbot)
}

// UpdateChatbotWithTenant updates an existing chatbot in the database with explicit tenant context
func (s *Storage) UpdateChatbotWithTenant(ctx context.Context, tenantID string, chatbot *Chatbot) error {
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

	var result pgconn.CommandTag
	err = database.WrapWithTenantAwareRole(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(
			ctx, query,
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
		return execErr
	})
	if err != nil {
		return fmt.Errorf("failed to update chatbot: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("chatbot not found: %s", chatbot.ID)
	}

	log.Info().
		Str("id", chatbot.ID).
		Str("name", chatbot.Name).
		Str("tenant_id", tenantID).
		Msg("Updated chatbot")

	return nil
}

// GetChatbot retrieves a chatbot by ID
func (s *Storage) GetChatbot(ctx context.Context, id string) (*Chatbot, error) {
	chatbot := &Chatbot{}
	var intentRulesJSON, requiredColumnsJSON []byte
	var defaultTable *string
	var responseLanguage *string
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		tenantID := database.TenantFromContext(ctx)
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
			WHERE id = $1 AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
		`
		return tx.QueryRow(ctx, query, id, database.TenantOrNil(tenantID)).Scan(
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
	})

	if errors.Is(err, pgx.ErrNoRows) {
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
	chatbot := &Chatbot{}
	var intentRulesJSON, requiredColumnsJSON []byte
	var defaultTable *string
	var responseLanguage *string
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		tenantID := database.TenantFromContext(ctx)
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
			WHERE namespace = $1 AND name = $2 AND (tenant_id = $3 OR ($3 IS NULL AND tenant_id IS NULL))
		`
		return tx.QueryRow(ctx, query, namespace, name, database.TenantOrNil(tenantID)).Scan(
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
	})

	if errors.Is(err, pgx.ErrNoRows) {
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
	var chatbots []*Chatbot
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		tenantID := database.TenantFromContext(ctx)
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
			WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
		`

		args := []interface{}{database.TenantOrNil(tenantID)}

		if enabledOnly {
			query += " AND enabled = true"
		}

		query += " ORDER BY namespace, name"

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to list chatbots: %w", err)
		}
		defer rows.Close()

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
				return fmt.Errorf("failed to scan chatbot row: %w", err)
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

		return nil
	})
	if err != nil {
		return nil, err
	}

	return chatbots, nil
}

// ListChatbotsByNamespace lists chatbots filtered by namespace
func (s *Storage) ListChatbotsByNamespace(ctx context.Context, namespace string) ([]*Chatbot, error) {
	var chatbots []*Chatbot
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		tenantID := database.TenantFromContext(ctx)
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
			WHERE namespace = $1 AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
			ORDER BY name
		`

		rows, err := tx.Query(ctx, query, namespace, database.TenantOrNil(tenantID))
		if err != nil {
			return fmt.Errorf("failed to list chatbots by namespace: %w", err)
		}
		defer rows.Close()

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
				return fmt.Errorf("failed to scan chatbot row: %w", err)
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

		return nil
	})
	if err != nil {
		return nil, err
	}

	return chatbots, nil
}

// FindChatbotsByName finds all chatbots with the given name across all namespaces
// Returns multiple chatbots if the name exists in multiple namespaces
// Used for smart chatbot lookup when namespace is not specified
func (s *Storage) FindChatbotsByName(ctx context.Context, name string, enabledOnly bool) ([]*Chatbot, error) {
	var chatbots []*Chatbot
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		tenantID := database.TenantFromContext(ctx)
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
			WHERE name = $1 AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
		`

		args := []interface{}{name, database.TenantOrNil(tenantID)}

		if enabledOnly {
			query += " AND enabled = true"
		}

		query += " ORDER BY namespace"

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to find chatbots by name: %w", err)
		}
		defer rows.Close()

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
				return fmt.Errorf("failed to scan chatbot row: %w", err)
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

		return nil
	})
	if err != nil {
		return nil, err
	}

	return chatbots, nil
}

// DeleteChatbot deletes a chatbot by ID
// Deprecated: Use DeleteChatbotWithTenant for tenant-scoped operations
func (s *Storage) DeleteChatbot(ctx context.Context, id string) error {
	tenantID := database.TenantFromContext(ctx)
	return s.DeleteChatbotWithTenant(ctx, tenantID, id)
}

// DeleteChatbotWithTenant deletes a chatbot by ID with explicit tenant context
func (s *Storage) DeleteChatbotWithTenant(ctx context.Context, tenantID string, id string) error {
	query := `DELETE FROM ai.chatbots WHERE id = $1`

	var result pgconn.CommandTag
	err := database.WrapWithTenantAwareRole(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(ctx, query, id)
		return execErr
	})
	if err != nil {
		return fmt.Errorf("failed to delete chatbot: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("chatbot not found: %s", id)
	}

	log.Info().Str("id", id).Str("tenant_id", tenantID).Msg("Deleted chatbot")

	return nil
}

// UpsertChatbot creates or updates a chatbot based on namespace and name
func (s *Storage) UpsertChatbot(ctx context.Context, chatbot *Chatbot) error {
	tenantID := database.TenantFromContext(ctx)
	return s.UpsertChatbotWithTenant(ctx, tenantID, chatbot)
}

// UpsertChatbotWithTenant creates or updates a chatbot based on namespace and name with tenant context
func (s *Storage) UpsertChatbotWithTenant(ctx context.Context, tenantID string, chatbot *Chatbot) error {
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
		return s.UpdateChatbotWithTenant(ctx, tenantID, chatbot)
	}

	// Create new
	return s.CreateChatbotWithTenant(ctx, tenantID, chatbot)
}
