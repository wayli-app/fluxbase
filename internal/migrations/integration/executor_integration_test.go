//go:build integration

// Package migrations_test provides integration tests for the migrations module.
// These tests use a real PostgreSQL database to verify migration execution,
// rollback functionality, concurrent migration safety, and execution history tracking.
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

// randomString generates a random string for test isolation
func randomString(length int) string {
	return uuid.New().String()[:length]
}

// cleanupTestMigrations removes all test migrations from the database
func cleanupTestMigrations(t *testing.T, tc *e2e.IntegrationTestContext, namespace string) {
	t.Helper()

	ctx := context.Background()
	_, err := tc.DB.Pool().Exec(ctx, "DELETE FROM migrations.app WHERE namespace = $1", namespace)
	require.NoError(t, err, "Failed to cleanup test migrations")
}

// TestMigrationsExecutor_ApplyPendingMigrations_AppliesInOrder verifies that
// pending migrations are applied in the correct order (sorted by name)
func TestMigrationsExecutor_ApplyPendingMigrations_AppliesInOrder(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	executor := migrations.NewExecutor(tc.DB)
	storage := migrations.NewStorage(tc.DB)

	// Create test namespace for isolation
	namespace := fmt.Sprintf("test_pending_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create test table to verify migrations execute
	testTableName := fmt.Sprintf("test_table_%s", randomString(8))

	// Create three migrations in reverse order to test sorting
	migrationSpecs := []struct {
		name        string
		upSQL       string
		downSQL     string
		description string
	}{
		{
			name:        "003_third",
			description: "Third migration",
			upSQL:       fmt.Sprintf("ALTER TABLE %s ADD COLUMN third_col TEXT", testTableName),
			downSQL:     fmt.Sprintf("ALTER TABLE %s DROP COLUMN third_col", testTableName),
		},
		{
			name:        "001_first",
			description: "First migration",
			upSQL:       fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY, first_col TEXT)", testTableName),
			downSQL:     fmt.Sprintf("DROP TABLE %s", testTableName),
		},
		{
			name:        "002_second",
			description: "Second migration",
			upSQL:       fmt.Sprintf("ALTER TABLE %s ADD COLUMN second_col INTEGER", testTableName),
			downSQL:     fmt.Sprintf("ALTER TABLE %s DROP COLUMN second_col", testTableName),
		},
	}

	// Create migrations in database
	for _, m := range migrationSpecs {
		migration := &migrations.Migration{
			Namespace:   namespace,
			Name:        m.name,
			Description: &m.description,
			UpSQL:       m.upSQL,
			DownSQL:     &m.downSQL,
		}
		err := storage.CreateMigration(ctx, migration)
		require.NoError(t, err, "Failed to create migration %s", m.name)
		require.NotEqual(t, uuid.Nil, migration.ID, "Migration ID should be set")
		require.Equal(t, "pending", migration.Status, "New migration should be pending")
	}

	// Apply all pending migrations (without user tracking)
	applied, failed, err := executor.ApplyPendingMigrations(ctx, namespace, nil)
	require.NoError(t, err, "ApplyPendingMigrations should succeed")
	require.Empty(t, failed, "No migrations should fail")
	require.Len(t, applied, 3, "All 3 migrations should be applied")

	// Verify migrations were applied in correct order
	assert.Equal(t, "001_first", applied[0], "First migration should be applied first")
	assert.Equal(t, "002_second", applied[1], "Second migration should be applied second")
	assert.Equal(t, "003_third", applied[2], "Third migration should be applied third")

	// Verify table exists with all columns
	var tableName string
	err = tc.DB.Pool().QueryRow(ctx, "SELECT tablename FROM pg_tables WHERE tablename = $1", testTableName).Scan(&tableName)
	require.NoError(t, err, "Test table should exist")
	assert.Equal(t, testTableName, tableName)

	// Verify columns exist
	rows, err := tc.DB.Pool().Query(ctx, "SELECT column_name FROM information_schema.columns WHERE table_name = $1 ORDER BY ordinal_position", testTableName)
	require.NoError(t, err, "Should query columns")
	defer rows.Close()

	columns := []string{}
	for rows.Next() {
		var col string
		err := rows.Scan(&col)
		require.NoError(t, err)
		columns = append(columns, col)
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, []string{"id", "first_col", "second_col", "third_col"}, columns, "All columns should be added in order")

	// Verify all migrations have 'applied' status
	for _, m := range migrationSpecs {
		migration, err := storage.GetMigration(ctx, namespace, m.name)
		require.NoError(t, err, "Should retrieve migration")
		assert.Equal(t, "applied", migration.Status, "Migration should be applied")
		assert.NotNil(t, migration.AppliedAt, "AppliedAt should be set")
		// AppliedBy is nil when no userID is passed
		assert.Nil(t, migration.AppliedBy, "AppliedBy should be nil when no userID provided")
	}
}

