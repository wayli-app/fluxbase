package secrets

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

type Handler struct {
	storage *Storage
}

func NewHandler(storage *Storage) *Handler {
	return &Handler{
		storage: storage,
	}
}

type CreateSecretRequest struct {
	Name        string     `json:"name"`
	Value       string     `json:"value"`
	Scope       string     `json:"scope"`
	Namespace   *string    `json:"namespace,omitempty"`
	Description *string    `json:"description,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type UpdateSecretRequest struct {
	Value       *string    `json:"value,omitempty"`
	Description *string    `json:"description,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

func (h *Handler) CreateSecret(c fiber.Ctx) error {
	var req CreateSecretRequest
	if err := c.Bind().Body(&req); err != nil {
		return errors.SendInvalidBody(c)
	}

	if req.Name == "" {
		return errors.SendMissingField(c, "Name")
	}

	if req.Value == "" {
		return errors.SendMissingField(c, "Value")
	}

	if req.Scope == "" {
		req.Scope = "global"
	}

	if req.Scope != "global" && req.Scope != "namespace" {
		return errors.SendBadRequest(c, "Scope must be 'global' or 'namespace'", errors.ErrCodeInvalidInput)
	}

	if req.Scope == "namespace" && (req.Namespace == nil || *req.Namespace == "") {
		return errors.SendBadRequest(c, "Namespace is required when scope is 'namespace'", errors.ErrCodeMissingField)
	}

	if req.Scope == "global" {
		req.Namespace = nil
	}

	userID := getUserIDFromContext(c)

	secret := &Secret{
		Name:        req.Name,
		Scope:       req.Scope,
		Namespace:   req.Namespace,
		Description: req.Description,
		ExpiresAt:   req.ExpiresAt,
	}

	if err := h.storage.CreateSecret(middleware.CtxWithTenant(c), secret, req.Value, userID); err != nil {
		if isDuplicateKeyError(err) {
			return errors.SendConflict(c, "A secret with this name already exists in the specified scope", errors.ErrCodeDuplicateKey)
		}
		return errors.SendInternalError(c, "failed to create secret")
	}

	return c.Status(fiber.StatusCreated).JSON(secret)
}

func (h *Handler) ListSecrets(c fiber.Ctx) error {
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
		return errors.SendInternalError(c, "failed to list secrets")
	}

	if secrets == nil {
		secrets = []SecretSummary{}
	}

	return c.JSON(secrets)
}

func (h *Handler) GetSecret(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.SendInvalidID(c, "secret ID")
	}

	secret, err := h.storage.GetSecret(middleware.CtxWithTenant(c), id)
	if err != nil {
		if isNotFoundError(err) {
			return errors.SendResourceNotFound(c, "Secret")
		}
		return errors.SendInternalError(c, "failed to get secret")
	}

	return c.JSON(secret)
}

func (h *Handler) UpdateSecret(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.SendInvalidID(c, "secret ID")
	}

	var req UpdateSecretRequest
	if err := c.Bind().Body(&req); err != nil {
		return errors.SendInvalidBody(c)
	}

	if req.Value == nil && req.Description == nil && req.ExpiresAt == nil {
		return errors.SendBadRequest(c, "At least one field (value, description, or expires_at) must be provided", errors.ErrCodeMissingField)
	}

	userID := getUserIDFromContext(c)

	if err := h.storage.UpdateSecret(middleware.CtxWithTenant(c), id, req.Value, req.Description, req.ExpiresAt, userID); err != nil {
		if isNotFoundError(err) {
			return errors.SendResourceNotFound(c, "Secret")
		}
		return errors.SendInternalError(c, "failed to update secret")
	}

	secret, err := h.storage.GetSecret(middleware.CtxWithTenant(c), id)
	if err != nil {
		return errors.SendInternalError(c, "Secret updated but failed to retrieve updated data")
	}

	return c.JSON(secret)
}

func (h *Handler) DeleteSecret(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.SendInvalidID(c, "secret ID")
	}

	if err := h.storage.DeleteSecret(middleware.CtxWithTenant(c), id); err != nil {
		if isNotFoundError(err) {
			return errors.SendResourceNotFound(c, "Secret")
		}
		return errors.SendInternalError(c, "failed to delete secret")
	}

	return c.JSON(fiber.Map{
		"message": "Secret deleted successfully",
	})
}

func (h *Handler) GetVersions(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.SendInvalidID(c, "secret ID")
	}

	versions, err := h.storage.GetVersions(middleware.CtxWithTenant(c), id)
	if err != nil {
		return errors.SendInternalError(c, "failed to get versions")
	}

	if versions == nil {
		versions = []SecretVersion{}
	}

	return c.JSON(versions)
}

func (h *Handler) RollbackToVersion(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.SendInvalidID(c, "secret ID")
	}

	versionStr := c.Params("version")
	version := 0
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil || version < 1 {
		return errors.SendBadRequest(c, "Invalid version number", errors.ErrCodeInvalidInput)
	}

	userID := getUserIDFromContext(c)

	if err := h.storage.RollbackToVersion(middleware.CtxWithTenant(c), id, version, userID); err != nil {
		if isNotFoundError(err) {
			return errors.SendNotFound(c, fmt.Sprintf("Version %d not found", version))
		}
		return errors.SendInternalError(c, "failed to rollback secret")
	}

	secret, err := h.storage.GetSecret(middleware.CtxWithTenant(c), id)
	if err != nil {
		return errors.SendInternalError(c, "Rollback successful but failed to retrieve updated data")
	}

	return c.JSON(secret)
}

