package branching

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nimbleflux/fluxbase/internal/config"
)

// =============================================================================
// Router Construction Tests
// =============================================================================

func TestNewRouter(t *testing.T) {
	t.Run("creates router with nil dependencies", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:       true,
			DefaultBranch: "",
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		assert.NotNil(t, router)
		assert.NotNil(t, router.pools)
		assert.NotNil(t, router.lruList)
		assert.Empty(t, router.pools)
	})

	t.Run("initializes with empty active branch (API-set only)", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:       true,
			DefaultBranch: "development",
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		// activeBranch should be empty initially (only set via API)
		activeBranch := router.activeBranch.Load()
		assert.Equal(t, "", activeBranch)

		// Config default branch is stored separately
		assert.Equal(t, "development", router.config.DefaultBranch)
	})

	t.Run("initializes with empty default branch", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:       true,
			DefaultBranch: "",
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		activeBranch := router.activeBranch.Load()
		assert.Equal(t, "", activeBranch)
	})
}

// =============================================================================
// Router GetPool Tests
// =============================================================================

func TestRouter_GetPool_MainBranch(t *testing.T) {
	t.Run("empty slug returns main pool", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled: true,
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		// With nil main pool, this should return nil
		pool, err := router.GetPool(context.TODO(), "")
		assert.NoError(t, err)
		assert.Nil(t, pool) // main pool is nil in this test
	})

	t.Run("main slug returns main pool", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled: true,
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		pool, err := router.GetPool(context.TODO(), "main")
		assert.NoError(t, err)
		assert.Nil(t, pool) // main pool is nil in this test
	})
}

func TestRouter_GetPool_BranchingDisabled(t *testing.T) {
	t.Run("returns error when branching disabled", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled: false,
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		_, err := router.GetPool(context.TODO(), "feature-branch")
		assert.Error(t, err)
		assert.Equal(t, ErrBranchingDisabled, err)
	})
}

// =============================================================================
// Router ClosePool Tests
// =============================================================================

func TestRouter_ClosePool(t *testing.T) {
	t.Run("closes non-existent pool gracefully", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled: true,
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		// Should not panic
		router.ClosePool("non-existent")
	})
}

// =============================================================================
// Router Active Branch Tests
// =============================================================================

func TestRouter_ActiveBranch(t *testing.T) {
	t.Run("get active branch", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:       true,
			DefaultBranch: "staging",
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		// GetActiveBranch returns empty when no branch set via API
		active := router.GetActiveBranch()
		assert.Equal(t, "", active)

		// GetDefaultBranch returns config default when no API-set branch
		defaultBranch := router.GetDefaultBranch()
		assert.Equal(t, "staging", defaultBranch)
	})

	t.Run("set active branch", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:       true,
			DefaultBranch: "",
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		router.SetActiveBranch("new-branch")

		active := router.GetActiveBranch()
		assert.Equal(t, "new-branch", active)
	})

	t.Run("clear active branch", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:       true,
			DefaultBranch: "initial",
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		router.SetActiveBranch("")

		active := router.GetActiveBranch()
		assert.Equal(t, "", active)
	})
}

// =============================================================================
// Router Pool Management Tests
// =============================================================================

func TestRouter_PoolManagement(t *testing.T) {
	t.Run("pools map is thread-safe", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled: true,
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		// Should be able to access pools map safely
		router.poolsMu.RLock()
		count := len(router.pools)
		router.poolsMu.RUnlock()

		assert.Equal(t, 0, count)
	})
}

// =============================================================================
// Router CloseAll Tests
// =============================================================================

func TestRouter_CloseAll(t *testing.T) {
	t.Run("closes all pools", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled: true,
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		// Should not panic
		router.CloseAllPools()

		// Pools should be empty
		router.poolsMu.RLock()
		count := len(router.pools)
		router.poolsMu.RUnlock()

		assert.Equal(t, 0, count)
	})
}

// =============================================================================
// Router Branch Connection URL Tests
// =============================================================================

