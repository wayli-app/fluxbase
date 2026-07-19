package integrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnvConfig_HasIntegration verifies the gate logic that the storage
// layer uses to decide whether to inject a synthetic FROM_CONFIG row.
func TestEnvConfig_HasIntegration(t *testing.T) {
	t.Run("nil config returns false for everything", func(t *testing.T) {
		var nc *EnvConfig
		assert.False(t, nc.HasIntegration(IntegrationTypeWebSearch))
		assert.False(t, nc.HasIntegration(IntegrationTypeFetchURL))
	})

	t.Run("empty config returns false for everything", func(t *testing.T) {
		empty := &EnvConfig{}
		assert.False(t, empty.HasIntegration(IntegrationTypeWebSearch))
		assert.False(t, empty.HasIntegration(IntegrationTypeFetchURL))
	})

	t.Run("TavilyAPIKey set enables both web_search and fetch_url", func(t *testing.T) {
		c := &EnvConfig{TavilyAPIKey: "tvly-test"}
		assert.True(t, c.HasIntegration(IntegrationTypeWebSearch),
			"web_search must be available when TavilyAPIKey is set")
		assert.True(t, c.HasIntegration(IntegrationTypeFetchURL),
			"fetch_url is auto-derived from web_search; must be available too")
	})

	t.Run("empty TavilyAPIKey disables everything", func(t *testing.T) {
		c := &EnvConfig{TavilyDefaultDepth: "advanced"} // key not set
		assert.False(t, c.HasIntegration(IntegrationTypeWebSearch))
		assert.False(t, c.HasIntegration(IntegrationTypeFetchURL))
	})
}

// TestEnvConfig_ToIntegration_BuildsSyntheticRow verifies the shape of
// the FROM_CONFIG integration produced from env/YAML config. This row
// must be marked read_only so the API/UI refuses mutations and tagged
// from_config so the UI can render a "Read-only" badge.
func TestEnvConfig_ToIntegration_BuildsSyntheticRow(t *testing.T) {
	c := &EnvConfig{
		TavilyAPIKey:       "tvly-test",
		TavilyDefaultDepth: "advanced",
		TavilyBaseURL:      "https://custom.tavily.example",
	}

	t.Run("web_search produces a full synthetic row", func(t *testing.T) {
		i := c.ToIntegration(IntegrationTypeWebSearch)
		require.NotNil(t, i)
		assert.Equal(t, "FROM_CONFIG", i.ID, "stable ID so the UI can detect it")
		assert.Equal(t, "config", i.Name)
		assert.Equal(t, IntegrationTypeWebSearch, i.IntegrationType)
		assert.Equal(t, ProviderTavily, i.Provider)
		assert.True(t, i.Enabled, "env-configured integrations are enabled by default")
		assert.True(t, i.IsDefault, "env-configured integration is the default for its type")
		assert.True(t, i.FromConfig, "from_config flag must be set so UI shows the badge")
		assert.True(t, i.ReadOnly, "read_only must be set so API refuses mutations")
		assert.Equal(t, "tvly-test", i.Config["api_key"],
			"api_key carried through verbatim — encryption happens at write time, not here")
	})

	t.Run("fetch_url auto-derives from web_search", func(t *testing.T) {
		i := c.ToIntegration(IntegrationTypeFetchURL)
		require.NotNil(t, i)
		assert.Equal(t, IntegrationTypeFetchURL, i.IntegrationType)
		assert.Equal(t, ProviderTavily, i.Provider, "fetch_url uses the same Tavily provider credentials")
	})

	t.Run("unknown integration type returns nil", func(t *testing.T) {
		// ponytail: there's no "unknown" IntegrationType today (the type
		// is a closed enum), but if a future type is added without an
		// env-config path, ToIntegration must return nil rather than a
		// misleading synthetic row.
		i := c.ToIntegration(IntegrationType("future_type"))
		assert.Nil(t, i)
	})
}

// TestEnvConfig_ToIntegration_NilOrEmptyReturnsNil is the contract the
// storage layer relies on: if env config is missing, the synthetic row
// is missing too — GetDefaultIntegration falls through to DB rows.
func TestEnvConfig_ToIntegration_NilOrEmptyReturnsNil(t *testing.T) {
	t.Run("nil config returns nil for every type", func(t *testing.T) {
		var nc *EnvConfig
		assert.Nil(t, nc.ToIntegration(IntegrationTypeWebSearch))
		assert.Nil(t, nc.ToIntegration(IntegrationTypeFetchURL))
	})

	t.Run("empty config returns nil for every type", func(t *testing.T) {
		empty := &EnvConfig{}
		assert.Nil(t, empty.ToIntegration(IntegrationTypeWebSearch))
		assert.Nil(t, empty.ToIntegration(IntegrationTypeFetchURL))
	})

	t.Run("config without TavilyAPIKey returns nil", func(t *testing.T) {
		c := &EnvConfig{TavilyDefaultDepth: "advanced"}
		assert.Nil(t, c.ToIntegration(IntegrationTypeWebSearch))
	})
}
