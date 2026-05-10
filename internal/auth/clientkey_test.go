package auth

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
)

// getEnv retrieves an environment variable or returns the default value
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// parseInt converts a string to int, returning the default value on error
func parseInt(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 5432
	}
	return val
}

func TestMain(m *testing.M) {
	flag.Parse()

	// Skip database setup for short tests — individual tests will skip via testing.Short()
	if testing.Short() {
		os.Exit(m.Run())
	}

	// Get database config from environment variables (for CI/CD compatibility)
	// Falls back to docker-compose defaults for local development
	cfg := &config.DatabaseConfig{
		Host:            getEnv("FLUXBASE_DATABASE_HOST", "postgres"),
		Port:            parseInt(getEnv("FLUXBASE_DATABASE_PORT", "5432")),
		User:            getEnv("FLUXBASE_DATABASE_USER", "postgres"),
		Password:        getEnv("FLUXBASE_DATABASE_PASSWORD", "postgres"),
		AdminUser:       getEnv("FLUXBASE_DATABASE_ADMIN_USER", "postgres"),
		AdminPassword:   getEnv("FLUXBASE_DATABASE_ADMIN_PASSWORD", "postgres"),
		Database:        getEnv("FLUXBASE_TEST_DATABASE", "fluxbase_test"),
		SSLMode:         getEnv("FLUXBASE_DATABASE_SSLMODE", "disable"),
		MaxConnections:  10,
		MinConnections:  2,
		MaxConnLifetime: 1 * time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
		HealthCheck:     1 * time.Minute,
	}

	var err error
	sharedTestDB, err = database.NewConnection(*cfg)
	if err != nil {
		panic(err)
	}

	// Run migrations to ensure tables exist (unless skipped)
	// In CI, migrations are already applied, so skip to avoid conflicts
	if skipMigrations := getEnv("SKIP_MIGRATIONS", ""); skipMigrations != "" {
		// Migrations already applied by CI, skip
	} else {
		err = sharedTestDB.Migrate()
		if err != nil {
			sharedTestDB.Close()
			panic(err)
		}
	}

	// Run tests
	code := m.Run()

	// Clean up test data before closing
	ctx := context.Background()
	_, _ = sharedTestDB.Exec(ctx, "DELETE FROM auth.client_keys WHERE name LIKE 'test-%'")
	_, _ = sharedTestDB.Exec(ctx, "DELETE FROM auth.users WHERE email LIKE '%@example.com'")

	// Close shared connection
	sharedTestDB.Close()

	os.Exit(code)
}

var (
	sharedTestDB   *database.Connection
	sharedTestDBMu sync.Mutex
)

// getSharedTestDB returns a shared database connection for all tests in the package
// This reduces connection pool exhaustion and improves test performance
func getSharedTestDB(t *testing.T) *pgxpool.Pool {
	sharedTestDBMu.Lock()
	defer sharedTestDBMu.Unlock()

	if sharedTestDB == nil {
		cfg := &config.DatabaseConfig{
			Host:            getEnv("FLUXBASE_DATABASE_HOST", "postgres"),
			Port:            parseInt(getEnv("FLUXBASE_DATABASE_PORT", "5432")),
			User:            getEnv("FLUXBASE_DATABASE_USER", "postgres"),
			Password:        getEnv("FLUXBASE_DATABASE_PASSWORD", "postgres"),
			AdminUser:       getEnv("FLUXBASE_DATABASE_ADMIN_USER", "postgres"),
			AdminPassword:   getEnv("FLUXBASE_DATABASE_ADMIN_PASSWORD", "postgres"),
			Database:        getEnv("FLUXBASE_TEST_DATABASE", "fluxbase_test"),
			SSLMode:         getEnv("FLUXBASE_DATABASE_SSLMODE", "disable"),
			MaxConnections:  10,
			MinConnections:  2,
			MaxConnLifetime: 1 * time.Hour,
			MaxConnIdleTime: 30 * time.Minute,
			HealthCheck:     1 * time.Minute,
		}

		var err error
		sharedTestDB, err = database.NewConnection(*cfg)
		require.NoError(t, err, "Failed to connect to test database")

		// Run migrations to ensure tables exist (unless skipped)
		if skipMigrations := getEnv("SKIP_MIGRATIONS", ""); skipMigrations == "" {
			err = sharedTestDB.Migrate()
			require.NoError(t, err, "Failed to run migrations")
		}
	}

	return sharedTestDB.Pool()
}

