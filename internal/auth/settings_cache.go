package auth

import (
	"context"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// SettingsCache provides a simple in-memory cache for settings with TTL
// It supports environment variable overrides that take precedence over database values
type SettingsCache struct {
	mu      sync.RWMutex
	cache   map[string]cacheEntry
	ttl     time.Duration
	service *SystemSettingsService
}

type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

// NewSettingsCache creates a new settings cache
func NewSettingsCache(service *SystemSettingsService, ttl time.Duration) *SettingsCache {
	return &SettingsCache{
		cache:   make(map[string]cacheEntry),
		ttl:     ttl,
		service: service,
	}
}

// GetBool retrieves a boolean setting with caching
// Priority: Environment variables > Cache > Database > Default value
func (c *SettingsCache) GetBool(ctx context.Context, key string, defaultValue bool) bool {
	// Convert app.* key format to viper config format (e.g., app.auth.enable_signup -> auth.enable_signup)
	viperKey := c.toViperKey(key)

	// Check if environment variable override exists
	if viper.IsSet(viperKey) {
		return viper.GetBool(viperKey)
	}

	// Check cache
	c.mu.RLock()
	if entry, exists := c.cache[key]; exists && time.Now().Before(entry.expiration) {
		c.mu.RUnlock()
		if val, ok := entry.value.(bool); ok {
			return val
		}
		return defaultValue
	}
	c.mu.RUnlock()

	// Cache miss or expired - fetch from database
	setting, err := c.service.GetSetting(ctx, key)
	if err != nil {
		return defaultValue
	}

	// Extract boolean value from the setting
	var boolValue bool
	if val, ok := setting.Value["value"].(bool); ok {
		boolValue = val
	} else {
		boolValue = defaultValue
	}

	// Store in cache
	c.mu.Lock()
	c.cache[key] = cacheEntry{
		value:      boolValue,
		expiration: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return boolValue
}

// GetInt retrieves an integer setting with caching
// Priority: Environment variables > Cache > Database > Default value
func (c *SettingsCache) GetInt(ctx context.Context, key string, defaultValue int) int {
	// Convert app.* key format to viper config format
	viperKey := c.toViperKey(key)

	// Check if environment variable override exists
	if viper.IsSet(viperKey) {
		return viper.GetInt(viperKey)
	}

	// Check cache
	c.mu.RLock()
	if entry, exists := c.cache[key]; exists && time.Now().Before(entry.expiration) {
		c.mu.RUnlock()
		if val, ok := entry.value.(int); ok {
			return val
		}
		return defaultValue
	}
	c.mu.RUnlock()

	// Cache miss or expired - fetch from database
	setting, err := c.service.GetSetting(ctx, key)
	if err != nil {
		return defaultValue
	}

	// Extract integer value from the setting
	var intValue int
	switch v := setting.Value["value"].(type) {
	case int:
		intValue = v
	case float64:
		intValue = int(v)
	default:
		intValue = defaultValue
	}

	// Store in cache
	c.mu.Lock()
	c.cache[key] = cacheEntry{
		value:      intValue,
		expiration: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return intValue
}

// toViperKey converts app.* key format to viper config format
// e.g., "app.auth.enable_signup" -> "auth.enable_signup"
// e.g., "app.features.enable_realtime" -> "features.enable_realtime"
func (c *SettingsCache) toViperKey(key string) string {
	if len(key) > 4 && key[:4] == "app." {
		return key[4:] // Remove "app." prefix
	}
	return key
}

// IsOverriddenByEnv checks if a setting is overridden by an environment variable
func (c *SettingsCache) IsOverriddenByEnv(key string) bool {
	viperKey := c.toViperKey(key)
	return viper.IsSet(viperKey)
}

// GetEnvVarName returns the environment variable name for a given setting key
// e.g., "app.auth.enable_signup" -> "FLUXBASE_AUTH_ENABLE_SIGNUP"
func (c *SettingsCache) GetEnvVarName(key string) string {
	viperKey := c.toViperKey(key)
	// Convert to uppercase and replace dots with underscores
	envVar := "FLUXBASE_"
	for _, char := range viperKey {
		if char == '.' {
			envVar += "_"
		} else if char >= 'a' && char <= 'z' {
			envVar += string(char - 32) // Convert to uppercase
		} else if char >= 'A' && char <= 'Z' {
			envVar += string(char)
		} else if char >= '0' && char <= '9' {
			envVar += string(char)
		} else {
			envVar += "_"
		}
	}
	return envVar
}

// Invalidate removes a key from the cache
func (c *SettingsCache) Invalidate(key string) {
	c.mu.Lock()
	delete(c.cache, key)
	c.mu.Unlock()
}

// InvalidateAll clears the entire cache
func (c *SettingsCache) InvalidateAll() {
	c.mu.Lock()
	c.cache = make(map[string]cacheEntry)
	c.mu.Unlock()
}