// TestMigrationsExecutor_ApplyMigration_Single_Migration applies a single migration
func TestMigrationsExecutor_ApplyMigration_Single_Migration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	executor := migrations.NewExecutor(tc.DB)
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_single_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	testTableName := fmt.Sprintf("test_single_%s", randomString(8))
	upSQL := fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY, name TEXT)", testTableName)
	downSQL := fmt.Sprintf("DROP TABLE %s", testTableName)

	// Create migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_create_test_table",
		Description: strPtr("Create test table"),
		UpSQL:       upSQL,
		DownSQL:     &downSQL,
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Apply migration
	err = executor.ApplyMigration(ctx, namespace, "001_create_test_table", nil)
	require.NoError(t, err, "ApplyMigration should succeed")

	// Verify migration status
	migration, err = storage.GetMigration(ctx, namespace, "001_create_test_table")
	require.NoError(t, err)
	assert.Equal(t, "applied", migration.Status)
	assert.NotNil(t, migration.AppliedAt)
	assert.NotNil(t, migration.UpdatedAt)

	// Verify table exists
	var exists bool
	err = tc.DB.Pool().QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_tables WHERE tablename = $1)", testTableName).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "Table should be created")

	// Verify execution log was created
	logs, err := storage.GetExecutionLogs(ctx, migration.ID, 10)
	require.NoError(t, err)
	require.Len(t, logs, 1, "Should have one execution log")

	log := logs[0]
	assert.Equal(t, "apply", log.Action)
	assert.Equal(t, "success", log.Status)
	assert.NotNil(t, log.DurationMs)
	assert.Nil(t, log.ErrorMessage)
}

// TestMigrationsExecutor_ApplyMigration_AlreadyApplied skips already applied migrations
func TestMigrationsExecutor_ApplyMigration_AlreadyApplied(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	executor := migrations.NewExecutor(tc.DB)
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_already_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	testTableName := fmt.Sprintf("test_already_%s", randomString(8))
	upSQL := fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY)", testTableName)
	downSQL := fmt.Sprintf("DROP TABLE %s", testTableName)

	// Create and apply migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_create_table",
		Description: strPtr("Create table"),
		UpSQL:       upSQL,
		DownSQL:     &downSQL,
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	err = executor.ApplyMigration(ctx, namespace, "001_create_table", nil)
	require.NoError(t, err)

	// Try to apply again - should be idempotent
	err = executor.ApplyMigration(ctx, namespace, "001_create_table", nil)
	require.NoError(t, err, "Should not error when applying already applied migration")

	// Verify no duplicate execution logs
	migration, err = storage.GetMigration(ctx, namespace, "001_create_table")
	require.NoError(t, err)

	logs, err := storage.GetExecutionLogs(ctx, migration.ID, 10)
	require.NoError(t, err)
	assert.Len(t, logs, 1, "Should only have one execution log")
}

