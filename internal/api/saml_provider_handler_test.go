package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// SAMLProviderHandler Construction Tests
// =============================================================================

func TestNewSAMLProviderHandler(t *testing.T) {
	t.Run("creates handler with nil dependencies", func(t *testing.T) {
		handler := NewSAMLProviderHandler(nil, nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.db)
		assert.Nil(t, handler.samlService)
		assert.NotNil(t, handler.httpClient)
	})

	t.Run("http client has timeout configured", func(t *testing.T) {
		handler := NewSAMLProviderHandler(nil, nil)
		assert.Equal(t, 30*time.Second, handler.httpClient.Timeout)
	})
}

// =============================================================================
// SAMLProviderConfig Struct Tests
// =============================================================================

func TestSAMLProviderConfig_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		id := uuid.New()
		now := time.Now()
		metadataURL := "https://idp.example.com/metadata"
		metadataXML := "<xml>test</xml>"
		entityID := "https://idp.example.com"
		ssoURL := "https://idp.example.com/sso"

		config := SAMLProviderConfig{
			ID:                   id,
			Name:                 "okta",
			DisplayName:          "Okta SSO",
			Enabled:              true,
			EntityID:             "https://sp.example.com/saml/okta",
			AcsURL:               "https://sp.example.com/api/v1/auth/saml/acs",
			IdPMetadataURL:       &metadataURL,
			IdPMetadataXML:       &metadataXML,
			IdPEntityID:          &entityID,
			IdPSsoURL:            &ssoURL,
			AttributeMapping:     map[string]string{"email": "email_claim"},
			AutoCreateUsers:      true,
			DefaultRole:          "authenticated",
			AllowDashboardLogin:  true,
			AllowAppLogin:        true,
			AllowIDPInitiated:    false,
			AllowedRedirectHosts: []string{"app.example.com"},
			RequiredGroups:       []string{"users"},
			RequiredGroupsAll:    []string{"admin"},
			DeniedGroups:         []string{"blocked"},
			GroupAttribute:       "groups",
			Source:               "database",
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		assert.Equal(t, id, config.ID)
		assert.Equal(t, "okta", config.Name)
		assert.Equal(t, "Okta SSO", config.DisplayName)
		assert.True(t, config.Enabled)
		assert.Equal(t, "https://sp.example.com/saml/okta", config.EntityID)
		assert.Equal(t, "https://sp.example.com/api/v1/auth/saml/acs", config.AcsURL)
		assert.Equal(t, &metadataURL, config.IdPMetadataURL)
		assert.Equal(t, &metadataXML, config.IdPMetadataXML)
		assert.Equal(t, &entityID, config.IdPEntityID)
		assert.Equal(t, &ssoURL, config.IdPSsoURL)
		assert.Equal(t, "email_claim", config.AttributeMapping["email"])
		assert.True(t, config.AutoCreateUsers)
		assert.Equal(t, "authenticated", config.DefaultRole)
		assert.True(t, config.AllowDashboardLogin)
		assert.True(t, config.AllowAppLogin)
		assert.False(t, config.AllowIDPInitiated)
		assert.Contains(t, config.AllowedRedirectHosts, "app.example.com")
		assert.Contains(t, config.RequiredGroups, "users")
		assert.Contains(t, config.RequiredGroupsAll, "admin")
		assert.Contains(t, config.DeniedGroups, "blocked")
		assert.Equal(t, "groups", config.GroupAttribute)
		assert.Equal(t, "database", config.Source)
	})

	t.Run("JSON serialization", func(t *testing.T) {
		config := SAMLProviderConfig{
			ID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			Name:        "azure-ad",
			DisplayName: "Azure Active Directory",
			Enabled:     true,
			Source:      "config",
		}

		data, err := json.Marshal(config)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"id":"550e8400-e29b-41d4-a716-446655440000"`)
		assert.Contains(t, string(data), `"name":"azure-ad"`)
		assert.Contains(t, string(data), `"display_name":"Azure Active Directory"`)
		assert.Contains(t, string(data), `"enabled":true`)
		assert.Contains(t, string(data), `"source":"config"`)
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{
			"id": "550e8400-e29b-41d4-a716-446655440000",
			"name": "google-workspace",
			"display_name": "Google Workspace",
			"enabled": true,
			"entity_id": "https://sp.example.com/saml/google",
			"acs_url": "https://sp.example.com/api/v1/auth/saml/acs",
			"auto_create_users": true,
			"default_role": "user",
			"allow_dashboard_login": false,
			"allow_app_login": true,
			"allow_idp_initiated": true,
			"source": "database"
		}`

		var config SAMLProviderConfig
		err := json.Unmarshal([]byte(jsonData), &config)
		require.NoError(t, err)

		assert.Equal(t, "google-workspace", config.Name)
		assert.Equal(t, "Google Workspace", config.DisplayName)
		assert.True(t, config.Enabled)
		assert.True(t, config.AutoCreateUsers)
		assert.False(t, config.AllowDashboardLogin)
		assert.True(t, config.AllowAppLogin)
		assert.True(t, config.AllowIDPInitiated)
	})
}

