package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// SetupTOTP initiates 2FA setup by generating a TOTP secret
// POST /auth/2fa/setup
func (h *AuthHandler) SetupTOTP(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	var req struct {
		Issuer string `json:"issuer"`
	}
	_ = c.Bind().Body(&req)

	response, err := h.authService.MFAService().SetupTOTP(middleware.CtxWithTenant(c), userID, req.Issuer)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to setup TOTP")
		return SendInternalError(c, "Failed to setup 2FA")
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// EnableTOTP enables 2FA after verifying the TOTP code
// POST /auth/2fa/enable
func (h *AuthHandler) EnableTOTP(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Code == "" {
		return SendMissingField(c, "Code")
	}

	backupCodes, err := h.authService.MFAService().EnableTOTP(middleware.CtxWithTenant(c), userID, req.Code)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to enable TOTP")
		return SendBadRequest(c, "Invalid 2FA code", ErrCodeInvalidInput)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":      true,
		"backup_codes": backupCodes,
		"message":      "2FA enabled successfully. Please save your backup codes in a secure location.",
	})
}

// VerifyTOTP verifies a TOTP code during login and issues JWT tokens
// POST /auth/2fa/verify
func (h *AuthHandler) VerifyTOTP(c fiber.Ctx) error {
	var req struct {
		UserID string `json:"user_id"`
		Code   string `json:"code"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.UserID == "" || req.Code == "" {
		return SendBadRequest(c, "User ID and code are required", ErrCodeMissingField)
	}

	// Verify the 2FA code
	err := h.authService.MFAService().VerifyTOTP(middleware.CtxWithTenant(c), req.UserID, req.Code)
	if err != nil {
		log.Warn().Err(err).Str("user_id", req.UserID).Msg("Failed to verify TOTP")
		return SendBadRequest(c, "Invalid 2FA code", ErrCodeInvalidCredentials)
	}

	// Generate a complete sign-in response with tokens
	resp, err := h.authService.GenerateTokensForUser(middleware.CtxWithTenant(c), req.UserID)
	if err != nil {
		log.Error().Err(err).Str("user_id", req.UserID).Msg("Failed to generate tokens after 2FA verification")
		return SendInternalError(c, "Failed to complete authentication")
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// DisableTOTP disables 2FA for a user
// POST /auth/2fa/disable
func (h *AuthHandler) DisableTOTP(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Password == "" {
		return SendMissingField(c, "Password")
	}

	err := h.authService.MFAService().DisableTOTP(middleware.CtxWithTenant(c), userID, req.Password)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to disable TOTP")
		return SendBadRequest(c, "Failed to disable 2FA", ErrCodeInvalidCredentials)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "2FA disabled successfully",
	})
}

// GetTOTPStatus checks if 2FA is enabled for a user
// GET /auth/2fa/status
func (h *AuthHandler) GetTOTPStatus(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	enabled, err := h.authService.MFAService().IsTOTPEnabled(middleware.CtxWithTenant(c), userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to check TOTP status")
		return SendInternalError(c, "Failed to check 2FA status")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"totp_enabled": enabled,
	})
}
