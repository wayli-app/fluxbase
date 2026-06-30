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
	defaultAnthropicModel = "claude-sonnet-4-5-20250929"
	anthropicTimeout      = 5 * time.Minute
)

// anthropicProvider implements Provider for the Anthropic Messages API.
//
// Anthropic differs from OpenAI in three material ways:
//   - System content is a top-level field (an array of content blocks), not a
//     message in the messages array.
//   - Tool calls and tool results are content blocks (type "tool_use" /
//     "tool_result"), not separate fields on a message.
//   - Prompt caching is explicit via "cache_control: {type: 'ephemeral'}"
//     breakpoints on content blocks and tool definitions.
//
// Cache strategy used here: the LAST system content block (i.e. the dynamic
// per-turn context after Step 2 of the prompt-caching work) is NOT marked, but
// every system block before it IS marked as cacheable. This is wrong-by-design
// reversed: actually we want the static prefix cached and the dynamic tail not.
// Implementation below marks only the first (static) system block.
type anthropicProvider struct {
	name       string
	config     AnthropicConfig
	httpClient *http.Client
}

func newAnthropicProviderInternal(name string, config AnthropicConfig) (*anthropicProvider, error) {
	config.BaseURL = strings.TrimSuffix(config.BaseURL, "/")
	return &anthropicProvider{
		name:   name,
		config: config,
		httpClient: &http.Client{
			Timeout: anthropicTimeout,
		},
	}, nil
}

