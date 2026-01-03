package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// hashPasswordResetToken creates a SHA-256 hash of a token and returns it as base64.
// SECURITY: Password reset tokens are stored as hashes to prevent exposure if the database is breached.
func hashPasswordResetToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(hash[:])
}

var (
	// ErrPasswordResetTokenNotFound is returned when a password reset token is not found
	ErrPasswordResetTokenNotFound = errors.New("password reset token not found")
	// ErrPasswordResetTokenExpired is returned when a password reset token has expired
	ErrPasswordResetTokenExpired = errors.New("password reset token has expired")
	// ErrPasswordResetTokenUsed is returned when a password reset token has already been used
	ErrPasswordResetTokenUsed = errors.New("password reset token has already been used")
	// ErrSMTPNotConfigured is returned when attempting password reset without SMTP configured
	ErrSMTPNotConfigured = errors.New("SMTP is not configured")
	// ErrEmailSendFailed is returned when the password reset email could not be sent
	ErrEmailSendFailed = errors.New("failed to send password reset email")
	// ErrInvalidRedirectURL is returned when the redirect URL is not a valid URL
	ErrInvalidRedirectURL = errors.New("invalid redirect URL: must be a valid HTTP or HTTPS URL")
)

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID        string     `json:"id" db:"id"`
	UserID    string     `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"` // SECURITY: Only hash is stored, never exposed in JSON
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// PasswordResetTokenWithPlaintext is returned from Create with the plaintext token for sending via email.
// The plaintext token is never stored in the database.
type PasswordResetTokenWithPlaintext struct {
	PasswordResetToken
	PlaintextToken string // The plaintext token to send to the user (not stored)
}

// PasswordResetRepository handles database operations for password reset tokens
type PasswordResetRepository struct {
	db *database.Connection
}

// NewPasswordResetRepository creates a new password reset repository
func NewPasswordResetRepository(db *database.Connection) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

// Create creates a new password reset token
// SECURITY: The plaintext token is returned only once for sending via email. Only the hash is stored.
func (r *PasswordResetRepository) Create(ctx context.Context, userID string, expiryDuration time.Duration) (*PasswordResetTokenWithPlaintext, error) {
	plaintextToken, err := GeneratePasswordResetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Hash the token before storing
	tokenHash := hashPasswordResetToken(plaintextToken)

	passwordResetToken := &PasswordResetTokenWithPlaintext{
		PasswordResetToken: PasswordResetToken{
			ID:        uuid.New().String(),
			UserID:    userID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(expiryDuration),
			CreatedAt: time.Now(),
		},
		PlaintextToken: plaintextToken,
	}

	query := `
		INSERT INTO auth.password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, token_hash, expires_at, used_at, created_at
	`

	err = database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			passwordResetToken.ID,
			passwordResetToken.UserID,
			passwordResetToken.TokenHash,
			passwordResetToken.ExpiresAt,
			passwordResetToken.CreatedAt,
		).Scan(
			&passwordResetToken.ID,
			&passwordResetToken.UserID,
			&passwordResetToken.TokenHash,
			&passwordResetToken.ExpiresAt,
			&passwordResetToken.UsedAt,
			&passwordResetToken.CreatedAt,
		)
	})

	if err != nil {
		return nil, err
	}

	return passwordResetToken, nil
}

// GetByToken retrieves a password reset token by token
// SECURITY: The incoming plaintext token is hashed before lookup
func (r *PasswordResetRepository) GetByToken(ctx context.Context, token string) (*PasswordResetToken, error) {
	// Hash the incoming token for comparison
	tokenHash := hashPasswordResetToken(token)

	query := `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM auth.password_reset_tokens
		WHERE token_hash = $1
	`

	passwordResetToken := &PasswordResetToken{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, tokenHash).Scan(
			&passwordResetToken.ID,
			&passwordResetToken.UserID,
			&passwordResetToken.TokenHash,
			&passwordResetToken.ExpiresAt,
			&passwordResetToken.UsedAt,
			&passwordResetToken.CreatedAt,
		)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPasswordResetTokenNotFound
		}
		return nil, err
	}

	return passwordResetToken, nil
}

// MarkAsUsed marks a password reset token as used
func (r *PasswordResetRepository) MarkAsUsed(ctx context.Context, id string) error {
	query := `
		UPDATE auth.password_reset_tokens
		SET used_at = NOW()
		WHERE id = $1
	`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, id)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return ErrPasswordResetTokenNotFound
		}

		return nil
	})
}

// Validate validates a password reset token
func (r *PasswordResetRepository) Validate(ctx context.Context, token string) (*PasswordResetToken, error) {
	passwordResetToken, err := r.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check if already used
	if passwordResetToken.UsedAt != nil {
		return nil, ErrPasswordResetTokenUsed
	}

	// Check if expired
	if time.Now().After(passwordResetToken.ExpiresAt) {
		return nil, ErrPasswordResetTokenExpired
	}

	return passwordResetToken, nil
}

