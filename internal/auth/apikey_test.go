package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
)

// setupAPIKeyTestDB creates a test database connection for API key tests
func setupAPIKeyTestDB(t *testing.T) *pgxpool.Pool {
	cfg := &config.DatabaseConfig{
		Host:            "postgres",
		Port:            5432,
		User:            "postgres",
		Password:        "postgres",
		Database:        "fluxbase_test",
		SSLMode:         "disable",
		MaxConnections:  10,
		MinConnections:  2,
		MaxConnLifetime: 1 * time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
		HealthCheck:     1 * time.Minute,
	}

	db, err := database.NewConnection(*cfg)
	require.NoError(t, err, "Failed to connect to test database")

	// Run migrations to ensure tables exist
	err = db.Migrate()
	require.NoError(t, err, "Failed to run migrations")

	return db.Pool()
}

// cleanupAPIKeys removes all test API keys and users
func cleanupAPIKeys(t *testing.T, db *pgxpool.Pool) {
	ctx := context.Background()
	// Delete API keys first (foreign key constraint)
	_, err := db.Exec(ctx, "DELETE FROM auth.api_keys WHERE name LIKE 'test-%'")
	require.NoError(t, err, "Failed to cleanup test API keys")
	// Delete test users
	_, err = db.Exec(ctx, "DELETE FROM auth.users WHERE email LIKE '%@example.com'")
	require.NoError(t, err, "Failed to cleanup test users")
}

// createTestUser creates a test user and returns the ID
func createTestUser(t *testing.T, db *pgxpool.Pool, email string) uuid.UUID {
	ctx := context.Background()
	var userID uuid.UUID
	err := db.QueryRow(ctx, `
		INSERT INTO auth.users (email, password_hash, email_verified)
		VALUES ($1, 'hashed_password', true)
		RETURNING id
	`, email).Scan(&userID)
	require.NoError(t, err, "Failed to create test user")
	return userID
}

func TestHashAPIKey(t *testing.T) {
	key1 := "fbk_test_key_123"
	key2 := "fbk_test_key_456"

	hash1 := hashAPIKey(key1)
	hash2 := hashAPIKey(key2)

	// Different keys should produce different hashes
	assert.NotEqual(t, hash1, hash2)

	// Same key should produce same hash
	hash1Again := hashAPIKey(key1)
	assert.Equal(t, hash1, hash1Again)

	// Hash should be non-empty
	assert.NotEmpty(t, hash1)
}

func TestGenerateAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAPIKeyTestDB(t)
	defer db.Close()
	cleanupAPIKeys(t, db)

	service := NewAPIKeyService(db)
	ctx := context.Background()

	t.Run("Generate API key with default values", func(t *testing.T) {
		result, err := service.GenerateAPIKey(ctx, "test-default-key", nil, nil, nil, 0, nil)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify plaintext key format
		assert.Contains(t, result.PlaintextKey, "fbk_")
		assert.Greater(t, len(result.PlaintextKey), 20)

		// Verify API key fields
		assert.Equal(t, "test-default-key", result.Name)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, 12, len(result.KeyPrefix)) // "fbk_" + 8 chars
		assert.NotEmpty(t, result.KeyHash)
		assert.NotEmpty(t, result.Scopes)
		assert.Equal(t, 100, result.RateLimitPerMinute) // default
		assert.Nil(t, result.LastUsedAt)
		assert.Nil(t, result.ExpiresAt)
		assert.Nil(t, result.RevokedAt)

		// Default scopes should be set
		expectedScopes := []string{"read:tables", "write:tables", "read:storage", "write:storage", "read:functions", "execute:functions"}
		assert.ElementsMatch(t, expectedScopes, result.Scopes)
	})

	t.Run("Generate API key with custom values", func(t *testing.T) {
		description := "Test API key with custom settings"
		// Create a test user to associate with the API key
		userID := createTestUser(t, db, "apikey-test@example.com")
		scopes := []string{"read:tables", "read:storage"}
		rateLimit := 200
		expiresAt := time.Now().Add(30 * 24 * time.Hour)

		result, err := service.GenerateAPIKey(ctx, "test-custom-key", &description, &userID, scopes, rateLimit, &expiresAt)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify custom fields
		assert.Equal(t, "test-custom-key", result.Name)
		assert.Equal(t, &description, result.Description)
		assert.Equal(t, &userID, result.UserID)
		assert.Equal(t, scopes, result.Scopes)
		assert.Equal(t, rateLimit, result.RateLimitPerMinute)
		assert.NotNil(t, result.ExpiresAt)
		assert.WithinDuration(t, expiresAt, *result.ExpiresAt, time.Second)
	})

	t.Run("Generate multiple unique API keys", func(t *testing.T) {
		key1, err := service.GenerateAPIKey(ctx, "test-unique-1", nil, nil, nil, 0, nil)
		require.NoError(t, err)

		key2, err := service.GenerateAPIKey(ctx, "test-unique-2", nil, nil, nil, 0, nil)
		require.NoError(t, err)

		// Keys should be unique
		assert.NotEqual(t, key1.PlaintextKey, key2.PlaintextKey)
		assert.NotEqual(t, key1.KeyHash, key2.KeyHash)
		assert.NotEqual(t, key1.ID, key2.ID)
	})
}

func TestValidateAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAPIKeyTestDB(t)
	defer db.Close()
	cleanupAPIKeys(t, db)

	service := NewAPIKeyService(db)
	ctx := context.Background()

	// Create a test API key
	created, err := service.GenerateAPIKey(ctx, "test-validate-key", nil, nil, nil, 0, nil)
	require.NoError(t, err)

	t.Run("Validate valid API key", func(t *testing.T) {
		apiKey, err := service.ValidateAPIKey(ctx, created.PlaintextKey)
		require.NoError(t, err)
		assert.NotNil(t, apiKey)
		assert.Equal(t, created.ID, apiKey.ID)
		assert.Equal(t, created.Name, apiKey.Name)
	})

	t.Run("Validate invalid API key", func(t *testing.T) {
		invalidKey := "fbk_invalid_key_that_does_not_exist"
		apiKey, err := service.ValidateAPIKey(ctx, invalidKey)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidAPIKey, err)
		assert.Nil(t, apiKey)
	})

	t.Run("Validate expired API key", func(t *testing.T) {
		// Create an expired key
		expiresAt := time.Now().Add(-1 * time.Hour) // expired 1 hour ago
		expired, err := service.GenerateAPIKey(ctx, "test-expired-key", nil, nil, nil, 0, &expiresAt)
		require.NoError(t, err)

		apiKey, err := service.ValidateAPIKey(ctx, expired.PlaintextKey)
		assert.Error(t, err)
		assert.Equal(t, ErrAPIKeyExpired, err)
		assert.Nil(t, apiKey)
	})

	t.Run("Validate revoked API key", func(t *testing.T) {
		// Create and then revoke a key
		revokable, err := service.GenerateAPIKey(ctx, "test-revokable-key", nil, nil, nil, 0, nil)
		require.NoError(t, err)

		err = service.RevokeAPIKey(ctx, revokable.ID)
		require.NoError(t, err)

		apiKey, err := service.ValidateAPIKey(ctx, revokable.PlaintextKey)
		assert.Error(t, err)
		assert.Equal(t, ErrAPIKeyRevoked, err)
		assert.Nil(t, apiKey)
	})

	t.Run("Validate updates last_used_at", func(t *testing.T) {
		// Create a fresh key
		fresh, err := service.GenerateAPIKey(ctx, "test-last-used", nil, nil, nil, 0, nil)
		require.NoError(t, err)
		assert.Nil(t, fresh.LastUsedAt)

		// Wait a moment to ensure timestamp difference
		time.Sleep(100 * time.Millisecond)

		// Validate the key
		validated, err := service.ValidateAPIKey(ctx, fresh.PlaintextKey)
		require.NoError(t, err)

		// Verify last_used_at was updated
		assert.NotNil(t, validated.LastUsedAt)
		assert.True(t, validated.LastUsedAt.After(fresh.CreatedAt))
	})
}

func TestListAPIKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAPIKeyTestDB(t)
	defer db.Close()
	cleanupAPIKeys(t, db)

	service := NewAPIKeyService(db)
	ctx := context.Background()

	// Create test users
	userID1 := createTestUser(t, db, "list-test1@example.com")
	userID2 := createTestUser(t, db, "list-test2@example.com")

	// Create test API keys
	_, err := service.GenerateAPIKey(ctx, "test-list-1", nil, &userID1, nil, 0, nil)
	require.NoError(t, err)
	_, err = service.GenerateAPIKey(ctx, "test-list-2", nil, &userID1, nil, 0, nil)
	require.NoError(t, err)
	_, err = service.GenerateAPIKey(ctx, "test-list-3", nil, &userID2, nil, 0, nil)
	require.NoError(t, err)

	t.Run("List all API keys", func(t *testing.T) {
		keys, err := service.ListAPIKeys(ctx, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(keys), 3)
	})

	t.Run("List API keys by user", func(t *testing.T) {
		keys, err := service.ListAPIKeys(ctx, &userID1)
		require.NoError(t, err)
		assert.Equal(t, 2, len(keys))

		// Verify all keys belong to userID1
		for _, key := range keys {
			assert.Equal(t, &userID1, key.UserID)
		}
	})

	t.Run("List API keys ordered by created_at DESC", func(t *testing.T) {
		keys, err := service.ListAPIKeys(ctx, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(keys), 3)

		// Verify descending order (most recent first)
		for i := 0; i < len(keys)-1; i++ {
			assert.True(t, keys[i].CreatedAt.After(keys[i+1].CreatedAt) || keys[i].CreatedAt.Equal(keys[i+1].CreatedAt))
		}
	})
}

func TestRevokeAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAPIKeyTestDB(t)
	defer db.Close()
	cleanupAPIKeys(t, db)

	service := NewAPIKeyService(db)
	ctx := context.Background()

	t.Run("Revoke existing API key", func(t *testing.T) {
		created, err := service.GenerateAPIKey(ctx, "test-revoke", nil, nil, nil, 0, nil)
		require.NoError(t, err)
		assert.Nil(t, created.RevokedAt)

		err = service.RevokeAPIKey(ctx, created.ID)
		require.NoError(t, err)

		// Verify revoked_at is set
		keys, err := service.ListAPIKeys(ctx, nil)
		require.NoError(t, err)

		for _, key := range keys {
			if key.ID == created.ID {
				assert.NotNil(t, key.RevokedAt)
				break
			}
		}
	})

	t.Run("Revoke non-existent API key", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := service.RevokeAPIKey(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestDeleteAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAPIKeyTestDB(t)
	defer db.Close()
	cleanupAPIKeys(t, db)

	service := NewAPIKeyService(db)
	ctx := context.Background()

	t.Run("Delete existing API key", func(t *testing.T) {
		created, err := service.GenerateAPIKey(ctx, "test-delete", nil, nil, nil, 0, nil)
		require.NoError(t, err)

		err = service.DeleteAPIKey(ctx, created.ID)
		require.NoError(t, err)

		// Verify key is deleted
		apiKey, err := service.ValidateAPIKey(ctx, created.PlaintextKey)
		assert.Error(t, err)
		assert.Nil(t, apiKey)
	})

	t.Run("Delete non-existent API key", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := service.DeleteAPIKey(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestUpdateAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAPIKeyTestDB(t)
	defer db.Close()
	cleanupAPIKeys(t, db)

	service := NewAPIKeyService(db)
	ctx := context.Background()

	created, err := service.GenerateAPIKey(ctx, "test-update", nil, nil, nil, 0, nil)
	require.NoError(t, err)

	t.Run("Update API key name", func(t *testing.T) {
		newName := "test-updated-name"
		err := service.UpdateAPIKey(ctx, created.ID, &newName, nil, nil, nil)
		require.NoError(t, err)

		// Verify update
		keys, err := service.ListAPIKeys(ctx, nil)
		require.NoError(t, err)

		for _, key := range keys {
			if key.ID == created.ID {
				assert.Equal(t, newName, key.Name)
				break
			}
		}
	})

	t.Run("Update API key scopes", func(t *testing.T) {
		newScopes := []string{"read:tables", "read:storage"}
		err := service.UpdateAPIKey(ctx, created.ID, nil, nil, newScopes, nil)
		require.NoError(t, err)

		// Verify update
		keys, err := service.ListAPIKeys(ctx, nil)
		require.NoError(t, err)

		for _, key := range keys {
			if key.ID == created.ID {
				assert.ElementsMatch(t, newScopes, key.Scopes)
				break
			}
		}
	})

	t.Run("Update API key rate limit", func(t *testing.T) {
		newRateLimit := 500
		err := service.UpdateAPIKey(ctx, created.ID, nil, nil, nil, &newRateLimit)
		require.NoError(t, err)

		// Verify update
		keys, err := service.ListAPIKeys(ctx, nil)
		require.NoError(t, err)

		for _, key := range keys {
			if key.ID == created.ID {
				assert.Equal(t, newRateLimit, key.RateLimitPerMinute)
				break
			}
		}
	})

	t.Run("Update non-existent API key", func(t *testing.T) {
		nonExistentID := uuid.New()
		newName := "should-fail"
		err := service.UpdateAPIKey(ctx, nonExistentID, &newName, nil, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestAPIKeyServiceNewAPIKeyService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupAPIKeyTestDB(t)
	defer db.Close()

	service := NewAPIKeyService(db)
	assert.NotNil(t, service)
	assert.NotNil(t, service.db)
}
