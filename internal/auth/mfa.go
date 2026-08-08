package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/crypto"
	"github.com/nimbleflux/fluxbase/internal/database"
)

type MFAService struct {
	userRepo        *UserRepository
	sessionRepo     *SessionRepository
	jwtManager      *JWTManager
	passwordHasher  *PasswordHasher
	db              *database.Connection
	config          *config.AuthConfig
	encryptionKey   []byte
	totpRateLimiter *TOTPRateLimiter
}

func NewMFAService(
	userRepo *UserRepository,
	sessionRepo *SessionRepository,
	jwtManager *JWTManager,
	passwordHasher *PasswordHasher,
	db *database.Connection,
	cfg *config.AuthConfig,
) *MFAService {
	return &MFAService{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		jwtManager:     jwtManager,
		passwordHasher: passwordHasher,
		db:             db,
		config:         cfg,
	}
}

func (m *MFAService) SetEncryptionKey(key []byte) {
	m.encryptionKey = key
}

func (m *MFAService) SetTOTPRateLimiter(limiter *TOTPRateLimiter) {
	m.totpRateLimiter = limiter
}

// validateTOTPEnable captures the pure, security-relevant preconditions for
// completing TOTP enrollment: the setup must not be expired and an encryption
// key must be configured (enabling TOTP without one would store the secret in
// plaintext). `now` is injected so the expiry check is deterministic in tests.
// Returns nil if both checks pass.
func validateTOTPEnable(expiresAt, now time.Time, encryptionKey []byte) error {
	if now.After(expiresAt) {
		return errors.New("2FA setup has expired, please start again")
	}
	if len(encryptionKey) == 0 {
		return errors.New("TOTP encryption key not configured - cannot store TOTP secrets securely")
	}
	return nil
}

// resolveTOTPSecret recovers the cleartext TOTP secret from its stored form.
//
// If the encryption key is absent, the stored value is treated as already
// cleartext and fellBackToPlaintext is true (decryptErr is nil). If the key is
// present but decryption fails — e.g. the secret was never encrypted, or got
// corrupted — the stored value is again treated as cleartext, fellBackToPlaintext
// is true, and decryptErr carries the failure (for caller-side logging). Only
// a successful decrypt returns fellBackToPlaintext=false.
//
// `decrypt` is injected so the logic is testable without touching the real
// crypto package. Callers pass crypto.DecryptWithBytesKey.
func resolveTOTPSecret(storedSecret string, encryptionKey []byte, decrypt func(string, []byte) (string, error)) (secret string, fellBackToPlaintext bool, decryptErr error) {
	if len(encryptionKey) == 0 {
		return storedSecret, true, nil
	}
	decrypted, err := decrypt(storedSecret, encryptionKey)
	if err != nil {
		return storedSecret, true, err
	}
	return decrypted, false, nil
}

// verifyTOTPDisablePassword enforces the password-reverification gate for
// disabling TOTP. Passwordless accounts (empty PasswordHash, e.g. OAuth/OIDC
// only) bypass the check by design. `cmp` is injected; callers pass
// PasswordHasher.ComparePassword.
func verifyTOTPDisablePassword(passwordHash, password string, cmp func(string, string) error) error {
	if passwordHash == "" {
		// Passwordless account — no password to verify; disable is allowed.
		return nil
	}
	if err := cmp(passwordHash, password); err != nil {
		return errors.New("invalid password")
	}
	return nil
}

type TOTPSetupResponse struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	TOTP struct {
		QRCode string `json:"qr_code"`
		Secret string `json:"secret"`
		URI    string `json:"uri"`
	} `json:"totp"`
}

