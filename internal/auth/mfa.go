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

	if time.Now().After(expiresAt) {
		return nil, errors.New("2FA setup has expired, please start again")
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

	if len(m.encryptionKey) == 0 {
		return nil, errors.New("TOTP encryption key not configured - cannot store TOTP secrets securely")
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

	secret := storedSecret
	if len(m.encryptionKey) == 0 {
		log.Warn().Str("user_id", userID).Msg("TOTP encryption key not configured - TOTP secrets may be stored insecurely")
	} else {
		decrypted, err := crypto.DecryptWithBytesKey(storedSecret, m.encryptionKey)
		if err != nil {
			log.Warn().
				Err(err).
				Str("user_id", userID).
				Msg("TOTP secret decrypted via plaintext fallback - consider migrating to encrypted storage")
		} else {
			secret = decrypted
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

	if user.PasswordHash != "" {
		err := m.passwordHasher.ComparePassword(user.PasswordHash, password)
		if err != nil {
			return errors.New("invalid password")
		}
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
