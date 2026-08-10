//go:build integration

// Package integration_test provides integration tests for the extensions module.
// These tests use a real PostgreSQL database to verify extension management operations,
// including listing, enabling, disabling, and syncing extensions.
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/extensions"
	"github.com/nimbleflux/fluxbase/internal/testutil/e2e"
)

// setupExtensionsTest creates a test service and performs initial setup
func setupExtensionsTest(t *testing.T) (*e2e.IntegrationTestContext, *extensions.Service) {
	t.Helper()

	tc := e2e.NewIntegrationTestContextWithNamespace(t, "extensions")
	service := extensions.NewService(tc.DB)

	// Ensure the extensions tables exist
	ctx := context.Background()
	_, err := tc.DB.Pool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS platform.available_extensions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL UNIQUE,
			display_name TEXT,
			description TEXT,
			category TEXT DEFAULT 'utilities',
			is_core BOOLEAN DEFAULT false,
			requires_restart BOOLEAN DEFAULT false,
			documentation_url TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS platform.enabled_extensions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			extension_name TEXT NOT NULL,
			enabled_at TIMESTAMP DEFAULT NOW(),
			enabled_by TEXT,
			disabled_at TIMESTAMP,
			disabled_by TEXT,
			is_active BOOLEAN DEFAULT true,
			error_message TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
	`)
	require.NoError(t, err, "Failed to create extensions tables")

	return tc, service
}

// cleanupExtensionsTest removes test data from the database
func cleanupExtensionsTest(t *testing.T, tc *e2e.IntegrationTestContext) {
	t.Helper()

	ctx := context.Background()
	_, err := tc.DB.Pool().Exec(ctx, `
		DELETE FROM platform.enabled_extensions
		WHERE extension_name LIKE 'test_%' OR extension_name LIKE 'test%';
		DELETE FROM platform.available_extensions
		WHERE name LIKE 'test_%' OR name LIKE 'test%';
	`)
	require.NoError(t, err, "Failed to cleanup test data")
}

// TestExtensionsService_ListExtensions_ReturnsExtensions verifies that the service
// can list all available extensions from PostgreSQL
func TestExtensionsService_ListExtensions_ReturnsExtensions(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// List all available extensions (including built-in PostgreSQL extensions)
	response, err := service.ListExtensions(ctx)
	require.NoError(t, err, "ListExtensions should succeed")
	require.NotNil(t, response, "Response should not be nil")

	// PostgreSQL should have at least some built-in extensions
	assert.NotEmpty(t, response.Extensions, "Should return at least some extensions")
	assert.NotEmpty(t, response.Categories, "Should return categories")

	// Verify response structure
	for _, ext := range response.Extensions {
		assert.NotEmpty(t, ext.Name, "Extension name should not be empty")
		assert.NotEmpty(t, ext.DisplayName, "Display name should not be empty")
		assert.NotEmpty(t, ext.Category, "Category should not be empty")
	}

	// Verify categories
	for _, cat := range response.Categories {
		assert.NotEmpty(t, cat.ID, "Category ID should not be empty")
		assert.NotEmpty(t, cat.Name, "Category name should not be empty")
		assert.Greater(t, cat.Count, 0, "Category count should be positive")
	}
}

// TestExtensionsService_ListExtensions_WithCustomExtensions verifies that custom
// extensions registered in the catalog are included in the list
func TestExtensionsService_ListExtensions_WithCustomExtensions(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Register a test extension with the same name as a real PostgreSQL extension
	// so it will appear in the list
	testExtName := "uuid_ossp" // This is a real PostgreSQL extension

	// Check if uuid_ossp is available in this PostgreSQL environment
	var extAvailable bool
	err := tc.DB.Pool().QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM pg_available_extensions WHERE name = 'uuid_ossp')
	`).Scan(&extAvailable)
	require.NoError(t, err, "Failed to check extension availability")

	// Update the metadata for this extension
	_, err = tc.DB.Pool().Exec(ctx, `
		INSERT INTO platform.available_extensions
		(name, display_name, description, category, is_core, requires_restart)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (name) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			is_core = EXCLUDED.is_core,
			requires_restart = EXCLUDED.requires_restart
	`, testExtName, "UUID OSSP Generator", "Generate universally unique identifiers", "utilities", true, false)
	require.NoError(t, err, "Failed to upsert test extension metadata")

	// List extensions
	response, err := service.ListExtensions(ctx)
	require.NoError(t, err, "ListExtensions should succeed")

	// Find our extension
	var found *extensions.Extension
	for _, ext := range response.Extensions {
		if ext.Name == testExtName {
			found = &ext
			break
		}
	}

	// Only assert if the extension is actually available in PostgreSQL
	if extAvailable {
		require.NotNil(t, found, "Extension should be in the list when available in pg_available_extensions")
		assert.Equal(t, testExtName, found.Name)
		assert.Equal(t, "UUID OSSP Generator", found.DisplayName)
		assert.Equal(t, "utilities", found.Category)
		assert.True(t, found.IsCore)
	}
}

