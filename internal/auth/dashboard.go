package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

// DashboardUser represents a dashboard/platform administrator user
type DashboardUser struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	FullName      *string    `json:"full_name,omitempty"`
	AvatarURL     *string    `json:"avatar_url,omitempty"`
	TOTPEnabled   bool       `json:"totp_enabled"`
	IsActive      bool       `json:"is_active"`
	IsLocked      bool       `json:"is_locked"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// DashboardSession represents an active dashboard session
type DashboardSession struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	TokenHash      string    `json:"-"`
	IPAddress      *net.IP   `json:"ip_address,omitempty"`
	UserAgent      *string   `json:"user_agent,omitempty"`
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
}

// DashboardAuthService handles authentication for dashboard administrators
type DashboardAuthService struct {
	db         *pgxpool.Pool
	jwtManager *JWTManager
	totpIssuer string // Default TOTP issuer for 2FA
}

// NewDashboardAuthService creates a new dashboard authentication service
func NewDashboardAuthService(db *pgxpool.Pool, jwtManager *JWTManager, totpIssuer string) *DashboardAuthService {
	return &DashboardAuthService{
		db:         db,
		jwtManager: jwtManager,
		totpIssuer: totpIssuer,
	}
}

// GetDB returns the database connection pool
func (s *DashboardAuthService) GetDB() *pgxpool.Pool {
	return s.db
}

// CreateUser creates a new dashboard user with email and password
func (s *DashboardAuthService) CreateUser(ctx context.Context, email, password, fullName string) (*DashboardUser, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &DashboardUser{}
	err = s.db.QueryRow(ctx, `
		INSERT INTO dashboard.users (email, password_hash, full_name, email_verified)
		VALUES ($1, $2, $3, false)
		RETURNING id, email, email_verified, full_name, avatar_url, totp_enabled,
		          is_active, is_locked, last_login_at, created_at, updated_at
	`, email, hashedPassword, fullName).Scan(
		&user.ID, &user.Email, &user.EmailVerified, &user.FullName, &user.AvatarURL,
		&user.TOTPEnabled, &user.IsActive, &user.IsLocked, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// HasExistingUsers checks if any dashboard users exist
func (s *DashboardAuthService) HasExistingUsers(ctx context.Context) (bool, error) {
	var count int
	err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM dashboard.users WHERE deleted_at IS NULL`).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check existing users: %w", err)
	}
	return count > 0, nil
}

