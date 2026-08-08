package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// validatePlatformPasswordLength enforces the dashboard/platform password length
// policy (min and max bounds). It is deliberately length-only: platform user
// flows accept passwords that the stricter PasswordHasher.ValidatePassword
// (which adds upper/lower/digit/symbol complexity rules) would reject.
// Consolidating with ValidatePassword would be a behavior change, not a dedup.
// Extracted from ChangePassword and ResetPassword so the policy is unit-testable.
func validatePlatformPasswordLength(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	if len(password) > MaxPasswordLength {
		return fmt.Errorf("password must be at most %d characters", MaxPasswordLength)
	}
	return nil
}

// ChangePassword changes a dashboard user's password
func (s *DashboardAuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string, ipAddress net.IP, userAgent string) error {
	if err := validatePlatformPasswordLength(newPassword); err != nil {
		return err
	}

	// Fetch current password hash
	var currentHash string
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT password_hash FROM platform.users WHERE id = $1 AND deleted_at IS NULL
		`, userID).Scan(&currentHash)
	})
	if err != nil {
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(currentPassword))
	if err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), DefaultBcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE platform.users
			SET password_hash = $1, updated_at = NOW()
			WHERE id = $2
		`, newHash, userID)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Log activity
	s.logActivity(ctx, userID, "password_change", "user", userID.String(), ipAddress, userAgent, nil)

	return nil
}

// UpdateProfile updates a dashboard user's profile information
func (s *DashboardAuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, fullName string, avatarURL *string) error {
	// Validate full name
	if err := ValidateName(fullName); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	// Validate avatar URL if provided
	if avatarURL != nil {
		if err := ValidateAvatarURL(*avatarURL); err != nil {
			return fmt.Errorf("invalid avatar URL: %w", err)
		}
	}

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE platform.users
			SET full_name = $1, avatar_url = $2, updated_at = NOW()
			WHERE id = $3 AND deleted_at IS NULL
		`, fullName, avatarURL, userID)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	return nil
}

// DeleteAccount soft-deletes a dashboard user account
func (s *DashboardAuthService) DeleteAccount(ctx context.Context, userID uuid.UUID, password string, ipAddress net.IP, userAgent string) error {
	// Verify password
	var passwordHash string
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT password_hash FROM platform.users WHERE id = $1 AND deleted_at IS NULL
		`, userID).Scan(&passwordHash)
	})
	if err != nil {
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		return errors.New("password is incorrect")
	}

	// Soft delete account
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE platform.users
			SET deleted_at = NOW(), updated_at = NOW()
			WHERE id = $1
		`, userID)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	// Delete all sessions
	_ = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			DELETE FROM platform.sessions WHERE user_id = $1
		`, userID)
		return err
	})

	// Log activity
	s.logActivity(ctx, userID, "account_delete", "user", userID.String(), ipAddress, userAgent, nil)

	return nil
}

// RequestPasswordReset creates a password reset token for a dashboard user
// Returns the plaintext token (to be sent via email) or nil if user not found
func (s *DashboardAuthService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	// Find user by email
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	if user == nil {
		// Don't reveal if user exists or not
		return "", nil
	}

	// Generate a secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Hash the token for storage. NOTE: platform reset tokens use hex encoding
	// here, which is intentionally distinct from the base64URL encoding in
	// password_reset.hashPasswordResetToken (app-user flow). The two code paths
	// store into different tables/columns. See that helper's doc comment — do
	// not consolidate without migrating persisted rows.
	tokenHash := sha256.Sum256([]byte(token))
	tokenHashHex := hex.EncodeToString(tokenHash[:])

	// Delete any existing tokens for this user
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM platform.password_reset_tokens WHERE user_id = $1`, user.ID)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("failed to clean up old tokens: %w", err)
	}

	// Create new token (expires in 1 hour)
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO platform.password_reset_tokens (user_id, token, expires_at)
			VALUES ($1, $2, NOW() + INTERVAL '1 hour')
		`, user.ID, tokenHashHex)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("failed to create password reset token: %w", err)
	}

	return token, nil
}

// VerifyPasswordResetToken verifies a password reset token is valid
func (s *DashboardAuthService) VerifyPasswordResetToken(ctx context.Context, token string) (bool, error) {
	// Hash the token for lookup
	tokenHash := sha256.Sum256([]byte(token))
	tokenHashHex := hex.EncodeToString(tokenHash[:])

	var exists bool
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM platform.password_reset_tokens
				WHERE token = $1 AND expires_at > NOW() AND used = false
			)
		`, tokenHashHex).Scan(&exists)
	})
	if err != nil {
		return false, fmt.Errorf("failed to verify token: %w", err)
	}

	return exists, nil
}

// ResetPassword resets a dashboard user's password using a valid reset token
func (s *DashboardAuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if err := validatePlatformPasswordLength(newPassword); err != nil {
		return err
	}

	// Hash the token for lookup
	tokenHash := sha256.Sum256([]byte(token))
	tokenHashHex := hex.EncodeToString(tokenHash[:])

	// Find the token and get user ID
	var userID uuid.UUID
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT user_id FROM platform.password_reset_tokens
			WHERE token = $1 AND expires_at > NOW() AND used = false
		`, tokenHashHex).Scan(&userID)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("invalid or expired password reset token")
		}
		return fmt.Errorf("failed to verify token: %w", err)
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), DefaultBcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password and mark token as used
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		// Update password
		_, err := tx.Exec(ctx, `
			UPDATE platform.users
			SET password_hash = $1, updated_at = NOW()
			WHERE id = $2
		`, newHash, userID)
		if err != nil {
			return err
		}

		// Mark token as used
		_, err = tx.Exec(ctx, `
			UPDATE platform.password_reset_tokens
			SET used = true, used_at = NOW()
			WHERE token = $1
		`, tokenHashHex)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	return nil
}
