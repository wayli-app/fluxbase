package ai

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"
)

// GetUserUsage returns the authenticated user's daily quota snapshot for a
// given chatbot: requests used/limit and tokens used/limit, plus when the
// counters reset. Cheap O(1) read against the in-memory limiter.
//
// GET /api/v1/ai/usage/:chatbotId
func (h *ChatHandler) GetUserUsage(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)
	chatbotID := c.Params("chatbotId")

	userID := middleware.GetUserID(c)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	chatbot, err := h.storage.GetChatbot(ctx, chatbotID)
	if err != nil {
		log.Error().Err(err).Str("chatbot_id", chatbotID).Msg("Failed to load chatbot for usage")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load chatbot",
		})
	}
	if chatbot == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Chatbot not found",
		})
	}

	usage := h.limiter.GetDailyUsage(chatbot.ID, userID, chatbot.DailyRequestLimit, chatbot.DailyTokenBudget)

	return c.JSON(fiber.Map{
		"chatbot_id": chatbot.ID,
		"user_id":    userID,
		"requests": fiber.Map{
			"used":  usage.RequestsUsed,
			"limit": usage.RequestsLimit,
		},
		"tokens": fiber.Map{
			"used":  usage.TokensUsed,
			"limit": usage.TokensLimit,
		},
		"resets_at": usage.ResetsAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
}
