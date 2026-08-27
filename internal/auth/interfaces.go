package auth

import (
	"context"
	"time"
)

// EmailService defines the unified interface for email operations.
// All email-related interfaces (FullEmailService, RealEmailService, EmailSender,
// PasswordResetEmailSender) have been consolidated into this single interface.
// The email.Manager / email.Service types satisfy this interface.
type EmailService interface {
	SendMagicLink(ctx context.Context, to, token, link string) error
	SendVerificationEmail(ctx context.Context, to, token, link string) error
	SendPasswordReset(ctx context.Context, to, token, link string) error
	SendInvitationEmail(ctx context.Context, to, inviterName, inviteLink string) error
	Send(ctx context.Context, to, subject, body string) error
	IsConfigured() bool
}

// NOTE ON INTERFACE SIZE:
// These repository interfaces have more than 3 methods, which exceeds the typical
// Go guideline of "keep interfaces small". However, this is acceptable here because:
//
// 1. They are only used for test mocking (production code uses concrete types)
// 2. Each interface represents a cohesive, complete repository (single responsibility)
// 3. All methods are closely related and typically used together
// 4. Splitting would create many small interfaces without clear benefit
//
// If future consumers need only a subset of these methods, smaller interfaces
// can be extracted at that time (following the "accept interfaces, return structs" principle).
//
// See: https://go.dev/blog/laws-of-reflection#use-interfaces-where-appropriate

// UserRepositoryInterface defines user data operations.
// Implementations can be backed by a real database or mocks for testing.
// Note: All IDs are strings (UUID strings) to match existing implementation.
type UserRepositoryInterface interface {
	// Create creates a new user with the given details
	Create(ctx context.Context, req CreateUserRequest, passwordHash string) (*User, error)

	// GetByID retrieves a user by their ID
	GetByID(ctx context.Context, id string) (*User, error)

	// GetByEmail retrieves a user by their email address
	GetByEmail(ctx context.Context, email string) (*User, error)

	// List returns a paginated list of users
	List(ctx context.Context, limit, offset int) ([]*User, error)

	// Update updates a user's details
	Update(ctx context.Context, id string, req UpdateUserRequest) (*User, error)

	// UpdatePassword updates a user's password hash
	UpdatePassword(ctx context.Context, id string, newPasswordHash string) error

	// VerifyEmail marks a user's email as verified
	VerifyEmail(ctx context.Context, id string) error

	// IncrementFailedLoginAttempts increments the failed login counter
	IncrementFailedLoginAttempts(ctx context.Context, userID string) error

	// ResetFailedLoginAttempts resets the failed login counter
	ResetFailedLoginAttempts(ctx context.Context, userID string) error

	// UnlockUser unlocks a locked user account
	UnlockUser(ctx context.Context, userID string) error

	// Delete removes a user by their ID
	Delete(ctx context.Context, id string) error

	// Count returns the total number of users
	Count(ctx context.Context) (int, error)
}

// SessionRepositoryInterface defines session data operations.
// Sessions track active user authentication states.
type SessionRepositoryInterface interface {
	// Create creates a new session for a user
	// Note: Tokens should be hashed before storage
	Create(ctx context.Context, userID, accessToken, refreshToken string, expiresAt time.Time) (*Session, error)

	// GetByAccessToken retrieves a session by its access token
	// Note: Token should be hashed for comparison
	GetByAccessToken(ctx context.Context, accessToken string) (*Session, error)

	// GetByRefreshToken retrieves a session by its refresh token
	// Note: Token should be hashed for comparison
	GetByRefreshToken(ctx context.Context, refreshToken string) (*Session, error)

	// GetByUserID retrieves all sessions for a user
	GetByUserID(ctx context.Context, userID string) ([]*Session, error)

	// UpdateTokens rotates the tokens and expiry for a session.
	// Compare-and-swap: it only applies while the stored refresh-token hash
	// still matches expectedRefreshToken; otherwise ErrRefreshTokenRotated.
	// Note: Tokens should be hashed before storage
	UpdateTokens(ctx context.Context, id, expectedRefreshToken, accessToken, refreshToken string, expiresAt time.Time) error

	// Delete removes a session by ID
	Delete(ctx context.Context, id string) error

	// DeleteByAccessToken removes a session by its access token
	DeleteByAccessToken(ctx context.Context, accessToken string) error

	// DeleteByUserID removes all sessions for a user
	DeleteByUserID(ctx context.Context, userID string) error

	// DeleteExpired removes all expired sessions (cleanup job)
	DeleteExpired(ctx context.Context) (int64, error)

	// Count returns the total number of active sessions
	Count(ctx context.Context) (int, error)
}

