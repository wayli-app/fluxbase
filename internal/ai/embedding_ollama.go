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

// ollamaEmbeddingProvider implements EmbeddingProvider for Ollama
type ollamaEmbeddingProvider struct {
	config     OllamaConfig
	httpClient *http.Client
}

// newOllamaEmbeddingProviderInternal creates a new Ollama embedding provider
func newOllamaEmbeddingProviderInternal(config OllamaConfig) (*ollamaEmbeddingProvider, error) {
	config.Endpoint = strings.TrimSuffix(config.Endpoint, "/")

	return &ollamaEmbeddingProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: embeddingTimeout,
		},
	}, nil
}

// Embed generates embeddings for the given texts
func (p *ollamaEmbeddingProvider) Embed(ctx context.Context, texts []string, model string) (*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided for embedding")
	}

	if model == "" {
		model = p.config.Model
	}

	// Ollama's embedding API processes one text at a time
	// We need to make multiple requests for batch embedding
	embeddings := make([][]float32, len(texts))
	var totalPromptTokens int

	for i, text := range texts {
		embedding, promptTokens, err := p.embedSingle(ctx, text, model)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = embedding
		totalPromptTokens += promptTokens
	}

	// Determine dimensions from first embedding
	dimensions := 0
	if len(embeddings) > 0 && len(embeddings[0]) > 0 {
		dimensions = len(embeddings[0])
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Model:      model,
		Dimensions: dimensions,
		Usage: &EmbeddingUsage{
			PromptTokens: totalPromptTokens,
			TotalTokens:  totalPromptTokens,
		},
	}, nil
}

// embedSingle generates an embedding for a single text
func (p *ollamaEmbeddingProvider) embedSingle(ctx context.Context, text, model string) ([]float32, int, error) {
	// Build request body for Ollama's embedding API
	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.Endpoint+"/api/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create embedding request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to send embedding request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read embedding response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error != "" {
			return nil, 0, fmt.Errorf("ollama embedding error: %s", errResp.Error)
		}
		return nil, 0, fmt.Errorf("ollama embedding returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var ollamaResp ollamaEmbeddingResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, 0, fmt.Errorf("failed to parse embedding response: %w", err)
	}

	// Convert float64 to float32
	embedding := make([]float32, len(ollamaResp.Embedding))
	for i, v := range ollamaResp.Embedding {
		embedding[i] = float32(v)
	}

	// Ollama doesn't provide token counts, estimate based on text length
	estimatedTokens := len(text) / 4

	return embedding, estimatedTokens, nil
}

// SupportedModels returns the embedding models supported by Ollama
func (p *ollamaEmbeddingProvider) SupportedModels() []EmbeddingModel {
	return OllamaEmbeddingModels
}

// DefaultModel returns the default embedding model
func (p *ollamaEmbeddingProvider) DefaultModel() string {
	if p.config.Model != "" {
		return p.config.Model
	}
	return "nomic-embed-text"
}

// ValidateConfig validates the provider configuration
func (p *ollamaEmbeddingProvider) ValidateConfig() error {
	if p.config.Model == "" {
		return fmt.Errorf("ollama: model is required")
	}
	return nil
}

// Ollama embedding API response struct
type ollamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}
