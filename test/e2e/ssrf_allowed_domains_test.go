//go:build integration

package e2e

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/test"
)

func TestSSRFProtection_AllowedDomains(t *testing.T) {
	rateLimiter, pubSub := test.NewInMemoryDependencies()
	tc := test.NewTestContextWithOptions(t, test.TestContextOptions{
		RateLimiter: rateLimiter,
		PubSub:      pubSub,
	})
	defer tc.Close()
	tc.EnsureAuthSchema()

	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-ssrf-domains-%d@test.com", timestamp)
	_, adminToken := tc.CreateDashboardAdminUser(email, "adminpass123456")

	t.Run("function with allowed_domains can fetch allowed domain", func(t *testing.T) {
		functionName := fmt.Sprintf("test_ssrf_allowed_%d", timestamp)
		resp := tc.NewRequest("POST", "/api/v1/functions").
			WithAuth(adminToken).
			WithJSON(map[string]interface{}{
				"name":           functionName,
				"code":           `export default async function handler(req) { try { const controller = new AbortController(); const tid = setTimeout(() => controller.abort(), 10000); const r = await fetch('https://example.com', { signal: controller.signal }); clearTimeout(tid); return new Response(JSON.stringify({ fetched: true, status: r.status }), { headers: { "Content-Type": "application/json" } }); } catch (e) { return new Response(JSON.stringify({ fetched: false, error: e.message }), { headers: { "Content-Type": "application/json" } }); } }`,
				"runtime":        "deno",
				"enabled":        true,
				"allow_net":      true,
				"allowed_domains": "example.com",
			}).
			Send()
		require.Equal(t, fiber.StatusCreated, resp.Status(), "Failed to create function: %s", string(resp.Body()))

		result := invokeFunction(t, tc, adminToken, functionName)
		fetched, _ := result["fetched"].(bool)
		assert.True(t, fetched, "Function should be able to fetch explicitly allowed domain example.com")
		status, _ := result["status"].(float64)
		assert.Equal(t, float64(200), status)
	})

	t.Run("function with allowed_domains cannot fetch blocked IP", func(t *testing.T) {
		functionName := fmt.Sprintf("test_ssrf_blocked_ip_%d", timestamp)
		resp := tc.NewRequest("POST", "/api/v1/functions").
			WithAuth(adminToken).
			WithJSON(map[string]interface{}{
				"name":            functionName,
				"code":            `export default async function handler(req) { try { const controller = new AbortController(); const tid = setTimeout(() => controller.abort(), 5000); const r = await fetch('http://169.254.169.254/latest/meta-data/', { signal: controller.signal }); clearTimeout(tid); return new Response(JSON.stringify({ fetched: true, status: r.status }), { headers: { "Content-Type": "application/json" } }); } catch (e) { return new Response(JSON.stringify({ fetched: false, error: e.message }), { headers: { "Content-Type": "application/json" } }); } }`,
				"runtime":         "deno",
				"enabled":         true,
				"allow_net":       true,
				"allowed_domains": "example.com",
			}).
			Send()
		require.Equal(t, fiber.StatusCreated, resp.Status(), "Failed to create function: %s", string(resp.Body()))

		result := invokeFunction(t, tc, adminToken, functionName)
		fetched, _ := result["fetched"].(bool)
		errMsg, _ := result["error"].(string)
		assert.False(t, fetched, "Function with allowed_domains=example.com must NOT fetch 169.254.169.254")
		if fetched {
			t.Logf("BUG: 169.254.169.254 was accessible despite allowlist (may be in cloud CI)")
		} else {
			assert.True(t,
				strings.Contains(errMsg, "error") ||
					strings.Contains(errMsg, "refused") ||
					strings.Contains(errMsg, "timed out") ||
					strings.Contains(errMsg, "abort") ||
					strings.Contains(errMsg, "failed") ||
					strings.Contains(errMsg, "ECONNREFUSED") ||
					strings.Contains(errMsg, "ENETUNREACH") ||
					strings.Contains(errMsg, "cancel") ||
					strings.Contains(errMsg, "NotAllowed") ||
					strings.Contains(errMsg, "Requires net access"),
				"Error should indicate connection failure: %s", errMsg)
			t.Logf("Allowed domains SSRF protection working: fetched=%v error=%s", fetched, errMsg)
		}
	})

	t.Run("function without allowed_domains is unrestricted", func(t *testing.T) {
		functionName := fmt.Sprintf("test_ssrf_unrestricted_%d", timestamp)
		resp := tc.NewRequest("POST", "/api/v1/functions").
			WithAuth(adminToken).
			WithJSON(map[string]interface{}{
				"name":    functionName,
				"code":    `export default async function handler(req) { try { const controller = new AbortController(); const tid = setTimeout(() => controller.abort(), 10000); const r = await fetch('https://example.com', { signal: controller.signal }); clearTimeout(tid); return new Response(JSON.stringify({ fetched: true, status: r.status }), { headers: { "Content-Type": "application/json" } }); } catch (e) { return new Response(JSON.stringify({ fetched: false, error: e.message }), { headers: { "Content-Type": "application/json" } }); } }`,
				"runtime": "deno",
				"enabled": true,
			}).
			Send()
		require.Equal(t, fiber.StatusCreated, resp.Status(), "Failed to create function: %s", string(resp.Body()))

		result := invokeFunction(t, tc, adminToken, functionName)
		fetched, _ := result["fetched"].(bool)
		assert.True(t, fetched, "Function without allowed_domains should have unrestricted network access")
	})

	t.Run("allowed_domains via annotation", func(t *testing.T) {
		functionName := fmt.Sprintf("test_ssrf_annotation_%d", timestamp)
		code := `/**
 * @fluxbase:allowed-domains example.com, httpbin.org
 */
export default async function handler(req) {
	try {
		const controller = new AbortController();
		const tid = setTimeout(() => controller.abort(), 10000);
		const r = await fetch('https://example.com', { signal: controller.signal });
		clearTimeout(tid);
		return new Response(JSON.stringify({ fetched: true, status: r.status }), {
			headers: { "Content-Type": "application/json" }
		});
	} catch (e) {
		return new Response(JSON.stringify({ fetched: false, error: e.message }), {
			headers: { "Content-Type": "application/json" }
		});
	}
}`

		resp := tc.NewRequest("POST", "/api/v1/functions").
			WithAuth(adminToken).
			WithJSON(map[string]interface{}{
				"name":    functionName,
				"code":    code,
				"runtime": "deno",
				"enabled": true,
			}).
			Send()
		require.Equal(t, fiber.StatusCreated, resp.Status(), "Failed to create function: %s", string(resp.Body()))

		var fn map[string]interface{}
		err := json.Unmarshal(resp.Body(), &fn)
		require.NoError(t, err)

		allowedDomains, _ := fn["allowed_domains"].(string)
		assert.Equal(t, "example.com, httpbin.org", allowedDomains, "allowed_domains should be parsed from annotation")

		result := invokeFunction(t, tc, adminToken, functionName)
		fetched, _ := result["fetched"].(bool)
		assert.True(t, fetched, "Function with annotation-parsed allowed_domains should fetch allowed domain")
	})
}
