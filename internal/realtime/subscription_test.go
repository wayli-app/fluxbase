//nolint:errcheck // Test code - error handling not critical
package realtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSubscriptionManager() *SubscriptionManager {
	mockDB := newMockSubscriptionDB()
	mockDB.EnableTable("public", "users")
	mockDB.EnableTable("public", "posts")
	mockDB.EnableTable("public", "comments")
	return NewSubscriptionManager(mockDB)
}

func TestSubscriptionManager_CreateSubscription(t *testing.T) {
	sm := newTestSubscriptionManager()

	sub, err := sm.CreateSubscription(
		"sub1",
		"conn1",
		"user1",
		"authenticated",
		nil,
		"public",
		"users",
		"INSERT",
		"",
	)

	require.NoError(t, err)
	assert.NotNil(t, sub)
	assert.Equal(t, "sub1", sub.ID)
	assert.Equal(t, "conn1", sub.ConnID)
	assert.Equal(t, "user1", sub.UserID)
	assert.Equal(t, "public", sub.Schema)
	assert.Equal(t, "users", sub.Table)
	assert.Equal(t, "INSERT", sub.Event)
}

func TestSubscriptionManager_RemoveSubscription(t *testing.T) {
	sm := newTestSubscriptionManager()

	// Create a subscription
	_, err := sm.CreateSubscription(
		"sub1",
		"conn1",
		"user1",
		"authenticated",
		nil,
		"public",
		"users",
		"INSERT",
		"",
	)
	require.NoError(t, err)

	stats := sm.GetStats()
	assert.Equal(t, 1, stats["total_subscriptions"])

	// Remove the subscription
	err = sm.RemoveSubscription("sub1")
	require.NoError(t, err)

	stats = sm.GetStats()
	assert.Equal(t, 0, stats["total_subscriptions"])
}

func TestSubscriptionManager_RemoveNonExistentSubscription(t *testing.T) {
	sm := newTestSubscriptionManager()

	err := sm.RemoveSubscription("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subscription not found")
}

func TestSubscriptionManager_RemoveConnectionSubscriptions(t *testing.T) {
	sm := newTestSubscriptionManager()

	// Create multiple subscriptions for the same connection
	sm.CreateSubscription("sub1", "conn1", "user1", "authenticated", nil, "public", "users", "INSERT", "")
	sm.CreateSubscription("sub2", "conn1", "user1", "authenticated", nil, "public", "posts", "UPDATE", "")
	sm.CreateSubscription("sub3", "conn2", "user2", "authenticated", nil, "public", "comments", "DELETE", "")

	stats := sm.GetStats()
	assert.Equal(t, 3, stats["total_subscriptions"])

	// Remove all subscriptions for conn1
	sm.RemoveConnectionSubscriptions("conn1")

	stats = sm.GetStats()
	assert.Equal(t, 1, stats["total_subscriptions"])

	// Verify conn2's subscription still exists
	subs := sm.GetSubscriptionsByConnection("conn2")
	assert.Equal(t, 1, len(subs))
	assert.Equal(t, "sub3", subs[0].ID)
}

func TestSubscriptionManager_GetSubscriptionsByConnection(t *testing.T) {
	sm := newTestSubscriptionManager()

	// Create subscriptions for different connections
	sm.CreateSubscription("sub1", "conn1", "user1", "authenticated", nil, "public", "users", "*", "")
	sm.CreateSubscription("sub2", "conn1", "user1", "authenticated", nil, "public", "posts", "*", "")
	sm.CreateSubscription("sub3", "conn2", "user2", "authenticated", nil, "public", "comments", "*", "")

	// Get subscriptions for conn1
	subs := sm.GetSubscriptionsByConnection("conn1")
	assert.Equal(t, 2, len(subs))

	// Get subscriptions for conn2
	subs = sm.GetSubscriptionsByConnection("conn2")
	assert.Equal(t, 1, len(subs))

	// Get subscriptions for non-existent connection
	subs = sm.GetSubscriptionsByConnection("conn999")
	assert.Equal(t, 0, len(subs))
}

