package extensions

import (
	"time"
)

// Extension represents a PostgreSQL extension with its current status
type Extension struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	DisplayName      string     `json:"display_name"`
	Description      string     `json:"description,omitempty"`
	Category         string     `json:"category"`
	IsCore           bool       `json:"is_core"`
	RequiresRestart  bool       `json:"requires_restart"`
	DocumentationURL string     `json:"documentation_url,omitempty"`
	IsEnabled        bool       `json:"is_enabled"`
	IsInstalled      bool       `json:"is_installed"`
	InstalledVersion string     `json:"installed_version,omitempty"`
	EnabledAt        *time.Time `json:"enabled_at,omitempty"`
	EnabledBy        *string    `json:"enabled_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// AvailableExtension represents an extension in the catalog
type AvailableExtension struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	DisplayName      string    `json:"display_name"`
	Description      string    `json:"description,omitempty"`
	Category         string    `json:"category"`
	IsCore           bool      `json:"is_core"`
	RequiresRestart  bool      `json:"requires_restart"`
	DocumentationURL string    `json:"documentation_url,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// EnabledExtension represents a record of an enabled extension
type EnabledExtension struct {
	ID            string     `json:"id"`
	ExtensionName string     `json:"extension_name"`
	EnabledAt     time.Time  `json:"enabled_at"`
	EnabledBy     *string    `json:"enabled_by,omitempty"`
	DisabledAt    *time.Time `json:"disabled_at,omitempty"`
	DisabledBy    *string    `json:"disabled_by,omitempty"`
	IsActive      bool       `json:"is_active"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Category represents an extension category with count
type Category struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ListExtensionsResponse is the response for listing extensions
type ListExtensionsResponse struct {
	Extensions []Extension `json:"extensions"`
	Categories []Category  `json:"categories"`
}

// ExtensionStatusResponse is the response for getting extension status
type ExtensionStatusResponse struct {
	Name             string `json:"name"`
	IsEnabled        bool   `json:"is_enabled"`
	IsInstalled      bool   `json:"is_installed"`
	InstalledVersion string `json:"installed_version,omitempty"`
	Error            string `json:"error,omitempty"`
}

// EnableExtensionRequest is the request body for enabling an extension
type EnableExtensionRequest struct {
	Schema string `json:"schema,omitempty"` // Optional schema to install into (defaults to public)
}

// EnableExtensionResponse is the response for enabling an extension
type EnableExtensionResponse struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Version string `json:"version,omitempty"`
}

// DisableExtensionResponse is the response for disabling an extension
type DisableExtensionResponse struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// PostgresExtension represents an extension as reported by pg_available_extensions
type PostgresExtension struct {
	Name             string  `json:"name"`
	DefaultVersion   string  `json:"default_version"`
	InstalledVersion *string `json:"installed_version,omitempty"`
	Comment          string  `json:"comment,omitempty"`
}

// CategoryDisplayNames maps category IDs to display names
var CategoryDisplayNames = map[string]string{
	"core":         "Core",
	"geospatial":   "Geospatial",
	"ai_ml":        "AI & Machine Learning",
	"monitoring":   "Monitoring",
	"scheduling":   "Scheduling",
	"data_types":   "Data Types",
	"text_search":  "Text Search",
	"indexing":     "Indexing",
	"networking":   "Networking",
	"testing":      "Testing",
	"maintenance":  "Maintenance",
	"performance":  "Performance",
	"foreign_data": "Foreign Data",
	"triggers":     "Triggers",
	"sampling":     "Sampling",
	"utilities":    "Utilities",
}