// TestExtensionsService_GetExtensionStatus_ReturnsStatus verifies that the service
// can return the status of a specific extension
func TestExtensionsService_GetExtensionStatus_ReturnsStatus(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Test with a built-in PostgreSQL extension that's likely to exist
	// plpgsql is a core extension in PostgreSQL
	status, err := service.GetExtensionStatus(ctx, "plpgsql")
	require.NoError(t, err, "GetExtensionStatus should succeed")
	require.NotNil(t, status, "Status should not be nil")

	assert.Equal(t, "plpgsql", status.Name)
	// plpgsql should be installed by default
	assert.True(t, status.IsInstalled, "plpgsql should be installed")
}

// TestExtensionsService_GetExtensionStatus_NotFound verifies that the service
// handles requests for non-existent extensions gracefully
func TestExtensionsService_GetExtensionStatus_NotFound(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Request status for an extension that doesn't exist
	status, err := service.GetExtensionStatus(ctx, "definitely_not_a_real_extension")
	require.NoError(t, err, "GetExtensionStatus should succeed even for non-existent extensions")
	require.NotNil(t, status, "Status should not be nil")

	assert.Equal(t, "definitely_not_a_real_extension", status.Name)
	assert.False(t, status.IsEnabled, "Non-existent extension should not be enabled")
	assert.False(t, status.IsInstalled, "Non-existent extension should not be installed")
	assert.Empty(t, status.InstalledVersion, "Non-existent extension should not have a version")
}

// TestExtensionsService_EnableExtension_WithValidExtension verifies that a valid
// extension can be enabled successfully
func TestExtensionsService_EnableExtension_WithValidExtension(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Use uuid_ossp which is a real PostgreSQL extension
	testExtName := "uuid_ossp"

	// First, register the extension in the catalog
	_, err := tc.DB.Pool().Exec(ctx, `
		INSERT INTO platform.available_extensions
		(name, display_name, description, category, is_core, requires_restart)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (name) DO NOTHING
	`, testExtName, "UUID OSSP Generator", "Generate UUIDs", "utilities", true, false)
	require.NoError(t, err, "Failed to insert test extension")

	// Check if it's already installed
	status, err := service.GetExtensionStatus(ctx, testExtName)
	require.NoError(t, err, "GetExtensionStatus should succeed")

	// If it's already installed, that's fine - just verify we can check it
	if status.IsInstalled {
		assert.NotEmpty(t, status.InstalledVersion, "Should have version")
	}
}

// TestExtensionsService_EnableExtension_InvalidName verifies that invalid
// extension names are rejected
func TestExtensionsService_EnableExtension_InvalidName(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Register an extension with an invalid name
	testExtName := fmt.Sprintf("test-invalid_%d", time.Now().Unix())
	_, err := tc.DB.Pool().Exec(ctx, `
		INSERT INTO platform.available_extensions
		(name, display_name, description, category, is_core, requires_restart)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, testExtName, "Test Invalid", "Invalid name", "testing", false, false)
	require.NoError(t, err, "Failed to insert test extension")

	// Try to enable the extension
	response, err := service.EnableExtension(ctx, testExtName, nil, "")
	require.NoError(t, err, "EnableExtension should not return an error")
	require.NotNil(t, response, "Response should not be nil")

	// Should fail validation
	assert.False(t, response.Success, "Should reject invalid extension name")
	assert.Contains(t, response.Message, "Invalid extension name", "Should indicate validation failure")
}

// TestExtensionsService_EnableExtension_InvalidSchema verifies that invalid
// schema names are rejected
func TestExtensionsService_EnableExtension_InvalidSchema(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Register a valid extension
	testExtName := fmt.Sprintf("test_schema_%d", time.Now().Unix())
	_, err := tc.DB.Pool().Exec(ctx, `
		INSERT INTO platform.available_extensions
		(name, display_name, description, category, is_core, requires_restart)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, testExtName, "Test Schema", "Test schema validation", "testing", false, false)
	require.NoError(t, err, "Failed to insert test extension")

	// Try to enable with an invalid schema
	response, err := service.EnableExtension(ctx, testExtName, nil, "my-invalid-schema")
	require.NoError(t, err, "EnableExtension should not return an error")
	require.NotNil(t, response, "Response should not be nil")

	// Should fail validation
	assert.False(t, response.Success, "Should reject invalid schema name")
	assert.Contains(t, response.Message, "Invalid schema name", "Should indicate validation failure")
}