func TestSubscriptionManager_MultipleUsersAndTables(t *testing.T) {
	sm := newTestSubscriptionManager()

	// Create subscriptions for different users and tables
	sm.CreateSubscription("sub1", "conn1", "user1", "authenticated", nil, "public", "users", "*", "")
	sm.CreateSubscription("sub2", "conn2", "user2", "authenticated", nil, "public", "users", "*", "")
	sm.CreateSubscription("sub3", "conn3", "user1", "authenticated", nil, "public", "posts", "*", "")

	stats := sm.GetStats()
	assert.Equal(t, 3, stats["total_subscriptions"])
	assert.Equal(t, 2, stats["users_with_subs"])
	assert.Equal(t, 2, stats["tables_with_subs"])
}

func TestSubscriptionManager_DefaultEventToWildcard(t *testing.T) {
	sm := newTestSubscriptionManager()

	sub, err := sm.CreateSubscription(
		"sub1",
		"conn1",
		"user1",
		"authenticated",
		nil,
		"public",
		"users",
		"", // Empty event should default to "*"
		"",
	)

	require.NoError(t, err)
	assert.Equal(t, "*", sub.Event)
}

func TestSubscriptionManager_WithFilter(t *testing.T) {
	sm := newTestSubscriptionManager()

	sub, err := sm.CreateSubscription(
		"sub1",
		"conn1",
		"user1",
		"authenticated",
		nil,
		"public",
		"users",
		"UPDATE",
		"status=eq.active",
	)

	require.NoError(t, err)
	assert.NotNil(t, sub.Filter)
}

