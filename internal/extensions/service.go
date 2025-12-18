package extensions

import (
	"context"
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// Service handles extension management operations
type Service struct {
	db *database.Connection
}

// NewService creates a new extension management service
func NewService(db *database.Connection) *Service {
	return &Service{db: db}
}

// ListExtensions returns all available extensions with their current status
func (s *Service) ListExtensions(ctx context.Context) (*ListExtensionsResponse, error) {
	// Query PostgreSQL's actual extension catalog as source of truth
	// Left join our metadata table for display names, descriptions, and categories
	query := `
		SELECT
			COALESCE(meta.id, gen_random_uuid()) as id,
			pg.name,
			COALESCE(meta.display_name, pg.name) as display_name,
			COALESCE(meta.description, pg.comment, '') as description,
			COALESCE(meta.category, 'utilities') as category,
			COALESCE(meta.is_core, false) as is_core,
			COALESCE(meta.requires_restart, false) as requires_restart,
			COALESCE(meta.documentation_url, '') as documentation_url,
			pg.installed_version IS NOT NULL as is_installed,
			pg.installed_version,
			ee.enabled_at,
			ee.enabled_by::text,
			COALESCE(meta.created_at, NOW()) as created_at,
			COALESCE(meta.updated_at, NOW()) as updated_at
		FROM pg_available_extensions pg
		LEFT JOIN dashboard.available_extensions meta
			ON pg.name = meta.name
		LEFT JOIN dashboard.enabled_extensions ee
			ON pg.name = ee.extension_name AND ee.is_active = true
		ORDER BY COALESCE(meta.category, 'utilities'), COALESCE(meta.display_name, pg.name)
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query extensions: %w", err)
	}
	defer rows.Close()

	var extensions []Extension
	categoryCount := make(map[string]int)

	for rows.Next() {
		var ext Extension
		var installedVersion *string

		err := rows.Scan(
			&ext.ID,
			&ext.Name,
			&ext.DisplayName,
			&ext.Description,
			&ext.Category,
			&ext.IsCore,
			&ext.RequiresRestart,
			&ext.DocumentationURL,
			&ext.IsInstalled,
			&installedVersion,
			&ext.EnabledAt,
			&ext.EnabledBy,
			&ext.CreatedAt,
			&ext.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan extension: %w", err)
		}

		// Set installed version if available
		if installedVersion != nil {
			ext.InstalledVersion = *installedVersion
		}

		// If extension is installed in PostgreSQL, it's enabled
		// This reflects actual PostgreSQL state, not just our tracking table
		ext.IsEnabled = ext.IsInstalled

		// Core extensions are always shown as enabled
		if ext.IsCore {
			ext.IsEnabled = true
		}

		extensions = append(extensions, ext)
		categoryCount[ext.Category]++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating extensions: %w", err)
	}

	// Build categories list
	var categories []Category
	for id, count := range categoryCount {
		name, ok := CategoryDisplayNames[id]
		if !ok {
			name = id
		}
		categories = append(categories, Category{
			ID:    id,
			Name:  name,
			Count: count,
		})
	}

	return &ListExtensionsResponse{
		Extensions: extensions,
		Categories: categories,
	}, nil
}

// GetExtensionStatus returns the status of a specific extension
func (s *Service) GetExtensionStatus(ctx context.Context, name string) (*ExtensionStatusResponse, error) {
	// Check if enabled in our catalog
	var isEnabled bool
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(
			(SELECT is_active FROM dashboard.enabled_extensions
			 WHERE extension_name = $1 AND is_active = true),
			false
		)
	`, name).Scan(&isEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to check extension status: %w", err)
	}

	// Check if installed in PostgreSQL
	installed, version := s.checkExtensionInstalled(ctx, name)

	return &ExtensionStatusResponse{
		Name:             name,
		IsEnabled:        isEnabled,
		IsInstalled:      installed,
		InstalledVersion: version,
	}, nil
}

