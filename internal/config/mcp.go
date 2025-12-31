package config

import (
	"fmt"
	"time"
)

// MCPConfig contains Model Context Protocol server settings
type MCPConfig struct {
	Enabled          bool          `mapstructure:"enabled"`           // Enable MCP server endpoint
	BasePath         string        `mapstructure:"base_path"`         // Base path for MCP endpoints (default: "/mcp")
	SessionTimeout   time.Duration `mapstructure:"session_timeout"`   // Session timeout for stateful connections
	MaxMessageSize   int           `mapstructure:"max_message_size"`  // Maximum message size in bytes
	AllowedTools     []string      `mapstructure:"allowed_tools"`     // Allowed tool names (empty = all enabled)
	AllowedResources []string      `mapstructure:"allowed_resources"` // Allowed resource URIs (empty = all enabled)
	RateLimitPerMin  int           `mapstructure:"rate_limit_per_min"` // Rate limit per minute per client key
}

// Validate validates MCP configuration
func (mc *MCPConfig) Validate() error {
	if !mc.Enabled {
		return nil // No validation needed if disabled
	}

	if mc.BasePath == "" {
		return fmt.Errorf("mcp base_path cannot be empty when enabled")
	}

	if mc.SessionTimeout < 0 {
		return fmt.Errorf("mcp session_timeout cannot be negative, got: %v", mc.SessionTimeout)
	}

	if mc.MaxMessageSize < 0 {
		return fmt.Errorf("mcp max_message_size cannot be negative, got: %d", mc.MaxMessageSize)
	}

	if mc.RateLimitPerMin < 0 {
		return fmt.Errorf("mcp rate_limit_per_min cannot be negative, got: %d", mc.RateLimitPerMin)
	}

	return nil
}
