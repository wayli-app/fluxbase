package routes

import (
	"github.com/gofiber/fiber/v3"
)

type InvitationDeps struct {
	AcceptLimiter      fiber.Handler
	ValidateInvitation fiber.Handler
	AcceptInvitation   fiber.Handler
}

func BuildInvitationRoutes(deps *InvitationDeps) *RouteGroup {
	acceptMiddlewares := []Middleware{}
	if deps.AcceptLimiter != nil {
		acceptMiddlewares = append(acceptMiddlewares, Middleware{Name: "AcceptLimiter", Handler: deps.AcceptLimiter})
	}

	return &RouteGroup{
		Name:   "invitations",
		Prefix: "/api/v1/invitations",
		Routes: []Route{
			{
				Method:  "GET",
				Path:    "/:token/validate",
				Handler: deps.ValidateInvitation,
				Summary: "Validate invitation token",
				Auth:    AuthNone,
				Public:  true,
			},
			{
				Method:      "POST",
				Path:        "/:token/accept",
				Handler:     deps.AcceptInvitation,
				Summary:     "Accept invitation",
				Auth:        AuthNone,
				Public:      true,
				Middlewares: acceptMiddlewares,
			},
		},
	}
}