// =============================================================================
// CreateSAMLProviderRequest Struct Tests
// =============================================================================

func TestCreateSAMLProviderRequest_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		metadataURL := "https://idp.example.com/metadata"
		autoCreate := true
		defaultRole := "authenticated"
		allowDashboard := true
		allowApp := true
		allowIDP := false
		groupAttr := "memberOf"

		req := CreateSAMLProviderRequest{
			Name:                 "okta-provider",
			DisplayName:          "Okta Provider",
			Enabled:              true,
			IdPMetadataURL:       &metadataURL,
			AttributeMapping:     map[string]string{"email": "emailClaim"},
			AutoCreateUsers:      &autoCreate,
			DefaultRole:          &defaultRole,
			AllowDashboardLogin:  &allowDashboard,
			AllowAppLogin:        &allowApp,
			AllowIDPInitiated:    &allowIDP,
			AllowedRedirectHosts: []string{"app.example.com", "admin.example.com"},
			RequiredGroups:       []string{"developers"},
			RequiredGroupsAll:    []string{"admin", "security"},
			DeniedGroups:         []string{"blocked"},
			GroupAttribute:       &groupAttr,
		}

		assert.Equal(t, "okta-provider", req.Name)
		assert.Equal(t, "Okta Provider", req.DisplayName)
		assert.True(t, req.Enabled)
		assert.Equal(t, &metadataURL, req.IdPMetadataURL)
		assert.Equal(t, &autoCreate, req.AutoCreateUsers)
		assert.Equal(t, &defaultRole, req.DefaultRole)
		assert.Equal(t, &allowDashboard, req.AllowDashboardLogin)
		assert.Equal(t, &allowApp, req.AllowAppLogin)
		assert.Equal(t, &allowIDP, req.AllowIDPInitiated)
		assert.Len(t, req.AllowedRedirectHosts, 2)
		assert.Contains(t, req.RequiredGroups, "developers")
	})

	t.Run("JSON deserialization with metadata URL", func(t *testing.T) {
		jsonData := `{
			"name": "azure-ad",
			"display_name": "Azure AD",
			"enabled": true,
			"idp_metadata_url": "https://login.microsoftonline.com/metadata",
			"attribute_mapping": {
				"email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
				"name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name"
			},
			"auto_create_users": true,
			"default_role": "authenticated"
		}`

		var req CreateSAMLProviderRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "azure-ad", req.Name)
		assert.Equal(t, "Azure AD", req.DisplayName)
		assert.True(t, req.Enabled)
		assert.NotNil(t, req.IdPMetadataURL)
		assert.Equal(t, "https://login.microsoftonline.com/metadata", *req.IdPMetadataURL)
		assert.Len(t, req.AttributeMapping, 2)
	})

	t.Run("JSON deserialization with metadata XML", func(t *testing.T) {
		jsonData := `{
			"name": "custom-idp",
			"enabled": false,
			"idp_metadata_xml": "<EntityDescriptor>...</EntityDescriptor>"
		}`

		var req CreateSAMLProviderRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "custom-idp", req.Name)
		assert.False(t, req.Enabled)
		assert.Nil(t, req.IdPMetadataURL)
		assert.NotNil(t, req.IdPMetadataXML)
		assert.Contains(t, *req.IdPMetadataXML, "EntityDescriptor")
	})

	t.Run("minimal request", func(t *testing.T) {
		jsonData := `{"name": "simple"}`

		var req CreateSAMLProviderRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "simple", req.Name)
		assert.False(t, req.Enabled)
		assert.Nil(t, req.IdPMetadataURL)
		assert.Nil(t, req.IdPMetadataXML)
	})
}

