package api

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/tenantdb"
)

// UploadTenantSchemaRequest represents the request body for uploading a tenant schema
type UploadTenantSchemaRequest struct {
	Schema string `json:"schema"`
}

// GetTenantSchemaStatus returns the status of a tenant's declarative schema
func (h *TenantHandler) GetTenantSchemaStatus(c fiber.Ctx) error {
	ctx := c.Context()
	tenantID := c.Params("id")

	// Check if tenant exists
	t, err := h.Storage.GetTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, tenantdb.ErrTenantNotFound) {
			return SendNotFound(c, "Tenant not found")
		}
		log.Error().Err(err).Msg("Failed to get tenant")
		return SendInternalError(c, "Failed to get tenant")
	}

	// Check if declarative service is configured
	declarativeSvc := h.Manager.GetDeclarativeService()
	if declarativeSvc == nil {
		return c.JSON(fiber.Map{
			"enabled":             false,
			"message":             "Tenant declarative schemas are not enabled",
			"has_schema_file":     false,
			"has_pending_changes": false,
		})
	}

	// Get schema status
	status, err := h.Manager.GetTenantSchemaStatus(ctx, tenantID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get tenant schema status")
		return SendInternalError(c, "Failed to get tenant schema status")
	}

	return c.JSON(fiber.Map{
		"enabled":                  true,
		"tenant_id":                tenantID,
		"tenant_slug":              t.Slug,
		"schema_file":              status.SchemaFile,
		"has_schema_file":          status.SchemaFingerprint != "",
		"schema_fingerprint":       status.SchemaFingerprint,
		"last_applied_fingerprint": status.LastAppliedFingerprint,
		"last_applied_at":          status.LastAppliedAt,
		"has_pending_changes":      status.HasPendingChanges,
		"uses_main_database":       t.UsesMainDatabase(),
	})
}

// ApplyTenantSchema applies the declarative schema for a tenant
func (h *TenantHandler) ApplyTenantSchema(c fiber.Ctx) error {
	ctx := c.Context()
	tenantID := c.Params("id")

	// Check if tenant exists
	t, err := h.Storage.GetTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, tenantdb.ErrTenantNotFound) {
			return SendNotFound(c, "Tenant not found")
		}
		log.Error().Err(err).Msg("Failed to get tenant")
		return SendInternalError(c, "Failed to get tenant")
	}

	// Check if tenant uses main database
	if t.UsesMainDatabase() {
		return SendBadRequest(c, "Cannot apply declarative schema to tenant using main database", ErrCodeInvalidInput)
	}

	// Check if declarative service is configured
	declarativeSvc := h.Manager.GetDeclarativeService()
	if declarativeSvc == nil {
		return SendBadRequest(c, "Tenant declarative schemas are not enabled", ErrCodeFeatureDisabled)
	}

	// Apply the schema
	if err := h.Manager.ApplyTenantDeclarativeSchema(ctx, tenantID); err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to apply tenant schema")
		return SendInternalError(c, "Failed to apply schema")
	}

	log.Info().Str("tenant_id", tenantID).Str("tenant_slug", t.Slug).Msg("Tenant schema applied")

	return c.JSON(fiber.Map{
		"status":      "applied",
		"tenant_id":   tenantID,
		"tenant_slug": t.Slug,
	})
}

// GetStoredSchema retrieves the stored schema content for a tenant
func (h *TenantHandler) GetStoredSchema(c fiber.Ctx) error {
	ctx := c.Context()
	tenantID := c.Params("id")

	// Check if tenant exists
	t, err := h.Storage.GetTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, tenantdb.ErrTenantNotFound) {
			return SendNotFound(c, "Tenant not found")
		}
		log.Error().Err(err).Msg("Failed to get tenant")
		return SendInternalError(c, "Failed to get tenant")
	}

	// Check if declarative service is configured
	declarativeSvc := h.Manager.GetDeclarativeService()
	if declarativeSvc == nil {
		return SendBadRequest(c, "Tenant declarative schemas are not enabled", ErrCodeFeatureDisabled)
	}

	// Get stored schema content
	content, fingerprint, updatedAt, err := declarativeSvc.GetStoredSchemaContent(ctx, t.Slug)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get stored schema")
		return SendInternalError(c, "Failed to get stored schema")
	}

	if content == "" {
		return c.JSON(fiber.Map{
			"has_schema":  false,
			"tenant_id":   tenantID,
			"tenant_slug": t.Slug,
		})
	}

	return c.JSON(fiber.Map{
		"has_schema":  true,
		"tenant_id":   tenantID,
		"tenant_slug": t.Slug,
		"schema":      content,
		"fingerprint": fingerprint,
		"updated_at":  updatedAt,
	})
}

