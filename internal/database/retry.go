package database

import (
	"fmt"
	"math"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
)

// ConnectWithRetry attempts to connect to the database with exponential backoff.
func ConnectWithRetry(cfg config.DatabaseConfig, maxAttempts int) (*Connection, error) {
	var db *Connection
	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Info().
			Int("attempt", attempt).
			Int("max_attempts", maxAttempts).
			Str("host", cfg.Host).
			Int("port", cfg.Port).
			Msg("Attempting to connect to database...")

		db, err = NewConnection(cfg)
		if err == nil {
			log.Info().Msg("Successfully connected to database")
			return db, nil
		}

		if attempt >= maxAttempts {
			break
		}

		backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
		log.Warn().
			Err(err).
			Int("attempt", attempt).
			Dur("retry_in", backoff).
			Msg("Database connection failed, retrying...")
		time.Sleep(backoff)
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxAttempts, err)
}
