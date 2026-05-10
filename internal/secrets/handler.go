package secrets

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
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

// CreateSecret creates a new secret
func (h *Handler) CreateSecret(c fiber.Ctx) error {
	var req CreateSecretRequest
	if err := c.Bind().Body(&req); err != nil {
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
	userID := getUserIDFromContext(c)

	secret := &Secret{
		Name:        req.Name,
		Scope:       req.Scope,
		Namespace:   req.Namespace,
		Description: req.Description,
		ExpiresAt:   req.ExpiresAt,
	}

	if err := h.storage.CreateSecret(middleware.CtxWithTenant(c), secret, req.Value, userID); err != nil {
		// Check for duplicate key error
		if isDuplicateKeyError(err) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "A secret with this name already exists in the specified scope",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create secret",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(secret)
}

// ListSecrets lists all secrets (metadata only, never values)
func (h *Handler) ListSecrets(c fiber.Ctx) error {
	// Parse query parameters
	var scope *string
	if s := c.Query("scope"); s != "" {
		scope = &s
	}

	var namespace *string
	if ns := c.Query("namespace"); ns != "" {
		if ns == "default" {
			ns = ""
		}
		namespace = &ns
	}

	secrets, err := h.storage.ListSecrets(middleware.CtxWithTenant(c), scope, namespace)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to list secrets",
		})
	}

	if secrets == nil {
		secrets = []SecretSummary{}
	}

	return c.JSON(secrets)
}

// GetSecret retrieves a single secret (metadata only)
func (h *Handler) GetSecret(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid secret ID",
		})
	}

	secret, err := h.storage.GetSecret(middleware.CtxWithTenant(c), id)
	if err != nil {
		if isNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get secret",
		})
	}

	return c.JSON(secret)
}

// UpdateSecret updates a secret's value or metadata
func (h *Handler) UpdateSecret(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid secret ID",
		})
	}

	var req UpdateSecretRequest
	if err := c.Bind().Body(&req); err != nil {
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

	userID := getUserIDFromContext(c)

	if err := h.storage.UpdateSecret(middleware.CtxWithTenant(c), id, req.Value, req.Description, req.ExpiresAt, userID); err != nil {
		if isNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update secret",
		})
	}

	// Return updated secret
	secret, err := h.storage.GetSecret(middleware.CtxWithTenant(c), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Secret updated but failed to retrieve updated data",
		})
	}

	return c.JSON(secret)
}

// DeleteSecret deletes a secret
func (h *Handler) DeleteSecret(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid secret ID",
		})
	}

	if err := h.storage.DeleteSecret(middleware.CtxWithTenant(c), id); err != nil {
		if isNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete secret",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Secret deleted successfully",
	})
}

// GetVersions retrieves the version history for a secret
func (h *Handler) GetVersions(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid secret ID",
		})
	}

	versions, err := h.storage.GetVersions(middleware.CtxWithTenant(c), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get versions",
		})
	}

	if versions == nil {
		versions = []SecretVersion{}
	}

	return c.JSON(versions)
}

// RollbackToVersion restores a secret to a previous version
func (h *Handler) RollbackToVersion(c fiber.Ctx) error {
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

	userID := getUserIDFromContext(c)

	if err := h.storage.RollbackToVersion(middleware.CtxWithTenant(c), id, version, userID); err != nil {
		if isNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": fmt.Sprintf("Version %d not found", version),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to rollback secret",
		})
	}

	// Return updated secret
	secret, err := h.storage.GetSecret(middleware.CtxWithTenant(c), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Rollback successful but failed to retrieve updated data",
		})
	}

	return c.JSON(secret)
}

// GetStats returns statistics about secrets
func (h *Handler) GetStats(c fiber.Ctx) error {
	total, expiringSoon, expired, err := h.storage.GetStats(middleware.CtxWithTenant(c))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get stats",
		})
	}

	return c.JSON(fiber.Map{
		"total":         total,
		"expiring_soon": expiringSoon,
		"expired":       expired,
	})
}