// UploadTenantSchema uploads and stores schema content for a tenant
func (h *TenantHandler) UploadTenantSchema(c fiber.Ctx) error {
	ctx := c.Context()
	tenantID := c.Params("id")

	// Check if tenant exists
	t, err := h.Storage.GetTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, tenantdb.ErrTenantNotFound) {
			return SendNotFound(c, "Tenant not found")
		}
		log.Error().Err(err).Msg("Failed to get tenant")
		return SendInternalError(c, "Failed to get tenant")
	}

	// Check if declarative service is configured
	declarativeSvc := h.Manager.GetDeclarativeService()
	if declarativeSvc == nil {
		return SendBadRequest(c, "Tenant declarative schemas are not enabled", ErrCodeFeatureDisabled)
	}

	// Parse request body
	var req UploadTenantSchemaRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Schema == "" {
		return SendBadRequest(c, "Schema content cannot be empty", ErrCodeInvalidInput)
	}

	// Store the schema content
	if err := declarativeSvc.StoreSchemaContent(ctx, t.Slug, req.Schema); err != nil {
		log.Error().Err(err).Msg("Failed to store schema")
		return SendInternalError(c, "Failed to store schema")
	}

	// Calculate fingerprint for response
	_, fingerprint, _, _ := declarativeSvc.GetStoredSchemaContent(ctx, t.Slug)

	log.Info().Str("tenant_id", tenantID).Str("tenant_slug", t.Slug).Msg("Tenant schema uploaded")

	return c.JSON(fiber.Map{
		"status":      "uploaded",
		"tenant_id":   tenantID,
		"tenant_slug": t.Slug,
		"fingerprint": fingerprint,
	})
}

// ApplyUploadedTenantSchema applies the previously uploaded schema for a tenant
func (h *TenantHandler) ApplyUploadedTenantSchema(c fiber.Ctx) error {
	ctx := c.Context()
	tenantID := c.Params("id")

	// Check if tenant exists
	t, err := h.Storage.GetTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, tenantdb.ErrTenantNotFound) {
			return SendNotFound(c, "Tenant not found")
		}
		log.Error().Err(err).Msg("Failed to get tenant")
		return SendInternalError(c, "Failed to get tenant")
	}

	// Check if tenant uses main database
	if t.UsesMainDatabase() {
		return SendBadRequest(c, "Cannot apply declarative schema to tenant using main database", ErrCodeInvalidInput)
	}

	// Check if declarative service is configured
	declarativeSvc := h.Manager.GetDeclarativeService()
	if declarativeSvc == nil {
		return SendBadRequest(c, "Tenant declarative schemas are not enabled", ErrCodeFeatureDisabled)
	}

	// Get stored schema content
	content, fingerprint, _, err := declarativeSvc.GetStoredSchemaContent(ctx, t.Slug)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get stored schema")
		return SendInternalError(c, "Failed to get stored schema")
	}

	if content == "" {
		return SendNotFound(c, "No stored schema found for this tenant. Upload a schema first.")
	}

	// Apply the schema from stored content
	if err := declarativeSvc.ApplyTenantSchemaFromContent(ctx, t, content); err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to apply tenant schema")
		return SendInternalError(c, "Failed to apply schema")
	}

	log.Info().Str("tenant_id", tenantID).Str("tenant_slug", t.Slug).Msg("Tenant stored schema applied")

	return c.JSON(fiber.Map{
		"status":      "applied",
		"tenant_id":   tenantID,
		"tenant_slug": t.Slug,
		"fingerprint": fingerprint,
	})
}

// DeleteStoredSchema deletes the stored schema content for a tenant
func (h *TenantHandler) DeleteStoredSchema(c fiber.Ctx) error {
	ctx := c.Context()
	tenantID := c.Params("id")

	// Check if tenant exists
	t, err := h.Storage.GetTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, tenantdb.ErrTenantNotFound) {
			return SendNotFound(c, "Tenant not found")
		}
		log.Error().Err(err).Msg("Failed to get tenant")
		return SendInternalError(c, "Failed to get tenant")
	}

	// Check if declarative service is configured
	declarativeSvc := h.Manager.GetDeclarativeService()
	if declarativeSvc == nil {
		return SendBadRequest(c, "Tenant declarative schemas are not enabled", ErrCodeFeatureDisabled)
	}

	// Delete the stored schema
	if err := declarativeSvc.DeleteStoredSchema(ctx, t.Slug); err != nil {
		log.Error().Err(err).Msg("Failed to delete stored schema")
		return SendInternalError(c, "Failed to delete stored schema")
	}

	log.Info().Str("tenant_id", tenantID).Str("tenant_slug", t.Slug).Msg("Tenant stored schema deleted")

	return c.SendStatus(fiber.StatusNoContent)
}
