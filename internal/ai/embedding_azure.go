package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// azureEmbeddingProvider implements EmbeddingProvider for Azure OpenAI
type azureEmbeddingProvider struct {
	config     AzureConfig
	httpClient *http.Client
}

// newAzureEmbeddingProviderInternal creates a new Azure OpenAI embedding provider
func newAzureEmbeddingProviderInternal(config AzureConfig) (*azureEmbeddingProvider, error) {
	config.Endpoint = strings.TrimSuffix(config.Endpoint, "/")

	return &azureEmbeddingProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: embeddingTimeout,
		},
	}, nil
}

// Embed generates embeddings for the given texts
func (p *azureEmbeddingProvider) Embed(ctx context.Context, texts []string, model string) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided for embedding")
	}

	// Azure uses deployment name instead of model in the URL
	// The model parameter is ignored - deployment determines the model
	deploymentName := p.config.DeploymentName

	// Build request body
	reqBody := map[string]interface{}{
		"input": texts,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	// Build Azure endpoint URL
	url := fmt.Sprintf("%s/openai/deployments/%s/embeddings?api-version=%s",
		p.config.Endpoint, deploymentName, p.config.APIVersion)

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", p.config.APIKey)

	// Execute request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send embedding request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedding response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("azure embedding error: %s (type: %s, code: %s)",
				errResp.Error.Message, errResp.Error.Type, errResp.Error.Code)
		}
		return nil, fmt.Errorf("azure embedding returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response (same format as OpenAI)
	var azureResp openAIEmbeddingResponse
	if err := json.Unmarshal(respBody, &azureResp); err != nil {
		return nil, fmt.Errorf("failed to parse embedding response: %w", err)
	}

	// Convert to our format
	embeddings := make([][]float32, len(azureResp.Data))
	for _, d := range azureResp.Data {
		embeddings[d.Index] = d.Embedding
	}

	// Determine dimensions from first embedding
	dimensions := 0
	if len(embeddings) > 0 && len(embeddings[0]) > 0 {
		dimensions = len(embeddings[0])
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Model:      azureResp.Model,
		Dimensions: dimensions,
		Usage: &EmbeddingUsage{
			PromptTokens: azureResp.Usage.PromptTokens,
			TotalTokens:  azureResp.Usage.TotalTokens,
		},
	}, nil
}

// SupportedModels returns the embedding models supported by Azure
func (p *azureEmbeddingProvider) SupportedModels() []EmbeddingModel {
	return AzureEmbeddingModels
}

// DefaultModel returns the default embedding model
func (p *azureEmbeddingProvider) DefaultModel() string {
	return "text-embedding-3-small"
}

// ValidateConfig validates the provider configuration
func (p *azureEmbeddingProvider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return fmt.Errorf("azure: api_key is required")
	}
	if p.config.Endpoint == "" {
		return fmt.Errorf("azure: endpoint is required")
	}
	if p.config.DeploymentName == "" {
		return fmt.Errorf("azure: deployment_name is required")
	}
	return nil
}
