package ai

import (
	"context"
	"fmt"
	"io"
)

// ProviderType represents the type of AI provider
type ProviderType string

const (
	ProviderTypeOpenAI ProviderType = "openai"
	ProviderTypeAzure  ProviderType = "azure"
	ProviderTypeOllama ProviderType = "ollama"
)

// Role represents the role of a message in a conversation
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message represents a message in a conversation
type Message struct {
	Role         Role          `json:"role"`
	Content      string        `json:"content"`
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID   string        `json:"tool_call_id,omitempty"`
	Name         string        `json:"name,omitempty"`
	QueryResults []QueryResult `json:"query_results,omitempty"` // SQL query results for assistant messages
}

// QueryResult represents the result of a SQL query execution
type QueryResult struct {
	Query    string           `json:"query"`     // The SQL query that was executed
	Summary  string           `json:"summary"`   // Human-readable summary (e.g., "Query returned 5 row(s)...")
	RowCount int              `json:"row_count"` // Number of rows returned
	Data     []map[string]any `json:"data"`      // The actual result rows
}

// ToolCall represents a function call requested by the model
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // always "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function details in a tool call
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// Tool represents a tool/function that can be called by the model
type Tool struct {
	Type     string       `json:"type"` // always "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction represents the function definition
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

// ChatRequest represents a request to the AI provider
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	// ToolChoice can be "none", "auto", or a specific tool name
	ToolChoice interface{} `json:"tool_choice,omitempty"`
}

// ChatResponse represents a non-streaming response from the AI provider
type ChatResponse struct {
	ID      string      `json:"id"`
	Model   string      `json:"model"`
	Choices []Choice    `json:"choices"`
	Usage   *UsageStats `json:"usage,omitempty"`
}

// Choice represents a single completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// UsageStats represents token usage statistics
type UsageStats struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamEvent represents a streaming event from the AI provider
type StreamEvent struct {
	// Type of event: "content", "tool_call", "done", "error"
	Type string `json:"type"`

	// Content delta for "content" events
	Delta string `json:"delta,omitempty"`

	// Tool call updates for "tool_call" events
	ToolCall *ToolCallDelta `json:"tool_call,omitempty"`

	// Usage stats for "done" events
	Usage *UsageStats `json:"usage,omitempty"`

	// Error message for "error" events
	Error string `json:"error,omitempty"`

	// FinishReason when stream completes
	FinishReason string `json:"finish_reason,omitempty"`
}

// ToolCallDelta represents incremental tool call data in streaming
type ToolCallDelta struct {
	Index          int    `json:"index"`
	ID             string `json:"id,omitempty"`
	Type           string `json:"type,omitempty"`
	FunctionName   string `json:"function_name,omitempty"`
	ArgumentsDelta string `json:"arguments_delta,omitempty"`
}

// StreamCallback is called for each streaming event
type StreamCallback func(event StreamEvent) error

// Provider defines the interface for AI providers
type Provider interface {
	// Name returns the provider name
	Name() string

	// Type returns the provider type
	Type() ProviderType

	// Chat sends a non-streaming chat request
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatStream sends a streaming chat request
	ChatStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error

	// ValidateConfig validates the provider configuration
	ValidateConfig() error

	// Close cleans up any resources
	Close() error
}

// ProviderConfig represents the base configuration for all providers
type ProviderConfig struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name"`
	Type        ProviderType      `json:"type"`
	Model       string            `json:"model"`
	Config      map[string]string `json:"config"` // Provider-specific config (api_key, endpoint, etc.)
}

// OpenAIConfig represents OpenAI-specific configuration
type OpenAIConfig struct {
	APIKey         string `json:"api_key"`
	Model          string `json:"model"`
	OrganizationID string `json:"organization_id,omitempty"`
	BaseURL        string `json:"base_url,omitempty"` // For API-compatible services
}

// AzureConfig represents Azure OpenAI-specific configuration
type AzureConfig struct {
	APIKey         string `json:"api_key"`
	Endpoint       string `json:"endpoint"`
	DeploymentName string `json:"deployment_name"`
	APIVersion     string `json:"api_version"`
}

