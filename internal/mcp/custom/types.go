// Package custom provides support for user-defined MCP tools and resources.
package custom

import (
	"time"

	"github.com/google/uuid"
)

// CustomTool represents a user-defined MCP tool implemented in TypeScript.
type CustomTool struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	Name        string         `json:"name" db:"name"`
	Namespace   string         `json:"namespace" db:"namespace"`
	Description string         `json:"description,omitempty" db:"description"`
	Code        string         `json:"code" db:"code"`
	InputSchema map[string]any `json:"input_schema" db:"input_schema"`

	// Execution settings
	RequiredScopes []string `json:"required_scopes" db:"required_scopes"`
	TimeoutSeconds int      `json:"timeout_seconds" db:"timeout_seconds"`
	MemoryLimitMB  int      `json:"memory_limit_mb" db:"memory_limit_mb"`

	// Deno sandbox permissions
	AllowNet   bool `json:"allow_net" db:"allow_net"`
	AllowEnv   bool `json:"allow_env" db:"allow_env"`
	AllowRead  bool `json:"allow_read" db:"allow_read"`
	AllowWrite bool `json:"allow_write" db:"allow_write"`

	// Metadata
	Enabled   bool       `json:"enabled" db:"enabled"`
	Version   int        `json:"version" db:"version"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// CustomResource represents a user-defined MCP resource implemented in TypeScript.
type CustomResource struct {
	ID          uuid.UUID `json:"id" db:"id"`
	URI         string    `json:"uri" db:"uri"`
	Name        string    `json:"name" db:"name"`
	Namespace   string    `json:"namespace" db:"namespace"`
	Description string    `json:"description,omitempty" db:"description"`
	MimeType    string    `json:"mime_type" db:"mime_type"`
	Code        string    `json:"code" db:"code"`
	IsTemplate  bool      `json:"is_template" db:"is_template"`

	// Security
	RequiredScopes []string `json:"required_scopes" db:"required_scopes"`

	// Execution settings
	TimeoutSeconds  int `json:"timeout_seconds" db:"timeout_seconds"`
	CacheTTLSeconds int `json:"cache_ttl_seconds" db:"cache_ttl_seconds"`

	// Metadata
	Enabled   bool       `json:"enabled" db:"enabled"`
	Version   int        `json:"version" db:"version"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateToolRequest represents a request to create a custom tool.
type CreateToolRequest struct {
	Name           string         `json:"name" validate:"required,min=1,max=64"`
	Namespace      string         `json:"namespace,omitempty"`
	Description    string         `json:"description,omitempty"`
	Code           string         `json:"code" validate:"required"`
	InputSchema    map[string]any `json:"input_schema,omitempty"`
	RequiredScopes []string       `json:"required_scopes,omitempty"`
	TimeoutSeconds int            `json:"timeout_seconds,omitempty"`
	MemoryLimitMB  int            `json:"memory_limit_mb,omitempty"`
	AllowNet       *bool          `json:"allow_net,omitempty"`
	AllowEnv       *bool          `json:"allow_env,omitempty"`
	AllowRead      *bool          `json:"allow_read,omitempty"`
	AllowWrite     *bool          `json:"allow_write,omitempty"`
	Enabled        *bool          `json:"enabled,omitempty"`
}

// UpdateToolRequest represents a request to update a custom tool.
type UpdateToolRequest struct {
	Name           *string        `json:"name,omitempty"`
	Description    *string        `json:"description,omitempty"`
	Code           *string        `json:"code,omitempty"`
	InputSchema    map[string]any `json:"input_schema,omitempty"`
	RequiredScopes []string       `json:"required_scopes,omitempty"`
	TimeoutSeconds *int           `json:"timeout_seconds,omitempty"`
	MemoryLimitMB  *int           `json:"memory_limit_mb,omitempty"`
	AllowNet       *bool          `json:"allow_net,omitempty"`
	AllowEnv       *bool          `json:"allow_env,omitempty"`
	AllowRead      *bool          `json:"allow_read,omitempty"`
	AllowWrite     *bool          `json:"allow_write,omitempty"`
	Enabled        *bool          `json:"enabled,omitempty"`
}

// CreateResourceRequest represents a request to create a custom resource.
type CreateResourceRequest struct {
	URI             string   `json:"uri" validate:"required,min=1,max=255"`
	Name            string   `json:"name" validate:"required,min=1,max=64"`
	Namespace       string   `json:"namespace,omitempty"`
	Description     string   `json:"description,omitempty"`
	MimeType        string   `json:"mime_type,omitempty"`
	Code            string   `json:"code" validate:"required"`
	IsTemplate      *bool    `json:"is_template,omitempty"`
	RequiredScopes  []string `json:"required_scopes,omitempty"`
	TimeoutSeconds  *int     `json:"timeout_seconds,omitempty"`
	CacheTTLSeconds *int     `json:"cache_ttl_seconds,omitempty"`
	Enabled         *bool    `json:"enabled,omitempty"`
}

// UpdateResourceRequest represents a request to update a custom resource.
type UpdateResourceRequest struct {
	URI             *string  `json:"uri,omitempty"`
	Name            *string  `json:"name,omitempty"`
	Description     *string  `json:"description,omitempty"`
	MimeType        *string  `json:"mime_type,omitempty"`
	Code            *string  `json:"code,omitempty"`
	IsTemplate      *bool    `json:"is_template,omitempty"`
	RequiredScopes  []string `json:"required_scopes,omitempty"`
	TimeoutSeconds  *int     `json:"timeout_seconds,omitempty"`
	CacheTTLSeconds *int     `json:"cache_ttl_seconds,omitempty"`
	Enabled         *bool    `json:"enabled,omitempty"`
}

// ListToolsFilter represents filters for listing custom tools.
type ListToolsFilter struct {
	Namespace   string
	EnabledOnly bool
	Limit       int
	Offset      int
}

// ListResourcesFilter represents filters for listing custom resources.
type ListResourcesFilter struct {
	Namespace   string
	EnabledOnly bool
	Limit       int
	Offset      int
}

// ToolExecutionResult represents the result of executing a custom tool.
type ToolExecutionResult struct {
	Success    bool           `json:"success"`
	Content    []Content      `json:"content,omitempty"`
	Error      string         `json:"error,omitempty"`
	DurationMs int64          `json:"duration_ms"`
	Logs       string         `json:"logs,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// ResourceReadResult represents the result of reading a custom resource.
type ResourceReadResult struct {
	Success    bool           `json:"success"`
	Content    []Content      `json:"content,omitempty"`
	Error      string         `json:"error,omitempty"`
	DurationMs int64          `json:"duration_ms"`
	Logs       string         `json:"logs,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// Content represents content returned by a tool or resource.
type Content struct {
	Type     string `json:"type"` // "text", "image", "resource"
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"` // Base64-encoded for binary
	URI      string `json:"uri,omitempty"`
}

// SyncToolRequest represents a request to sync (create or update) a tool by name.
type SyncToolRequest struct {
	CreateToolRequest
	// If true, update existing tool with same name instead of erroring
	Upsert bool `json:"upsert,omitempty"`
}

// SyncResourceRequest represents a request to sync (create or update) a resource by URI.
type SyncResourceRequest struct {
	CreateResourceRequest
	// If true, update existing resource with same URI instead of erroring
	Upsert bool `json:"upsert,omitempty"`
}
