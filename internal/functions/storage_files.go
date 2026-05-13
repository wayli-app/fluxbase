package functions

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// CreateSharedModule creates a new shared module or updates it if it already exists (upsert)
func (s *Storage) CreateSharedModule(ctx context.Context, module *SharedModule) error {
	query := `
		INSERT INTO functions.shared_modules (
			module_path, content, description, created_by
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (module_path) DO UPDATE SET
			content = EXCLUDED.content,
			description = EXCLUDED.description,
			version = functions.shared_modules.version + 1,
			updated_at = NOW()
		RETURNING id, version, created_at, updated_at
	`

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			module.ModulePath, module.Content, module.Description, module.CreatedBy,
		).Scan(&module.ID, &module.Version, &module.CreatedAt, &module.UpdatedAt)
	})
	if err != nil {
		return fmt.Errorf("failed to create shared module: %w", err)
	}

	return nil
}

// GetSharedModule retrieves a shared module by path
func (s *Storage) GetSharedModule(ctx context.Context, modulePath string) (*SharedModule, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `
		SELECT id, module_path, content, description, version, created_at, updated_at, created_by
		FROM functions.shared_modules
		WHERE module_path = $1
		  AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
	`

	module := &SharedModule{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, modulePath, database.TenantOrNil(tenantID)).Scan(
			&module.ID, &module.ModulePath, &module.Content, &module.Description,
			&module.Version, &module.CreatedAt, &module.UpdatedAt, &module.CreatedBy,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get shared module: %w", err)
	}

	return module, nil
}

// ListSharedModules returns all shared modules
func (s *Storage) ListSharedModules(ctx context.Context) ([]SharedModule, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `
		SELECT id, module_path, content, description, version, created_at, updated_at, created_by
		FROM functions.shared_modules
		WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
		ORDER BY module_path
	`

	var modules []SharedModule
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, database.TenantOrNil(tenantID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			module := SharedModule{}
			err := rows.Scan(
				&module.ID, &module.ModulePath, &module.Content, &module.Description,
				&module.Version, &module.CreatedAt, &module.UpdatedAt, &module.CreatedBy,
			)
			if err != nil {
				return err
			}
			modules = append(modules, module)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list shared modules: %w", err)
	}

	return modules, nil
}

// UpdateSharedModule updates an existing shared module
func (s *Storage) UpdateSharedModule(ctx context.Context, modulePath string, content string, description *string) error {
	tenantID := database.TenantFromContext(ctx)

	query := `
		UPDATE functions.shared_modules
		SET content = $1, description = $2, version = version + 1, updated_at = NOW()
		WHERE module_path = $3
		  AND (tenant_id = $4 OR ($4 IS NULL AND tenant_id IS NULL))
	`

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, content, description, modulePath, database.TenantOrNil(tenantID))
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update shared module: %w", err)
	}

	return nil
}

// DeleteSharedModule deletes a shared module
func (s *Storage) DeleteSharedModule(ctx context.Context, modulePath string) error {
	tenantID := database.TenantFromContext(ctx)

	query := "DELETE FROM functions.shared_modules WHERE module_path = $1 AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))"

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, modulePath, database.TenantOrNil(tenantID))
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to delete shared module: %w", err)
	}

	return nil
}

// SaveFunctionFiles stores supporting files for a function
func (s *Storage) SaveFunctionFiles(ctx context.Context, functionID uuid.UUID, files []FunctionFile) error {
	tenantID := database.TenantFromContext(ctx)

	// First, delete existing files for this function
	deleteQuery := "DELETE FROM functions.edge_files WHERE function_id = $1 AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))"

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, deleteQuery, functionID, database.TenantOrNil(tenantID))
		if err != nil {
			return err
		}

		// Insert new files
		insertQuery := `
			INSERT INTO functions.edge_files (
				function_id, file_path, content
			) VALUES ($1, $2, $3)
		`

		for _, file := range files {
			_, err := tx.Exec(ctx, insertQuery, functionID, file.FilePath, file.Content)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to save function files: %w", err)
	}

	return nil
}

// GetFunctionFiles retrieves all supporting files for a function
func (s *Storage) GetFunctionFiles(ctx context.Context, functionID uuid.UUID) ([]FunctionFile, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `
		SELECT id, function_id, file_path, content, created_at, updated_at
		FROM functions.edge_files
		WHERE function_id = $1
		  AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
		ORDER BY file_path
	`

	var files []FunctionFile
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, functionID, database.TenantOrNil(tenantID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			file := FunctionFile{}
			err := rows.Scan(
				&file.ID, &file.FunctionID, &file.FilePath, &file.Content,
				&file.CreatedAt, &file.UpdatedAt,
			)
			if err != nil {
				return err
			}
			files = append(files, file)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get function files: %w", err)
	}

	return files, nil
}

// Note: Execution logs are now stored in the central logging schema (logging.entries)
