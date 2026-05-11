//go:build integration

package e2e

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/test"
)

func TestSSRFProtection_PublicURLControl(t *testing.T) {
	rateLimiter, pubSub := test.NewInMemoryDependencies()
	tc := test.NewTestContextWithOptions(t, test.TestContextOptions{
		RateLimiter: rateLimiter,
		PubSub:      pubSub,
	})
	defer tc.Close()
	tc.EnsureAuthSchema()

	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-ssrf-ctrl-%d@test.com", timestamp)
	_, adminToken := tc.CreateDashboardAdminUser(email, "adminpass123456")

	t.Run("function_without_fetch_succeeds", func(t *testing.T) {
		functionName := fmt.Sprintf("test_ssrf_ctrl_ok_%d", timestamp)
		createTestFunction(t, tc, adminToken, functionName, `export default async function handler(req) {
			return new Response(JSON.stringify({ message: "hello", network: "allowed" }), {
				headers: { "Content-Type": "application/json" }
			});
		}`)

		result := invokeFunction(t, tc, adminToken, functionName)
		msg, ok := result["message"].(string)
		require.True(t, ok, "Response should contain message")
		assert.Equal(t, "hello", msg)
	})

	t.Run("function_can_fetch_public_url", func(t *testing.T) {
		functionName := fmt.Sprintf("test_ssrf_ctrl_fetch_%d", timestamp)
		createTestFunction(t, tc, adminToken, functionName, `export default async function handler(req) {
			try {
				const controller = new AbortController();
				const tid = setTimeout(() => controller.abort(), 10000);
				const resp = await fetch('https://example.com', { signal: controller.signal });
				clearTimeout(tid);
				const text = await resp.text();
				return new Response(JSON.stringify({ fetched: true, status: resp.status, hasBody: text.length > 0 }), {
					headers: { "Content-Type": "application/json" }
				});
			} catch (e) {
				return new Response(JSON.stringify({ fetched: false, error: e.message }), {
					status: 502,
					headers: { "Content-Type": "application/json" }
				});
			}
		}`)

		resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
			WithAuth(adminToken).
			WithJSON(map[string]interface{}{"test": "control"}).
			Send()

		var result map[string]interface{}
		err := json.Unmarshal(resp.Body(), &result)
		require.NoError(t, err)

		if resp.Status() == fiber.StatusOK {
			fetched, _ := result["fetched"].(bool)
			assert.True(t, fetched, "Function should be able to fetch public URL example.com")
			status, _ := result["status"].(float64)
			assert.Equal(t, float64(200), status, "Public URL should return HTTP 200")
			t.Logf("Public URL fetch succeeded: fetched=%v status=%v", fetched, status)
		} else {
			t.Logf("Public URL fetch returned status %d (may be network-limited environment): %v", resp.Status(), result)
		}
	})
}

func TestSSRFProtection_RestrictedAllowedDomains(t *testing.T) {
	rateLimiter, pubSub := test.NewInMemoryDependencies()
	tc := test.NewTestContextWithOptions(t, test.TestContextOptions{
		RateLimiter: rateLimiter,
		PubSub:      pubSub,
	})
	defer tc.Close()
	tc.EnsureAuthSchema()

	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-ssrf-restricted-%d@test.com", timestamp)
	_, adminToken := tc.CreateDashboardAdminUser(email, "adminpass123456")

	t.Run("function_with_explicit_allowed_domains_can_fetch_allowed", func(t *testing.T) {
		functionName := fmt.Sprintf("test_ssrf_allowed_%d", timestamp)
		resp := tc.NewRequest("POST", "/api/v1/functions").
			WithAuth(adminToken).
			WithJSON(map[string]interface{}{
				"name":    functionName,
				"code":    `export default async function handler(req) { try { const r = await fetch('https://example.com'); return new Response(JSON.stringify({ fetched: true, status: r.status }), { headers: { "Content-Type": "application/json" } }); } catch (e) { return new Response(JSON.stringify({ fetched: false, error: e.message }), { headers: { "Content-Type": "application/json" } }); } }`,
				"runtime": "deno",
				"enabled": true,
				"permissions": map[string]interface{}{
					"allow_net":       true,
					"allowed_domains": []string{"example.com"},
				},
			}).
			Send()
		require.Equal(t, fiber.StatusCreated, resp.Status(), "Failed to create function: %s", string(resp.Body()))

		result := invokeFunction(t, tc, adminToken, functionName)
		fetched, _ := result["fetched"].(bool)
		assert.True(t, fetched, "Function should be able to fetch explicitly allowed domain example.com")
	})

	t.Run("function_with_explicit_allowed_domains_cannot_fetch_blocked", func(t *testing.T) {
		functionName := fmt.Sprintf("test_ssrf_blocked_%d", timestamp)
		resp := tc.NewRequest("POST", "/api/v1/functions").
			WithAuth(adminToken).
			WithJSON(map[string]interface{}{
				"name":    functionName,
				"code":    `export default async function handler(req) { try { const r = await fetch('http://169.254.169.254/latest/meta-data/'); return new Response(JSON.stringify({ fetched: true, status: r.status }), { headers: { "Content-Type": "application/json" } }); } catch (e) { return new Response(JSON.stringify({ fetched: false, error: e.message }), { headers: { "Content-Type": "application/json" } }); } }`,
				"runtime": "deno",
				"enabled": true,
				"permissions": map[string]interface{}{
					"allow_net":       true,
					"allowed_domains": []string{"example.com"},
				},
			}).
			Send()
		require.Equal(t, fiber.StatusCreated, resp.Status(), "Failed to create function: %s", string(resp.Body()))

		result := invokeFunction(t, tc, adminToken, functionName)
		fetched, _ := result["fetched"].(bool)
		assert.False(t, fetched, "Function with allowlist must NOT fetch 169.254.169.254 (not in allowed_domains)")
		errMsg, _ := result["error"].(string)
		assert.NotEmpty(t, errMsg, "Error should be present when fetch to non-allowed domain fails")
		t.Logf("Blocked domain with allowlist: fetched=%v error=%s", fetched, errMsg)
	})
}
