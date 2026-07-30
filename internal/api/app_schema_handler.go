package api

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/migrations"
)

// AppSchemaHandler handles declarative app-schema management endpoints.
//
// These endpoints let an application developer manage their own tables (e.g. the
// `public` schema) declaratively: sync schema content, plan/preview changes, apply,
// and validate drift. This is the API counterpart to `fluxbase schema sync` and the
// opt-in alternative to imperative user migrations.
//
// Mounted under /api/v1/admin/app-schema/* (instance_admin only).
type AppSchemaHandler struct {
	service *migrations.AppDeclarativeService
	enabled bool
}

// NewAppSchemaHandler creates a new app-schema handler.
func NewAppSchemaHandler() *AppSchemaHandler {
	return &AppSchemaHandler{}
}

// Initialize wires the handler with a DeclarativeAppSchemaConfig. The service is
// always constructed so the API can sync/plan/apply on demand even when startup
// auto-apply is off; `enabled` reflects whether the feature is configured on.
func (h *AppSchemaHandler) Initialize(cfg *config.Config, db *database.Connection) {
	appCfg := cfg.Database.DeclarativeAppSchema
	if appCfg == nil {
		appCfg = &config.DeclarativeAppSchemaConfig{}
	}

	adminUser := cfg.Database.AdminUser
	if adminUser == "" {
		adminUser = cfg.Database.User
	}
	adminPassword := cfg.Database.AdminPassword
	if adminPassword == "" {
		adminPassword = cfg.Database.Password
	}

	svc := migrations.NewAppDeclarativeService(
		"pgschema",
		cfg.Database.Host,
		cfg.Database.Port,
		adminUser,
		adminPassword,
		cfg.Database.Database,
		appCfg.AllowDestructive,
	)
	svc.SetPool(db.Pool())
	svc.SetAppUser(cfg.Database.User)

	h.service = svc
	h.enabled = appCfg.Enabled

	log.Info().
		Bool("enabled", h.enabled).
		Str("schema", appCfg.Schema).
		Msg("App schema handler initialized")
}

// SyncSchema handles POST /api/v1/admin/app-schema/sync
// Stores schema content for a (namespace, schema) and optionally applies it.
func (h *AppSchemaHandler) SyncSchema(c fiber.Ctx) error {
	if h.service == nil {
		return SendInternalError(c, "App schema handler not initialized")
	}

	var req struct {
		Namespace string `json:"namespace"`
		Schema    string `json:"schema"`         // optional, defaults to "public"
		Content   string `json:"content"`        // the SQL schema body
		Ignore    string `json:"ignore_content"` // optional .pgschemaignore content
		Apply     bool   `json:"apply"`          // store + apply immediately
		NoApply   bool   `json:"no_apply"`       // explicit store-only
	}
	if err := c.Bind().Body(&req); err != nil && err != fiber.ErrUnprocessableEntity {
		return SendBadRequest(c, "Invalid request body", ErrCodeInvalidBody)
	}
	if req.Namespace == "" {
		return SendBadRequest(c, "namespace is required", ErrCodeInvalidBody)
	}
	if req.Content == "" {
		return SendBadRequest(c, "content is required", ErrCodeInvalidBody)
	}

	ctx := c.Context()

	fingerprint, changed, err := h.service.StoreSchemaContent(ctx, req.Namespace, req.Schema, req.Content, req.Ignore)
	if err != nil {
		return SendInternalError(c, fmt.Sprintf("Failed to store schema: %v", err))
	}

	resp := fiber.Map{
		"message":     "Schema stored successfully",
		"namespace":   req.Namespace,
		"schema":      effectiveSchema(req.Schema),
		"fingerprint": fingerprint,
		"changed":     changed,
		"applied":     false,
	}

	// Apply unless explicitly store-only.
	if !req.NoApply && (req.Apply || changed) {
		res, err := h.service.ApplyStored(ctx, req.Namespace, req.Schema)
		if err != nil {
			return SendInternalError(c, fmt.Sprintf("Failed to apply schema: %v", err))
		}
		resp["applied"] = true
		resp["changes"] = len(res.Applied)
		resp["duration"] = res.Duration.String()
		resp["fallback"] = res.Fallback
		if res.Error != nil {
			resp["message"] = res.Error.Error()
		}
	}

	return c.JSON(resp)
}

// ApplySchema handles POST /api/v1/admin/app-schema/apply
// Applies the already-stored schema content for a (namespace, schema).
func (h *AppSchemaHandler) ApplySchema(c fiber.Ctx) error {
	if h.service == nil {
		return SendInternalError(c, "App schema handler not initialized")
	}

	var req struct {
		Namespace string `json:"namespace"`
		Schema    string `json:"schema"`
	}
	if err := c.Bind().Body(&req); err != nil && err != fiber.ErrUnprocessableEntity {
		return SendBadRequest(c, "Invalid request body", ErrCodeInvalidBody)
	}
	if req.Namespace == "" {
		return SendBadRequest(c, "namespace is required", ErrCodeInvalidBody)
	}

	res, err := h.service.ApplyStored(c.Context(), req.Namespace, req.Schema)
	if err != nil {
		return SendInternalError(c, fmt.Sprintf("Failed to apply schema: %v", err))
	}

	resp := fiber.Map{
		"message":  "Schema applied successfully",
		"applied":  len(res.Applied),
		"duration": res.Duration.String(),
		"fallback": res.Fallback,
	}
	if res.Error != nil {
		resp["message"] = res.Error.Error()
	}
	return c.JSON(resp)
}

