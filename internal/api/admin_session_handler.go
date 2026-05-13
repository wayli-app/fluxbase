package api

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
)

// AdminSessionHandler handles admin session management
type AdminSessionHandler struct {
	sessionRepo *auth.SessionRepository
}

// NewAdminSessionHandler creates a new admin session handler
func NewAdminSessionHandler(sessionRepo *auth.SessionRepository) *AdminSessionHandler {
	return &AdminSessionHandler{
		sessionRepo: sessionRepo,
	}
}

func (h *AdminSessionHandler) requireService(c fiber.Ctx) error {
	if h.sessionRepo == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "Session service not initialized")
	}
	return nil
}

// ListSessions lists all active sessions with pagination
func (h *AdminSessionHandler) ListSessions(c fiber.Ctx) error {
	ctx := c.RequestCtx()

	// Check if we should include expired sessions
	includeExpired := c.Query("include_expired") == "true"

	// Parse pagination parameters
	limit := 25 // Default
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > 100 {
				limit = 100 // Max limit
			}
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	sessions, total, err := h.sessionRepo.ListAllPaginated(ctx, includeExpired, limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list sessions")
		return apperrors.SendInternalError(c, "Failed to list sessions")
	}

	return c.JSON(fiber.Map{
		"sessions":    sessions,
		"count":       len(sessions),
		"total_count": total,
		"limit":       limit,
		"offset":      offset,
	})
}

// RevokeSession revokes a specific session
func (h *AdminSessionHandler) RevokeSession(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	sessionID := c.Params("id")

	if sessionID == "" {
		return apperrors.SendMissingField(c, "session_id")
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	err := h.sessionRepo.Delete(ctx, sessionID)
	if err != nil {
		if errors.Is(err, auth.ErrSessionNotFound) {
			return apperrors.SendResourceNotFound(c, "Session")
		}
		log.Error().Err(err).Str("session_id", sessionID).Msg("Failed to revoke session")
		return apperrors.SendInternalError(c, "Failed to revoke session")
	}

	return apperrors.SendSuccess(c, "Session revoked successfully")
}

// RevokeUserSessions revokes all sessions for a specific user
func (h *AdminSessionHandler) RevokeUserSessions(c fiber.Ctx) error {
	ctx := c.RequestCtx()
	userID := c.Params("user_id")

	if userID == "" {
		return apperrors.SendMissingField(c, "user_id")
	}

	if err := h.requireService(c); err != nil {
		return err
	}

	err := h.sessionRepo.DeleteByUserID(ctx, userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to revoke user sessions")
		return apperrors.SendInternalError(c, "Failed to revoke user sessions")
	}

	return apperrors.SendSuccess(c, "All user sessions revoked successfully")
}

// fiber:context-methods migrated
