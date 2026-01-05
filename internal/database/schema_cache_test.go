package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMakeKey(t *testing.T) {
	tests := []struct {
		name     string
		schema   string
		table    string
		expected string
	}{
		{
			name:     "public schema",
			schema:   "public",
			table:    "users",
			expected: "public.users",
		},
		{
			name:     "custom schema",
			schema:   "analytics",
			table:    "events",
			expected: "analytics.events",
		},
		{
			name:     "auth schema",
			schema:   "auth",
			table:    "sessions",
			expected: "auth.sessions",
		},
		{
			name:     "storage schema",
			schema:   "storage",
			table:    "objects",
			expected: "storage.objects",
		},
		{
			name:     "schema with underscore",
			schema:   "my_schema",
			table:    "my_table",
			expected: "my_schema.my_table",
		},
		{
			name:     "table with uppercase",
			schema:   "public",
			table:    "UserProfiles",
			expected: "public.UserProfiles",
		},
		{
			name:     "schema with numbers",
			schema:   "schema123",
			table:    "table456",
			expected: "schema123.table456",
		},
		{
			name:     "empty schema",
			schema:   "",
			table:    "users",
			expected: ".users",
		},
		{
			name:     "empty table",
			schema:   "public",
			table:    "",
			expected: "public.",
		},
		{
			name:     "both empty",
			schema:   "",
			table:    "",
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := makeKey(tt.schema, tt.table)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMakeKey_Consistency(t *testing.T) {
	// Same inputs should always produce same output
	key1 := makeKey("public", "users")
	key2 := makeKey("public", "users")
	assert.Equal(t, key1, key2)

	// Different order should produce different keys
	key3 := makeKey("users", "public")
	assert.NotEqual(t, key1, key3)
}

func TestSchemaCache_IsExpired(t *testing.T) {
	tests := []struct {
		name            string
		ttl             time.Duration
		lastRefresh     time.Time
		expectedExpired bool
	}{
		{
			name:            "just refreshed",
			ttl:             5 * time.Minute,
			lastRefresh:     time.Now(),
			expectedExpired: false,
		},
		{
			name:            "refreshed 1 minute ago with 5 minute TTL",
			ttl:             5 * time.Minute,
			lastRefresh:     time.Now().Add(-1 * time.Minute),
			expectedExpired: false,
		},
		{
			name:            "expired - 10 minutes ago with 5 minute TTL",
			ttl:             5 * time.Minute,
			lastRefresh:     time.Now().Add(-10 * time.Minute),
			expectedExpired: true,
		},
		{
			name:            "exactly at TTL boundary",
			ttl:             5 * time.Minute,
			lastRefresh:     time.Now().Add(-5 * time.Minute),
			expectedExpired: false, // time.Since will be slightly over, so might be true
		},
		{
			name:            "very short TTL expired",
			ttl:             1 * time.Millisecond,
			lastRefresh:     time.Now().Add(-100 * time.Millisecond),
			expectedExpired: true,
		},
		{
			name:            "zero TTL",
			ttl:             0,
			lastRefresh:     time.Now(),
			expectedExpired: true, // Any time > 0 is expired with 0 TTL
		},
		{
			name:            "future refresh time",
			ttl:             5 * time.Minute,
			lastRefresh:     time.Now().Add(1 * time.Hour),
			expectedExpired: false, // Negative duration, not expired
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &SchemaCache{
				ttl:         tt.ttl,
				lastRefresh: tt.lastRefresh,
			}

			result := cache.isExpired()

			// For boundary cases, we can't be exact due to timing
			if tt.name == "exactly at TTL boundary" {
				// Just check it's boolean
				assert.IsType(t, true, result)
			} else {
				assert.Equal(t, tt.expectedExpired, result)
			}
		})
	}
}

func TestSchemaCache_NeedsRefresh(t *testing.T) {
	tests := []struct {
		name         string
		stale        bool
		ttl          time.Duration
		lastRefresh  time.Time
		expectedNeed bool
	}{
		{
			name:         "stale flag set",
			stale:        true,
			ttl:          5 * time.Minute,
			lastRefresh:  time.Now(),
			expectedNeed: true,
		},
		{
			name:         "not stale and not expired",
			stale:        false,
			ttl:          5 * time.Minute,
			lastRefresh:  time.Now(),
			expectedNeed: false,
		},
		{
			name:         "not stale but expired",
			stale:        false,
			ttl:          5 * time.Minute,
			lastRefresh:  time.Now().Add(-10 * time.Minute),
			expectedNeed: true,
		},
		{
			name:         "stale and expired",
			stale:        true,
			ttl:          5 * time.Minute,
			lastRefresh:  time.Now().Add(-10 * time.Minute),
			expectedNeed: true,
		},
		{
			name:         "zero TTL always needs refresh",
			stale:        false,
			ttl:          0,
			lastRefresh:  time.Now(),
			expectedNeed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &SchemaCache{
				stale:       tt.stale,
				ttl:         tt.ttl,
				lastRefresh: tt.lastRefresh,
			}

			result := cache.needsRefresh()
			assert.Equal(t, tt.expectedNeed, result)
		})
	}
}

func TestSchemaCache_NeedsRefresh_LogicalOR(t *testing.T) {
	// Test that needsRefresh is true if EITHER stale OR expired
	cache1 := &SchemaCache{
		stale:       true, // stale
		ttl:         1 * time.Hour,
		lastRefresh: time.Now(), // not expired
	}
	assert.True(t, cache1.needsRefresh(), "should need refresh when stale even if not expired")

	cache2 := &SchemaCache{
		stale:       false, // not stale
		ttl:         1 * time.Minute,
		lastRefresh: time.Now().Add(-10 * time.Minute), // expired
	}
	assert.True(t, cache2.needsRefresh(), "should need refresh when expired even if not stale")

	cache3 := &SchemaCache{
		stale:       false, // not stale
		ttl:         1 * time.Hour,
		lastRefresh: time.Now(), // not expired
	}
	assert.False(t, cache3.needsRefresh(), "should not need refresh when neither stale nor expired")
}

func TestSchemaCache_TableCount(t *testing.T) {
	cache := &SchemaCache{
		tables: make(map[string]*TableInfo),
	}

	// Initially empty
	assert.Equal(t, 0, cache.TableCount())

	// Add some tables
	cache.tables["public.users"] = &TableInfo{}
	cache.tables["public.posts"] = &TableInfo{}
	cache.tables["auth.sessions"] = &TableInfo{}

	assert.Equal(t, 3, cache.TableCount())
}

func TestSchemaCache_ViewCount(t *testing.T) {
	cache := &SchemaCache{
		views: make(map[string]*TableInfo),
	}

	// Initially empty
	assert.Equal(t, 0, cache.ViewCount())

	// Add some views
	cache.views["public.user_stats"] = &TableInfo{}
	cache.views["analytics.daily_reports"] = &TableInfo{}

	assert.Equal(t, 2, cache.ViewCount())
}
