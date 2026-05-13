package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type SettingEntry struct {
	Value map[string]interface{}
}

type SettingProvider interface {
	GetSetting(ctx context.Context, key string) (*SettingEntry, error)
	GetSettings(ctx context.Context, keys []string) (map[string]*SettingEntry, error)
}

type SettingsCache struct {
	mu      sync.RWMutex
	cache   map[string]cacheEntry
	ttl     time.Duration
	service SettingProvider
}

type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

func NewSettingsCache(service SettingProvider, ttl time.Duration) *SettingsCache {
	return &SettingsCache{
		cache:   make(map[string]cacheEntry),
		ttl:     ttl,
		service: service,
	}
}

func (c *SettingsCache) SetCachedValue(key string, value interface{}) {
	c.mu.Lock()
	c.cache[key] = cacheEntry{
		value:      value,
		expiration: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

func (c *SettingsCache) GetBool(ctx context.Context, key string, defaultValue bool) bool {
	envKey := c.GetEnvVarName(key)

	if envVal := os.Getenv(envKey); envVal != "" {
		envVal = strings.ToLower(envVal)
		return envVal == "true" || envVal == "1" || envVal == "yes"
	}

	c.mu.RLock()
	if entry, exists := c.cache[key]; exists && time.Now().Before(entry.expiration) {
		c.mu.RUnlock()
		if val, ok := entry.value.(bool); ok {
			return val
		}
		return defaultValue
	}
	c.mu.RUnlock()

	if c.service != nil {
		setting, err := c.service.GetSetting(ctx, key)
		if err == nil {
			var boolValue bool
			if val, ok := setting.Value["value"].(bool); ok {
				boolValue = val
			} else {
				boolValue = defaultValue
			}

			c.mu.Lock()
			c.cache[key] = cacheEntry{
				value:      boolValue,
				expiration: time.Now().Add(c.ttl),
			}
			c.mu.Unlock()

			return boolValue
		}
	}

	if c.isFeatureFlagKey(key) {
		viperKey := c.toViperKey(key)
		if viper.IsSet(viperKey) {
			return viper.GetBool(viperKey)
		}
	}

	return defaultValue
}

func (c *SettingsCache) GetInt(ctx context.Context, key string, defaultValue int) int {
	envKey := c.GetEnvVarName(key)

	if envVal := os.Getenv(envKey); envVal != "" {
		if intVal, err := strconv.Atoi(envVal); err == nil {
			return intVal
		}
	}

	c.mu.RLock()
	if entry, exists := c.cache[key]; exists && time.Now().Before(entry.expiration) {
		c.mu.RUnlock()
		if val, ok := entry.value.(int); ok {
			return val
		}
		return defaultValue
	}
	c.mu.RUnlock()

	if c.service != nil {
		setting, err := c.service.GetSetting(ctx, key)
		if err == nil {
			var intValue int
			switch v := setting.Value["value"].(type) {
			case int:
				intValue = v
			case float64:
				intValue = int(v)
			default:
				intValue = defaultValue
			}

			c.mu.Lock()
			c.cache[key] = cacheEntry{
				value:      intValue,
				expiration: time.Now().Add(c.ttl),
			}
			c.mu.Unlock()

			return intValue
		}
	}

	if c.isFeatureFlagKey(key) {
		viperKey := c.toViperKey(key)
		if viper.IsSet(viperKey) {
			return viper.GetInt(viperKey)
		}
	}

	return defaultValue
}

func (c *SettingsCache) GetString(ctx context.Context, key string, defaultValue string) string {
	envKey := c.GetEnvVarName(key)

	if envVal := os.Getenv(envKey); envVal != "" {
		return envVal
	}

	c.mu.RLock()
	if entry, exists := c.cache[key]; exists && time.Now().Before(entry.expiration) {
		c.mu.RUnlock()
		if val, ok := entry.value.(string); ok {
			return val
		}
		return defaultValue
	}
	c.mu.RUnlock()

	if c.service != nil {
		setting, err := c.service.GetSetting(ctx, key)
		if err == nil {
			var strValue string
			if val, ok := setting.Value["value"].(string); ok {
				strValue = val
			} else {
				strValue = defaultValue
			}

			c.mu.Lock()
			c.cache[key] = cacheEntry{
				value:      strValue,
				expiration: time.Now().Add(c.ttl),
			}
			c.mu.Unlock()

			return strValue
		}
	}

	if c.isFeatureFlagKey(key) {
		viperKey := c.toViperKey(key)
		if viper.IsSet(viperKey) {
			return viper.GetString(viperKey)
		}
	}

	return defaultValue
}

func (c *SettingsCache) GetDuration(ctx context.Context, key string, defaultValue time.Duration) time.Duration {
	strValue := c.GetString(ctx, key, "")

	if strValue == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(strValue)
	if err != nil {
		log.Warn().
			Err(err).
			Str("key", key).
			Str("value", strValue).
			Dur("default", defaultValue).
			Msg("Failed to parse duration setting, using default")
		return defaultValue
	}

	return duration
}

func (c *SettingsCache) GetJSON(ctx context.Context, key string, target interface{}) error {
	envKey := c.GetEnvVarName(key)

	if envVal := os.Getenv(envKey); envVal != "" {
		return json.Unmarshal([]byte(envVal), target)
	}

	c.mu.RLock()
	if entry, exists := c.cache[key]; exists && time.Now().Before(entry.expiration) {
		c.mu.RUnlock()
		if jsonBytes, ok := entry.value.([]byte); ok {
			return json.Unmarshal(jsonBytes, target)
		}
	}
	c.mu.RUnlock()

	if c.service == nil {
		return fmt.Errorf("failed to get setting: service not available")
	}

	setting, err := c.service.GetSetting(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get setting: %w", err)
	}

	jsonBytes, err := json.Marshal(setting.Value["value"])
	if err != nil {
		return fmt.Errorf("failed to marshal setting value: %w", err)
	}

	c.mu.Lock()
	c.cache[key] = cacheEntry{
		value:      jsonBytes,
		expiration: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return json.Unmarshal(jsonBytes, target)
}

func (c *SettingsCache) GetMany(ctx context.Context, keys []string) (map[string]interface{}, error) {
	result := make(map[string]interface{}, len(keys))

	if len(keys) == 0 {
		return result, nil
	}

	if c.service == nil {
		return result, nil
	}

	settings, err := c.service.GetSettings(ctx, keys)
	if err != nil {
		return nil, err
	}

	for key, setting := range settings {
		if val, ok := setting.Value["value"]; ok {
			result[key] = val
		}
	}

	return result, nil
}

func (c *SettingsCache) isFeatureFlagKey(key string) bool {
	return strings.HasSuffix(key, ".enabled")
}

func (c *SettingsCache) toViperKey(key string) string {
	if len(key) > 4 && key[:4] == "app." {
		return key[4:]
	}
	return key
}

func (c *SettingsCache) IsOverriddenByEnv(key string) bool {
	envKey := c.GetEnvVarName(key)
	return os.Getenv(envKey) != ""
}

func (c *SettingsCache) GetEnvVarName(key string) string {
	viperKey := c.toViperKey(key)

	envVar := "FLUXBASE_"
	for _, char := range viperKey {
		switch {
		case char == '.':
			envVar += "_"
		case char >= 'a' && char <= 'z':
			envVar += string(char - 32)
		case char >= 'A' && char <= 'Z':
			envVar += string(char)
		case char >= '0' && char <= '9':
			envVar += string(char)
		default:
			envVar += "_"
		}
	}
	return envVar
}

func (c *SettingsCache) Invalidate(key string) {
	c.mu.Lock()
	delete(c.cache, key)
	c.mu.Unlock()
}

func (c *SettingsCache) InvalidateAll() {
	c.mu.Lock()
	c.cache = make(map[string]cacheEntry)
	c.mu.Unlock()
}
