package mcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextContent(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected Content
	}{
		{
			name: "simple text",
			text: "Hello, world!",
			expected: Content{
				Type: ContentTypeText,
				Text: "Hello, world!",
			},
		},
		{
			name: "empty text",
			text: "",
			expected: Content{
				Type: ContentTypeText,
				Text: "",
			},
		},
		{
			name: "multiline text",
			text: "Line 1\nLine 2\nLine 3",
			expected: Content{
				Type: ContentTypeText,
				Text: "Line 1\nLine 2\nLine 3",
			},
		},
		{
			name: "text with special characters",
			text: "Special: !@#$%^&*()",
			expected: Content{
				Type: ContentTypeText,
				Text: "Special: !@#$%^&*()",
			},
		},
		{
			name: "text with unicode",
			text: "こんにちは世界",
			expected: Content{
				Type: ContentTypeText,
				Text: "こんにちは世界",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TextContent(tt.text)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, ContentTypeText, result.Type)
			assert.Equal(t, tt.text, result.Text)
		})
	}
}

func TestErrorContent(t *testing.T) {
	tests := []struct {
		name    string
		message string
	}{
		{"simple error", "An error occurred"},
		{"empty error", ""},
		{"detailed error", "Error: Failed to execute tool 'query' with params {table: 'users'}"},
		{"error with newlines", "Error on line 1\nDetails on line 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ErrorContent(tt.message)
			assert.Equal(t, ContentTypeText, result.Type)
			assert.Equal(t, tt.message, result.Text)
		})
	}
}

