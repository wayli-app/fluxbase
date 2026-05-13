package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// VerifyEmail verifies a user's email address using a verification token
// POST /auth/verify-email
func (h *AuthHandler) VerifyEmail(c fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Token == "" {
		return SendMissingField(c, "Token")
	}

	user, err := h.authService.VerifyEmailToken(middleware.CtxWithTenant(c), req.Token)
	if err != nil {
		// Check for specific token errors
		if errors.Is(err, auth.ErrEmailVerificationTokenNotFound) {
			return SendBadRequest(c, "Invalid or expired verification token", "INVALID_TOKEN")
		}
		if errors.Is(err, auth.ErrEmailVerificationTokenExpired) {
			return SendBadRequest(c, "Verification token has expired. Please request a new one.", "TOKEN_EXPIRED")
		}
		if errors.Is(err, auth.ErrEmailVerificationTokenUsed) {
			return SendBadRequest(c, "This verification token has already been used", "TOKEN_USED")
		}
		log.Error().Err(err).Msg("Failed to verify email")
		return SendBadRequest(c, "Email verification failed", ErrCodeInvalidInput)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Email verified successfully. You can now sign in.",
		"user":    user,
	})
}

// ResendVerificationEmail resends the verification email to a user
// POST /auth/verify-email/resend
func (h *AuthHandler) ResendVerificationEmail(c fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Email == "" {
		return SendMissingField(c, "Email")
	}

	// Get user by email
	user, err := h.authService.GetUserByEmail(middleware.CtxWithTenant(c), req.Email)
	if err != nil {
		// Don't reveal if email exists - return generic success message
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "If an account exists with this email, a verification link has been sent.",
		})
	}

	// Check if already verified
	if user.EmailVerified {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "Email is already verified. You can sign in.",
		})
	}

	// Send verification email
	if err := h.authService.SendEmailVerification(middleware.CtxWithTenant(c), user.ID, user.Email); err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("Failed to resend verification email")
		return SendInternalError(c, "Failed to send verification email. Please try again later.")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Verification email sent. Please check your inbox.",
	})
}
