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

func TestSSRFProtection_AWSMetadataEndpoint(t *testing.T) {
	rateLimiter, pubSub := test.NewInMemoryDependencies()
	tc := test.NewTestContextWithOptions(t, test.TestContextOptions{
		RateLimiter: rateLimiter,
		PubSub:      pubSub,
	})
	defer tc.Close()
	tc.EnsureAuthSchema()

	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-ssrf-aws-%d@test.com", timestamp)
	_, adminToken := tc.CreateDashboardAdminUser(email, "adminpass123456")

	functionName := fmt.Sprintf("test_ssrf_aws_%d", timestamp)
	createTestFunction(t, tc, adminToken, functionName, `export default async function handler(req) {
		try {
			const controller = new AbortController();
			const tid = setTimeout(() => controller.abort(), 5000);
			const resp = await fetch('http://169.254.169.254/latest/meta-data/', { signal: controller.signal });
			clearTimeout(tid);
			const text = await resp.text();
			return new Response(JSON.stringify({ fetched: true, status: resp.status, body: text }), {
				headers: { "Content-Type": "application/json" }
			});
		} catch (e) {
			return new Response(JSON.stringify({ fetched: false, error: e.message }), {
				headers: { "Content-Type": "application/json" }
			});
		}
	}`)

	resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{"test": "ssrf_aws"}).
		Send()
	require.Equal(t, fiber.StatusOK, resp.Status(), "Function should return 200: %s", string(resp.Body()))

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body(), &result)
	require.NoError(t, err)

	fetched, _ := result["fetched"].(bool)
	assert.False(t, fetched, "Function must NOT successfully fetch AWS metadata endpoint 169.254.169.254")

	errMsg, _ := result["error"].(string)
	assert.NotEmpty(t, errMsg, "Error message should be present when fetch to metadata endpoint fails")
	t.Logf("AWS metadata fetch blocked: fetched=%v error=%s", fetched, errMsg)
}

func TestSSRFProtection_GCPMetadataEndpoint(t *testing.T) {
	rateLimiter, pubSub := test.NewInMemoryDependencies()
	tc := test.NewTestContextWithOptions(t, test.TestContextOptions{
		RateLimiter: rateLimiter,
		PubSub:      pubSub,
	})
	defer tc.Close()
	tc.EnsureAuthSchema()

	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-ssrf-gcp-%d@test.com", timestamp)
	_, adminToken := tc.CreateDashboardAdminUser(email, "adminpass123456")

	functionName := fmt.Sprintf("test_ssrf_gcp_%d", timestamp)
	createTestFunction(t, tc, adminToken, functionName, `export default async function handler(req) {
		try {
			const controller = new AbortController();
			const tid = setTimeout(() => controller.abort(), 5000);
			const resp = await fetch('http://metadata.google.internal/computeMetadata/v1/', {
				headers: { 'Metadata-Flavor': 'Google' },
				signal: controller.signal
			});
			clearTimeout(tid);
			const text = await resp.text();
			return new Response(JSON.stringify({ fetched: true, status: resp.status, body: text }), {
				headers: { "Content-Type": "application/json" }
			});
		} catch (e) {
			return new Response(JSON.stringify({ fetched: false, error: e.message }), {
				headers: { "Content-Type": "application/json" }
			});
		}
	}`)

	resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{"test": "ssrf_gcp"}).
		Send()
	require.Equal(t, fiber.StatusOK, resp.Status(), "Function should return 200: %s", string(resp.Body()))

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body(), &result)
	require.NoError(t, err)

	fetched, _ := result["fetched"].(bool)
	assert.False(t, fetched, "Function must NOT successfully fetch GCP metadata endpoint metadata.google.internal")

	errMsg, _ := result["error"].(string)
	assert.NotEmpty(t, errMsg, "Error message should be present when fetch to GCP metadata fails")
	t.Logf("GCP metadata fetch blocked: fetched=%v error=%s", fetched, errMsg)
}

