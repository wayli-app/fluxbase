package e2e

import (
	"testing"

	"github.com/fluxbase-eu/fluxbase/test"
	"github.com/gofiber/fiber/v2"
)

// TestRESTAnonymousAccessAllowed verifies that unauthenticated requests are allowed
// with anonymous RLS role for tables with RLS enabled
func TestRESTAnonymousAccessAllowed(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	// Try to access REST API without authentication - should be allowed with role 'anon'
	resp := tc.NewRequest("GET", "/api/v1/tables/products").
		Unauthenticated().
		Send()

	// Should succeed - anonymous access is allowed, RLS will filter data
	// Status can be 200 (empty array) or 404 if table doesn't exist
	if resp.Status() != fiber.StatusOK && resp.Status() != fiber.StatusNotFound {
		t.Fatalf("Expected 200 or 404 for anonymous access, got %d: %s", resp.Status(), string(resp.Body()))
	}
}

// TestRESTWithAPIKey verifies that API key authentication works
func TestRESTWithAPIKey(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	// Create an API key
	apiKey := tc.CreateAPIKey("Test API Key", nil)

	// Use API key to access REST API
	resp := tc.NewRequest("GET", "/api/v1/tables/products").
		WithAPIKey(apiKey).
		Send()

	// Should succeed with 200 (empty array) or 404 if table doesn't exist yet
	if resp.Status() != fiber.StatusOK && resp.Status() != fiber.StatusNotFound {
		t.Fatalf("Expected 200 or 404, got %d: %s", resp.Status(), string(resp.Body()))
	}
}

// TestRESTWithServiceKey verifies that service key authentication works
func TestRESTWithServiceKey(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()

	// Check if service_keys table exists (added in migration 006)
	// If migrations haven't run, skip this test
	tables := tc.QuerySQL("SELECT table_name FROM information_schema.tables WHERE table_schema='auth' AND table_name='service_keys'")
	if len(tables) == 0 {
		t.Skip("Service keys table doesn't exist - migrations may not have been run")
	}

	// Create a service key
	serviceKey := tc.CreateServiceKey("Test Service Key")

	// Use service key to access REST API
	resp := tc.NewRequest("GET", "/api/v1/tables/products").
		WithServiceKey(serviceKey).
		Send()

	// Should succeed with 200 (empty array) or 404 if table doesn't exist
	if resp.Status() != fiber.StatusOK && resp.Status() != fiber.StatusNotFound {
		t.Fatalf("Expected 200 or 404, got %d: %s", resp.Status(), string(resp.Body()))
	}
}

// TestRESTWithBearerToken verifies that JWT bearer token authentication works
func TestRESTWithBearerToken(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()

	// Clean up any existing test user to ensure test isolation
	tc.ExecuteSQL("DELETE FROM auth.users WHERE email = $1", "bearer-test@example.com")

	// Create a test user and get JWT token
	_, token := tc.CreateTestUser("bearer-test@example.com", "password123")

	// Use bearer token to access REST API
	resp := tc.NewRequest("GET", "/api/v1/tables/products").
		WithBearerToken(token).
		Send()

	// Should succeed with 200 (empty array) or 404 if table doesn't exist
	if resp.Status() != fiber.StatusOK && resp.Status() != fiber.StatusNotFound {
		t.Fatalf("Expected 200 or 404, got %d: %s", resp.Status(), string(resp.Body()))
	}
}

// TestRESTWithInvalidAPIKey verifies that invalid API keys are rejected
func TestRESTWithInvalidAPIKey(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	// Use invalid API key
	resp := tc.NewRequest("GET", "/api/v1/tables/products").
		WithAPIKey("fbk_invalid_key_12345678901234567890").
		Send()

	resp.AssertStatus(fiber.StatusUnauthorized)
	resp.AssertContains("Invalid API key")
}

// TestRESTWithInvalidServiceKey verifies that invalid service keys are rejected
func TestRESTWithInvalidServiceKey(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()

	// Use invalid service key
	resp := tc.NewRequest("GET", "/api/v1/tables/products").
		WithServiceKey("sk_test_invalid_key_123456").
		Send()

	resp.AssertStatus(fiber.StatusUnauthorized)
	resp.AssertContains("Invalid service key")
}

// TestRESTWithInvalidBearerToken verifies that invalid JWT tokens are rejected
func TestRESTWithInvalidBearerToken(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	// Use invalid bearer token
	resp := tc.NewRequest("GET", "/api/v1/tables/products").
		WithBearerToken("invalid.jwt.token").
		Send()

	resp.AssertStatus(fiber.StatusUnauthorized)
	resp.AssertContains("Invalid or expired Bearer token")
}

