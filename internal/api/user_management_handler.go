package api

import (
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/gofiber/fiber/v2"
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
	const defaultLimit = 100
	const maxLimit = 1000

	excludeAdmins := c.QueryBool("exclude_admins", false)
	search := c.Query("search", "")
	limit := c.QueryInt("limit", defaultLimit)
	offset := c.QueryInt("offset", 0)
	userType := c.Query("type", "app") // "app" for auth.users, "dashboard" for dashboard.users

	// Enforce maximum limit
	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}
	if offset < 0 {
		offset = 0
	}

	users, err := h.userMgmtService.ListEnrichedUsers(c.Context(), userType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Ensure we never return null (nil slice serializes to null in JSON)
	if users == nil {
		users = []*auth.EnrichedUser{}
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

	// Search by email if provided (case-insensitive)
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

	// Calculate total before pagination
	total := len(filteredUsers)

	// Apply offset
	if offset >= len(filteredUsers) {
		filteredUsers = []*auth.EnrichedUser{}
	} else {
		filteredUsers = filteredUsers[offset:]
	}

	// Apply limit
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

// GetUserByID gets a single user by ID with enriched metadata
func (h *UserManagementHandler) GetUserByID(c *fiber.Ctx) error {
	userID := c.Params("id")
	userType := c.Query("type", "app") // "app" for auth.users, "dashboard" for dashboard.users

	user, err := h.userMgmtService.GetEnrichedUserByID(c.Context(), userID, userType)
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

// InviteUser invites a new user
func (h *UserManagementHandler) InviteUser(c *fiber.Ctx) error {
	var req auth.InviteUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userType := c.Query("type", "app") // "app" for auth.users, "dashboard" for dashboard.users

	resp, err := h.userMgmtService.InviteUser(c.Context(), req, userType)
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
	userType := c.Query("type", "app") // "app" for auth.users, "dashboard" for dashboard.users

	err := h.userMgmtService.DeleteUser(c.Context(), userID, userType)
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
	userType := c.Query("type", "app") // "app" for auth.users, "dashboard" for dashboard.users

	var req struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := h.userMgmtService.UpdateUserRole(c.Context(), userID, req.Role, userType)
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

// UpdateUser updates a user's information (email, role, password, user_metadata)
func (h *UserManagementHandler) UpdateUser(c *fiber.Ctx) error {
	userID := c.Params("id")
	userType := c.Query("type", "app") // "app" for auth.users, "dashboard" for dashboard.users

	var req auth.UpdateAdminUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := h.userMgmtService.UpdateUser(c.Context(), userID, req, userType)
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
	userType := c.Query("type", "app") // "app" for auth.users, "dashboard" for dashboard.users

	result, err := h.userMgmtService.ResetUserPassword(c.Context(), userID, userType)
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

// LockUser locks a user account
func (h *UserManagementHandler) LockUser(c *fiber.Ctx) error {
	userID := c.Params("id")
	userType := c.Query("type", "app")

	err := h.userMgmtService.LockUser(c.Context(), userID, userType)
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
		"message": "User account locked successfully",
	})
}

// UnlockUser unlocks a user account
func (h *UserManagementHandler) UnlockUser(c *fiber.Ctx) error {
	userID := c.Params("id")
	userType := c.Query("type", "app")

	err := h.userMgmtService.UnlockUser(c.Context(), userID, userType)
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
		"message": "User account unlocked successfully",
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
	admin.Get("/users/:id", h.GetUserByID)
	admin.Post("/users/invite", h.InviteUser)
	admin.Patch("/users/:id", h.UpdateUser)
	admin.Delete("/users/:id", h.DeleteUser)
	admin.Patch("/users/:id/role", h.UpdateUserRole)
	admin.Post("/users/:id/reset-password", h.ResetUserPassword)
	admin.Post("/users/:id/lock", h.LockUser)
	admin.Post("/users/:id/unlock", h.UnlockUser)
}