// EnableExtension enables a PostgreSQL extension
func (s *Service) EnableExtension(ctx context.Context, name string, userID *string, schema string) (*EnableExtensionResponse, error) {
	// Validate extension exists in catalog
	available, err := s.getAvailableExtension(ctx, name)
	if err != nil {
		return nil, err
	}
	if available == nil {
		return &EnableExtensionResponse{
			Name:    name,
			Success: false,
			Message: "Extension not found in catalog",
		}, nil
	}

	// Check if already enabled
	status, err := s.GetExtensionStatus(ctx, name)
	if err != nil {
		return nil, err
	}
	if status.IsInstalled {
		// Extension is already installed in PostgreSQL, but ensure it's tracked
		// This handles cases where extensions were installed manually or tracking failed
		_, err = s.db.Exec(ctx, `
			INSERT INTO dashboard.enabled_extensions (extension_name, enabled_by, is_active)
			VALUES ($1, $2, true)
			ON CONFLICT (extension_name) WHERE is_active = true
			DO UPDATE SET enabled_at = NOW(), enabled_by = $2, error_message = NULL
		`, name, userID)
		if err != nil {
			return nil, fmt.Errorf("extension is installed but failed to record in tracking table: %w", err)
		}

		return &EnableExtensionResponse{
			Name:    name,
			Success: true,
			Message: "Extension is already enabled",
			Version: status.InstalledVersion,
		}, nil
	}

	// Use admin connection to create extension (requires superuser)
	err = s.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		// Build CREATE EXTENSION statement
		sql := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %q", name)
		if schema != "" && schema != "public" {
			sql += fmt.Sprintf(" SCHEMA %q", schema)
		}

		_, err := conn.Exec(ctx, sql)
		return err
	})

	if err != nil {
		log.Error().Err(err).Str("extension", name).Msg("Failed to create extension")

		// Record the error
		s.recordExtensionError(ctx, name, userID, err.Error())

		return &EnableExtensionResponse{
			Name:    name,
			Success: false,
			Message: fmt.Sprintf("Failed to enable extension: %v", err),
		}, nil
	}

	// Get the installed version
	_, version := s.checkExtensionInstalled(ctx, name)

	// Record in enabled_extensions table
	_, err = s.db.Exec(ctx, `
		INSERT INTO dashboard.enabled_extensions (extension_name, enabled_by, is_active)
		VALUES ($1, $2, true)
		ON CONFLICT (extension_name) WHERE is_active = true
		DO UPDATE SET enabled_at = NOW(), enabled_by = $2, error_message = NULL
	`, name, userID)
	if err != nil {
		return nil, fmt.Errorf("extension created successfully but failed to record in tracking table: %w", err)
	}

	log.Info().Str("extension", name).Str("version", version).Msg("Extension enabled successfully")

	return &EnableExtensionResponse{
		Name:    name,
		Success: true,
		Message: "Extension enabled successfully",
		Version: version,
	}, nil
}

// DisableExtension disables a PostgreSQL extension
func (s *Service) DisableExtension(ctx context.Context, name string, userID *string) (*DisableExtensionResponse, error) {
	// Validate extension exists in catalog
	available, err := s.getAvailableExtension(ctx, name)
	if err != nil {
		return nil, err
	}
	if available == nil {
		return &DisableExtensionResponse{
			Name:    name,
			Success: false,
			Message: "Extension not found in catalog",
		}, nil
	}

	// Cannot disable core extensions
	if available.IsCore {
		return &DisableExtensionResponse{
			Name:    name,
			Success: false,
			Message: "Cannot disable core extension",
		}, nil
	}

	// Check if extension is installed
	status, err := s.GetExtensionStatus(ctx, name)
	if err != nil {
		return nil, err
	}
	if !status.IsInstalled {
		return &DisableExtensionResponse{
			Name:    name,
			Success: true,
			Message: "Extension is not currently enabled",
		}, nil
	}

	// Use admin connection to drop extension
	err = s.db.ExecuteWithAdminRole(ctx, func(conn *pgx.Conn) error {
		sql := fmt.Sprintf("DROP EXTENSION IF EXISTS %q CASCADE", name)
		_, err := conn.Exec(ctx, sql)
		return err
	})

	if err != nil {
		log.Error().Err(err).Str("extension", name).Msg("Failed to drop extension")
		return &DisableExtensionResponse{
			Name:    name,
			Success: false,
			Message: fmt.Sprintf("Failed to disable extension: %v", err),
		}, nil
	}

	// Update enabled_extensions table
	_, err = s.db.Exec(ctx, `
		UPDATE dashboard.enabled_extensions
		SET is_active = false, disabled_at = NOW(), disabled_by = $2
		WHERE extension_name = $1 AND is_active = true
	`, name, userID)
	if err != nil {
		log.Warn().Err(err).Str("extension", name).Msg("Failed to record extension disablement")
	}

	log.Info().Str("extension", name).Msg("Extension disabled successfully")

	return &DisableExtensionResponse{
		Name:    name,
		Success: true,
		Message: "Extension disabled successfully",
	}, nil
}