// =============================================================================
// UpdateSAMLProviderRequest Struct Tests
// =============================================================================

func TestUpdateSAMLProviderRequest_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		displayName := "Updated Display Name"
		enabled := false
		metadataURL := "https://new-idp.example.com/metadata"
		metadataXML := "<NewMetadata/>"
		autoCreate := false
		defaultRole := "admin"
		allowDashboard := true
		allowApp := false
		allowIDP := true
		groupAttr := "groups"

		req := UpdateSAMLProviderRequest{
			DisplayName:          &displayName,
			Enabled:              &enabled,
			IdPMetadataURL:       &metadataURL,
			IdPMetadataXML:       &metadataXML,
			AttributeMapping:     map[string]string{"email": "new_claim"},
			AutoCreateUsers:      &autoCreate,
			DefaultRole:          &defaultRole,
			AllowDashboardLogin:  &allowDashboard,
			AllowAppLogin:        &allowApp,
			AllowIDPInitiated:    &allowIDP,
			AllowedRedirectHosts: []string{"new.example.com"},
			RequiredGroups:       []string{"updated-group"},
			RequiredGroupsAll:    []string{"all-required"},
			DeniedGroups:         []string{"deny-this"},
			GroupAttribute:       &groupAttr,
		}

		assert.Equal(t, &displayName, req.DisplayName)
		assert.Equal(t, &enabled, req.Enabled)
		assert.Equal(t, &metadataURL, req.IdPMetadataURL)
		assert.Equal(t, &metadataXML, req.IdPMetadataXML)
		assert.Equal(t, "new_claim", req.AttributeMapping["email"])
	})

	t.Run("JSON deserialization partial update", func(t *testing.T) {
		jsonData := `{
			"display_name": "New Name",
			"enabled": false
		}`

		var req UpdateSAMLProviderRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.NotNil(t, req.DisplayName)
		assert.Equal(t, "New Name", *req.DisplayName)
		assert.NotNil(t, req.Enabled)
		assert.False(t, *req.Enabled)
		assert.Nil(t, req.IdPMetadataURL)
		assert.Nil(t, req.AttributeMapping)
	})

	t.Run("empty update request", func(t *testing.T) {
		jsonData := `{}`

		var req UpdateSAMLProviderRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Nil(t, req.DisplayName)
		assert.Nil(t, req.Enabled)
	})
}

// =============================================================================
// ValidateMetadataRequest Struct Tests
// =============================================================================