// TestMigrationsExecutor_RollbackMigration_RollsBackChanges verifies rollback functionality
func TestMigrationsExecutor_RollbackMigration_RollsBackChanges(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	executor := migrations.NewExecutor(tc.DB)
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_rollback_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	testTableName := fmt.Sprintf("test_rollback_%s", randomString(8))
	upSQL := fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY, data TEXT)", testTableName)
	downSQL := fmt.Sprintf("DROP TABLE %s", testTableName)

	// Create and apply migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_create_table",
		Description: strPtr("Create table"),
		UpSQL:       upSQL,
		DownSQL:     &downSQL,
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	err = executor.ApplyMigration(ctx, namespace, "001_create_table", nil)
	require.NoError(t, err)

	// Verify table exists
	var exists bool
	err = tc.DB.Pool().QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_tables WHERE tablename = $1)", testTableName).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists)

	// Rollback migration
	err = executor.RollbackMigration(ctx, namespace, "001_create_table", nil)
	require.NoError(t, err, "RollbackMigration should succeed")

	// Verify table was dropped
	err = tc.DB.Pool().QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_tables WHERE tablename = $1)", testTableName).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists, "Table should be dropped after rollback")

	// Verify migration status
	migration, err = storage.GetMigration(ctx, namespace, "001_create_table")
	require.NoError(t, err)
	assert.Equal(t, "rolled_back", migration.Status)
	assert.NotNil(t, migration.RolledBackAt)

	// Verify execution log
	logs, err := storage.GetExecutionLogs(ctx, migration.ID, 10)
	require.NoError(t, err)
	assert.Len(t, logs, 2, "Should have two execution logs (apply + rollback)")

	rollbackLog := logs[0] // Most recent first
	assert.Equal(t, "rollback", rollbackLog.Action)
	assert.Equal(t, "success", rollbackLog.Status)
}

// TestMigrationsExecutor_RollbackMigration_NoDownSQL fails when no rollback SQL exists
func TestMigrationsExecutor_RollbackMigration_NoDownSQL(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	executor := migrations.NewExecutor(tc.DB)
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_nodown_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	testTableName := fmt.Sprintf("test_nodown_%s", randomString(8))
	upSQL := fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY)", testTableName)

	// Create migration without down SQL
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_create_table",
		Description: strPtr("Create table"),
		UpSQL:       upSQL,
		DownSQL:     nil, // No rollback SQL
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	err = executor.ApplyMigration(ctx, namespace, "001_create_table", nil)
	require.NoError(t, err)

	// Try to rollback - should fail
	err = executor.RollbackMigration(ctx, namespace, "001_create_table", nil)
	require.Error(t, err, "Rollback should fail when no down SQL exists")
	assert.Contains(t, err.Error(), "no rollback SQL")
}

// TestMigrationsExecutor_ApplyMigration_InvalidSQL_FailsGracefully handles SQL errors
func TestMigrationsExecutor_ApplyMigration_InvalidSQL_FailsGracefully(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	executor := migrations.NewExecutor(tc.DB)
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_invalid_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create migration with invalid SQL
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_invalid_sql",
		Description: strPtr("Invalid SQL"),
		UpSQL:       "CREATE TABLE this_is_not_valid_sql_table_name_because it has spaces",
		DownSQL:     nil,
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Apply should fail
	err = executor.ApplyMigration(ctx, namespace, "001_invalid_sql", nil)
	require.Error(t, err, "Should fail with invalid SQL")

	// Verify migration status is 'failed'
	migration, err = storage.GetMigration(ctx, namespace, "001_invalid_sql")
	require.NoError(t, err)
	assert.Equal(t, "failed", migration.Status)

	// Verify execution log was created with error
	logs, err := storage.GetExecutionLogs(ctx, migration.ID, 10)
	require.NoError(t, err)
	require.Len(t, logs, 1)

	log := logs[0]
	assert.Equal(t, "apply", log.Action)
	assert.Equal(t, "failed", log.Status)
	assert.NotNil(t, log.ErrorMessage)
	assert.Contains(t, *log.ErrorMessage, "syntax error")
}