func (m *MFAService) SetupTOTP(ctx context.Context, userID string, issuer string) (*TOTPSetupResponse, error) {
	if issuer == "" {
		issuer = m.config.TOTPIssuer
	}

	secret, qrCodeDataURI, otpauthURI, err := GenerateTOTPSecret(issuer, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	factorID := uuid.New().String()

	query := `
		INSERT INTO auth.two_factor_setups (user_id, factor_id, secret, qr_code_data_uri, otpauth_uri, expires_at)
		VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '10 minutes')
		ON CONFLICT (user_id) DO UPDATE
			SET factor_id = EXCLUDED.factor_id,
			    secret = EXCLUDED.secret,
			    qr_code_data_uri = EXCLUDED.qr_code_data_uri,
			    otpauth_uri = EXCLUDED.otpauth_uri,
			    expires_at = EXCLUDED.expires_at,
			    verified = FALSE
	`

	_, err = m.db.Exec(ctx, query, userID, factorID, secret, qrCodeDataURI, otpauthURI)
	if err != nil {
		return nil, fmt.Errorf("failed to store TOTP setup: %w", err)
	}

	response := &TOTPSetupResponse{
		ID:   factorID,
		Type: "totp",
	}
	response.TOTP.QRCode = qrCodeDataURI
	response.TOTP.Secret = secret
	response.TOTP.URI = otpauthURI

	return response, nil
}

func (m *MFAService) EnableTOTP(ctx context.Context, userID, code string) ([]string, error) {
	var secret string
	var expiresAt time.Time
	query := `
		SELECT secret, expires_at
		FROM auth.two_factor_setups
		WHERE user_id = $1 AND verified = FALSE
	`

	err := m.db.QueryRow(ctx, query, userID).Scan(&secret, &expiresAt)
	if err != nil {
		return nil, fmt.Errorf("2FA setup not found or expired: %w", err)
	}

	if err := validateTOTPEnable(expiresAt, time.Now(), m.encryptionKey); err != nil {
		return nil, err
	}

	valid, err := VerifyTOTPCode(code, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to verify TOTP code: %w", err)
	}

	if !valid {
		return nil, errors.New("invalid TOTP code")
	}

	backupCodes, hashedCodes, err := GenerateBackupCodes(10)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	encryptedSecret, err := crypto.EncryptWithBytesKey(secret, m.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt TOTP secret: %w", err)
	}
	secretToStore := encryptedSecret

	updateQuery := `
		UPDATE auth.users
		SET totp_secret = $1, totp_enabled = TRUE, backup_codes = $2, updated_at = NOW()
		WHERE id = $3
	`

	_, err = m.db.Exec(ctx, updateQuery, secretToStore, hashedCodes, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to enable TOTP: %w", err)
	}

	_, _ = m.db.Exec(ctx, `
		UPDATE auth.two_factor_setups
		SET verified = TRUE
		WHERE user_id = $1
	`, userID)

	return backupCodes, nil
}

func (m *MFAService) VerifyTOTP(ctx context.Context, userID, code string) error {
	return m.VerifyTOTPWithContext(ctx, userID, code, "", "")
}

func (m *MFAService) VerifyTOTPWithContext(ctx context.Context, userID, code, ipAddress, userAgent string) error {
	if m.totpRateLimiter != nil {
		if err := m.totpRateLimiter.CheckRateLimit(ctx, userID); err != nil {
			return err
		}
	}

	var storedSecret string
	var backupCodes []string
	query := `
		SELECT totp_secret, COALESCE(backup_codes, ARRAY[]::text[])
		FROM auth.users
		WHERE id = $1 AND totp_enabled = TRUE
	`

	err := m.db.QueryRow(ctx, query, userID).Scan(&storedSecret, &backupCodes)
	if err != nil {
		return fmt.Errorf("2FA not enabled for this user: %w", err)
	}

	secret, fellBackToPlaintext, decryptErr := resolveTOTPSecret(storedSecret, m.encryptionKey, crypto.DecryptWithBytesKey)
	if fellBackToPlaintext {
		if len(m.encryptionKey) == 0 {
			log.Warn().Str("user_id", userID).Msg("TOTP encryption key not configured - TOTP secrets may be stored insecurely")
		} else {
			log.Warn().
				Err(decryptErr).
				Str("user_id", userID).
				Msg("TOTP secret decrypted via plaintext fallback - consider migrating to encrypted storage")
		}
	}

	valid, err := VerifyTOTPCode(code, secret)
	if err == nil && valid {
		if m.totpRateLimiter != nil {
			_ = m.totpRateLimiter.RecordAttempt(ctx, userID, true, ipAddress, userAgent)
		}
		return nil
	}

	for i, hashedCode := range backupCodes {
		match, err := VerifyBackupCode(code, hashedCode)
		if err == nil && match {
			backupCodes = append(backupCodes[:i], backupCodes[i+1:]...)

			_, err = m.db.Exec(ctx, `
				UPDATE auth.users
				SET backup_codes = $1, updated_at = NOW()
				WHERE id = $2
			`, backupCodes, userID)
			if err != nil {
				return fmt.Errorf("failed to update backup codes: %w", err)
			}

			_, _ = m.db.Exec(ctx, `
				INSERT INTO auth.two_factor_recovery_attempts (user_id, code_used, success)
				VALUES ($1, $2, TRUE)
			`, userID, "backup_code")

			if m.totpRateLimiter != nil {
				_ = m.totpRateLimiter.RecordAttempt(ctx, userID, true, ipAddress, userAgent)
			}

			return nil
		}
	}

	if m.totpRateLimiter != nil {
		_ = m.totpRateLimiter.RecordAttempt(ctx, userID, false, ipAddress, userAgent)
	} else {
		_, _ = m.db.Exec(ctx, `
			INSERT INTO auth.two_factor_recovery_attempts (user_id, code_used, success)
			VALUES ($1, $2, FALSE)
		`, userID, "totp_code")
	}

	return errors.New("invalid 2FA code")
}

func (m *MFAService) DisableTOTP(ctx context.Context, userID, password string) error {
	user, err := m.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if err := verifyTOTPDisablePassword(user.PasswordHash, password, m.passwordHasher.ComparePassword); err != nil {
		return err
	}

	query := `
		UPDATE auth.users
		SET totp_enabled = FALSE, totp_secret = NULL, backup_codes = NULL, updated_at = NOW()
		WHERE id = $1
	`

	_, err = m.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to disable 2FA: %w", err)
	}

	_, _ = m.db.Exec(ctx, `
		DELETE FROM auth.two_factor_setups WHERE user_id = $1
	`, userID)

	return nil
}

func (m *MFAService) IsTOTPEnabled(ctx context.Context, userID string) (bool, error) {
	var enabled bool
	query := `SELECT COALESCE(totp_enabled, FALSE) FROM auth.users WHERE id = $1`

	err := m.db.QueryRow(ctx, query, userID).Scan(&enabled)
	if err != nil {
		return false, fmt.Errorf("failed to check 2FA status: %w", err)
	}

	return enabled, nil
}

func (m *MFAService) GenerateTokensForUser(ctx context.Context, userID string) (*SignInResponse, error) {
	user, err := m.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	accessToken, refreshToken, _, err := m.jwtManager.GenerateTokenPair(user.ID, user.Email, user.Role, user.UserMetadata, user.AppMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	expiresAt := time.Now().Add(m.config.RefreshExpiry)
	_, err = m.sessionRepo.Create(ctx, user.ID, accessToken, refreshToken, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &SignInResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(m.config.JWTExpiry.Seconds()),
	}, nil
}
