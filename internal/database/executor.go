package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Executor defines the interface for database query operations.
// This interface abstracts database access for easier testing via mocks.
type Executor interface {
	// Query executes a query that returns rows
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)

	// QueryRow executes a query that returns a single row
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row

	// Exec executes a query that doesn't return rows
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)

	// BeginTx starts a new transaction
	BeginTx(ctx context.Context) (pgx.Tx, error)

	// Pool returns the underlying connection pool (for advanced operations)
	Pool() *pgxpool.Pool

	// Health checks the health of the database connection
	Health(ctx context.Context) error
}

// AdminExecutor extends Executor with privileged operations that require
// admin database credentials (e.g., for DDL operations, migrations).
type AdminExecutor interface {
	Executor

	// ExecuteWithAdminRole executes a database operation using admin credentials
	// Used for migrations that require DDL privileges (CREATE TABLE, ALTER, etc.)
	ExecuteWithAdminRole(ctx context.Context, fn func(conn *pgx.Conn) error) error

	// Inspector returns the schema inspector for introspecting database structure
	Inspector() *SchemaInspector
}

// Ensure Connection implements both interfaces
var _ Executor = (*Connection)(nil)
var _ AdminExecutor = (*Connection)(nil)