// LoginResponse contains the tokens returned from login
type LoginResponse struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// Login authenticates a dashboard user with email and password
func (s *DashboardAuthService) Login(ctx context.Context, email, password string, ipAddress net.IP, userAgent string) (*DashboardUser, *LoginResponse, error) {
	// Fetch user with password hash
	var user DashboardUser
	var passwordHash string
	var failedAttempts int

	err := s.db.QueryRow(ctx, `
		SELECT id, email, email_verified, password_hash, full_name, avatar_url,
		       totp_enabled, is_active, is_locked, failed_login_attempts,
		       last_login_at, created_at, updated_at
		FROM dashboard.users
		WHERE email = $1 AND deleted_at IS NULL
	`, email).Scan(
		&user.ID, &user.Email, &user.EmailVerified, &passwordHash, &user.FullName,
		&user.AvatarURL, &user.TOTPEnabled, &user.IsActive, &user.IsLocked,
		&failedAttempts, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, errors.New("invalid credentials")
		}
		return nil, nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Check if account is locked
	if user.IsLocked {
		return nil, nil, errors.New("account is locked")
	}

	// Check if account is active
	if !user.IsActive {
		return nil, nil, errors.New("account is inactive")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		// Increment failed login attempts
		_, _ = s.db.Exec(ctx, `
			UPDATE dashboard.users
			SET failed_login_attempts = failed_login_attempts + 1,
			    is_locked = CASE WHEN failed_login_attempts >= 4 THEN true ELSE false END
			WHERE id = $1
		`, user.ID)
		return nil, nil, errors.New("invalid credentials")
	}

	// Reset failed attempts on successful login
	_, err = s.db.Exec(ctx, `
		UPDATE dashboard.users
		SET failed_login_attempts = 0,
		    last_login_at = NOW()
		WHERE id = $1
	`, user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update login timestamp: %w", err)
	}

	// Generate JWT token pair (access + refresh) - dashboard users don't need metadata for now
	accessToken, refreshToken, sessionID, err := s.jwtManager.GenerateTokenPair(user.ID.String(), user.Email, "dashboard_admin", nil, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Hash the access token using SHA-256
	hash := sha256.Sum256([]byte(accessToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Handle nil IP address
	var ipAddressStr interface{}
	if ipAddress != nil {
		ipAddressStr = ipAddress.String()
	}

	// Delete any existing sessions for this user (allow only one active session)
	_, err = s.db.Exec(ctx, `
		DELETE FROM dashboard.sessions WHERE user_id = $1
	`, user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to clean up old sessions: %w", err)
	}

	// Create new session record with session ID from token
	_, err = s.db.Exec(ctx, `
		INSERT INTO dashboard.sessions (id, user_id, token, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '24 hours')
	`, sessionID, user.ID, tokenHash, ipAddressStr, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Log activity
	s.logActivity(ctx, user.ID, "login", "", "", ipAddress, userAgent, nil)

	// Return user and tokens
	return &user, &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(24 * 60 * 60), // 24 hours in seconds
	}, nil
}

// ChangePassword changes a dashboard user's password
func (s *DashboardAuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string, ipAddress net.IP, userAgent string) error {
	// Fetch current password hash
	var currentHash string
	err := s.db.QueryRow(ctx, `
		SELECT password_hash FROM dashboard.users WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(&currentHash)
	if err != nil {
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(currentPassword))
	if err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	_, err = s.db.Exec(ctx, `
		UPDATE dashboard.users
		SET password_hash = $1, updated_at = NOW()
		WHERE id = $2
	`, newHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Log activity
	s.logActivity(ctx, userID, "password_change", "user", userID.String(), ipAddress, userAgent, nil)

	return nil
}

// UpdateProfile updates a dashboard user's profile information
func (s *DashboardAuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, fullName string, avatarURL *string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE dashboard.users
		SET full_name = $1, avatar_url = $2, updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
	`, fullName, avatarURL, userID)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	return nil
}

// DeleteAccount soft-deletes a dashboard user account
func (s *DashboardAuthService) DeleteAccount(ctx context.Context, userID uuid.UUID, password string, ipAddress net.IP, userAgent string) error {
	// Verify password
	var passwordHash string
	err := s.db.QueryRow(ctx, `
		SELECT password_hash FROM dashboard.users WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(&passwordHash)
	if err != nil {
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		return errors.New("password is incorrect")
	}

	// Soft delete account
	_, err = s.db.Exec(ctx, `
		UPDATE dashboard.users
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	// Delete all sessions
	_, _ = s.db.Exec(ctx, `
		DELETE FROM dashboard.sessions WHERE user_id = $1
	`, userID)

	// Log activity
	s.logActivity(ctx, userID, "account_delete", "user", userID.String(), ipAddress, userAgent, nil)

	return nil
}

// SetupTOTP generates a new TOTP secret for 2FA
// If issuer is empty, uses the configured default
func (s *DashboardAuthService) SetupTOTP(ctx context.Context, userID uuid.UUID, email string, issuer string) (string, string, error) {
	// Use provided issuer, or fall back to configured default
	if issuer == "" {
		issuer = s.totpIssuer
	}

	// Generate TOTP secret
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: email,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	secret := key.Secret()

	// Store secret (not yet enabled)
	_, err = s.db.Exec(ctx, `
		UPDATE dashboard.users
		SET totp_secret = $1, totp_enabled = false, updated_at = NOW()
		WHERE id = $2
	`, secret, userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to store TOTP secret: %w", err)
	}

	// Return secret and QR code URL
	return secret, key.URL(), nil
}

// EnableTOTP enables 2FA after verifying the TOTP code
func (s *DashboardAuthService) EnableTOTP(ctx context.Context, userID uuid.UUID, code string, ipAddress net.IP, userAgent string) ([]string, error) {
	// Fetch TOTP secret
	var secret string
	err := s.db.QueryRow(ctx, `
		SELECT totp_secret FROM dashboard.users WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(&secret)
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
		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash backup code: %w", err)
		}
		hashedBackupCodes[i] = string(hash)
	}

	// Enable TOTP and store backup codes
	_, err = s.db.Exec(ctx, `
		UPDATE dashboard.users
		SET totp_enabled = true, backup_codes = $1, updated_at = NOW()
		WHERE id = $2
	`, hashedBackupCodes, userID)
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
	err := s.db.QueryRow(ctx, `
		SELECT totp_secret, COALESCE(backup_codes, ARRAY[]::text[])
		FROM dashboard.users
		WHERE id = $1 AND deleted_at IS NULL AND totp_enabled = true
	`, userID).Scan(&secret, &backupCodes)
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
			newBackupCodes := append(backupCodes[:i], backupCodes[i+1:]...)
			_, err = s.db.Exec(ctx, `
				UPDATE dashboard.users
				SET backup_codes = $1, updated_at = NOW()
				WHERE id = $2
			`, newBackupCodes, userID)
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
	err := s.db.QueryRow(ctx, `
		SELECT password_hash FROM dashboard.users WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(&passwordHash)
	if err != nil {
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		return errors.New("password is incorrect")
	}

	// Disable TOTP
	_, err = s.db.Exec(ctx, `
		UPDATE dashboard.users
		SET totp_enabled = false, totp_secret = NULL, backup_codes = NULL, updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to disable TOTP: %w", err)
	}

	// Log activity
	s.logActivity(ctx, userID, "2fa_disable", "user", userID.String(), ipAddress, userAgent, nil)

	return nil
}

// GetUserByID fetches a dashboard user by ID
func (s *DashboardAuthService) GetUserByID(ctx context.Context, userID uuid.UUID) (*DashboardUser, error) {
	user := &DashboardUser{}
	err := s.db.QueryRow(ctx, `
		SELECT id, email, email_verified, full_name, avatar_url, totp_enabled,
		       is_active, is_locked, last_login_at, created_at, updated_at
		FROM dashboard.users
		WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(
		&user.ID, &user.Email, &user.EmailVerified, &user.FullName, &user.AvatarURL,
		&user.TOTPEnabled, &user.IsActive, &user.IsLocked, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	return user, nil
}

// logActivity logs a dashboard user activity
func (s *DashboardAuthService) logActivity(ctx context.Context, userID uuid.UUID, action, resourceType, resourceID string, ipAddress net.IP, userAgent string, metadata map[string]interface{}) {
	// Convert empty strings to nil for nullable fields
	var resourceTypePtr *string
	if resourceType != "" {
		resourceTypePtr = &resourceType
	}

	var resourceIDPtr *string
	if resourceID != "" {
		resourceIDPtr = &resourceID
	}

	// Handle nil IP address
	var ipAddressStr *string
	if ipAddress != nil {
		str := ipAddress.String()
		ipAddressStr = &str
	}

	// Handle empty user agent
	var userAgentPtr *string
	if userAgent != "" {
		userAgentPtr = &userAgent
	}

	_, _ = s.db.Exec(ctx, `
		INSERT INTO dashboard.activity_log (user_id, action, resource_type, resource_id, ip_address, user_agent, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, userID, action, resourceTypePtr, resourceIDPtr, ipAddressStr, userAgentPtr, metadata)
}

// generateBackupCode generates a random 8-character backup code
func generateBackupCode() (string, error) {
	bytes := make([]byte, 5)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes), nil
}
