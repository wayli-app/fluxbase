package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// SetupTOTP generates a new TOTP secret for 2FA
func (h *DashboardAuthHandler) SetupTOTP(c fiber.Ctx) error {
	userID, _ := uuid.Parse(middleware.GetUserID(c))

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	user, err := h.authService.GetUserByID(c.RequestCtx(), userID)
	if err != nil {
		return SendNotFound(c, "User not found")
	}

	// Parse optional issuer from request body
	var req struct {
		Issuer string `json:"issuer"` // Optional: custom issuer name for the QR code
	}
	// Ignore parse errors - issuer is optional and will default to config value
	_ = c.Bind().Body(&req)

	secret, qrURL, err := h.authService.SetupTOTP(c.RequestCtx(), userID, user.Email, req.Issuer)
	if err != nil {
		return SendInternalError(c, "Failed to setup 2FA")
	}

	return c.JSON(fiber.Map{
		"secret": secret,
		"qr_url": qrURL,
	})
}

// EnableTOTP enables 2FA after verifying the TOTP code
func (h *DashboardAuthHandler) EnableTOTP(c fiber.Ctx) error {
	userID, _ := uuid.Parse(middleware.GetUserID(c))

	var req struct {
		Code string `json:"code"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Code == "" {
		return SendBadRequest(c, "Code is required", ErrCodeMissingField)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())

	backupCodes, err := h.authService.EnableTOTP(c.RequestCtx(), userID, req.Code, ipAddress, userAgent)
	if err != nil {
		if err.Error() == "invalid TOTP code" {
			return SendUnauthorized(c, "Invalid 2FA code", ErrCodeInvalidCredentials)
		}
		return SendInternalError(c, "Failed to enable 2FA")
	}

	return c.JSON(fiber.Map{
		"message":      "2FA enabled successfully",
		"backup_codes": backupCodes,
	})
}

// DisableTOTP disables 2FA for the current user
func (h *DashboardAuthHandler) DisableTOTP(c fiber.Ctx) error {
	userID, _ := uuid.Parse(middleware.GetUserID(c))

	var req struct {
		Password string `json:"password"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Password == "" {
		return SendBadRequest(c, "Password is required", ErrCodeMissingField)
	}

	if err := h.requireAuthService(c); err != nil {
		return err
	}

	ipAddress := getIPAddress(c)
	userAgent := string(c.Request().Header.UserAgent())

	err := h.authService.DisableTOTP(c.RequestCtx(), userID, req.Password, ipAddress, userAgent)
	if err != nil {
		if err.Error() == "password is incorrect" {
			return SendUnauthorized(c, "Password is incorrect", ErrCodeInvalidCredentials)
		}
		return SendInternalError(c, "Failed to disable 2FA")
	}

	return apperrors.SendSuccess(c, "2FA disabled successfully")
}
