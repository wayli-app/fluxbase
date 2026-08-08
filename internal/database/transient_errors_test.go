package database

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// IsTransientError Tests
//
// IsTransientError decides whether a DB-dependent startup step should retry.
// It must be true for connection/network problems and transient SQL states, and
// false for logic/config errors (constraint violations, nil) so those fail fast.
// =============================================================================

func TestIsTransientError_Nil(t *testing.T) {
	assert.False(t, IsTransientError(nil))
}

func TestIsTransientError_TransientPGStates(t *testing.T) {
	transient := []string{
		"08000", // connection_exception
		"08001", // sqlclient_unable_to_establish_sqlconnection
		"08003", // connection_does_not_exist
		"08004", // sqlserver_rejected_establishment_of_sqlconnection
		"08006", // connection_failure
		"08007", // transaction_resolution_unknown
		"57P03", // cannot_connect_now
		"40001", // serialization_failure
		"40P01", // deadlock_detected
		"53000", // insufficient_resources (class prefix 53)
		"53100", // disk_full
	}
	for _, code := range transient {
		err := &pgconn.PgError{Code: code, Message: "x"}
		assert.Truef(t, IsTransientError(err), "expected %s to be transient", code)
	}
}

func TestIsTransientError_NonTransientPGStates(t *testing.T) {
	nonTransient := []string{
		"23505", // unique_violation
		"23503", // foreign_key_violation
		"42601", // syntax_error
		"42501", // insufficient_privilege
		"42P01", // undefined_table
	}
	for _, code := range nonTransient {
		err := &pgconn.PgError{Code: code, Message: "x"}
		assert.Falsef(t, IsTransientError(err), "expected %s to NOT be transient", code)
	}
}

func TestIsTransientError_PGErrorWrapped(t *testing.T) {
	// Wrapped PgError must still be detected via errors.As.
	inner := &pgconn.PgError{Code: "08006"}
	wrapped := fmt.Errorf("during apply: %w", inner)
	assert.True(t, IsTransientError(wrapped))
}

func TestIsTransientError_Context(t *testing.T) {
	assert.True(t, IsTransientError(context.DeadlineExceeded))
	assert.False(t, IsTransientError(context.Canceled))
}

func TestIsTransientError_NetworkErrors(t *testing.T) {
	// net.OpError with timeout.
	ne := &net.OpError{Op: "dial", Net: "tcp", Err: &fakeTimeout{}}
	assert.True(t, IsTransientError(ne))

	// Common syscalls / io errors.
	assert.True(t, IsTransientError(syscall.ECONNREFUSED))
	assert.True(t, IsTransientError(syscall.ECONNRESET))
	assert.True(t, IsTransientError(syscall.EPIPE))
	assert.True(t, IsTransientError(io.EOF))
	assert.True(t, IsTransientError(io.ErrUnexpectedEOF))
}

func TestIsTransientError_MessageFragments(t *testing.T) {
	cases := []string{
		"dial tcp 127.0.0.1:5432: connect: connection refused",
		"server closed the connection unexpectedly",
		"the database system is starting up",
		"read tcp: i/o timeout",
		"lookup postgres: no such host",
	}
	for _, msg := range cases {
		assert.Truef(t, IsTransientError(errors.New(msg)), "expected transient: %s", msg)
	}
}

func TestIsTransientError_NonTransientPlainError(t *testing.T) {
	// A plain, non-network, non-pg error with no transient fragments is not
	// retried (fail fast).
	assert.False(t, IsTransientError(errors.New("permission denied for relation users")))
}

// fakeTimeout is a minimal net.Error reporting a timeout, for testing the
// net.Error branch without a real network.
type fakeTimeout struct{}

func (fakeTimeout) Error() string   { return "i/o timeout" }
func (fakeTimeout) Timeout() bool   { return true }
func (fakeTimeout) Temporary() bool { return true }