func (h *Handler) GetStats(c fiber.Ctx) error {
	total, expiringSoon, expired, err := h.storage.GetStats(middleware.CtxWithTenant(c))
	if err != nil {
		return errors.SendInternalError(c, "failed to get stats")
	}

	return c.JSON(fiber.Map{
		"total":         total,
		"expiring_soon": expiringSoon,
		"expired":       expired,
	})
}

func getNamespaceFromQuery(c fiber.Ctx) *string {
	if ns := c.Query("namespace"); ns != "" {
		if ns == "default" {
			ns = ""
		}
		return &ns
	}
	return nil
}

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

func (h *Handler) GetSecretByName(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return errors.SendMissingField(c, "Secret name")
	}

	namespace := getNamespaceFromQuery(c)

	secret, err := h.storage.GetSecretByName(middleware.CtxWithTenant(c), name, namespace)
	if err != nil {
		if isNotFoundError(err) {
			return errors.SendResourceNotFound(c, "Secret")
		}
		return errors.SendInternalError(c, "failed to get secret")
	}

	return c.JSON(secret)
}

func (h *Handler) UpdateSecretByName(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return errors.SendMissingField(c, "Secret name")
	}

	namespace := getNamespaceFromQuery(c)

	var req UpdateSecretRequest
	if err := c.Bind().Body(&req); err != nil {
		return errors.SendInvalidBody(c)
	}

	if req.Value == nil && req.Description == nil && req.ExpiresAt == nil {
		return errors.SendBadRequest(c, "At least one field (value, description, or expires_at) must be provided", errors.ErrCodeMissingField)
	}

	userID := getUserIDFromContext(c)

	secret, err := h.storage.GetSecretByName(middleware.CtxWithTenant(c), name, namespace)
	if err != nil {
		if isNotFoundError(err) {
			return errors.SendResourceNotFound(c, "Secret")
		}
		return errors.SendInternalError(c, "failed to get secret")
	}

	if err := h.storage.UpdateSecret(middleware.CtxWithTenant(c), secret.ID, req.Value, req.Description, req.ExpiresAt, userID); err != nil {
		return errors.SendInternalError(c, "failed to update secret")
	}

	updatedSecret, err := h.storage.GetSecret(middleware.CtxWithTenant(c), secret.ID)
	if err != nil {
		return errors.SendInternalError(c, "Secret updated but failed to retrieve updated data")
	}

	return c.JSON(updatedSecret)
}

func (h *Handler) DeleteSecretByName(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return errors.SendMissingField(c, "Secret name")
	}

	namespace := getNamespaceFromQuery(c)

	secret, err := h.storage.GetSecretByName(middleware.CtxWithTenant(c), name, namespace)
	if err != nil {
		if isNotFoundError(err) {
			return errors.SendResourceNotFound(c, "Secret")
		}
		return errors.SendInternalError(c, "failed to get secret")
	}

	if err := h.storage.DeleteSecret(middleware.CtxWithTenant(c), secret.ID); err != nil {
		return errors.SendInternalError(c, "failed to delete secret")
	}

	return c.JSON(fiber.Map{
		"message": "Secret deleted successfully",
	})
}

func (h *Handler) GetVersionsByName(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return errors.SendMissingField(c, "Secret name")
	}

	namespace := getNamespaceFromQuery(c)

	secret, err := h.storage.GetSecretByName(middleware.CtxWithTenant(c), name, namespace)
	if err != nil {
		if isNotFoundError(err) {
			return errors.SendResourceNotFound(c, "Secret")
		}
		return errors.SendInternalError(c, "failed to get secret")
	}

	versions, err := h.storage.GetVersions(middleware.CtxWithTenant(c), secret.ID)
	if err != nil {
		return errors.SendInternalError(c, "failed to get versions")
	}

	if versions == nil {
		versions = []SecretVersion{}
	}

	return c.JSON(versions)
}

func (h *Handler) RollbackByName(c fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return errors.SendMissingField(c, "Secret name")
	}

	versionStr := c.Params("version")
	version := 0
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil || version < 1 {
		return errors.SendBadRequest(c, "Invalid version number", errors.ErrCodeInvalidInput)
	}

	namespace := getNamespaceFromQuery(c)
	userID := getUserIDFromContext(c)

	secret, err := h.storage.GetSecretByName(middleware.CtxWithTenant(c), name, namespace)
	if err != nil {
		if isNotFoundError(err) {
			return errors.SendResourceNotFound(c, "Secret")
		}
		return errors.SendInternalError(c, "failed to get secret")
	}

	if err := h.storage.RollbackToVersion(middleware.CtxWithTenant(c), secret.ID, version, userID); err != nil {
		if isNotFoundError(err) {
			return errors.SendNotFound(c, fmt.Sprintf("Version %d not found", version))
		}
		return errors.SendInternalError(c, "failed to rollback secret")
	}

	updatedSecret, err := h.storage.GetSecret(middleware.CtxWithTenant(c), secret.ID)
	if err != nil {
		return errors.SendInternalError(c, "Rollback successful but failed to retrieve updated data")
	}

	return c.JSON(updatedSecret)
}

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
