//nolint:errcheck // Test code - error handling not critical
package realtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionManager_CreateSubscription(t *testing.T) {
	sm := NewSubscriptionManager(nil)

	sub, err := sm.CreateSubscription(
		"sub1",
		"conn1",
		"user1",
		"authenticated",
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
	sm := NewSubscriptionManager(nil)

	// Create a subscription
	_, err := sm.CreateSubscription(
		"sub1",
		"conn1",
		"user1",
		"authenticated",
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
	sm := NewSubscriptionManager(nil)

	err := sm.RemoveSubscription("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subscription not found")
}

func TestSubscriptionManager_RemoveConnectionSubscriptions(t *testing.T) {
	sm := NewSubscriptionManager(nil)

	// Create multiple subscriptions for the same connection
	sm.CreateSubscription("sub1", "conn1", "user1", "authenticated", "public", "users", "INSERT", "")
	sm.CreateSubscription("sub2", "conn1", "user1", "authenticated", "public", "posts", "UPDATE", "")
	sm.CreateSubscription("sub3", "conn2", "user2", "authenticated", "public", "comments", "DELETE", "")

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
	sm := NewSubscriptionManager(nil)

	// Create subscriptions for different connections
	sm.CreateSubscription("sub1", "conn1", "user1", "authenticated", "public", "users", "*", "")
	sm.CreateSubscription("sub2", "conn1", "user1", "authenticated", "public", "posts", "*", "")
	sm.CreateSubscription("sub3", "conn2", "user2", "authenticated", "public", "comments", "*", "")

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
	sm := NewSubscriptionManager(nil)

	// Create subscriptions for different users and tables
	sm.CreateSubscription("sub1", "conn1", "user1", "authenticated", "public", "users", "*", "")
	sm.CreateSubscription("sub2", "conn2", "user2", "authenticated", "public", "users", "*", "")
	sm.CreateSubscription("sub3", "conn3", "user1", "authenticated", "public", "posts", "*", "")

	stats := sm.GetStats()
	assert.Equal(t, 3, stats["total_subscriptions"])
	assert.Equal(t, 2, stats["users_with_subs"])
	assert.Equal(t, 2, stats["tables_with_subs"])
}

func TestSubscriptionManager_DefaultEventToWildcard(t *testing.T) {
	sm := NewSubscriptionManager(nil)

	sub, err := sm.CreateSubscription(
		"sub1",
		"conn1",
		"user1",
		"authenticated",
		"public",
		"users",
		"", // Empty event should default to "*"
		"",
	)

	require.NoError(t, err)
	assert.Equal(t, "*", sub.Event)
}

func TestSubscriptionManager_WithFilter(t *testing.T) {
	sm := NewSubscriptionManager(nil)

	sub, err := sm.CreateSubscription(
		"sub1",
		"conn1",
		"user1",
		"authenticated",
		"public",
		"users",
		"UPDATE",
		"status=eq.active",
	)

	require.NoError(t, err)
	assert.NotNil(t, sub.Filter)
}

func TestSubscriptionManager_InvalidFilter(t *testing.T) {
	sm := NewSubscriptionManager(nil)

	_, err := sm.CreateSubscription(
		"sub1",
		"conn1",
		"user1",
		"authenticated",
		"public",
		"users",
		"UPDATE",
		"invalid_filter_format",
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid filter")
}

func TestSubscriptionManager_CleanupOnRemove(t *testing.T) {
	sm := NewSubscriptionManager(nil)

	// Create subscription
	sm.CreateSubscription("sub1", "conn1", "user1", "authenticated", "public", "users", "*", "")

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
	sm := NewSubscriptionManager(nil)

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
	sm := NewSubscriptionManager(nil)

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
	sm := NewSubscriptionManager(nil)

	// Initial stats
	stats := sm.GetStats()
	assert.Equal(t, 0, stats["total_subscriptions"])
	assert.Equal(t, 0, stats["users_with_subs"])
	assert.Equal(t, 0, stats["tables_with_subs"])

	// Add subscriptions
	sm.CreateSubscription("sub1", "conn1", "user1", "authenticated", "public", "users", "*", "")
	sm.CreateSubscription("sub2", "conn2", "user2", "authenticated", "public", "posts", "*", "")

	stats = sm.GetStats()
	assert.Equal(t, 2, stats["total_subscriptions"])
	assert.Equal(t, 2, stats["users_with_subs"])
	assert.Equal(t, 2, stats["tables_with_subs"])
}