func TestValidateMetadataRequest_Struct(t *testing.T) {
	t.Run("with metadata URL", func(t *testing.T) {
		url := "https://idp.example.com/metadata"
		req := ValidateMetadataRequest{
			MetadataURL: &url,
		}

		assert.NotNil(t, req.MetadataURL)
		assert.Equal(t, "https://idp.example.com/metadata", *req.MetadataURL)
		assert.Nil(t, req.MetadataXML)
	})

	t.Run("with metadata XML", func(t *testing.T) {
		xml := "<EntityDescriptor>...</EntityDescriptor>"
		req := ValidateMetadataRequest{
			MetadataXML: &xml,
		}

		assert.Nil(t, req.MetadataURL)
		assert.NotNil(t, req.MetadataXML)
		assert.Contains(t, *req.MetadataXML, "EntityDescriptor")
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{"metadata_url": "https://example.com/saml/metadata"}`

		var req ValidateMetadataRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.NotNil(t, req.MetadataURL)
		assert.Equal(t, "https://example.com/saml/metadata", *req.MetadataURL)
	})
}

// =============================================================================
// ValidateMetadataResponse Struct Tests
// =============================================================================

func TestValidateMetadataResponse_Struct(t *testing.T) {
	t.Run("valid metadata response", func(t *testing.T) {
		resp := ValidateMetadataResponse{
			Valid:       true,
			EntityID:    "https://idp.example.com/entity",
			SsoURL:      "https://idp.example.com/sso",
			SloURL:      "https://idp.example.com/slo",
			Certificate: "MIIC...truncated...xyz=",
		}

		assert.True(t, resp.Valid)
		assert.Equal(t, "https://idp.example.com/entity", resp.EntityID)
		assert.Equal(t, "https://idp.example.com/sso", resp.SsoURL)
		assert.Equal(t, "https://idp.example.com/slo", resp.SloURL)
		assert.Equal(t, "MIIC...truncated...xyz=", resp.Certificate)
		assert.Nil(t, resp.Error)
	})

	t.Run("invalid metadata response", func(t *testing.T) {
		errMsg := "Invalid XML format"
		resp := ValidateMetadataResponse{
			Valid: false,
			Error: &errMsg,
		}

		assert.False(t, resp.Valid)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, "Invalid XML format", *resp.Error)
		assert.Empty(t, resp.EntityID)
	})

	t.Run("JSON serialization valid response", func(t *testing.T) {
		resp := ValidateMetadataResponse{
			Valid:    true,
			EntityID: "https://idp.test.com",
			SsoURL:   "https://idp.test.com/sso",
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"valid":true`)
		assert.Contains(t, string(data), `"entity_id":"https://idp.test.com"`)
		assert.Contains(t, string(data), `"sso_url":"https://idp.test.com/sso"`)
	})

	t.Run("JSON serialization error response", func(t *testing.T) {
		errMsg := "Connection timeout"
		resp := ValidateMetadataResponse{
			Valid: false,
			Error: &errMsg,
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"valid":false`)
		assert.Contains(t, string(data), `"error":"Connection timeout"`)
	})
}

// =============================================================================
// metadataValidationResult Struct Tests
// =============================================================================

func TestMetadataValidationResult_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		result := metadataValidationResult{
			EntityID:    "https://idp.example.com/entity",
			SsoURL:      "https://idp.example.com/sso",
			SloURL:      "https://idp.example.com/slo",
			Certificate: "MIIBkTCB+wIJAKHBfpQ...",
			CachedXML:   "<EntityDescriptor>...</EntityDescriptor>",
		}

		assert.Equal(t, "https://idp.example.com/entity", result.EntityID)
		assert.Equal(t, "https://idp.example.com/sso", result.SsoURL)
		assert.Equal(t, "https://idp.example.com/slo", result.SloURL)
		assert.Contains(t, result.Certificate, "MIIB")
		assert.Contains(t, result.CachedXML, "EntityDescriptor")
	})
}

// =============================================================================
// SAML Provider Name Pattern Tests
// =============================================================================

func TestSAMLProviderNamePattern(t *testing.T) {
	validNames := []string{
		"okta",
		"azure-ad",
		"google_workspace",
		"my-idp-123",
		"aa", // minimum 2 chars (after first letter)
		"provider_with_underscores",
		"provider-with-hyphens",
		"a1", // letter followed by number
	}

	invalidNames := []string{
		"",              // empty
		"a",             // too short (must be 2-50 chars)
		"1okta",         // starts with number
		"Okta",          // starts with uppercase
		"OKTA",          // all uppercase
		"-okta",         // starts with hyphen
		"_okta",         // starts with underscore
		"okta!",         // special character
		"okta provider", // space
		"okta.provider", // dot
		"okta@provider", // @ sign
		"this-is-a-very-long-provider-name-that-exceeds-fifty-characters-limit", // too long
	}

	for _, name := range validNames {
		t.Run("valid: "+name, func(t *testing.T) {
			assert.True(t, samlProviderNamePattern.MatchString(name), "Expected %q to be valid", name)
		})
	}

	for _, name := range invalidNames {
		t.Run("invalid: "+name, func(t *testing.T) {
			assert.False(t, samlProviderNamePattern.MatchString(name), "Expected %q to be invalid", name)
		})
	}
}

// =============================================================================
// CreateSAMLProvider Handler Validation Tests
// =============================================================================

func TestCreateSAMLProvider_Validation(t *testing.T) {
	t.Run("invalid request body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSAMLProviderHandler(nil, nil)

		app.Post("/saml/providers", handler.CreateSAMLProvider)

		req := httptest.NewRequest(http.MethodPost, "/saml/providers", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(body, &result)
		assert.Contains(t, result["error"], "Invalid request body")
	})

	t.Run("invalid provider name format", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSAMLProviderHandler(nil, nil)

		app.Post("/saml/providers", handler.CreateSAMLProvider)

		body := `{"name": "Invalid Name!", "idp_metadata_url": "https://example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/saml/providers", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Contains(t, result["error"], "Provider name must start with a letter")
	})

	t.Run("name starts with uppercase", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSAMLProviderHandler(nil, nil)

		app.Post("/saml/providers", handler.CreateSAMLProvider)

		body := `{"name": "Okta", "idp_metadata_url": "https://example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/saml/providers", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("missing metadata URL and XML", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSAMLProviderHandler(nil, nil)

		app.Post("/saml/providers", handler.CreateSAMLProvider)

		body := `{"name": "valid-name", "enabled": true}`
		req := httptest.NewRequest(http.MethodPost, "/saml/providers", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Contains(t, result["error"], "Either idp_metadata_url or idp_metadata_xml must be provided")
	})

	t.Run("empty metadata URL and XML", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSAMLProviderHandler(nil, nil)

		app.Post("/saml/providers", handler.CreateSAMLProvider)

		body := `{"name": "valid-name", "idp_metadata_url": "", "idp_metadata_xml": ""}`
		req := httptest.NewRequest(http.MethodPost, "/saml/providers", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// GetSAMLProvider Handler Validation Tests
// =============================================================================

func TestGetSAMLProvider_Validation(t *testing.T) {
	t.Run("invalid provider ID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSAMLProviderHandler(nil, nil)

		app.Get("/saml/providers/:id", handler.GetSAMLProvider)

		req := httptest.NewRequest(http.MethodGet, "/saml/providers/not-a-uuid", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(body, &result)
		assert.Equal(t, "Invalid provider ID", result["error"])
	})

	t.Run("valid UUID format is accepted", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSAMLProviderHandler(nil, nil)

		app.Get("/saml/providers/:id", handler.GetSAMLProvider)

		// Using a valid UUID - will fail at DB level (nil db) but not at validation
		req := httptest.NewRequest(http.MethodGet, "/saml/providers/550e8400-e29b-41d4-a716-446655440000", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		// Should not be 400 (validation passed)
		assert.NotEqual(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// UpdateSAMLProvider Handler Validation Tests
// =============================================================================

func TestUpdateSAMLProvider_Validation(t *testing.T) {
	t.Run("invalid provider ID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSAMLProviderHandler(nil, nil)

		app.Put("/saml/providers/:id", handler.UpdateSAMLProvider)

		body := `{"display_name": "Updated Name"}`
		req := httptest.NewRequest(http.MethodPut, "/saml/providers/invalid-uuid", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Equal(t, "Invalid provider ID", result["error"])
	})
}

// =============================================================================
// DeleteSAMLProvider Handler Validation Tests
// =============================================================================

func TestDeleteSAMLProvider_Validation(t *testing.T) {
	t.Run("invalid provider ID", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSAMLProviderHandler(nil, nil)

		app.Delete("/saml/providers/:id", handler.DeleteSAMLProvider)

		req := httptest.NewRequest(http.MethodDelete, "/saml/providers/bad-uuid", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(body, &result)
		assert.Equal(t, "Invalid provider ID", result["error"])
	})
}

// =============================================================================
// ValidateMetadata Handler Validation Tests
// =============================================================================

func TestValidateMetadata_Validation(t *testing.T) {
	t.Run("invalid request body", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSAMLProviderHandler(nil, nil)

		app.Post("/saml/validate-metadata", handler.ValidateMetadata)

		req := httptest.NewRequest(http.MethodPost, "/saml/validate-metadata", bytes.NewReader([]byte("{invalid")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(body, &result)
		assert.Contains(t, result["error"], "Invalid request body")
	})

	t.Run("missing both URL and XML", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSAMLProviderHandler(nil, nil)

		app.Post("/saml/validate-metadata", handler.ValidateMetadata)

		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/saml/validate-metadata", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		_ = json.Unmarshal(respBody, &result)
		assert.Contains(t, result["error"], "Either metadata_url or metadata_xml must be provided")
	})

	t.Run("empty URL and XML", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSAMLProviderHandler(nil, nil)

		app.Post("/saml/validate-metadata", handler.ValidateMetadata)

		body := `{"metadata_url": "", "metadata_xml": ""}`
		req := httptest.NewRequest(http.MethodPost, "/saml/validate-metadata", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// =============================================================================
// GetSPMetadata Handler Validation Tests
// =============================================================================

func TestGetSPMetadata_Validation(t *testing.T) {
	t.Run("no SAML service returns 503", func(t *testing.T) {
		app := newTestApp(t)
		handler := NewSAMLProviderHandler(nil, nil)

		app.Get("/saml/metadata/:provider", handler.GetSPMetadata)

		req := httptest.NewRequest(http.MethodGet, "/saml/metadata/okta", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "NOT_INITIALIZED")
	})
}

// =============================================================================
// JSON Serialization Tests
// =============================================================================

func TestSAMLRequests_JSONSerialization(t *testing.T) {
	t.Run("CreateSAMLProviderRequest serializes correctly", func(t *testing.T) {
		metadataURL := "https://idp.example.com/metadata"
		autoCreate := true

		req := CreateSAMLProviderRequest{
			Name:            "test-provider",
			DisplayName:     "Test Provider",
			Enabled:         true,
			IdPMetadataURL:  &metadataURL,
			AutoCreateUsers: &autoCreate,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"name":"test-provider"`)
		assert.Contains(t, string(data), `"display_name":"Test Provider"`)
		assert.Contains(t, string(data), `"enabled":true`)
		assert.Contains(t, string(data), `"idp_metadata_url":"https://idp.example.com/metadata"`)
		assert.Contains(t, string(data), `"auto_create_users":true`)
	})

	t.Run("UpdateSAMLProviderRequest serializes correctly", func(t *testing.T) {
		displayName := "Updated Name"
		enabled := false

		req := UpdateSAMLProviderRequest{
			DisplayName: &displayName,
			Enabled:     &enabled,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"display_name":"Updated Name"`)
		assert.Contains(t, string(data), `"enabled":false`)
	})

	t.Run("ValidateMetadataRequest serializes correctly", func(t *testing.T) {
		url := "https://test.example.com/metadata"
		req := ValidateMetadataRequest{
			MetadataURL: &url,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"metadata_url":"https://test.example.com/metadata"`)
	})
}

