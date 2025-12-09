package runtime

import (
	"context"
	"sync"
)

// CancelSignal is a signal that can be used to cancel an execution
type CancelSignal struct {
	mu        sync.RWMutex
	cancelled bool
	cancel    context.CancelFunc
	ctx       context.Context
}

// NewCancelSignal creates a new cancel signal with a cancellable context
func NewCancelSignal() *CancelSignal {
	ctx, cancel := context.WithCancel(context.Background())
	return &CancelSignal{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Cancel marks the execution as cancelled and cancels the context (which kills the process)
func (s *CancelSignal) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelled = true
	if s.cancel != nil {
		s.cancel()
	}
}

// IsCancelled returns true if the execution was cancelled
func (s *CancelSignal) IsCancelled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cancelled
}

// Context returns the cancellable context for this signal
func (s *CancelSignal) Context() context.Context {
	return s.ctx
}
