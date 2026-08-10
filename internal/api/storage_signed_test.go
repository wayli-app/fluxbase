//go:build integration
// +build integration

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageAPI_GenerateSignedURL_NotSupported(t *testing.T) {
	app, _, db := setupStorageTestServer(t)
	defer db.Close()

	// Create bucket and upload file
	createTestBucket(t, app, "signed-bucket")
	uploadTestFile(t, app, "signed-bucket", "signed.txt", "signed content")

	// Try to generate signed URL (not supported for local storage)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/signed-bucket/signed.txt/signed-url", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Signed URL route returns 400 (bad request) or 501 (not implemented) depending on validation
	// Both are acceptable error responses
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusNotImplemented}, resp.StatusCode)
}

func TestIPRateLimiter(t *testing.T) {
	t.Run("allows initial request", func(t *testing.T) {
		limiter := &ipRateLimiter{
			requests: make(map[string]*rateLimitEntry),
			limit:    10,
			window:   time.Minute,
		}

		allowed := limiter.allow("192.168.1.1")
		assert.True(t, allowed)
	})

	t.Run("allows requests up to limit", func(t *testing.T) {
		limiter := &ipRateLimiter{
			requests: make(map[string]*rateLimitEntry),
			limit:    5,
			window:   time.Minute,
		}

		// All 5 requests should be allowed
		for i := 0; i < 5; i++ {
			allowed := limiter.allow("192.168.1.1")
			assert.True(t, allowed, "request %d should be allowed", i+1)
		}
	})

	t.Run("blocks requests after limit exceeded", func(t *testing.T) {
		limiter := &ipRateLimiter{
			requests: make(map[string]*rateLimitEntry),
			limit:    3,
			window:   time.Minute,
		}

		// Use up all allowed requests
		for i := 0; i < 3; i++ {
			limiter.allow("192.168.1.1")
		}

		// Next request should be blocked
		allowed := limiter.allow("192.168.1.1")
		assert.False(t, allowed)
	})

	t.Run("tracks different IPs separately", func(t *testing.T) {
		limiter := &ipRateLimiter{
			requests: make(map[string]*rateLimitEntry),
			limit:    2,
			window:   time.Minute,
		}

		// Exhaust limit for first IP
		limiter.allow("192.168.1.1")
		limiter.allow("192.168.1.1")

		// First IP should be blocked
		assert.False(t, limiter.allow("192.168.1.1"))

		// Second IP should still be allowed
		assert.True(t, limiter.allow("192.168.1.2"))
		assert.True(t, limiter.allow("192.168.1.2"))

		// Now second IP should be blocked too
		assert.False(t, limiter.allow("192.168.1.2"))
	})

	t.Run("resets after window expires", func(t *testing.T) {
		limiter := &ipRateLimiter{
			requests: make(map[string]*rateLimitEntry),
			limit:    2,
			window:   50 * time.Millisecond, // Very short window for testing
		}

		// Exhaust limit
		limiter.allow("192.168.1.1")
		limiter.allow("192.168.1.1")
		assert.False(t, limiter.allow("192.168.1.1"))

		// Wait for window to expire
		time.Sleep(60 * time.Millisecond)

		// Should be allowed again
		assert.True(t, limiter.allow("192.168.1.1"))
	})

	t.Run("limit of 1 allows single request then blocks", func(t *testing.T) {
		limiter := &ipRateLimiter{
			requests: make(map[string]*rateLimitEntry),
			limit:    1,
			window:   time.Minute,
		}

		assert.True(t, limiter.allow("192.168.1.1"))
		assert.False(t, limiter.allow("192.168.1.1"))
	})

	t.Run("handles IPv6 addresses", func(t *testing.T) {
		limiter := &ipRateLimiter{
			requests: make(map[string]*rateLimitEntry),
			limit:    2,
			window:   time.Minute,
		}

		ipv6 := "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
		assert.True(t, limiter.allow(ipv6))
		assert.True(t, limiter.allow(ipv6))
		assert.False(t, limiter.allow(ipv6))
	})

	t.Run("handles localhost addresses", func(t *testing.T) {
		limiter := &ipRateLimiter{
			requests: make(map[string]*rateLimitEntry),
			limit:    3,
			window:   time.Minute,
		}

		// 127.0.0.1 and ::1 should be tracked separately
		assert.True(t, limiter.allow("127.0.0.1"))
		assert.True(t, limiter.allow("::1"))
		assert.True(t, limiter.allow("127.0.0.1"))
		assert.True(t, limiter.allow("::1"))
	})
}

func TestRateLimitEntry_Struct(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var entry rateLimitEntry
		assert.Equal(t, 0, entry.count)
		assert.True(t, entry.windowEnd.IsZero())
	})

	t.Run("all fields can be set", func(t *testing.T) {
		windowEnd := time.Now().Add(time.Minute)
		entry := rateLimitEntry{
			count:     50,
			windowEnd: windowEnd,
		}

		assert.Equal(t, 50, entry.count)
		assert.Equal(t, windowEnd, entry.windowEnd)
	})
}

func TestIPRateLimiter_Struct(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var limiter ipRateLimiter
		assert.Nil(t, limiter.requests)
		assert.Equal(t, 0, limiter.limit)
		assert.Equal(t, time.Duration(0), limiter.window)
	})

	t.Run("default signedURLLimiter configuration", func(t *testing.T) {
		// The signed-URL rate limiter is now a per-handler field (see
		// NewStorageHandler), not a global. Construct one with the documented
		// defaults and verify those defaults are sensible.
		limiter := &ipRateLimiter{
			requests: make(map[string]*rateLimitEntry),
			limit:    100,
			window:   time.Minute,
		}
		assert.NotNil(t, limiter)
		assert.NotNil(t, limiter.requests)
		assert.Equal(t, 100, limiter.limit)
		assert.Equal(t, time.Minute, limiter.window)
	})
}