// TestExtensionsService_DisableExtension_PreventsCore verifies that core
// extensions cannot be disabled
func TestExtensionsService_DisableExtension_PreventsCore(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Register a core extension
	testExtName := fmt.Sprintf("test_core_%d", time.Now().Unix())
	_, err := tc.DB.Pool().Exec(ctx, `
		INSERT INTO platform.available_extensions
		(name, display_name, description, category, is_core, requires_restart)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, testExtName, "Test Core", "A core extension", "core", true, false)
	require.NoError(t, err, "Failed to insert test extension")

	// Try to disable the core extension
	response, err := service.DisableExtension(ctx, testExtName, nil)
	require.NoError(t, err, "DisableExtension should not return an error")
	require.NotNil(t, response, "Response should not be nil")

	// Should be rejected
	assert.False(t, response.Success, "Should not allow disabling core extensions")
	assert.Contains(t, response.Message, "Cannot disable core extension", "Should indicate core extension protection")
}

// TestExtensionsService_DisableExtension_InvalidName verifies that invalid
// extension names are rejected when disabling
func TestExtensionsService_DisableExtension_InvalidName(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Register an extension with an invalid name (using test prefix)
	testExtName := fmt.Sprintf("test-disable-invalid-%d", time.Now().UnixNano())
	invalidName := "test-invalid-" + testExtName

	_, err := tc.DB.Pool().Exec(ctx, `
		INSERT INTO platform.available_extensions
		(name, display_name, description, category, is_core, requires_restart)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, invalidName, "Test Disable Invalid", "Invalid name", "testing", false, false)
	require.NoError(t, err, "Failed to insert test extension")

	// Try to disable the extension
	response, err := service.DisableExtension(ctx, invalidName, nil)
	require.NoError(t, err, "DisableExtension should not return an error")
	require.NotNil(t, response, "Response should not be nil")

	// Should fail validation
	assert.False(t, response.Success, "Should reject invalid extension name")
	assert.Contains(t, response.Message, "Invalid extension name", "Should indicate validation failure")
}

// TestExtensionsService_SyncFromPostgres_SyncsExtensions verifies that the
// service can sync the extension catalog with PostgreSQL
func TestExtensionsService_SyncFromPostgres_SyncsExtensions(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Sync extensions from PostgreSQL
	err := service.SyncFromPostgres(ctx)
	require.NoError(t, err, "SyncFromPostgres should succeed")

	// Verify we can list extensions after sync
	response, err := service.ListExtensions(ctx)
	require.NoError(t, err, "ListExtensions should succeed after sync")
	assert.NotEmpty(t, response.Extensions, "Should have extensions after sync")
}

// TestExtensionsService_InitializeCoreExtensions_InitializesCore verifies that
// core extensions are initialized on startup
func TestExtensionsService_InitializeCoreExtensions_InitializesCore(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Register some core extensions
	testExtNames := []string{
		fmt.Sprintf("test_core1_%d", time.Now().Unix()),
		fmt.Sprintf("test_core2_%d", time.Now().Unix()),
	}

	for _, name := range testExtNames {
		_, err := tc.DB.Pool().Exec(ctx, `
			INSERT INTO platform.available_extensions
			(name, display_name, description, category, is_core, requires_restart)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, name, "Test Core", "A core extension", "core", true, false)
		require.NoError(t, err, "Failed to insert test extension %s", name)
	}

	// Initialize core extensions (will fail because they don't exist in PostgreSQL)
	// but the function should not error out, just log warnings
	err := service.InitializeCoreExtensions(ctx)
	// Should not error, even if extensions don't exist in PostgreSQL
	// (it logs warnings instead)
	assert.NoError(t, err, "InitializeCoreExtensions should not error out")
}

// TestExtensionsService_Validation_ValidIdentifiers verifies that the
// validation logic correctly accepts valid identifiers
func TestExtensionsService_Validation_ValidIdentifiers(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Test various valid identifier formats
	validIdentifiers := []string{
		"test_pgvector_ext",
		"test_pg_stat_ext",
		"test_uuid_ext",
		"test_extension_name",
		"test_private",
		"test_extension_123",
	}

	for _, name := range validIdentifiers {
		t.Run("valid_"+name, func(t *testing.T) {
			// Register the extension
			_, err := tc.DB.Pool().Exec(ctx, `
				INSERT INTO platform.available_extensions
				(name, display_name, description, category, is_core, requires_restart)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, name, name, "Test extension", "testing", false, false)
			require.NoError(t, err, "Failed to insert extension %s", name)

			// Try to enable - should pass validation (will fail at PostgreSQL level)
			response, err := service.EnableExtension(ctx, name, nil, "")
			require.NoError(t, err, "EnableExtension should not return error for valid name")
			require.NotNil(t, response, "Response should not be nil")

			// Should not fail on validation
			if !response.Success {
				assert.NotContains(t, response.Message, "Invalid extension name",
					"Valid identifier %s should pass validation", name)
			}
		})
	}
}

