//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/migrations"
	"github.com/nimbleflux/fluxbase/internal/testutil/e2e"
)

// TestMigrationsStorage_CreateAndGet verifies creating and retrieving migrations
func TestMigrationsStorage_CreateAndGet(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_create_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_test_migration",
		Description: strPtr("Test migration"),
		UpSQL:       "CREATE TABLE test (id INTEGER)",
		DownSQL:     strPtr("DROP TABLE test"),
	}

	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Verify ID and timestamps were set
	assert.NotEqual(t, uuid.Nil, migration.ID)
	assert.Greater(t, migration.Version, 0)
	assert.Equal(t, "pending", migration.Status)
	assert.False(t, migration.CreatedAt.IsZero())
	assert.False(t, migration.UpdatedAt.IsZero())
	assert.Nil(t, migration.AppliedAt)
	assert.Nil(t, migration.RolledBackAt)

	// Retrieve migration
	retrieved, err := storage.GetMigration(ctx, namespace, "001_test_migration")
	require.NoError(t, err)

	assert.Equal(t, migration.ID, retrieved.ID)
	assert.Equal(t, namespace, retrieved.Namespace)
	assert.Equal(t, "001_test_migration", retrieved.Name)
	assert.Equal(t, "Test migration", *retrieved.Description)
	assert.Equal(t, "CREATE TABLE test (id INTEGER)", retrieved.UpSQL)
	assert.Equal(t, "DROP TABLE test", *retrieved.DownSQL)
	assert.Equal(t, "pending", retrieved.Status)
}

// TestMigrationsStorage_ListMigrations_FiltersByStatus verifies listing migrations with status filter
func TestMigrationsStorage_ListMigrations_FiltersByStatus(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_list_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create migrations with different statuses
	migrationSpecs := []struct {
		name        string
		upSQL       string
		statusAfter string // Will be set after creation
	}{
		{
			name:  "001_first",
			upSQL: "CREATE TABLE list_test_1 (id INTEGER)",
		},
		{
			name:  "002_second",
			upSQL: "CREATE TABLE list_test_2 (id INTEGER)",
		},
		{
			name:  "003_third",
			upSQL: "CREATE TABLE list_test_3 (id INTEGER)",
		},
	}

	for _, m := range migrationSpecs {
		migration := &migrations.Migration{
			Namespace:   namespace,
			Name:        m.name,
			Description: strPtr("List test"),
			UpSQL:       m.upSQL,
			DownSQL:     nil,
		}
		err := storage.CreateMigration(ctx, migration)
		require.NoError(t, err)
	}

	// List all migrations
	all, err := storage.ListMigrations(ctx, namespace, nil)
	require.NoError(t, err)
	assert.Len(t, all, 3, "Should have 3 migrations")

	// Verify sorted by name
	assert.Equal(t, "001_first", all[0].Name)
	assert.Equal(t, "002_second", all[1].Name)
	assert.Equal(t, "003_third", all[2].Name)

	// List pending migrations
	pending := "pending"
	pendingMigrations, err := storage.ListMigrations(ctx, namespace, &pending)
	require.NoError(t, err)
	assert.Len(t, pendingMigrations, 3, "All should be pending initially")
}

// TestMigrationsStorage_UpdateMigration_ModifiesPendingMigrations verifies updating pending migrations
func TestMigrationsStorage_UpdateMigration_ModifiesPendingMigrations(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_update_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create pending migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_update_test",
		Description: strPtr("Original description"),
		UpSQL:       "CREATE TABLE update_test (id INTEGER)",
		DownSQL:     strPtr("DROP TABLE update_test"),
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Update migration
	newDesc := "Updated description"
	updates := map[string]interface{}{
		"description": newDesc,
		"up_sql":      "CREATE TABLE update_test (id SERIAL PRIMARY KEY, name TEXT)",
	}
	err = storage.UpdateMigration(ctx, namespace, "001_update_test", updates)
	require.NoError(t, err)

	// Verify updates
	retrieved, err := storage.GetMigration(ctx, namespace, "001_update_test")
	require.NoError(t, err)
	assert.Equal(t, newDesc, *retrieved.Description)
	assert.Contains(t, retrieved.UpSQL, "name TEXT")
	assert.Greater(t, retrieved.UpdatedAt, migration.CreatedAt, "UpdatedAt should be after CreatedAt")
}

