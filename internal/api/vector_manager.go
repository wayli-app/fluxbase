package api

import (
	"context"
	"fmt"
	"sync"

	"github.com/fluxbase-eu/fluxbase/internal/ai"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/rs/zerolog/log"
)

// VectorManager manages the embedding service with support for dynamic configuration refresh
// from database-stored AI providers. It follows the pattern established by email.Manager.
type VectorManager struct {
	mu               sync.RWMutex
	embeddingService *ai.EmbeddingService
	aiStorage        *ai.Storage
	envConfig        *config.AIConfig
	schemaInspector  *database.SchemaInspector
	db               *database.Connection
}

// NewVectorManager creates a new vector manager with hot-reload capability
func NewVectorManager(envConfig *config.AIConfig, aiStorage *ai.Storage, schemaInspector *database.SchemaInspector, db *database.Connection) *VectorManager {
	m := &VectorManager{
		aiStorage:       aiStorage,
		envConfig:       envConfig,
		schemaInspector: schemaInspector,
		db:              db,
	}

	// Try to initialize embedding service from env config first
	// This follows the same initialization logic as the original NewVectorHandler
	m.initializeFromEnvConfig()

	// If no service from env config, try database providers
	if m.embeddingService == nil {
		ctx := context.Background()
		if err := m.RefreshFromDatabase(ctx); err != nil {
			log.Debug().Err(err).Msg("No embedding service initialized from database on startup")
		}
	}

	return m
}

// GetEmbeddingService returns the current embedding service (thread-safe)
func (m *VectorManager) GetEmbeddingService() *ai.EmbeddingService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.embeddingService
}

// GetEmbeddingServiceForProvider creates an embedding service for a specific provider by ID
// This is used when admins want to use a different provider than the default
func (m *VectorManager) GetEmbeddingServiceForProvider(ctx context.Context, providerID string) (*ai.EmbeddingService, error) {
	// Look up the provider
	provider, err := m.aiStorage.GetProvider(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	if provider == nil {
		return nil, fmt.Errorf("provider not found: %s", providerID)
	}
	if !provider.Enabled {
		return nil, fmt.Errorf("provider is disabled: %s", providerID)
	}

	// Build config and create service
	embeddingCfg, err := m.buildEmbeddingConfigFromProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to build embedding config for provider %s: %w", providerID, err)
	}

	service, err := ai.NewEmbeddingService(embeddingCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding service for provider %s: %w", providerID, err)
	}

	return service, nil
}

// initializeFromEnvConfig attempts to initialize embedding service from YAML/env configuration
func (m *VectorManager) initializeFromEnvConfig() {
	cfg := m.envConfig

	// Priority 1: Explicit embedding configuration
	if cfg.EmbeddingEnabled || cfg.EmbeddingProvider != "" {
		embeddingCfg, err := buildEmbeddingConfig(cfg)
		if err != nil {
			if cfg.EmbeddingEnabled {
				log.Warn().Err(err).Msg("Failed to build embedding config")
			} else {
				log.Debug().Err(err).Msg("Could not initialize explicit embedding configuration")
			}
			return
		}

		service, err := ai.NewEmbeddingService(embeddingCfg)
		if err != nil {
			if cfg.EmbeddingEnabled {
				log.Warn().Err(err).Msg("Failed to initialize embedding service")
			} else {
				log.Debug().Err(err).Msg("Failed to initialize embedding service")
			}
			return
		}

		m.embeddingService = service
		actualProvider := inferProviderType(cfg)
		log.Info().
			Str("provider", actualProvider).
			Msg("Embedding service initialized from explicit configuration")
		return
	}

	// Priority 2: Auto-enable from AI provider if AI provider is configured
	inferredProvider := inferProviderType(cfg)
	if inferredProvider != "" {
		embeddingCfg, err := buildEmbeddingConfigFromAIProvider(cfg)
		if err != nil {
			log.Debug().Err(err).Msg("Could not auto-enable embedding from AI provider")
			return
		}

		service, err := ai.NewEmbeddingService(embeddingCfg)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to initialize embedding service from AI provider")
			return
		}

		m.embeddingService = service
		log.Info().
			Str("provider", inferredProvider).
			Msg("Embedding service auto-enabled from AI provider")
	}
}

