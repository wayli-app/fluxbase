package logging

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/rs/zerolog/log"
)

// BatchWriteFunc is the function called to write a batch of entries.
type BatchWriteFunc func(ctx context.Context, entries []*storage.LogEntry) error

// Batcher buffers log entries and writes them in batches for efficiency.
// It flushes either when the batch size is reached or when the flush interval expires.
type Batcher struct {
	entries       chan *storage.LogEntry
	batchSize     int
	flushInterval time.Duration
	writeFunc     BatchWriteFunc
	wg            sync.WaitGroup
	done          chan struct{}
	flushReq      chan chan error // Channel to request a flush and receive result
	mu            sync.Mutex
	closed        bool

	// Metrics for monitoring backpressure
	droppedCount     atomic.Int64
	lastDroppedWarn  time.Time
	droppedWarnMu    sync.Mutex
}

// NewBatcher creates a new log entry batcher.
func NewBatcher(batchSize int, flushInterval time.Duration, bufferSize int, writeFunc BatchWriteFunc) *Batcher {
	if batchSize <= 0 {
		batchSize = 100
	}
	if flushInterval <= 0 {
		flushInterval = time.Second
	}
	if bufferSize <= 0 {
		bufferSize = 10000
	}

	b := &Batcher{
		entries:       make(chan *storage.LogEntry, bufferSize),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		writeFunc:     writeFunc,
		done:          make(chan struct{}),
		flushReq:      make(chan chan error),
	}

	b.wg.Add(1)
	go b.run()

	return b
}

// Add adds a log entry to the batch buffer.
// If the buffer is full, the entry is dropped and a warning is logged periodically.
func (b *Batcher) Add(entry *storage.LogEntry) {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.mu.Unlock()

	select {
	case b.entries <- entry:
		// Entry added successfully
	default:
		// Buffer is full, drop the entry and track metrics
		dropped := b.droppedCount.Add(1)

		// Log warning periodically (not for every dropped entry to avoid log spam)
		b.droppedWarnMu.Lock()
		if time.Since(b.lastDroppedWarn) > 10*time.Second {
			log.Warn().
				Int64("dropped_count", dropped).
				Int("buffer_size", cap(b.entries)).
				Msg("Log buffer full, entries being dropped. Consider increasing buffer_size or reducing log volume.")
			b.lastDroppedWarn = time.Now()
		}
		b.droppedWarnMu.Unlock()
	}
}

// Flush forces a flush of all buffered entries.
func (b *Batcher) Flush(ctx context.Context) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()

	// Send flush request to run loop and wait for response
	resultCh := make(chan error, 1)

	select {
	case b.flushReq <- resultCh:
		// Request sent, wait for result
		select {
		case err := <-resultCh:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-b.done:
		// Batcher is closing
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close shuts down the batcher, flushing any remaining entries.
func (b *Batcher) Close(ctx context.Context) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()

	// Signal shutdown - run loop will drain and flush remaining entries
	close(b.done)
	b.wg.Wait()

	return nil
}

// run is the main loop that collects and writes batches.
func (b *Batcher) run() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	var batch []*storage.LogEntry

	// Helper to flush current batch
	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := b.writeFunc(ctx, batch)
		cancel()
		batch = nil
		return err
	}

	for {
		select {
		case <-b.done:
			// Shutdown requested
			// Drain any remaining entries from channel
			draining := true
			for draining {
				select {
				case entry := <-b.entries:
					if entry != nil {
						batch = append(batch, entry)
					}
				default:
					draining = false
				}
			}
			// Flush remaining entries
			_ = flushBatch()
			return

		case resultCh := <-b.flushReq:
			// Drain any pending entries from channel first
			draining := true
			for draining {
				select {
				case entry := <-b.entries:
					if entry != nil {
						batch = append(batch, entry)
					}
				default:
					draining = false
				}
			}
			// Flush and send result
			err := flushBatch()
			resultCh <- err

		case entry := <-b.entries:
			if entry == nil {
				continue
			}
			batch = append(batch, entry)

			// Flush if batch is full
			if len(batch) >= b.batchSize {
				_ = flushBatch()
			}

		case <-ticker.C:
			// Flush on interval
			_ = flushBatch()
		}
	}
}

// Stats returns statistics about the batcher.
type BatcherStats struct {
	BufferSize    int
	BufferUsed    int
	BufferPercent float64
	DroppedCount  int64
}

// Stats returns current batcher statistics.
func (b *Batcher) Stats() BatcherStats {
	used := len(b.entries)
	cap := cap(b.entries)
	return BatcherStats{
		BufferSize:    cap,
		BufferUsed:    used,
		BufferPercent: float64(used) / float64(cap) * 100,
		DroppedCount:  b.droppedCount.Load(),
	}
}

// ResetDroppedCount resets the dropped entry counter.
func (b *Batcher) ResetDroppedCount() {
	b.droppedCount.Store(0)
}