// TestMigrationsStorage_UpdateMigration_CannotModifyApplied verifies applied migrations cannot be updated
func TestMigrationsStorage_UpdateMigration_CannotModifyApplied(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_update_applied_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create and apply migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_applied",
		Description: strPtr("Applied migration"),
		UpSQL:       "CREATE TABLE applied_test (id INTEGER)",
		DownSQL:     strPtr("DROP TABLE applied_test"),
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Mark as applied
	err = storage.UpdateMigrationStatus(ctx, migration.ID, "applied", nil)
	require.NoError(t, err)

	// Try to update applied migration
	err = storage.UpdateMigration(ctx, namespace, "001_applied", map[string]interface{}{
		"description": "New description",
	})
	require.Error(t, err, "Should not update applied migration")
	assert.Contains(t, err.Error(), "not found or already applied")
}

// TestMigrationsStorage_DeleteMigration_RemovesPendingMigration verifies deleting pending migrations
func TestMigrationsStorage_DeleteMigration_RemovesPendingMigration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_delete_pending_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create pending migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_deletable",
		Description: strPtr("Deletable migration"),
		UpSQL:       "CREATE TABLE deletable (id INTEGER)",
		DownSQL:     nil,
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Verify exists
	_, err = storage.GetMigration(ctx, namespace, "001_deletable")
	require.NoError(t, err)

	// Delete migration
	err = storage.DeleteMigration(ctx, namespace, "001_deletable")
	require.NoError(t, err)

	// Verify deleted
	_, err = storage.GetMigration(ctx, namespace, "001_deletable")
	require.Error(t, err, "Migration should be deleted")
}

