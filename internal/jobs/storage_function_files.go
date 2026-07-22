package jobs

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// ========== Job Function Files ==========

// CreateJobFunctionFile creates a supporting file for a job function
func (s *Storage) CreateJobFunctionFile(ctx context.Context, file *JobFunctionFile) error {
	tenantID := database.TenantFromContext(ctx)

	query := `
		INSERT INTO jobs.function_files (id, function_id, file_path, content, tenant_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (function_id, file_path) DO UPDATE SET content = EXCLUDED.content
		RETURNING created_at
	`

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			file.ID, file.JobFunctionID, file.FilePath, file.Content, database.TenantOrNil(tenantID),
		).Scan(&file.CreatedAt)
	})
}

// ListJobFunctionFiles lists all files for a job function
func (s *Storage) ListJobFunctionFiles(ctx context.Context, jobFunctionID uuid.UUID) ([]*JobFunctionFile, error) {
	query := `
		SELECT id, function_id, file_path, content, created_at
		FROM jobs.function_files
		WHERE function_id = $1
		ORDER BY file_path
	`

	var files []*JobFunctionFile
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, jobFunctionID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var file JobFunctionFile
			if err := rows.Scan(&file.ID, &file.JobFunctionID, &file.FilePath, &file.Content, &file.CreatedAt); err != nil {
				return err
			}
			files = append(files, &file)
		}
		return rows.Err()
	})
	return files, err
}

// DeleteJobFunctionFiles deletes all files for a job function
func (s *Storage) DeleteJobFunctionFiles(ctx context.Context, jobFunctionID uuid.UUID) error {
	query := `DELETE FROM jobs.function_files WHERE function_id = $1`
	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, jobFunctionID)
		return err
	})
}