// SyncFromPostgres syncs the extension catalog with what's available in PostgreSQL
func (s *Service) SyncFromPostgres(ctx context.Context) error {
	// Query available extensions from PostgreSQL
	rows, err := s.db.Query(ctx, `
		SELECT name, default_version, installed_version, comment
		FROM pg_available_extensions
		ORDER BY name
	`)
	if err != nil {
		return fmt.Errorf("failed to query pg_available_extensions: %w", err)
	}
	defer rows.Close()

	var pgExtensions []PostgresExtension
	for rows.Next() {
		var ext PostgresExtension
		var comment *string
		err := rows.Scan(&ext.Name, &ext.DefaultVersion, &ext.InstalledVersion, &comment)
		if err != nil {
			return fmt.Errorf("failed to scan extension: %w", err)
		}
		if comment != nil {
			ext.Comment = *comment
		}
		pgExtensions = append(pgExtensions, ext)
	}

	log.Info().Int("count", len(pgExtensions)).Msg("Synced extension list from PostgreSQL")

	return nil
}

// checkExtensionInstalled checks if an extension is installed in PostgreSQL
func (s *Service) checkExtensionInstalled(ctx context.Context, name string) (bool, string) {
	var version *string
	err := s.db.QueryRow(ctx, `
		SELECT installed_version FROM pg_available_extensions WHERE name = $1
	`, name).Scan(&version)
	if err != nil {
		return false, ""
	}
	if version == nil {
		return false, ""
	}
	return true, *version
}

// getAvailableExtension retrieves an extension from the catalog
func (s *Service) getAvailableExtension(ctx context.Context, name string) (*AvailableExtension, error) {
	var ext AvailableExtension
	err := s.db.QueryRow(ctx, `
		SELECT id, name, display_name, COALESCE(description, ''), category,
		       is_core, requires_restart, COALESCE(documentation_url, ''), created_at, updated_at
		FROM dashboard.available_extensions
		WHERE name = $1
	`, name).Scan(
		&ext.ID, &ext.Name, &ext.DisplayName, &ext.Description, &ext.Category,
		&ext.IsCore, &ext.RequiresRestart, &ext.DocumentationURL, &ext.CreatedAt, &ext.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get extension: %w", err)
	}
	return &ext, nil
}

// recordExtensionError records an error when enabling/disabling an extension fails
func (s *Service) recordExtensionError(ctx context.Context, name string, userID *string, errorMsg string) {
	_, err := s.db.Exec(ctx, `
		INSERT INTO dashboard.enabled_extensions (extension_name, enabled_by, is_active, error_message)
		VALUES ($1, $2, false, $3)
		ON CONFLICT (extension_name) WHERE is_active = true
		DO UPDATE SET error_message = $3, updated_at = NOW()
	`, name, userID, errorMsg)
	if err != nil {
		log.Warn().Err(err).Str("extension", name).Msg("Failed to record extension error")
	}
}

// InitializeCoreExtensions ensures core extensions are enabled on startup
func (s *Service) InitializeCoreExtensions(ctx context.Context) error {
	// Get core extensions
	rows, err := s.db.Query(ctx, `
		SELECT name FROM dashboard.available_extensions WHERE is_core = true
	`)
	if err != nil {
		return fmt.Errorf("failed to query core extensions: %w", err)
	}
	defer rows.Close()

	var coreExtensions []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("failed to scan core extension: %w", err)
		}
		coreExtensions = append(coreExtensions, name)
	}

	// Enable each core extension if not already enabled
	for _, name := range coreExtensions {
		status, err := s.GetExtensionStatus(ctx, name)
		if err != nil {
			log.Warn().Err(err).Str("extension", name).Msg("Failed to check core extension status")
			continue
		}
		if !status.IsInstalled {
			log.Info().Str("extension", name).Msg("Enabling core extension")
			_, err := s.EnableExtension(ctx, name, nil, "")
			if err != nil {
				log.Error().Err(err).Str("extension", name).Msg("Failed to enable core extension")
			}
		}
	}

	return nil
}