// DeleteExpired deletes all expired password reset tokens
func (r *PasswordResetRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM auth.password_reset_tokens WHERE expires_at < NOW()`

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

// DeleteByUserID deletes all password reset tokens for a user
func (r *PasswordResetRepository) DeleteByUserID(ctx context.Context, userID string) error {
	query := `DELETE FROM auth.password_reset_tokens WHERE user_id = $1`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, userID)
		return err
	})
}

// GeneratePasswordResetToken generates a secure random token for password resets
func GeneratePasswordResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// PasswordResetEmailSender defines the interface for sending password reset emails
type PasswordResetEmailSender interface {
	SendPasswordReset(ctx context.Context, to, token, link string) error
}

// PasswordResetService provides password reset functionality
type PasswordResetService struct {
	repo        *PasswordResetRepository
	userRepo    *UserRepository
	emailSender PasswordResetEmailSender
	tokenExpiry time.Duration
	baseURL     string
}

// NewPasswordResetService creates a new password reset service
func NewPasswordResetService(
	repo *PasswordResetRepository,
	userRepo *UserRepository,
	emailSender PasswordResetEmailSender,
	tokenExpiry time.Duration,
	baseURL string,
) *PasswordResetService {
	return &PasswordResetService{
		repo:        repo,
		userRepo:    userRepo,
		emailSender: emailSender,
		tokenExpiry: tokenExpiry,
		baseURL:     baseURL,
	}
}

// RequestPasswordReset sends a password reset email to the specified email
// SECURITY: Uses constant-time-ish processing to prevent email enumeration via timing attacks
// If redirectTo is provided, the email link will point to that URL instead of the default baseURL.
func (s *PasswordResetService) RequestPasswordReset(ctx context.Context, email string, redirectTo string) error {
	// Check if email sender is configured
	if s.emailSender == nil {
		return ErrSMTPNotConfigured
	}

	// Validate redirectTo URL if provided (prevents injection attacks in emails)
	if redirectTo != "" {
		parsedURL, err := url.Parse(redirectTo)
		if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			return ErrInvalidRedirectURL
		}
	}

	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if user exists or not (security best practice)
		// SECURITY FIX: Add small random delay to prevent timing-based email enumeration
		// Real requests take longer due to DB operations and email sending
		if errors.Is(err, ErrUserNotFound) {
			// Sleep for a random duration between 100-300ms to mimic real processing time
			time.Sleep(time.Duration(100+time.Now().UnixNano()%200) * time.Millisecond)
			return nil
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Invalidate old password reset tokens for this user
	_ = s.repo.DeleteByUserID(ctx, user.ID)

	// Create new password reset token (returns PasswordResetTokenWithPlaintext containing the plaintext token)
	resetToken, err := s.repo.Create(ctx, user.ID, s.tokenExpiry)
	if err != nil {
		return fmt.Errorf("failed to create password reset token: %w", err)
	}

	// Generate the full link URL using the plaintext token (only available at creation time)
	// Use custom redirectTo if provided, otherwise fall back to default baseURL
	var link string
	if redirectTo != "" {
		link = fmt.Sprintf("%s?token=%s", redirectTo, resetToken.PlaintextToken)
	} else {
		link = fmt.Sprintf("%s/auth/reset-password?token=%s", s.baseURL, resetToken.PlaintextToken)
	}

	// Send email with plaintext token
	if err := s.emailSender.SendPasswordReset(ctx, email, resetToken.PlaintextToken, link); err != nil {
		return fmt.Errorf("%w: %v", ErrEmailSendFailed, err)
	}

	return nil
}

// ResetPassword resets a user's password using a valid reset token
func (s *PasswordResetService) ResetPassword(ctx context.Context, token, newPassword string) (string, error) {
	// Validate the token
	resetToken, err := s.repo.Validate(ctx, token)
	if err != nil {
		return "", err
	}

	// Get the user
	user, err := s.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	// Hash the new password
	hasher := NewPasswordHasher()
	hashedPassword, err := hasher.HashPassword(newPassword)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Update the user's password
	if err := s.userRepo.UpdatePassword(ctx, user.ID, hashedPassword); err != nil {
		return "", fmt.Errorf("failed to update password: %w", err)
	}

	// Mark token as used
	if err := s.repo.MarkAsUsed(ctx, resetToken.ID); err != nil {
		return "", fmt.Errorf("failed to mark token as used: %w", err)
	}

	return user.ID, nil
}

// VerifyPasswordResetToken verifies if a password reset token is valid
func (s *PasswordResetService) VerifyPasswordResetToken(ctx context.Context, token string) error {
	_, err := s.repo.Validate(ctx, token)
	return err
}