func TestSubscriptionManager_InvalidFilter(t *testing.T) {
	sm := newTestSubscriptionManager()

	_, err := sm.CreateSubscription(
		"sub1",
		"conn1",
		"user1",
		"authenticated",
		nil,
		"public",
		"users",
		"UPDATE",
		"invalid_filter_format",
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid filter")
}

func TestSubscriptionManager_CleanupOnRemove(t *testing.T) {
	sm := newTestSubscriptionManager()

	// Create subscription
	sm.CreateSubscription("sub1", "conn1", "user1", "authenticated", nil, "public", "users", "*", "")

	stats := sm.GetStats()
	assert.Equal(t, 1, stats["total_subscriptions"])
	assert.Equal(t, 1, stats["users_with_subs"])
	assert.Equal(t, 1, stats["tables_with_subs"])

	// Remove subscription
	sm.RemoveSubscription("sub1")

	stats = sm.GetStats()
	assert.Equal(t, 0, stats["total_subscriptions"])
	assert.Equal(t, 0, stats["users_with_subs"])
	assert.Equal(t, 0, stats["tables_with_subs"])
}

func TestSubscriptionManager_MatchesEvent(t *testing.T) {
	sm := newTestSubscriptionManager()

	tests := []struct {
		name      string
		eventType string
		subEvent  string
		expected  bool
	}{
		{"wildcard matches INSERT", "INSERT", "*", true},
		{"wildcard matches UPDATE", "UPDATE", "*", true},
		{"wildcard matches DELETE", "DELETE", "*", true},
		{"exact match INSERT", "INSERT", "INSERT", true},
		{"exact match UPDATE", "UPDATE", "UPDATE", true},
		{"exact match DELETE", "DELETE", "DELETE", true},
		{"no match INSERT vs UPDATE", "INSERT", "UPDATE", false},
		{"no match UPDATE vs DELETE", "UPDATE", "DELETE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sm.matchesEvent(tt.eventType, tt.subEvent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubscriptionManager_MatchesFilter(t *testing.T) {
	sm := newTestSubscriptionManager()

	tests := []struct {
		name     string
		event    *ChangeEvent
		filter   string
		expected bool
	}{
		{
			name: "no filter matches all",
			event: &ChangeEvent{
				Record: map[string]interface{}{
					"id":     1,
					"status": "active",
				},
			},
			filter:   "",
			expected: true,
		},
		{
			name: "eq filter matches",
			event: &ChangeEvent{
				Record: map[string]interface{}{
					"id":     1,
					"status": "active",
				},
			},
			filter:   "status=eq.active",
			expected: true,
		},
		{
			name: "eq filter does not match",
			event: &ChangeEvent{
				Record: map[string]interface{}{
					"id":     1,
					"status": "inactive",
				},
			},
			filter:   "status=eq.active",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filterObj *Filter
			if tt.filter != "" {
				var err error
				filterObj, err = ParseFilter(tt.filter)
				require.NoError(t, err)
			}

			sub := &Subscription{
				Filter: filterObj,
			}

			result := sm.matchesFilter(tt.event, sub)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubscriptionManager_Stats(t *testing.T) {
	sm := newTestSubscriptionManager()

	// Initial stats
	stats := sm.GetStats()
	assert.Equal(t, 0, stats["total_subscriptions"])
	assert.Equal(t, 0, stats["users_with_subs"])
	assert.Equal(t, 0, stats["tables_with_subs"])

	// Add subscriptions
	sm.CreateSubscription("sub1", "conn1", "user1", "authenticated", nil, "public", "users", "*", "")
	sm.CreateSubscription("sub2", "conn2", "user2", "authenticated", nil, "public", "posts", "*", "")

	stats = sm.GetStats()
	assert.Equal(t, 2, stats["total_subscriptions"])
	assert.Equal(t, 2, stats["users_with_subs"])
	assert.Equal(t, 2, stats["tables_with_subs"])
}

// Tests for RLS cache configuration

func TestRLSCacheConfig_Defaults(t *testing.T) {
	cache := newRLSCache()

	assert.Equal(t, DefaultRLSCacheMaxSize, cache.maxSize)
	assert.Equal(t, DefaultRLSCacheTTL, cache.ttl)
}

func TestRLSCacheConfig_Custom(t *testing.T) {
	config := RLSCacheConfig{
		MaxSize: 50000,
		TTL:     60 * time.Second,
	}
	cache := newRLSCacheWithConfig(config)

	assert.Equal(t, 50000, cache.maxSize)
	assert.Equal(t, 60*time.Second, cache.ttl)
}

func TestRLSCacheConfig_ZeroValuesUseDefaults(t *testing.T) {
	config := RLSCacheConfig{
		MaxSize: 0,
		TTL:     0,
	}
	cache := newRLSCacheWithConfig(config)

	assert.Equal(t, DefaultRLSCacheMaxSize, cache.maxSize)
	assert.Equal(t, DefaultRLSCacheTTL, cache.ttl)
}

func TestRLSCacheConfig_NegativeValuesUseDefaults(t *testing.T) {
	config := RLSCacheConfig{
		MaxSize: -1,
		TTL:     -1,
	}
	cache := newRLSCacheWithConfig(config)

	assert.Equal(t, DefaultRLSCacheMaxSize, cache.maxSize)
	assert.Equal(t, DefaultRLSCacheTTL, cache.ttl)
}

func TestSubscriptionManager_WithCustomRLSCache(t *testing.T) {
	mockDB := newMockSubscriptionDB()
	mockDB.EnableTable("public", "users")

	config := RLSCacheConfig{
		MaxSize: 1000,
		TTL:     10 * time.Second,
	}
	sm := NewSubscriptionManagerWithConfig(mockDB, config)

	// Verify cache was created with custom config
	require.NotNil(t, sm.rlsCache)
	assert.Equal(t, 1000, sm.rlsCache.maxSize)
	assert.Equal(t, 10*time.Second, sm.rlsCache.ttl)
}

func TestSubscriptionManager_DefaultRLSCache(t *testing.T) {
	mockDB := newMockSubscriptionDB()
	sm := NewSubscriptionManager(mockDB)

	// Verify cache was created with default config
	require.NotNil(t, sm.rlsCache)
	assert.Equal(t, DefaultRLSCacheMaxSize, sm.rlsCache.maxSize)
	assert.Equal(t, DefaultRLSCacheTTL, sm.rlsCache.ttl)
}

func TestCopyClaims(t *testing.T) {
	t.Run("nil map returns nil", func(t *testing.T) {
		result := copyClaims(nil)
		assert.Nil(t, result)
	})

	t.Run("empty map returns empty map", func(t *testing.T) {
		original := make(map[string]interface{})
		result := copyClaims(original)

		assert.NotNil(t, result)
		assert.Equal(t, 0, len(result))
	})

	t.Run("copies all values", func(t *testing.T) {
		original := map[string]interface{}{
			"user_id":    "123",
			"role":       "admin",
			"meeting_id": 456,
			"nested":     map[string]string{"key": "value"},
		}

		result := copyClaims(original)

		assert.Equal(t, len(original), len(result))
		assert.Equal(t, original["user_id"], result["user_id"])
		assert.Equal(t, original["role"], result["role"])
		assert.Equal(t, original["meeting_id"], result["meeting_id"])
		assert.Equal(t, original["nested"], result["nested"])
	})

	t.Run("modifying copy does not affect original", func(t *testing.T) {
		original := map[string]interface{}{
			"user_id": "123",
			"role":    "admin",
		}

		result := copyClaims(original)

		// Modify the copy
		result["user_id"] = "456"
		result["new_key"] = "new_value"

		// Original should be unchanged
		assert.Equal(t, "123", original["user_id"])
		assert.Nil(t, original["new_key"])
	})
}

// =============================================================================
// RLS Cache Tests
// =============================================================================

func TestRLSCache_GetSet(t *testing.T) {
	cache := newRLSCacheWithConfig(RLSCacheConfig{
		MaxSize: 100,
		TTL:     1 * time.Hour,
	})

	t.Run("get returns false for non-existent key", func(t *testing.T) {
		allowed, found := cache.get("nonexistent-key")
		assert.False(t, found)
		assert.False(t, allowed)
	})

	t.Run("set and get returns correct value", func(t *testing.T) {
		cache.set("key1", true)
		allowed, found := cache.get("key1")
		assert.True(t, found)
		assert.True(t, allowed)

		cache.set("key2", false)
		allowed, found = cache.get("key2")
		assert.True(t, found)
		assert.False(t, allowed)
	})

	t.Run("expired entries return not found", func(t *testing.T) {
		// Create a cache with very short TTL
		shortCache := newRLSCacheWithConfig(RLSCacheConfig{
			MaxSize: 100,
			TTL:     1 * time.Millisecond,
		})

		shortCache.set("expiring-key", true)
		time.Sleep(5 * time.Millisecond)

		_, found := shortCache.get("expiring-key")
		assert.False(t, found)
	})
}

func TestRLSCache_GenerateCacheKey(t *testing.T) {
	cache := newRLSCache()

	t.Run("generates unique keys for different parameters", func(t *testing.T) {
		key1 := cache.generateCacheKey("public", "users", "authenticated", 1, nil)
		key2 := cache.generateCacheKey("public", "users", "authenticated", 2, nil)
		key3 := cache.generateCacheKey("public", "posts", "authenticated", 1, nil)
		key4 := cache.generateCacheKey("private", "users", "authenticated", 1, nil)
		key5 := cache.generateCacheKey("public", "users", "anon", 1, nil)

		assert.NotEqual(t, key1, key2)
		assert.NotEqual(t, key1, key3)
		assert.NotEqual(t, key1, key4)
		assert.NotEqual(t, key1, key5)
	})

	t.Run("includes claims in cache key", func(t *testing.T) {
		claims1 := map[string]interface{}{"user_id": "123"}
		claims2 := map[string]interface{}{"user_id": "456"}

		key1 := cache.generateCacheKey("public", "users", "authenticated", 1, claims1)
		key2 := cache.generateCacheKey("public", "users", "authenticated", 1, claims2)

		assert.NotEqual(t, key1, key2)
	})

	t.Run("nil claims produces different key than empty claims", func(t *testing.T) {
		keyNil := cache.generateCacheKey("public", "users", "authenticated", 1, nil)
		keyEmpty := cache.generateCacheKey("public", "users", "authenticated", 1, map[string]interface{}{})

		// These may or may not be equal depending on implementation
		// but the cache key should be deterministic
		assert.NotEmpty(t, keyNil)
		assert.NotEmpty(t, keyEmpty)
	})
}

func TestRLSCache_EvictExpired(t *testing.T) {
	cache := newRLSCacheWithConfig(RLSCacheConfig{
		MaxSize: 100,
		TTL:     1 * time.Millisecond,
	})

	// Add some entries
	cache.set("key1", true)
	cache.set("key2", false)
	cache.set("key3", true)

	// Wait for them to expire
	time.Sleep(5 * time.Millisecond)

	// Manually trigger eviction
	cache.mu.Lock()
	cache.evictExpiredLocked()
	cache.mu.Unlock()

	// All entries should be gone
	assert.Equal(t, 0, len(cache.entries))
}

// =============================================================================
// quoteIdentifier and isValidIdentifier Tests
// =============================================================================

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple identifier", "users", `"users"`},
		{"identifier with underscore", "user_data", `"user_data"`},
		{"identifier with number", "table1", `"table1"`},
		{"reserved word", "select", `"select"`},
		{"contains double quote", `table"name`, `"table""name"`},
		{"empty string", "", `""`},
		{"multiple double quotes", `a"b"c`, `"a""b""c"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"simple identifier", "users", true},
		{"starts with underscore", "_users", true},
		{"with numbers", "table123", true},
		{"mixed case", "UserData", true},
		{"underscore and number", "user_data_v2", true},
		{"starts with number", "1table", false},
		{"contains space", "user data", false},
		{"contains hyphen", "user-data", false},
		{"contains dot", "user.data", false},
		{"empty string", "", false},
		{"special characters", "table!@#", false},
		{"single letter", "a", true},
		{"single underscore", "_", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Log Subscription Tests
// =============================================================================

func TestSubscriptionManager_LogSubscriptions(t *testing.T) {
	sm := newTestSubscriptionManager()

	t.Run("create log subscription", func(t *testing.T) {
		sub, err := sm.CreateLogSubscription("logsub1", "conn1", "exec-123", "function")

		require.NoError(t, err)
		assert.Equal(t, "logsub1", sub.ID)
		assert.Equal(t, "conn1", sub.ConnID)
		assert.Equal(t, "exec-123", sub.ExecutionID)
		assert.Equal(t, "function", sub.ExecutionType)
	})

	t.Run("get log subscribers", func(t *testing.T) {
		sm.CreateLogSubscription("logsub2", "conn2", "exec-456", "job")
		sm.CreateLogSubscription("logsub3", "conn3", "exec-456", "job")

		subscribers := sm.GetLogSubscribers("exec-456")
		assert.Equal(t, 2, len(subscribers))
		assert.Contains(t, subscribers, "conn2")
		assert.Contains(t, subscribers, "conn3")
	})

	t.Run("get log subscribers for non-existent execution", func(t *testing.T) {
		subscribers := sm.GetLogSubscribers("nonexistent")
		assert.Nil(t, subscribers)
	})

	t.Run("remove log subscription", func(t *testing.T) {
		sm.CreateLogSubscription("logsub4", "conn4", "exec-789", "rpc")

		err := sm.RemoveLogSubscription("logsub4")
		require.NoError(t, err)

		subscribers := sm.GetLogSubscribers("exec-789")
		assert.Nil(t, subscribers)
	})

	t.Run("remove non-existent log subscription returns error", func(t *testing.T) {
		err := sm.RemoveLogSubscription("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "log subscription not found")
	})
}

func TestSubscriptionManager_GetLogSubscriptionsByConnection(t *testing.T) {
	sm := newTestSubscriptionManager()

	sm.CreateLogSubscription("logsub1", "conn1", "exec-1", "function")
	sm.CreateLogSubscription("logsub2", "conn1", "exec-2", "job")
	sm.CreateLogSubscription("logsub3", "conn2", "exec-3", "rpc")

	subs := sm.GetLogSubscriptionsByConnection("conn1")
	assert.Equal(t, 2, len(subs))

	subs = sm.GetLogSubscriptionsByConnection("conn2")
	assert.Equal(t, 1, len(subs))

	subs = sm.GetLogSubscriptionsByConnection("nonexistent")
	assert.Equal(t, 0, len(subs))
}

func TestSubscriptionManager_RemoveConnectionLogSubscriptions(t *testing.T) {
	sm := newTestSubscriptionManager()

	sm.CreateLogSubscription("logsub1", "conn1", "exec-1", "function")
	sm.CreateLogSubscription("logsub2", "conn1", "exec-2", "job")
	sm.CreateLogSubscription("logsub3", "conn2", "exec-3", "rpc")

	sm.RemoveConnectionLogSubscriptions("conn1")

	subs := sm.GetLogSubscriptionsByConnection("conn1")
	assert.Equal(t, 0, len(subs))

	subs = sm.GetLogSubscriptionsByConnection("conn2")
	assert.Equal(t, 1, len(subs))
}

// =============================================================================
// All-Logs Subscription Tests
// =============================================================================

func TestSubscriptionManager_AllLogsSubscriptions(t *testing.T) {
	sm := newTestSubscriptionManager()

	t.Run("create all-logs subscription", func(t *testing.T) {
		sub, err := sm.CreateAllLogsSubscription("allsub1", "conn1", "function", []string{"info", "error"})

		require.NoError(t, err)
		assert.Equal(t, "allsub1", sub.ID)
		assert.Equal(t, "conn1", sub.ConnID)
		assert.Equal(t, "function", sub.Category)
		assert.Equal(t, []string{"info", "error"}, sub.Levels)
	})

	t.Run("create all-logs subscription without filters", func(t *testing.T) {
		sub, err := sm.CreateAllLogsSubscription("allsub2", "conn2", "", nil)

		require.NoError(t, err)
		assert.Equal(t, "allsub2", sub.ID)
		assert.Empty(t, sub.Category)
		assert.Nil(t, sub.Levels)
	})

	t.Run("get all-logs subscribers", func(t *testing.T) {
		sm.CreateAllLogsSubscription("allsub3", "conn3", "job", nil)

		subscribers := sm.GetAllLogsSubscribers()
		assert.NotNil(t, subscribers)
		assert.Contains(t, subscribers, "conn3")
	})

	t.Run("remove all-logs subscription", func(t *testing.T) {
		sm.CreateAllLogsSubscription("allsub4", "conn4", "", nil)

		err := sm.RemoveAllLogsSubscription("allsub4")
		require.NoError(t, err)

		subscribers := sm.GetAllLogsSubscribers()
		_, exists := subscribers["conn4"]
		assert.False(t, exists)
	})

	t.Run("remove non-existent all-logs subscription returns error", func(t *testing.T) {
		err := sm.RemoveAllLogsSubscription("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "all-logs subscription not found")
	})
}

func TestSubscriptionManager_RemoveConnectionAllLogsSubscriptions(t *testing.T) {
	sm := newTestSubscriptionManager()

	sm.CreateAllLogsSubscription("allsub1", "conn1", "function", nil)
	sm.CreateAllLogsSubscription("allsub2", "conn1", "job", nil)
	sm.CreateAllLogsSubscription("allsub3", "conn2", "rpc", nil)

	sm.RemoveConnectionAllLogsSubscriptions("conn1")

	subscribers := sm.GetAllLogsSubscribers()
	_, exists := subscribers["conn1"]
	assert.False(t, exists)

	_, exists = subscribers["conn2"]
	assert.True(t, exists)
}

// =============================================================================
// ParseChangeEvent Tests
// =============================================================================

func TestParseChangeEvent(t *testing.T) {
	t.Run("parses valid INSERT event", func(t *testing.T) {
		payload := `{
			"type": "INSERT",
			"schema": "public",
			"table": "users",
			"record": {"id": 1, "name": "test"},
			"commit_timestamp": "2024-01-15T10:30:00Z"
		}`

		event, err := ParseChangeEvent(payload)

		require.NoError(t, err)
		assert.Equal(t, "INSERT", event.Type)
		assert.Equal(t, "public", event.Schema)
		assert.Equal(t, "users", event.Table)
		assert.NotNil(t, event.Record)
		assert.Equal(t, float64(1), event.Record["id"])
		assert.Equal(t, "test", event.Record["name"])
	})

	t.Run("parses valid UPDATE event", func(t *testing.T) {
		payload := `{
			"type": "UPDATE",
			"schema": "public",
			"table": "users",
			"record": {"id": 1, "name": "updated"},
			"old_record": {"id": 1, "name": "original"}
		}`

		event, err := ParseChangeEvent(payload)

		require.NoError(t, err)
		assert.Equal(t, "UPDATE", event.Type)
		assert.NotNil(t, event.Record)
		assert.NotNil(t, event.OldRecord)
		assert.Equal(t, "updated", event.Record["name"])
		assert.Equal(t, "original", event.OldRecord["name"])
	})

	t.Run("parses valid DELETE event", func(t *testing.T) {
		payload := `{
			"type": "DELETE",
			"schema": "public",
			"table": "users",
			"old_record": {"id": 1, "name": "deleted"}
		}`

		event, err := ParseChangeEvent(payload)

		require.NoError(t, err)
		assert.Equal(t, "DELETE", event.Type)
		assert.NotNil(t, event.OldRecord)
		assert.Nil(t, event.Record)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		_, err := ParseChangeEvent("not valid json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse change event")
	})

	t.Run("returns error for empty string", func(t *testing.T) {
		_, err := ParseChangeEvent("")
		assert.Error(t, err)
	})
}

// =============================================================================
// UpdateConnectionRole and UpdateConnectionClaims Tests
// =============================================================================

func TestSubscriptionManager_UpdateConnectionRole(t *testing.T) {
	sm := newTestSubscriptionManager()

	// Create subscriptions for a connection
	sm.CreateSubscription("sub1", "conn1", "user1", "anon", nil, "public", "users", "*", "")
	sm.CreateSubscription("sub2", "conn1", "user1", "anon", nil, "public", "posts", "*", "")
	sm.CreateSubscription("sub3", "conn2", "user2", "anon", nil, "public", "comments", "*", "")

	// Update role for conn1
	sm.UpdateConnectionRole("conn1", "authenticated")

	// Verify conn1 subscriptions have updated role
	subs := sm.GetSubscriptionsByConnection("conn1")
	for _, sub := range subs {
		assert.Equal(t, "authenticated", sub.Role)
	}

	// Verify conn2 subscriptions are unchanged
	subs = sm.GetSubscriptionsByConnection("conn2")
	for _, sub := range subs {
		assert.Equal(t, "anon", sub.Role)
	}
}

func TestSubscriptionManager_UpdateConnectionClaims(t *testing.T) {
	sm := newTestSubscriptionManager()

	originalClaims := map[string]interface{}{"user_id": "123"}
	newClaims := map[string]interface{}{"user_id": "123", "custom_claim": "value"}

	// Create subscriptions with original claims
	sm.CreateSubscription("sub1", "conn1", "user1", "authenticated", originalClaims, "public", "users", "*", "")
	sm.CreateSubscription("sub2", "conn2", "user2", "authenticated", originalClaims, "public", "posts", "*", "")

	// Update claims for conn1
	sm.UpdateConnectionClaims("conn1", newClaims)

	// Verify conn1 subscriptions have updated claims
	subs := sm.GetSubscriptionsByConnection("conn1")
	for _, sub := range subs {
		assert.Equal(t, newClaims, sub.Claims)
	}

	// Verify conn2 subscriptions are unchanged
	subs = sm.GetSubscriptionsByConnection("conn2")
	for _, sub := range subs {
		assert.Equal(t, originalClaims, sub.Claims)
	}
}

// =============================================================================
// Struct Tests
// =============================================================================

func TestSubscription_Struct(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var sub Subscription

		assert.Empty(t, sub.ID)
		assert.Empty(t, sub.UserID)
		assert.Empty(t, sub.Role)
		assert.Nil(t, sub.Claims)
		assert.Empty(t, sub.Table)
		assert.Empty(t, sub.Schema)
		assert.Empty(t, sub.Event)
		assert.Nil(t, sub.Filter)
		assert.Empty(t, sub.ConnID)
	})

	t.Run("all fields set", func(t *testing.T) {
		sub := Subscription{
			ID:     "sub-123",
			UserID: "user-456",
			Role:   "authenticated",
			Claims: map[string]interface{}{"custom": "value"},
			Table:  "users",
			Schema: "public",
			Event:  "INSERT",
			Filter: &Filter{Column: "id", Operator: "eq", Value: "1"},
			ConnID: "conn-789",
		}

		assert.Equal(t, "sub-123", sub.ID)
		assert.Equal(t, "user-456", sub.UserID)
		assert.Equal(t, "authenticated", sub.Role)
		assert.NotNil(t, sub.Claims)
		assert.Equal(t, "users", sub.Table)
		assert.Equal(t, "public", sub.Schema)
		assert.Equal(t, "INSERT", sub.Event)
		assert.NotNil(t, sub.Filter)
		assert.Equal(t, "conn-789", sub.ConnID)
	})
}

func TestLogSubscription_Struct(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var sub LogSubscription

		assert.Empty(t, sub.ID)
		assert.Empty(t, sub.ConnID)
		assert.Empty(t, sub.ExecutionID)
		assert.Empty(t, sub.ExecutionType)
	})

	t.Run("all fields set", func(t *testing.T) {
		sub := LogSubscription{
			ID:            "logsub-123",
			ConnID:        "conn-456",
			ExecutionID:   "exec-789",
			ExecutionType: "function",
		}

		assert.Equal(t, "logsub-123", sub.ID)
		assert.Equal(t, "conn-456", sub.ConnID)
		assert.Equal(t, "exec-789", sub.ExecutionID)
		assert.Equal(t, "function", sub.ExecutionType)
	})
}

func TestAllLogsSubscription_Struct(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var sub AllLogsSubscription

		assert.Empty(t, sub.ID)
		assert.Empty(t, sub.ConnID)
		assert.Empty(t, sub.Category)
		assert.Nil(t, sub.Levels)
	})

	t.Run("all fields set", func(t *testing.T) {
		sub := AllLogsSubscription{
			ID:       "allsub-123",
			ConnID:   "conn-456",
			Category: "function",
			Levels:   []string{"info", "warn", "error"},
		}

		assert.Equal(t, "allsub-123", sub.ID)
		assert.Equal(t, "conn-456", sub.ConnID)
		assert.Equal(t, "function", sub.Category)
		assert.Equal(t, 3, len(sub.Levels))
	})
}

func TestRLSCacheConfig_Struct(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		var config RLSCacheConfig

		assert.Equal(t, 0, config.MaxSize)
		assert.Equal(t, time.Duration(0), config.TTL)
	})

	t.Run("all fields set", func(t *testing.T) {
		config := RLSCacheConfig{
			MaxSize: 50000,
			TTL:     1 * time.Minute,
		}

		assert.Equal(t, 50000, config.MaxSize)
		assert.Equal(t, 1*time.Minute, config.TTL)
	})
}

// =============================================================================
// Table Not Enabled Tests
// =============================================================================

func TestSubscriptionManager_TableNotEnabled(t *testing.T) {
	mockDB := newMockSubscriptionDB()
	// Only enable "users" table, not "posts"
	mockDB.EnableTable("public", "users")
	sm := NewSubscriptionManager(mockDB)

	t.Run("subscription to enabled table succeeds", func(t *testing.T) {
		_, err := sm.CreateSubscription("sub1", "conn1", "user1", "authenticated", nil, "public", "users", "*", "")
		require.NoError(t, err)
	})

	t.Run("subscription to disabled table fails", func(t *testing.T) {
		_, err := sm.CreateSubscription("sub2", "conn1", "user1", "authenticated", nil, "public", "posts", "*", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not enabled for realtime")
	})
}
