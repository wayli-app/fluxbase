package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// GetUserIdentities gets all OAuth identities linked to a user
// GET /auth/user/identities
func (h *AuthHandler) GetUserIdentities(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	identities, err := h.authService.GetUserIdentities(middleware.CtxWithTenant(c), userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to get user identities")
		return SendInternalError(c, "Failed to retrieve identities")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"identities": identities,
	})
}

// LinkIdentity initiates OAuth flow to link a provider
// POST /auth/user/identities
func (h *AuthHandler) LinkIdentity(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	var req struct {
		Provider string `json:"provider"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Provider == "" {
		return SendMissingField(c, "Provider")
	}

	authURL, state, err := h.authService.LinkIdentity(middleware.CtxWithTenant(c), userID, req.Provider)
	if err != nil {
		log.Error().Err(err).Str("provider", req.Provider).Msg("Failed to initiate identity linking")
		return SendBadRequest(c, "Failed to link identity", ErrCodeInvalidInput)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"url":      authURL,
		"provider": req.Provider,
		"state":    state,
	})
}

// UnlinkIdentity removes an OAuth identity from a user
// DELETE /auth/user/identities/:id
func (h *AuthHandler) UnlinkIdentity(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	identityID := c.Params("id")
	if identityID == "" {
		return SendMissingField(c, "Identity ID")
	}

	err := h.authService.UnlinkIdentity(middleware.CtxWithTenant(c), userID, identityID)
	if err != nil {
		log.Error().Err(err).Str("identity_id", identityID).Msg("Failed to unlink identity")
		return SendBadRequest(c, "Failed to unlink identity", ErrCodeInvalidInput)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
	})
}

// Reauthenticate generates a security nonce
// POST /auth/reauthenticate
func (h *AuthHandler) Reauthenticate(c fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == "" {
		return SendMissingAuth(c)
	}

	nonce, err := h.authService.Reauthenticate(middleware.CtxWithTenant(c), userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to reauthenticate")
		return SendInternalError(c, "Failed to generate security nonce")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"nonce": nonce,
	})
}

// SignInWithIDToken handles OAuth ID token authentication (Google, Apple)
// POST /auth/signin/idtoken
func (h *AuthHandler) SignInWithIDToken(c fiber.Ctx) error {
	var req struct {
		Provider string  `json:"provider"`
		Token    string  `json:"token"`
		Nonce    *string `json:"nonce,omitempty"`
	}

	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if req.Provider == "" || req.Token == "" {
		return SendBadRequest(c, "Provider and token are required", ErrCodeMissingField)
	}

	nonce := ""
	if req.Nonce != nil {
		nonce = *req.Nonce
	}

	resp, err := h.authService.SignInWithIDToken(middleware.CtxWithTenant(c), req.Provider, req.Token, nonce)
	if err != nil {
		log.Error().Err(err).Str("provider", req.Provider).Msg("Failed to sign in with ID token")
		return SendBadRequest(c, "Invalid ID token", ErrCodeInvalidCredentials)
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}
