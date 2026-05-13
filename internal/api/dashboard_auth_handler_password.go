package api

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
)

// RequestPasswordReset initiates a password reset for a dashboard user
func (h *DashboardAuthHandler) RequestPasswordReset(c fiber.Ctx) error {
	// Check if email service is configured
	if h.emailService == nil {
		return SendBadRequest(c, "Email service is not configured. Please configure an email provider to enable password reset.", ErrCodeFeatureDisabled)
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Email == "" {
		return SendBadRequest(c, "Email is required", ErrCodeMissingField)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	token, err := h.authService.RequestPasswordReset(c.RequestCtx(), req.Email)
	if err != nil {
		// Log the error but don't reveal details to user
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to request password reset")
		// Still return success to prevent email enumeration
	}

	// If we got a token, send the password reset email
	if token != "" {
		resetLink := h.baseURL + "/admin/reset-password?token=" + token
		if err := h.emailService.SendPasswordReset(c.RequestCtx(), req.Email, token, resetLink); err != nil {
			log.Error().Err(err).Str("email", req.Email).Msg("Failed to send password reset email")
			// Don't return error to prevent email enumeration
		} else {
			log.Info().Str("email", req.Email).Msg("Password reset email sent")
		}
	}

	// Always return success to prevent email enumeration
	return c.JSON(fiber.Map{
		"message": "If an account with that email exists, a password reset link has been sent.",
	})
}

// VerifyPasswordResetToken verifies a password reset token is valid
func (h *DashboardAuthHandler) VerifyPasswordResetToken(c fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Token == "" {
		return SendBadRequest(c, "Token is required", ErrCodeMissingField)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	valid, err := h.authService.VerifyPasswordResetToken(c.RequestCtx(), req.Token)
	if err != nil {
		return SendInternalError(c, "Failed to verify token")
	}

	if !valid {
		return c.JSON(fiber.Map{
			"valid":   false,
			"message": "Invalid or expired token",
		})
	}

	return c.JSON(fiber.Map{
		"valid":   true,
		"message": "Token is valid",
	})
}

// ConfirmPasswordReset resets the password using a valid reset token
func (h *DashboardAuthHandler) ConfirmPasswordReset(c fiber.Ctx) error {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Token == "" || req.NewPassword == "" {
		return SendBadRequest(c, "Token and new password are required", ErrCodeMissingField)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	err := h.authService.ResetPassword(c.RequestCtx(), req.Token, req.NewPassword)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid or expired") {
			return SendBadRequest(c, "Invalid or expired password reset token", ErrCodeInvalidToken)
		}
		if strings.Contains(errMsg, "password must be") {
			return SendBadRequest(c, errMsg, ErrCodeValidationFailed)
		}
		return SendInternalError(c, "Failed to reset password")
	}

	return apperrors.SendSuccess(c, "Password reset successfully")
}
