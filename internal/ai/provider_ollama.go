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
	ollamaTimeout = 300 * time.Second // Longer timeout for local models
)

// ollamaProvider implements the Provider interface for Ollama
type ollamaProvider struct {
	name       string
	config     OllamaConfig
	httpClient *http.Client
}

// newOllamaProviderInternal creates a new Ollama provider instance
func newOllamaProviderInternal(name string, config OllamaConfig) (*ollamaProvider, error) {
	// Remove trailing slash from endpoint
	config.Endpoint = strings.TrimSuffix(config.Endpoint, "/")

	return &ollamaProvider{
		name:   name,
		config: config,
		httpClient: &http.Client{
			Timeout: ollamaTimeout,
		},
	}, nil
}

// Name returns the provider name
func (p *ollamaProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *ollamaProvider) Type() ProviderType {
	return ProviderTypeOllama
}

// ValidateConfig validates the provider configuration
func (p *ollamaProvider) ValidateConfig() error {
	if p.config.Endpoint == "" {
		return fmt.Errorf("ollama: endpoint is required")
	}
	if p.config.Model == "" {
		return fmt.Errorf("ollama: model is required")
	}
	return nil
}

// Close cleans up resources
func (p *ollamaProvider) Close() error {
	p.httpClient.CloseIdleConnections()
	return nil
}

// ollamaRequest represents the Ollama chat API request format
type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Tools    []ollamaTool    `json:"tools,omitempty"`
	Stream   bool            `json:"stream"`
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

type ollamaToolCall struct {
	Function ollamaFunctionCall `json:"function"`
}

type ollamaFunctionCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type ollamaTool struct {
	Type     string             `json:"type"`
	Function ollamaToolFunction `json:"function"`
}

type ollamaToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"` // Max tokens
}

