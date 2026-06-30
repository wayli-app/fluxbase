package routes

import (
	"github.com/gofiber/fiber/v3"
)

type AIDeps struct {
	RequireAIEnabled       fiber.Handler
	OptionalAuth           fiber.Handler
	RequireAuth            fiber.Handler
	TenantMiddleware       fiber.Handler
	HandleWebSocket        fiber.Handler
	ListPublicChatbots     fiber.Handler
	LookupChatbotByName    fiber.Handler
	GetPublicChatbot       fiber.Handler
	ListUserConversations  fiber.Handler
	GetUserConversation    fiber.Handler
	DeleteUserConversation fiber.Handler
	UpdateUserConversation fiber.Handler
	GetUserUsage           fiber.Handler // GET /api/v1/ai/usage/:chatbotId — per-user daily quota
}

func BuildAIRoutes(deps *AIDeps) *RouteGroup {
	middlewares := []Middleware{
		{Name: "RequireAIEnabled", Handler: deps.RequireAIEnabled},
	}
	if deps.TenantMiddleware != nil {
		middlewares = append(middlewares, Middleware{
			Name: "TenantContext", Handler: deps.TenantMiddleware,
		})
	}

	return &RouteGroup{
		Name:        "ai",
		Middlewares: middlewares,
		Routes: []Route{
			{
				Method:  "GET",
				Path:    "/ai/ws",
				Handler: deps.HandleWebSocket,
				Summary: "WebSocket endpoint for AI chat",
				Auth:    AuthOptional,
			},
			{
				Method:  "GET",
				Path:    "/api/v1/ai/chatbots",
				Handler: deps.ListPublicChatbots,
				Summary: "List public chatbots",
				Auth:    AuthOptional,
			},
			{
				Method:  "GET",
				Path:    "/api/v1/ai/chatbots/by-name/:name",
				Handler: deps.LookupChatbotByName,
				Summary: "Lookup chatbot by name (smart namespace resolution)",
				Auth:    AuthOptional,
			},
			{
				Method:  "GET",
				Path:    "/api/v1/ai/chatbots/:id",
				Handler: deps.GetPublicChatbot,
				Summary: "Get public chatbot by ID",
				Auth:    AuthOptional,
			},
			{
				Method:  "GET",
				Path:    "/api/v1/ai/conversations",
				Handler: deps.ListUserConversations,
				Summary: "List user's conversations",
				Auth:    AuthRequired,
			},
			{
				Method:  "GET",
				Path:    "/api/v1/ai/conversations/:id",
				Handler: deps.GetUserConversation,
				Summary: "Get user conversation",
				Auth:    AuthRequired,
			},
			{
				Method:  "DELETE",
				Path:    "/api/v1/ai/conversations/:id",
				Handler: deps.DeleteUserConversation,
				Summary: "Delete user conversation",
				Auth:    AuthRequired,
			},
			{
				Method:  "PATCH",
				Path:    "/api/v1/ai/conversations/:id",
				Handler: deps.UpdateUserConversation,
				Summary: "Update user conversation",
				Auth:    AuthRequired,
			},
			{
				Method:  "GET",
				Path:    "/api/v1/ai/usage/:chatbotId",
				Handler: deps.GetUserUsage,
				Summary: "Get the current user's daily quota for a chatbot",
				Auth:    AuthRequired,
			},
		},
		AuthMiddlewares: &AuthMiddlewares{
			Optional: deps.OptionalAuth,
			Required: deps.RequireAuth,
		},
	}
}