func (p *anthropicProvider) Name() string       { return p.name }
func (p *anthropicProvider) Type() ProviderType { return ProviderTypeAnthropic }
func (p *anthropicProvider) Close() error       { p.httpClient.CloseIdleConnections(); return nil }
func (p *anthropicProvider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return fmt.Errorf("anthropic: api_key is required")
	}
	if p.config.Model == "" {
		return fmt.Errorf("anthropic: model is required")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Wire types (Anthropic Messages API)
// ─────────────────────────────────────────────────────────────────────────────

// anthropicRequest is the request body for POST /v1/messages.
type anthropicRequest struct {
	Model       string                 `json:"model"`
	MaxTokens   int                    `json:"max_tokens"`
	System      []anthropicSystemBlock `json:"system,omitempty"`
	Messages    []anthropicMessage     `json:"messages"`
	Temperature float64                `json:"temperature,omitempty"`
	Tools       []anthropicTool        `json:"tools,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	StopSeqs    []string               `json:"stop_sequences,omitempty"`
}

// anthropicSystemBlock is a content block in the top-level system field.
// CacheControl marks the block as a cache breakpoint when non-nil.
type anthropicSystemBlock struct {
	Type         string                 `json:"type"` // always "text"
	Text         string                 `json:"text"`
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"`
}

type anthropicCacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// anthropicMessage is one message in the conversation. Content is always an
// array of typed blocks; for plain text we emit a single text block.
type anthropicMessage struct {
	Role    string             `json:"role"` // "user" or "assistant"
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type string `json:"type"`

	// For type == "text"
	Text string `json:"text,omitempty"`

	// For type == "tool_use" (assistant message)
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`

	// For type == "tool_result" (user message responding to a tool call)
	ToolUseID string `json:"tool_use_id,omitempty"`
	// Content of a tool_result is either a string or a content-block array.
	// We always emit a string for simplicity.
	Content string `json:"content,omitempty"`
	IsError bool   `json:"is_error,omitempty"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema"`
	// CacheControl on the LAST tool definition caches the tool block. Populated
	// by buildRequest when tools > 0.
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Response types (non-stream)
// ─────────────────────────────────────────────────────────────────────────────

type anthropicResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Model      string             `json:"model"`
	Content    []anthropicContent `json:"content"`
	StopReason string             `json:"stop_reason"`
	Usage      anthropicUsage     `json:"usage"`
}

// anthropicUsage is the Anthropic token-usage shape.
//
//	input_tokens                  — fresh input tokens billed at 1x
//	output_tokens                 — completion tokens billed at 1x
//	cache_creation_input_tokens   — tokens written to the cache this turn, 1.25x
//	cache_read_input_tokens       — tokens served from cache, 0.1x
//
// We surface cache_read_input_tokens as CachedTokens.
type anthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Stream event types
// ─────────────────────────────────────────────────────────────────────────────

type anthropicMessageStart struct {
	Message struct {
		ID    string         `json:"id"`
		Model string         `json:"model"`
		Usage anthropicUsage `json:"usage"`
	} `json:"message"`
}

type anthropicContentBlockStart struct {
	Index        int              `json:"index"`
	ContentBlock anthropicContent `json:"content_block"`
}

type anthropicContentBlockDelta struct {
	Index int                `json:"index"`
	Delta anthropicDeltaBody `json:"delta"`
}

// anthropicDeltaBody is the inner delta; Type discriminates "text_delta" vs
// "input_json_delta".
type anthropicDeltaBody struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`         // for text_delta
	PartialJSON string `json:"partial_json,omitempty"` // for input_json_delta
}

type anthropicMessageDelta struct {
	Delta struct {
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Chat (non-stream)
// ─────────────────────────────────────────────────────────────────────────────

func (p *anthropicProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	body, err := json.Marshal(p.buildRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var anthropicResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.convertResponse(&anthropicResp), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ChatStream
// ─────────────────────────────────────────────────────────────────────────────

func (p *anthropicProvider) ChatStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error {
	areq := p.buildRequest(req)
	areq.Stream = true

	body, err := json.Marshal(areq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anthropic returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return p.processStream(ctx, resp.Body, callback)
}

// ─────────────────────────────────────────────────────────────────────────────
// Request building
// ─────────────────────────────────────────────────────────────────────────────

func (p *anthropicProvider) buildRequest(req *ChatRequest) anthropicRequest {
	model := req.Model
	if model == "" {
		model = p.config.Model
	}
	if model == "" {
		model = defaultAnthropicModel
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		// Anthropic requires max_tokens. Use a sensible default if unset.
		maxTokens = 4096
	}

	out := anthropicRequest{
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
	}

	// Convert messages. System messages go to the top-level System field; all
	// others go to Messages. Each system message becomes one content block.
	var systemBlocks []anthropicSystemBlock
	var msgs []anthropicMessage

	for _, m := range req.Messages {
		switch m.Role {
		case RoleSystem:
			systemBlocks = append(systemBlocks, anthropicSystemBlock{
				Type: "text",
				Text: m.Content,
			})
		case RoleUser:
			msgs = append(msgs, anthropicMessage{
				Role:    "user",
				Content: []anthropicContent{{Type: "text", Text: m.Content}},
			})
		case RoleAssistant:
			content := []anthropicContent{}
			if m.Content != "" {
				content = append(content, anthropicContent{Type: "text", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				var input map[string]any
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
				content = append(content, anthropicContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}
			msgs = append(msgs, anthropicMessage{Role: "assistant", Content: content})
		case RoleTool:
			// OpenAI tool message → Anthropic "user" message with tool_result block.
			msgs = append(msgs, anthropicMessage{
				Role: "user",
				Content: []anthropicContent{{
					Type:      "tool_result",
					ToolUseID: m.ToolCallID,
					Content:   m.Content,
				}},
			})
		}
	}

	// Apply cache_control breakpoints. Mark all system blocks EXCEPT the last
	// one as cacheable. With our static+dynamic split (Step 2), the first
	// system block is the static prefix and the second is the dynamic per-turn
	// context (user ID, time, RAG). Caching just the static prefix is exactly
	// what we want.
	//
	// Anthropic allows up to 4 breakpoints; we use at most (system_blocks - 1)
	// here, plus 1 on the last tool definition below.
	for i := 0; i < len(systemBlocks)-1; i++ {
		systemBlocks[i].CacheControl = &anthropicCacheControl{Type: "ephemeral"}
	}
	out.System = systemBlocks
	out.Messages = msgs

	// Convert tools. Mark the LAST tool definition with cache_control so the
	// tool list is cached too (uses 1 of the 4 breakpoint budget; safe given
	// the system blocks above typically use only 1).
	if len(req.Tools) > 0 {
		tools := make([]anthropicTool, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = anthropicTool{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				InputSchema: t.Function.Parameters,
			}
		}
		tools[len(tools)-1].CacheControl = &anthropicCacheControl{Type: "ephemeral"}
		out.Tools = tools
	}

	return out
}

func (p *anthropicProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", p.config.APIVersion)
}

// ─────────────────────────────────────────────────────────────────────────────
// Response conversion (non-stream)
// ─────────────────────────────────────────────────────────────────────────────

func (p *anthropicProvider) convertResponse(resp *anthropicResponse) *ChatResponse {
	msg := Message{Role: RoleAssistant}
	var toolCalls []ToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			if msg.Content != "" {
				msg.Content += "\n"
			}
			msg.Content += block.Text
		case "tool_use":
			argsBytes, _ := json.Marshal(block.Input)
			toolCalls = append(toolCalls, ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      block.Name,
					Arguments: string(argsBytes),
				},
			})
		}
	}
	msg.ToolCalls = toolCalls

	finishReason := mapAnthropicStopReason(resp.StopReason)

	return &ChatResponse{
		ID:    resp.ID,
		Model: resp.Model,
		Choices: []Choice{{
			Index:        0,
			Message:      msg,
			FinishReason: finishReason,
		}},
		Usage: anthropicUsageToStats(&resp.Usage),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Stream processing
// ─────────────────────────────────────────────────────────────────────────────

func (p *anthropicProvider) processStream(ctx context.Context, reader io.Reader, callback StreamCallback) error {
	scanner := bufio.NewScanner(reader)
	// Anthropic SSE chunks can be sizable (tool_use inputs as partial_json);
	// raise the per-line cap to 1 MiB.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Track tool_use blocks by index so we can emit a single tool_call event
	// when the block starts and accumulate input_json_delta into ArgumentsDelta.
	toolBlocks := make(map[int]*ToolCallDelta)

	// Track usage; final "done" event carries the cumulative totals. Anthropic
	// emits input tokens in message_start and output tokens in message_delta.
	var inputTokens, outputTokens, cacheCreation, cacheRead int
	var stopReason string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		// Peek the type field to dispatch.
		var head struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(data), &head); err != nil {
			log.Warn().Err(err).Str("data", data).Msg("Failed to parse Anthropic stream envelope")
			continue
		}

		switch head.Type {
		case "message_start":
			var ev anthropicMessageStart
			if err := json.Unmarshal([]byte(data), &ev); err == nil {
				inputTokens = ev.Message.Usage.InputTokens
				cacheCreation = ev.Message.Usage.CacheCreationInputTokens
				cacheRead = ev.Message.Usage.CacheReadInputTokens
			}

		case "content_block_start":
			var ev anthropicContentBlockStart
			if err := json.Unmarshal([]byte(data), &ev); err == nil {
				if ev.ContentBlock.Type == "tool_use" {
					delta := &ToolCallDelta{
						Index:        ev.Index,
						ID:           ev.ContentBlock.ID,
						Type:         "function",
						FunctionName: ev.ContentBlock.Name,
					}
					toolBlocks[ev.Index] = delta
					if err := callback(StreamEvent{
						Type:     "tool_call",
						ToolCall: delta,
					}); err != nil {
						return err
					}
				}
			}

		case "content_block_delta":
			var ev anthropicContentBlockDelta
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				log.Warn().Err(err).Msg("Failed to parse content_block_delta")
				continue
			}
			switch ev.Delta.Type {
			case "text_delta":
				if err := callback(StreamEvent{
					Type:  "content",
					Delta: ev.Delta.Text,
				}); err != nil {
					return err
				}
			case "input_json_delta":
				if tc, ok := toolBlocks[ev.Index]; ok {
					tc.ArgumentsDelta += ev.Delta.PartialJSON
				}
			}

		case "content_block_stop":
			// No action; tool_call deltas were already emitted at start.
			// If we ever need to flush accumulated arguments here, we can.

		case "message_delta":
			var ev anthropicMessageDelta
			if err := json.Unmarshal([]byte(data), &ev); err == nil {
				if ev.Delta.StopReason != "" {
					stopReason = ev.Delta.StopReason
				}
				if ev.Usage.OutputTokens > 0 {
					outputTokens = ev.Usage.OutputTokens
				}
			}

		case "message_stop":
			usage := anthropicUsageToStats(&anthropicUsage{
				InputTokens:              inputTokens,
				OutputTokens:             outputTokens,
				CacheCreationInputTokens: cacheCreation,
				CacheReadInputTokens:     cacheRead,
			})
			return callback(StreamEvent{
				Type:         "done",
				FinishReason: mapAnthropicStopReason(stopReason),
				Usage:        usage,
			})

		case "ping", "error":
			// Anthropic sends periodic ping events; ignore. Errors are also
			// delivered as non-200 HTTP status (handled by the caller).
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream scanner error: %w", err)
	}

	// Stream ended without an explicit message_stop. Emit a synthesized done.
	usage := anthropicUsageToStats(&anthropicUsage{
		InputTokens:              inputTokens,
		OutputTokens:             outputTokens,
		CacheCreationInputTokens: cacheCreation,
		CacheReadInputTokens:     cacheRead,
	})
	return callback(StreamEvent{
		Type:         "done",
		FinishReason: mapAnthropicStopReason(stopReason),
		Usage:        usage,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// anthropicUsageToStats maps Anthropic's split input accounting to UsageStats.
// PromptTokens = all input billed (fresh + cache creation + cache read).
// CachedTokens = cache reads only.
func anthropicUsageToStats(u *anthropicUsage) *UsageStats {
	if u == nil {
		return nil
	}
	prompt := u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
	return &UsageStats{
		PromptTokens:     prompt,
		CompletionTokens: u.OutputTokens,
		TotalTokens:      prompt + u.OutputTokens,
		CachedTokens:     u.CacheReadInputTokens,
	}
}

func mapAnthropicStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "tool_use":
		return "tool_calls"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	case "":
		return ""
	default:
		return reason
	}
}
