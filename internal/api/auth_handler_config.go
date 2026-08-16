package api

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// GetAuthConfig returns the public authentication configuration for clients
// GET /auth/config
func (h *AuthHandler) GetAuthConfig(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	settingsCache := h.authService.GetSettingsCache()

	// Build response
	response := AuthConfigResponse{
		SignupEnabled:            h.authService.IsSignupEnabled(),
		RequireEmailVerification: settingsCache.GetBool(ctx, "app.auth.require_email_verification", false),
		MagicLinkEnabled:         settingsCache.GetBool(ctx, "app.auth.magic_link_enabled", false),
		PasswordLoginEnabled:     !settingsCache.GetBool(ctx, "app.auth.disable_app_password_login", false), // Inverted: disabled=false means enabled=true
		MFAAvailable:             true,                                                                      // MFA is always available, users opt-in
		PasswordMinLength:        settingsCache.GetInt(ctx, "app.auth.password_min_length", 8),
		PasswordRequireUppercase: settingsCache.GetBool(ctx, "app.auth.password_require_uppercase", false),
		PasswordRequireLowercase: settingsCache.GetBool(ctx, "app.auth.password_require_lowercase", false),
		PasswordRequireNumber:    settingsCache.GetBool(ctx, "app.auth.password_require_number", false),
		PasswordRequireSpecial:   settingsCache.GetBool(ctx, "app.auth.password_require_special", false),
		OAuthProviders:           []OAuthProviderPublic{},
		SAMLProviders:            []SAMLProviderPublic{},
		AnonKey:                  h.anonKey,
	}

	// Fetch OAuth providers
	oauthQuery := `
		SELECT provider_name, display_name, redirect_url
		FROM platform.oauth_providers
		WHERE enabled = TRUE AND allow_app_login = TRUE
		ORDER BY display_name
	`
	rows, err := h.db.Query(ctx, oauthQuery)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list OAuth providers for auth config")
	} else {
		defer rows.Close()
		for rows.Next() {
			var providerName, displayName, redirectURL string
			if err := rows.Scan(&providerName, &displayName, &redirectURL); err != nil {
				log.Error().Err(err).Msg("Failed to scan OAuth provider")
				continue
			}
			response.OAuthProviders = append(response.OAuthProviders, OAuthProviderPublic{
				Provider:     providerName,
				DisplayName:  displayName,
				AuthorizeURL: fmt.Sprintf("%s/api/v1/auth/oauth/%s/authorize", h.baseURL, providerName),
			})
		}
	}

	// Fetch SAML providers
	if h.samlService != nil {
		samlProviders := h.samlService.GetProvidersForApp()
		for _, provider := range samlProviders {
			response.SAMLProviders = append(response.SAMLProviders, SAMLProviderPublic{
				Provider:    provider.Name,
				DisplayName: provider.Name, // SAML providers use Name as display name
			})
		}
	}

	// Get CAPTCHA config
	if h.captchaService != nil {
		captchaConfig := h.captchaService.GetConfig()
		response.Captcha = &captchaConfig
	} else {
		response.Captcha = &auth.CaptchaConfigResponse{
			Enabled: false,
		}
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// isPasswordLoginDisabled checks if password login is disabled for app users
func (h *AuthHandler) isPasswordLoginDisabled(ctx context.Context) bool {
	// Emergency override via environment variable
	if os.Getenv("FLUXBASE_APP_FORCE_PASSWORD_LOGIN") == "true" {
		return false // Password login forced enabled
	}

	settingsCache := h.authService.GetSettingsCache()
	return settingsCache.GetBool(ctx, "app.auth.disable_app_password_login", false)
}

// resolvePublishableAnonKey returns the configured anon key (publishable —
// safe to expose to clients, same key the web app ships in its HTML).
// Reads the value or the key file; empty when neither is configured, in
// which case /auth/config simply omits the field (omitempty).
func resolvePublishableAnonKey(cfg *config.Config) string {
	key := cfg.Tenants.Default.AnonKey
	if cfg.Tenants.Default.AnonKeyFile != "" {
		if data, err := os.ReadFile(cfg.Tenants.Default.AnonKeyFile); err == nil {
			key = strings.TrimSpace(string(data))
		} else {
			log.Warn().Err(err).Str("file", cfg.Tenants.Default.AnonKeyFile).Msg("Failed to read anon key file")
		}
	}
	return strings.TrimSpace(key)
}
