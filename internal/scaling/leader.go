package scaling

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

const (
	JobsSchedulerLockID     int64 = 0x466C7578_00000001
	FunctionsSchedulerLockID int64 = 0x466C7578_00000002
	RPCSchedulerLockID      int64 = 0x466C7578_00000003
)

type LeaderElector struct {
	pool          *pgxpool.Pool
	lockID        int64
	lockName      string
	isLeader      bool
	isLeaderMu    sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	checkInterval time.Duration

	dedicatedConn *pgxpool.Conn
	started       bool
	startedMu     sync.Mutex
}

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

func (le *LeaderElector) Start(onBecomeLeader, onLoseLeadership func()) {
	le.startedMu.Lock()
	if le.started {
		le.startedMu.Unlock()
		return
	}
	le.started = true
	le.startedMu.Unlock()

	log.Info().
		Str("lock", le.lockName).
		Int64("lock_id", le.lockID).
		Msg("Starting leader election")

	go le.electionLoop(onBecomeLeader, onLoseLeadership)
}

func (le *LeaderElector) Stop() {
	log.Info().
		Str("lock", le.lockName).
		Bool("was_leader", le.IsLeader()).
		Msg("Stopping leader election")

	le.cancel()

	le.releaseLock()
}

func (le *LeaderElector) IsLeader() bool {
	le.isLeaderMu.RLock()
	defer le.isLeaderMu.RUnlock()
	return le.isLeader
}

func (le *LeaderElector) electionLoop(onBecomeLeader, onLoseLeadership func()) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Error().
				Interface("panic", rec).
				Str("lock", le.lockName).
				Msg("Panic in leader election loop - recovered")
			time.Sleep(5 * time.Second)
			if le.ctx.Err() == nil {
				go le.electionLoop(onBecomeLeader, onLoseLeadership)
			}
		}
	}()

	ticker := time.NewTicker(le.checkInterval)
	defer ticker.Stop()

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

func (le *LeaderElector) tryAcquireLock(onBecomeLeader, onLoseLeadership func()) {
	if le.dedicatedConn == nil {
		conn, err := le.pool.Acquire(le.ctx)
		if err != nil {
			if le.ctx.Err() == nil {
				log.Error().Err(err).Str("lock", le.lockName).Msg("Failed to acquire dedicated connection")
			}
			return
		}
		le.dedicatedConn = conn
	}

	ctx, cancel := context.WithTimeout(le.ctx, 5*time.Second)
	defer cancel()

	var acquired bool
	err := le.dedicatedConn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", le.lockID).Scan(&acquired)
	if err != nil {
		if le.ctx.Err() == nil {
			log.Error().Err(err).Str("lock", le.lockName).Msg("Failed to try advisory lock")
		}
		le.dedicatedConn.Release()
		le.dedicatedConn = nil
		return
	}

	le.isLeaderMu.Lock()
	wasLeader := le.isLeader
	le.isLeader = acquired
	le.isLeaderMu.Unlock()

	if acquired && !wasLeader {
		log.Info().Str("lock", le.lockName).Msg("Acquired leader lock - this instance is now the leader")
		if onBecomeLeader != nil {
			le.safeCallback(onBecomeLeader, "onBecomeLeader")
		}
	} else if !acquired && wasLeader {
		le.dedicatedConn.Release()
		le.dedicatedConn = nil
		log.Warn().Str("lock", le.lockName).Msg("Lost leader lock - this instance is no longer the leader")
		if onLoseLeadership != nil {
			le.safeCallback(onLoseLeadership, "onLoseLeadership")
		}
	}
}

func (le *LeaderElector) safeCallback(fn func(), name string) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Error().
				Interface("panic", rec).
				Str("lock", le.lockName).
				Str("callback", name).
				Msg("Panic in leader election callback - recovered")
		}
	}()
	fn()
}

func (le *LeaderElector) releaseLock() {
	if le.dedicatedConn != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		var released bool
		_ = le.dedicatedConn.QueryRow(ctx, "SELECT pg_advisory_unlock($1)", le.lockID).Scan(&released)

		le.dedicatedConn.Release()
		le.dedicatedConn = nil
	}

	le.isLeaderMu.Lock()
	le.isLeader = false
	le.isLeaderMu.Unlock()
}
