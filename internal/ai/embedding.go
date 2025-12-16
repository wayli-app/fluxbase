package ai

import (
	"context"
	"fmt"
)

// EmbeddingProvider defines the interface for embedding generation
type EmbeddingProvider interface {
	// Embed generates embeddings for the given texts
	Embed(ctx context.Context, texts []string, model string) (*EmbeddingResponse, error)

	// SupportedModels returns the embedding models supported by this provider
	SupportedModels() []EmbeddingModel

	// DefaultModel returns the default embedding model
	DefaultModel() string

	// ValidateConfig validates the provider configuration
	ValidateConfig() error
}

// EmbeddingRequest represents a request to generate embeddings
type EmbeddingRequest struct {
	Texts []string `json:"texts"`
	Model string   `json:"model,omitempty"`
}

// EmbeddingResponse represents the embedding response
type EmbeddingResponse struct {
	Embeddings [][]float32     `json:"embeddings"`
	Model      string          `json:"model"`
	Dimensions int             `json:"dimensions"`
	Usage      *EmbeddingUsage `json:"usage,omitempty"`
}

// EmbeddingUsage represents token usage for embedding request
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// EmbeddingModel describes an embedding model
type EmbeddingModel struct {
	Name       string `json:"name"`
	Dimensions int    `json:"dimensions"`
	MaxTokens  int    `json:"max_tokens"`
}

// Common embedding models by provider
var (
	OpenAIEmbeddingModels = []EmbeddingModel{
		{Name: "text-embedding-3-small", Dimensions: 1536, MaxTokens: 8191},
		{Name: "text-embedding-3-large", Dimensions: 3072, MaxTokens: 8191},
		{Name: "text-embedding-ada-002", Dimensions: 1536, MaxTokens: 8191},
	}

	AzureEmbeddingModels = []EmbeddingModel{
		{Name: "text-embedding-3-small", Dimensions: 1536, MaxTokens: 8191},
		{Name: "text-embedding-3-large", Dimensions: 3072, MaxTokens: 8191},
		{Name: "text-embedding-ada-002", Dimensions: 1536, MaxTokens: 8191},
	}

	OllamaEmbeddingModels = []EmbeddingModel{
		{Name: "nomic-embed-text", Dimensions: 768, MaxTokens: 8192},
		{Name: "mxbai-embed-large", Dimensions: 1024, MaxTokens: 512},
		{Name: "all-minilm", Dimensions: 384, MaxTokens: 256},
	}
)

// NewEmbeddingProvider creates a new embedding provider based on the configuration
func NewEmbeddingProvider(config ProviderConfig) (EmbeddingProvider, error) {
	switch config.Type {
	case ProviderTypeOpenAI:
		return NewOpenAIEmbeddingProvider(config)
	case ProviderTypeAzure:
		return NewAzureEmbeddingProvider(config)
	case ProviderTypeOllama:
		return NewOllamaEmbeddingProvider(config)
	default:
		return nil, fmt.Errorf("unsupported embedding provider type: %s", config.Type)
	}
}

// NewOpenAIEmbeddingProvider creates an OpenAI embedding provider
func NewOpenAIEmbeddingProvider(config ProviderConfig) (EmbeddingProvider, error) {
	openaiConfig := OpenAIConfig{
		APIKey:         config.Config["api_key"],
		Model:          config.Model,
		OrganizationID: config.Config["organization_id"],
		BaseURL:        config.Config["base_url"],
	}

	if openaiConfig.APIKey == "" {
		return nil, fmt.Errorf("openai: api_key is required for embeddings")
	}

	if openaiConfig.Model == "" {
		openaiConfig.Model = "text-embedding-3-small"
	}

	return newOpenAIEmbeddingProviderInternal(openaiConfig)
}

// NewAzureEmbeddingProvider creates an Azure OpenAI embedding provider
func NewAzureEmbeddingProvider(config ProviderConfig) (EmbeddingProvider, error) {
	azureConfig := AzureConfig{
		APIKey:         config.Config["api_key"],
		Endpoint:       config.Config["endpoint"],
		DeploymentName: config.Config["deployment_name"],
		APIVersion:     config.Config["api_version"],
	}

	if azureConfig.APIKey == "" {
		return nil, fmt.Errorf("azure: api_key is required for embeddings")
	}

	if azureConfig.Endpoint == "" {
		return nil, fmt.Errorf("azure: endpoint is required for embeddings")
	}

	if azureConfig.DeploymentName == "" {
		return nil, fmt.Errorf("azure: deployment_name is required for embeddings")
	}

	if azureConfig.APIVersion == "" {
		azureConfig.APIVersion = "2024-02-15-preview"
	}

	return newAzureEmbeddingProviderInternal(azureConfig)
}

// NewOllamaEmbeddingProvider creates an Ollama embedding provider
func NewOllamaEmbeddingProvider(config ProviderConfig) (EmbeddingProvider, error) {
	ollamaConfig := OllamaConfig{
		Endpoint: config.Config["endpoint"],
		Model:    config.Model,
	}

	if ollamaConfig.Endpoint == "" {
		ollamaConfig.Endpoint = "http://localhost:11434"
	}

	if ollamaConfig.Model == "" {
		ollamaConfig.Model = "nomic-embed-text"
	}

	return newOllamaEmbeddingProviderInternal(ollamaConfig)
}

// GetEmbeddingModelDimensions returns the dimensions for a known model, or 0 if unknown
func GetEmbeddingModelDimensions(model string) int {
	allModels := append(append(OpenAIEmbeddingModels, AzureEmbeddingModels...), OllamaEmbeddingModels...)
	for _, m := range allModels {
		if m.Name == model {
			return m.Dimensions
		}
	}
	return 0
}