// TokenBlacklistRepositoryInterface defines token blacklist operations.
// Used to track revoked tokens that haven't yet expired.
type TokenBlacklistRepositoryInterface interface {
	// Add adds a token to the blacklist
	// jti is the JWT ID from the token claims
	// revokedBy can be nil for tokens without a user (e.g., anonymous or service tokens)
	Add(ctx context.Context, jti string, revokedBy *string, reason string, expiresAt time.Time) error

	// IsBlacklisted checks if a token (by JTI) has been revoked
	IsBlacklisted(ctx context.Context, jti string) (bool, error)

	// GetByJTI retrieves a blacklist entry by JTI
	GetByJTI(ctx context.Context, jti string) (*TokenBlacklistEntry, error)

	// RevokeAllUserTokens blacklists all tokens for a user
	// expiry is the duration for which the revocation marker should persist
	RevokeAllUserTokens(ctx context.Context, userID, reason string, expiry time.Duration) error

	// DeleteExpired removes expired blacklist entries (cleanup job)
	DeleteExpired(ctx context.Context) (int64, error)

	// DeleteByUser removes all blacklist entries for a user
	DeleteByUser(ctx context.Context, userID string) error
}

// MagicLinkRepositoryInterface defines magic link (passwordless) operations.
type MagicLinkRepositoryInterface interface {
	// Create generates a new magic link for an email
	// Returns MagicLinkWithToken containing the plaintext token (for sending via email)
	// SECURITY: Only the hash is stored in the database
	Create(ctx context.Context, email string, expiryDuration time.Duration) (*MagicLinkWithToken, error)

	// GetByToken retrieves a magic link by its token
	// SECURITY: The incoming token is hashed before lookup
	GetByToken(ctx context.Context, token string) (*MagicLink, error)

	// Validate checks if a magic link token is valid and not expired
	Validate(ctx context.Context, token string) (*MagicLink, error)

	// MarkAsUsed marks a magic link as used
	MarkAsUsed(ctx context.Context, id string) error

	// DeleteExpired removes expired magic links (cleanup job)
	DeleteExpired(ctx context.Context) (int64, error)

	// DeleteByEmail removes all magic links for an email
	DeleteByEmail(ctx context.Context, email string) error
}

// PasswordResetRepositoryInterface defines password reset token operations.
type PasswordResetRepositoryInterface interface {
	// Create generates a new password reset token for a user
	// Returns PasswordResetTokenWithPlaintext containing the plaintext token (for sending via email)
	// SECURITY: Only the hash is stored in the database
	Create(ctx context.Context, userID string, expiryDuration time.Duration) (*PasswordResetTokenWithPlaintext, error)

	// GetByToken retrieves a password reset token
	// SECURITY: The incoming token is hashed before lookup
	GetByToken(ctx context.Context, token string) (*PasswordResetToken, error)

	// GetLatestByUserID retrieves the most recent password reset token for a user
	GetLatestByUserID(ctx context.Context, userID string) (*PasswordResetToken, error)

	// Validate checks if a password reset token is valid and not expired
	Validate(ctx context.Context, token string) (*PasswordResetToken, error)

	// MarkAsUsed marks a password reset token as used
	MarkAsUsed(ctx context.Context, id string) error

	// DeleteExpired removes expired tokens (cleanup job)
	DeleteExpired(ctx context.Context) (int64, error)

	// DeleteByUserID removes all reset tokens for a user
	DeleteByUserID(ctx context.Context, userID string) error
}

// OTPRepositoryInterface defines OTP (one-time password) operations.
type OTPRepositoryInterface interface {
	// Create generates a new OTP for email or phone, returning the code with plaintext for sending
	Create(ctx context.Context, email, phone *string, otpType, purpose string, expiryDuration time.Duration) (*OTPCodeWithPlaintext, error)

	// GetByCode retrieves an OTP by its code and contact info
	GetByCode(ctx context.Context, email, phone *string, code string) (*OTPCode, error)

	// Validate checks if an OTP is valid and not expired
	Validate(ctx context.Context, email, phone *string, code string) (*OTPCode, error)

	// IncrementAttempts increments the verification attempt counter
	IncrementAttempts(ctx context.Context, id string) error

	// MarkAsUsed marks an OTP as used
	MarkAsUsed(ctx context.Context, id string) error

	// DeleteExpired removes expired OTPs (cleanup job)
	DeleteExpired(ctx context.Context) (int64, error)

	// DeleteByEmail removes all OTPs for an email
	DeleteByEmail(ctx context.Context, email string) error

	// DeleteByPhone removes all OTPs for a phone number
	DeleteByPhone(ctx context.Context, phone string) error
}

// Ensure concrete implementations satisfy interfaces.
// These compile-time checks help catch interface drift.
var (
	_ UserRepositoryInterface           = (*UserRepository)(nil)
	_ SessionRepositoryInterface        = (*SessionRepository)(nil)
	_ TokenBlacklistRepositoryInterface = (*TokenBlacklistRepository)(nil)
	_ MagicLinkRepositoryInterface      = (*MagicLinkRepository)(nil)
	_ PasswordResetRepositoryInterface  = (*PasswordResetRepository)(nil)
	_ OTPRepositoryInterface            = (*OTPRepository)(nil)
)