func TestRouter_BranchConnectionURL(t *testing.T) {
	t.Run("generates connection URL for branch", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled: true,
		}

		router := NewRouter(nil, cfg, nil, "postgresql://user:pass@localhost:5432/fluxbase")

		branch := &Branch{
			DatabaseName: "branch_feature-123",
		}

		url, err := router.getBranchConnectionURL(branch)
		assert.NoError(t, err)
		assert.Contains(t, url, "branch_feature-123")
		assert.Contains(t, url, "localhost")
	})

	t.Run("handles simple URL format", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled: true,
		}

		router := NewRouter(nil, cfg, nil, "not-a-valid-url")

		branch := &Branch{
			DatabaseName: "branch_test",
		}

		// url.Parse is lenient and won't error on simple strings
		// It just treats them as relative paths
		url, err := router.getBranchConnectionURL(branch)
		assert.NoError(t, err)
		assert.Contains(t, url, "branch_test")
	})
}

// =============================================================================
// Router Config Tests
// =============================================================================

func TestRouter_Config(t *testing.T) {
	t.Run("stores config reference", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:            true,
			MaxBranchesPerUser: 10,
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		assert.Equal(t, cfg.Enabled, router.config.Enabled)
		assert.Equal(t, cfg.MaxBranchesPerUser, router.config.MaxBranchesPerUser)
	})
}

// =============================================================================
// Router Storage Tests
// =============================================================================

func TestRouter_Storage(t *testing.T) {
	t.Run("stores storage reference", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled: true,
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		// Storage is nil when passed as nil
		assert.Nil(t, router.storage)
	})
}

// =============================================================================
// Router Main Pool Tests
// =============================================================================

func TestRouter_MainPool(t *testing.T) {
	t.Run("stores main pool reference", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled: true,
		}

		router := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

		// Main pool is nil when passed as nil
		assert.Nil(t, router.mainPool)
	})
}

// =============================================================================
// Router Main DB URL Tests
// =============================================================================

func TestRouter_MainDBURL(t *testing.T) {
	t.Run("stores main database URL", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled: true,
		}

		url := "postgresql://user:pass@localhost:5432/fluxbase"
		router := NewRouter(nil, cfg, nil, url)

		assert.Equal(t, url, router.mainDBURL)
	})
}

// =============================================================================
// IsMainBranch / Pool-Presence / Default-Branch Tests
// =============================================================================
//
// These cover pure, 0.0%-coverage router helpers that existing -short tests
// never reach (they're exercised only by integration tests). All use nil-pool
// routers built via NewRouter(nil, cfg, nil, url), matching TestNewRouter.

func TestIsMainBranch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		slug string
		want bool
	}{
		{"empty is main", "", true},
		{"literal main", "main", true},
		{"development is not main", "development", false},
		{"main-main is not main (exact match only)", "main-main", false},
		{"arbitrary slug", "feature-x", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsMainBranch(tt.slug))
		})
	}
}

func TestRouter_GetActiveBranchSource(t *testing.T) {
	t.Parallel()
	t.Run("api-set wins over config", func(t *testing.T) {
		t.Parallel()
		cfg := config.BranchingConfig{Enabled: true, DefaultBranch: "development"}
		r := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")
		r.SetActiveBranch("feature-x")
		assert.Equal(t, "api", r.GetActiveBranchSource())
	})

	t.Run("config default when no api branch", func(t *testing.T) {
		t.Parallel()
		cfg := config.BranchingConfig{Enabled: true, DefaultBranch: "development"}
		r := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")
		assert.Equal(t, "config", r.GetActiveBranchSource())
	})

	t.Run("default when neither api nor config set", func(t *testing.T) {
		t.Parallel()
		// DefaultBranch empty → the previously-uncovered fallback arm.
		cfg := config.BranchingConfig{Enabled: true, DefaultBranch: ""}
		r := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")
		assert.Equal(t, "default", r.GetActiveBranchSource())
	})
}

func TestRouter_GetDefaultBranch_FallbackToMain(t *testing.T) {
	t.Parallel()
	// Neither API-set nor config default → "main" (the arm GetDefaultBranch
	// didn't cover under -short).
	cfg := config.BranchingConfig{Enabled: true, DefaultBranch: ""}
	r := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")
	assert.Equal(t, "main", r.GetDefaultBranch())
}

