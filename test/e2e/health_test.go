package e2e

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/wayli-app/fluxbase/test"
)

// TestHealth verifies that the basic health check endpoint works
// This validates that TestContext and routing infrastructure are functioning
func TestHealth(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	// Test health endpoint
	resp := tc.NewRequest("GET", "/health").
		Send().
		AssertStatus(fiber.StatusOK)

	// Verify response contains status
	var result map[string]interface{}
	resp.JSON(&result)

	if status, ok := result["status"]; !ok || status != "ok" {
		t.Errorf("Expected status 'ok', got: %v", status)
	}
}

// TestHealthWithDatabase verifies health check with database connectivity
func TestHealthWithDatabase(t *testing.T) {
	tc := test.NewTestContext(t)
	defer tc.Close()

	// Test health endpoint - should confirm database is connected
	resp := tc.NewRequest("GET", "/health").
		Send().
		AssertStatus(fiber.StatusOK)

	// Verify response structure
	var result map[string]interface{}
	resp.JSON(&result)

	if status, ok := result["status"]; !ok || status != "ok" {
		t.Errorf("Expected status 'ok', got: %v", status)
	}

	// Verify database service is healthy
	if services, ok := result["services"].(map[string]interface{}); ok {
		if db, ok := services["database"].(map[string]interface{}); ok {
			if dbStatus, ok := db["healthy"].(bool); !ok || !dbStatus {
				t.Errorf("Expected database to be healthy, got: %v", db)
			}
		}
	}
}