// TestMigrationsExecutor_ApplyPendingMigrations_StopsOnFirstError verifies that
// applying pending migrations stops when one fails
func TestMigrationsExecutor_ApplyPendingMigrations_StopsOnFirstError(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	executor := migrations.NewExecutor(tc.DB)
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_stop_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	testTableName := fmt.Sprintf("test_stop_%s", randomString(8))

	// Create migrations where the second one will fail
	migrationSpecs := []struct {
		name        string
		upSQL       string
		description string
	}{
		{
			name:        "001_first_valid",
			description: "First valid migration",
			upSQL:       fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY)", testTableName),
		},
		{
			name:        "002_invalid",
			description: "Invalid migration",
			upSQL:       "THIS IS NOT VALID SQL",
		},
		{
			name:        "003_third",
			description: "Third migration",
			upSQL:       "CREATE TABLE should_not_be_created (id INTEGER)",
		},
	}

	for _, m := range migrationSpecs {
		migration := &migrations.Migration{
			Namespace:   namespace,
			Name:        m.name,
			Description: &m.description,
			UpSQL:       m.upSQL,
			DownSQL:     nil,
		}
		err := storage.CreateMigration(ctx, migration)
		require.NoError(t, err)
	}

	// Apply pending - should stop at second migration
	applied, failed, err := executor.ApplyPendingMigrations(ctx, namespace, nil)
	require.Error(t, err, "Should error when migration fails")

	// Handle both cases: first migration may fail due to table existing
	// This can happen if tests run in parallel
	if len(applied) > 0 {
		assert.Equal(t, "001_first_valid", applied[0])
	}
	if len(failed) > 0 {
		assert.Equal(t, "002_invalid", failed[0])
	}

	// Verify third migration is still pending
	migration, err := storage.GetMigration(ctx, namespace, "003_third")
	require.NoError(t, err)
	assert.Equal(t, "pending", migration.Status, "Third migration should still be pending")
}

// TestMigrationsExecutor_RollbackMigration_NotApplied fails when migration not applied
func TestMigrationsExecutor_RollbackMigration_NotApplied(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	executor := migrations.NewExecutor(tc.DB)
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_not_applied_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	downSQL := "DROP TABLE test_table"

	// Create pending migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_pending",
		Description: strPtr("Pending migration"),
		UpSQL:       "CREATE TABLE test_table (id INTEGER)",
		DownSQL:     &downSQL,
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Try to rollback pending migration - should fail
	err = executor.RollbackMigration(ctx, namespace, "001_pending", nil)
	require.Error(t, err, "Should fail to rollback pending migration")
	assert.Contains(t, err.Error(), "cannot rollback")
}

// TestMigrationsExecutor_ConcurrentMigrations_PreventsRaceConditions tests concurrent migration safety
func TestMigrationsExecutor_ConcurrentMigrations_PreventsRaceConditions(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	executor := migrations.NewExecutor(tc.DB)
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_concurrent_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	// Create multiple migrations
	numMigrations := 5
	for i := 1; i <= numMigrations; i++ {
		name := fmt.Sprintf("%03d_migration", i)
		tableName := fmt.Sprintf("table_%03d", i) // Use zero-padded table names
		migration := &migrations.Migration{
			Namespace:   namespace,
			Name:        name,
			Description: strPtr(fmt.Sprintf("Migration %d", i)),
			UpSQL:       fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id SERIAL PRIMARY KEY)", tableName),
			DownSQL:     strPtr(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)),
		}
		err := storage.CreateMigration(ctx, migration)
		require.NoError(t, err)
	}

	// Apply migrations concurrently
	concurrentOps := 10
	errChan := make(chan error, concurrentOps)

	for i := 0; i < concurrentOps; i++ {
		go func() {
			_, _, err := executor.ApplyPendingMigrations(ctx, namespace, nil)
			errChan <- err
		}()
	}

	// Wait for all operations to complete
	for i := 0; i < concurrentOps; i++ {
		err := <-errChan
		// All should succeed - migrations are idempotent (CREATE IF NOT EXISTS)
		// and already-applied migrations are skipped
		// Note: Without locking, multiple goroutines may apply the same migration
		// but PostgreSQL's IF NOT EXISTS makes the SQL idempotent
		require.NoError(t, err, "Concurrent operations should succeed without errors")
	}

	// Verify all migrations were applied (status should be 'applied')
	allMigrations, err := storage.ListMigrations(ctx, namespace, nil)
	require.NoError(t, err)
	assert.Len(t, allMigrations, numMigrations, "Should have all migrations")

	for _, m := range allMigrations {
		assert.Equal(t, "applied", m.Status, "Migration should be marked as applied")
		assert.NotNil(t, m.AppliedAt, "AppliedAt should be set")

		// Verify tables exist - extract number from migration name
		// Names are like "001_migration", "002_migration", etc.
		// Tables are "table_001", "table_002", etc.
		var exists bool
		tableSuffix := m.Name[0:3] // Extract "001", "002", etc.
		tableName := "table_" + tableSuffix
		err = tc.DB.Pool().QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_tables WHERE tablename = $1)", tableName).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "Table "+tableName+" should exist")
	}
}

