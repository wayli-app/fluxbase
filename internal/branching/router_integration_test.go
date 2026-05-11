//go:build integration

package branching

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/test/dbhelpers"
)

const routerTestEncryptionKey = "test-encryption-key-must-be-32-chars!"

// TestRouter_GetPool_Integration tests connection pool retrieval.
func TestRouter_GetPool_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:       true,
		DefaultBranch: "",
	}

	router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

	t.Run("returns main pool for empty slug", func(t *testing.T) {
		pool, err := router.GetPool(context.Background(), "")
		require.NoError(t, err)
		assert.NotNil(t, pool)
	})

	t.Run("returns main pool for main slug", func(t *testing.T) {
		pool, err := router.GetPool(context.Background(), "main")
		require.NoError(t, err)
		assert.NotNil(t, pool)
	})

	t.Run("returns main pool directly", func(t *testing.T) {
		pool := router.GetMainPool()
		assert.NotNil(t, pool)
	})

	t.Run("returns error when branching disabled", func(t *testing.T) {
		cfgDisabled := config.BranchingConfig{Enabled: false}
		routerDisabled := NewRouter(storage, cfgDisabled, testCtx.Pool, testCtx.DatabaseURL())

		_, err := routerDisabled.GetPool(context.Background(), "feature-branch")
		assert.Equal(t, ErrBranchingDisabled, err)
	})
}

// TestRouter_BranchNotReady_Integration tests handling of non-ready branches.
func TestRouter_BranchNotReady_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:       true,
		DefaultBranch: "",
	}

	router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

	t.Run("returns error for non-existent branch", func(t *testing.T) {
		_, err := router.GetPool(context.Background(), "nonexistent-branch")
		assert.Error(t, err)
		assert.Equal(t, ErrBranchNotFound, err)
	})
}

// TestRouter_PoolCaching_Integration tests pool caching behavior.
func TestRouter_PoolCaching_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:       true,
		DefaultBranch: "",
	}

	router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

	t.Run("caches main pool", func(t *testing.T) {
		// Main pool is always cached
		pool1, err := router.GetPool(context.Background(), "")
		require.NoError(t, err)

		pool2, err := router.GetPool(context.Background(), "")
		require.NoError(t, err)

		assert.Same(t, pool1, pool2, "Should return same pool instance for main")
	})
}

// TestRouter_HasPool_Integration tests pool existence checking.
func TestRouter_HasPool_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:       true,
		DefaultBranch: "",
	}

	router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

	t.Run("main branch always has pool", func(t *testing.T) {
		assert.True(t, router.HasPool(""))
		assert.True(t, router.HasPool("main"))
	})

	t.Run("non-existent branch returns false", func(t *testing.T) {
		assert.False(t, router.HasPool("nonexistent-branch"))
	})
}

// TestRouter_ClosePool_Integration tests pool cleanup.
func TestRouter_ClosePool_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:       true,
		DefaultBranch: "",
	}

	router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

	t.Run("closes non-existent pool gracefully", func(t *testing.T) {
		// Should not panic
		router.ClosePool("non-existent-branch")
	})

	t.Run("close and clear all pools", func(t *testing.T) {
		router.CloseAllPools()

		// Should still be able to get main pool
		pool, err := router.GetPool(context.Background(), "")
		require.NoError(t, err)
		assert.NotNil(t, pool)
	})
}

// TestRouter_ActivePools_Integration tests active pool tracking.
func TestRouter_ActivePools_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:       true,
		DefaultBranch: "",
	}

	router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

	t.Run("lists active pools", func(t *testing.T) {
		pools := router.GetActivePools()
		assert.NotNil(t, pools)
		// Main pool is not tracked in the pools map
		assert.Empty(t, pools)
	})
}

// TestRouter_PoolStats_Integration tests pool statistics.
func TestRouter_PoolStats_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:       true,
		DefaultBranch: "",
	}

	router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

	t.Run("returns pool statistics", func(t *testing.T) {
		stats := router.GetPoolStats()
		assert.NotNil(t, stats)

		// Should have main pool stats
		mainStats, exists := stats["main"]
		assert.True(t, exists, "Should have stats for main pool")
		assert.Equal(t, int32(0), mainStats.AcquiredConns)
	})
}

