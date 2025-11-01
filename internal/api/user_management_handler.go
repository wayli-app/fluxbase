package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/wayli-app/fluxbase/internal/auth"
)

// UserManagementHandler handles admin user management operations
type UserManagementHandler struct {
	userMgmtService *auth.UserManagementService
	authService     *auth.Service
}

// NewUserManagementHandler creates a new user management handler
func NewUserManagementHandler(userMgmtService *auth.UserManagementService, authService *auth.Service) *UserManagementHandler {
	return &UserManagementHandler{
		userMgmtService: userMgmtService,
		authService:     authService,
	}
}

// ListUsers lists all users with enriched metadata
func (h *UserManagementHandler) ListUsers(c *fiber.Ctx) error {
	excludeAdmins := c.QueryBool("exclude_admins", false)
	search := c.Query("search", "")
	limit := c.QueryInt("limit", 0) // 0 means no limit

	users, err := h.userMgmtService.ListEnrichedUsers(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Filter users based on query parameters
	filteredUsers := users

	// Exclude admins if requested
	if excludeAdmins {
		nonAdminUsers := make([]*auth.EnrichedUser, 0)
		for _, user := range filteredUsers {
			if user.Role != "admin" {
				nonAdminUsers = append(nonAdminUsers, user)
			}
		}
		filteredUsers = nonAdminUsers
	}

	// Search by email if provided
	if search != "" {
		searchResults := make([]*auth.EnrichedUser, 0)
		for _, user := range filteredUsers {
			if len(user.Email) >= len(search) && user.Email[:len(search)] == search {
				searchResults = append(searchResults, user)
			} else if contains(user.Email, search) {
				searchResults = append(searchResults, user)
			}
		}
		filteredUsers = searchResults
	}

	// Apply limit if specified
	if limit > 0 && len(filteredUsers) > limit {
		filteredUsers = filteredUsers[:limit]
	}

	return c.JSON(fiber.Map{
		"users": filteredUsers,
		"total": len(filteredUsers),
	})
}

// contains is a simple case-insensitive substring check
func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// InviteUser invites a new user
func (h *UserManagementHandler) InviteUser(c *fiber.Ctx) error {
	var req auth.InviteUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	resp, err := h.userMgmtService.InviteUser(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// DeleteUser deletes a user
func (h *UserManagementHandler) DeleteUser(c *fiber.Ctx) error {
	userID := c.Params("id")

	err := h.userMgmtService.DeleteUser(c.Context(), userID)
	if err != nil {
		if err == auth.ErrUserNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}

// UpdateUserRole updates a user's role
func (h *UserManagementHandler) UpdateUserRole(c *fiber.Ctx) error {
	userID := c.Params("id")

	var req struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := h.userMgmtService.UpdateUserRole(c.Context(), userID, req.Role)
	if err != nil {
		if err == auth.ErrUserNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(user)
}

// ResetUserPassword resets a user's password
func (h *UserManagementHandler) ResetUserPassword(c *fiber.Ctx) error {
	userID := c.Params("id")

	result, err := h.userMgmtService.ResetUserPassword(c.Context(), userID)
	if err != nil {
		if err == auth.ErrUserNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": result,
	})
}

// RegisterRoutes registers user management routes
func (h *UserManagementHandler) RegisterRoutes(app *fiber.App) {
	admin := app.Group("/api/v1/admin",
		AuthMiddleware(h.authService),
		RequireRole("admin"),
	)

	// User management routes (admin only)
	admin.Get("/users", h.ListUsers)
	admin.Post("/users/invite", h.InviteUser)
	admin.Delete("/users/:id", h.DeleteUser)
	admin.Patch("/users/:id/role", h.UpdateUserRole)
	admin.Post("/users/:id/reset-password", h.ResetUserPassword)
}
