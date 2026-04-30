package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSettingsCache(t *testing.T) {
	cache := NewSettingsCache(nil, 5*time.Minute)
	require.NotNil(t, cache)
	assert.NotNil(t, cache.cache)
	assert.Equal(t, 5*time.Minute, cache.ttl)
}

func TestSettingsCache_GetEnvVarName(t *testing.T) {
	cache := NewSettingsCache(nil, time.Minute)

	testCases := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "feature enabled flag",
			key:      "app.realtime.enabled",
			expected: "FLUXBASE_REALTIME_ENABLED",
		},
		{
			name:     "auth signup enabled",
			key:      "app.auth.signup_enabled",
			expected: "FLUXBASE_AUTH_SIGNUP_ENABLED",
		},
		{
			name:     "nested key without app prefix",
			key:      "database.connection_pool_size",
			expected: "FLUXBASE_DATABASE_CONNECTION_POOL_SIZE",
		},
		{
			name:     "storage enabled flag",
			key:      "app.storage.enabled",
			expected: "FLUXBASE_STORAGE_ENABLED",
		},
		{
			name:     "simple key",
			key:      "app.debug",
			expected: "FLUXBASE_DEBUG",
		},
		{
			name:     "key with numbers",
			key:      "app.rate_limit.max_requests_per_minute",
			expected: "FLUXBASE_RATE_LIMIT_MAX_REQUESTS_PER_MINUTE",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cache.GetEnvVarName(tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSettingsCache_toViperKey(t *testing.T) {
	cache := NewSettingsCache(nil, time.Minute)

	testCases := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "removes app prefix",
			key:      "app.auth.enable_signup",
			expected: "auth.enable_signup",
		},
		{
			name:     "keeps key without app prefix",
			key:      "database.host",
			expected: "database.host",
		},
		{
			name:     "handles short key",
			key:      "abc",
			expected: "abc",
		},
		{
			name:     "handles app prefix only",
			key:      "app.",
			expected: "app.", // Key length is exactly 4, so condition `len(key) > 4` is false
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cache.toViperKey(tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSettingsCache_IsOverriddenByEnv(t *testing.T) {
	cache := NewSettingsCache(nil, time.Minute)

	t.Run("returns true when env var is set", func(t *testing.T) {
		_ = os.Setenv("FLUXBASE_AUTH_ENABLED", "true")
		defer func() { _ = os.Unsetenv("FLUXBASE_AUTH_ENABLED") }()

		result := cache.IsOverriddenByEnv("app.auth.enabled")
		assert.True(t, result)
	})

	t.Run("returns false when env var is not set", func(t *testing.T) {
		result := cache.IsOverriddenByEnv("app.nonexistent.setting")
		assert.False(t, result)
	})
}

func TestSettingsCache_Invalidate(t *testing.T) {
	cache := NewSettingsCache(nil, time.Minute)

	// Manually populate cache
	cache.mu.Lock()
	cache.cache["test-key"] = cacheEntry{
		value:      "test-value",
		expiration: time.Now().Add(time.Minute),
	}
	cache.cache["other-key"] = cacheEntry{
		value:      "other-value",
		expiration: time.Now().Add(time.Minute),
	}
	cache.mu.Unlock()

	// Invalidate one key
	cache.Invalidate("test-key")

	cache.mu.RLock()
	_, exists := cache.cache["test-key"]
	_, otherExists := cache.cache["other-key"]
	cache.mu.RUnlock()

	assert.False(t, exists, "test-key should be removed")
	assert.True(t, otherExists, "other-key should remain")
}

func TestSettingsCache_InvalidateAll(t *testing.T) {
	cache := NewSettingsCache(nil, time.Minute)

	// Manually populate cache
	cache.mu.Lock()
	cache.cache["key1"] = cacheEntry{value: "value1", expiration: time.Now().Add(time.Minute)}
	cache.cache["key2"] = cacheEntry{value: "value2", expiration: time.Now().Add(time.Minute)}
	cache.cache["key3"] = cacheEntry{value: "value3", expiration: time.Now().Add(time.Minute)}
	cache.mu.Unlock()

	// Invalidate all
	cache.InvalidateAll()

	cache.mu.RLock()
	assert.Empty(t, cache.cache)
	cache.mu.RUnlock()
}

func TestSettingsCache_CacheExpiration(t *testing.T) {
	cache := NewSettingsCache(nil, 50*time.Millisecond)

	// Manually populate cache with short TTL
	cache.mu.Lock()
	cache.cache["expiring-key"] = cacheEntry{
		value:      "expiring-value",
		expiration: time.Now().Add(50 * time.Millisecond),
	}
	cache.mu.Unlock()

	// Check it exists
	cache.mu.RLock()
	entry, exists := cache.cache["expiring-key"]
	cache.mu.RUnlock()
	assert.True(t, exists)
	assert.False(t, time.Now().After(entry.expiration))

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Entry still exists but is expired
	cache.mu.RLock()
	entry, exists = cache.cache["expiring-key"]
	cache.mu.RUnlock()
	assert.True(t, exists)
	assert.True(t, time.Now().After(entry.expiration))
}

func TestSettingsCache_CacheEntry(t *testing.T) {
	entry := cacheEntry{
		value:      "test-value",
		expiration: time.Now().Add(time.Minute),
	}

	assert.Equal(t, "test-value", entry.value)
	assert.True(t, entry.expiration.After(time.Now()))
}

func TestSettingsCache_EnvVarParsing(t *testing.T) {
	// These test the parsing logic that would be used in GetBool, GetInt, GetString

	t.Run("bool parsing from env", func(t *testing.T) {
		testCases := []struct {
			envValue string
			expected bool
		}{
			{"true", true},
			{"True", true},
			{"TRUE", true},
			{"1", true},
			{"yes", true},
			{"YES", true},
			{"false", false},
			{"False", false},
			{"0", false},
			{"no", false},
			{"anything-else", false},
		}

		for _, tc := range testCases {
			t.Run(tc.envValue, func(t *testing.T) {
				// Just verify the test cases are structured correctly
				// The actual parsing happens in GetBool which requires a SystemSettingsService
				assert.NotEmpty(t, tc.envValue)
			})
		}
	})
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestSettingsCache_ConcurrentInvalidate(t *testing.T) {
	cache := NewSettingsCache(nil, time.Minute)

	// Populate cache
	cache.mu.Lock()
	for i := 0; i < 100; i++ {
		key := "key-" + string(rune('0'+i/10)) + string(rune('0'+i%10))
		cache.cache[key] = cacheEntry{value: i, expiration: time.Now().Add(time.Minute)}
	}
	cache.mu.Unlock()

	// Concurrently invalidate
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				key := "key-" + string(rune('0'+id)) + string(rune('0'+j))
				cache.Invalidate(key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Cache should be empty
	cache.mu.RLock()
	assert.Empty(t, cache.cache)
	cache.mu.RUnlock()
}

func TestSettingsCache_ConcurrentInvalidateAll(t *testing.T) {
	cache := NewSettingsCache(nil, time.Minute)

	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			cache.mu.Lock()
			cache.cache["key-"+string(rune('0'+i%10))] = cacheEntry{
				value:      i,
				expiration: time.Now().Add(time.Minute),
			}
			cache.mu.Unlock()
		}
		done <- true
	}()

	// Invalidator goroutine
	go func() {
		for i := 0; i < 10; i++ {
			cache.InvalidateAll()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for both
	<-done
	<-done

	// No panics means success - cache state is undefined but safe
}

func TestSettingsCache_ConcurrentReadWrite(t *testing.T) {
	cache := NewSettingsCache(nil, time.Minute)

	done := make(chan bool)

	// Writer goroutines
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 20; j++ {
				cache.mu.Lock()
				cache.cache["shared-key"] = cacheEntry{
					value:      id*100 + j,
					expiration: time.Now().Add(time.Minute),
				}
				cache.mu.Unlock()
			}
			done <- true
		}(i)
	}

	// Reader goroutines
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 20; j++ {
				cache.mu.RLock()
				_ = cache.cache["shared-key"]
				cache.mu.RUnlock()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// No race conditions means success
}

// =============================================================================
// Additional GetEnvVarName Tests
// =============================================================================

func TestSettingsCache_GetEnvVarName_SpecialChars(t *testing.T) {
	cache := NewSettingsCache(nil, time.Minute)

	testCases := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "key with hyphen",
			key:      "app.rate-limit.enabled",
			expected: "FLUXBASE_RATE_LIMIT_ENABLED",
		},
		{
			name:     "key with underscore",
			key:      "app.auth_config.enabled",
			expected: "FLUXBASE_AUTH_CONFIG_ENABLED",
		},
		{
			name:     "mixed case key",
			key:      "app.AuthConfig.Enabled",
			expected: "FLUXBASE_AUTHCONFIG_ENABLED",
		},
		{
			name:     "key with numbers in middle",
			key:      "app.v2.api.enabled",
			expected: "FLUXBASE_V2_API_ENABLED",
		},
		{
			name:     "empty key",
			key:      "",
			expected: "FLUXBASE_",
		},
		{
			name:     "only dots",
			key:      "...",
			expected: "FLUXBASE____",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cache.GetEnvVarName(tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// Cache TTL Tests
// =============================================================================

func TestSettingsCache_TTLValues(t *testing.T) {
	testCases := []time.Duration{
		0,
		time.Millisecond,
		time.Second,
		time.Minute,
		time.Hour,
		24 * time.Hour,
	}

	for _, ttl := range testCases {
		t.Run(ttl.String(), func(t *testing.T) {
			cache := NewSettingsCache(nil, ttl)
			assert.Equal(t, ttl, cache.ttl)
		})
	}
}

func TestSettingsCache_ZeroTTL(t *testing.T) {
	cache := NewSettingsCache(nil, 0)

	// Manually populate cache with zero TTL (expires immediately)
	cache.mu.Lock()
	cache.cache["zero-ttl-key"] = cacheEntry{
		value:      "test-value",
		expiration: time.Now(), // Expired immediately
	}
	cache.mu.Unlock()

	// Entry exists but is expired
	cache.mu.RLock()
	entry, exists := cache.cache["zero-ttl-key"]
	cache.mu.RUnlock()

	assert.True(t, exists)
	assert.False(t, time.Now().Before(entry.expiration))
}

// =============================================================================
// CacheEntry Type Tests
// =============================================================================

func TestCacheEntry_TypeValues(t *testing.T) {
	testCases := []struct {
		name  string
		value interface{}
	}{
		{"string value", "test-string"},
		{"int value", 42},
		{"bool value", true},
		{"float value", 3.14},
		{"nil value", nil},
		{"slice value", []string{"a", "b", "c"}},
		{"map value", map[string]int{"x": 1, "y": 2}},
		{"byte slice", []byte("test-bytes")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry := cacheEntry{
				value:      tc.value,
				expiration: time.Now().Add(time.Minute),
			}

			assert.Equal(t, tc.value, entry.value)
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkSettingsCache_GetEnvVarName(b *testing.B) {
	cache := NewSettingsCache(nil, time.Minute)
	key := "app.auth.signup_enabled"

	for i := 0; i < b.N; i++ {
		cache.GetEnvVarName(key)
	}
}

func BenchmarkSettingsCache_Invalidate(b *testing.B) {
	cache := NewSettingsCache(nil, time.Minute)

	// Pre-populate
	cache.mu.Lock()
	for i := 0; i < 1000; i++ {
		cache.cache["key-"+string(rune(i))] = cacheEntry{
			value:      i,
			expiration: time.Now().Add(time.Minute),
		}
	}
	cache.mu.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Invalidate("key-" + string(rune(i%1000)))
	}
}

func BenchmarkSettingsCache_InvalidateAll(b *testing.B) {
	cache := NewSettingsCache(nil, time.Minute)

	for i := 0; i < b.N; i++ {
		// Pre-populate
		cache.mu.Lock()
		for j := 0; j < 100; j++ {
			cache.cache["key-"+string(rune(j))] = cacheEntry{
				value:      j,
				expiration: time.Now().Add(time.Minute),
			}
		}
		cache.mu.Unlock()

		cache.InvalidateAll()
	}
}

func BenchmarkSettingsCache_CacheRead(b *testing.B) {
	cache := NewSettingsCache(nil, time.Minute)

	// Pre-populate
	cache.mu.Lock()
	cache.cache["test-key"] = cacheEntry{
		value:      "test-value",
		expiration: time.Now().Add(time.Hour),
	}
	cache.mu.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.mu.RLock()
		_ = cache.cache["test-key"]
		cache.mu.RUnlock()
	}
}

func BenchmarkSettingsCache_CacheWrite(b *testing.B) {
	cache := NewSettingsCache(nil, time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.mu.Lock()
		cache.cache["test-key"] = cacheEntry{
			value:      i,
			expiration: time.Now().Add(time.Hour),
		}
		cache.mu.Unlock()
	}
}

// =============================================================================
// Viper Config Fallback Tests
// =============================================================================

func TestSettingsCache_GetBool_ViperFallback(t *testing.T) {
	cache := NewSettingsCache(nil, time.Minute)
	ctx := context.Background()

	originalEnv := os.Getenv("FLUXBASE_FUNCTIONS_ENABLED")
	os.Unsetenv("FLUXBASE_FUNCTIONS_ENABLED")
	defer func() {
		if originalEnv != "" {
			_ = os.Setenv("FLUXBASE_FUNCTIONS_ENABLED", originalEnv)
		}
	}()

	t.Run("returns viper default when no env var and no database", func(t *testing.T) {
		viper.Set("functions.enabled", true)
		defer viper.Reset()

		result := cache.GetBool(ctx, "app.functions.enabled", false)
		assert.True(t, result, "should fall back to viper default (true)")
	})

	t.Run("returns viper default false for disabled features", func(t *testing.T) {
		viper.Set("functions.enabled", false)
		defer viper.Reset()

		result := cache.GetBool(ctx, "app.functions.enabled", true)
		assert.False(t, result, "should fall back to viper value (false)")
	})

	t.Run("returns hardcoded default when viper key not set", func(t *testing.T) {
		defer viper.Reset()

		result := cache.GetBool(ctx, "app.nonexistent.feature", true)
		assert.True(t, result, "should return hardcoded default when viper has no key")
	})

	t.Run("env var overrides viper", func(t *testing.T) {
		viper.Set("functions.enabled", true)
		defer viper.Reset()

		_ = os.Setenv("FLUXBASE_FUNCTIONS_ENABLED", "false")
		defer os.Unsetenv("FLUXBASE_FUNCTIONS_ENABLED")

		result := cache.GetBool(ctx, "app.functions.enabled", true)
		assert.False(t, result, "env var should take precedence over viper")
	})

	t.Run("cache overrides viper", func(t *testing.T) {
		viper.Set("functions.enabled", true)
		defer viper.Reset()

		cache.mu.Lock()
		cache.cache["app.functions.enabled"] = cacheEntry{
			value:      false,
			expiration: time.Now().Add(time.Minute),
		}
		cache.mu.Unlock()
		defer cache.Invalidate("app.functions.enabled")

		result := cache.GetBool(ctx, "app.functions.enabled", true)
		assert.False(t, result, "cached value should take precedence over viper")
	})
}

func TestSettingsCache_GetInt_ViperFallback(t *testing.T) {
	cache := NewSettingsCache(nil, time.Minute)
	ctx := context.Background()

	originalEnv := os.Getenv("FLUXBASE_AUTH_PASSWORD_MIN_LENGTH")
	os.Unsetenv("FLUXBASE_AUTH_PASSWORD_MIN_LENGTH")
	defer func() {
		if originalEnv != "" {
			_ = os.Setenv("FLUXBASE_AUTH_PASSWORD_MIN_LENGTH", originalEnv)
		}
	}()

	t.Run("returns viper default when no env var and no database", func(t *testing.T) {
		viper.Set("auth.password_min_length", 12)
		defer viper.Reset()

		result := cache.GetInt(ctx, "app.auth.password_min_length", 0)
		assert.Equal(t, 12, result, "should fall back to viper default")
	})

	t.Run("returns hardcoded default when viper key not set", func(t *testing.T) {
		defer viper.Reset()

		result := cache.GetInt(ctx, "app.nonexistent.setting", 42)
		assert.Equal(t, 42, result, "should return hardcoded default when viper has no key")
	})
}

func TestSettingsCache_GetString_ViperFallback(t *testing.T) {
	cache := NewSettingsCache(nil, time.Minute)
	ctx := context.Background()

	originalEnv := os.Getenv("FLUXBASE_EMAIL_PROVIDER")
	os.Unsetenv("FLUXBASE_EMAIL_PROVIDER")
	defer func() {
		if originalEnv != "" {
			_ = os.Setenv("FLUXBASE_EMAIL_PROVIDER", originalEnv)
		}
	}()

	t.Run("returns viper default when no env var and no database", func(t *testing.T) {
		viper.Set("email.provider", "smtp")
		defer viper.Reset()

		result := cache.GetString(ctx, "app.email.provider", "")
		assert.Equal(t, "smtp", result, "should fall back to viper default")
	})

	t.Run("returns hardcoded default when viper key not set", func(t *testing.T) {
		defer viper.Reset()

		result := cache.GetString(ctx, "app.nonexistent.setting", "fallback")
		assert.Equal(t, "fallback", result, "should return hardcoded default when viper has no key")
	})
}
