package api

import (
	"strconv"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
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

// ListSessions lists all active sessions with pagination
func (h *AdminSessionHandler) ListSessions(c *fiber.Ctx) error {
	ctx := c.Context()

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

	sessions, total, err := h.sessionRepo.ListAllPaginated(ctx, includeExpired, limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list sessions")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list sessions",
		})
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
func (h *AdminSessionHandler) RevokeSession(c *fiber.Ctx) error {
	ctx := c.Context()
	sessionID := c.Params("id")

	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID is required",
		})
	}

	err := h.sessionRepo.Delete(ctx, sessionID)
	if err != nil {
		if err == auth.ErrSessionNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Session not found",
			})
		}
		log.Error().Err(err).Str("session_id", sessionID).Msg("Failed to revoke session")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to revoke session",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Session revoked successfully",
	})
}

// RevokeUserSessions revokes all sessions for a specific user
func (h *AdminSessionHandler) RevokeUserSessions(c *fiber.Ctx) error {
	ctx := c.Context()
	userID := c.Params("user_id")

	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	err := h.sessionRepo.DeleteByUserID(ctx, userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to revoke user sessions")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to revoke user sessions",
		})
	}

	return c.JSON(fiber.Map{
		"message": "All user sessions revoked successfully",
	})
}