// RefreshFromDatabase rebuilds the embedding service from database-stored AI providers
func (m *VectorManager) RefreshFromDatabase(ctx context.Context) error {
	// Priority 1: If explicit embedding config exists in YAML/env, don't reload from database
	if m.envConfig.EmbeddingEnabled || m.envConfig.EmbeddingProvider != "" {
		log.Debug().Msg("Explicit embedding config set, skipping database provider reload")
		return nil
	}

	// Priority 2: Explicit embedding provider preference (database override)
	embeddingPref, err := m.aiStorage.GetEmbeddingProviderPreference(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get embedding provider preference")
		// Continue to fallback
	}

	var provider *ai.ProviderRecord
	if embeddingPref != nil {
		// Use explicit embedding provider
		provider = embeddingPref
		log.Debug().
			Str("provider_id", provider.ID).
			Str("provider_name", provider.DisplayName).
			Msg("Using explicit embedding provider preference")
	} else {
		// Priority 3: Default provider (fallback/auto mode)
		defaultProvider, err := m.aiStorage.GetEffectiveDefaultProvider(ctx)
		if err != nil {
			return fmt.Errorf("failed to get default provider: %w", err)
		}
		provider = defaultProvider
		if provider != nil {
			log.Debug().
				Str("provider_id", provider.ID).
				Str("provider_name", provider.DisplayName).
				Msg("No explicit embedding preference, using default provider")
		}
	}

	if provider == nil {
		// No provider available - clear embedding service
		m.mu.Lock()
		m.embeddingService = nil
		m.mu.Unlock()
		log.Debug().Msg("No embedding provider available")
		return nil
	}

	// Check if this is a config-based provider (from env/YAML)
	// If so, we don't need to reload as it's already loaded in initializeFromEnvConfig
	if provider.ReadOnly {
		log.Debug().Msg("Provider is from config, already initialized")
		return nil
	}

	// Build embedding config from database provider
	embeddingCfg, err := m.buildEmbeddingConfigFromProvider(provider)
	if err != nil {
		log.Warn().
			Err(err).
			Str("provider_id", provider.ID).
			Str("provider_type", provider.ProviderType).
			Msg("Failed to build embedding config from database provider")
		return fmt.Errorf("failed to build embedding config: %w", err)
	}

	// Create new embedding service
	service, err := ai.NewEmbeddingService(embeddingCfg)
	if err != nil {
		log.Warn().
			Err(err).
			Str("provider_id", provider.ID).
			Str("provider_type", provider.ProviderType).
			Msg("Failed to create embedding service from database provider")
		return fmt.Errorf("failed to create embedding service: %w", err)
	}

	// Atomically swap the service
	m.mu.Lock()
	m.embeddingService = service
	m.mu.Unlock()

	log.Info().
		Str("provider_id", provider.ID).
		Str("provider_type", provider.ProviderType).
		Str("provider_name", provider.DisplayName).
		Str("embedding_model", embeddingCfg.DefaultModel).
		Msg("Embedding service reloaded from database provider")

	return nil
}

// buildEmbeddingConfigFromProvider converts a database ProviderRecord to EmbeddingServiceConfig
func (m *VectorManager) buildEmbeddingConfigFromProvider(provider *ai.ProviderRecord) (ai.EmbeddingServiceConfig, error) {
	if provider == nil {
		return ai.EmbeddingServiceConfig{}, fmt.Errorf("provider is nil")
	}

	providerType := provider.ProviderType
	config := provider.Config

	// Build provider config
	providerCfg := ai.ProviderConfig{
		Type:   ai.ProviderType(providerType),
		Config: make(map[string]string),
	}

	// Determine embedding model with priority:
	// 1. provider.EmbeddingModel (explicit embedding model from database)
	// 2. config["model"] (provider's chat model fallback)
	// 3. Provider-specific default
	var embeddingModel string

	switch providerType {
	case "openai":
		// Required: api_key
		apiKey, ok := config["api_key"]
		if !ok || apiKey == "" {
			return ai.EmbeddingServiceConfig{}, fmt.Errorf("openai provider missing api_key")
		}
		providerCfg.Config["api_key"] = apiKey

		// Optional: organization_id, base_url
		if orgID, ok := config["organization_id"]; ok && orgID != "" {
			providerCfg.Config["organization_id"] = orgID
		}
		if baseURL, ok := config["base_url"]; ok && baseURL != "" {
			providerCfg.Config["base_url"] = baseURL
		}

		// Model priority: EmbeddingModel > config["model"] > default
		if provider.EmbeddingModel != nil && *provider.EmbeddingModel != "" {
			embeddingModel = *provider.EmbeddingModel
		} else if model, ok := config["model"]; ok && model != "" {
			embeddingModel = model
		} else {
			embeddingModel = "text-embedding-3-small"
		}

	case "azure":
		// Required: api_key, endpoint, deployment_name
		apiKey, ok := config["api_key"]
		if !ok || apiKey == "" {
			return ai.EmbeddingServiceConfig{}, fmt.Errorf("azure provider missing api_key")
		}
		endpoint, ok := config["endpoint"]
		if !ok || endpoint == "" {
			return ai.EmbeddingServiceConfig{}, fmt.Errorf("azure provider missing endpoint")
		}
		deploymentName, ok := config["deployment_name"]
		if !ok || deploymentName == "" {
			return ai.EmbeddingServiceConfig{}, fmt.Errorf("azure provider missing deployment_name")
		}

		providerCfg.Config["api_key"] = apiKey
		providerCfg.Config["endpoint"] = endpoint
		providerCfg.Config["deployment_name"] = deploymentName

		// Optional: api_version
		if apiVersion, ok := config["api_version"]; ok && apiVersion != "" {
			providerCfg.Config["api_version"] = apiVersion
		}

		// Model priority: EmbeddingModel > config["model"] > default
		if provider.EmbeddingModel != nil && *provider.EmbeddingModel != "" {
			embeddingModel = *provider.EmbeddingModel
		} else if model, ok := config["model"]; ok && model != "" {
			embeddingModel = model
		} else {
			embeddingModel = "text-embedding-ada-002"
		}

	case "ollama":
		// Optional: endpoint (defaults to localhost:11434)
		if endpoint, ok := config["endpoint"]; ok && endpoint != "" {
			providerCfg.Config["endpoint"] = endpoint
		}

		// Model priority: EmbeddingModel > config["model"] > default
		if provider.EmbeddingModel != nil && *provider.EmbeddingModel != "" {
			embeddingModel = *provider.EmbeddingModel
		} else if model, ok := config["model"]; ok && model != "" {
			embeddingModel = model
		} else {
			embeddingModel = "nomic-embed-text"
		}

	default:
		return ai.EmbeddingServiceConfig{}, fmt.Errorf("unsupported provider type: %s", providerType)
	}

	providerCfg.Model = embeddingModel

	return ai.EmbeddingServiceConfig{
		Provider:     providerCfg,
		DefaultModel: embeddingModel,
		CacheEnabled: true,
	}, nil
}
