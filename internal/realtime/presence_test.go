package realtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPresenceManager(t *testing.T) {
	pm := NewPresenceManager()
	require.NotNil(t, pm)
	assert.NotNil(t, pm.presences)
	assert.NotNil(t, pm.connPresences)
}

func TestPresenceManager_Track(t *testing.T) {
	t.Run("tracks new presence and returns isNew=true", func(t *testing.T) {
		pm := NewPresenceManager()

		userID := "user-123"
		state := PresenceState{"status": "online"}

		info, isNew := pm.Track("room:1", "user:123", state, &userID, "conn-1")

		require.NotNil(t, info)
		assert.True(t, isNew)
		assert.Equal(t, "user:123", info.Key)
		assert.Equal(t, state, info.State)
		assert.Equal(t, &userID, info.UserID)
		assert.Equal(t, "conn-1", info.ConnID)
		assert.NotZero(t, info.JoinedAt)
	})

	t.Run("updates existing presence and returns isNew=false", func(t *testing.T) {
		pm := NewPresenceManager()

		userID := "user-123"
		state1 := PresenceState{"status": "online"}
		state2 := PresenceState{"status": "busy"}

		info1, isNew1 := pm.Track("room:1", "user:123", state1, &userID, "conn-1")
		info2, isNew2 := pm.Track("room:1", "user:123", state2, &userID, "conn-1")

		assert.True(t, isNew1)
		assert.False(t, isNew2)
		assert.Equal(t, state2, info2.State)
		// Join time should be preserved
		assert.Equal(t, info1.JoinedAt, info2.JoinedAt)
	})

	t.Run("tracks presence with nil userID", func(t *testing.T) {
		pm := NewPresenceManager()

		state := PresenceState{"name": "anonymous"}

		info, isNew := pm.Track("room:1", "anon:1", state, nil, "conn-1")

		require.NotNil(t, info)
		assert.True(t, isNew)
		assert.Nil(t, info.UserID)
	})

	t.Run("tracks multiple presences in same channel", func(t *testing.T) {
		pm := NewPresenceManager()

		pm.Track("room:1", "user:1", PresenceState{}, nil, "conn-1")
		pm.Track("room:1", "user:2", PresenceState{}, nil, "conn-2")
		pm.Track("room:1", "user:3", PresenceState{}, nil, "conn-3")

		count := pm.GetChannelPresenceCount("room:1")
		assert.Equal(t, 3, count)
	})

	t.Run("tracks presences across multiple channels", func(t *testing.T) {
		pm := NewPresenceManager()

		pm.Track("room:1", "user:1", PresenceState{}, nil, "conn-1")
		pm.Track("room:2", "user:1", PresenceState{}, nil, "conn-1")

		assert.Equal(t, 1, pm.GetChannelPresenceCount("room:1"))
		assert.Equal(t, 1, pm.GetChannelPresenceCount("room:2"))
	})
}

func TestPresenceManager_Untrack(t *testing.T) {
	t.Run("removes presence and returns info", func(t *testing.T) {
		pm := NewPresenceManager()

		userID := "user-123"
		state := PresenceState{"status": "online"}
		pm.Track("room:1", "user:123", state, &userID, "conn-1")

		info := pm.Untrack("room:1", "user:123", "conn-1")

		require.NotNil(t, info)
		assert.Equal(t, "user:123", info.Key)
		assert.Equal(t, state, info.State)

		// Channel should now be empty
		assert.Equal(t, 0, pm.GetChannelPresenceCount("room:1"))
	})

	t.Run("returns nil for non-existent presence", func(t *testing.T) {
		pm := NewPresenceManager()

		info := pm.Untrack("room:1", "user:123", "conn-1")
		assert.Nil(t, info)
	})

	t.Run("cleans up empty channel map", func(t *testing.T) {
		pm := NewPresenceManager()

		pm.Track("room:1", "user:1", PresenceState{}, nil, "conn-1")
		pm.Untrack("room:1", "user:1", "conn-1")

		// The channel should be removed from the map
		pm.mu.RLock()
		_, exists := pm.presences["room:1"]
		pm.mu.RUnlock()
		assert.False(t, exists)
	})
}

