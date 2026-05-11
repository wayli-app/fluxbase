package api

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/webhook"
)

// WebhookResponse represents a webhook response without the secret
// H-21: WebhookResponse DTO excludes secret field for security
type WebhookResponse struct {
	ID                  uuid.UUID             `json:"id"`
	Name                string                `json:"name"`
	Description         *string               `json:"description,omitempty"`
	URL                 string                `json:"url"`
	Enabled             bool                  `json:"enabled"`
	Events              []webhook.EventConfig `json:"events"`
	MaxRetries          int                   `json:"max_retries"`
	RetryBackoffSeconds int                   `json:"retry_backoff_seconds"`
	TimeoutSeconds      int                   `json:"timeout_seconds"`
	Headers             map[string]string     `json:"headers"`
	Scope               string                `json:"scope"` // "user" or "global"
	CreatedBy           *uuid.UUID            `json:"created_by,omitempty"`
	CreatedAt           time.Time             `json:"created_at"`
	UpdatedAt           time.Time             `json:"updated_at"`
}

// toWebhookResponse converts a webhook.Webhook to WebhookResponse (without secret)
func toWebhookResponse(w webhook.Webhook) WebhookResponse {
	return WebhookResponse{
		ID:                  w.ID,
		Name:                w.Name,
		Description:         w.Description,
		URL:                 w.URL,
		Enabled:             w.Enabled,
		Events:              w.Events,
		MaxRetries:          w.MaxRetries,
		RetryBackoffSeconds: w.RetryBackoffSeconds,
		TimeoutSeconds:      w.TimeoutSeconds,
		Headers:             w.Headers,
		Scope:               w.Scope,
		CreatedBy:           w.CreatedBy,
		CreatedAt:           w.CreatedAt,
		UpdatedAt:           w.UpdatedAt,
	}
}

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

func (h *WebhookHandler) requireService(c fiber.Ctx) error {
	if h.webhookService == nil {
		return SendNotInitialized(c, "Webhook service")
	}
	return nil
}

// CreateWebhook creates a new webhook
func (h *WebhookHandler) CreateWebhook(c fiber.Ctx) error {
	var req webhook.Webhook
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	// Validation
	if req.Name == "" {
		return SendMissingField(c, "Name")
	}
	if req.URL == "" {
		return SendMissingField(c, "URL")
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
	if req.Scope == "" {
		req.Scope = "user"
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	// Set CreatedBy from authenticated user
	if uid := middleware.GetUserID(c); uid != "" {
		if parsed, err := uuid.Parse(uid); err == nil {
			req.CreatedBy = &parsed
		}
	}

	err := h.webhookService.Create(middleware.CtxWithTenant(c), &req)
	if err != nil {
		return SendInternalError(c, "Failed to create webhook")
	}

	// H-21: Return WebhookResponse (without secret)
	return c.Status(fiber.StatusCreated).JSON(toWebhookResponse(req))
}

// ListWebhooks lists all webhooks
func (h *WebhookHandler) ListWebhooks(c fiber.Ctx) error {
	if err := h.requireService(c); err != nil {
		return err
	}

	webhooks, err := h.webhookService.List(middleware.CtxWithTenant(c))
	if err != nil {
		return SendInternalError(c, "Failed to list webhooks")
	}

	// H-21: Convert to WebhookResponse (without secret)
	responses := make([]WebhookResponse, len(webhooks))
	for i, wh := range webhooks {
		responses[i] = toWebhookResponse(*wh)
	}

	return c.JSON(responses)
}

// GetWebhook retrieves a webhook by ID
func (h *WebhookHandler) GetWebhook(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return SendBadRequest(c, "Invalid webhook ID", ErrCodeInvalidID)
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	wh, err := h.webhookService.Get(middleware.CtxWithTenant(c), id)
	if err != nil {
		return SendResourceNotFound(c, "Webhook")
	}

	// H-21: Return WebhookResponse (without secret)
	return c.JSON(toWebhookResponse(*wh))
}

// UpdateWebhook updates a webhook
func (h *WebhookHandler) UpdateWebhook(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return SendBadRequest(c, "Invalid webhook ID", ErrCodeInvalidID)
	}

	var req webhook.Webhook
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	err = h.webhookService.Update(middleware.CtxWithTenant(c), id, &req)
	if err != nil {
		return SendInternalError(c, "Failed to update webhook")
	}

	return apperrors.SendSuccess(c, "Webhook updated successfully")
}

// DeleteWebhook deletes a webhook
func (h *WebhookHandler) DeleteWebhook(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return SendBadRequest(c, "Invalid webhook ID", ErrCodeInvalidID)
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	err = h.webhookService.Delete(middleware.CtxWithTenant(c), id)
	if err != nil {
		return SendInternalError(c, "Failed to delete webhook")
	}

	return apperrors.SendSuccess(c, "Webhook deleted successfully")
}

// TestWebhook sends a test webhook
func (h *WebhookHandler) TestWebhook(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return SendBadRequest(c, "Invalid webhook ID", ErrCodeInvalidID)
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	wh, err := h.webhookService.Get(middleware.CtxWithTenant(c), id)
	if err != nil {
		return SendResourceNotFound(c, "Webhook")
	}

	// Create test payload
	testPayload := &webhook.WebhookPayload{
		Event:     "TEST",
		Table:     "test",
		Schema:    "public",
		Record:    []byte(`{"test": true}`),
		Timestamp: c.RequestCtx().Time(),
	}

	err = h.webhookService.Deliver(middleware.CtxWithTenant(c), wh, testPayload)
	if err != nil {
		return SendInternalError(c, "Failed to send test webhook")
	}

	return apperrors.SendSuccess(c, "Test webhook sent successfully")
}

// ListDeliveries lists webhook deliveries
func (h *WebhookHandler) ListDeliveries(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return SendBadRequest(c, "Invalid webhook ID", ErrCodeInvalidID)
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	// Default limit is 50
	limit := 50
	if limitParam := c.Query("limit"); limitParam != "" {
		parsedLimit := fiber.Query[int](c, "limit", 50)
		limit = parsedLimit
	}

	deliveries, err := h.webhookService.ListDeliveries(middleware.CtxWithTenant(c), id, limit)
	if err != nil {
		return SendInternalError(c, "Failed to list deliveries")
	}

	return c.JSON(deliveries)
}

// fiber:context-methods migrated
