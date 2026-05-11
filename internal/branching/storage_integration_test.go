//go:build integration

package branching

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/test/dbhelpers"
)

const storageTestEncryptionKey = "test-encryption-key-must-be-32-chars!"

// TestStorage_CreateBranch_Integration tests branch creation.
func TestStorage_CreateBranch_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(storageTestEncryptionKey))

	t.Run("creates branch successfully", func(t *testing.T) {
		branch := &Branch{
			ID:            uuid.New(),
			Name:          fmt.Sprintf("test-branch-%s", uuid.New().String()),
			Slug:          fmt.Sprintf("test-branch-%s", uuid.New().String()),
			DatabaseName:  "test_db",
			Status:        BranchStatusReady,
			Type:          BranchTypePreview,
			DataCloneMode: DataCloneModeSchemaOnly,
		}

		err := storage.CreateBranch(context.Background(), branch)
		require.NoError(t, err)
		assert.NotZero(t, branch.CreatedAt)
		assert.NotZero(t, branch.UpdatedAt)

		// Clean up
		testCtx.ExecuteSQL(`DELETE FROM branching.branches WHERE id = $1`, branch.ID)
	})

	t.Run("generates ID if not provided", func(t *testing.T) {
		branch := &Branch{
			Name:          fmt.Sprintf("auto-id-branch-%s", uuid.New().String()),
			Slug:          fmt.Sprintf("auto-id-branch-%s", uuid.New().String()),
			DatabaseName:  "auto_id_db",
			Status:        BranchStatusReady,
			Type:          BranchTypePreview,
			DataCloneMode: DataCloneModeSchemaOnly,
		}

		err := storage.CreateBranch(context.Background(), branch)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, branch.ID)

		// Clean up
		testCtx.ExecuteSQL(`DELETE FROM branching.branches WHERE id = $1`, branch.ID)
	})
}

// TestStorage_GetBranch_Integration tests branch retrieval.
func TestStorage_GetBranch_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(storageTestEncryptionKey))

	// Create test branch
	branchID := uuid.New()
	branch := &Branch{
		ID:            branchID,
		Name:          fmt.Sprintf("get-test-branch-%s", uuid.New().String()),
		Slug:          fmt.Sprintf("get-test-branch-%s", uuid.New().String()),
		DatabaseName:  "get_test_db",
		Status:        BranchStatusReady,
		Type:          BranchTypePreview,
		DataCloneMode: DataCloneModeSchemaOnly,
	}

	err := storage.CreateBranch(context.Background(), branch)
	require.NoError(t, err)

	t.Run("retrieves existing branch", func(t *testing.T) {
		retrieved, err := storage.GetBranch(context.Background(), branchID, nil)
		require.NoError(t, err)

		assert.Equal(t, branch.ID, retrieved.ID)
		assert.Equal(t, branch.Name, retrieved.Name)
		assert.Equal(t, branch.Slug, retrieved.Slug)
		assert.Equal(t, branch.Status, retrieved.Status)
		assert.Equal(t, branch.Type, retrieved.Type)
	})

	t.Run("returns error for non-existent branch", func(t *testing.T) {
		_, err := storage.GetBranch(context.Background(), uuid.New(), nil)
		assert.Error(t, err)
		assert.Equal(t, ErrBranchNotFound, err)
	})

	// Clean up
	testCtx.ExecuteSQL(`DELETE FROM branching.branches WHERE id = $1`, branchID)
}

// TestStorage_GetBranchBySlug_Integration tests slug-based retrieval.
func TestStorage_GetBranchBySlug_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(storageTestEncryptionKey))

	// Create test branch
	slug := fmt.Sprintf("slug-test-branch-%s", uuid.New().String())
	branch := &Branch{
		Name:          fmt.Sprintf("Slug Test Branch-%s", uuid.New().String()),
		Slug:          slug,
		DatabaseName:  "slug_test_db",
		Status:        BranchStatusReady,
		Type:          BranchTypePreview,
		DataCloneMode: DataCloneModeSchemaOnly,
	}

	err := storage.CreateBranch(context.Background(), branch)
	require.NoError(t, err)

	t.Run("retrieves branch by slug", func(t *testing.T) {
		retrieved, err := storage.GetBranchBySlug(context.Background(), slug, nil)
		require.NoError(t, err)

		assert.Equal(t, branch.Name, retrieved.Name)
		assert.Equal(t, slug, retrieved.Slug)
	})

	t.Run("returns error for non-existent slug", func(t *testing.T) {
		_, err := storage.GetBranchBySlug(context.Background(), "nonexistent-slug", nil)
		assert.Error(t, err)
		assert.Equal(t, ErrBranchNotFound, err)
	})

	// Clean up
	testCtx.ExecuteSQL(`DELETE FROM branching.branches WHERE slug = $1`, slug)
}

