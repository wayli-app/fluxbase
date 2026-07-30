package extensions

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// validIdentifierRegex ensures identifier names are safe PostgreSQL identifiers
// Only allows: letters, digits, underscores, starting with letter or underscore
var validIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// quoteIdentifier safely quotes a PostgreSQL identifier to prevent SQL injection.
// It wraps the identifier in double quotes and escapes any embedded double quotes.
func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

// isValidIdentifier checks if a string is a valid PostgreSQL identifier
func isValidIdentifier(s string) bool {
	return validIdentifierRegex.MatchString(s)
}

// requiredExtRegex matches PostgreSQL errors like:
// "required extension \"cube\" is not installed"
var requiredExtRegex = regexp.MustCompile(`required extension "([a-zA-Z_][a-zA-Z0-9_]*)" is not installed`)

// Service handles extension management operations
type Service struct {
	db *database.Connection
}

// NewService creates a new extension management service
func NewService(db *database.Connection) *Service {
	return &Service{db: db}
}

// ListExtensions returns all available extensions with their current status (default tenant).
func (s *Service) ListExtensions(ctx context.Context) (*ListExtensionsResponse, error) {
	return s.ListExtensionsForTenant(ctx, nil, nil)
}

// ListExtensionsForTenant returns extensions available in the tenant's database.
// tenantPool is used to query pg_available_extensions on the tenant's database.
// When nil, the main database pool is used.
func (s *Service) ListExtensionsForTenant(ctx context.Context, tenantID *string, tenantPool *pgxpool.Pool) (*ListExtensionsResponse, error) {
	// Query pg_available_extensions from the tenant's database (or main if no separate DB)
	pool := tenantPool
	if pool == nil {
		pool = s.db.Pool()
	}

	pgRows, err := pool.Query(ctx, `
		SELECT name, default_version, installed_version, comment
		FROM pg_available_extensions
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query pg_available_extensions: %w", err)
	}
	defer pgRows.Close()

	// Build map of pg extensions: name -> PostgresExtension
	pgExts := make(map[string]PostgresExtension)
	for pgRows.Next() {
		var ext PostgresExtension
		var comment *string
		if err := pgRows.Scan(&ext.Name, &ext.DefaultVersion, &ext.InstalledVersion, &comment); err != nil {
			return nil, fmt.Errorf("failed to scan pg extension: %w", err)
		}
		if comment != nil {
			ext.Comment = *comment
		}
		pgExts[ext.Name] = ext
	}
	if err := pgRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pg extensions: %w", err)
	}

	// Query catalog metadata and enabled_extensions tracking from the main database
	var metaMap map[string]extMeta
	var trackingMap map[string]trackInfo
	err = database.WrapWithTenantAwareRole(ctx, s.db, tenantIDStr(tenantID), func(tx pgx.Tx) error {
		// Query platform.available_extensions catalog
		metaRows, err := tx.Query(ctx, `
			SELECT id, name, display_name, COALESCE(description, ''), category,
			       is_core, requires_restart, COALESCE(documentation_url, ''), created_at, updated_at
			FROM platform.available_extensions
		`)
		if err != nil {
			return fmt.Errorf("failed to query extension catalog: %w", err)
		}
		defer metaRows.Close()

		metaMap = make(map[string]extMeta)
		for metaRows.Next() {
			var m extMeta
			if err := metaRows.Scan(&m.ID, &m.Name, &m.DisplayName, &m.Description, &m.Category,
				&m.IsCore, &m.RequiresRestart, &m.DocumentationURL, &m.CreatedAt, &m.UpdatedAt); err != nil {
				return fmt.Errorf("failed to scan extension metadata: %w", err)
			}
			metaMap[m.Name] = m
		}
		if err := metaRows.Err(); err != nil {
			return fmt.Errorf("error iterating extension metadata: %w", err)
		}

		// Query platform.enabled_extensions tracking
		filter := mustTenantFilter(tenantID, 1)
		var trackingRows pgx.Rows
		if tenantID != nil {
			trackingRows, err = tx.Query(ctx, fmt.Sprintf(`
				SELECT extension_name, enabled_at, enabled_by::text
				FROM platform.enabled_extensions
				WHERE is_active = true AND %s
			`, filter), *tenantID)
		} else {
			trackingRows, err = tx.Query(ctx, fmt.Sprintf(`
				SELECT extension_name, enabled_at, enabled_by::text
				FROM platform.enabled_extensions
				WHERE is_active = true AND %s
			`, filter))
		}
		if err != nil {
			return fmt.Errorf("failed to query enabled extensions: %w", err)
		}
		defer trackingRows.Close()

		trackingMap = make(map[string]trackInfo)
		for trackingRows.Next() {
			var tName string
			var info trackInfo
			if err := trackingRows.Scan(&tName, &info.EnabledAt, &info.EnabledBy); err != nil {
				return fmt.Errorf("failed to scan enabled extension tracking: %w", err)
			}
			trackingMap[tName] = info
		}
		if err := trackingRows.Err(); err != nil {
			return fmt.Errorf("error iterating enabled extensions: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Merge results
	var extensions []Extension
	categoryCount := make(map[string]int)

	for name, pg := range pgExts {
		meta, hasMeta := metaMap[name]
		track, hasTracking := trackingMap[name]

		isInstalled := pg.InstalledVersion != nil
		installedVersion := ""
		if isInstalled {
			installedVersion = *pg.InstalledVersion
		}

		displayName := name
		description := pg.Comment
		category := "utilities"
		isCore := false
		requiresRestart := false
		docURL := ""
		var id string
		createdAt := time.Now()
		updatedAt := time.Now()

		if hasMeta {
			id = meta.ID
			displayName = meta.DisplayName
			description = meta.Description
			category = meta.Category
			isCore = meta.IsCore
			requiresRestart = meta.RequiresRestart
			docURL = meta.DocumentationURL
			createdAt = meta.CreatedAt
			updatedAt = meta.UpdatedAt
		}

		ext := Extension{
			ID:               id,
			Name:             name,
			DisplayName:      displayName,
			Description:      description,
			Category:         category,
			IsCore:           isCore,
			RequiresRestart:  requiresRestart,
			DocumentationURL: docURL,
			IsInstalled:      isInstalled,
			InstalledVersion: installedVersion,
			IsEnabled:        isInstalled || isCore,
			CreatedAt:        createdAt,
			UpdatedAt:        updatedAt,
		}

		if hasTracking {
			ext.EnabledAt = &track.EnabledAt
			ext.EnabledBy = track.EnabledBy
		}

		if isCore {
			ext.IsEnabled = true
		}

		extensions = append(extensions, ext)
		categoryCount[category]++
	}

	sort.Slice(extensions, func(i, j int) bool {
		if extensions[i].Category != extensions[j].Category {
			return extensions[i].Category < extensions[j].Category
		}
		return extensions[i].DisplayName < extensions[j].DisplayName
	})

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

	sort.Slice(categories, func(i, j int) bool {
		return categories[i].ID < categories[j].ID
	})

	return &ListExtensionsResponse{
		Extensions: extensions,
		Categories: categories,
	}, nil
}

// extMeta holds extension catalog metadata scanned from platform.available_extensions
type extMeta struct {
	ID               string
	Name             string
	DisplayName      string
	Description      string
	Category         string
	IsCore           bool
	RequiresRestart  bool
	DocumentationURL string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// trackInfo holds enabled extension tracking info scanned from platform.enabled_extensions
type trackInfo struct {
	EnabledAt time.Time
	EnabledBy *string
}

// mustTenantFilter returns a SQL WHERE fragment for filtering by tenant_id.
func mustTenantFilter(tenantID *string, idx int) string {
	if tenantID == nil {
		return "tenant_id IS NULL"
	}
	return fmt.Sprintf("tenant_id = $%d", idx)
}

// tenantIDStr dereferences a tenant ID pointer, returning "" for nil.
func tenantIDStr(tid *string) string {
	if tid == nil {
		return ""
	}
	return *tid
}

// GetExtensionStatus returns the status of a specific extension (default tenant).
func (s *Service) GetExtensionStatus(ctx context.Context, name string) (*ExtensionStatusResponse, error) {
	return s.GetExtensionStatusForTenant(ctx, name, nil, nil)
}

// GetExtensionStatusForTenant returns the status of a specific extension for a tenant.
func (s *Service) GetExtensionStatusForTenant(ctx context.Context, name string, tenantID *string, tenantPool *pgxpool.Pool) (*ExtensionStatusResponse, error) {
	filter := mustTenantFilter(tenantID, 2)
	var isEnabled bool
	var err error
	err = database.WrapWithTenantAwareRole(ctx, s.db, tenantIDStr(tenantID), func(tx pgx.Tx) error {
		if tenantID != nil {
			return tx.QueryRow(ctx, fmt.Sprintf(`
				SELECT COALESCE(
					(SELECT is_active FROM platform.enabled_extensions
					 WHERE extension_name = $1 AND is_active = true AND %s),
					false
				)
			`, filter), name, *tenantID).Scan(&isEnabled)
		}
		return tx.QueryRow(ctx, fmt.Sprintf(`
			SELECT COALESCE(
				(SELECT is_active FROM platform.enabled_extensions
				 WHERE extension_name = $1 AND is_active = true AND %s),
				false
			)
		`, filter), name).Scan(&isEnabled)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check extension status: %w", err)
	}

	// Check if installed in PostgreSQL
	installed, version := s.checkExtensionInstalledForPool(ctx, name, tenantPool)

	return &ExtensionStatusResponse{
		Name:             name,
		IsEnabled:        isEnabled,
		IsInstalled:      installed,
		InstalledVersion: version,
	}, nil
}

// EnableExtension enables a PostgreSQL extension (default tenant).
func (s *Service) EnableExtension(ctx context.Context, name string, userID *string, schema string) (*EnableExtensionResponse, error) {
	return s.EnableExtensionForTenant(ctx, name, userID, schema, nil, "")
}

// EnableExtensionForTenant enables a PostgreSQL extension for a specific tenant.
// tenantDBName is the database name for separate-DB tenants (empty for default tenant).
func (s *Service) EnableExtensionForTenant(ctx context.Context, name string, userID *string, schema string, tenantID *string, tenantDBName string) (*EnableExtensionResponse, error) {
	return s.enableExtensionRecursive(ctx, name, userID, schema, tenantID, tenantDBName, 0)
}

// enableExtensionRecursive enables an extension, automatically resolving dependencies.
// maxDepth prevents infinite recursion for circular dependencies.
func (s *Service) enableExtensionRecursive(ctx context.Context, name string, userID *string, schema string, tenantID *string, tenantDBName string, depth int) (*EnableExtensionResponse, error) {
	const maxDepth = 10
	if depth > maxDepth {
		return nil, fmt.Errorf("extension dependency chain too deep (>%d), possible circular dependency", maxDepth)
	}

	// Validate extension exists in catalog, falling back to pg_available_extensions.
	// The catalog (platform.available_extensions) may be empty on fresh installs
	// (it's populated lazily by SyncExtensions). If the extension isn't in the
	// catalog, check whether PostgreSQL itself knows about it — if so, allow
	// enabling it (the CREATE EXTENSION runs as admin regardless).
	available, err := s.getAvailableExtension(ctx, name)
	if err != nil {
		return nil, err
	}
	if available == nil {
		// Fallback: check if the extension is available in PostgreSQL directly.
		pgAvailable, pgErr := s.isExtensionAvailableInPostgres(ctx, name)
		if pgErr != nil || !pgAvailable {
			return &EnableExtensionResponse{
				Name:    name,
				Success: false,
				Message: "Extension not found in catalog or available in PostgreSQL",
			}, nil
		}
		// Extension is available in PostgreSQL but not in the catalog — proceed.
		// Use a minimal available-extension stub so downstream logic works.
		available = &AvailableExtension{Name: name, Category: categoryForExtension(name)}
	}

	// Check if already enabled
	status, err := s.GetExtensionStatusForTenant(ctx, name, tenantID, nil)
	if err != nil {
		return nil, err
	}
	if status.IsInstalled {
		err = database.WrapWithTenantAwareRole(ctx, s.db, tenantIDStr(tenantID), func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				INSERT INTO platform.enabled_extensions (extension_name, tenant_id, enabled_by, is_active)
				VALUES ($1, $2, $3, true)
				ON CONFLICT (extension_name, tenant_id) WHERE is_active = true
				DO UPDATE SET enabled_at = NOW(), enabled_by = $3, error_message = NULL
			`, name, tenantID, userID)
			return err
		})
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

	if !isValidIdentifier(name) {
		return &EnableExtensionResponse{
			Name:    name,
			Success: false,
			Message: "Invalid extension name: must contain only letters, digits, and underscores",
		}, nil
	}

	if schema != "" && schema != "public" && !isValidIdentifier(schema) {
		return &EnableExtensionResponse{
			Name:    name,
			Success: false,
			Message: "Invalid schema name: must contain only letters, digits, and underscores",
		}, nil
	}

	sql := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", quoteIdentifier(name))
	if schema != "" && schema != "public" {
		sql += fmt.Sprintf(" SCHEMA %s", quoteIdentifier(schema))
	}

	if tenantDBName != "" {
		err = s.db.ExecuteWithAdminRoleForDB(ctx, tenantDBName, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, sql)
			return err
		})
	} else {
		err = s.db.ExecuteWithAdminRole(ctx, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, sql)
			return err
		})
	}
	if err != nil {
		// Check if the error is a missing dependency
		errMsg := err.Error()
		if matches := requiredExtRegex.FindStringSubmatch(errMsg); len(matches) == 2 {
			depName := matches[1]
			log.Info().Str("extension", name).Str("dependency", depName).Msg("Auto-resolving extension dependency")

			depResp, depErr := s.enableExtensionRecursive(ctx, depName, userID, "", tenantID, tenantDBName, depth+1)
			if depErr != nil {
				return nil, fmt.Errorf("failed to enable dependency %q: %w", depName, depErr)
			}
			if !depResp.Success {
				return &EnableExtensionResponse{
					Name:    name,
					Success: false,
					Message: fmt.Sprintf("Failed to enable dependency %q: %s", depName, depResp.Message),
				}, nil
			}

			// Retry enabling the original extension
			if tenantDBName != "" {
				err = s.db.ExecuteWithAdminRoleForDB(ctx, tenantDBName, func(tx pgx.Tx) error {
					_, err := tx.Exec(ctx, sql)
					return err
				})
			} else {
				err = s.db.ExecuteWithAdminRole(ctx, func(tx pgx.Tx) error {
					_, err := tx.Exec(ctx, sql)
					return err
				})
			}
		}

		if err != nil {
			log.Error().Err(err).Str("extension", name).Msg("Failed to create extension")
			s.recordExtensionErrorForTenant(ctx, name, userID, err.Error(), tenantID)
			return &EnableExtensionResponse{
				Name:    name,
				Success: false,
				Message: fmt.Sprintf("Failed to enable extension: %v", err),
			}, nil
		}
	}

	_, version := s.checkExtensionInstalledForPool(ctx, name, nil)

	err = database.WrapWithTenantAwareRole(ctx, s.db, tenantIDStr(tenantID), func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO platform.enabled_extensions (extension_name, tenant_id, enabled_by, is_active)
			VALUES ($1, $2, $3, true)
			ON CONFLICT (extension_name, tenant_id) WHERE is_active = true
			DO UPDATE SET enabled_at = NOW(), enabled_by = $3, error_message = NULL
		`, name, tenantID, userID)
		return err
	})
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

// DisableExtension disables a PostgreSQL extension (default tenant).
func (s *Service) DisableExtension(ctx context.Context, name string, userID *string) (*DisableExtensionResponse, error) {
	return s.DisableExtensionForTenant(ctx, name, userID, nil, "")
}

// DisableExtensionForTenant disables a PostgreSQL extension for a specific tenant.
func (s *Service) DisableExtensionForTenant(ctx context.Context, name string, userID *string, tenantID *string, tenantDBName string) (*DisableExtensionResponse, error) {
	// Validate extension name is a safe identifier (defense in depth)
	if !isValidIdentifier(name) {
		return &DisableExtensionResponse{
			Name:    name,
			Success: false,
			Message: "Invalid extension name: must contain only letters, digits, and underscores",
		}, nil
	}

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
	status, err := s.GetExtensionStatusForTenant(ctx, name, tenantID, nil)
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

	// Build DROP EXTENSION statement
	sql := fmt.Sprintf("DROP EXTENSION IF EXISTS %s CASCADE", quoteIdentifier(name))

	// Execute on the appropriate database
	if tenantDBName != "" {
		err = s.db.ExecuteWithAdminRoleForDB(ctx, tenantDBName, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, sql)
			return err
		})
	} else {
		err = s.db.ExecuteWithAdminRole(ctx, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, sql)
			return err
		})
	}
	if err != nil {
		log.Error().Err(err).Str("extension", name).Msg("Failed to drop extension")
		return &DisableExtensionResponse{
			Name:    name,
			Success: false,
			Message: fmt.Sprintf("Failed to disable extension: %v", err),
		}, nil
	}

	// Update enabled_extensions table
	filter := mustTenantFilter(tenantID, 2)
	err = database.WrapWithTenantAwareRole(ctx, s.db, tenantIDStr(tenantID), func(tx pgx.Tx) error {
		if tenantID != nil {
			_, err := tx.Exec(ctx, fmt.Sprintf(`
				UPDATE platform.enabled_extensions
				SET is_active = false, disabled_at = NOW(), disabled_by = $3
				WHERE extension_name = $1 AND is_active = true AND %s
			`, filter), name, *tenantID, userID)
			return err
		}
		_, err := tx.Exec(ctx, fmt.Sprintf(`
			UPDATE platform.enabled_extensions
			SET is_active = false, disabled_at = NOW(), disabled_by = $2
			WHERE extension_name = $1 AND is_active = true AND %s
		`, filter), name, userID)
		return err
	})
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

// checkExtensionInstalledForPool checks if an extension is installed using the given pool.
// When pool is nil, uses the main database pool.
func (s *Service) checkExtensionInstalledForPool(ctx context.Context, name string, pool *pgxpool.Pool) (bool, string) {
	if pool == nil {
		pool = s.db.Pool()
	}
	var version *string
	err := pool.QueryRow(ctx, `
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
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, name, display_name, COALESCE(description, ''), category,
			       is_core, requires_restart, COALESCE(documentation_url, ''), created_at, updated_at
			FROM platform.available_extensions
			WHERE name = $1
		`, name).Scan(
			&ext.ID, &ext.Name, &ext.DisplayName, &ext.Description, &ext.Category,
			&ext.IsCore, &ext.RequiresRestart, &ext.DocumentationURL, &ext.CreatedAt, &ext.UpdatedAt,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get extension: %w", err)
	}
	return &ext, nil
}

// SyncExtensionCatalog populates platform.available_extensions from
// pg_available_extensions, upserting any missing rows with sensible category
// defaults. Returns the number of extensions synced. Called on startup and via
// the POST /api/v1/admin/extensions/sync endpoint.
func (s *Service) SyncExtensionCatalog(ctx context.Context) (int, error) {
	// Query all available extensions from PostgreSQL.
	type pgExt struct {
		Name    string
		Version string
		Comment string
	}
	var exts []pgExt
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT name, COALESCE(default_version, ''), COALESCE(comment, '')
			FROM pg_available_extensions
			ORDER BY name
		`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e pgExt
			if err := rows.Scan(&e.Name, &e.Version, &e.Comment); err != nil {
				return err
			}
			exts = append(exts, e)
		}
		return rows.Err()
	})
	if err != nil {
		return 0, fmt.Errorf("failed to query pg_available_extensions: %w", err)
	}

	// Upsert each into the catalog (skip rows that already exist — don't overwrite
	// admin-customized metadata).
	synced := 0
	for _, e := range exts {
		category := categoryForExtension(e.Name)
		displayName := strings.Title(strings.ReplaceAll(e.Name, "_", " "))
		var affected int64
		err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, `
				INSERT INTO platform.available_extensions (name, display_name, description, category)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (name) DO NOTHING
			`, e.Name, displayName, e.Comment, category)
			if err != nil {
				return err
			}
			affected = tag.RowsAffected()
			return nil
		})
		if err != nil {
			log.Warn().Err(err).Str("extension", e.Name).Msg("Failed to upsert extension in catalog")
			continue
		}
		if affected > 0 {
			synced++
		}
	}

	log.Info().Int("synced", synced).Int("total_available", len(exts)).Msg("Extension catalog synced from PostgreSQL")
	return synced, nil
}

