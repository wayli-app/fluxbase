package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
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
	envKey := c.GetEnvVarName(key)

	// Check if environment variable override exists
	// Parse directly from env var instead of viper to avoid viper initialization issues
	if envVal := os.Getenv(envKey); envVal != "" {
		envVal = strings.ToLower(envVal)
		return envVal == "true" || envVal == "1" || envVal == "yes"
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
	envKey := c.GetEnvVarName(key)

	// Check if environment variable override exists
	// Parse directly from env var instead of viper
	if envVal := os.Getenv(envKey); envVal != "" {
		if intVal, err := strconv.Atoi(envVal); err == nil {
			return intVal
		}
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

// GetString retrieves a string setting with caching
// Priority: Environment variables > Cache > Database > Default value
func (c *SettingsCache) GetString(ctx context.Context, key string, defaultValue string) string {
	envKey := c.GetEnvVarName(key)

	// Check if environment variable override exists
	if envVal := os.Getenv(envKey); envVal != "" {
		return envVal
	}

	// Check cache
	c.mu.RLock()
	if entry, exists := c.cache[key]; exists && time.Now().Before(entry.expiration) {
		c.mu.RUnlock()
		if val, ok := entry.value.(string); ok {
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

	// Extract string value from the setting
	var strValue string
	if val, ok := setting.Value["value"].(string); ok {
		strValue = val
	} else {
		strValue = defaultValue
	}

	// Store in cache
	c.mu.Lock()
	c.cache[key] = cacheEntry{
		value:      strValue,
		expiration: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return strValue
}

// GetJSON retrieves a JSON setting and unmarshals it into the target
// Priority: Environment variables > Cache > Database > Error
func (c *SettingsCache) GetJSON(ctx context.Context, key string, target interface{}) error {
	envKey := c.GetEnvVarName(key)

	// Check if environment variable override exists
	if envVal := os.Getenv(envKey); envVal != "" {
		return json.Unmarshal([]byte(envVal), target)
	}

	// Check cache
	c.mu.RLock()
	if entry, exists := c.cache[key]; exists && time.Now().Before(entry.expiration) {
		c.mu.RUnlock()
		// Cache stores the raw value, marshal and unmarshal to target
		if jsonBytes, ok := entry.value.([]byte); ok {
			return json.Unmarshal(jsonBytes, target)
		}
	}
	c.mu.RUnlock()

	// Cache miss or expired - fetch from database
	setting, err := c.service.GetSetting(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get setting: %w", err)
	}

	// Marshal the value to JSON bytes
	jsonBytes, err := json.Marshal(setting.Value["value"])
	if err != nil {
		return fmt.Errorf("failed to marshal setting value: %w", err)
	}

	// Store in cache
	c.mu.Lock()
	c.cache[key] = cacheEntry{
		value:      jsonBytes,
		expiration: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	// Unmarshal into target
	return json.Unmarshal(jsonBytes, target)
}

// GetMany retrieves multiple settings at once
// Returns a map of key -> value (the actual setting value, not the full setting object)
// Missing or unauthorized settings are omitted from the result (no error)
func (c *SettingsCache) GetMany(ctx context.Context, keys []string) (map[string]interface{}, error) {
	result := make(map[string]interface{}, len(keys))

	if len(keys) == 0 {
		return result, nil
	}

	// Use batch query to fetch all settings at once
	settings, err := c.service.GetSettings(ctx, keys)
	if err != nil {
		return nil, err
	}

	// Extract values from settings
	for key, setting := range settings {
		if val, ok := setting.Value["value"]; ok {
			result[key] = val
		}
	}

	return result, nil
}

// toViperKey converts app.* key format to viper config format
// e.g., "app.auth.enable_signup" -> "auth.enable_signup"
// e.g., "app.realtime.enabled" -> "realtime.enabled"
func (c *SettingsCache) toViperKey(key string) string {
	if len(key) > 4 && key[:4] == "app." {
		return key[4:] // Remove "app." prefix
	}
	return key
}

// IsOverriddenByEnv checks if a setting is overridden by an environment variable
func (c *SettingsCache) IsOverriddenByEnv(key string) bool {
	envKey := c.GetEnvVarName(key)
	return os.Getenv(envKey) != ""
}

// GetEnvVarName returns the environment variable name for a given setting key
// e.g., "app.auth.signup_enabled" -> "FLUXBASE_AUTH_SIGNUP_ENABLED"
// e.g., "app.realtime.enabled" -> "FLUXBASE_REALTIME_ENABLED"
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
