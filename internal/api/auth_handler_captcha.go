package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// GetCSRFToken returns the current CSRF token for the client
// Clients should call this endpoint first, then include the token in the X-CSRF-Token header
// GET /auth/csrf
func (h *AuthHandler) GetCSRFToken(c fiber.Ctx) error {
	// The CSRF middleware has already set the cookie
	// Return the token value so clients can use it in the X-CSRF-Token header
	token := c.Cookies("csrf_token")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"csrf_token": token,
	})
}

// GetCaptchaConfig returns the public CAPTCHA configuration for clients
// GET /auth/captcha/config
func (h *AuthHandler) GetCaptchaConfig(c fiber.Ctx) error {
	if h.captchaService == nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"enabled": false,
		})
	}

	config := h.captchaService.GetConfig()
	return c.Status(fiber.StatusOK).JSON(config)
}

// CheckCaptcha performs a pre-flight check to determine if CAPTCHA is required
// POST /auth/captcha/check
//
// This endpoint evaluates trust signals and returns whether CAPTCHA verification
// is needed for the subsequent auth action. It issues a challenge_id that must
// be included in the actual auth request.
//
// Request body:
//
//	{
//	  "endpoint": "login",                    // Required: signup, login, password_reset, magic_link
//	  "email": "user@example.com",            // Optional: for trust lookup
//	  "device_fingerprint": "abc123",         // Optional: browser fingerprint
//	  "trust_token": "tt_..."                 // Optional: token from previous CAPTCHA
//	}
//
// Response:
//
//	{
//	  "captcha_required": true,
//	  "reason": "new_ip_address",
//	  "trust_score": 35,
//	  "provider": "hcaptcha",
//	  "site_key": "...",
//	  "challenge_id": "ch_abc123...",
//	  "expires_at": "2024-01-15T10:05:00Z"
//	}
func (h *AuthHandler) CheckCaptcha(c fiber.Ctx) error {
	// Parse request
	var req auth.CaptchaCheckRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate endpoint
	validEndpoints := map[string]bool{
		"signup":         true,
		"login":          true,
		"password_reset": true,
		"magic_link":     true,
	}
	if !validEndpoints[req.Endpoint] {
		return SendBadRequest(c, "Invalid endpoint. Must be one of: signup, login, password_reset, magic_link", "INVALID_ENDPOINT")
	}

	// If CAPTCHA is not enabled at all, return early
	if h.captchaService == nil || !h.captchaService.IsEnabled() {
		return c.Status(fiber.StatusOK).JSON(auth.CaptchaCheckResponse{
			CaptchaRequired: false,
			Reason:          "captcha_disabled",
			ChallengeID:     "", // No challenge needed
		})
	}

	// If adaptive trust service is available, use it
	if h.captchaTrustService != nil {
		response, err := h.captchaTrustService.CheckCaptchaRequired(middleware.CtxWithTenant(c), req, c.IP(), c.Get("User-Agent"))
		if err != nil {
			log.Error().Err(err).Msg("Failed to check CAPTCHA requirement")
			// Fall back to requiring CAPTCHA on error
			return c.Status(fiber.StatusOK).JSON(auth.CaptchaCheckResponse{
				CaptchaRequired: true,
				Reason:          "trust_check_error",
				Provider:        h.captchaService.GetProvider(),
				SiteKey:         h.captchaService.GetSiteKey(),
			})
		}
		return c.Status(fiber.StatusOK).JSON(response)
	}

	// Fall back to static check (adaptive trust not configured)
	required := h.captchaService.IsEnabledForEndpoint(req.Endpoint)
	response := auth.CaptchaCheckResponse{
		CaptchaRequired: required,
		ChallengeID:     "", // No challenge tracking without trust service
	}
	if required {
		response.Reason = "captcha_enabled_for_endpoint"
		response.Provider = h.captchaService.GetProvider()
		response.SiteKey = h.captchaService.GetSiteKey()
	} else {
		response.Reason = "captcha_not_required_for_endpoint"
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
