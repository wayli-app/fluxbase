package routes

import (
	"github.com/gofiber/fiber/v3"
)

type RealtimeDeps struct {
	RequireRealtimeEnabled fiber.Handler
	OptionalAuth           fiber.Handler
	RequireAuth            fiber.Handler
	RequireScope           func(...string) fiber.Handler
	TenantMiddleware       fiber.Handler
	HandleWebSocket        fiber.Handler
	HandleStats            fiber.Handler
}

func BuildRealtimeRoutes(deps *RealtimeDeps) *RouteGroup {
	middlewares := []Middleware{
		{Name: "RequireRealtimeEnabled", Handler: deps.RequireRealtimeEnabled},
	}
	if deps.TenantMiddleware != nil {
		middlewares = append(middlewares, Middleware{
			Name: "TenantContext", Handler: deps.TenantMiddleware,
		})
	}

	return &RouteGroup{
		Name:        "realtime",
		Middlewares: middlewares,
		Routes: []Route{
			{
				Method:  "GET",
				Path:    "/realtime",
				Handler: deps.HandleWebSocket,
				Summary: "WebSocket endpoint for realtime subscriptions",
				Auth:    AuthOptional,
				Scopes:  []string{"realtime:connect"},
				Public:  false,
			},
			{
				Method:  "GET",
				Path:    "/api/v1/realtime/stats",
				Handler: deps.HandleStats,
				Summary: "Get realtime connection statistics",
				Auth:    AuthRequired,
				Scopes:  []string{"realtime:connect"},
			},
		},
		AuthMiddlewares: &AuthMiddlewares{
			Optional: deps.OptionalAuth,
			Required: deps.RequireAuth,
		},
		RequireScope: deps.RequireScope,
	}
}
