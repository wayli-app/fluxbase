package api

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// SendOTP sends an OTP code via email or SMS
// POST /auth/otp/signin
func (h *AuthHandler) SendOTP(c fiber.Ctx) error {
	var req struct {
		Email   *string                 `json:"email,omitempty"`
		Phone   *string                 `json:"phone,omitempty"`
		Options *map[string]interface{} `json:"options,omitempty"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate that either email or phone is provided
	if err := auth.ValidateOTPContact(req.Email, req.Phone); err != nil {
		return SendBadRequest(c, "Email or phone is required", ErrCodeMissingField)
	}

	// Send OTP
	var err error
	purpose := "signin" // Default purpose
	if req.Options != nil {
		if p, ok := (*req.Options)["purpose"].(string); ok {
			purpose = p
		}
	}

	if req.Email != nil {
		err = h.authService.SendOTP(middleware.CtxWithTenant(c), *req.Email, purpose)
	} else if req.Phone != nil {
		// SMS OTP not yet fully implemented
		err = fmt.Errorf("SMS OTP not yet implemented")
	}

	if err != nil {
		log.Error().Str("error", err.Error()).Msg("Failed to send OTP")
		return SendInternalError(c, "Failed to send OTP code")
	}

	// Return standard OTP response
	// For send requests, user and session are both nil (OTP delivered but not verified yet)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":    nil,
		"session": nil,
	})
}

// VerifyOTP verifies an OTP code and creates a session
// POST /auth/otp/verify
func (h *AuthHandler) VerifyOTP(c fiber.Ctx) error {
	var req struct {
		Email *string `json:"email,omitempty"`
		Phone *string `json:"phone,omitempty"`
		Token string  `json:"token"`
		Type  string  `json:"type"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Token == "" {
		return SendMissingField(c, "OTP token")
	}

	// Verify OTP
	var otpCode *auth.OTPCode
	var err error

	// Validate that either email or phone is provided
	if err := auth.ValidateOTPContact(req.Email, req.Phone); err != nil {
		return SendBadRequest(c, "Email or phone is required", ErrCodeMissingField)
	}

	if req.Email != nil {
		otpCode, err = h.authService.OTPService().VerifyEmailOTP(middleware.CtxWithTenant(c), *req.Email, req.Token)
	} else if req.Phone != nil {
		// Phone OTP not yet fully implemented
		return SendErrorWithCode(c, 501, "Phone-based OTP authentication not yet implemented", "NOT_IMPLEMENTED")
	}

	if err != nil {
		log.Warn().Err(err).Msg("Failed to verify OTP")
		return SendUnauthorized(c, "Invalid or expired OTP code", ErrCodeInvalidCredentials)
	}

	// Get existing user - auto-creation is disabled for security
	// Users must register via signup endpoint first
	var user *auth.User
	if req.Email != nil && otpCode.Email != nil {
		user, err = h.authService.GetUserByEmail(middleware.CtxWithTenant(c), *otpCode.Email)
		if err != nil {
			log.Warn().Str("email", *otpCode.Email).Msg("OTP verification for non-existent user")
			return SendNotFound(c, "No account found for this email - please sign up first")
		}
	}

	// Generate tokens
	resp, err := h.authService.GenerateTokensForUser(middleware.CtxWithTenant(c), user.ID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate tokens")
		return SendInternalError(c, "Failed to complete authentication")
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// ResendOTP resends an OTP code
// POST /auth/otp/resend
func (h *AuthHandler) ResendOTP(c fiber.Ctx) error {
	var req struct {
		Type    string                  `json:"type"`
		Email   *string                 `json:"email,omitempty"`
		Phone   *string                 `json:"phone,omitempty"`
		Options *map[string]interface{} `json:"options,omitempty"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validate that either email or phone is provided
	if err := auth.ValidateOTPContact(req.Email, req.Phone); err != nil {
		return SendBadRequest(c, "Email or phone is required", ErrCodeMissingField)
	}

	purpose := "signin" // Default purpose
	if req.Options != nil {
		if p, ok := (*req.Options)["purpose"].(string); ok {
			purpose = p
		}
	}

	// Resend OTP
	var err error
	if req.Email != nil {
		err = h.authService.OTPService().ResendEmailOTP(middleware.CtxWithTenant(c), *req.Email, purpose)
	} else if req.Phone != nil {
		// SMS OTP not yet fully implemented
		err = fmt.Errorf("SMS OTP not yet implemented")
	}

	if err != nil {
		log.Error().Err(err).Msg("Failed to resend OTP")
		return SendInternalError(c, "Failed to resend OTP code")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"user":    nil,
		"session": nil,
	})
}
