package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// ValidateToken validates an access token and returns the claims
func (s *Service) ValidateToken(token string) (*TokenClaims, error) {
	return s.jwtManager.ValidateToken(token)
}

// ValidateTokenWithSecret validates an access token using a specific secret key
// This is used for multi-tenant scenarios where each tenant may have a different JWT secret
func (s *Service) ValidateTokenWithSecret(token, secretKey string) (*TokenClaims, error) {
	return s.jwtManager.ValidateTokenWithSecret(token, secretKey)
}

// ValidateServiceRoleToken validates a JWT containing a role claim (anon, service_role, authenticated)
// This is used for client keys which are JWTs with role claims.
// Unlike user tokens, these don't require user lookup or revocation checks.
func (s *Service) ValidateServiceRoleToken(token string) (*TokenClaims, error) {
	return s.jwtManager.ValidateServiceRoleToken(token)
}

// GetOAuthManager returns the OAuth manager for configuring providers
func (s *Service) GetOAuthManager() *OAuthManager {
	return s.oauthManager
}

// RequestPasswordReset sends a password reset email
// If redirectTo is provided, the email link will point to that URL instead of the default.
func (s *Service) RequestPasswordReset(ctx context.Context, email string, redirectTo string) error {
	return s.passwordResetService.RequestPasswordReset(ctx, email, redirectTo)
}

// ResetPassword resets a user's password using a valid reset token
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) (string, error) {
	return s.passwordResetService.ResetPassword(ctx, token, newPassword)
}

// VerifyPasswordResetToken verifies if a password reset token is valid
func (s *Service) VerifyPasswordResetToken(ctx context.Context, token string) error {
	return s.passwordResetService.VerifyPasswordResetToken(ctx, token)
}

// RevokeToken revokes a specific JWT token
func (s *Service) RevokeToken(ctx context.Context, token, reason string) error {
	return s.tokenBlacklistService.RevokeToken(ctx, token, reason)
}

// IsTokenRevoked checks if a JWT token has been revoked
// This is a convenience wrapper that only checks exact JTI revocation
// For full revocation checking including user-wide revocation, use IsTokenRevokedWithClaims
func (s *Service) IsTokenRevoked(ctx context.Context, jti string) (bool, error) {
	return s.tokenBlacklistService.IsTokenRevoked(ctx, jti, "", time.Time{})
}

// IsTokenRevokedWithClaims checks if a JWT token has been revoked
// It checks both exact JTI revocation and user-wide revocation
// This is the preferred method for token revocation checking
func (s *Service) IsTokenRevokedWithClaims(ctx context.Context, jti string, userID string, tokenIssuedAt time.Time) (bool, error) {
	return s.tokenBlacklistService.IsTokenRevoked(ctx, jti, userID, tokenIssuedAt)
}

// RevokeAllUserTokens revokes all tokens for a specific user
func (s *Service) RevokeAllUserTokens(ctx context.Context, userID, reason string) error {
	return s.tokenBlacklistService.RevokeAllUserTokens(ctx, userID, reason)
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

// Impersonation wrapper methods

// StartImpersonation starts an admin impersonation session
func (s *Service) StartImpersonation(ctx context.Context, adminUserID string, tenantID string, req StartImpersonationRequest) (*StartImpersonationResponse, error) {
	return s.impersonationService.StartImpersonation(ctx, adminUserID, tenantID, req)
}

// StopImpersonation stops the active impersonation session for an admin
func (s *Service) StopImpersonation(ctx context.Context, adminUserID string) error {
	return s.impersonationService.StopImpersonation(ctx, adminUserID)
}

// GetActiveImpersonation gets the active impersonation session for an admin
func (s *Service) GetActiveImpersonation(ctx context.Context, adminUserID string) (*ImpersonationSession, error) {
	return s.impersonationService.GetActiveSession(ctx, adminUserID)
}

// ListImpersonationSessions lists impersonation sessions for audit purposes
func (s *Service) ListImpersonationSessions(ctx context.Context, adminUserID string, limit, offset int) ([]*ImpersonationSession, error) {
	return s.impersonationService.ListSessions(ctx, adminUserID, limit, offset)
}

// StartAnonImpersonation starts an impersonation session as anonymous user
func (s *Service) StartAnonImpersonation(ctx context.Context, adminUserID string, tenantID string, reason string, ipAddress string, userAgent string) (*StartImpersonationResponse, error) {
	return s.impersonationService.StartAnonImpersonation(ctx, adminUserID, tenantID, reason, ipAddress, userAgent)
}

// StartServiceImpersonation starts an impersonation session with service role
func (s *Service) StartServiceImpersonation(ctx context.Context, adminUserID string, tenantID string, reason string, ipAddress string, userAgent string) (*StartImpersonationResponse, error) {
	return s.impersonationService.StartServiceImpersonation(ctx, adminUserID, tenantID, reason, ipAddress, userAgent)
}

// MFA/TOTP methods

// SetupTOTP generates a new TOTP secret for 2FA setup
func (s *Service) SetupTOTP(ctx context.Context, userID string, issuer string) (*TOTPSetupResponse, error) {
	return s.mfaService.SetupTOTP(ctx, userID, issuer)
}

// EnableTOTP enables 2FA after verifying the TOTP code
func (s *Service) EnableTOTP(ctx context.Context, userID, code string) ([]string, error) {
	return s.mfaService.EnableTOTP(ctx, userID, code)
}

// VerifyTOTP verifies a TOTP code during login
func (s *Service) VerifyTOTP(ctx context.Context, userID, code string) error {
	return s.mfaService.VerifyTOTP(ctx, userID, code)
}

// VerifyTOTPWithContext verifies a TOTP code with IP address and user agent for rate limiting
func (s *Service) VerifyTOTPWithContext(ctx context.Context, userID, code, ipAddress, userAgent string) error {
	return s.mfaService.VerifyTOTPWithContext(ctx, userID, code, ipAddress, userAgent)
}

// DisableTOTP disables 2FA for a user
func (s *Service) DisableTOTP(ctx context.Context, userID, password string) error {
	return s.mfaService.DisableTOTP(ctx, userID, password)
}

// IsTOTPEnabled checks if 2FA is enabled for a user
func (s *Service) IsTOTPEnabled(ctx context.Context, userID string) (bool, error) {
	return s.mfaService.IsTOTPEnabled(ctx, userID)
}

// GenerateTokensForUser generates JWT tokens for a user after successful 2FA verification
func (s *Service) GenerateTokensForUser(ctx context.Context, userID string) (*SignInResponse, error) {
	return s.mfaService.GenerateTokensForUser(ctx, userID)
}

// Nonce methods

func (s *Service) Reauthenticate(ctx context.Context, userID string) (string, error) {
	return s.nonceService.Reauthenticate(ctx, userID)
}

func (s *Service) VerifyNonce(ctx context.Context, nonce, userID string) bool {
	return s.nonceService.VerifyNonce(ctx, nonce, userID)
}

func (s *Service) CleanupExpiredNonces(ctx context.Context) (int64, error) {
	return s.nonceService.CleanupExpiredNonces(ctx)
}
