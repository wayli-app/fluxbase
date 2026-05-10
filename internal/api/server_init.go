package api

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/storage/memory/v2"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/adminui"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/observability"
	"github.com/nimbleflux/fluxbase/internal/pubsub"
	"github.com/nimbleflux/fluxbase/internal/ratelimit"
	"github.com/nimbleflux/fluxbase/internal/scaling"
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

func (s *Server) initBackgroundServices() {
	cfg := s.config

	if !cfg.Scaling.DisableRealtime && !cfg.Scaling.WorkerOnly {
		if err := s.Realtime.Listener.Start(); err != nil {
			log.Error().Err(err).Msg("Failed to start realtime listener")
		}
	} else {
		log.Info().
			Bool("disable_realtime", cfg.Scaling.DisableRealtime).
			Bool("worker_only", cfg.Scaling.WorkerOnly).
			Msg("Realtime listener disabled by scaling configuration")
	}

	s.Scaling.FunctionsLeader = s.startSchedulerWithLeaderElection(
		"functions-scheduler", scaling.FunctionsSchedulerLockID,
		func() {
			log.Info().Msg("This instance is now the functions scheduler leader")
			if err := s.Functions.Scheduler.Start(); err != nil {
				log.Error().Err(err).Msg("Failed to start edge functions scheduler")
			}
		},
		func() {
			log.Warn().Msg("Lost functions scheduler leadership - stopping scheduler")
			s.Functions.Scheduler.Stop()
		},
	)

	if cfg.Jobs.Enabled && s.Jobs.Manager != nil {
		workerCount := cfg.Jobs.EmbeddedWorkerCount
		if workerCount <= 0 {
			workerCount = 4
		}
		if err := s.Jobs.Manager.Start(context.Background(), workerCount); err != nil {
			log.Error().Err(err).Msg("Failed to start jobs manager")
		} else {
			log.Info().Int("workers", workerCount).Msg("Jobs manager started successfully")
		}

		if s.Jobs.Scheduler != nil {
			s.Scaling.JobsLeader = s.startSchedulerWithLeaderElection(
				"jobs-scheduler", scaling.JobsSchedulerLockID,
				func() {
					log.Info().Msg("This instance is now the jobs scheduler leader")
					if err := s.Jobs.Scheduler.Start(); err != nil {
						log.Error().Err(err).Msg("Failed to start jobs scheduler")
					}
				},
				func() {
					log.Warn().Msg("Lost jobs scheduler leadership - stopping scheduler")
					s.Jobs.Scheduler.Stop()
				},
			)
		}
	}

	if cfg.RPC.Enabled && s.RPC.Scheduler != nil {
		s.Scaling.RPCLeader = s.startSchedulerWithLeaderElection(
			"rpc-scheduler", scaling.RPCSchedulerLockID,
			func() {
				log.Info().Msg("This instance is now the RPC scheduler leader")
				if err := s.RPC.Scheduler.Start(); err != nil {
					log.Error().Err(err).Msg("Failed to start RPC scheduler")
				}
			},
			func() {
				log.Warn().Msg("Lost RPC scheduler leadership - stopping scheduler")
				s.RPC.Scheduler.Stop()
			},
		)
	}

	if err := s.Webhook.Trigger.Start(context.Background()); err != nil {
		log.Error().Err(err).Msg("Failed to start webhook trigger service")
	}

	if s.Logging.Retention != nil {
		s.Logging.Retention.Start()
		log.Info().
			Dur("interval", cfg.Logging.RetentionCheckInterval).
			Msg("Log retention cleanup service started")
	}

	if s.Branching.Scheduler != nil {
		s.Branching.Scheduler.Start()
	}
}

