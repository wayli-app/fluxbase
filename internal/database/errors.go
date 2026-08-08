package database

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL error codes
const (
	// ErrCodeUniqueViolation is the PostgreSQL error code for unique constraint violations
	ErrCodeUniqueViolation = "23505"
	// ErrCodeForeignKeyViolation is the PostgreSQL error code for foreign key violations
	ErrCodeForeignKeyViolation = "23503"
	// ErrCodeCheckViolation is the PostgreSQL error code for check constraint violations
	ErrCodeCheckViolation = "23514"
)

// transientSQLStateCodes are PostgreSQL SQLSTATE codes that indicate a transient
// (retryable) failure rather than a logic/config error: the connection / admin
// state is temporarily unavailable. See https://www.postgresql.org/docs/current/errcodes-appendix.html
var transientSQLStateCodes = map[string]struct{}{
	// Class 08 — Connection Exception
	"08000": {}, // connection_exception
	"08003": {}, // connection_does_not_exist
	"08006": {}, // connection_failure
	"08001": {}, // sqlclient_unable_to_establish_sqlconnection
	"08004": {}, // sqlserver_rejected_establishment_of_sqlconnection
	"08007": {}, // transaction_resolution_unknown
	// Class 57 — Operator Intervention
	"57P03": {}, // cannot_connect_now
	"55006": {}, // object_in_use (transient during DDL)
	// Class 40 — Transaction Rollback
	"40001": {}, // serialization_failure
	"40P01": {}, // deadlock_detected
	"53":    {}, // Class 53 — Insufficient Resources (prefix match handled separately)
}

// isTransientSQLState reports whether a SQLSTATE code is in the transient set,
// including a prefix match for class 53 (insufficient_resources, 53000-53114).
func isTransientSQLState(code string) bool {
	if _, ok := transientSQLStateCodes[code]; ok {
		return true
	}
	// Class 53 — insufficient resources (53000..53114) are all retryable.
	if len(code) == 5 && code[:2] == "53" {
		return true
	}
	return false
}

// IsTransientError reports whether err represents a failure that may succeed on
// retry: connection/network problems, the server temporarily refusing
// connections (e.g. still starting up), context deadlines during connect, or
// transient SQL states (serialization/deadlock/resource limits). It is used to
// decide whether to retry DB-dependent startup steps instead of aborting.
//
// Non-transient errors (constraint violations, syntax errors, permission
// errors, nil) report false so they fail fast rather than retrying forever.
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Wrapped SQLSTATE.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return isTransientSQLState(pgErr.Code)
	}

	// Context deadline during a connect/operation is transient for startup.
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Network-layer errors: timeouts, refused, reset, closed, EOF.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	// pgx/pq connection errors often arrive as opaque strings wrapping these
	// substrings (e.g. "failed to connect to ... connection refused"). Match
	// conservatively to cover pool/wrapper errors that don't carry a typed cause.
	msg := strings.ToLower(err.Error())
	for _, frag := range transientErrorFragments {
		if strings.Contains(msg, frag) {
			return true
		}
	}

	return false
}

// transientErrorFragments are message substrings that indicate a transient
// connection failure when the underlying error is not a typed net/syscall error.
var transientErrorFragments = []string{
	"connection refused",
	"connection reset",
	"broken pipe",
	"timeout",
	"timed out",
	"no such host", // DNS hiccup
	"server closed the connection unexpectedly",
	"the database system is starting up",
	"cannot connect now",
	"eof",
	"i/o timeout",
}

// IsUniqueViolation checks if an error is a unique constraint violation
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == ErrCodeUniqueViolation
	}
	return false
}

// IsForeignKeyViolation checks if an error is a foreign key violation
func IsForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == ErrCodeForeignKeyViolation
	}
	return false
}

// IsCheckViolation checks if an error is a check constraint violation
func IsCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == ErrCodeCheckViolation
	}
	return false
}

// GetConstraintName returns the constraint name from a PostgreSQL error
func GetConstraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}
