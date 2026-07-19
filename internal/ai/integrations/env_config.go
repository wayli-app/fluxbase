package integrations

import (
	"time"

	"github.com/google/uuid"
)

// EnvConfig holds Tavily/websearch credentials loaded from YAML or env
// vars. When populated, the integrations storage injects a synthetic
// FROM_CONFIG row (read-only) so instance-level deployments work the
// same as multi-tenant dashboard deployments.
//
// Mirrors the pattern in internal/ai/provider_storage.go:buildConfigBasedProvider.
type EnvConfig struct {
	// TavilyAPIKey enables websearch via Tavily. When non-empty, a
	// synthetic web_search integration row is exposed.
	TavilyAPIKey string

	// TavilyDefaultDepth is the default search depth: "basic" or
	// "advanced". Empty defaults to "basic" at the client layer.
	TavilyDefaultDepth string

	// TavilyBaseURL overrides the Tavily API endpoint (rare; mostly for
	// self-hosted mirrors or testing).
	TavilyBaseURL string
}

// HasIntegration reports whether env/YAML config provides an integration
// of the given type. Currently only web_search is configurable via env.
func (c *EnvConfig) HasIntegration(t IntegrationType) bool {
	if c == nil {
		return false
	}
	switch t {
	case IntegrationTypeWebSearch:
		return c.TavilyAPIKey != ""
	case IntegrationTypeFetchURL:
		// fetch_url is always auto-derived from web_search
		return c.TavilyAPIKey != ""
	}
	return false
}

// ToIntegration returns the synthetic FROM_CONFIG row for the given
// integration type, or nil if env config doesn't provide one.
//
// The returned row has a stable ID ("FROM_CONFIG") and ReadOnly=true
// so the API/UI can display it but refuse mutations.
func (c *EnvConfig) ToIntegration(t IntegrationType) *Integration {
	if c == nil || !c.HasIntegration(t) {
		return nil
	}
	now := time.Now()
	return &Integration{
		ID:              "FROM_CONFIG",
		Name:            "config",
		IntegrationType: t,
		Provider:        ProviderTavily,
		Enabled:         true,
		IsDefault:       true,
		FromConfig:      true,
		ReadOnly:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
		Config: map[string]string{
			"api_key": c.TavilyAPIKey,
		},
	}
}

// compileTimeUUIDImport keeps the uuid import non-empty for the
// ToIntegration helper above (used in case future revisions return
// uuid-based IDs). Remove when stable.
var _ = uuid.New
