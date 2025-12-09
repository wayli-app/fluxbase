package middleware

import (
	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/gofiber/fiber/v2"
)

// RequireFeatureEnabled returns a middleware that checks if a feature flag is enabled
// If the feature is disabled, it returns HTTP 404 Not Found
// Feature flags can be controlled via database settings or environment variables
func RequireFeatureEnabled(settingsCache *auth.SettingsCache, featureKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if feature is enabled (checks env vars first, then cache, then database)
		ctx := c.Context()
		isEnabled := settingsCache.GetBool(ctx, featureKey, false)

		if !isEnabled {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Feature not available",
				"code":  "FEATURE_DISABLED",
			})
		}

		return c.Next()
	}
}

// RequireRealtimeEnabled returns a middleware that ensures realtime feature is enabled
func RequireRealtimeEnabled(settingsCache *auth.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.features.enable_realtime")
}

// RequireStorageEnabled returns a middleware that ensures storage feature is enabled
func RequireStorageEnabled(settingsCache *auth.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.features.enable_storage")
}

// RequireFunctionsEnabled returns a middleware that ensures edge functions feature is enabled
func RequireFunctionsEnabled(settingsCache *auth.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.features.enable_functions")
}

// RequireJobsEnabled returns a middleware that ensures jobs feature is enabled
func RequireJobsEnabled(settingsCache *auth.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.features.enable_jobs")
}

// RequireAIEnabled returns a middleware that ensures AI chatbot feature is enabled
func RequireAIEnabled(settingsCache *auth.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.features.enable_ai")
}

// RequireRPCEnabled returns a middleware that ensures RPC feature is enabled
func RequireRPCEnabled(settingsCache *auth.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.features.enable_rpc")
}
