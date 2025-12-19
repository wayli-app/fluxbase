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
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	openAITimeout        = 120 * time.Second
)

// openAIProvider implements the Provider interface for OpenAI
type openAIProvider struct {
	name       string
	config     OpenAIConfig
	httpClient *http.Client
}

// newOpenAIProviderInternal creates a new OpenAI provider instance
func newOpenAIProviderInternal(name string, config OpenAIConfig) (*openAIProvider, error) {
	if config.BaseURL == "" {
		config.BaseURL = defaultOpenAIBaseURL
	}

	// Remove trailing slash from base URL
	config.BaseURL = strings.TrimSuffix(config.BaseURL, "/")

	return &openAIProvider{
		name:   name,
		config: config,
		httpClient: &http.Client{
			Timeout: openAITimeout,
		},
	}, nil
}

// Name returns the provider name
func (p *openAIProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *openAIProvider) Type() ProviderType {
	return ProviderTypeOpenAI
}

// ValidateConfig validates the provider configuration
func (p *openAIProvider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return fmt.Errorf("openai: api_key is required")
	}
	if p.config.Model == "" {
		return fmt.Errorf("openai: model is required")
	}
	return nil
}

// Close cleans up resources
func (p *openAIProvider) Close() error {
	p.httpClient.CloseIdleConnections()
	return nil
}

// openAIRequest represents the OpenAI API request format
type openAIRequest struct {
	Model             string               `json:"model"`
	Messages          []openAIMessage      `json:"messages"`
	Tools             []openAITool         `json:"tools,omitempty"`
	MaxTokens         int                  `json:"max_tokens,omitempty"`
	Temperature       float64              `json:"temperature,omitempty"`
	Stream            bool                 `json:"stream,omitempty"`
	StreamOptions     *openAIStreamOptions `json:"stream_options,omitempty"`
	ToolChoice        interface{}          `json:"tool_choice,omitempty"`
	ParallelToolCalls *bool                `json:"parallel_tool_calls,omitempty"`
}

// openAIStreamOptions configures streaming behavior
type openAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
}

type openAIToolCall struct {
	Index    int                `json:"index"`
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIToolFunc `json:"function"`
}

type openAIToolFunc struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// openAIResponse represents the OpenAI API response format
type openAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   *openAIUsage   `json:"usage,omitempty"`
	Error   *openAIError   `json:"error,omitempty"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// openAIStreamChunk represents a streaming response chunk
type openAIStreamChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []openAIStreamChoice `json:"choices"`
	Usage   *openAIUsage         `json:"usage,omitempty"`
}

type openAIStreamChoice struct {
	Index        int               `json:"index"`
	Delta        openAIStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason,omitempty"`
}

type openAIStreamDelta struct {
	Role      string           `json:"role,omitempty"`
	Content   string           `json:"content,omitempty"`
	ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
}

// Chat sends a non-streaming chat request to OpenAI
func (p *openAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Build OpenAI request
	openaiReq := p.buildRequest(req)
	openaiReq.Stream = false

	// Make HTTP request
	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/chat/completions", bytes.NewReader(body))
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

	// Parse response
	var openaiResp openAIResponse
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error
	if openaiResp.Error != nil {
		return nil, fmt.Errorf("openai error: %s (type: %s, code: %s)",
			openaiResp.Error.Message, openaiResp.Error.Type, openaiResp.Error.Code)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Convert to our response format
	return p.convertResponse(&openaiResp), nil
}

// ChatStream sends a streaming chat request to OpenAI
func (p *openAIProvider) ChatStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error {
	// Build OpenAI request
	openaiReq := p.buildRequest(req)
	openaiReq.Stream = true
	openaiReq.StreamOptions = &openAIStreamOptions{IncludeUsage: true}

	// Make HTTP request
	body, err := json.Marshal(openaiReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/chat/completions", bytes.NewReader(body))
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
		return fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Process SSE stream
	return p.processStream(ctx, resp.Body, callback)
}

// buildRequest converts our ChatRequest to OpenAI format
func (p *openAIProvider) buildRequest(req *ChatRequest) openAIRequest {
	// Use request model or fall back to provider default
	model := req.Model
	if model == "" {
		model = p.config.Model
	}

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

	return openAIRequest{
		Model:             model,
		Messages:          messages,
		Tools:             tools,
		MaxTokens:         req.MaxTokens,
		Temperature:       req.Temperature,
		ToolChoice:        req.ToolChoice,
		ParallelToolCalls: parallelToolCalls,
	}
}

// setHeaders sets the required headers for OpenAI API requests
func (p *openAIProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	if p.config.OrganizationID != "" {
		req.Header.Set("OpenAI-Organization", p.config.OrganizationID)
	}
}

// convertResponse converts OpenAI response to our format
func (p *openAIProvider) convertResponse(resp *openAIResponse) *ChatResponse {
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

// processStream processes the SSE stream from OpenAI
func (p *openAIProvider) processStream(ctx context.Context, reader io.Reader, callback StreamCallback) error {
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
			// Send final done event with accumulated usage if available
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
				// Get or create tool call tracker using OpenAI's index field
				idx := tc.Index

				if _, exists := toolCalls[idx]; !exists {
					toolCalls[idx] = &ToolCallDelta{
						Index: idx,
					}
					log.Debug().Int("index", idx).Msg("AI: New tool call started")
				}

				tcDelta := toolCalls[idx]

				// Update ID if present
				if tc.ID != "" {
					tcDelta.ID = tc.ID
				}

				// Update type if present
				if tc.Type != "" {
					tcDelta.Type = tc.Type
				}

				// Update function name if present
				if tc.Function.Name != "" {
					tcDelta.FunctionName = tc.Function.Name
					log.Debug().Int("index", idx).Str("function", tc.Function.Name).Msg("AI: Tool call function name received")
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
					log.Debug().Int("total_tool_calls", len(toolCalls)).Msg("AI: Emitting tool calls")
					for _, tcDelta := range toolCalls {
						log.Debug().Int("index", tcDelta.Index).Str("id", tcDelta.ID).Str("function", tcDelta.FunctionName).Msg("AI: Emitting tool call")
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
