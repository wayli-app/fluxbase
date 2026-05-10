package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/adminui"
)

func (s *Server) setupRoutes() {
	s.auditRoutesAtStartup()

	if err := s.registerRoutesViaRegistry(); err != nil {
		log.Fatal().Err(err).Msg("Failed to setup routes via registry")
	}

	if s.config.Admin.Enabled {
		if s.config.Security.SetupToken == "" {
			log.Error().Msg("Admin UI is enabled but FLUXBASE_SECURITY_SETUP_TOKEN is not set. Admin UI will not be registered for security reasons.")
		} else {
			adminUI := adminui.New(s.config.GetPublicBaseURL())
			adminUI.RegisterRoutes(s.app)
		}
	}

	s.app.Use(func(c fiber.Ctx) error {
		return c.Status(404).JSON(fiber.Map{
			"error": "Not Found",
			"path":  c.Path(),
		})
	})
}

func (s *Server) auditRoutesAtStartup() {
	entries := s.auditRegisteredRoutes()
	publicCount := 0
	authRequiredCount := 0
	for _, e := range entries {
		if e.Public {
			publicCount++
		}
		if e.Auth == "required" || e.Auth == "service_key" || e.Auth == "dashboard" {
			authRequiredCount++
		}
	}
	log.Info().
		Int("total", len(entries)).
		Int("public", publicCount).
		Int("auth_required", authRequiredCount).
		Msg("Route audit completed")
}
