package ai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

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
// Deprecated: Use CreateProviderWithTenant for tenant-scoped operations
func (s *Storage) CreateProvider(ctx context.Context, provider *ProviderRecord) error {
	tenantID := database.TenantFromContext(ctx)
	return s.CreateProviderWithTenant(ctx, tenantID, provider)
}

// CreateProviderWithTenant creates a new AI provider with tenant context
func (s *Storage) CreateProviderWithTenant(ctx context.Context, tenantID string, provider *ProviderRecord) error {
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

	err := database.WrapWithTenantAwareRole(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(
			ctx, query,
			provider.ID, provider.Name, provider.DisplayName, provider.ProviderType,
			provider.IsDefault, provider.UseForEmbeddings, provider.EmbeddingModel, provider.Config, provider.Enabled, provider.CreatedBy,
			provider.CreatedAt, provider.UpdatedAt,
		)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	log.Info().
		Str("id", provider.ID).
		Str("name", provider.Name).
		Str("type", provider.ProviderType).
		Str("tenant_id", tenantID).
		Msg("Created AI provider")

	return nil
}

// UpdateProvider updates an existing AI provider
// Deprecated: Use UpdateProviderWithTenant for tenant-scoped operations
func (s *Storage) UpdateProvider(ctx context.Context, provider *ProviderRecord) error {
	tenantID := database.TenantFromContext(ctx)
	return s.UpdateProviderWithTenant(ctx, tenantID, provider)
}

// UpdateProviderWithTenant updates an existing AI provider with tenant context
func (s *Storage) UpdateProviderWithTenant(ctx context.Context, tenantID string, provider *ProviderRecord) error {
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

	var result pgconn.CommandTag
	err := database.WrapWithTenantAwareRole(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(
			ctx, query,
			provider.ID,
			provider.DisplayName,
			provider.Config,
			provider.Enabled,
			provider.UseForEmbeddings,
			provider.EmbeddingModel,
			provider.UpdatedAt,
		)
		return execErr
	})
	if err != nil {
		return fmt.Errorf("failed to update provider: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("provider not found: %s", provider.ID)
	}

	log.Info().
		Str("id", provider.ID).
		Str("display_name", provider.DisplayName).
		Str("tenant_id", tenantID).
		Msg("Updated AI provider")

	return nil
}

// GetProvider retrieves a provider by ID
func (s *Storage) GetProvider(ctx context.Context, id string) (*ProviderRecord, error) {
	provider := &ProviderRecord{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		tenantID := database.TenantFromContext(ctx)
		query := `
			SELECT id, name, display_name, provider_type, is_default, use_for_embeddings, embedding_model, config, enabled, created_by, created_at, updated_at
			FROM ai.providers
			WHERE id = $1 AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
		`
		return tx.QueryRow(ctx, query, id, database.TenantOrNil(tenantID)).Scan(
			&provider.ID, &provider.Name, &provider.DisplayName, &provider.ProviderType,
			&provider.IsDefault, &provider.UseForEmbeddings, &provider.EmbeddingModel, &provider.Config, &provider.Enabled, &provider.CreatedBy,
			&provider.CreatedAt, &provider.UpdatedAt,
		)
	})

	if errors.Is(err, pgx.ErrNoRows) {
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

	provider := &ProviderRecord{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		tenantID := database.TenantFromContext(ctx)
		query := `
			SELECT id, name, display_name, provider_type, is_default, use_for_embeddings, embedding_model, config, enabled, created_by, created_at, updated_at
			FROM ai.providers
			WHERE name = $1 AND enabled = true AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
		`
		return tx.QueryRow(ctx, query, name, database.TenantOrNil(tenantID)).Scan(
			&provider.ID, &provider.Name, &provider.DisplayName, &provider.ProviderType,
			&provider.IsDefault, &provider.UseForEmbeddings, &provider.EmbeddingModel, &provider.Config, &provider.Enabled, &provider.CreatedBy,
			&provider.CreatedAt, &provider.UpdatedAt,
		)
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get provider by name: %w", err)
	}

	return provider, nil
}

// GetDefaultProvider retrieves the default provider
func (s *Storage) GetDefaultProvider(ctx context.Context) (*ProviderRecord, error) {
	provider := &ProviderRecord{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		tenantID := database.TenantFromContext(ctx)
		query := `
			SELECT id, name, display_name, provider_type, is_default, use_for_embeddings, embedding_model, config, enabled, created_by, created_at, updated_at
			FROM ai.providers
			WHERE is_default = true AND enabled = true AND (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
			LIMIT 1
		`
		return tx.QueryRow(ctx, query, database.TenantOrNil(tenantID)).Scan(
			&provider.ID, &provider.Name, &provider.DisplayName, &provider.ProviderType,
			&provider.IsDefault, &provider.UseForEmbeddings, &provider.EmbeddingModel, &provider.Config, &provider.Enabled, &provider.CreatedBy,
			&provider.CreatedAt, &provider.UpdatedAt,
		)
	})

	if errors.Is(err, pgx.ErrNoRows) {
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
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		tenantID := database.TenantFromContext(ctx)
		query := `
			SELECT id, name, display_name, provider_type, is_default, use_for_embeddings, embedding_model, config, enabled, created_by, created_at, updated_at
			FROM ai.providers
			WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
		`

		args := []interface{}{database.TenantOrNil(tenantID)}

		if enabledOnly {
			query += " AND enabled = true"
		}

		query += " ORDER BY is_default DESC, name"

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to list providers: %w", err)
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
				return fmt.Errorf("failed to scan provider row: %w", err)
			}
			// Set ReadOnly to false for database providers
			provider.ReadOnly = false
			providers = append(providers, provider)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return providers, nil
}

