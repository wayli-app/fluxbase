package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// StartImpersonation starts an admin impersonation session
func (h *AuthHandler) StartImpersonation(c fiber.Ctx) error {
	adminUserID := middleware.GetUserID(c)
	if adminUserID == "" {
		return SendMissingAuth(c)
	}

	var req auth.StartImpersonationRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	req.IPAddress = c.IP()
	req.UserAgent = c.Get("User-Agent")

	tenantID := c.Get("X-FB-Tenant")

	resp, err := h.authService.StartImpersonation(middleware.CtxWithTenant(c), adminUserID, tenantID, req)
	if err != nil {
		if errors.Is(err, auth.ErrNotAdmin) || errors.Is(err, auth.ErrNotTenantAdmin) {
			return SendForbidden(c, "Insufficient permissions", ErrCodeAccessDenied)
		} else if errors.Is(err, auth.ErrSelfImpersonation) {
			return SendBadRequest(c, "Cannot impersonate yourself", ErrCodeInvalidInput)
		} else if errors.Is(err, auth.ErrTargetUserNotInTenant) {
			return SendForbidden(c, "Target user is not in this tenant", ErrCodeAccessDenied)
		}
		return SendInternalError(c, "Failed to start impersonation")
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// StopImpersonation stops the active impersonation session
func (h *AuthHandler) StopImpersonation(c fiber.Ctx) error {
	adminUserID := middleware.GetUserID(c)
	if adminUserID == "" {
		return SendMissingAuth(c)
	}

	err := h.authService.StopImpersonation(middleware.CtxWithTenant(c), adminUserID)
	if err != nil {
		if errors.Is(err, auth.ErrNoActiveImpersonation) {
			return SendNotFound(c, "No active impersonation session found")
		}
		return SendInternalError(c, "Failed to stop impersonation")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Impersonation session ended",
	})
}

// GetActiveImpersonation gets the active impersonation session
func (h *AuthHandler) GetActiveImpersonation(c fiber.Ctx) error {
	adminUserID := middleware.GetUserID(c)
	if adminUserID == "" {
		return SendMissingAuth(c)
	}

	session, err := h.authService.GetActiveImpersonation(middleware.CtxWithTenant(c), adminUserID)
	if err != nil {
		if errors.Is(err, auth.ErrNoActiveImpersonation) {
			return SendNotFound(c, "No active impersonation session found")
		}
		return SendInternalError(c, "Failed to get active impersonation")
	}

	return c.Status(fiber.StatusOK).JSON(session)
}

// ListImpersonationSessions lists impersonation sessions for audit
func (h *AuthHandler) ListImpersonationSessions(c fiber.Ctx) error {
	adminUserID := middleware.GetUserID(c)
	if adminUserID == "" {
		return SendMissingAuth(c)
	}

	limit := fiber.Query[int](c, "limit", 50)
	offset := fiber.Query[int](c, "offset", 0)

	sessions, err := h.authService.ListImpersonationSessions(middleware.CtxWithTenant(c), adminUserID, limit, offset)
	if err != nil {
		return SendInternalError(c, "Failed to list impersonation sessions")
	}

	return c.Status(fiber.StatusOK).JSON(sessions)
}

// StartAnonImpersonation starts impersonation as anonymous user
func (h *AuthHandler) StartAnonImpersonation(c fiber.Ctx) error {
	adminUserID := middleware.GetUserID(c)
	if adminUserID == "" {
		return SendMissingAuth(c)
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Reason == "" {
		return SendMissingField(c, "Reason")
	}

	ipAddress := c.IP()
	userAgent := c.Get("User-Agent")
	tenantID := c.Get("X-FB-Tenant")

	resp, err := h.authService.StartAnonImpersonation(middleware.CtxWithTenant(c), adminUserID, tenantID, req.Reason, ipAddress, userAgent)
	if err != nil {
		if errors.Is(err, auth.ErrNotAdmin) || errors.Is(err, auth.ErrNotTenantAdmin) {
			return SendForbidden(c, "Insufficient permissions", ErrCodeAccessDenied)
		}
		return SendInternalError(c, "Failed to start anonymous impersonation")
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *AuthHandler) StartServiceImpersonation(c fiber.Ctx) error {
	adminUserID := middleware.GetUserID(c)
	if adminUserID == "" {
		return SendMissingAuth(c)
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Reason == "" {
		return SendMissingField(c, "Reason")
	}

	ipAddress := c.IP()
	userAgent := c.Get("User-Agent")
	tenantID := c.Get("X-FB-Tenant")

	resp, err := h.authService.StartServiceImpersonation(middleware.CtxWithTenant(c), adminUserID, tenantID, req.Reason, ipAddress, userAgent)
	if err != nil {
		if errors.Is(err, auth.ErrNotAdmin) || errors.Is(err, auth.ErrNotTenantAdmin) {
			return SendForbidden(c, "Insufficient permissions", ErrCodeAccessDenied)
		}
		return SendInternalError(c, "Failed to start service impersonation")
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
