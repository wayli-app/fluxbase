package api

import (
	"fmt"
	"time"

	"github.com/wayli-app/fluxbase/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// APIKeyHandler handles API key-related requests
type APIKeyHandler struct {
	apiKeyService *auth.APIKeyService
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler(apiKeyService *auth.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
	}
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	Name               string     `json:"name"`
	Description        *string    `json:"description,omitempty"`
	Scopes             []string   `json:"scopes"`
	RateLimitPerMinute int        `json:"rate_limit_per_minute"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
}

// UpdateAPIKeyRequest represents a request to update an API key
type UpdateAPIKeyRequest struct {
	Name               *string  `json:"name,omitempty"`
	Description        *string  `json:"description,omitempty"`
	Scopes             []string `json:"scopes,omitempty"`
	RateLimitPerMinute *int     `json:"rate_limit_per_minute,omitempty"`
}

// RegisterRoutes registers API key routes
func (h *APIKeyHandler) RegisterRoutes(app *fiber.App) {
	apiKeys := app.Group("/api/v1/api-keys")

	apiKeys.Post("/", h.CreateAPIKey)
	apiKeys.Get("/", h.ListAPIKeys)
	apiKeys.Get("/:id", h.GetAPIKey)
	apiKeys.Patch("/:id", h.UpdateAPIKey)
	apiKeys.Delete("/:id", h.DeleteAPIKey)
	apiKeys.Post("/:id/revoke", h.RevokeAPIKey)
}

// CreateAPIKey creates a new API key
func (h *APIKeyHandler) CreateAPIKey(c *fiber.Ctx) error {
	var req CreateAPIKeyRequest
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

	apiKey, err := h.apiKeyService.GenerateAPIKey(
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
			"error": fmt.Sprintf("Failed to create API key: %v", err),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(apiKey)
}

// ListAPIKeys lists all API keys
func (h *APIKeyHandler) ListAPIKeys(c *fiber.Ctx) error {
	// Optionally filter by user ID
	var userID *uuid.UUID
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		id, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid user ID",
			})
		}
		userID = &id
	}

	apiKeys, err := h.apiKeyService.ListAPIKeys(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to list API keys: %v", err),
		})
	}

	return c.JSON(apiKeys)
}

// GetAPIKey retrieves a single API key
func (h *APIKeyHandler) GetAPIKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid API key ID",
		})
	}

	// For simplicity, we'll just list and filter (in production, add a GetByID method)
	apiKeys, err := h.apiKeyService.ListAPIKeys(c.Context(), nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get API key: %v", err),
		})
	}

	for _, key := range apiKeys {
		if key.ID == id {
			return c.JSON(key)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"error": "API key not found",
	})
}

// UpdateAPIKey updates an API key's metadata
func (h *APIKeyHandler) UpdateAPIKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid API key ID",
		})
	}

	var req UpdateAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	err = h.apiKeyService.UpdateAPIKey(c.Context(), id, req.Name, req.Description, req.Scopes, req.RateLimitPerMinute)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to update API key: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "API key updated successfully",
	})
}

// RevokeAPIKey revokes an API key
func (h *APIKeyHandler) RevokeAPIKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid API key ID",
		})
	}

	err = h.apiKeyService.RevokeAPIKey(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to revoke API key: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "API key revoked successfully",
	})
}

// DeleteAPIKey permanently deletes an API key
func (h *APIKeyHandler) DeleteAPIKey(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid API key ID",
		})
	}

	err = h.apiKeyService.DeleteAPIKey(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to delete API key: %v", err),
		})
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}
