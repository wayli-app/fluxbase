package auth

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// SetupTOTP generates a new TOTP secret for 2FA
// If issuer is empty, uses the configured default
func (s *DashboardAuthService) SetupTOTP(ctx context.Context, userID uuid.UUID, email string, issuer string) (string, string, error) {
	// Use provided issuer, or fall back to configured default
	if issuer == "" {
		issuer = s.totpIssuer
	}

	// Generate TOTP secret with QR code as data URI
	secret, qrCodeDataURI, _, err := GenerateTOTPSecret(issuer, email)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// Store secret (not yet enabled)
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE platform.users
			SET totp_secret = $1, totp_enabled = false, updated_at = NOW()
			WHERE id = $2
		`, secret, userID)
		return err
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to store TOTP secret: %w", err)
	}

	// Return secret and QR code data URI
	return secret, qrCodeDataURI, nil
}

// EnableTOTP enables 2FA after verifying the TOTP code
func (s *DashboardAuthService) EnableTOTP(ctx context.Context, userID uuid.UUID, code string, ipAddress net.IP, userAgent string) ([]string, error) {
	// Fetch TOTP secret
	var secret string
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT totp_secret FROM platform.users WHERE id = $1 AND deleted_at IS NULL
		`, userID).Scan(&secret)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch TOTP secret: %w", err)
	}

	if secret == "" {
		return nil, errors.New("TOTP not set up")
	}

	// Verify code
	valid := totp.Validate(code, secret)
	if !valid {
		return nil, errors.New("invalid TOTP code")
	}

	// Generate backup codes
	backupCodes := make([]string, 10)
	hashedBackupCodes := make([]string, 10)
	for i := 0; i < 10; i++ {
		code, err := generateBackupCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate backup code: %w", err)
		}
		backupCodes[i] = code

		// Hash the backup code
		hash, err := bcrypt.GenerateFromPassword([]byte(code), DefaultBcryptCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash backup code: %w", err)
		}
		hashedBackupCodes[i] = string(hash)
	}

	// Enable TOTP and store backup codes
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE platform.users
			SET totp_enabled = true, backup_codes = $1, updated_at = NOW()
			WHERE id = $2
		`, hashedBackupCodes, userID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to enable TOTP: %w", err)
	}

	// Log activity
	s.logActivity(ctx, userID, "2fa_enable", "user", userID.String(), ipAddress, userAgent, nil)

	return backupCodes, nil
}

// VerifyTOTP verifies a TOTP code during login
func (s *DashboardAuthService) VerifyTOTP(ctx context.Context, userID uuid.UUID, code string) error {
	// Fetch TOTP secret and backup codes
	var secret string
	var backupCodes []string
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT totp_secret, COALESCE(backup_codes, ARRAY[]::text[])
			FROM platform.users
			WHERE id = $1 AND deleted_at IS NULL AND totp_enabled = true
		`, userID).Scan(&secret, &backupCodes)
	})
	if err != nil {
		return fmt.Errorf("failed to fetch TOTP data: %w", err)
	}

	// Try TOTP code first
	valid := totp.Validate(code, secret)
	if valid {
		return nil
	}

	// Try backup codes
	for i, hashedCode := range backupCodes {
		err := bcrypt.CompareHashAndPassword([]byte(hashedCode), []byte(code))
		if err == nil {
			// Remove used backup code
			backupCodes = append(backupCodes[:i], backupCodes[i+1:]...)
			err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `
					UPDATE platform.users
					SET backup_codes = $1, updated_at = NOW()
					WHERE id = $2
				`, backupCodes, userID)
				return err
			})
			if err != nil {
				return fmt.Errorf("failed to update backup codes: %w", err)
			}
			return nil
		}
	}

	return errors.New("invalid TOTP code")
}

// DisableTOTP disables 2FA for a user
func (s *DashboardAuthService) DisableTOTP(ctx context.Context, userID uuid.UUID, password string, ipAddress net.IP, userAgent string) error {
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

	// Disable TOTP
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE platform.users
			SET totp_enabled = false, totp_secret = NULL, backup_codes = NULL, updated_at = NOW()
			WHERE id = $1
		`, userID)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to disable TOTP: %w", err)
	}

	// Log activity
	s.logActivity(ctx, userID, "2fa_disable", "user", userID.String(), ipAddress, userAgent, nil)

	return nil
}
