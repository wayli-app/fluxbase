package auth

import (
	"context"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/jackc/pgx/v5"
)

// NonceRepository handles database operations for reauthentication nonces.
// This enables stateless multi-instance deployments without sticky sessions.
type NonceRepository struct {
	db *database.Connection
}

// NewNonceRepository creates a new nonce repository
func NewNonceRepository(db *database.Connection) *NonceRepository {
	return &NonceRepository{db: db}
}

// Set stores a nonce with its associated user ID and TTL
func (r *NonceRepository) Set(ctx context.Context, nonce, userID string, ttl time.Duration) error {
	query := `
		INSERT INTO auth.nonces (nonce, user_id, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (nonce) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			expires_at = EXCLUDED.expires_at
	`

	expiresAt := time.Now().Add(ttl)

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, nonce, userID, expiresAt)
		return err
	})
}

// Validate checks if a nonce is valid for the given user and removes it (single-use).
// Returns true if the nonce exists, belongs to the user, and hasn't expired.
// Uses atomic DELETE with RETURNING to ensure single-use semantics across instances.
func (r *NonceRepository) Validate(ctx context.Context, nonce, userID string) (bool, error) {
	// Atomically delete and return the row if it matches all criteria
	// This ensures single-use even with concurrent requests across instances
	query := `
		DELETE FROM auth.nonces
		WHERE nonce = $1
		  AND user_id = $2
		  AND expires_at > NOW()
		RETURNING nonce
	`

	var deletedNonce string
	var valid bool

	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, query, nonce, userID).Scan(&deletedNonce)
		if err == pgx.ErrNoRows {
			// Nonce doesn't exist, doesn't match user, or is expired
			valid = false
			return nil
		}
		if err != nil {
			return err
		}
		valid = true
		return nil
	})

	if err != nil {
		return false, err
	}

	return valid, nil
}

// Cleanup removes expired nonces from the database
func (r *NonceRepository) Cleanup(ctx context.Context) (int64, error) {
	query := `DELETE FROM auth.nonces WHERE expires_at < NOW()`

	var rowsAffected int64
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query)
		if err != nil {
			return err
		}
		rowsAffected = result.RowsAffected()
		return nil
	})

	return rowsAffected, err
}
