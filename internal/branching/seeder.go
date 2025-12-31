package branching

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Seeder handles seed data execution for branches
type Seeder struct {
	seedsPath string
}

// NewSeeder creates a new seeder instance
func NewSeeder(seedsPath string) *Seeder {
	return &Seeder{seedsPath: seedsPath}
}

// SeedFile represents a seed SQL file
type SeedFile struct {
	Name    string // e.g., "001_initial_users"
	Path    string // Full file path
	Content string // SQL content
}

// DiscoverSeedFiles scans the seeds directory and returns sorted seed files
func (s *Seeder) DiscoverSeedFiles(ctx context.Context) ([]SeedFile, error) {
	// Check if seeds directory exists
	if _, err := os.Stat(s.seedsPath); os.IsNotExist(err) {
		log.Warn().Str("path", s.seedsPath).Msg("Seeds directory not found")
		return []SeedFile{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to check seeds directory: %w", err)
	}

	// Read all files in the directory
	entries, err := os.ReadDir(s.seedsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read seeds directory: %w", err)
	}

	var seeds []SeedFile
	for _, entry := range entries {
		// Skip directories and non-SQL files
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Get full path
		fullPath := filepath.Join(s.seedsPath, entry.Name())

		// Read file content
		content, err := os.ReadFile(fullPath)
		if err != nil {
			log.Warn().Err(err).Str("file", entry.Name()).Msg("Failed to read seed file, skipping")
			continue
		}

		// Extract name without extension
		name := strings.TrimSuffix(entry.Name(), ".sql")

		seeds = append(seeds, SeedFile{
			Name:    name,
			Path:    fullPath,
			Content: string(content),
		})
	}

	// Sort by filename (lexicographic order ensures 001_ comes before 002_)
	sort.Slice(seeds, func(i, j int) bool {
		return seeds[i].Name < seeds[j].Name
	})

	log.Info().Int("count", len(seeds)).Str("path", s.seedsPath).Msg("Discovered seed files")

	return seeds, nil
}

// ExecuteSeeds runs all seed files against the target database in order
func (s *Seeder) ExecuteSeeds(ctx context.Context, pool *pgxpool.Pool, branchID uuid.UUID) error {
	// Discover all seed files
	seeds, err := s.DiscoverSeedFiles(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover seed files: %w", err)
	}

	if len(seeds) == 0 {
		log.Info().Msg("No seed files found, skipping seed execution")
		return nil
	}

	// Get list of already-executed seeds for this branch
	executedSeeds, err := s.getExecutedSeeds(ctx, pool, branchID)
	if err != nil {
		return fmt.Errorf("failed to get executed seeds: %w", err)
	}

	// Execute each seed file in order
	for _, seed := range seeds {
		// Skip if already executed successfully
		if executedSeeds[seed.Name] {
			log.Debug().Str("seed", seed.Name).Msg("Seed already executed, skipping")
			continue
		}

		// Execute the seed
		if err := s.executeSingleSeed(ctx, pool, branchID, seed); err != nil {
			return fmt.Errorf("seed %s failed: %w", seed.Name, err)
		}
	}

	log.Info().Int("total", len(seeds)).Msg("All seed files executed successfully")
	return nil
}

// executeSingleSeed executes a single seed file in a transaction
func (s *Seeder) executeSingleSeed(ctx context.Context, pool *pgxpool.Pool, branchID uuid.UUID, seed SeedFile) error {
	startTime := time.Now()

	log.Info().Str("seed", seed.Name).Msg("Executing seed file")

	// Begin transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			log.Warn().Err(err).Msg("Failed to rollback transaction")
		}
	}()

	// Log seed execution start
	if err := s.logSeedExecution(ctx, tx, branchID, seed.Name, "started", 0, ""); err != nil {
		return fmt.Errorf("failed to log seed execution start: %w", err)
	}

	// Execute SQL
	_, execErr := tx.Exec(ctx, seed.Content)
	duration := time.Since(startTime)
	durationMs := int(duration.Milliseconds())

	if execErr != nil {
		// Log failure
		errMsg := execErr.Error()
		if err := s.logSeedExecution(ctx, tx, branchID, seed.Name, "failed", durationMs, errMsg); err != nil {
			log.Warn().Err(err).Msg("Failed to log seed execution failure")
		}

		// Try to commit the failure log (even though seed execution failed)
		if err := tx.Commit(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to commit seed failure log")
		}

		return fmt.Errorf("SQL execution failed: %w", execErr)
	}

	// Log success
	if err := s.logSeedExecution(ctx, tx, branchID, seed.Name, "success", durationMs, ""); err != nil {
		return fmt.Errorf("failed to log seed execution success: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info().
		Str("seed", seed.Name).
		Int("duration_ms", durationMs).
		Msg("Seed file executed successfully")

	return nil
}

// logSeedExecution records seed execution to branching.seed_execution_log
func (s *Seeder) logSeedExecution(ctx context.Context, tx pgx.Tx, branchID uuid.UUID, fileName, status string, durationMs int, errMsg string) error {
	query := `
		INSERT INTO branching.seed_execution_log (branch_id, seed_file_name, status, error_message, duration_ms)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (branch_id, seed_file_name)
		DO UPDATE SET
			status = EXCLUDED.status,
			error_message = EXCLUDED.error_message,
			executed_at = NOW(),
			duration_ms = EXCLUDED.duration_ms
	`

	var errorMessage *string
	if errMsg != "" {
		errorMessage = &errMsg
	}

	var duration *int
	if durationMs > 0 {
		duration = &durationMs
	}

	_, err := tx.Exec(ctx, query, branchID, fileName, status, errorMessage, duration)
	return err
}

// getExecutedSeeds retrieves list of seeds already run successfully on this branch
func (s *Seeder) getExecutedSeeds(ctx context.Context, pool *pgxpool.Pool, branchID uuid.UUID) (map[string]bool, error) {
	query := `
		SELECT seed_file_name
		FROM branching.seed_execution_log
		WHERE branch_id = $1 AND status = 'success'
	`

	rows, err := pool.Query(ctx, query, branchID)
	if err != nil {
		return nil, fmt.Errorf("failed to query executed seeds: %w", err)
	}
	defer rows.Close()

	executed := make(map[string]bool)
	for rows.Next() {
		var fileName string
		if err := rows.Scan(&fileName); err != nil {
			return nil, fmt.Errorf("failed to scan seed file name: %w", err)
		}
		executed[fileName] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating executed seeds: %w", err)
	}

	return executed, nil
}
