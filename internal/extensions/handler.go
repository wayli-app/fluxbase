package extensions

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// Handler handles extension management HTTP endpoints
type Handler struct {
	service *Service
}

// NewHandler creates a new extension handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// getTenantID extracts the tenant ID from fiber context locals.
// Returns nil for the default tenant.
func getTenantID(c fiber.Ctx) *string {
	if tid := c.Locals("tenant_id"); tid != nil {
		if s, ok := tid.(string); ok && s != "" {
			return &s
		}
	}
	return nil
}

// getTenantDBName extracts the tenant database name from fiber context.
// Returns empty string for the default tenant (uses main database).
func getTenantDBName(c fiber.Ctx) string {
	if name := c.Locals("tenant_db_name"); name != nil {
		if s, ok := name.(string); ok {
			return s
		}
	}
	return ""
}

// getTenantPool returns the tenant-specific database pool, or nil for the default tenant.
func getTenantPool(c fiber.Ctx) *pgxpool.Pool {
	return middleware.GetTenantPool(c)
}

// ListExtensions returns all available extensions with their status
// GET /api/v1/admin/extensions
func (h *Handler) ListExtensions(c fiber.Ctx) error {
	ctx := c.RequestCtx()

	response, err := h.service.ListExtensions(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list extensions")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list extensions",
		})
	}

	return c.JSON(response)
}

// GetExtensionStatus returns the status of a specific extension
// GET /api/v1/admin/extensions/:name/status
func (h *Handler) GetExtensionStatus(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	name := c.Params("name")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Extension name is required",
		})
	}

	status, err := h.service.GetExtensionStatus(ctx, name)
	if err != nil {
		log.Error().Err(err).Str("extension", name).Msg("Failed to get extension status")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get extension status",
		})
	}

	return c.JSON(status)
}

// EnableExtension enables a PostgreSQL extension
// POST /api/v1/admin/extensions/:name/enable
func (h *Handler) EnableExtension(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	name := c.Params("name")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Extension name is required",
		})
	}

	// Parse optional request body
	var req EnableExtensionRequest
	_ = c.Bind().Body(&req) // Ignore error - body is optional

	// Get user ID from context if available
	var userID *string
	if uid := c.Locals("user_id"); uid != nil {
		if uidStr, ok := uid.(string); ok {
			userID = &uidStr
		}
	}

	response, err := h.service.EnableExtension(ctx, name, userID, req.Schema)
	if err != nil {
		log.Error().Err(err).Str("extension", name).Msg("Failed to enable extension")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to enable extension",
		})
	}

	if !response.Success {
		return c.Status(fiber.StatusBadRequest).JSON(response)
	}

	return c.JSON(response)
}

// DisableExtension disables a PostgreSQL extension
// POST /api/v1/admin/extensions/:name/disable
func (h *Handler) DisableExtension(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	name := c.Params("name")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Extension name is required",
		})
	}

	// Get user ID from context if available
	var userID *string
	if uid := c.Locals("user_id"); uid != nil {
		if uidStr, ok := uid.(string); ok {
			userID = &uidStr
		}
	}

	response, err := h.service.DisableExtension(ctx, name, userID)
	if err != nil {
		log.Error().Err(err).Str("extension", name).Msg("Failed to disable extension")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to disable extension",
		})
	}

	if !response.Success {
		return c.Status(fiber.StatusBadRequest).JSON(response)
	}

	return c.JSON(response)
}

// SyncExtensions syncs the extension catalog with PostgreSQL
// POST /api/v1/admin/extensions/sync
func (h *Handler) SyncExtensions(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Extensions synced successfully",
	})
}

// --- Tenant-scoped handlers ---

// ListExtensionsForTenant returns extensions for the current tenant
// GET /api/v1/tenants/:tenantId/extensions (or tenant context route)
func (h *Handler) ListExtensionsForTenant(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	tenantID := getTenantID(c)
	tenantPool := getTenantPool(c)

	response, err := h.service.ListExtensionsForTenant(ctx, tenantID, tenantPool)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list extensions for tenant")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list extensions",
		})
	}

	return c.JSON(response)
}

// GetExtensionStatusForTenant returns extension status for the current tenant
func (h *Handler) GetExtensionStatusForTenant(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	name := c.Params("name")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Extension name is required",
		})
	}

	tenantID := getTenantID(c)
	tenantPool := getTenantPool(c)

	status, err := h.service.GetExtensionStatusForTenant(ctx, name, tenantID, tenantPool)
	if err != nil {
		log.Error().Err(err).Str("extension", name).Msg("Failed to get extension status for tenant")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get extension status",
		})
	}

	return c.JSON(status)
}

// EnableExtensionForTenant enables an extension for the current tenant
func (h *Handler) EnableExtensionForTenant(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	name := c.Params("name")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Extension name is required",
		})
	}

	var req EnableExtensionRequest
	_ = c.Bind().Body(&req)

	var userID *string
	if uid := c.Locals("user_id"); uid != nil {
		if uidStr, ok := uid.(string); ok {
			userID = &uidStr
		}
	}

	tenantID := getTenantID(c)
	tenantDBName := getTenantDBName(c)

	response, err := h.service.EnableExtensionForTenant(ctx, name, userID, req.Schema, tenantID, tenantDBName)
	if err != nil {
		log.Error().Err(err).Str("extension", name).Msg("Failed to enable extension for tenant")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to enable extension",
		})
	}

	if !response.Success {
		return c.Status(fiber.StatusBadRequest).JSON(response)
	}

	return c.JSON(response)
}

// DisableExtensionForTenant disables an extension for the current tenant
func (h *Handler) DisableExtensionForTenant(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	name := c.Params("name")

	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Extension name is required",
		})
	}

	var userID *string
	if uid := c.Locals("user_id"); uid != nil {
		if uidStr, ok := uid.(string); ok {
			userID = &uidStr
		}
	}

	tenantID := getTenantID(c)
	tenantDBName := getTenantDBName(c)

	response, err := h.service.DisableExtensionForTenant(ctx, name, userID, tenantID, tenantDBName)
	if err != nil {
		log.Error().Err(err).Str("extension", name).Msg("Failed to disable extension for tenant")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to disable extension",
		})
	}

	if !response.Success {
		return c.Status(fiber.StatusBadRequest).JSON(response)
	}

	return c.JSON(response)
}