// TestMigrationsStorage_UpdateMigrationStatus_TransitionsStatus verifies status transitions
func TestMigrationsStorage_UpdateMigrationStatus_TransitionsStatus(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_status_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_status_test",
		Description: strPtr("Status test"),
		UpSQL:       "CREATE TABLE status_test (id INTEGER)",
		DownSQL:     strPtr("DROP TABLE status_test"),
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Transition to applied (use nil for executedBy since we don't have a valid user)
	err = storage.UpdateMigrationStatus(ctx, migration.ID, "applied", nil)
	require.NoError(t, err)

	// Verify status
	retrieved, err := storage.GetMigration(ctx, namespace, "001_status_test")
	require.NoError(t, err)
	assert.Equal(t, "applied", retrieved.Status)
	assert.NotNil(t, retrieved.AppliedAt)
	assert.Nil(t, retrieved.AppliedBy, "AppliedBy should be nil when nil is passed")

	// Transition to rolled_back
	err = storage.UpdateMigrationStatus(ctx, migration.ID, "rolled_back", nil)
	require.NoError(t, err)

	// Verify status
	retrieved, err = storage.GetMigration(ctx, namespace, "001_status_test")
	require.NoError(t, err)
	assert.Equal(t, "rolled_back", retrieved.Status)
	assert.NotNil(t, retrieved.RolledBackAt)
}

// TestMigrationsStorage_LogExecution_CreatesAuditTrail verifies execution logging
func TestMigrationsStorage_LogExecution_CreatesAuditTrail(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_log_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_log_test",
		Description: strPtr("Log test"),
		UpSQL:       "CREATE TABLE log_test (id INTEGER)",
		DownSQL:     strPtr("DROP TABLE log_test"),
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Log execution (use nil for executedBy since we don't have a valid user)
	durationMs := 150
	log := &migrations.ExecutionLog{
		MigrationID: migration.ID,
		Action:      "apply",
		Status:      "success",
		DurationMs:  &durationMs,
		ExecutedBy:  nil,
	}

	err = storage.LogExecution(ctx, log)
	require.NoError(t, err)

	// Verify log was created
	assert.NotEqual(t, uuid.Nil, log.ID)
	assert.False(t, log.ExecutedAt.IsZero())

	// Retrieve logs
	logs, err := storage.GetExecutionLogs(ctx, migration.ID, 10)
	require.NoError(t, err)
	require.Len(t, logs, 1)

	retrievedLog := logs[0]
	assert.Equal(t, migration.ID, retrievedLog.MigrationID)
	assert.Equal(t, "apply", retrievedLog.Action)
	assert.Equal(t, "success", retrievedLog.Status)
	assert.Equal(t, 150, *retrievedLog.DurationMs)
	assert.Nil(t, retrievedLog.ErrorMessage)
	assert.Nil(t, retrievedLog.ExecutedBy, "ExecutedBy should be nil when nil is passed")
}

// TestMigrationsStorage_GetExecutionLogs_ReturnsMostRecent verifies log ordering and limiting
func TestMigrationsStorage_GetExecutionLogs_ReturnsMostRecent(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_log_order_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_log_order",
		Description: strPtr("Log order test"),
		UpSQL:       "CREATE TABLE log_order_test (id INTEGER)",
		DownSQL:     strPtr("DROP TABLE log_order_test"),
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Create multiple execution logs
	for i := 1; i <= 5; i++ {
		durationMs := i * 100
		err := storage.LogExecution(ctx, &migrations.ExecutionLog{
			MigrationID: migration.ID,
			Action:      "apply",
			Status:      "success",
			DurationMs:  &durationMs,
		})
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Retrieve logs with limit
	logs, err := storage.GetExecutionLogs(ctx, migration.ID, 3)
	require.NoError(t, err)
	assert.Len(t, logs, 3, "Should respect limit")

	// Verify most recent first (should be descending by executed_at)
	// The most recent should have the highest duration
	assert.Equal(t, 500, *logs[0].DurationMs)
	assert.Equal(t, 400, *logs[1].DurationMs)
	assert.Equal(t, 300, *logs[2].DurationMs)
}

// TestMigrationsStorage_LogExecutionWithError captures error messages
func TestMigrationsStorage_LogExecutionWithError(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_error_log_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_error_log",
		Description: strPtr("Error log test"),
		UpSQL:       "INVALID SQL",
		DownSQL:     nil,
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Log failed execution
	errorMsg := "syntax error at or near INVALID"
	durationMs := 50
	log := &migrations.ExecutionLog{
		MigrationID:  migration.ID,
		Action:       "apply",
		Status:       "failed",
		DurationMs:   &durationMs,
		ErrorMessage: &errorMsg,
	}

	err = storage.LogExecution(ctx, log)
	require.NoError(t, err)

	// Retrieve and verify
	logs, err := storage.GetExecutionLogs(ctx, migration.ID, 10)
	require.NoError(t, err)
	require.Len(t, logs, 1)

	retrievedLog := logs[0]
	assert.Equal(t, "failed", retrievedLog.Status)
	assert.NotNil(t, retrievedLog.ErrorMessage)
	assert.Contains(t, *retrievedLog.ErrorMessage, "syntax error")
}

// TestMigrationsStorage_UniqueNamespaceNameConstraint enforces unique constraint
func TestMigrationsStorage_UniqueNamespaceNameConstraint(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_unique_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create first migration
	migration1 := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_duplicate",
		Description: strPtr("First migration"),
		UpSQL:       "CREATE TABLE unique_test (id INTEGER)",
		DownSQL:     nil,
	}
	err := storage.CreateMigration(ctx, migration1)
	require.NoError(t, err)

	// Try to create duplicate migration with same namespace and name
	migration2 := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_duplicate",
		Description: strPtr("Duplicate migration"),
		UpSQL:       "CREATE TABLE another_table (id INTEGER)",
		DownSQL:     nil,
	}
	err = storage.CreateMigration(ctx, migration2)
	require.Error(t, err, "Should fail with unique constraint violation")
	assert.Contains(t, err.Error(), "unique")
}

// TestMigrationsStorage_CascadeDelete deletes execution logs when migration is deleted
func TestMigrationsStorage_CascadeDelete(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_cascade_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_cascade",
		Description: strPtr("Cascade delete test"),
		UpSQL:       "CREATE TABLE cascade_test (id INTEGER)",
		DownSQL:     strPtr("DROP TABLE cascade_test"),
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Create execution logs
	for i := 0; i < 3; i++ {
		durationMs := (i + 1) * 100
		err := storage.LogExecution(ctx, &migrations.ExecutionLog{
			MigrationID: migration.ID,
			Action:      "apply",
			Status:      "success",
			DurationMs:  &durationMs,
		})
		require.NoError(t, err)
	}

	// Verify logs exist
	logs, err := storage.GetExecutionLogs(ctx, migration.ID, 10)
	require.NoError(t, err)
	assert.Len(t, logs, 3)

	// Delete migration (should cascade to logs)
	err = storage.DeleteMigration(ctx, namespace, "001_cascade")
	require.NoError(t, err)

	// Verify logs are deleted
	logs, err = storage.GetExecutionLogs(ctx, migration.ID, 10)
	require.NoError(t, err)
	assert.Len(t, logs, 0, "Logs should be cascade deleted")
}

// TestMigrationsStorage_VersionAutoIncrement auto-increments version numbers
func TestMigrationsStorage_VersionAutoIncrement(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_version_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create multiple migrations
	versions := []int{}
	for i := 1; i <= 3; i++ {
		migration := &migrations.Migration{
			Namespace:   namespace,
			Name:        fmt.Sprintf("%03d_migration", i),
			Description: strPtr(fmt.Sprintf("Migration %d", i)),
			UpSQL:       fmt.Sprintf("CREATE TABLE version_test_%d (id INTEGER)", i),
			DownSQL:     nil,
		}
		err := storage.CreateMigration(ctx, migration)
		require.NoError(t, err)
		versions = append(versions, migration.Version)
	}

	// Verify all have default version of 1
	// The version field is just a counter for how many times a migration has been applied
	// It defaults to 1 and doesn't auto-increment
	assert.Equal(t, []int{1, 1, 1}, versions, "All migrations should start with version 1")
}

// TestMigrationsStorage_UpdateMigration_ResetStatusToPending allows resetting failed migrations
func TestMigrationsStorage_UpdateMigration_ResetStatusToPending(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_reset_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_reset",
		Description: strPtr("Reset test"),
		UpSQL:       "CREATE TABLE reset_test (id INTEGER)",
		DownSQL:     nil,
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Mark as failed
	err = storage.UpdateMigrationStatus(ctx, migration.ID, "failed", nil)
	require.NoError(t, err)

	// Verify failed status
	retrieved, err := storage.GetMigration(ctx, namespace, "001_reset")
	require.NoError(t, err)
	assert.Equal(t, "failed", retrieved.Status)

	// Reset to pending
	err = storage.UpdateMigration(ctx, namespace, "001_reset", map[string]interface{}{
		"status": "pending",
	})
	require.NoError(t, err, "Should allow resetting failed migration to pending")

	// Verify reset
	retrieved, err = storage.GetMigration(ctx, namespace, "001_reset")
	require.NoError(t, err)
	assert.Equal(t, "pending", retrieved.Status)
}

// TestMigrationsStorage_UpdateMigration_InvalidStatusUpdate rejects invalid status updates
func TestMigrationsStorage_UpdateMigration_InvalidStatusUpdate(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_invalid_status_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_invalid",
		Description: strPtr("Invalid status test"),
		UpSQL:       "CREATE TABLE invalid_test (id INTEGER)",
		DownSQL:     nil,
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Try to set invalid status
	err = storage.UpdateMigration(ctx, namespace, "001_invalid", map[string]interface{}{
		"status": "applied", // Can only reset to pending, not set to applied
	})
	require.Error(t, err, "Should reject status other than 'pending'")
	assert.Contains(t, err.Error(), "can only reset status")
}