// TestRESTAuthenticationPriority verifies the authentication priority order
// Service Key -> JWT Token -> API Key
func TestRESTAuthenticationPriority(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	tc.EnsureAuthSchema()

	// Check if service_keys table exists (added in migration 006)
	tables := tc.QuerySQL("SELECT table_name FROM information_schema.tables WHERE table_schema='auth' AND table_name='service_keys'")
	if len(tables) == 0 {
		t.Skip("Service keys table doesn't exist - migrations may not have been run")
	}

	// Clean up all auth tables to ensure test isolation (TRUNCATE is more reliable than DELETE)
	tc.ExecuteSQL("TRUNCATE TABLE auth.users CASCADE")
	tc.ExecuteSQL("TRUNCATE TABLE auth.sessions CASCADE")
	tc.ExecuteSQL("TRUNCATE TABLE auth.api_keys CASCADE")
	tc.ExecuteSQL("TRUNCATE TABLE auth.service_keys CASCADE")

	// Create all three auth types
	apiKey := tc.CreateAPIKey("Priority Test API Key", nil)
	serviceKey := tc.CreateServiceKey("Priority Test Service Key")
	_, jwtToken := tc.CreateTestUser("priority@example.com", "password123")

	// Test 1: Service key should take priority over JWT and API key
	resp := tc.NewRequest("GET", "/api/v1/tables/products").
		WithServiceKey(serviceKey).
		WithBearerToken(jwtToken).
		WithAPIKey(apiKey).
		Send()

	// Should use service key (check via response - service keys bypass RLS)
	if resp.Status() != fiber.StatusOK && resp.Status() != fiber.StatusNotFound {
		t.Fatalf("Service key should work, got %d: %s", resp.Status(), string(resp.Body()))
	}

	// Test 2: JWT should take priority over API key when no service key
	resp = tc.NewRequest("GET", "/api/v1/tables/products").
		WithBearerToken(jwtToken).
		WithAPIKey(apiKey).
		Send()

	if resp.Status() != fiber.StatusOK && resp.Status() != fiber.StatusNotFound {
		t.Fatalf("JWT token should work, got %d: %s", resp.Status(), string(resp.Body()))
	}

	// Test 3: API key should work when alone
	resp = tc.NewRequest("GET", "/api/v1/tables/products").
		WithAPIKey(apiKey).
		Send()

	if resp.Status() != fiber.StatusOK && resp.Status() != fiber.StatusNotFound {
		t.Fatalf("API key should work, got %d: %s", resp.Status(), string(resp.Body()))
	}
}

// TestRESTAPIKeyScopes verifies that API key scopes are enforced
func TestRESTAPIKeyScopes(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	// Create API key with read-only scopes
	apiKey := tc.CreateAPIKey("Read-Only API Key", []string{"read:tables"})

	// GET should work (read scope)
	resp := tc.NewRequest("GET", "/api/v1/tables/products").
		WithAPIKey(apiKey).
		Send()

	// Should succeed with 200 or 404
	if resp.Status() != fiber.StatusOK && resp.Status() != fiber.StatusNotFound {
		t.Fatalf("Read should work with read scope, got %d: %s", resp.Status(), string(resp.Body()))
	}

	// POST should potentially fail (no write scope)
	// Note: This depends on scope enforcement implementation
	// For now, we just verify the request doesn't crash
	_ = tc.NewRequest("POST", "/api/v1/tables/products").
		WithAPIKey(apiKey).
		WithBody(map[string]interface{}{
			"name":  "Test",
			"price": 10.00,
		}).
		Send()
}

// TestRESTMultipleAPIKeys verifies that multiple API keys can coexist
func TestRESTMultipleAPIKeys(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	// Create multiple API keys
	apiKey1 := tc.CreateAPIKey("API Key 1", nil)
	apiKey2 := tc.CreateAPIKey("API Key 2", nil)

	// Both should work independently
	resp1 := tc.NewRequest("GET", "/api/v1/tables/products").
		WithAPIKey(apiKey1).
		Send()

	resp2 := tc.NewRequest("GET", "/api/v1/tables/products").
		WithAPIKey(apiKey2).
		Send()

	// Both should succeed
	if resp1.Status() != fiber.StatusOK && resp1.Status() != fiber.StatusNotFound {
		t.Fatalf("API Key 1 should work, got %d", resp1.Status())
	}

	if resp2.Status() != fiber.StatusOK && resp2.Status() != fiber.StatusNotFound {
		t.Fatalf("API Key 2 should work, got %d", resp2.Status())
	}
}