// ollamaResponse represents the Ollama chat API response format
type ollamaResponse struct {
	Model              string        `json:"model"`
	CreatedAt          string        `json:"created_at"`
	Message            ollamaMessage `json:"message"`
	Done               bool          `json:"done"`
	DoneReason         string        `json:"done_reason,omitempty"`
	TotalDuration      int64         `json:"total_duration,omitempty"`
	LoadDuration       int64         `json:"load_duration,omitempty"`
	PromptEvalCount    int           `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64         `json:"prompt_eval_duration,omitempty"`
	EvalCount          int           `json:"eval_count,omitempty"`
	EvalDuration       int64         `json:"eval_duration,omitempty"`
}

// Chat sends a non-streaming chat request to Ollama
func (p *ollamaProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Build Ollama request
	ollamaReq := p.buildRequest(req)
	ollamaReq.Stream = false

	// Make HTTP request
	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.Endpoint+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to our response format
	return p.convertResponse(&ollamaResp), nil
}

// ChatStream sends a streaming chat request to Ollama
func (p *ollamaProvider) ChatStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error {
	// Build Ollama request
	ollamaReq := p.buildRequest(req)
	ollamaReq.Stream = true

	// Make HTTP request
	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.Endpoint+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Process stream (Ollama uses newline-delimited JSON)
	return p.processStream(ctx, resp.Body, callback)
}

// buildRequest converts our ChatRequest to Ollama format
func (p *ollamaProvider) buildRequest(req *ChatRequest) ollamaRequest {
	// Use request model or fall back to provider default
	model := req.Model
	if model == "" {
		model = p.config.Model
	}

	// Convert messages
	messages := make([]ollamaMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = ollamaMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}

		// Convert tool calls
		if len(msg.ToolCalls) > 0 {
			messages[i].ToolCalls = make([]ollamaToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				// Parse arguments from JSON string to map
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					args = make(map[string]interface{})
				}
				messages[i].ToolCalls[j] = ollamaToolCall{
					Function: ollamaFunctionCall{
						Name:      tc.Function.Name,
						Arguments: args,
					},
				}
			}
		}
	}

	// Convert tools
	var tools []ollamaTool
	if len(req.Tools) > 0 {
		tools = make([]ollamaTool, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = ollamaTool{
				Type: t.Type,
				Function: ollamaToolFunction{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}
	}

	// Build options
	var options *ollamaOptions
	if req.MaxTokens > 0 || req.Temperature > 0 {
		options = &ollamaOptions{
			Temperature: req.Temperature,
			NumPredict:  req.MaxTokens,
		}
	}

	return ollamaRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
		Options:  options,
	}
}

// convertResponse converts Ollama response to our format
func (p *ollamaProvider) convertResponse(resp *ollamaResponse) *ChatResponse {
	msg := Message{
		Role:    Role(resp.Message.Role),
		Content: resp.Message.Content,
	}

	// Convert native tool calls
	if len(resp.Message.ToolCalls) > 0 {
		msg.ToolCalls = make([]ToolCall, len(resp.Message.ToolCalls))
		for i, tc := range resp.Message.ToolCalls {
			// Convert arguments map to JSON string
			argsJSON, _ := json.Marshal(tc.Function.Arguments)
			msg.ToolCalls[i] = ToolCall{
				ID:   fmt.Sprintf("call_%d", i),
				Type: "function",
				Function: FunctionCall{
					Name:      tc.Function.Name,
					Arguments: string(argsJSON),
				},
			}
		}
	} else if tc := tryParseToolCallFromContent(msg.Content); tc != nil {
		// Fallback: try to parse tool call from content
		// (for models that don't support native tool calling)
		argsJSON, _ := json.Marshal(tc.Function.Arguments)
		msg.ToolCalls = []ToolCall{{
			ID:   "call_0",
			Type: "function",
			Function: FunctionCall{
				Name:      tc.Function.Name,
				Arguments: string(argsJSON),
			},
		}}
		// Clear content since it was actually a tool call
		msg.Content = ""
		log.Debug().
			Str("tool", tc.Function.Name).
			Msg("Parsed tool call from content (model lacks native tool calling support)")
	}

	// Calculate token usage from durations (approximate)
	var usage *UsageStats
	if resp.PromptEvalCount > 0 || resp.EvalCount > 0 {
		usage = &UsageStats{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		}
	}

	finishReason := "stop"
	if resp.DoneReason != "" {
		finishReason = resp.DoneReason
	}

	return &ChatResponse{
		ID:    resp.CreatedAt,
		Model: resp.Model,
		Choices: []Choice{
			{
				Index:        0,
				Message:      msg,
				FinishReason: finishReason,
			},
		},
		Usage: usage,
	}
}

// processStream processes the newline-delimited JSON stream from Ollama
func (p *ollamaProvider) processStream(ctx context.Context, reader io.Reader, callback StreamCallback) error {
	scanner := bufio.NewScanner(reader)

	var lastUsage *UsageStats
	var accumulatedContent strings.Builder
	var pendingToolCallBuffer strings.Builder // Buffer for potential JSON tool call
	hasNativeToolCalls := false
	bufferingToolCall := false // Currently buffering a potential tool call

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse chunk
		var chunk ollamaResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			log.Warn().Err(err).Str("line", line).Msg("Failed to parse Ollama streaming chunk")
			continue
		}

		// Track usage
		if chunk.PromptEvalCount > 0 || chunk.EvalCount > 0 {
			lastUsage = &UsageStats{
				PromptTokens:     chunk.PromptEvalCount,
				CompletionTokens: chunk.EvalCount,
				TotalTokens:      chunk.PromptEvalCount + chunk.EvalCount,
			}
		}

		// Check if done
		if chunk.Done {
			// Check if buffered content is a tool call
			if !hasNativeToolCalls && pendingToolCallBuffer.Len() > 0 {
				if tc := tryParseToolCallFromContent(pendingToolCallBuffer.String()); tc != nil {
					argsJSON, _ := json.Marshal(tc.Function.Arguments)
					if err := callback(StreamEvent{
						Type: "tool_call",
						ToolCall: &ToolCallDelta{
							Index:          0,
							ID:             "call_0",
							Type:           "function",
							FunctionName:   tc.Function.Name,
							ArgumentsDelta: string(argsJSON),
						},
					}); err != nil {
						return err
					}
					log.Debug().
						Str("tool", tc.Function.Name).
						Msg("Parsed tool call from content (model lacks native tool calling support)")
				} else {
					// Not a tool call after all - flush buffered content
					if err := callback(StreamEvent{
						Type:  "content",
						Delta: pendingToolCallBuffer.String(),
					}); err != nil {
						return err
					}
				}
			}

			event := StreamEvent{
				Type:         "done",
				FinishReason: "stop",
				Usage:        lastUsage,
			}
			if chunk.DoneReason != "" {
				event.FinishReason = chunk.DoneReason
			}
			return callback(event)
		}

		// Handle content
		if chunk.Message.Content != "" {
			accumulatedContent.WriteString(chunk.Message.Content)

			// Check if we should start buffering (content chunk starts with '{')
			trimmed := strings.TrimSpace(chunk.Message.Content)
			if !bufferingToolCall && strings.HasPrefix(trimmed, "{") {
				bufferingToolCall = true
			}

			if bufferingToolCall {
				// Buffer content that might be a tool call
				pendingToolCallBuffer.WriteString(chunk.Message.Content)
			} else {
				// Stream content immediately
				if err := callback(StreamEvent{
					Type:  "content",
					Delta: chunk.Message.Content,
				}); err != nil {
					return err
				}
			}
		}

		// Handle native tool calls
		for i, tc := range chunk.Message.ToolCalls {
			hasNativeToolCalls = true
			argsJSON, _ := json.Marshal(tc.Function.Arguments)
			if err := callback(StreamEvent{
				Type: "tool_call",
				ToolCall: &ToolCallDelta{
					Index:          i,
					ID:             fmt.Sprintf("call_%d", i),
					Type:           "function",
					FunctionName:   tc.Function.Name,
					ArgumentsDelta: string(argsJSON),
				},
			}); err != nil {
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	return nil
}

// tryParseToolCallFromContent attempts to extract a tool call from content text.
// Some Ollama models don't support native tool calling and instead output JSON
// in the content field. This function detects and parses such responses.
// Returns the tool call if found, or nil if content is not a tool call.
func tryParseToolCallFromContent(content string) *ollamaToolCall {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "{") || !strings.HasSuffix(content, "}") {
		return nil
	}

	// Try to parse as a tool call
	var tc struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal([]byte(content), &tc); err != nil {
		return nil
	}

	// Validate it's a known tool with required fields
	if tc.Name == "" || tc.Arguments == nil {
		return nil
	}
	if tc.Name != "execute_sql" && tc.Name != "http_request" {
		return nil
	}

	return &ollamaToolCall{
		Function: ollamaFunctionCall{
			Name:      tc.Name,
			Arguments: tc.Arguments,
		},
	}
}