func TestNewError(t *testing.T) {
	tests := []struct {
		name     string
		id       any
		code     ErrorCode
		message  string
		data     any
		expected *Response
	}{
		{
			name:    "error with string ID",
			id:      "req-123",
			code:    ErrorCodeInternalError,
			message: "Internal server error",
			data:    map[string]string{"detail": "Database connection failed"},
			expected: &Response{
				JSONRPC: JSONRPCVersion,
				ID:      "req-123",
				Error: &Error{
					Code:    ErrorCodeInternalError,
					Message: "Internal server error",
					Data:    map[string]string{"detail": "Database connection failed"},
				},
			},
		},
		{
			name:    "error with numeric ID",
			id:      42,
			code:    ErrorCodeMethodNotFound,
			message: "Method not found",
			data:    nil,
			expected: &Response{
				JSONRPC: JSONRPCVersion,
				ID:      42,
				Error: &Error{
					Code:    ErrorCodeMethodNotFound,
					Message: "Method not found",
					Data:    nil,
				},
			},
		},
		{
			name:    "error with nil ID",
			id:      nil,
			code:    ErrorCodeParseError,
			message: "Parse error",
			data:    "Invalid JSON",
			expected: &Response{
				JSONRPC: JSONRPCVersion,
				ID:      nil,
				Error: &Error{
					Code:    ErrorCodeParseError,
					Message: "Parse error",
					Data:    "Invalid JSON",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewError(tt.id, tt.code, tt.message, tt.data)
			assert.Equal(t, tt.expected.JSONRPC, result.JSONRPC)
			assert.Equal(t, tt.expected.ID, result.ID)
			assert.NotNil(t, result.Error)
			assert.Equal(t, tt.expected.Error.Code, result.Error.Code)
			assert.Equal(t, tt.expected.Error.Message, result.Error.Message)
			assert.Equal(t, tt.expected.Error.Data, result.Error.Data)
		})
	}
}

func TestNewResult(t *testing.T) {
	tests := []struct {
		name   string
		id     any
		result any
	}{
		{
			name:   "result with string ID",
			id:     "req-456",
			result: map[string]string{"status": "success"},
		},
		{
			name:   "result with numeric ID",
			id:     99,
			result: []string{"tool1", "tool2", "tool3"},
		},
		{
			name:   "result with nil value",
			id:     "req-789",
			result: nil,
		},
		{
			name: "result with complex structure",
			id:   1,
			result: InitializeResult{
				ProtocolVersion: "2024-11-05",
				ServerInfo: ServerInfo{
					Name:    "Fluxbase MCP",
					Version: "1.0.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := NewResult(tt.id, tt.result)
			assert.Equal(t, JSONRPCVersion, response.JSONRPC)
			assert.Equal(t, tt.id, response.ID)
			assert.Equal(t, tt.result, response.Result)
			assert.Nil(t, response.Error)
		})
	}
}

func TestNewParseError(t *testing.T) {
	data := "Unexpected token at position 42"
	response := NewParseError(data)

	assert.Equal(t, JSONRPCVersion, response.JSONRPC)
	assert.Nil(t, response.ID)
	assert.NotNil(t, response.Error)
	assert.Equal(t, ErrorCodeParseError, response.Error.Code)
	assert.Equal(t, "Parse error", response.Error.Message)
	assert.Equal(t, data, response.Error.Data)
}

func TestNewInvalidRequest(t *testing.T) {
	id := "invalid-req"
	data := map[string]string{"reason": "missing method field"}
	response := NewInvalidRequest(id, data)

	assert.Equal(t, JSONRPCVersion, response.JSONRPC)
	assert.Equal(t, id, response.ID)
	assert.NotNil(t, response.Error)
	assert.Equal(t, ErrorCodeInvalidRequest, response.Error.Code)
	assert.Equal(t, "Invalid Request", response.Error.Message)
	assert.Equal(t, data, response.Error.Data)
}

func TestNewMethodNotFound(t *testing.T) {
	id := "req-123"
	method := "tools/unknown"
	response := NewMethodNotFound(id, method)

	assert.Equal(t, JSONRPCVersion, response.JSONRPC)
	assert.Equal(t, id, response.ID)
	assert.NotNil(t, response.Error)
	assert.Equal(t, ErrorCodeMethodNotFound, response.Error.Code)
	assert.Contains(t, response.Error.Message, method)
	assert.Contains(t, response.Error.Message, "Method not found")
}

func TestNewInvalidParams(t *testing.T) {
	id := "req-456"
	message := "missing required field 'uri'"
	response := NewInvalidParams(id, message)

	assert.Equal(t, JSONRPCVersion, response.JSONRPC)
	assert.Equal(t, id, response.ID)
	assert.NotNil(t, response.Error)
	assert.Equal(t, ErrorCodeInvalidParams, response.Error.Code)
	assert.Contains(t, response.Error.Message, message)
	assert.Contains(t, response.Error.Message, "Invalid params")
}

func TestNewInternalError(t *testing.T) {
	id := 789
	message := "database connection failed"
	response := NewInternalError(id, message)

	assert.Equal(t, JSONRPCVersion, response.JSONRPC)
	assert.Equal(t, id, response.ID)
	assert.NotNil(t, response.Error)
	assert.Equal(t, ErrorCodeInternalError, response.Error.Code)
	assert.Contains(t, response.Error.Message, message)
	assert.Contains(t, response.Error.Message, "Internal error")
}

func TestNewUnauthorized(t *testing.T) {
	id := "auth-req"
	message := "invalid API key"
	response := NewUnauthorized(id, message)

	assert.Equal(t, JSONRPCVersion, response.JSONRPC)
	assert.Equal(t, id, response.ID)
	assert.NotNil(t, response.Error)
	assert.Equal(t, ErrorCodeUnauthorized, response.Error.Code)
	assert.Contains(t, response.Error.Message, message)
	assert.Contains(t, response.Error.Message, "Unauthorized")
}

func TestNewForbidden(t *testing.T) {
	id := "perm-req"
	message := "insufficient permissions"
	response := NewForbidden(id, message)

	assert.Equal(t, JSONRPCVersion, response.JSONRPC)
	assert.Equal(t, id, response.ID)
	assert.NotNil(t, response.Error)
	assert.Equal(t, ErrorCodeForbidden, response.Error.Code)
	assert.Contains(t, response.Error.Message, message)
	assert.Contains(t, response.Error.Message, "Forbidden")
}

func TestNewToolNotFound(t *testing.T) {
	id := "tool-req"
	toolName := "nonexistent_tool"
	response := NewToolNotFound(id, toolName)

	assert.Equal(t, JSONRPCVersion, response.JSONRPC)
	assert.Equal(t, id, response.ID)
	assert.NotNil(t, response.Error)
	assert.Equal(t, ErrorCodeToolNotFound, response.Error.Code)
	assert.Contains(t, response.Error.Message, toolName)
	assert.Contains(t, response.Error.Message, "Tool not found")
}

func TestNewResourceNotFound(t *testing.T) {
	id := "resource-req"
	uri := "fluxbase://schema/nonexistent"
	response := NewResourceNotFound(id, uri)

	assert.Equal(t, JSONRPCVersion, response.JSONRPC)
	assert.Equal(t, id, response.ID)
	assert.NotNil(t, response.Error)
	assert.Equal(t, ErrorCodeResourceNotFound, response.Error.Code)
	assert.Contains(t, response.Error.Message, uri)
	assert.Contains(t, response.Error.Message, "Resource not found")
}

func TestNewToolExecutionError(t *testing.T) {
	id := "exec-req"
	message := "query execution failed: syntax error"
	response := NewToolExecutionError(id, message)

	assert.Equal(t, JSONRPCVersion, response.JSONRPC)
	assert.Equal(t, id, response.ID)
	assert.NotNil(t, response.Error)
	assert.Equal(t, ErrorCodeToolExecutionError, response.Error.Code)
	assert.Contains(t, response.Error.Message, message)
	assert.Contains(t, response.Error.Message, "Tool execution error")
}

func TestNewRateLimited(t *testing.T) {
	id := "rate-req"
	response := NewRateLimited(id)

	assert.Equal(t, JSONRPCVersion, response.JSONRPC)
	assert.Equal(t, id, response.ID)
	assert.NotNil(t, response.Error)
	assert.Equal(t, ErrorCodeRateLimited, response.Error.Code)
	assert.Equal(t, "Rate limit exceeded", response.Error.Message)
	assert.Nil(t, response.Error.Data)
}

func TestErrorCodes(t *testing.T) {
	// Verify standard JSON-RPC error codes
	assert.Equal(t, ErrorCode(-32700), ErrorCodeParseError)
	assert.Equal(t, ErrorCode(-32600), ErrorCodeInvalidRequest)
	assert.Equal(t, ErrorCode(-32601), ErrorCodeMethodNotFound)
	assert.Equal(t, ErrorCode(-32602), ErrorCodeInvalidParams)
	assert.Equal(t, ErrorCode(-32603), ErrorCodeInternalError)

	// Verify MCP-specific error codes
	assert.Equal(t, ErrorCode(-32002), ErrorCodeResourceNotFound)
	assert.Equal(t, ErrorCode(-32003), ErrorCodeToolNotFound)
	assert.Equal(t, ErrorCode(-32004), ErrorCodeToolExecutionError)
	assert.Equal(t, ErrorCode(-32005), ErrorCodeUnauthorized)
	assert.Equal(t, ErrorCode(-32006), ErrorCodeForbidden)
	assert.Equal(t, ErrorCode(-32007), ErrorCodeRateLimited)
}

func TestJSONRPCVersion(t *testing.T) {
	assert.Equal(t, "2.0", JSONRPCVersion)
}

func TestContentTypes(t *testing.T) {
	assert.Equal(t, ContentType("text"), ContentTypeText)
	assert.Equal(t, ContentType("image"), ContentTypeImage)
	assert.Equal(t, ContentType("resource"), ContentTypeResource)
}

func TestMethodNames(t *testing.T) {
	// Initialize methods
	assert.Equal(t, "initialize", MethodInitialize)
	assert.Equal(t, "ping", MethodPing)

	// Tool methods
	assert.Equal(t, "tools/list", MethodToolsList)
	assert.Equal(t, "tools/call", MethodToolsCall)

	// Resource methods
	assert.Equal(t, "resources/list", MethodResourcesList)
	assert.Equal(t, "resources/read", MethodResourcesRead)
	assert.Equal(t, "resources/templates", MethodResourcesTemplates)
}

func TestResponseSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response *Response
	}{
		{
			name: "success response",
			response: NewResult("test-id", map[string]string{
				"status": "ok",
			}),
		},
		{
			name:     "error response",
			response: NewInternalError("error-id", "something went wrong"),
		},
		{
			name:     "parse error",
			response: NewParseError("invalid JSON at line 5"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize to JSON
			data, err := json.Marshal(tt.response)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Deserialize back
			var decoded Response
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			// Verify JSONRPC version
			assert.Equal(t, JSONRPCVersion, decoded.JSONRPC)
			assert.Equal(t, tt.response.ID, decoded.ID)
		})
	}
}

