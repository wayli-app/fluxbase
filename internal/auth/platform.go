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
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/nimbleflux/fluxbase/internal/database"
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
	LockedUntil   *time.Time `json:"locked_until,omitempty"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Role          string     `json:"role,omitempty"` // Set from JWT claims, not stored in DB
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
	db         *database.Connection
	jwtManager *JWTManager
	totpIssuer string // Default TOTP issuer for 2FA
}

// NewDashboardAuthService creates a new dashboard authentication service
func NewDashboardAuthService(db *database.Connection, jwtManager *JWTManager, totpIssuer string) *DashboardAuthService {
	return &DashboardAuthService{
		db:         db,
		jwtManager: jwtManager,
		totpIssuer: totpIssuer,
	}
}

// GetDB returns the database connection
func (s *DashboardAuthService) GetDB() *database.Connection {
	return s.db
}

// CreateUser creates a new dashboard user with email and password
func (s *DashboardAuthService) CreateUser(ctx context.Context, email, password, fullName string) (*DashboardUser, error) {
	// Validate email format and length
	if err := ValidateEmail(email); err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	// Validate full name
	if err := ValidateName(fullName); err != nil {
		return nil, fmt.Errorf("invalid name: %w", err)
	}

	// Validate password length (bcrypt has a 72 byte limit)
	if len(password) < MinPasswordLength {
		return nil, fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	if len(password) > MaxPasswordLength {
		return nil, fmt.Errorf("password must be at most %d characters", MaxPasswordLength)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), DefaultBcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &DashboardUser{}
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			INSERT INTO platform.users (email, password_hash, full_name, email_verified)
			VALUES ($1, $2, $3, false)
			RETURNING id, email, email_verified, full_name, avatar_url, role, totp_enabled,
			          is_active, is_locked, last_login_at, created_at, updated_at
		`, email, hashedPassword, fullName).Scan(
			&user.ID, &user.Email, &user.EmailVerified, &user.FullName, &user.AvatarURL,
			&user.Role, &user.TOTPEnabled, &user.IsActive, &user.IsLocked, &user.LastLoginAt,
			&user.CreatedAt, &user.UpdatedAt,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// HasExistingUsers checks if any dashboard users exist
func (s *DashboardAuthService) HasExistingUsers(ctx context.Context) (bool, error) {
	var count int
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT COUNT(*) FROM platform.users WHERE deleted_at IS NULL`).Scan(&count)
	})
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
	// Safe IP address string for logging
	var ipStr string
	if ipAddress != nil {
		ipStr = ipAddress.String()
	}

	// Fetch user with password hash
	var user DashboardUser
	var passwordHash string
	var failedAttempts int

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, email, email_verified, password_hash, full_name, avatar_url, role,
			       totp_enabled, is_active, is_locked, locked_until, failed_login_attempts,
			       last_login_at, created_at, updated_at
			FROM platform.users
			WHERE email = $1 AND deleted_at IS NULL
		`, email).Scan(
			&user.ID, &user.Email, &user.EmailVerified, &passwordHash, &user.FullName,
			&user.AvatarURL, &user.Role, &user.TOTPEnabled, &user.IsActive, &user.IsLocked,
			&user.LockedUntil, &failedAttempts, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Log failed login attempt for non-existent user
			LogSecurityEvent(ctx, SecurityEvent{
				Type:      SecurityEventLoginFailed,
				Email:     email,
				IPAddress: ipStr,
				UserAgent: userAgent,
				Details:   map[string]interface{}{"reason": "user_not_found", "dashboard": true},
			})
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Check if account is locked
	if user.IsLocked {
		// Check if the lock has expired and auto-unlock if so
		if user.LockedUntil != nil && time.Now().After(*user.LockedUntil) {
			// Lock has expired, auto-unlock the account
			err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `
					UPDATE platform.users
					SET is_locked = false, locked_until = NULL, failed_login_attempts = 0, updated_at = NOW()
					WHERE id = $1
				`, user.ID)
				return err
			})
			if err != nil {
				return nil, nil, fmt.Errorf("failed to auto-unlock account: %w", err)
			}
			// Update local state
			user.IsLocked = false
			user.LockedUntil = nil
			// Log the auto-unlock
			LogSecurityEvent(ctx, SecurityEvent{
				Type:      SecurityEventAccountUnlocked,
				UserID:    user.ID.String(),
				Email:     user.Email,
				IPAddress: ipStr,
				UserAgent: userAgent,
				Details:   map[string]interface{}{"reason": "lock_expired", "dashboard": true, "auto_unlock": true},
			})
		} else {
			// Account is still locked
			LogSecurityWarning(ctx, SecurityEvent{
				Type:      SecurityEventLoginFailed,
				UserID:    user.ID.String(),
				Email:     user.Email,
				IPAddress: ipStr,
				UserAgent: userAgent,
				Details:   map[string]interface{}{"reason": "account_locked", "dashboard": true},
			})
			return nil, nil, ErrAccountLocked
		}
	}

	// Check if account is active
	if !user.IsActive {
		LogSecurityWarning(ctx, SecurityEvent{
			Type:      SecurityEventLoginFailed,
			UserID:    user.ID.String(),
			Email:     user.Email,
			IPAddress: ipStr,
			UserAgent: userAgent,
			Details:   map[string]interface{}{"reason": "account_inactive", "dashboard": true},
		})
		return nil, nil, errors.New("account is inactive") // No standard error for this case
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		// Increment failed login attempts and lock if threshold exceeded
		_ = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				UPDATE platform.users
				SET failed_login_attempts = failed_login_attempts + 1,
				    is_locked = CASE WHEN failed_login_attempts >= 4 THEN true ELSE false END,
				    locked_until = CASE WHEN failed_login_attempts >= 4 THEN NOW() + INTERVAL '15 minutes' ELSE locked_until END
				WHERE id = $1
			`, user.ID)
			return err
		})
		// Log failed login due to wrong password
		LogSecurityEvent(ctx, SecurityEvent{
			Type:      SecurityEventLoginFailed,
			UserID:    user.ID.String(),
			Email:     user.Email,
			IPAddress: ipStr,
			UserAgent: userAgent,
			Details:   map[string]interface{}{"reason": "invalid_password", "dashboard": true},
		})
		return nil, nil, ErrInvalidCredentials
	}

	// Reset failed attempts on successful login
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE platform.users
			SET failed_login_attempts = 0,
			    locked_until = NULL,
			    last_login_at = NOW()
			WHERE id = $1
		`, user.ID)
		return err
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update login timestamp: %w", err)
	}

	// Log successful login
	LogSecurityEvent(ctx, SecurityEvent{
		Type:      SecurityEventLoginSuccess,
		UserID:    user.ID.String(),
		Email:     user.Email,
		IPAddress: ipStr,
		UserAgent: userAgent,
		Details:   map[string]interface{}{"dashboard": true},
	})

	// Prepare user metadata for JWT
	userMetadata := map[string]interface{}{}
	if user.FullName != nil {
		userMetadata["name"] = *user.FullName
	}
	if user.AvatarURL != nil {
		userMetadata["avatar"] = *user.AvatarURL
	}

	// Use the actual role from the database, defaulting to dashboard_user if empty
	userRole := user.Role
	if userRole == "" {
		userRole = "dashboard_user"
	}

	// Determine tenant context for JWT claims
	tenantOpts := TenantTokenOptions{
		IsInstanceAdmin: userRole == "instance_admin",
	}

	membership := s.resolveTenantMembership(ctx, user.ID)
	if membership.tenantID != nil {
		tenantOpts.TenantID = membership.tenantID
		tenantOpts.TenantRole = membership.tenantRole
	}

	// Generate JWT token pair with tenant context (access + refresh)
	accessToken, refreshToken, sessionID, err := s.jwtManager.GenerateTokenPairWithTenant(user.ID.String(), user.Email, userRole, userMetadata, nil, tenantOpts)
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
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			DELETE FROM platform.sessions WHERE user_id = $1
		`, user.ID)
		return err
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to clean up old sessions: %w", err)
	}

	// Create new session record with session ID from token
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO platform.sessions (id, user_id, token, ip_address, user_agent, expires_at)
			VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '24 hours')
		`, sessionID, user.ID, tokenHash, ipAddressStr, userAgent)
		return err
	})
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

