package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// SendMagicLink handles sending magic link
// POST /auth/magiclink
func (h *AuthHandler) SendMagicLink(c fiber.Ctx) error {
	var req struct {
		Email        string `json:"email"`
		CaptchaToken string `json:"captcha_token,omitempty"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Verify CAPTCHA if enabled for magic_link
	if h.captchaService != nil {
		if err := h.captchaService.VerifyForEndpoint(middleware.CtxWithTenant(c), "magic_link", req.CaptchaToken, c.IP()); err != nil {
			if errors.Is(err, auth.ErrCaptchaRequired) {
				return SendBadRequest(c, "CAPTCHA verification required", "CAPTCHA_REQUIRED")
			}
			log.Warn().Err(err).Str("email", req.Email).Msg("CAPTCHA verification failed for magic link")
			return SendBadRequest(c, "CAPTCHA verification failed", "CAPTCHA_INVALID")
		}
	}

	// Validate email
	if req.Email == "" {
		return SendMissingField(c, "Email")
	}

	// Send magic link
	if err := h.authService.SendMagicLink(middleware.CtxWithTenant(c), req.Email); err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to send magic link")
		return SendBadRequest(c, "Failed to send magic link", ErrCodeInvalidInput)
	}

	// Return standard OTP response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":    nil,
		"session": nil,
	})
}

// VerifyMagicLink handles magic link verification
// POST /auth/magiclink/verify
func (h *AuthHandler) VerifyMagicLink(c fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate token
	if req.Token == "" {
		return SendMissingField(c, "Token")
	}

	// Verify magic link
	resp, err := h.authService.VerifyMagicLink(middleware.CtxWithTenant(c), req.Token)
	if err != nil {
		log.Error().Err(err).Msg("Failed to verify magic link")
		return SendBadRequest(c, "Invalid or expired magic link token", ErrCodeInvalidInput)
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