// PlanSchema handles POST /api/v1/admin/app-schema/plan
// Previews pending changes between stored content and the live database.
func (h *AppSchemaHandler) PlanSchema(c fiber.Ctx) error {
	if h.service == nil {
		return SendInternalError(c, "App schema handler not initialized")
	}

	var req struct {
		Namespace string `json:"namespace"`
		Schema    string `json:"schema"`
	}
	if err := c.Bind().Body(&req); err != nil && err != fiber.ErrUnprocessableEntity {
		return SendBadRequest(c, "Invalid request body", ErrCodeInvalidBody)
	}
	if req.Namespace == "" {
		return SendBadRequest(c, "namespace is required", ErrCodeInvalidBody)
	}

	plan, err := h.service.Plan(c.Context(), req.Namespace, req.Schema)
	if err != nil {
		return SendInternalError(c, fmt.Sprintf("Failed to plan schema: %v", err))
	}

	return c.JSON(fiber.Map{
		"plan": fiber.Map{
			"changes":  plan.Changes,
			"ddl":      plan.DDL,
			"duration": plan.Duration.String(),
			"summary": fiber.Map{
				"total_changes":     len(plan.Changes),
				"create_count":      countByType(plan.Changes, migrations.ChangeCreate),
				"alter_count":       countByType(plan.Changes, migrations.ChangeAlter),
				"drop_count":        countByType(plan.Changes, migrations.ChangeDrop),
				"destructive_count": countDestructive(plan.Changes),
			},
		},
	})
}

// GetStatus handles GET /api/v1/admin/app-schema/status
// Reports status for a single (namespace, schema) or lists all stored app schemas.
func (h *AppSchemaHandler) GetStatus(c fiber.Ctx) error {
	if h.service == nil {
		return SendInternalError(c, "App schema handler not initialized")
	}

	ctx := c.Context()
	ns := c.Query("namespace")
	schema := c.Query("schema")

	if ns != "" {
		status, err := h.service.GetStatus(ctx, ns, schema)
		if err != nil {
			return SendInternalError(c, fmt.Sprintf("Failed to get status: %v", err))
		}
		return c.JSON(fiber.Map{"status": status, "enabled": h.enabled})
	}

	records, err := h.service.ListStoredSchemas(ctx, "")
	if err != nil {
		return SendInternalError(c, fmt.Sprintf("Failed to list schemas: %v", err))
	}
	return c.JSON(fiber.Map{"schemas": records, "enabled": h.enabled})
}

// ValidateSchema handles GET /api/v1/admin/app-schema/validate
// Checks for drift between stored content and the live database.
func (h *AppSchemaHandler) ValidateSchema(c fiber.Ctx) error {
	if h.service == nil {
		return SendInternalError(c, "App schema handler not initialized")
	}

	ns := c.Query("namespace")
	schema := c.Query("schema")
	if ns == "" {
		return SendBadRequest(c, "namespace query parameter is required", ErrCodeInvalidBody)
	}

	plan, err := h.service.Plan(c.Context(), ns, schema)
	if err != nil {
		return SendInternalError(c, fmt.Sprintf("Failed to validate schema: %v", err))
	}

	return c.JSON(fiber.Map{
		"valid":  len(plan.Changes) == 0,
		"drifts": plan.Changes,
		"summary": fiber.Map{
			"total_changes":     len(plan.Changes),
			"destructive_count": countDestructive(plan.Changes),
		},
	})
}

// DeleteSchema handles DELETE /api/v1/admin/app-schema
// Removes stored schema content (does not drop database objects).
func (h *AppSchemaHandler) DeleteSchema(c fiber.Ctx) error {
	if h.service == nil {
		return SendInternalError(c, "App schema handler not initialized")
	}

	ns := c.Query("namespace")
	schema := c.Query("schema")
	if ns == "" {
		return SendBadRequest(c, "namespace query parameter is required", ErrCodeInvalidBody)
	}

	if err := h.service.DeleteStoredSchema(c.Context(), ns, schema); err != nil {
		return SendInternalError(c, fmt.Sprintf("Failed to delete schema: %v", err))
	}
	return c.JSON(fiber.Map{
		"message":   "Schema content removed (database objects left intact)",
		"namespace": ns,
		"schema":    effectiveSchema(schema),
	})
}

// effectiveSchema normalizes an empty schema name to the default "public".
func effectiveSchema(s string) string {
	if s == "" {
		return "public"
	}
	return s
}

// Ensure Interface compliance
var _ fmt.Stringer = (*AppSchemaHandler)(nil)

func (h *AppSchemaHandler) String() string { return "AppSchemaHandler" }