// GetUserByID fetches a dashboard user by ID
func (s *DashboardAuthService) GetUserByID(ctx context.Context, userID uuid.UUID) (*DashboardUser, error) {
	user := &DashboardUser{}
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, email, email_verified, full_name, avatar_url, role, totp_enabled,
			       is_active, is_locked, last_login_at, created_at, updated_at
			FROM platform.users
			WHERE id = $1 AND deleted_at IS NULL
		`, userID).Scan(
			&user.ID, &user.Email, &user.EmailVerified, &user.FullName, &user.AvatarURL,
			&user.Role, &user.TOTPEnabled, &user.IsActive, &user.IsLocked, &user.LastLoginAt,
			&user.CreatedAt, &user.UpdatedAt,
		)
	})
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

	_ = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO platform.activity_log (user_id, action, resource_type, resource_id, ip_address, user_agent, details)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, userID, action, resourceTypePtr, resourceIDPtr, ipAddressStr, userAgentPtr, metadata)
		return err
	})
}

// generateBackupCode generates a random 8-character backup code
func generateBackupCode() (string, error) {
	bytes := make([]byte, 5)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes), nil
}

type tenantMembership struct {
	tenantID   *string
	tenantRole string
}

func (s *DashboardAuthService) resolveTenantMembership(ctx context.Context, userID uuid.UUID) tenantMembership {
	var membership tenantMembership
	var tenantID string
	var tenantRole string

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			SELECT t.id::text
			FROM platform.tenants t
			WHERE t.is_default = true AND t.deleted_at IS NULL
			LIMIT 1
		`).Scan(&tenantID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return tx.QueryRow(ctx, `
					SELECT taa.tenant_id::text
					FROM platform.tenant_admin_assignments taa
					INNER JOIN platform.tenants t ON t.id = taa.tenant_id
					WHERE taa.user_id = $1 AND t.deleted_at IS NULL
					ORDER BY t.is_default DESC, t.created_at ASC
					LIMIT 1
				`, userID).Scan(&tenantID)
			}
			return err
		}

		var isAssigned bool
		err = tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM platform.tenant_admin_assignments
				WHERE tenant_id = $1 AND user_id = $2
			)
		`, tenantID, userID).Scan(&isAssigned)
		if err != nil {
			return err
		}

		if isAssigned {
			tenantRole = "tenant_admin"
		} else {
			err = tx.QueryRow(ctx, `
				SELECT taa.tenant_id::text
				FROM platform.tenant_admin_assignments taa
				INNER JOIN platform.tenants t ON t.id = taa.tenant_id
				WHERE taa.user_id = $1 AND t.deleted_at IS NULL
				ORDER BY t.is_default DESC, t.created_at ASC
				LIMIT 1
			`, userID).Scan(&tenantID)
			if err == nil {
				tenantRole = "tenant_admin"
			}
		}
		return nil
	})
	if err != nil {
		log.Debug().Err(err).Str("user_id", userID.String()).Msg("Failed to get tenant membership, using default")
	}

	if tenantID != "" {
		membership.tenantID = &tenantID
		membership.tenantRole = tenantRole
	}
	return membership
}

// RefreshToken generates a new access token using a refresh token for dashboard users
func (s *DashboardAuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// Validate refresh token
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("invalid token type")
	}

	// Verify the token is for a dashboard user (any dashboard role is valid)
	validDashboardRoles := map[string]bool{
		"instance_admin": true,
		"tenant_admin":   true,
	}
	if !validDashboardRoles[claims.Role] {
		return nil, fmt.Errorf("invalid token: not a dashboard user token")
	}

	// Generate new access token
	newAccessToken, err := s.jwtManager.RefreshAccessToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken, // Refresh token stays the same
		ExpiresIn:    int64(24 * 60 * 60),
	}, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *DashboardAuthService) ValidateToken(token string) (*TokenClaims, error) {
	return s.jwtManager.ValidateToken(token)
}