// setupClientKeyTestDB creates a test database connection for client key tests
//
// Deprecated: Use getSharedTestDB instead for better connection pooling.
// Note: Currently unused but kept for reference.
/*
func setupClientKeyTestDB(t *testing.T) *pgxpool.Pool {
	cfg := &config.DatabaseConfig{
		Host:            "postgres",
		Port:            5432,
		User:            "postgres",
		Password:        "postgres",
		AdminUser:       "postgres",
		AdminPassword:   "postgres",
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

	// Run migrations to ensure tables exist (unless skipped)
	if getEnv("SKIP_MIGRATIONS", "") == "" {
		err = db.Migrate()
		require.NoError(t, err, "Failed to run migrations")
	}

	return db.Pool()
}
*/

// cleanupClientKeys removes all test client keys and users.
// Note: Currently unused but kept for reference.
/*
func cleanupClientKeys(t *testing.T, db *pgxpool.Pool) {
	ctx := context.Background()
	// Delete client keys first (foreign key constraint)
	_, err := db.Exec(ctx, "DELETE FROM auth.client_keys WHERE name LIKE 'test-%'")
	require.NoError(t, err, "Failed to cleanup test client keys")
	// Delete test users
	_, err = db.Exec(ctx, "DELETE FROM auth.users WHERE email LIKE '%@example.com'")
	require.NoError(t, err, "Failed to cleanup test users")
}
*/

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

func TestHashClientKey(t *testing.T) {
	key1 := "fbk_test_key_123"
	key2 := "fbk_test_key_456"

	hash1 := hashClientKey(key1)
	hash2 := hashClientKey(key2)

	// Different keys should produce different hashes
	assert.NotEqual(t, hash1, hash2)

	// Same key should produce same hash
	hash1Again := hashClientKey(key1)
	assert.Equal(t, hash1, hash1Again)

	// Hash should be non-empty
	assert.NotEmpty(t, hash1)
}

func TestGenerateClientKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := getSharedTestDB(t)
	// Note: Don't close db as it's shared across all tests in the package

	service := NewClientKeyService(database.NewConnectionWithPool(db), nil)
	ctx := context.Background()

	defaultScopes := []string{"tables:read", "tables:write", "storage:read", "storage:write", "functions:read", "functions:execute"}

	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "DELETE FROM auth.client_keys WHERE name IN ('test-default-key', 'test-custom-key', 'test-unique-1', 'test-unique-2')")
		_, _ = db.Exec(ctx, "DELETE FROM auth.users WHERE email LIKE 'clientkey-test-%@example.com'")
	})

	t.Run("Generate client key with default values", func(t *testing.T) {
		result, err := service.GenerateClientKey(ctx, "test-default-key", nil, nil, defaultScopes, 0, nil)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Verify plaintext key format
		assert.Contains(t, result.PlaintextKey, "fbk_")
		assert.Greater(t, len(result.PlaintextKey), 20)

		// Verify client key fields
		assert.Equal(t, "test-default-key", result.Name)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, 12, len(result.KeyPrefix)) // "fbk_" + 8 chars
		assert.NotEmpty(t, result.KeyHash)
		assert.NotEmpty(t, result.Scopes)
		assert.Equal(t, 100, result.RateLimitPerMinute) // default
		assert.Nil(t, result.LastUsedAt)
		assert.Nil(t, result.ExpiresAt)
		assert.Nil(t, result.RevokedAt)

		// Scopes should match what was provided
		assert.ElementsMatch(t, defaultScopes, result.Scopes)
	})

	t.Run("Generate client key with custom values", func(t *testing.T) {
		description := "Test client key with custom settings"
		// Create a test user to associate with the client key
		// Use unique email to avoid conflicts when tests run sequentially
		email := fmt.Sprintf("clientkey-test-%s@example.com", uuid.New().String()[:8])
		userID := createTestUser(t, db, email)
		scopes := []string{"tables:read", "storage:read"}
		rateLimit := 200
		expiresAt := time.Now().Add(30 * 24 * time.Hour)

		result, err := service.GenerateClientKey(ctx, "test-custom-key", &description, &userID, scopes, rateLimit, &expiresAt)
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

	t.Run("Generate multiple unique client keys", func(t *testing.T) {
		key1, err := service.GenerateClientKey(ctx, "test-unique-1", nil, nil, defaultScopes, 0, nil)
		require.NoError(t, err)

		key2, err := service.GenerateClientKey(ctx, "test-unique-2", nil, nil, defaultScopes, 0, nil)
		require.NoError(t, err)

		// Keys should be unique
		assert.NotEqual(t, key1.PlaintextKey, key2.PlaintextKey)
		assert.NotEqual(t, key1.KeyHash, key2.KeyHash)
		assert.NotEqual(t, key1.ID, key2.ID)
	})
}

func TestValidateClientKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := getSharedTestDB(t)
	// Note: Don't close db as it's shared across all tests in the package

	service := NewClientKeyService(database.NewConnectionWithPool(db), nil)
	ctx := context.Background()

	// Default scopes used in tests
	defaultScopes := []string{"tables:read", "tables:write"}

	// Create a test client key
	created, err := service.GenerateClientKey(ctx, "test-validate-key", nil, nil, defaultScopes, 0, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "DELETE FROM auth.client_keys WHERE name IN ('test-validate-key', 'test-expired-key', 'test-revokable-key', 'test-last-used')")
	})

	t.Run("Validate valid client key", func(t *testing.T) {
		clientKey, err := service.ValidateClientKey(ctx, created.PlaintextKey)
		require.NoError(t, err)
		assert.NotNil(t, clientKey)
		assert.Equal(t, created.ID, clientKey.ID)
		assert.Equal(t, created.Name, clientKey.Name)
	})

	t.Run("Validate invalid client key", func(t *testing.T) {
		invalidKey := "fbk_invalid_key_that_does_not_exist"
		clientKey, err := service.ValidateClientKey(ctx, invalidKey)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidClientKey, err)
		assert.Nil(t, clientKey)
	})

	t.Run("Validate expired client key", func(t *testing.T) {
		// Create an expired key
		expiresAt := time.Now().Add(-1 * time.Hour) // expired 1 hour ago
		expired, err := service.GenerateClientKey(ctx, "test-expired-key", nil, nil, defaultScopes, 0, &expiresAt)
		require.NoError(t, err)

		clientKey, err := service.ValidateClientKey(ctx, expired.PlaintextKey)
		assert.Error(t, err)
		assert.Equal(t, ErrClientKeyExpired, err)
		assert.Nil(t, clientKey)
	})

	t.Run("Validate revoked client key", func(t *testing.T) {
		// Create and then revoke a key
		revokable, err := service.GenerateClientKey(ctx, "test-revokable-key", nil, nil, defaultScopes, 0, nil)
		require.NoError(t, err)

		err = service.RevokeClientKey(ctx, revokable.ID)
		require.NoError(t, err)

		clientKey, err := service.ValidateClientKey(ctx, revokable.PlaintextKey)
		assert.Error(t, err)
		assert.Equal(t, ErrClientKeyRevoked, err)
		assert.Nil(t, clientKey)
	})

	t.Run("Validate updates last_used_at", func(t *testing.T) {
		// Create a fresh key
		fresh, err := service.GenerateClientKey(ctx, "test-last-used", nil, nil, defaultScopes, 0, nil)
		require.NoError(t, err)
		assert.Nil(t, fresh.LastUsedAt)

		// Wait a moment to ensure timestamp difference
		time.Sleep(100 * time.Millisecond)

		// Validate the key
		validated, err := service.ValidateClientKey(ctx, fresh.PlaintextKey)
		require.NoError(t, err)

		// Verify last_used_at was updated
		assert.NotNil(t, validated.LastUsedAt)
		assert.True(t, validated.LastUsedAt.After(fresh.CreatedAt))
	})
}

func TestListClientKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := getSharedTestDB(t)
	// Note: Don't close db as it's shared across all tests in the package

	service := NewClientKeyService(database.NewConnectionWithPool(db), nil)
	ctx := context.Background()

	// Default scopes used in tests
	defaultScopes := []string{"tables:read", "tables:write"}

	// Create test users with unique emails to avoid conflicts when tests run sequentially
	userID1 := createTestUser(t, db, fmt.Sprintf("list-test-%s@example.com", uuid.New().String()[:8]))
	userID2 := createTestUser(t, db, fmt.Sprintf("list-test-%s@example.com", uuid.New().String()[:8]))

	// Create test client keys
	_, err := service.GenerateClientKey(ctx, "test-list-1", nil, &userID1, defaultScopes, 0, nil)
	require.NoError(t, err)
	_, err = service.GenerateClientKey(ctx, "test-list-2", nil, &userID1, defaultScopes, 0, nil)
	require.NoError(t, err)
	_, err = service.GenerateClientKey(ctx, "test-list-3", nil, &userID2, defaultScopes, 0, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "DELETE FROM auth.client_keys WHERE name IN ('test-list-1', 'test-list-2', 'test-list-3')")
		_, _ = db.Exec(ctx, "DELETE FROM auth.users WHERE email LIKE 'list-test-%@example.com'")
	})

	t.Run("List all client keys", func(t *testing.T) {
		keys, err := service.ListClientKeys(ctx, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(keys), 3)
	})

	t.Run("List client keys by user", func(t *testing.T) {
		keys, err := service.ListClientKeys(ctx, &userID1)
		require.NoError(t, err)
		assert.Len(t, keys, 2)

		for _, key := range keys {
			assert.Equal(t, &userID1, key.UserID)
		}
	})

	t.Run("List client keys ordered by created_at DESC", func(t *testing.T) {
		keys, err := service.ListClientKeys(ctx, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(keys), 3)

		// Verify descending order (most recent first)
		for i := 0; i < len(keys)-1; i++ {
			assert.True(t, keys[i].CreatedAt.After(keys[i+1].CreatedAt) || keys[i].CreatedAt.Equal(keys[i+1].CreatedAt))
		}
	})
}

func TestRevokeClientKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := getSharedTestDB(t)
	// Note: Don't close db as it's shared across all tests in the package

	service := NewClientKeyService(database.NewConnectionWithPool(db), nil)
	ctx := context.Background()

	// Default scopes used in tests
	defaultScopes := []string{"tables:read", "tables:write"}

	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "DELETE FROM auth.client_keys WHERE name = 'test-revoke'")
	})

	t.Run("Revoke existing client key", func(t *testing.T) {
		created, err := service.GenerateClientKey(ctx, "test-revoke", nil, nil, defaultScopes, 0, nil)
		require.NoError(t, err)
		assert.Nil(t, created.RevokedAt)

		err = service.RevokeClientKey(ctx, created.ID)
		require.NoError(t, err)

		// Verify revoked_at is set
		keys, err := service.ListClientKeys(ctx, nil)
		require.NoError(t, err)

		for _, key := range keys {
			if key.ID == created.ID {
				assert.NotNil(t, key.RevokedAt)
				break
			}
		}
	})

	t.Run("Revoke non-existent client key", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := service.RevokeClientKey(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestDeleteClientKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := getSharedTestDB(t)
	// Note: Don't close db as it's shared across all tests in the package

	service := NewClientKeyService(database.NewConnectionWithPool(db), nil)
	ctx := context.Background()

	// Default scopes used in tests
	defaultScopes := []string{"tables:read", "tables:write"}

	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "DELETE FROM auth.client_keys WHERE name = 'test-delete'")
	})

	t.Run("Delete existing client key", func(t *testing.T) {
		created, err := service.GenerateClientKey(ctx, "test-delete", nil, nil, defaultScopes, 0, nil)
		require.NoError(t, err)

		err = service.DeleteClientKey(ctx, created.ID)
		require.NoError(t, err)

		// Verify key is deleted
		clientKey, err := service.ValidateClientKey(ctx, created.PlaintextKey)
		assert.Error(t, err)
		assert.Nil(t, clientKey)
	})

	t.Run("Delete non-existent client key", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := service.DeleteClientKey(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestUpdateClientKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := getSharedTestDB(t)
	// Note: Don't close db as it's shared across all tests in the package

	service := NewClientKeyService(database.NewConnectionWithPool(db), nil)
	ctx := context.Background()

	// Default scopes used in tests
	defaultScopes := []string{"tables:read", "tables:write"}

	created, err := service.GenerateClientKey(ctx, "test-update", nil, nil, defaultScopes, 0, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "DELETE FROM auth.client_keys WHERE name IN ('test-update', 'test-updated-name')")
	})

	t.Run("Update client key name", func(t *testing.T) {
		newName := "test-updated-name"
		err := service.UpdateClientKey(ctx, created.ID, &newName, nil, nil, nil)
		require.NoError(t, err)

		// Verify update
		keys, err := service.ListClientKeys(ctx, nil)
		require.NoError(t, err)

		for _, key := range keys {
			if key.ID == created.ID {
				assert.Equal(t, newName, key.Name)
				break
			}
		}
	})

	t.Run("Update client key scopes", func(t *testing.T) {
		newScopes := []string{"tables:read", "storage:read"}
		err := service.UpdateClientKey(ctx, created.ID, nil, nil, newScopes, nil)
		require.NoError(t, err)

		// Verify update
		keys, err := service.ListClientKeys(ctx, nil)
		require.NoError(t, err)

		for _, key := range keys {
			if key.ID == created.ID {
				assert.ElementsMatch(t, newScopes, key.Scopes)
				break
			}
		}
	})

	t.Run("Update client key rate limit", func(t *testing.T) {
		newRateLimit := 500
		err := service.UpdateClientKey(ctx, created.ID, nil, nil, nil, &newRateLimit)
		require.NoError(t, err)

		// Verify update
		keys, err := service.ListClientKeys(ctx, nil)
		require.NoError(t, err)

		for _, key := range keys {
			if key.ID == created.ID {
				assert.Equal(t, newRateLimit, key.RateLimitPerMinute)
				break
			}
		}
	})

	t.Run("Update non-existent client key", func(t *testing.T) {
		nonExistentID := uuid.New()
		newName := "should-fail"
		err := service.UpdateClientKey(ctx, nonExistentID, &newName, nil, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestClientKeyServiceNewClientKeyService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := getSharedTestDB(t)

	service := NewClientKeyService(database.NewConnectionWithPool(db), nil)
	assert.NotNil(t, service)
	assert.NotNil(t, service.db)
}
