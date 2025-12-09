// Package e2e tests the AI chatbot functionality
package e2e

import (
	"testing"

	test "github.com/fluxbase-eu/fluxbase/test"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// TestAIChatbotsListEndpoint tests the public chatbot listing endpoint
func TestAIChatbotsListEndpoint(t *testing.T) {
	// Skip if AI feature is not enabled
	t.Skip("AI feature requires AI configuration - skipping integration test")

	// GIVEN: A clean test environment
	tc := setupAIChatbotTest(t)
	defer tc.Close()

	apiKey := tc.CreateAPIKey("AI Test Key", nil)

	// WHEN: Listing available chatbots
	resp := tc.NewRequest("GET", "/api/v1/ai/chatbots").
		WithAPIKey(apiKey).
		Send()

	// THEN: Request succeeds
	resp.AssertStatus(fiber.StatusOK)

	// AND: Response contains chatbots array
	var result map[string]interface{}
	resp.JSON(&result)

	require.NotNil(t, result["chatbots"], "Response should contain chatbots array")
}

// TestAIChatbotsAdminSync tests the admin sync endpoint
func TestAIChatbotsAdminSync(t *testing.T) {
	// Skip if AI feature is not enabled
	t.Skip("AI feature requires AI configuration - skipping integration test")

	// GIVEN: An authenticated admin user
	tc := setupAIChatbotTest(t)
	defer tc.Close()

	tc.EnsureAuthSchema()
	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// WHEN: Syncing chatbots from filesystem
	resp := tc.NewRequest("POST", "/api/v1/admin/ai/chatbots/sync").
		WithAuth(adminToken).
		Send()

	// THEN: Sync succeeds
	resp.AssertStatus(fiber.StatusOK)

	// AND: Response contains sync results
	var result map[string]interface{}
	resp.JSON(&result)

	require.NotNil(t, result["created"], "Response should contain created count")
	require.NotNil(t, result["updated"], "Response should contain updated count")
}

// TestAIChatbotsAdminList tests the admin chatbot listing
func TestAIChatbotsAdminList(t *testing.T) {
	// Skip if AI feature is not enabled
	t.Skip("AI feature requires AI configuration - skipping integration test")

	// GIVEN: An authenticated admin user
	tc := setupAIChatbotTest(t)
	defer tc.Close()

	tc.EnsureAuthSchema()
	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// WHEN: Listing all chatbots as admin
	resp := tc.NewRequest("GET", "/api/v1/admin/ai/chatbots").
		WithAuth(adminToken).
		Send()

	// THEN: Request succeeds
	resp.AssertStatus(fiber.StatusOK)

	// AND: Response contains chatbots
	var result map[string]interface{}
	resp.JSON(&result)

	require.NotNil(t, result["chatbots"], "Response should contain chatbots array")
}

// TestAIProvidersAdminList tests the admin provider listing
func TestAIProvidersAdminList(t *testing.T) {
	// Skip if AI feature is not enabled
	t.Skip("AI feature requires AI configuration - skipping integration test")

	// GIVEN: An authenticated admin user
	tc := setupAIChatbotTest(t)
	defer tc.Close()

	tc.EnsureAuthSchema()
	adminToken := tc.GetDashboardAuthToken("admin@test.com", "SecurePassword123!")

	// WHEN: Listing providers
	resp := tc.NewRequest("GET", "/api/v1/admin/ai/providers").
		WithAuth(adminToken).
		Send()

	// THEN: Request succeeds
	resp.AssertStatus(fiber.StatusOK)

	// AND: Response contains providers array
	var result map[string]interface{}
	resp.JSON(&result)

	require.NotNil(t, result["providers"], "Response should contain providers array")
}

// TestAIUnauthenticatedAccess tests that unauthenticated access is blocked
func TestAIUnauthenticatedAccess(t *testing.T) {
	// Skip if AI feature is not enabled
	t.Skip("AI feature requires AI configuration - skipping integration test")

	// GIVEN: A test environment
	tc := setupAIChatbotTest(t)
	defer tc.Close()

	// WHEN: Attempting to access admin endpoint without authentication
	resp := tc.NewRequest("GET", "/api/v1/admin/ai/chatbots").
		Unauthenticated().
		Send()

	// THEN: Request is rejected
	resp.AssertStatus(fiber.StatusUnauthorized)
}

func setupAIChatbotTest(t *testing.T) *test.TestContext {
	tc := test.NewTestContext(t)
	return tc
}