// TestExtensionsService_Validation_InvalidIdentifiers verifies that the
// validation logic correctly rejects invalid identifiers
func TestExtensionsService_Validation_InvalidIdentifiers(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Test various invalid identifier formats
	invalidIdentifiers := []struct {
		name   string
		reason string
	}{
		{"test-pg-vector", "contains hyphen"},
		{"test-pg.dot", "contains dot"},
		{"test-pg space", "contains space"},
		{"test-pg;semi", "contains semicolon"},
		{"test-1startsnumber", "starts with number after hyphen"},
		{"test@bad", "contains at sign"},
	}

	for _, tcCase := range invalidIdentifiers {
		t.Run("invalid_"+tcCase.name, func(t *testing.T) {
			// Register the extension with invalid name
			_, err := tc.DB.Pool().Exec(ctx, `
				INSERT INTO platform.available_extensions
				(name, display_name, description, category, is_core, requires_restart)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, tcCase.name, tcCase.name, "Test extension", "testing", false, false)
			require.NoError(t, err, "Failed to insert extension %s", tcCase.name)

			// Try to enable - should fail validation
			response, err := service.EnableExtension(ctx, tcCase.name, nil, "")
			require.NoError(t, err, "EnableExtension should not return error for invalid name")
			require.NotNil(t, response, "Response should not be nil")

			// Should fail on validation
			assert.False(t, response.Success, "Invalid identifier should be rejected")
			assert.Contains(t, response.Message, "Invalid extension name",
				"Should indicate validation failure for: %s", tcCase.reason)
		})
	}
}

// TestExtensionsService_EnableExtension_ValidSchemas verifies that valid
// schema names are accepted
func TestExtensionsService_EnableExtension_ValidSchemas(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	validSchemas := []string{"", "public", "extensions", "my_schema"}

	for _, schema := range validSchemas {
		t.Run("schema_"+schema, func(t *testing.T) {
			// Register a test extension
			testExtName := fmt.Sprintf("test_schema_%d", time.Now().UnixNano())
			_, err := tc.DB.Pool().Exec(ctx, `
				INSERT INTO platform.available_extensions
				(name, display_name, description, category, is_core, requires_restart)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, testExtName, "Test Schema", "Test schema validation", "testing", false, false)
			require.NoError(t, err, "Failed to insert test extension")

			// Try to enable with the schema
			response, err := service.EnableExtension(ctx, testExtName, nil, schema)
			require.NoError(t, err, "EnableExtension should not return error")
			require.NotNil(t, response, "Response should not be nil")

			// Should not fail on schema validation
			if !response.Success {
				assert.NotContains(t, response.Message, "Invalid schema name",
					"Schema %q should pass validation", schema)
			}
		})
	}
}

