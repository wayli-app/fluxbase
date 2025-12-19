// Package scaling provides utilities for horizontal scaling of Fluxbase instances.
package scaling

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// LeaderLockID constants for PostgreSQL advisory locks
// These are unique identifiers for different leader election purposes
const (
	// JobsSchedulerLockID is the advisory lock ID for the jobs scheduler
	// Only one instance should run the jobs scheduler at a time
	JobsSchedulerLockID int64 = 0x466C7578_00000001 // "Flux" + 1

	// FunctionsSchedulerLockID is the advisory lock ID for the edge functions scheduler
	FunctionsSchedulerLockID int64 = 0x466C7578_00000002 // "Flux" + 2

	// RPCSchedulerLockID is the advisory lock ID for the RPC scheduler
	RPCSchedulerLockID int64 = 0x466C7578_00000003 // "Flux" + 3
)

// LeaderElector manages leader election using PostgreSQL advisory locks.
// It uses pg_try_advisory_lock for non-blocking lock acquisition, allowing
// the instance to gracefully handle not being the leader.
type LeaderElector struct {
	pool          *pgxpool.Pool
	lockID        int64
	lockName      string
	isLeader      bool
	isLeaderMu    sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	checkInterval time.Duration
}

// NewLeaderElector creates a new leader elector for the given lock ID.
// The lock name is used for logging purposes.
func NewLeaderElector(pool *pgxpool.Pool, lockID int64, lockName string) *LeaderElector {
	ctx, cancel := context.WithCancel(context.Background())
	return &LeaderElector{
		pool:          pool,
		lockID:        lockID,
		lockName:      lockName,
		isLeader:      false,
		ctx:           ctx,
		cancel:        cancel,
		checkInterval: 5 * time.Second,
	}
}

// Start begins the leader election process. It will periodically try to
// acquire the advisory lock and update the leadership status.
// The onBecomeLeader callback is called when this instance becomes the leader.
// The onLoseLeadership callback is called when this instance loses leadership.
func (le *LeaderElector) Start(onBecomeLeader, onLoseLeadership func()) {
	log.Info().
		Str("lock", le.lockName).
		Int64("lock_id", le.lockID).
		Msg("Starting leader election")

	go le.electionLoop(onBecomeLeader, onLoseLeadership)
}

// Stop stops the leader election process and releases the lock if held.
func (le *LeaderElector) Stop() {
	log.Info().
		Str("lock", le.lockName).
		Bool("was_leader", le.IsLeader()).
		Msg("Stopping leader election")

	le.cancel()

	// Release the lock if we were the leader
	if le.IsLeader() {
		le.releaseLock()
	}
}

// IsLeader returns true if this instance currently holds the leader lock.
func (le *LeaderElector) IsLeader() bool {
	le.isLeaderMu.RLock()
	defer le.isLeaderMu.RUnlock()
	return le.isLeader
}

// electionLoop periodically tries to acquire/maintain the leader lock.
func (le *LeaderElector) electionLoop(onBecomeLeader, onLoseLeadership func()) {
	ticker := time.NewTicker(le.checkInterval)
	defer ticker.Stop()

	// Try to acquire lock immediately
	le.tryAcquireLock(onBecomeLeader, onLoseLeadership)

	for {
		select {
		case <-le.ctx.Done():
			return
		case <-ticker.C:
			le.tryAcquireLock(onBecomeLeader, onLoseLeadership)
		}
	}
}

// tryAcquireLock attempts to acquire the advisory lock.
// PostgreSQL advisory locks are session-level, so we need to keep the connection alive.
func (le *LeaderElector) tryAcquireLock(onBecomeLeader, onLoseLeadership func()) {
	ctx, cancel := context.WithTimeout(le.ctx, 5*time.Second)
	defer cancel()

	// Try to acquire the lock (non-blocking)
	var acquired bool
	err := le.pool.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", le.lockID).Scan(&acquired)
	if err != nil {
		log.Error().
			Err(err).
			Str("lock", le.lockName).
			Msg("Failed to try advisory lock")
		return
	}

	le.isLeaderMu.Lock()
	wasLeader := le.isLeader
	le.isLeader = acquired
	le.isLeaderMu.Unlock()

	// Handle state transitions
	if acquired && !wasLeader {
		log.Info().
			Str("lock", le.lockName).
			Msg("Acquired leader lock - this instance is now the leader")
		if onBecomeLeader != nil {
			onBecomeLeader()
		}
	} else if !acquired && wasLeader {
		log.Warn().
			Str("lock", le.lockName).
			Msg("Lost leader lock - this instance is no longer the leader")
		if onLoseLeadership != nil {
			onLoseLeadership()
		}
	}
}

// releaseLock releases the advisory lock if held.
func (le *LeaderElector) releaseLock() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var released bool
	err := le.pool.QueryRow(ctx, "SELECT pg_advisory_unlock($1)", le.lockID).Scan(&released)
	if err != nil {
		log.Error().
			Err(err).
			Str("lock", le.lockName).
			Msg("Failed to release advisory lock")
		return
	}

	if released {
		log.Info().
			Str("lock", le.lockName).
			Msg("Released leader lock")
	}

	le.isLeaderMu.Lock()
	le.isLeader = false
	le.isLeaderMu.Unlock()
}

// TryAcquireOnce tries to acquire the lock once and returns immediately.
// This is useful for one-time checks without starting the election loop.
func (le *LeaderElector) TryAcquireOnce(ctx context.Context) (bool, error) {
	var acquired bool
	err := le.pool.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", le.lockID).Scan(&acquired)
	if err != nil {
		return false, err
	}

	le.isLeaderMu.Lock()
	le.isLeader = acquired
	le.isLeaderMu.Unlock()

	return acquired, nil
}
