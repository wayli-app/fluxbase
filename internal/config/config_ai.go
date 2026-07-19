package config

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// AIConfig contains AI chatbot settings
type AIConfig struct {
	Enabled              bool          `mapstructure:"enabled"`                // Enable AI chatbot functionality
	ChatbotsDir          string        `mapstructure:"chatbots_dir"`           // Directory for chatbot definitions
	AutoLoadOnBoot       bool          `mapstructure:"auto_load_on_boot"`      // Load chatbots from filesystem at boot
	DefaultMaxTokens     int           `mapstructure:"default_max_tokens"`     // Default max tokens per request
	DefaultModel         string        `mapstructure:"default_model"`          // Default AI model
	QueryTimeout         time.Duration `mapstructure:"query_timeout"`          // SQL query execution timeout
	MaxRowsPerQuery      int           `mapstructure:"max_rows_per_query"`     // Max rows returned per query
	ConversationCacheTTL time.Duration `mapstructure:"conversation_cache_ttl"` // TTL for conversation cache
	MaxConversationTurns int           `mapstructure:"max_conversation_turns"` // Max turns per conversation
	SyncAllowedIPRanges  []string      `mapstructure:"sync_allowed_ip_ranges"` // IP CIDR ranges allowed to sync chatbots

	// Provider Configuration (read-only in dashboard when set)
	// If ProviderType is set, a config-based provider will be added to the list
	ProviderType  string `mapstructure:"provider_type"`  // Provider type: openai, azure, ollama
	ProviderName  string `mapstructure:"provider_name"`  // Display name for config provider
	ProviderModel string `mapstructure:"provider_model"` // Default model for config provider

	// Embedding Configuration (for vector search)
	EmbeddingEnabled  bool   `mapstructure:"embedding_enabled"`  // Enable embedding generation for vector search
	EmbeddingProvider string `mapstructure:"embedding_provider"` // Embedding provider: openai, azure, ollama (defaults to ProviderType)
	EmbeddingModel    string `mapstructure:"embedding_model"`    // Embedding model: text-embedding-3-small, text-embedding-3-large, etc.

	// OpenAI Settings
	OpenAIAPIKey         string `mapstructure:"openai_api_key"`
	OpenAIOrganizationID string `mapstructure:"openai_organization_id"`
	OpenAIBaseURL        string `mapstructure:"openai_base_url"`

	// Azure Settings
	AzureAPIKey         string `mapstructure:"azure_api_key"`
	AzureEndpoint       string `mapstructure:"azure_endpoint"`
	AzureDeploymentName string `mapstructure:"azure_deployment_name"`
	AzureAPIVersion     string `mapstructure:"azure_api_version"`

	// Azure Embedding Settings (optional, falls back to Azure Settings)
	AzureEmbeddingDeploymentName string `mapstructure:"azure_embedding_deployment_name"` // Separate deployment for embeddings

	// Ollama Settings
	OllamaEndpoint string `mapstructure:"ollama_endpoint"`
	OllamaModel    string `mapstructure:"ollama_model"`

	// Tavily websearch Settings (for Web Agent specialist)
	// When TavilyAPIKey is set, a read-only synthetic FROM_CONFIG tool
	// integration row is exposed so instance-level deployments work the
	// same as multi-tenant dashboard deployments. Mirror the OpenAI/
	// Ollama env-config pattern.
	TavilyAPIKey       string `mapstructure:"tavily_api_key"`
	TavilyDefaultDepth string `mapstructure:"tavily_default_depth"` // "basic" (default) or "advanced"
	TavilyBaseURL      string `mapstructure:"tavily_base_url"`      // optional override

	// OCR Configuration (for image-based PDF extraction in knowledge bases)
	OCREnabled   bool     `mapstructure:"ocr_enabled"`   // Enable OCR for image-based PDFs
	OCRProvider  string   `mapstructure:"ocr_provider"`  // OCR provider: tesseract
	OCRLanguages []string `mapstructure:"ocr_languages"` // Default languages for OCR (e.g., ["eng", "deu"])

	// RAG Configuration (for retrieval-augmented generation)
	RAGGraphBoostWeight float64 `mapstructure:"rag_graph_boost_weight"` // How much to weight entity matches vs vector similarity (0.0-1.0, default 0)
}

// Validate validates AI configuration
func (ac *AIConfig) Validate() error {
	// Validate chatbots directory
	if ac.ChatbotsDir == "" {
		return fmt.Errorf("chatbots_dir cannot be empty")
	}

	// Validate token settings
	if ac.DefaultMaxTokens <= 0 {
		return fmt.Errorf("default_max_tokens must be positive, got: %d", ac.DefaultMaxTokens)
	}

	// Validate query timeout
	if ac.QueryTimeout <= 0 {
		return fmt.Errorf("query_timeout must be positive, got: %v", ac.QueryTimeout)
	}

	// Validate max rows per query
	if ac.MaxRowsPerQuery <= 0 {
		return fmt.Errorf("max_rows_per_query must be positive, got: %d", ac.MaxRowsPerQuery)
	}

	// Validate conversation settings
	if ac.ConversationCacheTTL <= 0 {
		return fmt.Errorf("conversation_cache_ttl must be positive, got: %v", ac.ConversationCacheTTL)
	}
	if ac.MaxConversationTurns <= 0 {
		return fmt.Errorf("max_conversation_turns must be positive, got: %d", ac.MaxConversationTurns)
	}

	// Warn if max rows is very high
	if ac.MaxRowsPerQuery > 10000 {
		log.Warn().Int("max_rows_per_query", ac.MaxRowsPerQuery).Msg("max_rows_per_query is over 10000 - large result sets may impact performance")
	}

	return nil
}
