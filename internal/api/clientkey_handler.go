package api

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/nimbleflux/fluxbase/internal/auth"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

type ClientKeyHandler struct {
	clientKeyService *auth.ClientKeyService
}

func NewClientKeyHandler(clientKeyService *auth.ClientKeyService) *ClientKeyHandler {
	return &ClientKeyHandler{
		clientKeyService: clientKeyService,
	}
}

type CreateClientKeyRequest struct {
	Name               string     `json:"name"`
	Description        *string    `json:"description,omitempty"`
	Scopes             []string   `json:"scopes"`
	RateLimitPerMinute int        `json:"rate_limit_per_minute"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
}

type UpdateClientKeyRequest struct {
	Name               *string  `json:"name,omitempty"`
	Description        *string  `json:"description,omitempty"`
	Scopes             []string `json:"scopes,omitempty"`
	RateLimitPerMinute *int     `json:"rate_limit_per_minute,omitempty"`
}

func (h *ClientKeyHandler) requireService(c fiber.Ctx) error {
	if h.clientKeyService == nil {
		return SendNotInitialized(c, "Client key service")
	}
	return nil
}

func (h *ClientKeyHandler) CreateClientKey(c fiber.Ctx) error {
	var req CreateClientKeyRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Name == "" {
		return SendMissingField(c, "Name")
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	var userIDPtr *uuid.UUID

	clientKey, err := h.clientKeyService.GenerateClientKey(
		c.RequestCtx(),
		req.Name,
		req.Description,
		userIDPtr,
		req.Scopes,
		req.RateLimitPerMinute,
		req.ExpiresAt,
	)
	if err != nil {
		return SendInternalError(c, "Failed to create client key")
	}

	return c.Status(fiber.StatusCreated).JSON(clientKey)
}

func (h *ClientKeyHandler) ListClientKeys(c fiber.Ctx) error {
	currentUserID := middleware.GetUserID(c)
	role, _ := c.Locals("user_role").(string)
	isAdmin := role == "admin" || role == "instance_admin" || role == "service_role" || role == "tenant_service"

	var userID *uuid.UUID

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		id, err := uuid.Parse(userIDStr)
		if err != nil {
			return SendInvalidID(c, "user ID")
		}

		if !isAdmin && userIDStr != currentUserID {
			return SendForbidden(c, "Cannot view other users' client keys", ErrCodeAccessDenied)
		}
		userID = &id
	} else if !isAdmin && currentUserID != "" {
		id, err := uuid.Parse(currentUserID)
		if err == nil {
			userID = &id
		}
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	clientKeys, err := h.clientKeyService.ListClientKeys(c.RequestCtx(), userID)
	if err != nil {
		return SendInternalError(c, "Failed to list client keys")
	}

	return c.JSON(clientKeys)
}

func (h *ClientKeyHandler) GetClientKey(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendInvalidID(c, "client key ID")
	}

	currentUserID := middleware.GetUserID(c)
	role, _ := c.Locals("user_role").(string)
	isAdmin := role == "admin" || role == "instance_admin" || role == "service_role" || role == "tenant_service"

	if err := h.requireService(c); err != nil {
		return err
	}

	clientKeys, err := h.clientKeyService.ListClientKeys(c.RequestCtx(), nil)
	if err != nil {
		return SendInternalError(c, "Failed to get client key")
	}

	for _, key := range clientKeys {
		if key.ID == id {
			if !isAdmin && key.UserID != nil && key.UserID.String() != currentUserID {
				return SendForbidden(c, "Cannot view other users' client keys", ErrCodeAccessDenied)
			}
			return c.JSON(key)
		}
	}

	return SendNotFound(c, "Client key not found")
}

func (h *ClientKeyHandler) UpdateClientKey(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendInvalidID(c, "client key ID")
	}

	var req UpdateClientKeyRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	err = h.clientKeyService.UpdateClientKey(c.RequestCtx(), id, req.Name, req.Description, req.Scopes, req.RateLimitPerMinute)
	if err != nil {
		return SendInternalError(c, "Failed to update client key")
	}

	return apperrors.SendSuccess(c, "Client key updated successfully")
}

func (h *ClientKeyHandler) RevokeClientKey(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendInvalidID(c, "client key ID")
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	err = h.clientKeyService.RevokeClientKey(c.RequestCtx(), id)
	if err != nil {
		return SendInternalError(c, "Failed to revoke client key")
	}

	return apperrors.SendSuccess(c, "Client key revoked successfully")
}

func (h *ClientKeyHandler) DeleteClientKey(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return SendInvalidID(c, "client key ID")
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	err = h.clientKeyService.DeleteClientKey(c.RequestCtx(), id)
	if err != nil {
		return SendInternalError(c, "Failed to delete client key")
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// fiber:context-methods migrated
