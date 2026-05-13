package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// RevokeToken revokes a specific JWT token
func (s *Service) RevokeToken(ctx context.Context, token, reason string) error {
	return s.tokenBlacklistService.RevokeToken(ctx, token, reason)
}

// IsServiceRoleTokenRevoked checks if a service_role token has been emergency revoked
// This provides a mechanism to revoke compromised service_role tokens immediately
// without waiting for token expiry
func (s *Service) IsServiceRoleTokenRevoked(ctx context.Context, jti string) (bool, error) {
	// First check if there's a global revocation (all service_role tokens revoked)
	var globalRevocation bool
	err := database.WrapWithServiceRole(ctx, s.userRepo.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM auth.emergency_revocation
				WHERE revokes_all = TRUE AND expires_at > NOW()
			)
		`).Scan(&globalRevocation)
	})
	if err != nil {
		return false, fmt.Errorf("failed to check global revocation status: %w", err)
	}

	if globalRevocation {
		return true, nil
	}

	// Check if this specific token (JTI) has been revoked
	var tokenRevoked bool
	err = database.WrapWithServiceRole(ctx, s.userRepo.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM auth.emergency_revocation
				WHERE revoked_jti = $1 AND expires_at > NOW()
			)
		`, jti).Scan(&tokenRevoked)
	})
	if err != nil {
		return false, fmt.Errorf("failed to check token revocation status: %w", err)
	}

	return tokenRevoked, nil
}

// EmergencyRevokeAllServiceRoleTokens revokes ALL service_role tokens globally
// This should be used in security emergencies when service_role keys may be compromised
// Returns the ID of the revocation record for audit purposes
func (s *Service) EmergencyRevokeAllServiceRoleTokens(ctx context.Context, revokedBy, reason string) (int64, error) {
	var id int64
	err := database.WrapWithServiceRole(ctx, s.userRepo.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			INSERT INTO auth.emergency_revocation (revokes_all, revoked_by, reason, expires_at)
			VALUES (TRUE, $1, $2, NOW() + INTERVAL '7 days')
			RETURNING id
		`, revokedBy, reason).Scan(&id)
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create emergency revocation: %w", err)
	}

	// Log security event
	LogSecurityWarning(ctx, SecurityEvent{
		Type:   "emergency_revocation",
		UserID: revokedBy,
		Details: map[string]interface{}{
			"revokes_all": true,
			"reason":      reason,
		},
	})

	return id, nil
}

// EmergencyRevokeServiceRoleToken revokes a specific service_role token by JTI
// This allows selective revocation of individual compromised tokens
func (s *Service) EmergencyRevokeServiceRoleToken(ctx context.Context, jti, revokedBy, reason string) error {
	err := database.WrapWithServiceRole(ctx, s.userRepo.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO auth.emergency_revocation (revoked_jti, revoked_by, reason, expires_at)
			VALUES ($1, $2, $3, NOW() + INTERVAL '7 days')
			ON CONFLICT (revoked_jti) DO NOTHING
		`, jti, revokedBy, reason)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to create emergency revocation: %w", err)
	}

	// Log security event
	LogSecurityWarning(ctx, SecurityEvent{
		Type:   "emergency_revocation",
		UserID: revokedBy,
		Details: map[string]interface{}{
			"revoked_jti": jti,
			"reason":      reason,
		},
	})

	return nil
}

// VerifyTOTPWithContext verifies a TOTP code with IP address and user agent for rate limiting
func (s *Service) VerifyTOTPWithContext(ctx context.Context, userID, code, ipAddress, userAgent string) error {
	return s.mfaService.VerifyTOTPWithContext(ctx, userID, code, ipAddress, userAgent)
}

// GenerateTokensForUser generates JWT tokens for a user after successful 2FA verification
func (s *Service) GenerateTokensForUser(ctx context.Context, userID string) (*SignInResponse, error) {
	return s.mfaService.GenerateTokensForUser(ctx, userID)
}