func TestSSRFProtection_IPv6Localhost(t *testing.T) {
	rateLimiter, pubSub := test.NewInMemoryDependencies()
	tc := test.NewTestContextWithOptions(t, test.TestContextOptions{
		RateLimiter: rateLimiter,
		PubSub:      pubSub,
	})
	defer tc.Close()
	tc.EnsureAuthSchema()

	timestamp := time.Now().UnixNano()
	email := fmt.Sprintf("admin-ssrf-ipv6-%d@test.com", timestamp)
	_, adminToken := tc.CreateDashboardAdminUser(email, "adminpass123456")

	functionName := fmt.Sprintf("test_ssrf_ipv6_%d", timestamp)
	createTestFunction(t, tc, adminToken, functionName, `export default async function handler(req) {
		try {
			const controller = new AbortController();
			const tid = setTimeout(() => controller.abort(), 5000);
			const resp = await fetch('http://[::1]/', { signal: controller.signal });
			clearTimeout(tid);
			const text = await resp.text();
			return new Response(JSON.stringify({ fetched: true, status: resp.status, body: text }), {
				headers: { "Content-Type": "application/json" }
			});
		} catch (e) {
			return new Response(JSON.stringify({ fetched: false, error: e.message }), {
				headers: { "Content-Type": "application/json" }
			});
		}
	}`)

	resp := tc.NewRequest("POST", fmt.Sprintf("/api/v1/functions/%s/invoke", functionName)).
		WithAuth(adminToken).
		WithJSON(map[string]interface{}{"test": "ssrf_ipv6"}).
		Send()
	require.Equal(t, fiber.StatusOK, resp.Status(), "Function should return 200: %s", string(resp.Body()))

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body(), &result)
	require.NoError(t, err)

	fetched, _ := result["fetched"].(bool)
	assert.False(t, fetched, "Function must NOT successfully fetch IPv6 localhost [::1]")

	errMsg, _ := result["error"].(string)
	assert.NotEmpty(t, errMsg, "Error message should be present when fetch to IPv6 localhost fails")
	t.Logf("IPv6 localhost fetch blocked: fetched=%v error=%s", fetched, errMsg)
}

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

func TestSSRFProtection_BlockedDomainsInDefaultPermissions(t *testing.T) {
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

	blockedURLs := []struct {
		name string
		url  string
	}{
		{"aws_metadata", "http://169.254.169.254/latest/meta-data/"},
		{"gcp_metadata", "http://metadata.google.internal/computeMetadata/v1/"},
		{"kubernetes_api", "http://kubernetes.default.svc/api/v1/namespaces/default/"},
	}

	functionName := fmt.Sprintf("test_ssrf_domains_%d", timestamp)
	createTestFunction(t, tc, adminToken, functionName, fmt.Sprintf(`export default async function handler(req) {
		const targets = %s;
		const results = [];
		for (const t of targets) {
			try {
				const controller = new AbortController();
				const tid = setTimeout(() => controller.abort(), 3000);
				const resp = await fetch(t.url, { signal: controller.signal });
				clearTimeout(tid);
				results.push({ name: t.name, fetched: true, status: resp.status });
			} catch (e) {
				results.push({ name: t.name, fetched: false, error: e.message });
			}
		}
		return new Response(JSON.stringify({ results: results }), {
			headers: { "Content-Type": "application/json" }
		});
	}`, func() string {
		jsonBytes, _ := json.Marshal(blockedURLs)
		return string(jsonBytes)
	}()))

	result := invokeFunction(t, tc, adminToken, functionName)

	results, ok := result["results"].([]interface{})
	require.True(t, ok, "Response should contain results array")
	assert.Len(t, results, len(blockedURLs), "Should have results for all blocked URLs")

	for _, r := range results {
		entry, ok := r.(map[string]interface{})
		require.True(t, ok, "Result entry should be a map")
		name, _ := entry["name"].(string)
		fetched, _ := entry["fetched"].(bool)
		assert.False(t, fetched, "%s endpoint should NOT be reachable", name)
		if !fetched {
			errMsg, _ := entry["error"].(string)
			assert.True(t,
				strings.Contains(errMsg, "error") ||
					strings.Contains(errMsg, "refused") ||
					strings.Contains(errMsg, "timed out") ||
					strings.Contains(errMsg, "abort") ||
					strings.Contains(errMsg, "failed") ||
					strings.Contains(errMsg, "ECONNREFUSED") ||
					strings.Contains(errMsg, "ENOTFOUND") ||
					strings.Contains(errMsg, "cancel"),
				"Error message should indicate connection failure: %s", errMsg)
		}
		t.Logf("Blocked domain %s: fetched=%v error=%v", name, fetched, entry["error"])
	}
}
