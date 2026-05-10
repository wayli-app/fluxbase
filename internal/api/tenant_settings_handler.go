package api

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/settings"
	"github.com/nimbleflux/fluxbase/internal/tenantdb"
)

// TenantSettingsHandler handles tenant-level settings API endpoints
type TenantSettingsHandler struct {
	settingsSvc *settings.UnifiedService
	tenantDB    *tenantdb.Repository
}

// NewTenantSettingsHandler creates a new tenant settings handler
func NewTenantSettingsHandler(settingsSvc *settings.UnifiedService, tenantDB *tenantdb.Repository) *TenantSettingsHandler {
	return &TenantSettingsHandler{
		settingsSvc: settingsSvc,
		tenantDB:    tenantDB,
	}
}

// TenantSettingsResponse represents the response for tenant settings
type TenantSettingsResponse struct {
	TenantID  string                              `json:"tenant_id"`
	Settings  map[string]settings.ResolvedSetting `json:"settings"`
	CreatedAt string                              `json:"created_at,omitempty"`
	UpdatedAt string                              `json:"updated_at,omitempty"`
}

// UpdateTenantSettingsRequest represents the request to update tenant settings
type UpdateTenantSettingsRequest struct {
	Settings map[string]any `json:"settings"`
	Secrets  map[string]any `json:"secrets,omitempty"`
}

// GetTenantSettings returns all tenant-specific settings with resolved values
// GET /admin/tenants/:id/settings
func (h *TenantSettingsHandler) GetTenantSettings(c fiber.Ctx) error {
	ctx := c.Context()
	tenantID := c.Params("id")

	if tenantID == "" {
		return SendMissingField(c, "tenant_id")
	}

	// Verify tenant exists
	tenant, err := h.tenantDB.GetTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, tenantdb.ErrTenantNotFound) {
			return SendResourceNotFound(c, "Tenant")
		}
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get tenant")
		return SendInternalError(c, "Failed to get tenant")
	}

	// Get tenant settings
	tenantSettings, err := h.settingsSvc.GetTenantSettings(ctx, tenantID)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get tenant settings")
		return SendInternalError(c, "Failed to get tenant settings")
	}

	// Get instance settings for overridable list
	instanceSettings, err := h.settingsSvc.GetInstanceSettings(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get instance settings")
		return SendInternalError(c, "Failed to get settings")
	}

	// Build response with resolved settings
	resolvedSettings := make(map[string]settings.ResolvedSetting)

	// Add all tenant settings with source info
	for path, value := range tenantSettings.Settings {
		overridable := h.isPathOverridable(path, instanceSettings.OverridableSettings)
		resolvedSettings[path] = settings.ResolvedSetting{
			Value:         value,
			Source:        "tenant",
			IsOverridable: overridable,
		}
	}

	return c.JSON(fiber.Map{
		"tenant_id":            tenantID,
		"tenant_name":          tenant.Name,
		"settings":             resolvedSettings,
		"overridable_settings": instanceSettings.OverridableSettings,
		"created_at":           tenantSettings.CreatedAt,
		"updated_at":           tenantSettings.UpdatedAt,
	})
}

// UpdateTenantSettings updates tenant-specific settings
// PATCH /admin/tenants/:id/settings
func (h *TenantSettingsHandler) UpdateTenantSettings(c fiber.Ctx) error {
	ctx := c.Context()
	tenantID := c.Params("id")

	if tenantID == "" {
		return SendMissingField(c, "tenant_id")
	}

	// Verify tenant exists
	_, err := h.tenantDB.GetTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, tenantdb.ErrTenantNotFound) {
			return SendResourceNotFound(c, "Tenant")
		}
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get tenant")
		return SendInternalError(c, "Failed to get tenant")
	}

	var req UpdateTenantSettingsRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Settings == nil && req.Secrets == nil {
		return SendBadRequest(c, "Settings or secrets are required", ErrCodeMissingField)
	}

	// Update regular settings
	for path, value := range req.Settings {
		if err := h.settingsSvc.SetTenantSetting(ctx, tenantID, path, value, false); err != nil {
			if errors.Is(err, settings.ErrNotOverridable) {
				return SendBadRequest(c, "Setting '"+path+"' is not overridable at tenant level", ErrCodeInvalidInput)
			}
			log.Error().Err(err).Str("tenant_id", tenantID).Str("path", path).Msg("Failed to set tenant setting")
			return SendInternalError(c, "Failed to update tenant settings")
		}
	}

	// Update secret settings
	for path, value := range req.Secrets {
		if err := h.settingsSvc.SetTenantSetting(ctx, tenantID, path, value, true); err != nil {
			if errors.Is(err, settings.ErrNotOverridable) {
				return SendBadRequest(c, "Secret '"+path+"' is not overridable at tenant level", ErrCodeInvalidInput)
			}
			log.Error().Err(err).Str("tenant_id", tenantID).Str("path", path).Msg("Failed to set tenant secret")
			return SendInternalError(c, "Failed to update tenant secrets")
		}
	}

	log.Info().Str("tenant_id", tenantID).Int("settings", len(req.Settings)).Int("secrets", len(req.Secrets)).Msg("Updated tenant settings")

	// Return updated settings
	return h.GetTenantSettings(c)
}

