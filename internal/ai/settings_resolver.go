package ai

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/settings"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// SettingsResolver handles settings template resolution for chatbots.
// It resolves {{key}}, {{user:key}}, and {{system:key}} placeholders in system prompts.
type SettingsResolver struct {
	secretsService *settings.SecretsService
	cache          *settingsCache
	cacheTTL       time.Duration
}

// settingsCache provides per-user caching of resolved settings
type settingsCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry // key: userID or "system"
}

type cacheEntry struct {
	settings  map[string]string
	expiresAt time.Time
}

// templatePattern matches {{key}}, {{user:key}}, and {{system:key}} placeholders
// Captures the optional prefix (user: or system:) and the key name
var templatePattern = regexp.MustCompile(`\{\{((?:user:|system:)?[a-zA-Z][a-zA-Z0-9_.]*)\}\}`)

// reservedKeys are built-in template variables that should not be resolved as settings
var reservedKeys = map[string]bool{
	"user_id": true,
}

// NewSettingsResolver creates a new settings resolver
func NewSettingsResolver(secretsService *settings.SecretsService, cacheTTL time.Duration) *SettingsResolver {
	return &SettingsResolver{
		secretsService: secretsService,
		cache: &settingsCache{
			entries: make(map[string]*cacheEntry),
		},
		cacheTTL: cacheTTL,
	}
}

// ResolveTemplate replaces {{setting.key}} placeholders in text with actual values.
// Supports three resolution modes:
//   - {{key}} - default: user → system fallback
//   - {{user:key}} - user-only: no fallback, empty if user setting doesn't exist
//   - {{system:key}} - system-only: ignores any user overrides
//
// Reserved keys like {{user_id}} are left untouched for other processors.
func (r *SettingsResolver) ResolveTemplate(ctx context.Context, text string, userID *uuid.UUID) (string, error) {
	if r.secretsService == nil {
		return text, nil
	}

	// Find all matches
	matches := templatePattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text, nil
	}

	// Pre-load settings if we have matches
	keys := r.ExtractSettingKeys(text)
	if len(keys) == 0 {
		return text, nil
	}

	// Load settings (with caching)
	userSettings, systemSettings, err := r.loadSettings(ctx, userID)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load settings for template resolution")
		// Continue with empty maps - templates will resolve to empty strings
		if userSettings == nil {
			userSettings = make(map[string]string)
		}
		if systemSettings == nil {
			systemSettings = make(map[string]string)
		}
	}

	// Replace all matches
	result := templatePattern.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the key (without {{ }})
		inner := match[2 : len(match)-2]

		// Check for reserved keys
		if reservedKeys[inner] {
			return match // Leave unchanged
		}

		// Parse prefix and key
		prefix, key := parseTemplateKey(inner)

		// Resolve based on prefix
		switch prefix {
		case "user":
			// User-only: no fallback
			if val, ok := userSettings[key]; ok {
				return val
			}
			return ""

		case "system":
			// System-only: ignore user settings
			if val, ok := systemSettings[key]; ok {
				return val
			}
			return ""

		default:
			// Default: user → system fallback
			if val, ok := userSettings[key]; ok {
				return val
			}
			if val, ok := systemSettings[key]; ok {
				return val
			}
			return ""
		}
	})

	return result, nil
}

// ExtractSettingKeys extracts all setting keys from text (excluding reserved keys).
// Returns keys without prefixes.
func (r *SettingsResolver) ExtractSettingKeys(text string) []string {
	matches := templatePattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var keys []string

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		inner := match[1]

		// Skip reserved keys
		if reservedKeys[inner] {
			continue
		}

		// Extract key without prefix
		_, key := parseTemplateKey(inner)

		if !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}

	return keys
}

// parseTemplateKey parses a template key into prefix and key name.
// Examples:
//   - "user:api_key" -> ("user", "api_key")
//   - "system:base_url" -> ("system", "base_url")
//   - "pelias.endpoint" -> ("", "pelias.endpoint")
func parseTemplateKey(s string) (prefix, key string) {
	if strings.HasPrefix(s, "user:") {
		return "user", s[5:]
	}
	if strings.HasPrefix(s, "system:") {
		return "system", s[7:]
	}
	return "", s
}

// loadSettings loads user and system settings with caching.
func (r *SettingsResolver) loadSettings(ctx context.Context, userID *uuid.UUID) (userSettings, systemSettings map[string]string, err error) {
	now := time.Now()

	// Try to get system settings from cache
	r.cache.mu.RLock()
	if entry, ok := r.cache.entries["system"]; ok && entry.expiresAt.After(now) {
		systemSettings = entry.settings
	}

	// Try to get user settings from cache
	var userCacheKey string
	if userID != nil {
		userCacheKey = userID.String()
		if entry, ok := r.cache.entries[userCacheKey]; ok && entry.expiresAt.After(now) {
			userSettings = entry.settings
		}
	}
	r.cache.mu.RUnlock()

	// Load system settings if not cached
	if systemSettings == nil {
		systemSettings, err = r.secretsService.GetAllSystemSettings(ctx)
		if err != nil {
			return nil, nil, err
		}

		// Cache system settings
		r.cache.mu.Lock()
		r.cache.entries["system"] = &cacheEntry{
			settings:  systemSettings,
			expiresAt: now.Add(r.cacheTTL),
		}
		r.cache.mu.Unlock()
	}

	// Load user settings if not cached and userID provided
	if userSettings == nil && userID != nil {
		userSettings, err = r.secretsService.GetAllUserSettings(ctx, *userID)
		if err != nil {
			return nil, systemSettings, err
		}

		// Cache user settings
		r.cache.mu.Lock()
		r.cache.entries[userCacheKey] = &cacheEntry{
			settings:  userSettings,
			expiresAt: now.Add(r.cacheTTL),
		}
		r.cache.mu.Unlock()
	}

	if userSettings == nil {
		userSettings = make(map[string]string)
	}

	return userSettings, systemSettings, nil
}

// InvalidateCache clears all cached settings.
func (r *SettingsResolver) InvalidateCache() {
	r.cache.mu.Lock()
	r.cache.entries = make(map[string]*cacheEntry)
	r.cache.mu.Unlock()
}

// InvalidateUserCache clears cached settings for a specific user.
func (r *SettingsResolver) InvalidateUserCache(userID uuid.UUID) {
	r.cache.mu.Lock()
	delete(r.cache.entries, userID.String())
	r.cache.mu.Unlock()
}