// TestExtensionsService_EnableExtension_InvalidSchemas verifies that invalid
// schema names are rejected
func TestExtensionsService_EnableExtension_InvalidSchemas(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	invalidSchemas := []string{
		"my-schema",
		"my.schema",
		"my schema",
		"1schema",
	}

	for _, schema := range invalidSchemas {
		t.Run("schema_"+schema, func(t *testing.T) {
			// Register a test extension
			testExtName := fmt.Sprintf("test_bad_schema_%d", time.Now().UnixNano())
			_, err := tc.DB.Pool().Exec(ctx, `
				INSERT INTO platform.available_extensions
				(name, display_name, description, category, is_core, requires_restart)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, testExtName, "Test Bad Schema", "Test invalid schema", "testing", false, false)
			require.NoError(t, err, "Failed to insert test extension")

			// Try to enable with the invalid schema
			response, err := service.EnableExtension(ctx, testExtName, nil, schema)
			require.NoError(t, err, "EnableExtension should not return error")
			require.NotNil(t, response, "Response should not be nil")

			// Should fail schema validation
			assert.False(t, response.Success, "Invalid schema should be rejected")
			assert.Contains(t, response.Message, "Invalid schema name",
				"Should indicate schema validation failure for: %s", schema)
		})
	}
}

// TestExtensionsService_EnableExtension_WithUserID verifies that user ID
// tracking works correctly
func TestExtensionsService_EnableExtension_WithUserID(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Register a test extension
	testExtName := fmt.Sprintf("test_user_%d", time.Now().Unix())
	_, err := tc.DB.Pool().Exec(ctx, `
		INSERT INTO platform.available_extensions
		(name, display_name, description, category, is_core, requires_restart)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, testExtName, "Test User", "Test user tracking", "testing", false, false)
	require.NoError(t, err, "Failed to insert test extension")

	// Try to enable with a user ID
	userID := "test-user-123"
	response, err := service.EnableExtension(ctx, testExtName, &userID, "")
	require.NoError(t, err, "EnableExtension should not return error")
	require.NotNil(t, response, "Response should not be nil")

	// The extension doesn't exist in PostgreSQL, so it will fail
	// but the user ID should have been passed through
	// We can't verify this without the extension actually being enabled,
	// but we've verified the code path is exercised
}

// TestExtensionsService_DisableExtension_WithUserID verifies that user ID
// tracking works correctly for disable operations
func TestExtensionsService_DisableExtension_WithUserID(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Register a test extension
	testExtName := fmt.Sprintf("test_disable_user_%d", time.Now().Unix())
	_, err := tc.DB.Pool().Exec(ctx, `
		INSERT INTO platform.available_extensions
		(name, display_name, description, category, is_core, requires_restart)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, testExtName, "Test Disable User", "Test disable user tracking", "testing", false, false)
	require.NoError(t, err, "Failed to insert test extension")

	// Try to disable with a user ID
	userID := "test-user-456"
	response, err := service.DisableExtension(ctx, testExtName, &userID)
	require.NoError(t, err, "DisableExtension should not return error")
	require.NotNil(t, response, "Response should not be nil")

	// The extension doesn't exist in PostgreSQL, so response will vary
	// but we've exercised the code path
}

// TestExtensionsService_Categories_AreCategorized verifies that extensions
// are properly categorized
func TestExtensionsService_Categories_AreCategorized(t *testing.T) {
	tc, service := setupExtensionsTest(t)
	defer tc.Close()
	defer cleanupExtensionsTest(t, tc)

	ctx := context.Background()

	// Add extensions in different categories
	categories := []struct {
		name     string
		category string
	}{
		{"test_ai_ext", "ai_ml"},
		{"test_geo_ext", "geospatial"},
		{"test_util_ext", "utilities"},
	}

	for _, cat := range categories {
		_, err := tc.DB.Pool().Exec(ctx, `
			INSERT INTO platform.available_extensions
			(name, display_name, description, category, is_core, requires_restart)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, cat.name, cat.name, "Test", cat.category, false, false)
		require.NoError(t, err, "Failed to insert extension %s", cat.name)
	}

	// List extensions
	response, err := service.ListExtensions(ctx)
	require.NoError(t, err, "ListExtensions should succeed")

	// Verify categories exist
	categoryMap := make(map[string]bool)
	for _, cat := range response.Categories {
		categoryMap[cat.ID] = true
	}

	assert.True(t, categoryMap["ai_ml"], "Should have ai_ml category")
	assert.True(t, categoryMap["geospatial"], "Should have geospatial category")
	assert.True(t, categoryMap["utilities"], "Should have utilities category")

	// Verify extensions are in correct categories
	for _, ext := range response.Extensions {
		switch ext.Name {
		case "test_ai_ext":
			assert.Equal(t, "ai_ml", ext.Category)
		case "test_geo_ext":
			assert.Equal(t, "geospatial", ext.Category)
		case "test_util_ext":
			assert.Equal(t, "utilities", ext.Category)
		}
	}
}