// TestRouter_ActiveBranch_Integration tests active branch management.
func TestRouter_ActiveBranch_Integration(t *testing.T) {
	t.Run("gets default branch", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:       true,
			DefaultBranch: "staging",
		}

		testCtx := dbhelpers.NewDBTestContext(t)
		defer testCtx.Close()

		storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
		router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

		// GetActiveBranch should return empty when no API-set branch
		active := router.GetActiveBranch()
		assert.Equal(t, "", active)

		// GetDefaultBranch should return config default
		defaultBranch := router.GetDefaultBranch()
		assert.Equal(t, "staging", defaultBranch)
	})

	t.Run("sets and gets active branch", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:       true,
			DefaultBranch: "",
		}

		testCtx := dbhelpers.NewDBTestContext(t)
		defer testCtx.Close()

		storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
		router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

		// Set active branch
		router.SetActiveBranch("feature-branch")

		active := router.GetActiveBranch()
		assert.Equal(t, "feature-branch", active)
	})

	t.Run("clears active branch", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:       true,
			DefaultBranch: "default-branch",
		}

		testCtx := dbhelpers.NewDBTestContext(t)
		defer testCtx.Close()

		storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
		router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

		// Set then clear
		router.SetActiveBranch("temp-branch")
		router.SetActiveBranch("")

		active := router.GetActiveBranch()
		assert.Equal(t, "", active)
	})

	t.Run("gets effective default branch", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:       true,
			DefaultBranch: "config-default",
		}

		testCtx := dbhelpers.NewDBTestContext(t)
		defer testCtx.Close()

		storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
		router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

		// Without API override, should return config default
		defaultBranch := router.GetDefaultBranch()
		assert.Equal(t, "config-default", defaultBranch)

		// With API override, should return API branch
		router.SetActiveBranch("api-override")
		defaultBranch = router.GetDefaultBranch()
		assert.Equal(t, "api-override", defaultBranch)
	})

	t.Run("identifies active branch source", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:       true,
			DefaultBranch: "config-branch",
		}

		testCtx := dbhelpers.NewDBTestContext(t)
		defer testCtx.Close()

		storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
		router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

		// Initially should be config source
		source := router.GetActiveBranchSource()
		assert.Equal(t, "config", source)

		// After setting, should be api source
		router.SetActiveBranch("api-branch")
		source = router.GetActiveBranchSource()
		assert.Equal(t, "api", source)

		// After clearing, should return to config
		router.SetActiveBranch("")
		source = router.GetActiveBranchSource()
		assert.Equal(t, "config", source)
	})
}

// TestRouter_WarmupPool_Integration tests pool warmup.
func TestRouter_WarmupPool_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:       true,
		DefaultBranch: "",
	}

	router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

	t.Run("warms up main pool", func(t *testing.T) {
		err := router.WarmupPool(context.Background(), "")
		assert.NoError(t, err)
	})

	t.Run("returns error for non-existent branch", func(t *testing.T) {
		err := router.WarmupPool(context.Background(), "nonexistent-branch")
		assert.Error(t, err)
	})
}

// TestRouter_RefreshPool_Integration tests pool refresh.
func TestRouter_RefreshPool_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:       true,
		DefaultBranch: "",
	}

	router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

	t.Run("returns error for non-existent branch", func(t *testing.T) {
		err := router.RefreshPool(context.Background(), "nonexistent-branch")
		assert.Error(t, err)
	})
}

// TestRouter_ThreadSafety_Integration tests concurrent access.
func TestRouter_ThreadSafety_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:       true,
		DefaultBranch: "",
	}

	router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

	t.Run("concurrent pool access is safe", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Try to get main pool concurrently
				_, _ = router.GetPool(context.Background(), "")
			}()
		}

		wg.Wait()
		// If we get here without panic or deadlock, test passed
	})

	t.Run("concurrent active branch changes are safe", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				branchName := ""
				if n%2 == 0 {
					branchName = "branch-a"
				} else {
					branchName = "branch-b"
				}
				router.SetActiveBranch(branchName)
			}(i)
		}

		wg.Wait()
		// If we get here without panic, test passed
	})
}

// TestRouter_GetStorage_Integration tests storage access.
func TestRouter_GetStorage_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(routerTestEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:       true,
		DefaultBranch: "",
	}

	router := NewRouter(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())

	t.Run("returns storage instance", func(t *testing.T) {
		retStorage := router.GetStorage()
		assert.Same(t, storage, retStorage)
	})
}

// TestRouter_IsMainBranch_Integration tests main branch detection.
func TestRouter_IsMainBranch_Integration(t *testing.T) {
	t.Run("identifies main branch variants", func(t *testing.T) {
		assert.True(t, IsMainBranch(""))
		assert.True(t, IsMainBranch("main"))
		assert.False(t, IsMainBranch("feature-branch"))
		assert.False(t, IsMainBranch("development"))
	})
}
