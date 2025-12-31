package mcp

import (
	"encoding/json"
)

// JSON-RPC 2.0 Types

// JSONRPCVersion is the JSON-RPC protocol version
const JSONRPCVersion = "2.0"

// Request represents a JSON-RPC 2.0 request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"` // string, number, or null
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// Error represents a JSON-RPC 2.0 error
type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Data    any       `json:"data,omitempty"`
}

// ErrorCode represents JSON-RPC 2.0 error codes
type ErrorCode int

// Standard JSON-RPC 2.0 error codes
const (
	ErrorCodeParseError     ErrorCode = -32700
	ErrorCodeInvalidRequest ErrorCode = -32600
	ErrorCodeMethodNotFound ErrorCode = -32601
	ErrorCodeInvalidParams  ErrorCode = -32602
	ErrorCodeInternalError  ErrorCode = -32603
)

// MCP-specific error codes (reserved range: -32000 to -32099)
const (
	ErrorCodeResourceNotFound   ErrorCode = -32002
	ErrorCodeToolNotFound       ErrorCode = -32003
	ErrorCodeToolExecutionError ErrorCode = -32004
	ErrorCodeUnauthorized       ErrorCode = -32005
	ErrorCodeForbidden          ErrorCode = -32006
	ErrorCodeRateLimited        ErrorCode = -32007
)

// MCP Protocol Types

// ServerInfo contains information about the MCP server
type ServerInfo struct {
	Name            string             `json:"name"`
	Version         string             `json:"version"`
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
}

// ServerCapabilities describes what the server supports
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
}

// ToolsCapability indicates the server supports tools
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"` // Server can notify when tool list changes
}

// ResourcesCapability indicates the server supports resources
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`   // Clients can subscribe to resource changes
	ListChanged bool `json:"listChanged,omitempty"` // Server can notify when resource list changes
}

// PromptsCapability indicates the server supports prompts
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"` // Server can notify when prompt list changes
}

// Tool Types

// Tool represents an MCP tool definition
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema"` // JSON Schema for parameters
}

// ToolCall represents a request to call a tool
type ToolCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content Types

// ContentType represents the type of content
type ContentType string

const (
	ContentTypeText     ContentType = "text"
	ContentTypeImage    ContentType = "image"
	ContentTypeResource ContentType = "resource"
)

// Content represents content in a tool result or resource
type Content struct {
	Type     ContentType `json:"type"`
	Text     string      `json:"text,omitempty"`
	MimeType string      `json:"mimeType,omitempty"`
	Data     string      `json:"data,omitempty"` // Base64-encoded for binary content
	URI      string      `json:"uri,omitempty"`  // For resource content
}

// TextContent creates a text content item
func TextContent(text string) Content {
	return Content{
		Type: ContentTypeText,
		Text: text,
	}
}

// ErrorContent creates an error text content item
func ErrorContent(message string) Content {
	return Content{
		Type: ContentTypeText,
		Text: message,
	}
}

// Resource Types

// Resource represents an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceTemplate represents a parameterized resource URI
type ResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceContents represents the contents of a resource
type ResourceContents struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // Base64-encoded binary data
}

// MCP Method Names

// Initialize methods
const (
	MethodInitialize = "initialize"
	MethodPing       = "ping"
)

// Tool methods
const (
	MethodToolsList = "tools/list"
	MethodToolsCall = "tools/call"
)

// Resource methods
const (
	MethodResourcesList      = "resources/list"
	MethodResourcesRead      = "resources/read"
	MethodResourcesTemplates = "resources/templates"
)

// Request/Response Parameter Types

// InitializeParams contains parameters for initialize request
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
	Capabilities    ClientCapabilities `json:"capabilities"`
}

// ClientInfo contains information about the MCP client
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities describes what the client supports
type ClientCapabilities struct {
	Roots    *RootsCapability    `json:"roots,omitempty"`
	Sampling *SamplingCapability `json:"sampling,omitempty"`
}

// RootsCapability indicates the client supports roots
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability indicates the client supports sampling
type SamplingCapability struct{}

// InitializeResult contains the result of initialize request
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
	Capabilities    ServerCapabilities `json:"capabilities"`
}

// ToolsListResult contains the result of tools/list request
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ToolsCallParams contains parameters for tools/call request
type ToolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// ResourcesListResult contains the result of resources/list request
type ResourcesListResult struct {
	Resources []Resource `json:"resources"`
}

// ResourcesTemplatesResult contains the result of resources/templates request
type ResourcesTemplatesResult struct {
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
}

// ResourcesReadParams contains parameters for resources/read request
type ResourcesReadParams struct {
	URI string `json:"uri"`
}

// ResourcesReadResult contains the result of resources/read request
type ResourcesReadResult struct {
	Contents []ResourceContents `json:"contents"`
}

// PingResult contains the result of ping request
type PingResult struct{}

// Helper functions

// NewError creates a new JSON-RPC error response
func NewError(id any, code ErrorCode, message string, data any) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// NewResult creates a new JSON-RPC success response
func NewResult(id any, result any) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  result,
	}
}

// NewParseError creates a parse error response
func NewParseError(data any) *Response {
	return NewError(nil, ErrorCodeParseError, "Parse error", data)
}

// NewInvalidRequest creates an invalid request error response
func NewInvalidRequest(id any, data any) *Response {
	return NewError(id, ErrorCodeInvalidRequest, "Invalid Request", data)
}

// NewMethodNotFound creates a method not found error response
func NewMethodNotFound(id any, method string) *Response {
	return NewError(id, ErrorCodeMethodNotFound, "Method not found: "+method, nil)
}

// NewInvalidParams creates an invalid params error response
func NewInvalidParams(id any, message string) *Response {
	return NewError(id, ErrorCodeInvalidParams, "Invalid params: "+message, nil)
}

// NewInternalError creates an internal error response
func NewInternalError(id any, message string) *Response {
	return NewError(id, ErrorCodeInternalError, "Internal error: "+message, nil)
}

// NewUnauthorized creates an unauthorized error response
func NewUnauthorized(id any, message string) *Response {
	return NewError(id, ErrorCodeUnauthorized, "Unauthorized: "+message, nil)
}

// NewForbidden creates a forbidden error response
func NewForbidden(id any, message string) *Response {
	return NewError(id, ErrorCodeForbidden, "Forbidden: "+message, nil)
}

// NewToolNotFound creates a tool not found error response
func NewToolNotFound(id any, toolName string) *Response {
	return NewError(id, ErrorCodeToolNotFound, "Tool not found: "+toolName, nil)
}

// NewResourceNotFound creates a resource not found error response
func NewResourceNotFound(id any, uri string) *Response {
	return NewError(id, ErrorCodeResourceNotFound, "Resource not found: "+uri, nil)
}

// NewToolExecutionError creates a tool execution error response
func NewToolExecutionError(id any, message string) *Response {
	return NewError(id, ErrorCodeToolExecutionError, "Tool execution error: "+message, nil)
}

// NewRateLimited creates a rate limited error response
func NewRateLimited(id any) *Response {
	return NewError(id, ErrorCodeRateLimited, "Rate limit exceeded", nil)
}
