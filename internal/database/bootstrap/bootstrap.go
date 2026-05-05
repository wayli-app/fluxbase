package bootstrap

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

//go:embed bootstrap.sql
var bootstrapSQL string

// Embed the bootstrap.sql file
var _ = embed.FS{}

// Config holds bootstrap configuration
type Config struct {
	Host          string
	Port          int
	Database      string
	User          string
	Password      string
	AdminUser     string
	AdminPassword string
}

// Service handles database bootstrap operations
type Service struct {
	pool      *pgxpool.Pool
	adminPool *pgxpool.Pool
	config    Config
}

// NewService creates a new bootstrap service
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// NewServiceWithConfig creates a new bootstrap service with config for admin connections
func NewServiceWithConfig(pool *pgxpool.Pool, config Config) *Service {
	return &Service{
		pool:   pool,
		config: config,
	}
}

// SetAdminPool sets an explicit admin pool
func (s *Service) SetAdminPool(pool *pgxpool.Pool) {
	s.adminPool = pool
}

// getAdminPool returns an admin pool, creating one if necessary
func (s *Service) getAdminPool(ctx context.Context) (*pgxpool.Pool, error) {
	if s.adminPool != nil {
		return s.adminPool, nil
	}

	// If we have config, create an admin pool
	if s.config.AdminUser != "" && s.config.AdminPassword != "" {
		connStr := fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=disable",
			s.config.AdminUser,
			s.config.AdminPassword,
			s.config.Host,
			s.config.Port,
			s.config.Database,
		)

		pool, err := pgxpool.New(ctx, connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to create admin pool: %w", err)
		}
		s.adminPool = pool
		return pool, nil
	}

	// Fall back to regular pool (may not have sufficient privileges)
	return s.pool, nil
}

// State represents the current bootstrap state
type State struct {
	Bootstrapped   bool
	Version        string
	Checksum       string
	BootstrappedAt string
}

// NeedsBootstrap checks if the database needs bootstrapping
func (s *Service) NeedsBootstrap(ctx context.Context) (bool, error) {
	// Check if bootstrap_state table exists
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'platform'
			AND table_name = 'bootstrap_state'
		)
	`).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check bootstrap_state table existence: %w", err)
	}

	if !exists {
		return true, nil
	}

	// Check if we have a bootstrap record
	var count int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM platform.bootstrap_state
	`).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check bootstrap_state records: %w", err)
	}

	return count == 0, nil
}

// IsBootstrapped returns true if the database has been bootstrapped
func (s *Service) IsBootstrapped(ctx context.Context) (bool, error) {
	needs, err := s.NeedsBootstrap(ctx)
	if err != nil {
		return false, err
	}
	return !needs, nil
}

// GetState returns the current bootstrap state
func (s *Service) GetState(ctx context.Context) (*State, error) {
	var state State
	err := s.pool.QueryRow(ctx, `
		SELECT version, checksum, bootstrapped_at::text
		FROM platform.bootstrap_state
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&state.Version, &state.Checksum, &state.BootstrappedAt)
	if err != nil {
		return &State{Bootstrapped: false}, nil
	}

	state.Bootstrapped = true
	return &state, nil
}

// RunBootstrap executes the bootstrap SQL with admin privileges
func (s *Service) RunBootstrap(ctx context.Context) error {
	log.Info().Msg("Running database bootstrap...")

	// Get admin pool for running bootstrap
	adminPool, err := s.getAdminPool(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin pool: %w", err)
	}

	sql, err := SubstituteAppUser(bootstrapSQL, s.config.User)
	if err != nil {
		return fmt.Errorf("invalid app user for bootstrap: %w", err)
	}
	_, err = adminPool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to execute bootstrap SQL: %w", err)
	}

	log.Info().Msg("Database bootstrap completed successfully")
	return nil
}

// EnsureBootstrap runs the bootstrap SQL unconditionally.
// bootstrap.sql is fully idempotent (uses IF NOT EXISTS, DO $$ blocks with IF FOUND),
// so it is safe to run on every startup. This ensures data migrations (e.g., legacy
// role conversions) are applied before the declarative schema validates constraints.
func (s *Service) EnsureBootstrap(ctx context.Context) error {
	return s.RunBootstrap(ctx)
}

// RunBootstrapOnDB connects to a specific database and runs bootstrap SQL.
// Used for bootstrapping newly created tenant databases.
// The appUser parameter is substituted into {{APP_USER}} placeholders in the SQL.
func RunBootstrapOnDB(ctx context.Context, dbURL string, appUser string) error {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	sql, err := SubstituteAppUser(bootstrapSQL, appUser)
	if err != nil {
		return fmt.Errorf("invalid app user for bootstrap: %w", err)
	}
	_, err = pool.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to execute bootstrap SQL: %w", err)
	}

	return nil
}

// Close closes any admin pool resources
func (s *Service) Close() {
	if s.adminPool != nil {
		s.adminPool.Close()
	}
}
