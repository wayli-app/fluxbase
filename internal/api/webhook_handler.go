package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/wayli-app/fluxbase/internal/webhook"
)

// WebhookHandler handles HTTP requests for webhooks
type WebhookHandler struct {
	webhookService *webhook.WebhookService
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(webhookService *webhook.WebhookService) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
	}
}

// RegisterRoutes registers webhook routes
func (h *WebhookHandler) RegisterRoutes(app *fiber.App) {
	webhooks := app.Group("/api/v1/webhooks")
	webhooks.Post("/", h.CreateWebhook)
	webhooks.Get("/", h.ListWebhooks)
	webhooks.Get("/:id", h.GetWebhook)
	webhooks.Patch("/:id", h.UpdateWebhook)
	webhooks.Delete("/:id", h.DeleteWebhook)
	webhooks.Post("/:id/test", h.TestWebhook)
	webhooks.Get("/:id/deliveries", h.ListDeliveries)
}

// CreateWebhook creates a new webhook
func (h *WebhookHandler) CreateWebhook(c *fiber.Ctx) error {
	var req webhook.Webhook
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validation
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}
	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "URL is required",
		})
	}

	// Set defaults
	if req.MaxRetries == 0 {
		req.MaxRetries = 3
	}
	if req.RetryBackoffSeconds == 0 {
		req.RetryBackoffSeconds = 5
	}
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 30
	}
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}

	err := h.webhookService.Create(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(req)
}

// ListWebhooks lists all webhooks
func (h *WebhookHandler) ListWebhooks(c *fiber.Ctx) error {
	webhooks, err := h.webhookService.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(webhooks)
}

// GetWebhook retrieves a webhook by ID
func (h *WebhookHandler) GetWebhook(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	wh, err := h.webhookService.Get(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Webhook not found",
		})
	}

	return c.JSON(wh)
}

// UpdateWebhook updates a webhook
func (h *WebhookHandler) UpdateWebhook(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	var req webhook.Webhook
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	err = h.webhookService.Update(c.Context(), id, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Webhook updated successfully",
	})
}

// DeleteWebhook deletes a webhook
func (h *WebhookHandler) DeleteWebhook(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	err = h.webhookService.Delete(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Webhook deleted successfully",
	})
}

// TestWebhook sends a test webhook
func (h *WebhookHandler) TestWebhook(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	wh, err := h.webhookService.Get(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Webhook not found",
		})
	}

	// Create test payload
	testPayload := &webhook.WebhookPayload{
		Event:     "TEST",
		Table:     "test",
		Schema:    "public",
		Record:    []byte(`{"test": true}`),
		Timestamp: c.Context().Time(),
	}

	err = h.webhookService.Deliver(c.Context(), wh, testPayload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Test webhook sent successfully",
	})
}

// ListDeliveries lists webhook deliveries
func (h *WebhookHandler) ListDeliveries(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook ID",
		})
	}

	// Default limit is 50
	limit := 50
	if limitParam := c.Query("limit"); limitParam != "" {
		parsedLimit := c.QueryInt("limit", 50)
		limit = parsedLimit
	}

	deliveries, err := h.webhookService.ListDeliveries(c.Context(), id, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(deliveries)
}
