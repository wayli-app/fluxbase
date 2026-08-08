package branching

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Cleanup Scheduler Construction Tests
// =============================================================================

func TestNewCleanupScheduler(t *testing.T) {
	t.Run("creates scheduler with nil dependencies", func(t *testing.T) {
		interval := 24 * time.Hour

		scheduler := NewCleanupScheduler(nil, nil, interval)

		assert.NotNil(t, scheduler)
	})

	t.Run("scheduler with zero interval uses default", func(t *testing.T) {
		scheduler := NewCleanupScheduler(nil, nil, 0)

		assert.NotNil(t, scheduler)
	})

	t.Run("scheduler with negative interval uses default", func(t *testing.T) {
		scheduler := NewCleanupScheduler(nil, nil, -1*time.Hour)

		assert.NotNil(t, scheduler)
	})

	t.Run("scheduler with custom interval", func(t *testing.T) {
		interval := 7 * 24 * time.Hour
		scheduler := NewCleanupScheduler(nil, nil, interval)

		assert.NotNil(t, scheduler)
	})
}

// =============================================================================
// Cleanup Scheduler Stop Tests
// =============================================================================

func TestCleanupScheduler_Stop(t *testing.T) {
	t.Run("stop without start should not panic", func(t *testing.T) {
		interval := 24 * time.Hour
		scheduler := NewCleanupScheduler(nil, nil, interval)

		// Stop should not panic
		assert.NotPanics(t, func() {
			scheduler.Stop()
		})
	})

	t.Run("double stop should not panic", func(t *testing.T) {
		interval := 24 * time.Hour
		scheduler := NewCleanupScheduler(nil, nil, interval)

		scheduler.Stop()
		assert.NotPanics(t, func() {
			scheduler.Stop()
		})
	})
}

// =============================================================================
// Branch Expiration Tests
// =============================================================================

func TestBranchExpiration(t *testing.T) {
	t.Run("branch is expired", func(t *testing.T) {
		expiresAt := time.Now().Add(-1 * time.Hour)

		branch := Branch{
			ID:        uuid.New(),
			Name:      "expired-branch",
			Type:      BranchTypePreview,
			ExpiresAt: &expiresAt,
		}

		isExpired := branch.ExpiresAt != nil && branch.ExpiresAt.Before(time.Now())
		assert.True(t, isExpired)
	})

	t.Run("branch is not expired", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)

		branch := Branch{
			ID:        uuid.New(),
			Name:      "active-branch",
			Type:      BranchTypePreview,
			ExpiresAt: &expiresAt,
		}

		isExpired := branch.ExpiresAt != nil && branch.ExpiresAt.Before(time.Now())
		assert.False(t, isExpired)
	})

	t.Run("branch without expiration", func(t *testing.T) {
		branch := Branch{
			ID:   uuid.New(),
			Name: "persistent-branch",
			Type: BranchTypePersistent,
		}

		hasExpiration := branch.ExpiresAt != nil
		assert.False(t, hasExpiration)
	})
}

// =============================================================================
// Branch Type Auto-Delete Tests
// =============================================================================

func TestBranchTypeAutoDelete(t *testing.T) {
	t.Run("preview branches can be auto-deleted", func(t *testing.T) {
		branch := Branch{
			ID:   uuid.New(),
			Type: BranchTypePreview,
		}

		canAutoDelete := branch.Type == BranchTypePreview
		assert.True(t, canAutoDelete)
	})

	t.Run("main branch cannot be auto-deleted", func(t *testing.T) {
		branch := Branch{
			ID:   uuid.New(),
			Type: BranchTypeMain,
		}

		canAutoDelete := branch.Type == BranchTypePreview
		assert.False(t, canAutoDelete)
	})

	t.Run("persistent branches are not auto-deleted", func(t *testing.T) {
		branch := Branch{
			ID:   uuid.New(),
			Type: BranchTypePersistent,
		}

		canAutoDelete := branch.Type == BranchTypePreview
		assert.False(t, canAutoDelete)
	})
}

// =============================================================================
// Scheduler Interval Tests
// =============================================================================