func TestPresenceManager_GetPresenceState(t *testing.T) {
	t.Run("returns empty map for empty channel", func(t *testing.T) {
		pm := NewPresenceManager()

		state := pm.GetPresenceState("room:1")
		assert.NotNil(t, state)
		assert.Empty(t, state)
	})

	t.Run("returns presence state for channel", func(t *testing.T) {
		pm := NewPresenceManager()

		pm.Track("room:1", "user:1", PresenceState{"status": "online"}, nil, "conn-1")
		pm.Track("room:1", "user:2", PresenceState{"status": "away"}, nil, "conn-2")

		state := pm.GetPresenceState("room:1")

		assert.Len(t, state, 2)
		assert.Len(t, state["user:1"], 1)
		assert.Equal(t, "online", state["user:1"][0]["status"])
		assert.Len(t, state["user:2"], 1)
		assert.Equal(t, "away", state["user:2"][0]["status"])
	})
}

func TestPresenceManager_CleanupConnection(t *testing.T) {
	t.Run("removes all presences for connection", func(t *testing.T) {
		pm := NewPresenceManager()

		// Track presence in multiple channels for the same connection
		pm.Track("room:1", "user:1", PresenceState{}, nil, "conn-1")
		pm.Track("room:2", "user:1", PresenceState{}, nil, "conn-1")
		pm.Track("room:3", "user:1", PresenceState{}, nil, "conn-1")

		// Another connection in room:1
		pm.Track("room:1", "user:2", PresenceState{}, nil, "conn-2")

		removed := pm.CleanupConnection("conn-1")

		assert.Len(t, removed, 3)
		assert.NotNil(t, removed["room:1"])
		assert.NotNil(t, removed["room:2"])
		assert.NotNil(t, removed["room:3"])

		// room:1 should still have user:2
		assert.Equal(t, 1, pm.GetChannelPresenceCount("room:1"))
		// room:2 and room:3 should be empty
		assert.Equal(t, 0, pm.GetChannelPresenceCount("room:2"))
		assert.Equal(t, 0, pm.GetChannelPresenceCount("room:3"))
	})

	t.Run("returns empty map for unknown connection", func(t *testing.T) {
		pm := NewPresenceManager()

		removed := pm.CleanupConnection("unknown-conn")
		assert.Empty(t, removed)
	})

	t.Run("cleans up connection tracking", func(t *testing.T) {
		pm := NewPresenceManager()

		pm.Track("room:1", "user:1", PresenceState{}, nil, "conn-1")
		pm.CleanupConnection("conn-1")

		pm.mu.RLock()
		_, exists := pm.connPresences["conn-1"]
		pm.mu.RUnlock()
		assert.False(t, exists)
	})
}

func TestPresenceManager_GetChannelPresenceCount(t *testing.T) {
	t.Run("returns 0 for empty channel", func(t *testing.T) {
		pm := NewPresenceManager()

		count := pm.GetChannelPresenceCount("room:1")
		assert.Equal(t, 0, count)
	})

	t.Run("returns correct count", func(t *testing.T) {
		pm := NewPresenceManager()

		pm.Track("room:1", "user:1", PresenceState{}, nil, "conn-1")
		pm.Track("room:1", "user:2", PresenceState{}, nil, "conn-2")

		count := pm.GetChannelPresenceCount("room:1")
		assert.Equal(t, 2, count)
	})

	t.Run("updates count after untrack", func(t *testing.T) {
		pm := NewPresenceManager()

		pm.Track("room:1", "user:1", PresenceState{}, nil, "conn-1")
		pm.Track("room:1", "user:2", PresenceState{}, nil, "conn-2")
		pm.Untrack("room:1", "user:1", "conn-1")

		count := pm.GetChannelPresenceCount("room:1")
		assert.Equal(t, 1, count)
	})
}

func TestPresenceInfo_Struct(t *testing.T) {
	userID := "user-123"
	info := PresenceInfo{
		Key:    "user:123",
		State:  PresenceState{"status": "online", "typing": true},
		UserID: &userID,
		ConnID: "conn-1",
	}

	assert.Equal(t, "user:123", info.Key)
	assert.Equal(t, "online", info.State["status"])
	assert.Equal(t, true, info.State["typing"])
	assert.Equal(t, "user-123", *info.UserID)
	assert.Equal(t, "conn-1", info.ConnID)
}
