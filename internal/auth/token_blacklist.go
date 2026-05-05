package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// ErrTokenBlacklisted is returned when a token is found in the blacklist
var ErrTokenBlacklisted = errors.New("token has been revoked")

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
		_, err := tx.Exec(
			ctx, query,
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
// The expiry parameter determines how long the revocation marker persists
func (r *TokenBlacklistRepository) RevokeAllUserTokens(ctx context.Context, userID, reason string, expiry time.Duration) error {
	// This is a bit tricky - we can't blacklist tokens we don't know about
	// Instead, we invalidate all the user's sessions
	// The session-based approach is better for "revoke all" scenarios

	// For now, we'll add a marker entry that can be checked
	// A better approach would be to track session revocation separately

	// Use a special JTI pattern for "all tokens" revocation
	specialJTI := "user:" + userID + ":all:" + uuid.New().String()

	return r.Add(ctx, specialJTI, &userID, reason, time.Now().Add(expiry))
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
	// Cache for user-wide revocation checks to avoid per-request DB queries
	userRevocationCache sync.Map // map[userID]*userRevocationCacheEntry
	cacheTTL            time.Duration
}

// userRevocationCacheEntry stores cached user revocation information
type userRevocationCacheEntry struct {
	revokedAt time.Time
	cachedAt  time.Time
}

// NewTokenBlacklistService creates a new token blacklist service
func NewTokenBlacklistService(repo *TokenBlacklistRepository, jwtManager *JWTManager) *TokenBlacklistService {
	return &TokenBlacklistService{
		repo:       repo,
		jwtManager: jwtManager,
		cacheTTL:   5 * time.Second,
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
	// This includes tokens with role "anon", "service_role", or any token validated by ValidateServiceRoleToken
	if _, err := s.jwtManager.ValidateServiceRoleToken(token); err == nil {
		// Block ALL service role tokens (anon, service_role, authenticated via this path)
		// These are system-level credentials that should use emergency revocation instead
		return ErrCannotRevokeServiceRole
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
// It checks both exact JTI revocation and user-wide revocation
func (s *TokenBlacklistService) IsTokenRevoked(ctx context.Context, jti string, userID string, tokenIssuedAt time.Time) (bool, error) {
	// First, check exact JTI match (existing behavior)
	isBlacklisted, err := s.repo.IsBlacklisted(ctx, jti)
	if err != nil {
		return false, fmt.Errorf("failed to check token blacklist: %w", err)
	}
	if isBlacklisted {
		return true, nil
	}

	// If userID is empty, skip user-wide check (backward compatibility)
	if userID == "" {
		return false, nil
	}

	// Check for user-wide revocation: look for any entry matching `user:{userID}:all:*`
	// where created_at >= tokenIssuedAt (token was issued before the revocation)
	revokedAt, err := s.getUserRevocationTime(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check user revocation: %w", err)
	}

	// If a user-wide revocation exists and the token was issued before it, the token is revoked
	if !revokedAt.IsZero() && tokenIssuedAt.Before(revokedAt) {
		return true, nil
	}

	return false, nil
}

// getUserRevocationTime checks the cache first, then the DB for user-wide revocation
// Returns the time of the most recent user-wide revocation, or zero time if none exists
func (s *TokenBlacklistService) getUserRevocationTime(ctx context.Context, userID string) (time.Time, error) {
	// Check cache first
	if cached, ok := s.userRevocationCache.Load(userID); ok {
		entry := cached.(*userRevocationCacheEntry)
		// If cache entry is fresh (within TTL), use it
		if time.Since(entry.cachedAt) < s.cacheTTL {
			// If revokedAt is zero, no revocation exists
			return entry.revokedAt, nil
		}
		// Cache expired, remove it
		s.userRevocationCache.Delete(userID)
	}

	// Cache miss or expired - query the database
	// Look for the most recent entry matching `user:{userID}:all:*`
	var revokedAt time.Time
	query := `
		SELECT created_at
		FROM auth.token_blacklist
		WHERE token_jti LIKE $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	pattern := "user:" + userID + ":all:*"

	err := database.WrapWithServiceRole(ctx, s.repo.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, pattern).Scan(&revokedAt)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No user-wide revocation found - cache this fact
			s.userRevocationCache.Store(userID, &userRevocationCacheEntry{
				revokedAt: time.Time{},
				cachedAt:  time.Now(),
			})
			return time.Time{}, nil
		}
		return time.Time{}, err
	}

	// Cache the revocation time
	s.userRevocationCache.Store(userID, &userRevocationCacheEntry{
		revokedAt: revokedAt,
		cachedAt:  time.Now(),
	})

	return revokedAt, nil
}

// invalidateUserCache removes the cache entry for a specific user
// Should be called when RevokeAllUserTokens is invoked
func (s *TokenBlacklistService) invalidateUserCache(userID string) {
	s.userRevocationCache.Delete(userID)
}

// RevokeAllUserTokens revokes all tokens for a user
// The revocation marker expires after the maximum token TTL to ensure it covers
// all valid tokens at the time of revocation
func (s *TokenBlacklistService) RevokeAllUserTokens(ctx context.Context, userID, reason string) error {
	// Invalidate the cache for this user
	s.invalidateUserCache(userID)

	return s.repo.RevokeAllUserTokens(ctx, userID, reason, s.jwtManager.getMaxTokenTTL())
}

// CleanupExpiredTokens removes expired tokens from the blacklist
func (s *TokenBlacklistService) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	return s.repo.DeleteExpired(ctx)
}