func TestRouter_HasPool(t *testing.T) {
	t.Parallel()
	cfg := config.BranchingConfig{Enabled: true}
	r := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

	t.Run("main slug always has pool", func(t *testing.T) {
		t.Parallel()
		assert.True(t, r.HasPool("main"))
		assert.True(t, r.HasPool(""))
	})

	t.Run("unknown slug returns false", func(t *testing.T) {
		t.Parallel()
		assert.False(t, r.HasPool("nope"))
	})

	t.Run("injected slug returns true", func(t *testing.T) {
		t.Parallel()
		// HasPool/GetActivePools only check map presence, never dereference the
		// pool, so a nil-pool entry is a valid fixture.
		r.poolsMu.Lock()
		r.pools["feature-x"] = &poolEntry{slug: "feature-x"}
		r.poolsMu.Unlock()
		assert.True(t, r.HasPool("feature-x"))
	})
}

func TestRouter_GetActivePools(t *testing.T) {
	t.Parallel()
	cfg := config.BranchingConfig{Enabled: true}

	t.Run("empty router returns non-nil empty slice", func(t *testing.T) {
		t.Parallel()
		r := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")
		got := r.GetActivePools()
		// Contract: make([]string, 0, ...) — non-nil even when empty.
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("returns injected slugs", func(t *testing.T) {
		t.Parallel()
		r := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")
		r.poolsMu.Lock()
		r.pools["a"] = &poolEntry{slug: "a"}
		r.pools["b"] = &poolEntry{slug: "b"}
		r.poolsMu.Unlock()

		got := r.GetActivePools()
		assert.ElementsMatch(t, []string{"a", "b"}, got)
	})
}

func TestRouter_GetMainPoolAndGetStorage(t *testing.T) {
	t.Parallel()
	t.Run("nil deps return nil", func(t *testing.T) {
		t.Parallel()
		cfg := config.BranchingConfig{Enabled: true}
		r := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")
		assert.Nil(t, r.GetMainPool())
		assert.Nil(t, r.GetStorage())
	})

	t.Run("non-nil storage returned by identity", func(t *testing.T) {
		t.Parallel()
		cfg := config.BranchingConfig{Enabled: true}
		// NewStorage(nil, nil) is the existing nil-deps construction (used in
		// storage_test.go). It does not open a connection.
		st := NewStorage(nil, nil)
		r := NewRouter(st, cfg, nil, "postgresql://localhost/fluxbase")
		assert.Same(t, st, r.GetStorage())
	})
}

// =============================================================================
// GetPool / WarmupPool cheap paths (no DB needed)
// =============================================================================

func TestRouter_GetPool_CheapPaths(t *testing.T) {
	t.Parallel()
	cfg := config.BranchingConfig{Enabled: true}
	r := NewRouter(nil, cfg, nil, "postgresql://localhost/fluxbase")

	t.Run("main slug returns main pool (nil here) with no error", func(t *testing.T) {
		t.Parallel()
		p, err := r.GetPool(context.Background(), "main")
		assert.NoError(t, err)
		assert.Nil(t, p) // mainPool is nil in this fixture
	})

	t.Run("empty slug treated as main", func(t *testing.T) {
		t.Parallel()
		p, err := r.GetPool(context.Background(), "")
		assert.NoError(t, err)
		assert.Nil(t, p)
	})

	t.Run("non-main slug with branching disabled returns ErrBranchingDisabled", func(t *testing.T) {
		t.Parallel()
		disabled := NewRouter(nil, config.BranchingConfig{Enabled: false}, nil, "postgresql://localhost/fluxbase")
		_, err := disabled.GetPool(context.Background(), "feature-x")
		assert.ErrorIs(t, err, ErrBranchingDisabled)
	})
}

func TestRouter_WarmupPool_Disabled(t *testing.T) {
	t.Parallel()
	// WarmupPool delegates to GetPool; with branching disabled and a non-main
	// slug it returns ErrBranchingDisabled without touching the network.
	r := NewRouter(nil, config.BranchingConfig{Enabled: false}, nil, "postgresql://localhost/fluxbase")
	err := r.WarmupPool(context.Background(), "feature-x")
	assert.ErrorIs(t, err, ErrBranchingDisabled)
}
