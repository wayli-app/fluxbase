package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// hashEmailVerificationToken creates a SHA-256 hash of a token and returns it as base64.
// SECURITY: Email verification tokens are stored as hashes to prevent exposure if the database is breached.
func hashEmailVerificationToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(hash[:])
}

var (
	// ErrEmailVerificationTokenNotFound is returned when a token is not found
	ErrEmailVerificationTokenNotFound = errors.New("email verification token not found")
	// ErrEmailVerificationTokenExpired is returned when a token has expired
	ErrEmailVerificationTokenExpired = errors.New("email verification token has expired")
	// ErrEmailVerificationTokenUsed is returned when a token has already been used
	ErrEmailVerificationTokenUsed = errors.New("email verification token has already been used")
	// ErrEmailNotVerified is returned when a user's email is not verified but verification is required
	ErrEmailNotVerified = errors.New("email not verified")
)

// EmailVerificationToken represents an email verification token
type EmailVerificationToken struct {
	ID        string     `json:"id" db:"id"`
	UserID    string     `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"` // SECURITY: Only hash is stored, never exposed in JSON
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	Used      bool       `json:"used" db:"used"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// EmailVerificationTokenWithPlaintext is returned from Create with the plaintext token for sending via email.
// The plaintext token is never stored in the database.
type EmailVerificationTokenWithPlaintext struct {
	EmailVerificationToken
	PlaintextToken string // The plaintext token to send to the user (not stored)
}

// EmailVerificationRepository handles database operations for email verification tokens
type EmailVerificationRepository struct {
	db *database.Connection
}

// NewEmailVerificationRepository creates a new email verification repository
func NewEmailVerificationRepository(db *database.Connection) *EmailVerificationRepository {
	return &EmailVerificationRepository{db: db}
}

// generateEmailVerificationToken generates a secure random token
func generateEmailVerificationToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Create creates a new email verification token for a user
// SECURITY: The plaintext token is returned only once for sending via email. Only the hash is stored.
func (r *EmailVerificationRepository) Create(ctx context.Context, userID string, expiryDuration time.Duration) (*EmailVerificationTokenWithPlaintext, error) {
	plaintextToken, err := generateEmailVerificationToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Hash the token before storing
	tokenHash := hashEmailVerificationToken(plaintextToken)

	token := &EmailVerificationTokenWithPlaintext{
		EmailVerificationToken: EmailVerificationToken{
			ID:        uuid.New().String(),
			UserID:    userID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(expiryDuration),
			Used:      false,
			CreatedAt: time.Now(),
		},
		PlaintextToken: plaintextToken,
	}

	query := `
		INSERT INTO auth.email_verification_tokens (id, user_id, token_hash, expires_at, used, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, token_hash, expires_at, used, used_at, created_at
	`

	err = database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			token.ID,
			token.UserID,
			token.TokenHash,
			token.ExpiresAt,
			token.Used,
			token.CreatedAt,
		).Scan(
			&token.ID,
			&token.UserID,
			&token.TokenHash,
			&token.ExpiresAt,
			&token.Used,
			&token.UsedAt,
			&token.CreatedAt,
		)
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}

// GetByToken retrieves an email verification token by its plaintext token
// SECURITY: The incoming plaintext token is hashed before lookup
func (r *EmailVerificationRepository) GetByToken(ctx context.Context, token string) (*EmailVerificationToken, error) {
	// Hash the incoming token for comparison
	tokenHash := hashEmailVerificationToken(token)

	query := `
		SELECT id, user_id, token_hash, expires_at, used, used_at, created_at
		FROM auth.email_verification_tokens
		WHERE token_hash = $1
	`

	emailToken := &EmailVerificationToken{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, tokenHash).Scan(
			&emailToken.ID,
			&emailToken.UserID,
			&emailToken.TokenHash,
			&emailToken.ExpiresAt,
			&emailToken.Used,
			&emailToken.UsedAt,
			&emailToken.CreatedAt,
		)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEmailVerificationTokenNotFound
		}
		return nil, err
	}

	return emailToken, nil
}

// Validate validates an email verification token
func (r *EmailVerificationRepository) Validate(ctx context.Context, token string) (*EmailVerificationToken, error) {
	emailToken, err := r.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check if already used
	if emailToken.Used {
		return nil, ErrEmailVerificationTokenUsed
	}

	// Check if expired
	if time.Now().After(emailToken.ExpiresAt) {
		return nil, ErrEmailVerificationTokenExpired
	}

	return emailToken, nil
}

// MarkAsUsed marks an email verification token as used
func (r *EmailVerificationRepository) MarkAsUsed(ctx context.Context, id string) error {
	query := `
		UPDATE auth.email_verification_tokens
		SET used = true, used_at = NOW()
		WHERE id = $1
	`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, id)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return ErrEmailVerificationTokenNotFound
		}

		return nil
	})
}

// DeleteByUserID deletes all email verification tokens for a user
func (r *EmailVerificationRepository) DeleteByUserID(ctx context.Context, userID string) error {
	query := `DELETE FROM auth.email_verification_tokens WHERE user_id = $1`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, userID)
		return err
	})
}

// DeleteExpired deletes all expired email verification tokens
func (r *EmailVerificationRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM auth.email_verification_tokens WHERE expires_at < NOW()`

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