// TestStorage_UpdateBranchStatus_Integration tests status updates.
func TestStorage_UpdateBranchStatus_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(storageTestEncryptionKey))

	// Create test branch
	branch := &Branch{
		Name:          fmt.Sprintf("status-test-branch-%s", uuid.New().String()),
		Slug:          fmt.Sprintf("status-test-branch-%s", uuid.New().String()),
		DatabaseName:  "status_test_db",
		Status:        BranchStatusCreating,
		Type:          BranchTypePreview,
		DataCloneMode: DataCloneModeSchemaOnly,
	}

	err := storage.CreateBranch(context.Background(), branch)
	require.NoError(t, err)

	t.Run("updates branch status", func(t *testing.T) {
		err := storage.UpdateBranchStatus(context.Background(), branch.ID, BranchStatusReady, nil)
		require.NoError(t, err)

		// Verify update
		updated, err := storage.GetBranch(context.Background(), branch.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, BranchStatusReady, updated.Status)
	})

	t.Run("updates status with error message", func(t *testing.T) {
		errorMsg := "Something went wrong"
		err := storage.UpdateBranchStatus(context.Background(), branch.ID, BranchStatusError, &errorMsg)
		require.NoError(t, err)

		// Verify update
		updated, err := storage.GetBranch(context.Background(), branch.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, BranchStatusError, updated.Status)
		assert.Equal(t, &errorMsg, updated.ErrorMessage)
	})

	// Clean up
	testCtx.ExecuteSQL(`DELETE FROM branching.branches WHERE id = $1`, branch.ID)
}

// TestStorage_DeleteBranch_Integration tests branch deletion.
func TestStorage_DeleteBranch_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(storageTestEncryptionKey))

	// Create test branch
	branch := &Branch{
		Name:          fmt.Sprintf("delete-test-branch-%s", uuid.New().String()),
		Slug:          fmt.Sprintf("delete-test-branch-%s", uuid.New().String()),
		DatabaseName:  "delete_test_db",
		Status:        BranchStatusReady,
		Type:          BranchTypePreview,
		DataCloneMode: DataCloneModeSchemaOnly,
	}

	err := storage.CreateBranch(context.Background(), branch)
	require.NoError(t, err)

	t.Run("deletes branch successfully", func(t *testing.T) {
		err := storage.DeleteBranch(context.Background(), branch.ID, nil)
		require.NoError(t, err)

		// Verify deletion
		_, err = storage.GetBranch(context.Background(), branch.ID, nil)
		assert.Error(t, err)
		assert.Equal(t, ErrBranchNotFound, err)
	})
}

// TestStorage_CountBranches_Integration tests branch counting.
func TestStorage_CountBranches_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(storageTestEncryptionKey))

	t.Run("counts all branches", func(t *testing.T) {
		count, err := storage.CountBranches(context.Background(), ListBranchesFilter{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 0)
	})

	t.Run("counts by status", func(t *testing.T) {
		status := BranchStatusReady
		count, err := storage.CountBranches(context.Background(), ListBranchesFilter{
			Status: &status,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 0)
	})
}

// TestStorage_Transaction_Integration tests transaction helper.
func TestStorage_Transaction_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(storageTestEncryptionKey))

	t.Run("executes transaction successfully", func(t *testing.T) {
		err := storage.Transaction(context.Background(), func(tx pgx.Tx) error {
			// Execute query within transaction
			_, err := tx.Exec(context.Background(), "SELECT 1")
			return err
		})
		assert.NoError(t, err)
	})

	t.Run("rolls back on error", func(t *testing.T) {
		executed := false
		err := storage.Transaction(context.Background(), func(tx pgx.Tx) error {
			executed = true
			return assert.AnError
		})
		assert.Error(t, err)
		assert.True(t, executed)
	})
}

// TestStorage_Helpers_Integration tests helper functions.
func TestStorage_Helpers_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(storageTestEncryptionKey))

	t.Run("gets pool", func(t *testing.T) {
		pool := storage.GetPool()
		assert.NotNil(t, pool)
	})

	t.Run("sets pool", func(t *testing.T) {
		storage.SetPool(testCtx.Pool)
		pool := storage.GetPool()
		assert.Same(t, testCtx.Pool, pool)
	})
}

// TestStorage_SetBranchExpiresAt_Integration tests expiration time setting.
func TestStorage_SetBranchExpiresAt_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(storageTestEncryptionKey))

	// Create test branch
	branch := &Branch{
		Name:          fmt.Sprintf("expiry-test-branch-%s", uuid.New().String()),
		Slug:          fmt.Sprintf("expiry-test-branch-%s", uuid.New().String()),
		DatabaseName:  "expiry_test_db",
		Status:        BranchStatusReady,
		Type:          BranchTypePreview,
		DataCloneMode: DataCloneModeSchemaOnly,
	}

	err := storage.CreateBranch(context.Background(), branch)
	require.NoError(t, err)

	t.Run("sets expiration time", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)
		err := storage.SetBranchExpiresAt(context.Background(), branch.ID, &expiresAt)
		require.NoError(t, err)

		// Verify update
		updated, err := storage.GetBranch(context.Background(), branch.ID, nil)
		require.NoError(t, err)
		assert.NotNil(t, updated.ExpiresAt)
		assert.WithinDuration(t, expiresAt, *updated.ExpiresAt, time.Second)
	})

	t.Run("clears expiration time", func(t *testing.T) {
		err := storage.SetBranchExpiresAt(context.Background(), branch.ID, nil)
		require.NoError(t, err)

		// Verify update
		updated, err := storage.GetBranch(context.Background(), branch.ID, nil)
		require.NoError(t, err)
		assert.Nil(t, updated.ExpiresAt)
	})

	// Clean up
	testCtx.ExecuteSQL(`DELETE FROM branching.branches WHERE id = $1`, branch.ID)
}
