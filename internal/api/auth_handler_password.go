package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// RequestPasswordReset handles password reset requests
// POST /auth/password/reset
func (h *AuthHandler) RequestPasswordReset(c fiber.Ctx) error {
	var req struct {
		Email        string `json:"email"`
		RedirectTo   string `json:"redirect_to,omitempty"`
		CaptchaToken string `json:"captcha_token,omitempty"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Verify CAPTCHA if enabled for password_reset
	if h.captchaService != nil {
		if err := h.captchaService.VerifyForEndpoint(middleware.CtxWithTenant(c), "password_reset", req.CaptchaToken, c.IP()); err != nil {
			if errors.Is(err, auth.ErrCaptchaRequired) {
				return SendBadRequest(c, "CAPTCHA verification required", "CAPTCHA_REQUIRED")
			}
			log.Warn().Err(err).Str("email", req.Email).Msg("CAPTCHA verification failed for password reset")
			return SendBadRequest(c, "CAPTCHA verification failed", "CAPTCHA_INVALID")
		}
	}

	// Validate email
	if req.Email == "" {
		return SendMissingField(c, "Email")
	}

	// Request password reset (this won't reveal if user exists)
	if err := h.authService.RequestPasswordReset(middleware.CtxWithTenant(c), req.Email, req.RedirectTo); err != nil {
		// Check for SMTP not configured error - this should be returned to the user
		if errors.Is(err, auth.ErrSMTPNotConfigured) {
			return SendBadRequest(c, "SMTP is not configured. Please configure an email provider to enable password reset.", "SMTP_NOT_CONFIGURED")
		}
		// Check for invalid redirect URL - return error to prevent misuse
		if errors.Is(err, auth.ErrInvalidRedirectURL) {
			return SendBadRequest(c, "Invalid redirect_to URL. Must be a valid HTTP or HTTPS URL.", "INVALID_REDIRECT_URL")
		}
		// Check for rate limiting - user requested reset too soon
		if errors.Is(err, auth.ErrPasswordResetTooSoon) {
			return SendErrorWithCode(c, 429, "Password reset requested too recently. Please wait 60 seconds before trying again.", ErrCodeRateLimited)
		}
		// Check for email sending failure - this should be returned to the user
		if errors.Is(err, auth.ErrEmailSendFailed) {
			log.Error().Err(err).Str("email", req.Email).Msg("Failed to send password reset email")
			return SendInternalError(c, "Failed to send password reset email. Please try again later.")
		}
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to request password reset")
		// Don't reveal if user exists - always return success
	}

	// Return standard OTP response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":    nil,
		"session": nil,
	})
}

// ResetPassword handles password reset with token
// POST /auth/password/reset/confirm
func (h *AuthHandler) ResetPassword(c fiber.Ctx) error {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate required fields
	if req.Token == "" {
		return SendMissingField(c, "Token")
	}
	if req.NewPassword == "" {
		return SendMissingField(c, "New password")
	}

	// Reset password and get user ID
	userID, err := h.authService.ResetPassword(middleware.CtxWithTenant(c), req.Token, req.NewPassword)
	if err != nil {
		log.Error().Err(err).Msg("Failed to reset password")
		return SendBadRequest(c, "Invalid or expired reset token", ErrCodeInvalidInput)
	}

	// Generate new tokens for the user
	resp, err := h.authService.GenerateTokensForUser(middleware.CtxWithTenant(c), userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate tokens after password reset")
		return SendInternalError(c, "Failed to generate authentication tokens")
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// VerifyPasswordResetToken handles password reset token verification
// POST /auth/password/reset/verify
func (h *AuthHandler) VerifyPasswordResetToken(c fiber.Ctx) error {
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

	// Verify token
	if err := h.authService.VerifyPasswordResetToken(middleware.CtxWithTenant(c), req.Token); err != nil {
		log.Error().Err(err).Msg("Failed to verify password reset token")
		return SendBadRequest(c, "Invalid or expired reset token", ErrCodeInvalidInput)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Token is valid",
	})
}