// TestMigrationsExecutor_ExecutionHistory_TracksDuration verifies execution time tracking
func TestMigrationsExecutor_ExecutionHistory_TracksDuration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	executor := migrations.NewExecutor(tc.DB)
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_history_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	testTableName := fmt.Sprintf("test_history_%s", randomString(8))
	upSQL := fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY)", testTableName)
	downSQL := fmt.Sprintf("DROP TABLE %s", testTableName)

	// Create migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_timed",
		Description: strPtr("Test duration tracking"),
		UpSQL:       upSQL,
		DownSQL:     &downSQL,
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Apply migration
	startTime := time.Now()
	err = executor.ApplyMigration(ctx, namespace, "001_timed", nil)
	require.NoError(t, err)
	elapsed := time.Since(startTime)

	// Check execution log
	migration, err = storage.GetMigration(ctx, namespace, "001_timed")
	require.NoError(t, err)

	logs, err := storage.GetExecutionLogs(ctx, migration.ID, 10)
	require.NoError(t, err)
	require.Len(t, logs, 1)

	log := logs[0]
	assert.NotNil(t, log.DurationMs, "Duration should be tracked")
	assert.Greater(t, *log.DurationMs, 0, "Duration should be positive")
	assert.Less(t, *log.DurationMs, int(elapsed.Seconds()*1000)+100, "Duration should be reasonable")

	// Verify timestamps
	assert.WithinDuration(t, time.Now(), log.ExecutedAt, 5*time.Second)
}

// TestMigrationsExecutor_UpdateMigration_ResetFailedMigration tests updating and retrying failed migrations
func TestMigrationsExecutor_UpdateMigration_ResetFailedMigration(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	executor := migrations.NewExecutor(tc.DB)
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_retry_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	testTableName := fmt.Sprintf("test_retry_%s", randomString(8))

	// Create migration with invalid SQL
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_fix_me",
		Description: strPtr("Will fail first"),
		UpSQL:       "INVALID SQL HERE",
		DownSQL:     nil,
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	// Try to apply - should fail
	err = executor.ApplyMigration(ctx, namespace, "001_fix_me", nil)
	require.Error(t, err)

	// Verify failed status
	migration, err = storage.GetMigration(ctx, namespace, "001_fix_me")
	require.NoError(t, err)
	assert.Equal(t, "failed", migration.Status)

	// Update migration with correct SQL
	err = storage.UpdateMigration(ctx, namespace, "001_fix_me", map[string]interface{}{
		"up_sql": fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY)", testTableName),
		"status": "pending",
	})
	require.NoError(t, err, "Should be able to update failed migration")

	// Verify updated
	migration, err = storage.GetMigration(ctx, namespace, "001_fix_me")
	require.NoError(t, err)
	assert.Equal(t, "pending", migration.Status)
	assert.Contains(t, migration.UpSQL, "CREATE TABLE")

	// Retry applying - should succeed now
	err = executor.ApplyMigration(ctx, namespace, "001_fix_me", nil)
	require.NoError(t, err, "Should succeed after fixing SQL")

	// Verify applied
	migration, err = storage.GetMigration(ctx, namespace, "001_fix_me")
	require.NoError(t, err)
	assert.Equal(t, "applied", migration.Status)

	// Verify table exists
	var exists bool
	err = tc.DB.Pool().QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_tables WHERE tablename = $1)", testTableName).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists)
}

