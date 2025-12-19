package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	azureTimeout = 120 * time.Second
)

// azureProvider implements the Provider interface for Azure OpenAI
type azureProvider struct {
	name       string
	config     AzureConfig
	httpClient *http.Client
}

// newAzureProviderInternal creates a new Azure OpenAI provider instance
func newAzureProviderInternal(name string, config AzureConfig) (*azureProvider, error) {
	// Remove trailing slash from endpoint
	config.Endpoint = strings.TrimSuffix(config.Endpoint, "/")

	return &azureProvider{
		name:   name,
		config: config,
		httpClient: &http.Client{
			Timeout: azureTimeout,
		},
	}, nil
}

// Name returns the provider name
func (p *azureProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *azureProvider) Type() ProviderType {
	return ProviderTypeAzure
}

// ValidateConfig validates the provider configuration
func (p *azureProvider) ValidateConfig() error {
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

// Close cleans up resources
func (p *azureProvider) Close() error {
	p.httpClient.CloseIdleConnections()
	return nil
}

// getEndpointURL returns the full endpoint URL for chat completions
func (p *azureProvider) getEndpointURL() string {
	return fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		p.config.Endpoint, p.config.DeploymentName, p.config.APIVersion)
}

// Chat sends a non-streaming chat request to Azure OpenAI
func (p *azureProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Build request (Azure uses same format as OpenAI)
	azureReq := p.buildRequest(req)
	azureReq.Stream = false

	// Make HTTP request
	body, err := json.Marshal(azureReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.getEndpointURL(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response (same format as OpenAI)
	var azureResp openAIResponse
	if err := json.Unmarshal(respBody, &azureResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error
	if azureResp.Error != nil {
		return nil, fmt.Errorf("azure error: %s (type: %s, code: %s)",
			azureResp.Error.Message, azureResp.Error.Type, azureResp.Error.Code)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("azure returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Convert to our response format
	return p.convertResponse(&azureResp), nil
}

// ChatStream sends a streaming chat request to Azure OpenAI
func (p *azureProvider) ChatStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error {
	// Build request
	azureReq := p.buildRequest(req)
	azureReq.Stream = true
	azureReq.StreamOptions = &azureStreamOptions{IncludeUsage: true}

	// Make HTTP request
	body, err := json.Marshal(azureReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.getEndpointURL(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("azure returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Process SSE stream (same format as OpenAI)
	return p.processStream(ctx, resp.Body, callback)
}

// azureRequest is the same as OpenAI request
type azureRequest struct {
	Messages          []openAIMessage      `json:"messages"`
	Tools             []openAITool         `json:"tools,omitempty"`
	MaxTokens         int                  `json:"max_tokens,omitempty"`
	Temperature       float64              `json:"temperature,omitempty"`
	Stream            bool                 `json:"stream,omitempty"`
	StreamOptions     *azureStreamOptions  `json:"stream_options,omitempty"`
	ToolChoice        interface{}          `json:"tool_choice,omitempty"`
	ParallelToolCalls *bool                `json:"parallel_tool_calls,omitempty"`
}

// azureStreamOptions configures streaming behavior
type azureStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// buildRequest converts our ChatRequest to Azure format
func (p *azureProvider) buildRequest(req *ChatRequest) azureRequest {
	// Convert messages
	messages := make([]openAIMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openAIMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
			Name:       msg.Name,
		}

		// Convert tool calls
		if len(msg.ToolCalls) > 0 {
			messages[i].ToolCalls = make([]openAIToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				messages[i].ToolCalls[j] = openAIToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: openAIFunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}

	// Convert tools
	var tools []openAITool
	if len(req.Tools) > 0 {
		tools = make([]openAITool, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = openAITool{
				Type: t.Type,
				Function: openAIToolFunc{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}
	}

	// Enable parallel tool calls when tools are available
	var parallelToolCalls *bool
	if len(tools) > 0 {
		t := true
		parallelToolCalls = &t
	}

	return azureRequest{
		Messages:          messages,
		Tools:             tools,
		MaxTokens:         req.MaxTokens,
		Temperature:       req.Temperature,
		ToolChoice:        req.ToolChoice,
		ParallelToolCalls: parallelToolCalls,
	}
}

// setHeaders sets the required headers for Azure OpenAI API requests
func (p *azureProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.config.APIKey)
}

// convertResponse converts Azure response to our format (same as OpenAI)
func (p *azureProvider) convertResponse(resp *openAIResponse) *ChatResponse {
	choices := make([]Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		msg := Message{
			Role:    Role(c.Message.Role),
			Content: c.Message.Content,
		}

		// Convert tool calls
		if len(c.Message.ToolCalls) > 0 {
			msg.ToolCalls = make([]ToolCall, len(c.Message.ToolCalls))
			for j, tc := range c.Message.ToolCalls {
				msg.ToolCalls[j] = ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		choices[i] = Choice{
			Index:        c.Index,
			Message:      msg,
			FinishReason: c.FinishReason,
		}
	}

	var usage *UsageStats
	if resp.Usage != nil {
		usage = &UsageStats{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return &ChatResponse{
		ID:      resp.ID,
		Model:   resp.Model,
		Choices: choices,
		Usage:   usage,
	}
}

// processStream processes the SSE stream from Azure (same format as OpenAI)
func (p *azureProvider) processStream(ctx context.Context, reader io.Reader, callback StreamCallback) error {
	scanner := bufio.NewScanner(reader)

	// Track accumulated tool calls
	toolCalls := make(map[int]*ToolCallDelta)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// SSE format: "data: {json}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream end
		if data == "[DONE]" {
			return callback(StreamEvent{
				Type:         "done",
				FinishReason: "stop",
			})
		}

		// Parse chunk
		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.Warn().Err(err).Str("data", data).Msg("Failed to parse streaming chunk")
			continue
		}

		// Process each choice
		for _, choice := range chunk.Choices {
			// Handle content delta
			if choice.Delta.Content != "" {
				if err := callback(StreamEvent{
					Type:  "content",
					Delta: choice.Delta.Content,
				}); err != nil {
					return err
				}
			}

			// Accumulate tool calls (arguments come in chunks)
			for _, tc := range choice.Delta.ToolCalls {
				// Use OpenAI's index field to track which tool call this chunk belongs to
				idx := tc.Index

				if _, exists := toolCalls[idx]; !exists {
					toolCalls[idx] = &ToolCallDelta{
						Index: idx,
					}
				}

				tcDelta := toolCalls[idx]

				if tc.ID != "" {
					tcDelta.ID = tc.ID
				}
				if tc.Type != "" {
					tcDelta.Type = tc.Type
				}
				if tc.Function.Name != "" {
					tcDelta.FunctionName = tc.Function.Name
				}
				// Accumulate arguments (they come in chunks during streaming)
				if tc.Function.Arguments != "" {
					tcDelta.ArgumentsDelta += tc.Function.Arguments
				}
			}

			// Check for finish reason - emit complete tool calls when ready
			if choice.FinishReason != "" {
				// If finish reason is "tool_calls", emit all accumulated tool calls
				if choice.FinishReason == "tool_calls" {
					for _, tcDelta := range toolCalls {
						if err := callback(StreamEvent{
							Type:     "tool_call",
							ToolCall: tcDelta,
						}); err != nil {
							return err
						}
					}
				}

				event := StreamEvent{
					Type:         "done",
					FinishReason: choice.FinishReason,
				}
				if chunk.Usage != nil {
					event.Usage = &UsageStats{
						PromptTokens:     chunk.Usage.PromptTokens,
						CompletionTokens: chunk.Usage.CompletionTokens,
						TotalTokens:      chunk.Usage.TotalTokens,
					}
				}
				if err := callback(event); err != nil {
					return err
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	return nil
}
