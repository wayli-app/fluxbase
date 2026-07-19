package integrations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsSecretField pins the secret-field classifier. New providers that
// add additional secret fields (e.g., oauth_refresh_token) must extend
// this list — this test will fail until they do, surfacing the change.
func TestIsSecretField(t *testing.T) {
	assert.True(t, isSecretField("api_key"), "api_key must always be treated as a secret")
	assert.False(t, isSecretField("default_depth"))
	assert.False(t, isSecretField("base_url"))
	assert.False(t, isSecretField(""))
}

// TestIntegrationToAPIResponse_MasksSecrets verifies the API response
// shape: secrets are masked, non-secrets pass through, and the optional
// from_config / read_only / last_test_* fields render correctly.
func TestIntegrationToAPIResponse_MasksSecrets(t *testing.T) {
	i := &Integration{
		ID:              "int-1",
		Name:            "Tavily prod",
		IntegrationType: IntegrationTypeWebSearch,
		Provider:        ProviderTavily,
		Config: map[string]string{
			"api_key":       "tvly-secret-value-12345",
			"default_depth": "advanced",
			"base_url":      "https://api.tavily.com",
		},
		Enabled:        true,
		IsDefault:      true,
		FromConfig:     true,
		ReadOnly:       true,
		LastTestStatus: "ok",
		LastTestError:  "",
		CreatedAt:      time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 7, 19, 12, 30, 0, 0, time.UTC),
		CreatedBy:      "user-1",
	}

	resp := integrationToAPIResponse(i)
	require.NotNil(t, resp)

	assert.Equal(t, "int-1", resp["id"])
	assert.Equal(t, "Tavily prod", resp["name"])
	assert.Equal(t, "web_search", resp["integration_type"])
	assert.Equal(t, "tavily", resp["provider"])

	// Secrets MUST be masked
	configOut, ok := resp["config"].(map[string]string)
	require.True(t, ok, "config must be a map[string]string")
	assert.Equal(t, MaskPlaceholder, configOut["api_key"],
		"api_key must be masked to the placeholder, not the real value")
	assert.Equal(t, "advanced", configOut["default_depth"],
		"non-secret config fields must pass through unchanged")
	assert.Equal(t, "https://api.tavily.com", configOut["base_url"])

	assert.Equal(t, true, resp["enabled"])
	assert.Equal(t, true, resp["is_default"])
	assert.Equal(t, true, resp["from_config"])
	assert.Equal(t, true, resp["read_only"])
	assert.Equal(t, "ok", resp["last_test_status"])
	assert.Equal(t, "user-1", resp["created_by"])

	// Time format must be RFC3339 for SDK compatibility
	assert.Contains(t, resp["created_at"].(string), "2026-07-19T12:00:00Z")
	assert.Contains(t, resp["updated_at"].(string), "2026-07-19T12:30:00Z")
}

// TestIntegrationToAPIResponse_EmptySecretNotMasked verifies the empty-
// secret edge case: when no api_key is set, the response must NOT surface
// the mask placeholder (which would misleadingly imply "there used to be
// a key here"). Empty stays empty.
func TestIntegrationToAPIResponse_EmptySecretNotMasked(t *testing.T) {
	i := &Integration{
		ID:              "int-1",
		Name:            "Anon",
		IntegrationType: IntegrationTypeWebSearch,
		Provider:        ProviderTavily,
		Config:          map[string]string{}, // no api_key
	}

	resp := integrationToAPIResponse(i)
	configOut := resp["config"].(map[string]string)
	_, hasKey := configOut["api_key"]
	assert.False(t, hasKey,
		"empty api_key must not be present in response (preserve 'no key configured' semantics)")
}

// TestIntegrationToAPIResponse_NilReturnsNil is the nil-safe contract
// the handler's GetIntegration relies on when a row is missing.
func TestIntegrationToAPIResponse_NilReturnsNil(t *testing.T) {
	assert.Nil(t, integrationToAPIResponse(nil))
}

