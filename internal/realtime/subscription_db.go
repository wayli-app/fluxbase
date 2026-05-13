package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// validIdentifierRegex ensures identifier names are safe PostgreSQL identifiers
var validIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// quoteIdentifier safely quotes a PostgreSQL identifier to prevent SQL injection.
func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

// isValidIdentifier checks if a string is a valid PostgreSQL identifier
func isValidIdentifier(s string) bool {
	return validIdentifierRegex.MatchString(s)
}

// SubscriptionDB defines the database operations needed by SubscriptionManager.
// This interface allows for easier testing with mocks.
type SubscriptionDB interface {
	// IsTableRealtimeEnabled checks if a table is enabled for realtime in the schema registry.
	IsTableRealtimeEnabled(ctx context.Context, schema, table string) (bool, error)
	// CheckRLSAccess verifies if a user can access a record based on RLS policies.
	// The claims map contains the full JWT claims to be passed to PostgreSQL for RLS evaluation.
	CheckRLSAccess(ctx context.Context, schema, table, role string, claims map[string]interface{}, recordID interface{}) (bool, error)
	// CheckRPCOwnership checks if a user owns an RPC execution.
	CheckRPCOwnership(ctx context.Context, execID, userID uuid.UUID) (isOwner bool, exists bool, err error)
	// CheckJobOwnership checks if a user owns a job execution.
	CheckJobOwnership(ctx context.Context, execID, userID uuid.UUID) (isOwner bool, exists bool, err error)
	// CheckFunctionOwnership checks if a user owns a function execution.
	CheckFunctionOwnership(ctx context.Context, execID, userID uuid.UUID) (isOwner bool, exists bool, err error)
}

// pgxSubscriptionDB implements SubscriptionDB using a pgxpool.Pool.
type pgxSubscriptionDB struct {
	pool *pgxpool.Pool
}

// NewPgxSubscriptionDB creates a SubscriptionDB backed by a pgx pool.
func NewPgxSubscriptionDB(pool *pgxpool.Pool) SubscriptionDB {
	return &pgxSubscriptionDB{pool: pool}
}

func (db *pgxSubscriptionDB) IsTableRealtimeEnabled(ctx context.Context, schema, table string) (bool, error) {
	var enabled bool
	err := db.pool.QueryRow(ctx, `
		SELECT realtime_enabled FROM realtime.schema_registry
		WHERE schema_name = $1 AND table_name = $2
	`, schema, table).Scan(&enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return enabled, nil
}

func (db *pgxSubscriptionDB) CheckRLSAccess(ctx context.Context, schema, table, role string, claims map[string]interface{}, recordID interface{}) (bool, error) {
	// Validate schema and table names to prevent SQL injection
	if !isValidIdentifier(schema) {
		return false, fmt.Errorf("invalid schema name: %s", schema)
	}
	if !isValidIdentifier(table) {
		return false, fmt.Errorf("invalid table name: %s", table)
	}

	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Release()

	// Start a transaction for SET LOCAL (required by PostgreSQL)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Use provided claims, ensuring role is set
	jwtClaims := claims
	if jwtClaims == nil {
		jwtClaims = make(map[string]interface{})
	}
	// Ensure role is set in claims for RLS policies that use it
	jwtClaims["role"] = role

	jwtClaimsJSON, err := json.Marshal(jwtClaims)
	if err != nil {
		return false, err
	}

	// Map application role to database role (hardcoded values - safe)
	// Using quoteIdentifier for defense in depth
	dbRole := "authenticated"
	switch role {
	case "service_role":
		dbRole = "service_role"
	case "anon", "":
		dbRole = "anon"
	}

	_, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL ROLE %s", quoteIdentifier(dbRole)))
	if err != nil {
		return false, err
	}

	_, err = tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", string(jwtClaimsJSON))
	if err != nil {
		return false, err
	}

	if tid, ok := claims["tenant_id"].(string); ok && tid != "" {
		_, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tid)
		if err != nil {
			return false, err
		}
	}

	var count int
	// Use quoteIdentifier to prevent SQL injection even though we validated above
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s WHERE id = $1", quoteIdentifier(schema), quoteIdentifier(table))
	err = tx.QueryRow(ctx, query, recordID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (db *pgxSubscriptionDB) CheckRPCOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	var ownerID *uuid.UUID
	err := db.pool.QueryRow(ctx, "SELECT user_id FROM rpc.executions WHERE id = $1", execID).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if ownerID == nil {
		return true, true, nil
	}
	return *ownerID == userID, true, nil
}

func (db *pgxSubscriptionDB) CheckJobOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	var ownerID *uuid.UUID
	err := db.pool.QueryRow(ctx, "SELECT created_by FROM jobs.queue WHERE id = $1", execID).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if ownerID == nil {
		return true, true, nil
	}
	return *ownerID == userID, true, nil
}

func (db *pgxSubscriptionDB) CheckFunctionOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	var ownerID *uuid.UUID
	err := db.pool.QueryRow(ctx, `
		SELECT ef.created_by
		FROM functions.edge_executions ee
		JOIN functions.edge_functions ef ON ee.function_id = ef.id
		WHERE ee.id = $1
	`, execID).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if ownerID == nil {
		return true, true, nil
	}
	return *ownerID == userID, true, nil
}

// copyClaims creates a shallow copy of claims map to prevent concurrent map access during logging.
// This is necessary because zerolog's Interface() iterates over the map, which can race with
// concurrent modifications to the claims map from other goroutines.
func copyClaims(claims map[string]interface{}) map[string]interface{} {
	if claims == nil {
		return nil
	}
	copied := make(map[string]interface{}, len(claims))
	for k, v := range claims {
		copied[k] = v
	}
	return copied
}
