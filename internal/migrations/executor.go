package migrations

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/database"
)

// Executor handles migration execution
type Executor struct {
	storage *Storage
	db      *database.Connection
}

// NewExecutor creates a new migration executor
func NewExecutor(db *database.Connection) *Executor {
	return &Executor{
		storage: NewStorage(db),
		db:      db,
	}
}

// ApplyMigration applies a single migration
func (e *Executor) ApplyMigration(ctx context.Context, namespace, name string, executedBy *uuid.UUID) error {
	// Get migration
	migration, err := e.storage.GetMigration(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("failed to get migration: %w", err)
	}

	// Check if already applied
	if migration.Status == "applied" {
		log.Info().
			Str("namespace", namespace).
			Str("name", name).
			Msg("Migration already applied, skipping")
		return nil
	}

	// Check if status allows application
	if migration.Status != "pending" && migration.Status != "failed" {
		return fmt.Errorf("migration status is %s, cannot apply", migration.Status)
	}

	log.Info().
		Str("namespace", namespace).
		Str("name", name).
		Msg("Applying migration")

	startTime := time.Now()
	var executionLog *ExecutionLog

	// Execute migration in transaction with admin credentials
	// Migrations require DDL privileges (CREATE TABLE, ALTER, etc.)
	err = e.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		// Execute the up SQL
		_, err := conn.Exec(ctx, migration.UpSQL)
		return err
	})

	durationMs := int(time.Since(startTime).Milliseconds())

	if err != nil {
		// Migration failed
		log.Error().
			Err(err).
			Str("namespace", namespace).
			Str("name", name).
			Int("duration_ms", durationMs).
			Msg("Migration failed")

		errMsg := err.Error()
		executionLog = &ExecutionLog{
			MigrationID:  migration.ID,
			Action:       "apply",
			Status:       "failed",
			DurationMs:   &durationMs,
			ErrorMessage: &errMsg,
			ExecutedBy:   executedBy,
		}

		// Log execution
		_ = e.storage.LogExecution(ctx, executionLog)

		// Update migration status to failed
		_ = e.storage.UpdateMigrationStatus(ctx, migration.ID, "failed", executedBy)

		return fmt.Errorf("migration failed: %w", err)
	}

	// Migration succeeded
	log.Info().
		Str("namespace", namespace).
		Str("name", name).
		Int("duration_ms", durationMs).
		Msg("Migration applied successfully")

	executionLog = &ExecutionLog{
		MigrationID: migration.ID,
		Action:      "apply",
		Status:      "success",
		DurationMs:  &durationMs,
		ExecutedBy:  executedBy,
	}

	// Log execution
	if err := e.storage.LogExecution(ctx, executionLog); err != nil {
		log.Warn().Err(err).Msg("Failed to log migration execution")
	}

	// Update migration status to applied
	if err := e.storage.UpdateMigrationStatus(ctx, migration.ID, "applied", executedBy); err != nil {
		return fmt.Errorf("failed to update migration status: %w", err)
	}

	return nil
}

// RollbackMigration rolls back a single migration
func (e *Executor) RollbackMigration(ctx context.Context, namespace, name string, executedBy *uuid.UUID) error {
	// Get migration
	migration, err := e.storage.GetMigration(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("failed to get migration: %w", err)
	}

	// Check if applied
	if migration.Status != "applied" {
		return fmt.Errorf("migration status is %s, cannot rollback", migration.Status)
	}

	// Check if down SQL exists
	if migration.DownSQL == nil || *migration.DownSQL == "" {
		return fmt.Errorf("migration has no rollback SQL")
	}

	log.Info().
		Str("namespace", namespace).
		Str("name", name).
		Msg("Rolling back migration")

	startTime := time.Now()
	var executionLog *ExecutionLog

	// Execute rollback in transaction with admin credentials
	// Rollbacks may require DDL privileges (DROP TABLE, ALTER, etc.)
	err = e.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		// Execute the down SQL
		_, err := conn.Exec(ctx, *migration.DownSQL)
		return err
	})

	durationMs := int(time.Since(startTime).Milliseconds())

	if err != nil {
		// Rollback failed
		log.Error().
			Err(err).
			Str("namespace", namespace).
			Str("name", name).
			Int("duration_ms", durationMs).
			Msg("Rollback failed")

		errMsg := err.Error()
		executionLog = &ExecutionLog{
			MigrationID:  migration.ID,
			Action:       "rollback",
			Status:       "failed",
			DurationMs:   &durationMs,
			ErrorMessage: &errMsg,
			ExecutedBy:   executedBy,
		}

		// Log execution
		_ = e.storage.LogExecution(ctx, executionLog)

		return fmt.Errorf("rollback failed: %w", err)
	}

	// Rollback succeeded
	log.Info().
		Str("namespace", namespace).
		Str("name", name).
		Int("duration_ms", durationMs).
		Msg("Migration rolled back successfully")

	executionLog = &ExecutionLog{
		MigrationID: migration.ID,
		Action:      "rollback",
		Status:      "success",
		DurationMs:  &durationMs,
		ExecutedBy:  executedBy,
	}

	// Log execution
	if err := e.storage.LogExecution(ctx, executionLog); err != nil {
		log.Warn().Err(err).Msg("Failed to log migration execution")
	}

	// Update migration status to rolled_back
	if err := e.storage.UpdateMigrationStatus(ctx, migration.ID, "rolled_back", executedBy); err != nil {
		return fmt.Errorf("failed to update migration status: %w", err)
	}

	return nil
}

// ApplyPendingMigrations applies all pending migrations in a namespace in order
func (e *Executor) ApplyPendingMigrations(ctx context.Context, namespace string, executedBy *uuid.UUID) ([]string, []string, error) {
	// Get pending migrations
	pendingStatus := "pending"
	migrations, err := e.storage.ListMigrations(ctx, namespace, &pendingStatus)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list pending migrations: %w", err)
	}

	if len(migrations) == 0 {
		log.Info().Str("namespace", namespace).Msg("No pending migrations to apply")
		return []string{}, []string{}, nil
	}

	log.Info().
		Str("namespace", namespace).
		Int("count", len(migrations)).
		Msg("Applying pending migrations")

	applied := []string{}
	failed := []string{}

	// Apply migrations in order (sorted by name in ListMigrations)
	for _, migration := range migrations {
		err := e.ApplyMigration(ctx, namespace, migration.Name, executedBy)
		if err != nil {
			log.Error().
				Err(err).
				Str("namespace", namespace).
				Str("name", migration.Name).
				Msg("Failed to apply migration, stopping")
			failed = append(failed, migration.Name)
			return applied, failed, fmt.Errorf("failed to apply migration %s: %w", migration.Name, err)
		}
		applied = append(applied, migration.Name)
	}

	log.Info().
		Str("namespace", namespace).
		Int("applied", len(applied)).
		Msg("All pending migrations applied successfully")

	return applied, failed, nil
}

// ValidateMigration validates migration SQL without applying it (dry run)
func (e *Executor) ValidateMigration(ctx context.Context, sql string) error {
	// Try to prepare the statement (validates syntax)
	err := database.WrapWithServiceRole(ctx, e.db, func(tx pgx.Tx) error {
		// Parse and validate SQL (but don't execute)
		// This is a simple validation - checks syntax but not semantic correctness
		conn := tx.Conn()
		_, err := conn.Prepare(ctx, "validate_migration", sql)
		if err != nil {
			return fmt.Errorf("invalid SQL: %w", err)
		}
		// Deallocate prepared statement
		_, _ = conn.Exec(ctx, "DEALLOCATE validate_migration")
		return nil
	})

	return err
}
