package secrets

import (
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler manages HTTP endpoints for secrets
type Handler struct {
	storage *Storage
}

// NewHandler creates a new secrets handler
func NewHandler(storage *Storage) *Handler {
	return &Handler{
		storage: storage,
	}
}

// CreateSecretRequest represents a request to create a secret
type CreateSecretRequest struct {
	Name        string     `json:"name"`
	Value       string     `json:"value"`
	Scope       string     `json:"scope"`               // "global" or "namespace"
	Namespace   *string    `json:"namespace,omitempty"` // Required if scope is "namespace"
	Description *string    `json:"description,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// UpdateSecretRequest represents a request to update a secret
type UpdateSecretRequest struct {
	Value       *string    `json:"value,omitempty"`
	Description *string    `json:"description,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// RegisterRoutes registers secrets routes with authentication
func (h *Handler) RegisterRoutes(app *fiber.App, authService *auth.Service, clientKeyService *auth.ClientKeyService, db *pgxpool.Pool, jwtManager *auth.JWTManager) {
	// Apply authentication middleware to all secrets routes
	secrets := app.Group("/api/v1/secrets",
		middleware.RequireAuthOrServiceKey(authService, clientKeyService, db, jwtManager),
	)

	// Read operations require read:secrets scope
	secrets.Get("/", middleware.RequireScope(auth.ScopeSecretsRead), h.ListSecrets)
	secrets.Get("/stats", middleware.RequireScope(auth.ScopeSecretsRead), h.GetStats)
	secrets.Get("/:id", middleware.RequireScope(auth.ScopeSecretsRead), h.GetSecret)
	secrets.Get("/:id/versions", middleware.RequireScope(auth.ScopeSecretsRead), h.GetVersions)

	// Write operations require write:secrets scope
	secrets.Post("/", middleware.RequireScope(auth.ScopeSecretsWrite), h.CreateSecret)
	secrets.Put("/:id", middleware.RequireScope(auth.ScopeSecretsWrite), h.UpdateSecret)
	secrets.Delete("/:id", middleware.RequireScope(auth.ScopeSecretsWrite), h.DeleteSecret)
	secrets.Post("/:id/rollback/:version", middleware.RequireScope(auth.ScopeSecretsWrite), h.RollbackToVersion)
}

// CreateSecret creates a new secret
func (h *Handler) CreateSecret(c *fiber.Ctx) error {
	var req CreateSecretRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}

	if req.Value == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Value is required",
		})
	}

	// Validate scope
	if req.Scope == "" {
		req.Scope = "global" // Default to global
	}

	if req.Scope != "global" && req.Scope != "namespace" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Scope must be 'global' or 'namespace'",
		})
	}

	if req.Scope == "namespace" && (req.Namespace == nil || *req.Namespace == "") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Namespace is required when scope is 'namespace'",
		})
	}

	if req.Scope == "global" {
		req.Namespace = nil // Ensure namespace is nil for global secrets
	}

	// Get user ID from context
	var userID *uuid.UUID
	if uid, ok := c.Locals("user_id").(uuid.UUID); ok {
		userID = &uid
	} else if uidStr, ok := c.Locals("user_id").(string); ok && uidStr != "" {
		if uid, err := uuid.Parse(uidStr); err == nil {
			userID = &uid
		}
	}

	secret := &Secret{
		Name:        req.Name,
		Scope:       req.Scope,
		Namespace:   req.Namespace,
		Description: req.Description,
		ExpiresAt:   req.ExpiresAt,
	}

	if err := h.storage.CreateSecret(c.Context(), secret, req.Value, userID); err != nil {
		// Check for duplicate key error
		if isDuplicateKeyError(err) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "A secret with this name already exists in the specified scope",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create secret: %v", err),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(secret)
}

// ListSecrets lists all secrets (metadata only, never values)
func (h *Handler) ListSecrets(c *fiber.Ctx) error {
	// Parse query parameters
	var scope *string
	if s := c.Query("scope"); s != "" {
		scope = &s
	}

	var namespace *string
	if ns := c.Query("namespace"); ns != "" {
		namespace = &ns
	}

	secrets, err := h.storage.ListSecrets(c.Context(), scope, namespace)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to list secrets: %v", err),
		})
	}

	if secrets == nil {
		secrets = []SecretSummary{}
	}

	return c.JSON(secrets)
}

// GetSecret retrieves a single secret (metadata only)
func (h *Handler) GetSecret(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid secret ID",
		})
	}

	secret, err := h.storage.GetSecret(c.Context(), id)
	if err != nil {
		if isNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get secret: %v", err),
		})
	}

	return c.JSON(secret)
}

// UpdateSecret updates a secret's value or metadata
func (h *Handler) UpdateSecret(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid secret ID",
		})
	}

	var req UpdateSecretRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Require at least one field to update
	if req.Value == nil && req.Description == nil && req.ExpiresAt == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one field (value, description, or expires_at) must be provided",
		})
	}

	// Get user ID from context
	var userID *uuid.UUID
	if uid, ok := c.Locals("user_id").(uuid.UUID); ok {
		userID = &uid
	} else if uidStr, ok := c.Locals("user_id").(string); ok && uidStr != "" {
		if uid, err := uuid.Parse(uidStr); err == nil {
			userID = &uid
		}
	}

	if err := h.storage.UpdateSecret(c.Context(), id, req.Value, req.Description, req.ExpiresAt, userID); err != nil {
		if isNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to update secret: %v", err),
		})
	}

	// Return updated secret
	secret, err := h.storage.GetSecret(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Secret updated but failed to retrieve updated data",
		})
	}

	return c.JSON(secret)
}

// DeleteSecret deletes a secret
func (h *Handler) DeleteSecret(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid secret ID",
		})
	}

	if err := h.storage.DeleteSecret(c.Context(), id); err != nil {
		if isNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to delete secret: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Secret deleted successfully",
	})
}

// GetVersions retrieves the version history for a secret
func (h *Handler) GetVersions(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid secret ID",
		})
	}

	versions, err := h.storage.GetVersions(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get versions: %v", err),
		})
	}

	if versions == nil {
		versions = []SecretVersion{}
	}

	return c.JSON(versions)
}

// RollbackToVersion restores a secret to a previous version
func (h *Handler) RollbackToVersion(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid secret ID",
		})
	}

	versionStr := c.Params("version")
	version := 0
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil || version < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid version number",
		})
	}

	// Get user ID from context
	var userID *uuid.UUID
	if uid, ok := c.Locals("user_id").(uuid.UUID); ok {
		userID = &uid
	} else if uidStr, ok := c.Locals("user_id").(string); ok && uidStr != "" {
		if uid, err := uuid.Parse(uidStr); err == nil {
			userID = &uid
		}
	}

	if err := h.storage.RollbackToVersion(c.Context(), id, version, userID); err != nil {
		if isNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": fmt.Sprintf("Version %d not found", version),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to rollback: %v", err),
		})
	}

	// Return updated secret
	secret, err := h.storage.GetSecret(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Rollback successful but failed to retrieve updated data",
		})
	}

	return c.JSON(secret)
}

// GetStats returns statistics about secrets
func (h *Handler) GetStats(c *fiber.Ctx) error {
	total, expiringSoon, expired, err := h.storage.GetStats(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get stats: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"total":         total,
		"expiring_soon": expiringSoon,
		"expired":       expired,
	})
}

// Helper functions for error detection
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "duplicate key") || contains(errStr, "unique constraint") || contains(errStr, "unique_secret_name_scope")
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "no rows") || contains(errStr, "not found")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