// DeleteTenantSetting removes a tenant-specific setting (resets to instance default)
// DELETE /admin/tenants/:id/settings/*path
func (h *TenantSettingsHandler) DeleteTenantSetting(c fiber.Ctx) error {
	ctx := c.Context()
	tenantID := c.Params("id")
	settingPath := c.Params("*")

	if tenantID == "" {
		return SendMissingField(c, "tenant_id")
	}

	if settingPath == "" {
		return SendMissingField(c, "path")
	}

	// Verify tenant exists
	_, err := h.tenantDB.GetTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, tenantdb.ErrTenantNotFound) {
			return SendResourceNotFound(c, "Tenant")
		}
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get tenant")
		return SendInternalError(c, "Failed to get tenant")
	}

	// Delete the setting
	if err := h.settingsSvc.DeleteTenantSetting(ctx, tenantID, settingPath); err != nil {
		if errors.Is(err, settings.ErrSettingNotFound) {
			return SendNotFound(c, "Setting not found")
		}
		log.Error().Err(err).Str("tenant_id", tenantID).Str("path", settingPath).Msg("Failed to delete tenant setting")
		return SendInternalError(c, "Failed to delete tenant setting")
	}

	log.Info().Str("tenant_id", tenantID).Str("path", settingPath).Msg("Deleted tenant setting")

	return c.SendStatus(fiber.StatusNoContent)
}

// GetTenantSetting returns a specific tenant setting with resolved value
// GET /admin/tenants/:id/settings/*path
func (h *TenantSettingsHandler) GetTenantSetting(c fiber.Ctx) error {
	ctx := c.Context()
	tenantID := c.Params("id")
	settingPath := c.Params("*")

	if tenantID == "" {
		return SendMissingField(c, "tenant_id")
	}

	if settingPath == "" {
		return SendMissingField(c, "path")
	}

	// Verify tenant exists
	tenant, err := h.tenantDB.GetTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, tenantdb.ErrTenantNotFound) {
			return SendResourceNotFound(c, "Tenant")
		}
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get tenant")
		return SendInternalError(c, "Failed to get tenant")
	}

	// Resolve the setting
	resolved, err := h.settingsSvc.ResolveSetting(ctx, tenantID, settingPath, tenant.IsDefault, tenant.Slug)
	if err != nil {
		if errors.Is(err, settings.ErrSettingNotFound) {
			return SendNotFound(c, "Setting not found")
		}
		log.Error().Err(err).Str("tenant_id", tenantID).Str("path", settingPath).Msg("Failed to resolve setting")
		return SendInternalError(c, "Failed to get setting")
	}

	return c.JSON(fiber.Map{
		"tenant_id":      tenantID,
		"path":           settingPath,
		"value":          resolved.Value,
		"source":         resolved.Source,
		"is_overridable": resolved.IsOverridable,
	})
}

// isPathOverridable checks if a path matches any overridable setting
func (h *TenantSettingsHandler) isPathOverridable(path string, overridableSettings []string) bool {
	if len(overridableSettings) == 0 {
		return true // All settings overridable if list is empty
	}

	for _, allowed := range overridableSettings {
		if path == allowed || strings.HasPrefix(path, allowed+".") {
			return true
		}
	}

	return false
}
