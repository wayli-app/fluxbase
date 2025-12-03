package migrations

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wayli-app/fluxbase/internal/database"
)

// Migration represents a user-defined database migration
type Migration struct {
	ID           uuid.UUID  `json:"id"`
	Namespace    string     `json:"namespace"`
	Name         string     `json:"name"`
	Description  *string    `json:"description"`
	UpSQL        string     `json:"up_sql"`
	DownSQL      *string    `json:"down_sql"`
	Version      int        `json:"version"`
	Status       string     `json:"status"` // pending, applied, failed, rolled_back
	CreatedBy    *uuid.UUID `json:"created_by"`
	AppliedBy    *uuid.UUID `json:"applied_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	AppliedAt    *time.Time `json:"applied_at"`
	RolledBackAt *time.Time `json:"rolled_back_at"`
}

// ExecutionLog represents a migration execution attempt
type ExecutionLog struct {
	ID           uuid.UUID  `json:"id"`
	MigrationID  uuid.UUID  `json:"migration_id"`
	Action       string     `json:"action"` // apply, rollback
	Status       string     `json:"status"` // success, failed
	DurationMs   *int       `json:"duration_ms"`
	ErrorMessage *string    `json:"error_message"`
	Logs         *string    `json:"logs"`
	ExecutedAt   time.Time  `json:"executed_at"`
	ExecutedBy   *uuid.UUID `json:"executed_by"`
}

// Storage handles database operations for migrations
type Storage struct {
	db *database.Connection
}

// NewStorage creates a new migrations storage
func NewStorage(db *database.Connection) *Storage {
	return &Storage{db: db}
}

// CreateMigration creates a new migration
func (s *Storage) CreateMigration(ctx context.Context, m *Migration) error {
	query := `
		INSERT INTO migrations.app (
			namespace, name, description, up_sql, down_sql, created_by
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, version, status, created_at, updated_at
	`

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			m.Namespace, m.Name, m.Description, m.UpSQL, m.DownSQL, m.CreatedBy,
		).Scan(&m.ID, &m.Version, &m.Status, &m.CreatedAt, &m.UpdatedAt)
	})

	if err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	return nil
}

// GetMigration retrieves a migration by name and namespace
func (s *Storage) GetMigration(ctx context.Context, namespace, name string) (*Migration, error) {
	query := `
		SELECT id, namespace, name, description, up_sql, down_sql, version, status,
		       created_by, applied_by, created_at, updated_at, applied_at, rolled_back_at
		FROM migrations.app
		WHERE namespace = $1 AND name = $2
	`

	m := &Migration{}
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, namespace, name).Scan(
			&m.ID, &m.Namespace, &m.Name, &m.Description, &m.UpSQL, &m.DownSQL,
			&m.Version, &m.Status, &m.CreatedBy, &m.AppliedBy,
			&m.CreatedAt, &m.UpdatedAt, &m.AppliedAt, &m.RolledBackAt,
		)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get migration: %w", err)
	}

	return m, nil
}

// ListMigrations returns all migrations in a namespace, optionally filtered by status
func (s *Storage) ListMigrations(ctx context.Context, namespace string, status *string) ([]Migration, error) {
	query := `
		SELECT id, namespace, name, description, up_sql, down_sql, version, status,
		       created_by, applied_by, created_at, updated_at, applied_at, rolled_back_at
		FROM migrations.app
		WHERE namespace = $1
	`

	args := []interface{}{namespace}

	if status != nil {
		query += " AND status = $2"
		args = append(args, *status)
	}

	query += " ORDER BY name ASC"

	var migrations []Migration
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			m := Migration{}
			err := rows.Scan(
				&m.ID, &m.Namespace, &m.Name, &m.Description, &m.UpSQL, &m.DownSQL,
				&m.Version, &m.Status, &m.CreatedBy, &m.AppliedBy,
				&m.CreatedAt, &m.UpdatedAt, &m.AppliedAt, &m.RolledBackAt,
			)
			if err != nil {
				return err
			}
			migrations = append(migrations, m)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list migrations: %w", err)
	}

	return migrations, nil
}

// UpdateMigration updates a migration (only allowed if status is pending)
func (s *Storage) UpdateMigration(ctx context.Context, namespace, name string, updates map[string]interface{}) error {
	// Build dynamic UPDATE query
	query := "UPDATE migrations.app SET updated_at = NOW()"
	args := []interface{}{}
	argCount := 1

	allowedFields := map[string]bool{
		"description": true,
		"up_sql":      true,
		"down_sql":    true,
	}

	for key, value := range updates {
		if !allowedFields[key] {
			continue
		}
		query += fmt.Sprintf(", %s = $%d", key, argCount)
		args = append(args, value)
		argCount++
	}

	query += fmt.Sprintf(" WHERE namespace = $%d AND name = $%d AND status = 'pending'", argCount, argCount+1)
	args = append(args, namespace, name)

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return fmt.Errorf("migration not found or already applied")
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to update migration: %w", err)
	}

	return nil
}

// DeleteMigration deletes a migration (only allowed if status is pending)
func (s *Storage) DeleteMigration(ctx context.Context, namespace, name string) error {
	query := "DELETE FROM migrations.app WHERE namespace = $1 AND name = $2 AND status = 'pending'"

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, namespace, name)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return fmt.Errorf("migration not found or already applied")
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to delete migration: %w", err)
	}

	return nil
}

// UpdateMigrationStatus updates the status of a migration
func (s *Storage) UpdateMigrationStatus(ctx context.Context, id uuid.UUID, status string, appliedBy *uuid.UUID) error {
	var query string
	var args []interface{}

	switch status {
	case "applied":
		query = "UPDATE migrations.app SET status = $1, applied_at = NOW(), applied_by = $2, updated_at = NOW() WHERE id = $3"
		args = []interface{}{status, appliedBy, id}
	case "rolled_back":
		query = "UPDATE migrations.app SET status = $1, rolled_back_at = NOW(), updated_at = NOW() WHERE id = $2"
		args = []interface{}{status, id}
	case "failed":
		query = "UPDATE migrations.app SET status = $1, updated_at = NOW() WHERE id = $2"
		args = []interface{}{status, id}
	default:
		return fmt.Errorf("invalid status: %s", status)
	}

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, args...)
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to update migration status: %w", err)
	}

	return nil
}

// LogExecution logs a migration execution attempt
func (s *Storage) LogExecution(ctx context.Context, log *ExecutionLog) error {
	query := `
		INSERT INTO migrations.execution_logs (
			migration_id, action, status, duration_ms, error_message, logs, executed_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, executed_at
	`

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			log.MigrationID, log.Action, log.Status, log.DurationMs,
			log.ErrorMessage, log.Logs, log.ExecutedBy,
		).Scan(&log.ID, &log.ExecutedAt)
	})

	if err != nil {
		return fmt.Errorf("failed to log execution: %w", err)
	}

	return nil
}

// GetExecutionLogs returns execution logs for a migration
func (s *Storage) GetExecutionLogs(ctx context.Context, migrationID uuid.UUID, limit int) ([]ExecutionLog, error) {
	query := `
		SELECT id, migration_id, action, status, duration_ms, error_message, logs, executed_at, executed_by
		FROM migrations.execution_logs
		WHERE migration_id = $1
		ORDER BY executed_at DESC
		LIMIT $2
	`

	var logs []ExecutionLog
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, migrationID, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			log := ExecutionLog{}
			err := rows.Scan(
				&log.ID, &log.MigrationID, &log.Action, &log.Status,
				&log.DurationMs, &log.ErrorMessage, &log.Logs,
				&log.ExecutedAt, &log.ExecutedBy,
			)
			if err != nil {
				return err
			}
			logs = append(logs, log)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get execution logs: %w", err)
	}

	return logs, nil
}