// OllamaConfig represents Ollama-specific configuration
type OllamaConfig struct {
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`
}

// NewProvider creates a new AI provider based on the configuration
func NewProvider(config ProviderConfig) (Provider, error) {
	switch config.Type {
	case ProviderTypeOpenAI:
		return NewOpenAIProvider(config)
	case ProviderTypeAzure:
		return NewAzureProvider(config)
	case ProviderTypeOllama:
		return NewOllamaProvider(config)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}

// NewOpenAIProvider creates a new OpenAI provider (implemented in provider_openai.go)
func NewOpenAIProvider(config ProviderConfig) (Provider, error) {
	// Parse OpenAI-specific config from Config map
	openaiConfig := OpenAIConfig{
		APIKey:         config.Config["api_key"],
		Model:          config.Model,
		OrganizationID: config.Config["organization_id"],
		BaseURL:        config.Config["base_url"],
	}

	if openaiConfig.APIKey == "" {
		return nil, fmt.Errorf("openai: api_key is required")
	}

	if openaiConfig.Model == "" {
		openaiConfig.Model = "gpt-4-turbo"
	}

	return newOpenAIProviderInternal(config.Name, openaiConfig)
}

// NewAzureProvider creates a new Azure OpenAI provider (implemented in provider_azure.go)
func NewAzureProvider(config ProviderConfig) (Provider, error) {
	// Parse Azure-specific config from Config map
	azureConfig := AzureConfig{
		APIKey:         config.Config["api_key"],
		Endpoint:       config.Config["endpoint"],
		DeploymentName: config.Config["deployment_name"],
		APIVersion:     config.Config["api_version"],
	}

	if azureConfig.APIKey == "" {
		return nil, fmt.Errorf("azure: api_key is required")
	}

	if azureConfig.Endpoint == "" {
		return nil, fmt.Errorf("azure: endpoint is required")
	}

	if azureConfig.DeploymentName == "" {
		return nil, fmt.Errorf("azure: deployment_name is required")
	}

	if azureConfig.APIVersion == "" {
		azureConfig.APIVersion = "2024-02-15-preview"
	}

	return newAzureProviderInternal(config.Name, azureConfig)
}

// NewOllamaProvider creates a new Ollama provider (implemented in provider_ollama.go)
func NewOllamaProvider(config ProviderConfig) (Provider, error) {
	// Parse Ollama-specific config from Config map
	ollamaConfig := OllamaConfig{
		Endpoint: config.Config["endpoint"],
		Model:    config.Model,
	}

	if ollamaConfig.Endpoint == "" {
		ollamaConfig.Endpoint = "http://localhost:11434"
	}

	if ollamaConfig.Model == "" {
		return nil, fmt.Errorf("ollama: model is required")
	}

	return newOllamaProviderInternal(config.Name, ollamaConfig)
}

// ExecuteSQLTool is the standard tool definition for SQL execution
var ExecuteSQLTool = Tool{
	Type: "function",
	Function: ToolFunction{
		Name:        "execute_sql",
		Description: "Execute a read-only SQL query against the database. Returns a summary of results.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"sql": map[string]interface{}{
					"type":        "string",
					"description": "The SQL SELECT query to execute",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "A brief description of what this query is meant to find",
				},
			},
			"required": []string{"sql", "description"},
		},
	},
}

// HttpRequestTool is the standard tool definition for HTTP requests to external APIs
var HttpRequestTool = Tool{
	Type: "function",
	Function: ToolFunction{
		Name:        "http_request",
		Description: "Make an HTTP GET request to an external API. Only whitelisted domains configured for this chatbot are allowed.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "The full URL to request (must be HTTPS and on an allowed domain)",
				},
				"method": map[string]interface{}{
					"type":        "string",
					"description": "HTTP method - only GET is currently supported",
					"enum":        []string{"GET"},
				},
			},
			"required": []string{"url", "method"},
		},
	},
}

// ReadCloserWrapper wraps an io.Reader with a no-op Close method
type ReadCloserWrapper struct {
	io.Reader
}

// Close implements io.Closer
func (r *ReadCloserWrapper) Close() error {
	return nil
}