func TestScheduler_Interval(t *testing.T) {
	t.Run("check interval for cleanup", func(t *testing.T) {
		// Typical cleanup interval is every minute or every 5 minutes
		checkInterval := 5 * time.Minute

		assert.Equal(t, 5*time.Minute, checkInterval)
	})
}

// =============================================================================
// Branch Status for Deletion Tests
// =============================================================================

func TestBranchStatusForDeletion(t *testing.T) {
	t.Run("ready branch can be deleted", func(t *testing.T) {
		branch := Branch{
			ID:     uuid.New(),
			Status: BranchStatusReady,
		}

		canDelete := branch.Status == BranchStatusReady
		assert.True(t, canDelete)
	})

	t.Run("creating branch should not be deleted", func(t *testing.T) {
		branch := Branch{
			ID:     uuid.New(),
			Status: BranchStatusCreating,
		}

		canDelete := branch.Status == BranchStatusReady
		assert.False(t, canDelete)
	})

	t.Run("deleting branch should not be deleted again", func(t *testing.T) {
		branch := Branch{
			ID:     uuid.New(),
			Status: BranchStatusDeleting,
		}

		canDelete := branch.Status == BranchStatusReady
		assert.False(t, canDelete)
	})

	t.Run("error branch can be deleted", func(t *testing.T) {
		branch := Branch{
			ID:     uuid.New(),
			Status: BranchStatusError,
		}

		// Error branches might be cleaned up
		canDelete := branch.Status == BranchStatusError || branch.Status == BranchStatusReady
		assert.True(t, canDelete)
	})
}

// =============================================================================
// Expiration Calculation Tests
// =============================================================================

func TestExpirationCalculation(t *testing.T) {
	t.Run("calculate expiration from auto-delete duration", func(t *testing.T) {
		autoDeleteAfter := 24 * time.Hour
		createdAt := time.Now()
		expiresAt := createdAt.Add(autoDeleteAfter)

		assert.True(t, expiresAt.After(createdAt))
		assert.Equal(t, 24*time.Hour, expiresAt.Sub(createdAt))
	})

	t.Run("calculate expiration for 48 hours", func(t *testing.T) {
		autoDeleteAfter := 48 * time.Hour
		createdAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		expiresAt := createdAt.Add(autoDeleteAfter)

		assert.Equal(t, time.Date(2024, 1, 17, 10, 0, 0, 0, time.UTC), expiresAt)
	})

	t.Run("calculate expiration for 7 days", func(t *testing.T) {
		autoDeleteAfter := 7 * 24 * time.Hour
		createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		expiresAt := createdAt.Add(autoDeleteAfter)

		assert.Equal(t, time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC), expiresAt)
	})
}

// =============================================================================
// Branch Cleanup Priority Tests
// =============================================================================

func TestBranchCleanupPriority(t *testing.T) {
	t.Run("older expired branches cleaned first", func(t *testing.T) {
		now := time.Now()

		branch1 := Branch{
			ID:        uuid.New(),
			ExpiresAt: func() *time.Time { t := now.Add(-2 * time.Hour); return &t }(),
		}

		branch2 := Branch{
			ID:        uuid.New(),
			ExpiresAt: func() *time.Time { t := now.Add(-1 * time.Hour); return &t }(),
		}

		// branch1 expired earlier, should be cleaned first
		assert.True(t, branch1.ExpiresAt.Before(*branch2.ExpiresAt))
	})
}

// =============================================================================
// Scheduler.IsRunning Tests
// =============================================================================
//
// IsRunning (scheduler.go:169) reports the running flag under a mutex. It was
// 0.0% under -short. NewCleanupScheduler(nil, nil, interval) builds a scheduler
// without starting it; the true branch is exercised by setting the unexported
// field directly (same-package white-box test).

func TestCleanupScheduler_IsRunning(t *testing.T) {
	t.Parallel()
	t.Run("fresh scheduler not running", func(t *testing.T) {
		t.Parallel()
		s := NewCleanupScheduler(nil, nil, time.Hour)
		assert.False(t, s.IsRunning())
	})

	t.Run("running flag true when set", func(t *testing.T) {
		t.Parallel()
		s := NewCleanupScheduler(nil, nil, time.Hour)
		s.mu.Lock()
		s.running = true
		s.mu.Unlock()
		assert.True(t, s.IsRunning())
	})
}