// getNamespaceFromQuery extracts the optional namespace query parameter
func getNamespaceFromQuery(c fiber.Ctx) *string {
	if ns := c.Query("namespace"); ns != "" {
		if ns == "default" {
			ns = ""
		}
		return &ns
	}
	return nil
}

// getUserIDFromContext extracts user ID from fiber context
func getUserIDFromContext(c fiber.Ctx) *uuid.UUID {
	uidStr := middleware.GetUserID(c)
	if uidStr == "" {
		return nil
	}
	uid, err := uuid.Parse(uidStr)
	if err != nil {
		return nil
	}
	return &uid
}

// GetSecretByName retrieves a secret by name (metadata only)
func (h *Handler) GetSecretByName(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Secret name is required",
		})
	}

	namespace := getNamespaceFromQuery(c)

	secret, err := h.storage.GetSecretByName(middleware.CtxWithTenant(c), name, namespace)
	if err != nil {
		if isNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get secret",
		})
	}

	return c.JSON(secret)
}

// UpdateSecretByName updates a secret's value or metadata by name
func (h *Handler) UpdateSecretByName(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Secret name is required",
		})
	}

	namespace := getNamespaceFromQuery(c)

	var req UpdateSecretRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Value == nil && req.Description == nil && req.ExpiresAt == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "At least one field (value, description, or expires_at) must be provided",
		})
	}

	userID := getUserIDFromContext(c)

	secret, err := h.storage.GetSecretByName(middleware.CtxWithTenant(c), name, namespace)
	if err != nil {
		if isNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get secret",
		})
	}

	if err := h.storage.UpdateSecret(middleware.CtxWithTenant(c), secret.ID, req.Value, req.Description, req.ExpiresAt, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update secret",
		})
	}

	updatedSecret, err := h.storage.GetSecret(middleware.CtxWithTenant(c), secret.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Secret updated but failed to retrieve updated data",
		})
	}

	return c.JSON(updatedSecret)
}

// DeleteSecretByName deletes a secret by name
func (h *Handler) DeleteSecretByName(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Secret name is required",
		})
	}

	namespace := getNamespaceFromQuery(c)

	secret, err := h.storage.GetSecretByName(middleware.CtxWithTenant(c), name, namespace)
	if err != nil {
		if isNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get secret",
		})
	}

	if err := h.storage.DeleteSecret(middleware.CtxWithTenant(c), secret.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete secret",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Secret deleted successfully",
	})
}

// GetVersionsByName retrieves the version history for a secret by name
func (h *Handler) GetVersionsByName(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Secret name is required",
		})
	}

	namespace := getNamespaceFromQuery(c)

	secret, err := h.storage.GetSecretByName(middleware.CtxWithTenant(c), name, namespace)
	if err != nil {
		if isNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get secret",
		})
	}

	versions, err := h.storage.GetVersions(middleware.CtxWithTenant(c), secret.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get versions",
		})
	}

	if versions == nil {
		versions = []SecretVersion{}
	}

	return c.JSON(versions)
}

// RollbackByName restores a secret to a previous version by name
func (h *Handler) RollbackByName(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Secret name is required",
		})
	}

	versionStr := c.Params("version")
	version := 0
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil || version < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid version number",
		})
	}

	namespace := getNamespaceFromQuery(c)
	userID := getUserIDFromContext(c)

	secret, err := h.storage.GetSecretByName(middleware.CtxWithTenant(c), name, namespace)
	if err != nil {
		if isNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Secret not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get secret",
		})
	}

	if err := h.storage.RollbackToVersion(middleware.CtxWithTenant(c), secret.ID, version, userID); err != nil {
		if isNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": fmt.Sprintf("Version %d not found", version),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to rollback secret",
		})
	}

	updatedSecret, err := h.storage.GetSecret(middleware.CtxWithTenant(c), secret.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Rollback successful but failed to retrieve updated data",
		})
	}

	return c.JSON(updatedSecret)
}

// Helper functions for error detection
func isDuplicateKeyError(err error) bool {
	return database.IsUniqueViolation(err)
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "no rows") || strings.Contains(errStr, "not found")
}

// fiber:context-methods migrated
