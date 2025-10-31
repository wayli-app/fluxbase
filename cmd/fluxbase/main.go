package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wayli-app/fluxbase/internal/api"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// Version information (set via ldflags during build)
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"

	// CLI flags
	showVersion = flag.Bool("version", false, "Show version information")
)

func main() {
	flag.Parse()

	// Show version and exit
	if *showVersion {
		fmt.Printf("Fluxbase %s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Build Date: %s\n", BuildDate)
		os.Exit(0)
	}

	// Initialize logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Info().
		Str("version", Version).
		Str("commit", Commit).
		Str("build_date", BuildDate).
		Msg("Starting Fluxbase")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Set log level
	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Initialize database connection
	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}

	// Initialize API server
	server := api.NewServer(cfg, db)

	// Start server in a goroutine
	go func() {
		log.Info().Str("address", cfg.Server.Address).Msg("Starting Fluxbase server")
		if err := server.Start(); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown
	ctx := context.Background()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}