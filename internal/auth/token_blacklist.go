package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

var (
	// ErrTokenBlacklisted is returned when a token is found in the blacklist
	ErrTokenBlacklisted = errors.New("token has been revoked")
)

// TokenBlacklistEntry represents a blacklisted token
type TokenBlacklistEntry struct {
	ID        string    `json:"id" db:"id"`
	TokenJTI  string    `json:"token_jti" db:"token_jti"`
	RevokedBy string    `json:"revoked_by" db:"revoked_by"`
	Reason    string    `json:"reason" db:"reason"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
}

// TokenBlacklistRepository handles database operations for token blacklist
type TokenBlacklistRepository struct {
	db *database.Connection
}

// NewTokenBlacklistRepository creates a new token blacklist repository
func NewTokenBlacklistRepository(db *database.Connection) *TokenBlacklistRepository {
	return &TokenBlacklistRepository{db: db}
}

// Add adds a token to the blacklist. revokedBy can be nil for tokens without a user.
func (r *TokenBlacklistRepository) Add(ctx context.Context, jti string, revokedBy *string, reason string, expiresAt time.Time) error {
	query := `
		INSERT INTO auth.token_blacklist (id, token_jti, revoked_by, reason, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (token_jti) DO NOTHING
	`

	logEvent := log.Info().Str("jti", jti).Str("reason", reason)
	if revokedBy != nil {
		logEvent = logEvent.Str("revoked_by", *revokedBy)
	}
	logEvent.Msg("Blacklisting token")

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query,
			uuid.New().String(),
			jti,
			revokedBy,
			reason,
			expiresAt,
		)
		return err
	})
}

// IsBlacklisted checks if a token JTI is in the blacklist
func (r *TokenBlacklistRepository) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM auth.token_blacklist
			WHERE token_jti = $1
		)
	`

	var exists bool
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, jti).Scan(&exists)
	})
	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetByJTI retrieves a blacklist entry by token JTI
func (r *TokenBlacklistRepository) GetByJTI(ctx context.Context, jti string) (*TokenBlacklistEntry, error) {
	query := `
		SELECT id, token_jti, revoked_by, reason, created_at, expires_at
		FROM auth.token_blacklist
		WHERE token_jti = $1
	`

	entry := &TokenBlacklistEntry{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, jti).Scan(
			&entry.ID,
			&entry.TokenJTI,
			&entry.RevokedBy,
			&entry.Reason,
			&entry.CreatedAt,
			&entry.ExpiresAt,
		)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return entry, nil
}

// RevokeAllUserTokens revokes all tokens for a specific user
func (r *TokenBlacklistRepository) RevokeAllUserTokens(ctx context.Context, userID, reason string) error {
	// This is a bit tricky - we can't blacklist tokens we don't know about
	// Instead, we invalidate all the user's sessions
	// The session-based approach is better for "revoke all" scenarios

	// For now, we'll add a marker entry that can be checked
	// A better approach would be to track session revocation separately

	// Use a special JTI pattern for "all tokens" revocation
	specialJTI := "user:" + userID + ":all:" + uuid.New().String()

	return r.Add(ctx, specialJTI, &userID, reason, time.Now().Add(24*time.Hour))
}

// DeleteExpired removes expired tokens from the blacklist
func (r *TokenBlacklistRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM auth.token_blacklist WHERE expires_at < NOW()`

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

// DeleteByUser removes all blacklist entries for a user
func (r *TokenBlacklistRepository) DeleteByUser(ctx context.Context, userID string) error {
	query := `DELETE FROM auth.token_blacklist WHERE revoked_by = $1`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, userID)
		return err
	})
}

// TokenBlacklistService provides token blacklisting/revocation functionality
type TokenBlacklistService struct {
	repo       *TokenBlacklistRepository
	jwtManager *JWTManager
}

// NewTokenBlacklistService creates a new token blacklist service
func NewTokenBlacklistService(repo *TokenBlacklistRepository, jwtManager *JWTManager) *TokenBlacklistService {
	return &TokenBlacklistService{
		repo:       repo,
		jwtManager: jwtManager,
	}
}

// ErrCannotRevokeServiceRole is returned when attempting to revoke a service role token
var ErrCannotRevokeServiceRole = errors.New("cannot revoke service role tokens")

// ErrCannotRevokeServiceKey is returned when attempting to blacklist a service key
var ErrCannotRevokeServiceKey = errors.New("cannot blacklist service keys - use disable endpoint instead")

// ErrCannotRevokeClientKey is returned when attempting to blacklist a client key
var ErrCannotRevokeClientKey = errors.New("cannot blacklist client keys - use revoke endpoint instead")

// RevokeToken revokes a specific token
func (s *TokenBlacklistService) RevokeToken(ctx context.Context, token, reason string) error {
	// Service keys (sk_) should never be blacklisted - they have their own disable mechanism
	if strings.HasPrefix(token, "sk_") {
		return ErrCannotRevokeServiceKey
	}

	// Client keys (fbk_) should never be blacklisted - they have their own revoke mechanism
	if strings.HasPrefix(token, "fbk_") {
		return ErrCannotRevokeClientKey
	}

	// Service role tokens should never be revoked - they are system-level credentials
	if serviceRoleClaims, err := s.jwtManager.ValidateServiceRoleToken(token); err == nil {
		if serviceRoleClaims.Role == "service_role" {
			return ErrCannotRevokeServiceRole
		}
	}

	// Validate and parse the token to get the JTI
	claims, err := s.jwtManager.ValidateToken(token)
	if err != nil {
		// If token is already expired or invalid, no need to blacklist
		if errors.Is(err, ErrExpiredToken) {
			return nil
		}
		return err
	}

	// Add to blacklist (convert empty userID to nil for NULL in database)
	var revokedBy *string
	if claims.UserID != "" {
		revokedBy = &claims.UserID
	}
	return s.repo.Add(ctx, claims.ID, revokedBy, reason, claims.ExpiresAt.Time)
}

// IsTokenRevoked checks if a token has been revoked
func (s *TokenBlacklistService) IsTokenRevoked(ctx context.Context, jti string) (bool, error) {
	return s.repo.IsBlacklisted(ctx, jti)
}

// RevokeAllUserTokens revokes all tokens for a user
func (s *TokenBlacklistService) RevokeAllUserTokens(ctx context.Context, userID, reason string) error {
	return s.repo.RevokeAllUserTokens(ctx, userID, reason)
}

// CleanupExpiredTokens removes expired tokens from the blacklist
func (s *TokenBlacklistService) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	return s.repo.DeleteExpired(ctx)
}