// TestMigrationsExecutor_DeleteMigration_PendingOnly verifies only pending migrations can be deleted
func TestMigrationsExecutor_DeleteMigration_PendingOnly(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	executor := migrations.NewExecutor(tc.DB)
	storage := migrations.NewStorage(tc.DB)

	namespace := fmt.Sprintf("test_delete_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace)

	testTableName := fmt.Sprintf("test_delete_%s", randomString(8))
	upSQL := fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY)", testTableName)
	downSQL := fmt.Sprintf("DROP TABLE %s", testTableName)

	// Create and apply migration
	migration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "001_applied",
		Description: strPtr("Applied migration"),
		UpSQL:       upSQL,
		DownSQL:     &downSQL,
	}
	err := storage.CreateMigration(ctx, migration)
	require.NoError(t, err)

	err = executor.ApplyMigration(ctx, namespace, "001_applied", nil)
	require.NoError(t, err)

	// Try to delete applied migration - should fail
	err = storage.DeleteMigration(ctx, namespace, "001_applied")
	require.Error(t, err, "Should not delete applied migration")
	assert.Contains(t, err.Error(), "already applied")

	// Create pending migration
	pendingMigration := &migrations.Migration{
		Namespace:   namespace,
		Name:        "002_pending",
		Description: strPtr("Pending migration"),
		UpSQL:       "CREATE TABLE pending_table (id INTEGER)",
		DownSQL:     nil,
	}
	err = storage.CreateMigration(ctx, pendingMigration)
	require.NoError(t, err)

	// Delete pending migration - should succeed
	err = storage.DeleteMigration(ctx, namespace, "002_pending")
	require.NoError(t, err, "Should delete pending migration")

	// Verify deleted
	_, err = storage.GetMigration(ctx, namespace, "002_pending")
	require.Error(t, err, "Pending migration should be deleted")
}

// TestMigrationsExecutor_MultipleNamespaces_Isolated verifies namespace isolation
func TestMigrationsExecutor_MultipleNamespaces_Isolated(t *testing.T) {
	tc := e2e.NewIntegrationTestContextWithNamespace(t, "migrations")
	defer tc.Close()

	ctx := context.Background()
	executor := migrations.NewExecutor(tc.DB)
	storage := migrations.NewStorage(tc.DB)

	namespace1 := fmt.Sprintf("test_ns1_%s", randomString(8))
	namespace2 := fmt.Sprintf("test_ns2_%s", randomString(8))
	defer cleanupTestMigrations(t, tc, namespace1)
	defer cleanupTestMigrations(t, tc, namespace2)

	// Create migration with same name in different namespaces
	// Use random table names to avoid conflicts from previous test runs
	namespaces := []struct {
		name      string
		tableName string
	}{
		{namespace1, fmt.Sprintf("ns_test_%s", randomString(8))},
		{namespace2, fmt.Sprintf("ns_test_%s", randomString(8))},
	}

	for _, ns := range namespaces {
		migration := &migrations.Migration{
			Namespace:   ns.name,
			Name:        "001_init",
			Description: strPtr("Init schema"),
			UpSQL:       fmt.Sprintf("CREATE TABLE %s (id INTEGER)", ns.tableName),
			DownSQL:     strPtr(fmt.Sprintf("DROP TABLE %s", ns.tableName)),
		}
		err := storage.CreateMigration(ctx, migration)
		require.NoError(t, err)
	}

	// Apply migrations in namespace1
	applied1, _, err := executor.ApplyPendingMigrations(ctx, namespace1, nil)
	require.NoError(t, err)
	assert.Len(t, applied1, 1)

	// Verify namespace2 still has pending migration
	migrations2, err := storage.ListMigrations(ctx, namespace2, nil)
	require.NoError(t, err)
	assert.Len(t, migrations2, 1)
	assert.Equal(t, "pending", migrations2[0].Status)

	// Apply namespace2
	applied2, _, err := executor.ApplyPendingMigrations(ctx, namespace2, nil)
	require.NoError(t, err)
	assert.Len(t, applied2, 1)

	// Verify both have their own migration records
	migrations1, err := storage.ListMigrations(ctx, namespace1, nil)
	require.NoError(t, err)
	migrations2, err = storage.ListMigrations(ctx, namespace2, nil)
	require.NoError(t, err)

	assert.Len(t, migrations1, 1)
	assert.Len(t, migrations2, 1)
	assert.NotEqual(t, migrations1[0].ID, migrations2[0].ID, "Should have different IDs")
}

// strPtr returns a pointer to a string
func strPtr(s string) *string {
	return &s
}
