// Package integrations provides storage and configuration for non-LLM
// tool integrations (web search, URL fetch, etc.) consumed by chatbot
// specialist agents.
//
// Handler.go exposes the REST API surface. Routes mirror the AI provider
// CRUD: list/get/create/update/delete + set-default + test.
package integrations

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// Handler exposes the integrations REST API. Mirrors the shape of
// internal/ai/handler.go's provider CRUD: stateless handlers that read
// tenant from request context and delegate to Storage.
type Handler struct {
	storage *Storage
}

// NewHandler constructs a Handler bound to the given storage.
func NewHandler(storage *Storage) *Handler {
	return &Handler{storage: storage}
}

// Storage returns the handler's underlying storage. Exposed so the
// supervisor / Web Agent can resolve integrations from the same instance
// the API uses.
func (h *Handler) Storage() *Storage { return h.storage }

// ListIntegrations handles GET /api/v1/admin/ai/integrations.
// Optional query param `?type=web_search` filters by integration_type.
func (h *Handler) ListIntegrations(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	var integrationType *IntegrationType
	if t := c.Query("type"); t != "" {
		if !IsValidIntegrationType(t) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid integration_type: " + t,
			})
		}
		it := IntegrationType(t)
		integrationType = &it
	}

	integrations, err := h.storage.ListIntegrations(ctx, integrationType)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list integrations")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list integrations",
		})
	}

	// Mask secrets in responses
	out := make([]map[string]any, 0, len(integrations))
	for _, i := range integrations {
		out = append(out, integrationToAPIResponse(i))
	}
	return c.JSON(fiber.Map{
		"integrations": out,
		"count":        len(out),
	})
}

// GetIntegration handles GET /api/v1/admin/ai/integrations/:id.
func (h *Handler) GetIntegration(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id is required"})
	}

	i, err := h.storage.GetIntegration(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("id", id).Msg("Failed to get integration")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get integration",
		})
	}
	if i == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Integration not found",
		})
	}
	return c.JSON(integrationToAPIResponse(i))
}

// CreateIntegration handles POST /api/v1/admin/ai/integrations.
func (h *Handler) CreateIntegration(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	tenantID := database.TenantFromContext(ctx)

	var req CreateIntegrationRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	created, err := h.storage.CreateIntegration(ctx, tenantID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusCreated).JSON(integrationToAPIResponse(created))
}

// UpdateIntegration handles PUT /api/v1/admin/ai/integrations/:id.
func (h *Handler) UpdateIntegration(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	tenantID := database.TenantFromContext(ctx)
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id is required"})
	}

	var req UpdateIntegrationRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	updated, err := h.storage.UpdateIntegration(ctx, tenantID, id, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(integrationToAPIResponse(updated))
}

// DeleteIntegration handles DELETE /api/v1/admin/ai/integrations/:id.
func (h *Handler) DeleteIntegration(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	tenantID := database.TenantFromContext(ctx)
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id is required"})
	}

	if err := h.storage.DeleteIntegration(ctx, tenantID, id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// SetDefaultIntegration handles PUT /api/v1/admin/ai/integrations/:id/default.
// Marks the integration as the default for its integration_type within
// the current tenant. The storage layer clears any prior default inside
// the same UPDATE transaction.
func (h *Handler) SetDefaultIntegration(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	tenantID := database.TenantFromContext(ctx)
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id is required"})
	}

	trueVal := true
	_, err := h.storage.UpdateIntegration(ctx, tenantID, id, &UpdateIntegrationRequest{
		IsDefault: &trueVal,
	})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// TestIntegration handles POST /api/v1/admin/ai/integrations/:id/test.
// Runs a "hello world" call against the integration's provider and stores
// the result. Used by the admin UI's Test button to surface health info.
func (h *Handler) TestIntegration(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	tenantID := database.TenantFromContext(ctx)
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id is required"})
	}

	i, err := h.storage.GetIntegration(ctx, id)
	if err != nil || i == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Integration not found",
		})
	}

	status, errMsg := testIntegration(ctx, i)
	_ = h.storage.UpdateTestStatus(ctx, tenantID, id, status, errMsg)

	resp := fiber.Map{
		"status":         status,
		"last_tested_at": time.Now().UTC().Format(time.RFC3339),
	}
	if errMsg != "" {
		resp["error"] = errMsg
	}
	return c.JSON(resp)
}

// testIntegration runs a real provider call to verify credentials.
// Returns ("ok", "") on success or ("failed", "error message") on failure.
// Times out after 30 seconds via the request context.
func testIntegration(ctx context.Context, i *Integration) (status, errMsg string) {
	switch i.Provider {
	case ProviderTavily:
		client := NewTavilyClient(i.Config["api_key"], i.Config["base_url"], nil)
		if err := client.Ping(ctx); err != nil {
			return "failed", err.Error()
		}
		return "ok", ""
	case ProviderBrave, ProviderJina, ProviderSingleFetch:
		// Not implemented yet; surface a clear message rather than failing
		// silently. Schema allows these providers so future PRs can add
		// implementations without a migration.
		return "failed", "Provider " + string(i.Provider) + " is not yet implemented"
	default:
		return "failed", "Unknown provider: " + string(i.Provider)
	}
}

// integrationToAPIResponse converts an Integration to the JSON shape
// returned by the API. Secrets in Config are masked to "***masked***";
// the read_only + from_config flags surface whether the row came from
// env/YAML config.
func integrationToAPIResponse(i *Integration) map[string]any {
	if i == nil {
		return nil
	}
	// Clone config and mask secrets
	configOut := map[string]string{}
	for k, v := range i.Config {
		if isSecretField(k) {
			configOut[k] = MaskSecret(v)
		} else {
			configOut[k] = v
		}
	}

	resp := map[string]any{
		"id":               i.ID,
		"name":             i.Name,
		"integration_type": string(i.IntegrationType),
		"provider":         string(i.Provider),
		"config":           configOut,
		"enabled":          i.Enabled,
		"is_default":       i.IsDefault,
		"created_at":       i.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":       i.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if i.FromConfig {
		resp["from_config"] = true
	}
	if i.ReadOnly {
		resp["read_only"] = true
	}
	if i.LastTestedAt != nil {
		resp["last_tested_at"] = i.LastTestedAt.UTC().Format(time.RFC3339)
	}
	if i.LastTestStatus != "" {
		resp["last_test_status"] = i.LastTestStatus
	}
	if i.LastTestError != "" {
		resp["last_test_error"] = i.LastTestError
	}
	if i.CreatedBy != "" {
		resp["created_by"] = i.CreatedBy
	}
	return resp
}

// isSecretField reports whether the given config key holds a sensitive
// value that should be masked in API responses. Currently just api_key;
// extend when providers add additional secret fields.
func isSecretField(k string) bool {
	return k == "api_key"
}
