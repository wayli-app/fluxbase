package middleware

import (
	"github.com/gofiber/fiber/v3"

	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/settings"
)

func RequireFeatureEnabled(settingsCache *settings.SettingsCache, featureKey string) fiber.Handler {
	return func(c fiber.Ctx) error {
		// If settings cache is nil, treat the feature as disabled
		if settingsCache == nil {
			return apperrors.SendErrorWithDetails(c, fiber.StatusServiceUnavailable, "Feature not available", apperrors.ErrCodeFeatureDisabled, "", "", fiber.Map{"feature_key": featureKey})
		}

		ctx := c.RequestCtx()
		isEnabled := settingsCache.GetBool(ctx, featureKey, true)

		if !isEnabled {
			return apperrors.SendErrorWithDetails(c, fiber.StatusServiceUnavailable, "Feature is not enabled", apperrors.ErrCodeFeatureDisabled, "", "", fiber.Map{"feature_key": featureKey})
		}

		return c.Next()
	}
}

// RequireRealtimeEnabled returns a middleware that ensures realtime feature is enabled
func RequireRealtimeEnabled(settingsCache *settings.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.realtime.enabled")
}

// RequireStorageEnabled returns a middleware that ensures storage feature is enabled
func RequireStorageEnabled(settingsCache *settings.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.storage.enabled")
}

// RequireFunctionsEnabled returns a middleware that ensures edge functions feature is enabled
func RequireFunctionsEnabled(settingsCache *settings.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.functions.enabled")
}

// RequireJobsEnabled returns a middleware that ensures jobs feature is enabled
func RequireJobsEnabled(settingsCache *settings.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.jobs.enabled")
}

// RequireAIEnabled returns a middleware that ensures AI chatbot feature is enabled
func RequireAIEnabled(settingsCache *settings.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.ai.enabled")
}

// RequireRPCEnabled returns a middleware that ensures RPC feature is enabled
func RequireRPCEnabled(settingsCache *settings.SettingsCache) fiber.Handler {
	return RequireFeatureEnabled(settingsCache, "app.rpc.enabled")
}

// fiber:context-methods migrated
