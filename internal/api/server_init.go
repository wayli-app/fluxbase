package api

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/storage/memory/v2"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/observability"
	"github.com/nimbleflux/fluxbase/internal/pubsub"
	"github.com/nimbleflux/fluxbase/internal/ratelimit"
)

func (s *Server) initCore() {
	cfg := s.config
	db := s.db

	app := fiber.New(fiber.Config{
		ServerHeader:      "Fluxbase",
		AppName:           fmt.Sprintf("Fluxbase v%s", s.version),
		BodyLimit:         cfg.Server.BodyLimit,
		StreamRequestBody: true,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		ErrorHandler:      customErrorHandler,
	})

	if cfg.Debug {
		app.Use(func(c fiber.Ctx) error {
			c.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			c.Set("Pragma", "no-cache")
			c.Set("Expires", "0")
			return c.Next()
		})
	}

	tracer, err := observability.NewTracer(context.Background(), cfg.Tracing)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize OpenTelemetry tracer, tracing will be disabled")
	}

	rateLimitStore, err := ratelimit.NewStore(&cfg.Scaling, db.Pool())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize rate limit store, falling back to memory")
		rateLimitStore = nil
	} else {
		log.Info().Str("backend", cfg.Scaling.Backend).Msg("Rate limit store initialized")
	}

	ps, err := pubsub.NewPubSub(&cfg.Scaling, db.Pool())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize pub/sub, cross-instance broadcasting disabled")
		ps = nil
	} else {
		log.Info().Str("backend", cfg.Scaling.Backend).Msg("Pub/sub initialized for cross-instance broadcasting")
	}

	gcInterval := 10 * time.Minute
	if os.Getenv("FLUXBASE_TEST_MODE") == "1" {
		gcInterval = 24 * time.Hour
	}
	sharedMiddlewareStorage := memory.New(memory.Config{
		GCInterval: gcInterval,
	})

	s.app = app
	s.tracer = tracer
	s.rateLimiter = rateLimitStore
	s.pubSub = ps
	s.sharedMiddlewareStorage = sharedMiddlewareStorage
}