// =============================================================================
// Provider Source Tests
// =============================================================================

func TestProviderSource(t *testing.T) {
	t.Run("database source provider", func(t *testing.T) {
		config := SAMLProviderConfig{
			Name:   "db-provider",
			Source: "database",
		}

		assert.Equal(t, "database", config.Source)
	})

	t.Run("config source provider", func(t *testing.T) {
		config := SAMLProviderConfig{
			Name:   "config-provider",
			Source: "config",
		}

		assert.Equal(t, "config", config.Source)
	})
}

// =============================================================================
// Attribute Mapping Tests
// =============================================================================

func TestAttributeMapping(t *testing.T) {
	t.Run("common attribute mappings", func(t *testing.T) {
		mapping := map[string]string{
			"email":       "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			"name":        "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
			"given_name":  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname",
			"family_name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname",
		}

		config := SAMLProviderConfig{
			AttributeMapping: mapping,
		}

		assert.Len(t, config.AttributeMapping, 4)
		assert.Contains(t, config.AttributeMapping["email"], "emailaddress")
	})

	t.Run("custom attribute mapping", func(t *testing.T) {
		mapping := map[string]string{
			"email":       "customEmailClaim",
			"department":  "customDepartment",
			"employee_id": "customEmployeeId",
		}

		config := SAMLProviderConfig{
			AttributeMapping: mapping,
		}

		assert.Equal(t, "customEmailClaim", config.AttributeMapping["email"])
		assert.Equal(t, "customDepartment", config.AttributeMapping["department"])
	})

	t.Run("empty attribute mapping", func(t *testing.T) {
		config := SAMLProviderConfig{
			AttributeMapping: nil,
		}

		assert.Nil(t, config.AttributeMapping)
	})
}

