package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wayli-app/fluxbase/internal/database"
)

var (
	// ErrPasswordResetTokenNotFound is returned when a password reset token is not found
	ErrPasswordResetTokenNotFound = errors.New("password reset token not found")
	// ErrPasswordResetTokenExpired is returned when a password reset token has expired
	ErrPasswordResetTokenExpired = errors.New("password reset token has expired")
	// ErrPasswordResetTokenUsed is returned when a password reset token has already been used
	ErrPasswordResetTokenUsed = errors.New("password reset token has already been used")
)

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID        string     `json:"id" db:"id"`
	UserID    string     `json:"user_id" db:"user_id"`
	Token     string     `json:"token" db:"token"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
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
func (r *PasswordResetRepository) Create(ctx context.Context, userID string, expiryDuration time.Duration) (*PasswordResetToken, error) {
	token, err := GeneratePasswordResetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	passwordResetToken := &PasswordResetToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(expiryDuration),
		CreatedAt: time.Now(),
	}

	query := `
		INSERT INTO auth.password_reset_tokens (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, token, expires_at, used_at, created_at
	`

	err = r.db.QueryRow(ctx, query,
		passwordResetToken.ID,
		passwordResetToken.UserID,
		passwordResetToken.Token,
		passwordResetToken.ExpiresAt,
		passwordResetToken.CreatedAt,
	).Scan(
		&passwordResetToken.ID,
		&passwordResetToken.UserID,
		&passwordResetToken.Token,
		&passwordResetToken.ExpiresAt,
		&passwordResetToken.UsedAt,
		&passwordResetToken.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return passwordResetToken, nil
}

// GetByToken retrieves a password reset token by token
func (r *PasswordResetRepository) GetByToken(ctx context.Context, token string) (*PasswordResetToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, used_at, created_at
		FROM auth.password_reset_tokens
		WHERE token = $1
	`

	passwordResetToken := &PasswordResetToken{}
	err := r.db.QueryRow(ctx, query, token).Scan(
		&passwordResetToken.ID,
		&passwordResetToken.UserID,
		&passwordResetToken.Token,
		&passwordResetToken.ExpiresAt,
		&passwordResetToken.UsedAt,
		&passwordResetToken.CreatedAt,
	)

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

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrPasswordResetTokenNotFound
	}

	return nil
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

	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

// DeleteByUserID deletes all password reset tokens for a user
func (r *PasswordResetRepository) DeleteByUserID(ctx context.Context, userID string) error {
	query := `DELETE FROM auth.password_reset_tokens WHERE user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	return err
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
func (s *PasswordResetService) RequestPasswordReset(ctx context.Context, email string) error {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if user exists or not (security best practice)
		if errors.Is(err, ErrUserNotFound) {
			return nil
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Invalidate old password reset tokens for this user
	_ = s.repo.DeleteByUserID(ctx, user.ID)

	// Create new password reset token
	resetToken, err := s.repo.Create(ctx, user.ID, s.tokenExpiry)
	if err != nil {
		return fmt.Errorf("failed to create password reset token: %w", err)
	}

	// Generate the full link URL
	link := fmt.Sprintf("%s/auth/reset-password?token=%s", s.baseURL, resetToken.Token)

	// Send email
	if err := s.emailSender.SendPasswordReset(ctx, email, resetToken.Token, link); err != nil {
		return fmt.Errorf("failed to send password reset email: %w", err)
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
