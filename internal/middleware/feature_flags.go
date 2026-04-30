package middleware

import (
	"github.com/gofiber/fiber/v3"

	"github.com/nimbleflux/fluxbase/internal/auth"
)

// RequireFeatureEnabled returns a middleware that checks if a feature flag is enabled
// If the feature is disabled, it returns HTTP 503 Service Unavailable
// Feature flags can be controlled via database settings or environment variables
func RequireFeatureEnabled(settingsCache *auth.SettingsCache, featureKey string) fiber.Handler {
	return func(c fiber.Ctx) error {
		// If settings cache is nil, treat the feature as disabled
		if settingsCache == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":       "Feature not available",
				"code":        "FEATURE_DISABLED",
				"feature_key": featureKey,
			})
		}

		ctx := c.RequestCtx()
		isEnabled := settingsCache.GetBool(ctx, featureKey, true)

		if !isEnabled {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":       "Feature is not enabled",
				"code":        "FEATURE_DISABLED",
				"feature_key": featureKey,
			})
		}

		return c.Next()
	}
}

// RequireRealtimeEnabled returns a middleware that ensures realtime feature is enabled
func RequireRealtimeEnabled(settingsCache *auth.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.realtime.enabled")
}

// RequireStorageEnabled returns a middleware that ensures storage feature is enabled
func RequireStorageEnabled(settingsCache *auth.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.storage.enabled")
}

// RequireFunctionsEnabled returns a middleware that ensures edge functions feature is enabled
func RequireFunctionsEnabled(settingsCache *auth.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.functions.enabled")
}

// RequireJobsEnabled returns a middleware that ensures jobs feature is enabled
func RequireJobsEnabled(settingsCache *auth.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.jobs.enabled")
}

// RequireAIEnabled returns a middleware that ensures AI chatbot feature is enabled
func RequireAIEnabled(settingsCache *auth.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.ai.enabled")
}

// RequireRPCEnabled returns a middleware that ensures RPC feature is enabled
func RequireRPCEnabled(settingsCache *auth.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.rpc.enabled")
}

// fiber:context-methods migrated