func TestRequestSerialization(t *testing.T) {
	req := Request{
		JSONRPC: JSONRPCVersion,
		ID:      "req-123",
		Method:  MethodToolsList,
		Params:  json.RawMessage(`{"filter":"enabled"}`),
	}

	// Serialize
	data, err := json.Marshal(req)
	require.NoError(t, err)

	// Deserialize
	var decoded Request
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.JSONRPC, decoded.JSONRPC)
	assert.Equal(t, req.ID, decoded.ID)
	assert.Equal(t, req.Method, decoded.Method)
	assert.JSONEq(t, string(req.Params), string(decoded.Params))
}

func TestContentSerialization(t *testing.T) {
	content := TextContent("Test message")

	data, err := json.Marshal(content)
	require.NoError(t, err)

	var decoded Content
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, content.Type, decoded.Type)
	assert.Equal(t, content.Text, decoded.Text)
}

func TestToolResultSerialization(t *testing.T) {
	result := ToolResult{
		Content: []Content{
			TextContent("Result line 1"),
			TextContent("Result line 2"),
		},
		IsError: false,
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded ToolResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, len(result.Content), len(decoded.Content))
	assert.Equal(t, result.IsError, decoded.IsError)
}

func TestAllErrorHelpers(t *testing.T) {
	// Test that all error helpers create valid responses
	errorHelpers := []struct {
		name     string
		response *Response
	}{
		{"ParseError", NewParseError("data")},
		{"InvalidRequest", NewInvalidRequest("id", "data")},
		{"MethodNotFound", NewMethodNotFound("id", "method")},
		{"InvalidParams", NewInvalidParams("id", "message")},
		{"InternalError", NewInternalError("id", "message")},
		{"Unauthorized", NewUnauthorized("id", "message")},
		{"Forbidden", NewForbidden("id", "message")},
		{"ToolNotFound", NewToolNotFound("id", "tool")},
		{"ResourceNotFound", NewResourceNotFound("id", "uri")},
		{"ToolExecutionError", NewToolExecutionError("id", "message")},
		{"RateLimited", NewRateLimited("id")},
	}

	for _, tt := range errorHelpers {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, JSONRPCVersion, tt.response.JSONRPC)
			assert.NotNil(t, tt.response.Error)
			assert.NotEmpty(t, tt.response.Error.Message)
			assert.NotEqual(t, ErrorCode(0), tt.response.Error.Code)
		})
	}
}