func (s *Server) setupMiddlewares() {
	log.Debug().Msg("Adding requestid middleware")
	s.app.Use(requestid.New())

	if s.config.Tracing.Enabled && s.tracer != nil && s.tracer.IsEnabled() {
		log.Debug().Msg("Adding OpenTelemetry tracing middleware")
		s.app.Use(middleware.TracingMiddleware(middleware.TracingConfig{
			Enabled:            true,
			ServiceName:        s.config.Tracing.ServiceName,
			SkipPaths:          []string{"/health", "/ready", "/metrics"},
			RecordRequestBody:  false,
			RecordResponseBody: false,
		}))
	}

	if s.config.Metrics.Enabled && s.Metrics.Metrics != nil {
		log.Debug().Msg("Adding Prometheus metrics middleware")
		s.app.Use(s.Metrics.Metrics.MetricsMiddleware())
	}

	log.Debug().Msg("Adding security headers middleware")
	s.app.Use(func(c fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/admin") {
			return middleware.AdminUISecurityHeaders()(c)
		}
		return middleware.SecurityHeaders()(c)
	})

	log.Debug().Msg("Adding structured logger middleware")
	s.app.Use(middleware.StructuredLogger(middleware.StructuredLoggerConfig{
		SkipPaths:              []string{"/health", "/ready", "/metrics"},
		SkipSuccessfulRequests: !s.config.Debug,
	}))

	log.Debug().Msg("Adding recover middleware")
	s.app.Use(recover.New(recover.Config{
		EnableStackTrace: s.config.Debug,
	}))

	corsCredentials := s.config.CORS.AllowCredentials
	corsOrigins := s.config.CORS.AllowedOrigins

	hasWildcard := false
	for _, origin := range corsOrigins {
		if origin == "*" {
			hasWildcard = true
			break
		}
	}

	if hasWildcard && corsCredentials {
		log.Warn().Msg("CORS: AllowCredentials disabled because AllowOrigins contains '*' (not allowed per CORS spec)")
		corsCredentials = false
	}
	if !hasWildcard && s.config.PublicBaseURL != "" {
		found := false
		for _, origin := range corsOrigins {
			if origin == s.config.PublicBaseURL {
				found = true
				break
			}
		}
		if !found {
			corsOrigins = append(corsOrigins, s.config.PublicBaseURL)
			log.Debug().Str("public_url", s.config.PublicBaseURL).Msg("Added public base URL to CORS origins")
		}
	}
	log.Debug().
		Strs("origins", corsOrigins).
		Bool("credentials", corsCredentials).
		Msg("Adding CORS middleware")

	corsConfig := cors.Config{
		AllowMethods:     s.config.CORS.AllowedMethods,
		AllowHeaders:     s.config.CORS.AllowedHeaders,
		ExposeHeaders:    s.config.CORS.ExposedHeaders,
		AllowCredentials: corsCredentials,
		MaxAge:           s.config.CORS.MaxAge,
	}

	if hasWildcard {
		corsConfig.AllowOriginsFunc = func(origin string) bool {
			return true
		}
	} else {
		corsConfig.AllowOrigins = corsOrigins
	}

	s.app.Use(cors.New(corsConfig))
	log.Debug().Msg("CORS middleware added")

	if len(s.config.Server.AllowedIPRanges) > 0 {
		log.Info().
			Int("ranges", len(s.config.Server.AllowedIPRanges)).
			Strs("ranges", s.config.Server.AllowedIPRanges).
			Msg("Adding global IP allowlist middleware")
		s.app.Use(middleware.RequireGlobalIPAllowlist(&s.config.Server))
	} else {
		log.Debug().Msg("Global IP allowlist disabled (no ranges configured)")
	}

	s.app.Use(middleware.DynamicGlobalAPILimiter(s.Auth.Handler.authService.GetSettingsCache(), s.sharedMiddlewareStorage))

	if s.config.Server.BodyLimits.Enabled {
		bodyLimitConfig := middleware.BodyLimitsFromConfig(
			s.config.Server.BodyLimits.DefaultLimit,
			s.config.Server.BodyLimits.RESTLimit,
			s.config.Server.BodyLimits.AuthLimit,
			s.config.Server.BodyLimits.StorageLimit,
			s.config.Server.BodyLimits.BulkLimit,
			s.config.Server.BodyLimits.AdminLimit,
			s.config.Server.BodyLimits.MaxJSONDepth,
		)
		s.app.Use(middleware.BodyLimitMiddleware(bodyLimitConfig))
		log.Info().
			Int64("default", s.config.Server.BodyLimits.DefaultLimit).
			Int64("rest", s.config.Server.BodyLimits.RESTLimit).
			Int64("auth", s.config.Server.BodyLimits.AuthLimit).
			Int64("storage", s.config.Server.BodyLimits.StorageLimit).
			Int("max_json_depth", s.config.Server.BodyLimits.MaxJSONDepth).
			Msg("Per-endpoint body limits enabled")
	}

	idempotencyConfig := middleware.DefaultIdempotencyConfig()
	idempotencyConfig.DB = s.db.Pool()
	s.Middleware.Idempotency = middleware.NewIdempotencyMiddleware(idempotencyConfig)
	s.app.Use(s.Middleware.Idempotency.Middleware())
	log.Info().
		Str("header", idempotencyConfig.HeaderName).
		Dur("ttl", idempotencyConfig.TTL).
		Msg("Idempotency key support enabled")

	s.app.Use(compress.New(compress.Config{
		Level: compress.LevelDefault,
	}))
}

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

func (s *Server) startSchedulerWithLeaderElection(name string, lockID int64, startFn, stopFn func()) *scaling.LeaderElector {
	if s.config.Scaling.DisableScheduler || s.config.Scaling.WorkerOnly {
		log.Info().
			Bool("disable_scheduler", s.config.Scaling.DisableScheduler).
			Bool("worker_only", s.config.Scaling.WorkerOnly).
			Msgf("%s disabled by scaling configuration", name)
		return nil
	}
	if s.config.Scaling.EnableSchedulerLeaderElection {
		elector := scaling.NewLeaderElector(s.db.Pool(), lockID, name)
		elector.Start(startFn, stopFn)
		return elector
	}
	startFn()
	return nil
}
