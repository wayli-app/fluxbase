package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	embeddingTimeout = 60 * time.Second
)

// openAIEmbeddingProvider implements EmbeddingProvider for OpenAI
type openAIEmbeddingProvider struct {
	config     OpenAIConfig
	httpClient *http.Client
}

// newOpenAIEmbeddingProviderInternal creates a new OpenAI embedding provider
func newOpenAIEmbeddingProviderInternal(config OpenAIConfig) (*openAIEmbeddingProvider, error) {
	if config.BaseURL == "" {
		config.BaseURL = defaultOpenAIBaseURL
	}

	config.BaseURL = strings.TrimSuffix(config.BaseURL, "/")

	return &openAIEmbeddingProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: embeddingTimeout,
		},
	}, nil
}

// Embed generates embeddings for the given texts
func (p *openAIEmbeddingProvider) Embed(ctx context.Context, texts []string, model string) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided for embedding")
	}

	if model == "" {
		model = p.config.Model
	}

	// Build request body
	reqBody := map[string]interface{}{
		"model": model,
		"input": texts,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	if p.config.OrganizationID != "" {
		httpReq.Header.Set("OpenAI-Organization", p.config.OrganizationID)
	}

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
			return nil, fmt.Errorf("openai embedding error: %s (type: %s, code: %s)",
				errResp.Error.Message, errResp.Error.Type, errResp.Error.Code)
		}
		return nil, fmt.Errorf("openai embedding returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var openaiResp openAIEmbeddingResponse
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse embedding response: %w", err)
	}

	// Convert to our format
	embeddings := make([][]float32, len(openaiResp.Data))
	for _, d := range openaiResp.Data {
		embeddings[d.Index] = d.Embedding
	}

	// Determine dimensions from first embedding
	dimensions := 0
	if len(embeddings) > 0 && len(embeddings[0]) > 0 {
		dimensions = len(embeddings[0])
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Model:      openaiResp.Model,
		Dimensions: dimensions,
		Usage: &EmbeddingUsage{
			PromptTokens: openaiResp.Usage.PromptTokens,
			TotalTokens:  openaiResp.Usage.TotalTokens,
		},
	}, nil
}

// SupportedModels returns the embedding models supported by OpenAI
func (p *openAIEmbeddingProvider) SupportedModels() []EmbeddingModel {
	return OpenAIEmbeddingModels
}

// DefaultModel returns the default embedding model
func (p *openAIEmbeddingProvider) DefaultModel() string {
	if p.config.Model != "" {
		return p.config.Model
	}
	return "text-embedding-3-small"
}

// ValidateConfig validates the provider configuration
func (p *openAIEmbeddingProvider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return fmt.Errorf("openai: api_key is required")
	}
	return nil
}

// OpenAI embedding API response structs
type openAIEmbeddingResponse struct {
	Object string `json:"object"`
	Model  string `json:"model"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}
