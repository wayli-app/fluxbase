package api

import (
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ClientKeyHandler handles client key-related requests
type ClientKeyHandler struct {
	clientKeyService *auth.ClientKeyService
}

// NewClientKeyHandler creates a new client key handler
func NewClientKeyHandler(clientKeyService *auth.ClientKeyService) *ClientKeyHandler {
	return &ClientKeyHandler{
		clientKeyService: clientKeyService,
	}
}

// CreateClientKeyRequest represents a request to create a client key
type CreateClientKeyRequest struct {
	Name               string     `json:"name"`
	Description        *string    `json:"description,omitempty"`
	Scopes             []string   `json:"scopes"`
	RateLimitPerMinute int        `json:"rate_limit_per_minute"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
}

// UpdateClientKeyRequest represents a request to update a client key
type UpdateClientKeyRequest struct {
	Name               *string  `json:"name,omitempty"`
	Description        *string  `json:"description,omitempty"`
	Scopes             []string `json:"scopes,omitempty"`
	RateLimitPerMinute *int     `json:"rate_limit_per_minute,omitempty"`
}

// RegisterRoutes registers client key routes with authentication
func (h *ClientKeyHandler) RegisterRoutes(app *fiber.App, authService *auth.Service, clientKeyService *auth.ClientKeyService, db *pgxpool.Pool, jwtManager *auth.JWTManager) {
	// Apply authentication middleware to all client key routes
	clientKeys := app.Group("/api/v1/client-keys",
		middleware.RequireAuthOrServiceKey(authService, clientKeyService, db, jwtManager),
	)

	// Read operations require read:clientkeys scope
	clientKeys.Get("/", middleware.RequireScope(auth.ScopeClientKeysRead), h.ListClientKeys)
	clientKeys.Get("/:id", middleware.RequireScope(auth.ScopeClientKeysRead), h.GetClientKey)

	// Write operations require write:clientkeys scope
	clientKeys.Post("/", middleware.RequireScope(auth.ScopeClientKeysWrite), h.CreateClientKey)
	clientKeys.Patch("/:id", middleware.RequireScope(auth.ScopeClientKeysWrite), h.UpdateClientKey)
	clientKeys.Delete("/:id", middleware.RequireScope(auth.ScopeClientKeysWrite), h.DeleteClientKey)
	clientKeys.Post("/:id/revoke", middleware.RequireScope(auth.ScopeClientKeysWrite), h.RevokeClientKey)
}

// CreateClientKey creates a new client key
func (h *ClientKeyHandler) CreateClientKey(c *fiber.Ctx) error {
	var req CreateClientKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}

	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	var userIDPtr *uuid.UUID
	if ok {
		userIDPtr = &userID
	}

	clientKey, err := h.clientKeyService.GenerateClientKey(
		c.Context(),
		req.Name,
		req.Description,
		userIDPtr,
		req.Scopes,
		req.RateLimitPerMinute,
		req.ExpiresAt,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create client key: %v", err),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(clientKey)
}

// ListClientKeys lists client keys
// Non-admin users can only see their own keys
func (h *ClientKeyHandler) ListClientKeys(c *fiber.Ctx) error {
	// Get current user info
	currentUserID, _ := c.Locals("user_id").(string)
	role, _ := c.Locals("user_role").(string)
	isAdmin := role == "admin" || role == "dashboard_admin" || role == "service_role"

	// Determine which user's keys to list
	var userID *uuid.UUID

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		// User specified a user_id filter
		id, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid user ID",
			})
		}

		// Non-admin users can only view their own keys
		if !isAdmin && userIDStr != currentUserID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Cannot view other users' client keys",
			})
		}
		userID = &id
	} else if !isAdmin && currentUserID != "" {
		// Non-admin users without filter: only show their own keys
		id, err := uuid.Parse(currentUserID)
		if err == nil {
			userID = &id
		}
	}
	// Admins without filter: show all keys (userID stays nil)

	clientKeys, err := h.clientKeyService.ListClientKeys(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to list client keys: %v", err),
		})
	}

	return c.JSON(clientKeys)
}

// GetClientKey retrieves a single client key
// Non-admin users can only view their own keys
func (h *ClientKeyHandler) GetClientKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid client key ID",
		})
	}

	// Get current user info
	currentUserID, _ := c.Locals("user_id").(string)
	role, _ := c.Locals("user_role").(string)
	isAdmin := role == "admin" || role == "dashboard_admin" || role == "service_role"

	// For simplicity, we'll just list and filter (in production, add a GetByID method)
	clientKeys, err := h.clientKeyService.ListClientKeys(c.Context(), nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get client key: %v", err),
		})
	}

	for _, key := range clientKeys {
		if key.ID == id {
			// Non-admin users can only view their own keys
			if !isAdmin && key.UserID != nil && key.UserID.String() != currentUserID {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Cannot view other users' client keys",
				})
			}
			return c.JSON(key)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "Client key not found",
	})
}

// UpdateClientKey updates a client key's metadata
func (h *ClientKeyHandler) UpdateClientKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid client key ID",
		})
	}

	var req UpdateClientKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	err = h.clientKeyService.UpdateClientKey(c.Context(), id, req.Name, req.Description, req.Scopes, req.RateLimitPerMinute)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to update client key: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Client key updated successfully",
	})
}

// RevokeClientKey revokes a client key
func (h *ClientKeyHandler) RevokeClientKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid client key ID",
		})
	}

	err = h.clientKeyService.RevokeClientKey(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to revoke client key: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Client key revoked successfully",
	})
}

// DeleteClientKey permanently deletes a client key
func (h *ClientKeyHandler) DeleteClientKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid client key ID",
		})
	}

	err = h.clientKeyService.DeleteClientKey(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to delete client key: %v", err),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}
