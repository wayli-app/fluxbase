package realtime

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// PresenceState represents the state data for a presence
type PresenceState map[string]interface{}

// PresenceInfo tracks presence information for a user/key
type PresenceInfo struct {
	Key      string
	State    PresenceState
	UserID   *string
	ConnID   string
	JoinedAt time.Time
}

// PresenceManager manages presence tracking across channels
type PresenceManager struct {
	// channel -> (presence_key -> PresenceInfo)
	presences map[string]map[string]*PresenceInfo
	// connection_id -> (channel -> presence_key)
	connPresences map[string]map[string]string
	mu            sync.RWMutex
}

// NewPresenceManager creates a new presence manager
func NewPresenceManager() *PresenceManager {
	return &PresenceManager{
		presences:     make(map[string]map[string]*PresenceInfo),
		connPresences: make(map[string]map[string]string),
	}
}

// Track adds or updates presence for a given channel and key
func (pm *PresenceManager) Track(channel, key string, state PresenceState, userID *string, connID string) (*PresenceInfo, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Initialize channel map if needed
	if pm.presences[channel] == nil {
		pm.presences[channel] = make(map[string]*PresenceInfo)
	}

	// Check if this is a new presence (join) or update
	_, exists := pm.presences[channel][key]

	// Create or update presence info
	info := &PresenceInfo{
		Key:      key,
		State:    state,
		UserID:   userID,
		ConnID:   connID,
		JoinedAt: time.Now(),
	}

	// If updating existing presence, preserve join time
	if exists {
		if existing := pm.presences[channel][key]; existing != nil {
			info.JoinedAt = existing.JoinedAt
		}
	}

	pm.presences[channel][key] = info

	// Track connection -> channel -> key mapping
	if pm.connPresences[connID] == nil {
		pm.connPresences[connID] = make(map[string]string)
	}
	pm.connPresences[connID][channel] = key

	log.Debug().
		Str("channel", channel).
		Str("key", key).
		Str("connID", connID).
		Bool("isNew", !exists).
		Msg("Presence tracked")

	return info, !exists // Return whether this was a new presence
}

// Untrack removes presence for a given channel and key
func (pm *PresenceManager) Untrack(channel, key, connID string) *PresenceInfo {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Get presence info before removing
	var info *PresenceInfo
	if pm.presences[channel] != nil {
		info = pm.presences[channel][key]
		delete(pm.presences[channel], key)

		// Clean up empty channel map
		if len(pm.presences[channel]) == 0 {
			delete(pm.presences, channel)
		}
	}

	// Remove from connection tracking
	if pm.connPresences[connID] != nil {
		delete(pm.connPresences[connID], channel)

		// Clean up empty connection map
		if len(pm.connPresences[connID]) == 0 {
			delete(pm.connPresences, connID)
		}
	}

	log.Debug().
		Str("channel", channel).
		Str("key", key).
		Str("connID", connID).
		Msg("Presence untracked")

	return info
}

// GetPresenceState returns the current presence state for a channel
// Returns in the format expected by the SDK: map[key][]PresenceState
func (pm *PresenceManager) GetPresenceState(channel string) map[string][]PresenceState {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string][]PresenceState)

	if channelPresences := pm.presences[channel]; channelPresences != nil {
		for key, info := range channelPresences {
			result[key] = []PresenceState{info.State}
		}
	}

	return result
}

// CleanupConnection removes all presence entries for a disconnected connection
// Returns a map of channel -> presence info that was removed
func (pm *PresenceManager) CleanupConnection(connID string) map[string]*PresenceInfo {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	removed := make(map[string]*PresenceInfo)

	// Get all channels this connection has presence in
	channels := pm.connPresences[connID]
	if channels == nil {
		return removed
	}

	// Remove presence from each channel
	for channel, key := range channels {
		if pm.presences[channel] != nil {
			if info := pm.presences[channel][key]; info != nil {
				removed[channel] = info
				delete(pm.presences[channel], key)

				// Clean up empty channel map
				if len(pm.presences[channel]) == 0 {
					delete(pm.presences, channel)
				}
			}
		}
	}

	// Remove connection tracking
	delete(pm.connPresences, connID)

	log.Debug().
		Str("connID", connID).
		Int("removedCount", len(removed)).
		Msg("Presence cleaned up for disconnected connection")

	return removed
}

// GetChannelPresenceCount returns the number of active presences in a channel
func (pm *PresenceManager) GetChannelPresenceCount(channel string) int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.presences[channel] != nil {
		return len(pm.presences[channel])
	}
	return 0
}
