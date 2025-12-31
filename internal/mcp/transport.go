package mcp

import (
	"encoding/json"
	"fmt"
)

// Transport handles JSON-RPC 2.0 message parsing and serialization
type Transport struct{}

// NewTransport creates a new MCP transport
func NewTransport() *Transport {
	return &Transport{}
}

// ParseRequest parses a JSON-RPC 2.0 request from raw JSON
func (t *Transport) ParseRequest(data []byte) (*Request, error) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to parse request: %w", err)
	}

	// Validate JSON-RPC version
	if req.JSONRPC != JSONRPCVersion {
		return nil, fmt.Errorf("invalid JSON-RPC version: expected %s, got %s", JSONRPCVersion, req.JSONRPC)
	}

	// Validate method is present
	if req.Method == "" {
		return nil, fmt.Errorf("method is required")
	}

	return &req, nil
}

// SerializeResponse serializes a JSON-RPC 2.0 response to JSON
func (t *Transport) SerializeResponse(resp *Response) ([]byte, error) {
	return json.Marshal(resp)
}

// ParseParams parses request params into a specific type
func ParseParams[T any](params json.RawMessage) (*T, error) {
	if len(params) == 0 {
		return nil, nil
	}

	var result T
	if err := json.Unmarshal(params, &result); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}
	return &result, nil
}

// MustParseParams parses request params into a specific type, returning an error if params are missing
func MustParseParams[T any](params json.RawMessage) (*T, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("params are required")
	}

	var result T
	if err := json.Unmarshal(params, &result); err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}
	return &result, nil
}

// IsNotification returns true if the request is a notification (no ID)
func IsNotification(req *Request) bool {
	return req.ID == nil
}

// ValidateID validates that the request ID is a valid type (string, number, or null)
func ValidateID(id any) bool {
	if id == nil {
		return true
	}
	switch id.(type) {
	case string, float64, int, int64:
		return true
	default:
		return false
	}
}