// SetDefaultProvider sets a provider as the default
func (s *Storage) SetDefaultProvider(ctx context.Context, id string) error {
	tenantID := database.TenantFromContext(ctx)

	return database.WrapWithTenantAwareRole(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		// Clear existing default
		_, err := tx.Exec(ctx, "UPDATE ai.providers SET is_default = false WHERE is_default = true")
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

		log.Info().Str("id", id).Msg("Set default AI provider")
		return nil
	})
}

// DeleteProvider deletes a provider by ID
// Deprecated: Use DeleteProviderWithTenant for tenant-scoped operations
func (s *Storage) DeleteProvider(ctx context.Context, id string) error {
	tenantID := database.TenantFromContext(ctx)
	return s.DeleteProviderWithTenant(ctx, tenantID, id)
}

// DeleteProviderWithTenant deletes a provider by ID with tenant context
func (s *Storage) DeleteProviderWithTenant(ctx context.Context, tenantID string, id string) error {
	query := `DELETE FROM ai.providers WHERE id = $1`

	var result pgconn.CommandTag
	err := database.WrapWithTenantAwareRole(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(ctx, query, id)
		return execErr
	})
	if err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("provider not found: %s", id)
	}

	log.Info().Str("id", id).Str("tenant_id", tenantID).Msg("Deleted AI provider")

	return nil
}

// GetEmbeddingProviderPreference returns the provider explicitly set for embeddings (if any)
// Returns nil if no explicit preference is set (use default provider in auto mode)
func (s *Storage) GetEmbeddingProviderPreference(ctx context.Context) (*ProviderRecord, error) {
	provider := &ProviderRecord{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		tenantID := database.TenantFromContext(ctx)
		query := `
			SELECT id, name, display_name, provider_type, is_default, use_for_embeddings, embedding_model, config, enabled, created_by, created_at, updated_at
			FROM ai.providers
			WHERE use_for_embeddings = true AND enabled = true AND (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
			LIMIT 1
		`
		return tx.QueryRow(ctx, query, database.TenantOrNil(tenantID)).Scan(
			&provider.ID, &provider.Name, &provider.DisplayName, &provider.ProviderType,
			&provider.IsDefault, &provider.UseForEmbeddings, &provider.EmbeddingModel, &provider.Config, &provider.Enabled, &provider.CreatedBy,
			&provider.CreatedAt, &provider.UpdatedAt,
		)
	})

	if errors.Is(err, pgx.ErrNoRows) {
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
	tenantID := database.TenantFromContext(ctx)

	return database.WrapWithTenantAwareRole(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		// Clear any existing embedding preference (set to NULL for auto mode)
		_, err := tx.Exec(ctx, "UPDATE ai.providers SET use_for_embeddings = NULL WHERE use_for_embeddings = true")
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

		return nil
	})
}