// =============================================================================
// Group Authorization Tests
// =============================================================================

func TestGroupAuthorization(t *testing.T) {
	t.Run("required groups any", func(t *testing.T) {
		config := SAMLProviderConfig{
			RequiredGroups: []string{"developers", "admins"},
		}

		assert.Len(t, config.RequiredGroups, 2)
		assert.Contains(t, config.RequiredGroups, "developers")
		assert.Contains(t, config.RequiredGroups, "admins")
	})

	t.Run("required groups all", func(t *testing.T) {
		config := SAMLProviderConfig{
			RequiredGroupsAll: []string{"verified", "active"},
		}

		assert.Len(t, config.RequiredGroupsAll, 2)
		assert.Contains(t, config.RequiredGroupsAll, "verified")
		assert.Contains(t, config.RequiredGroupsAll, "active")
	})

	t.Run("denied groups", func(t *testing.T) {
		config := SAMLProviderConfig{
			DeniedGroups: []string{"blocked", "suspended"},
		}

		assert.Len(t, config.DeniedGroups, 2)
		assert.Contains(t, config.DeniedGroups, "blocked")
	})

	t.Run("custom group attribute", func(t *testing.T) {
		config := SAMLProviderConfig{
			GroupAttribute: "memberOf",
		}

		assert.Equal(t, "memberOf", config.GroupAttribute)
	})
}