// isExtensionAvailableInPostgres checks whether an extension is installable
// on the main database by querying pg_available_extensions. This is the
// fallback when the extension isn't in the platform.available_extensions catalog.
func (s *Service) isExtensionAvailableInPostgres(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM pg_available_extensions WHERE name = $1)
		`, name).Scan(&exists)
	})
	return exists, err
}

// categoryForExtension returns a sensible default category for known extensions.
// Used when enabling an extension that isn't yet in the catalog.
func categoryForExtension(name string) string {
	switch strings.ToLower(name) {
	case "postgis", "postgis_topology", "postgis_raster", "postgis_sfcgal",
		"address_standardizer", "address_standardizer_data_us", "pgrouting":
		return "geospatial"
	case "vector", "vectorscale", "ai":
		return "ai_ml"
	case "pg_trgm", "unaccent", "fuzzystrmatch":
		return "text_search"
	case "pg_stat_statements", "pg_stat_monitor", "pg_buffercache":
		return "monitoring"
	case "timescaledb", "pg_cron", "pg_partman":
		return "scheduling"
	case "pgcrypto", "pgjwt":
		return "data_types"
	case "uuid-ossp", "btree_gin", "btree_gist", "intarray", "citext":
		return "indexing"
	case "postgres_fdw", "dblink", "file_fdw":
		return "foreign_data"
	case "pg_repack", "pg_squeeze":
		return "maintenance"
	default:
		return "utilities"
	}
}

// recordExtensionErrorForTenant records an error for a tenant's extension operation
func (s *Service) recordExtensionErrorForTenant(ctx context.Context, name string, userID *string, errorMsg string, tenantID *string) {
	var tID string
	if tenantID != nil {
		tID = *tenantID
	}
	err := database.WrapWithServiceRoleAndTenant(ctx, s.db, tID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO platform.enabled_extensions (extension_name, tenant_id, enabled_by, is_active, error_message)
			VALUES ($1, $2, $3, false, $4)
			ON CONFLICT (extension_name, tenant_id) WHERE is_active = true
			DO UPDATE SET error_message = $4, updated_at = NOW()
		`, name, tenantID, userID, errorMsg)
		return err
	})
	if err != nil {
		log.Warn().Err(err).Str("extension", name).Msg("Failed to record extension error")
	}
}

// InitializeCoreExtensions ensures core extensions are enabled on startup (default tenant)
func (s *Service) InitializeCoreExtensions(ctx context.Context) error {
	return s.InitializeCoreExtensionsForTenant(ctx, nil, "")
}

// InitializeCoreExtensionsForTenant ensures core extensions are enabled for a tenant's database.
// tenantDBName is the database name for separate-DB tenants (empty for default tenant).
func (s *Service) InitializeCoreExtensionsForTenant(ctx context.Context, tenantID *string, tenantDBName string) error {
	var coreExtensions []string
	var tID string
	if tenantID != nil {
		tID = *tenantID
	}
	err := database.WrapWithServiceRoleAndTenant(ctx, s.db, tID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT name FROM platform.available_extensions WHERE is_core = true
		`)
		if err != nil {
			return fmt.Errorf("failed to query core extensions: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return fmt.Errorf("failed to scan core extension: %w", err)
			}
			coreExtensions = append(coreExtensions, name)
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, name := range coreExtensions {
		_, err := s.EnableExtensionForTenant(ctx, name, nil, "", tenantID, tenantDBName)
		if err != nil {
			log.Error().Err(err).Str("extension", name).Msg("Failed to enable core extension")
		}
	}

	return nil
}