// TestIntegrationToAPIResponse_OmitsEmptyOptionalFields verifies that
// optional fields (from_config, read_only, last_test_*, created_by)
// don't appear in the response when unset. Keeps payloads small and
// matches SDK optional field semantics.
func TestIntegrationToAPIResponse_OmitsEmptyOptionalFields(t *testing.T) {
	i := &Integration{
		ID:              "int-1",
		Name:            "Plain",
		IntegrationType: IntegrationTypeWebSearch,
		Provider:        ProviderTavily,
		Config:          map[string]string{},
		Enabled:         true,
		IsDefault:       false,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	resp := integrationToAPIResponse(i)
	_, hasFromConfig := resp["from_config"]
	assert.False(t, hasFromConfig, "from_config must be omitted when false")
	_, hasReadOnly := resp["read_only"]
	assert.False(t, hasReadOnly, "read_only must be omitted when false")
	_, hasLastTestedAt := resp["last_tested_at"]
	assert.False(t, hasLastTestedAt, "last_tested_at must be omitted when never tested")
	_, hasLastTestStatus := resp["last_test_status"]
	assert.False(t, hasLastTestStatus, "last_test_status must be omitted when empty")
	_, hasCreatedBy := resp["created_by"]
	assert.False(t, hasCreatedBy, "created_by must be omitted when empty")
}

// TestTestIntegration_TavilySuccess verifies the test-connection code
// path against a real fake Tavily server. Exercises the same code the
// /test endpoint invokes.
func TestTestIntegration_TavilySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results": []}`))
	}))
	defer server.Close()

	i := &Integration{
		Provider: ProviderTavily,
		Config: map[string]string{
			"api_key":  "tvly-test",
			"base_url": server.URL,
		},
	}
	status, errMsg := testIntegration(context.Background(), i)
	assert.Equal(t, "ok", status)
	assert.Equal(t, "", errMsg)
}

// TestTestIntegration_TavilyFailure verifies the failure path: when
// Tavily returns an error, the status is "failed" and the error message
// is preserved for the UI to display.
func TestTestIntegration_TavilyFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"detail": "Invalid API key"}`))
	}))
	defer server.Close()

	i := &Integration{
		Provider: ProviderTavily,
		Config: map[string]string{
			"api_key":  "tvly-bad",
			"base_url": server.URL,
		},
	}
	status, errMsg := testIntegration(context.Background(), i)
	assert.Equal(t, "failed", status)
	assert.Contains(t, errMsg, "Invalid API key")
}

// TestTestIntegration_UnimplementedProviders verifies that providers
// the schema allows but no client implementation exists for surface a
// clear "not yet implemented" message rather than a generic error.
// Locks in the v1 contract: only Tavily is wired up; schema reserves
// names for future providers.
func TestTestIntegration_UnimplementedProviders(t *testing.T) {
	for _, p := range []ProviderName{ProviderBrave, ProviderJina, ProviderSingleFetch} {
		t.Run(string(p), func(t *testing.T) {
			i := &Integration{Provider: p}
			status, errMsg := testIntegration(context.Background(), i)
			assert.Equal(t, "failed", status)
			assert.Contains(t, errMsg, "not yet implemented")
			assert.Contains(t, errMsg, string(p))
		})
	}
}

// TestTestIntegration_UnknownProvider is the defensive path: the schema
// CHECK constraint should prevent unknown providers from being stored,
// but if one slips through (manual DB edit, migration drift), the test
// returns a clear error instead of panicking.
func TestTestIntegration_UnknownProvider(t *testing.T) {
	i := &Integration{Provider: ProviderName("future-provider")}
	status, errMsg := testIntegration(context.Background(), i)
	assert.Equal(t, "failed", status)
	assert.Contains(t, errMsg, "Unknown provider")
}