// =============================================================================
// Login Permission Tests
// =============================================================================

func TestLoginPermissions(t *testing.T) {
	t.Run("dashboard only login", func(t *testing.T) {
		config := SAMLProviderConfig{
			AllowDashboardLogin: true,
			AllowAppLogin:       false,
		}

		assert.True(t, config.AllowDashboardLogin)
		assert.False(t, config.AllowAppLogin)
	})

	t.Run("app only login", func(t *testing.T) {
		config := SAMLProviderConfig{
			AllowDashboardLogin: false,
			AllowAppLogin:       true,
		}

		assert.False(t, config.AllowDashboardLogin)
		assert.True(t, config.AllowAppLogin)
	})

	t.Run("both logins allowed", func(t *testing.T) {
		config := SAMLProviderConfig{
			AllowDashboardLogin: true,
			AllowAppLogin:       true,
		}

		assert.True(t, config.AllowDashboardLogin)
		assert.True(t, config.AllowAppLogin)
	})

	t.Run("IDP initiated login", func(t *testing.T) {
		config := SAMLProviderConfig{
			AllowIDPInitiated: true,
		}

		assert.True(t, config.AllowIDPInitiated)
	})
}

// =============================================================================
// Redirect Host Tests
// =============================================================================

func TestAllowedRedirectHosts(t *testing.T) {
	t.Run("multiple redirect hosts", func(t *testing.T) {
		config := SAMLProviderConfig{
			AllowedRedirectHosts: []string{
				"app.example.com",
				"admin.example.com",
				"localhost:3000",
			},
		}

		assert.Len(t, config.AllowedRedirectHosts, 3)
		assert.Contains(t, config.AllowedRedirectHosts, "app.example.com")
		assert.Contains(t, config.AllowedRedirectHosts, "localhost:3000")
	})

	t.Run("empty redirect hosts", func(t *testing.T) {
		config := SAMLProviderConfig{
			AllowedRedirectHosts: []string{},
		}

		assert.Empty(t, config.AllowedRedirectHosts)
	})
}
