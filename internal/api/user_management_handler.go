package api

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/nimbleflux/fluxbase/internal/auth"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

type UserManagementHandler struct {
	userMgmtService *auth.UserManagementService
	authService     *auth.Service
}

func NewUserManagementHandler(userMgmtService *auth.UserManagementService, authService *auth.Service) *UserManagementHandler {
	return &UserManagementHandler{
		userMgmtService: userMgmtService,
		authService:     authService,
	}
}

func (h *UserManagementHandler) requireService(c fiber.Ctx) error {
	if h.userMgmtService == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "not_initialized")
	}
	return nil
}

func (h *UserManagementHandler) ListUsers(c fiber.Ctx) error {
	const defaultLimit = 100
	const maxLimit = 1000

	excludeAdmins := fiber.Query[bool](c, "exclude_admins", false)
	search := c.Query("search", "")
	limit := fiber.Query[int](c, "limit", defaultLimit)
	offset := fiber.Query[int](c, "offset", 0)
	userType := c.Query("type", "app")

	tenantID := middleware.GetTenantID(c)
	tenantSource, _ := c.Locals("tenant_source").(string)
	isInstanceAdmin, _ := c.Locals("is_instance_admin").(bool)
	isDefaultTenant, _ := c.Locals("is_default_tenant").(bool)

	if isInstanceAdmin && (tenantSource == "default" || isDefaultTenant) {
		tenantID = ""
	}

	limit, offset = NormalizePaginationParams(limit, offset, defaultLimit, maxLimit)

	if err := h.requireService(c); err != nil {
		return err
	}

	users, err := h.userMgmtService.ListEnrichedUsers(c.RequestCtx(), userType, tenantID)
	if err != nil {
		return SendInternalError(c, "Failed to list users")
	}

	if users == nil {
		users = []*auth.EnrichedUser{}
	}

	filteredUsers := users

	if excludeAdmins {
		nonAdminUsers := make([]*auth.EnrichedUser, 0)
		for _, user := range filteredUsers {
			if user.Role != "admin" {
				nonAdminUsers = append(nonAdminUsers, user)
			}
		}
		filteredUsers = nonAdminUsers
	}

	if search != "" {
		searchLower := strings.ToLower(search)
		searchResults := make([]*auth.EnrichedUser, 0)
		for _, user := range filteredUsers {
			emailLower := strings.ToLower(user.Email)
			if strings.Contains(emailLower, searchLower) {
				searchResults = append(searchResults, user)
			}
		}
		filteredUsers = searchResults
	}

	total := len(filteredUsers)

	if offset >= len(filteredUsers) {
		filteredUsers = []*auth.EnrichedUser{}
	} else {
		filteredUsers = filteredUsers[offset:]
	}

	if len(filteredUsers) > limit {
		filteredUsers = filteredUsers[:limit]
	}

	return c.JSON(fiber.Map{
		"users":  filteredUsers,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *UserManagementHandler) GetUserByID(c fiber.Ctx) error {
	userID := c.Params("id")
	userType := c.Query("type", "app")

	if err := h.requireService(c); err != nil {
		return err
	}

	user, err := h.userMgmtService.GetEnrichedUserByID(c.RequestCtx(), userID, userType)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return SendNotFound(c, "User not found")
		}
		return SendInternalError(c, "Failed to get user")
	}

	return c.JSON(user)
}

func (h *UserManagementHandler) InviteUser(c fiber.Ctx) error {
	var req auth.InviteUserRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	userType := c.Query("type", "app")

	if req.TenantID == "" {
		tenantID := middleware.GetTenantID(c)
		req.TenantID = tenantID
	}

	resp, err := h.userMgmtService.InviteUser(c.RequestCtx(), req, userType)
	if err != nil {
		return SendInternalError(c, "Failed to invite user")
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *UserManagementHandler) DeleteUser(c fiber.Ctx) error {
	userID := c.Params("id")
	userType := c.Query("type", "app")

	if err := h.requireService(c); err != nil {
		return err
	}

	err := h.userMgmtService.DeleteUser(c.RequestCtx(), userID, userType)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return SendNotFound(c, "User not found")
		}
		return SendInternalError(c, "Failed to delete user")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}

func (h *UserManagementHandler) UpdateUserRole(c fiber.Ctx) error {
	userID := c.Params("id")
	userType := c.Query("type", "app")

	var req struct {
		Role string `json:"role"`
	}
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	user, err := h.userMgmtService.UpdateUserRole(c.RequestCtx(), userID, req.Role, userType)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return SendNotFound(c, "User not found")
		}
		return SendInternalError(c, "Failed to update user role")
	}

	return c.JSON(user)
}

func (h *UserManagementHandler) UpdateUser(c fiber.Ctx) error {
	userID := c.Params("id")
	userType := c.Query("type", "app")

	var req auth.UpdateAdminUserRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	user, err := h.userMgmtService.UpdateUser(c.RequestCtx(), userID, req, userType)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return SendNotFound(c, "User not found")
		}
		return SendInternalError(c, "Failed to update user")
	}

	return c.JSON(user)
}

func (h *UserManagementHandler) ResetUserPassword(c fiber.Ctx) error {
	userID := c.Params("id")
	userType := c.Query("type", "app")

	if err := h.requireService(c); err != nil {
		return err
	}

	result, err := h.userMgmtService.ResetUserPassword(c.RequestCtx(), userID, userType)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return SendNotFound(c, "User not found")
		}
		return SendInternalError(c, "Failed to reset user password")
	}

	return c.JSON(fiber.Map{
		"message": result,
	})
}

func (h *UserManagementHandler) LockUser(c fiber.Ctx) error {
	userID := c.Params("id")
	userType := c.Query("type", "app")

	if err := h.requireService(c); err != nil {
		return err
	}

	err := h.userMgmtService.LockUser(c.RequestCtx(), userID, userType)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return SendNotFound(c, "User not found")
		}
		return SendInternalError(c, "Failed to lock user")
	}

	return apperrors.SendSuccess(c, "User account locked successfully")
}

func (h *UserManagementHandler) UnlockUser(c fiber.Ctx) error {
	userID := c.Params("id")
	userType := c.Query("type", "app")

	if err := h.requireService(c); err != nil {
		return err
	}

	err := h.userMgmtService.UnlockUser(c.RequestCtx(), userID, userType)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return SendNotFound(c, "User not found")
		}
		return SendInternalError(c, "Failed to unlock user")
	}

	return apperrors.SendSuccess(c, "User account unlocked successfully")
}

// fiber:context-methods migrated
