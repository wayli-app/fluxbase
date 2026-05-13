package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// SSOIdentity represents a linked SSO identity for a dashboard user
type SSOIdentity struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	Provider       string    `json:"provider"`
	ProviderUserID string    `json:"provider_user_id"`
	Email          *string   `json:"email,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// GetUserBySSOIdentity finds a dashboard user by their SSO identity
func (s *DashboardAuthService) GetUserBySSOIdentity(ctx context.Context, provider, providerUserID string) (*DashboardUser, error) {
	// Split provider into type and name (format: "oauth:authelia" or "saml:okta")
	parts := strings.SplitN(provider, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid provider format (expected 'type:name'): %s", provider)
	}
	providerType := parts[0]
	providerName := parts[1]

	user := &DashboardUser{}
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT u.id, u.email, u.email_verified, u.full_name, u.avatar_url, u.role, u.totp_enabled,
			       u.is_active, u.is_locked, u.last_login_at, u.created_at, u.updated_at
			FROM platform.users u
			INNER JOIN platform.sso_identities si ON si.user_id = u.id
			WHERE si.provider_type = $1 AND si.provider_name = $2 AND si.provider_user_id = $3 AND u.deleted_at IS NULL
		`, providerType, providerName, providerUserID).Scan(
			&user.ID, &user.Email, &user.EmailVerified, &user.FullName, &user.AvatarURL,
			&user.Role, &user.TOTPEnabled, &user.IsActive, &user.IsLocked, &user.LastLoginAt,
			&user.CreatedAt, &user.UpdatedAt,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No user found with this SSO identity
		}
		return nil, fmt.Errorf("failed to fetch user by SSO identity: %w", err)
	}
	return user, nil
}

// GetUserByEmail finds a dashboard user by email
func (s *DashboardAuthService) GetUserByEmail(ctx context.Context, email string) (*DashboardUser, error) {
	user := &DashboardUser{}
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, email, email_verified, full_name, avatar_url, role, totp_enabled,
			       is_active, is_locked, last_login_at, created_at, updated_at
			FROM platform.users
			WHERE email = $1 AND deleted_at IS NULL
		`, email).Scan(
			&user.ID, &user.Email, &user.EmailVerified, &user.FullName, &user.AvatarURL,
			&user.Role, &user.TOTPEnabled, &user.IsActive, &user.IsLocked, &user.LastLoginAt,
			&user.CreatedAt, &user.UpdatedAt,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No user found
		}
		return nil, fmt.Errorf("failed to fetch user by email: %w", err)
	}
	return user, nil
}

// FindOrCreateUserBySSO finds an existing user by SSO identity or email, or creates a new one
// Returns the user and a boolean indicating if a new user was created
func (s *DashboardAuthService) FindOrCreateUserBySSO(ctx context.Context, email, name, provider, providerUserID string) (*DashboardUser, bool, error) {
	// First, try to find by SSO identity
	user, err := s.GetUserBySSOIdentity(ctx, provider, providerUserID)
	if err != nil {
		return nil, false, err
	}
	if user != nil {
		// Update last login
		_ = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				UPDATE platform.users SET last_login_at = NOW() WHERE id = $1
			`, user.ID)
			return err
		})
		return user, false, nil
	}

	// Try to find by email
	user, err = s.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, false, err
	}

	if user != nil {
		// Link SSO identity to existing user
		err = s.LinkSSOIdentity(ctx, user.ID, provider, providerUserID, email)
		if err != nil {
			return nil, false, err
		}
		// Update last login
		_ = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				UPDATE platform.users SET last_login_at = NOW() WHERE id = $1
			`, user.ID)
			return err
		})
		return user, false, nil
	}

	// Create new user (JIT provisioning)
	// Split provider into type and name (format: "oauth:authelia" or "saml:okta")
	parts := strings.SplitN(provider, ":", 2)
	if len(parts) != 2 {
		return nil, false, fmt.Errorf("invalid provider format (expected 'type:name'): %s", provider)
	}
	providerType := parts[0]
	providerName := parts[1]

	user = &DashboardUser{}
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		// Create user without password (SSO-only user)
		err := tx.QueryRow(ctx, `
			INSERT INTO platform.users (email, full_name, email_verified, is_active)
			VALUES ($1, $2, true, true)
			RETURNING id, email, email_verified, full_name, avatar_url, role, totp_enabled,
			          is_active, is_locked, last_login_at, created_at, updated_at
		`, email, name).Scan(
			&user.ID, &user.Email, &user.EmailVerified, &user.FullName, &user.AvatarURL,
			&user.Role, &user.TOTPEnabled, &user.IsActive, &user.IsLocked, &user.LastLoginAt,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return err
		}

		// Link SSO identity
		_, err = tx.Exec(ctx, `
			INSERT INTO platform.sso_identities (user_id, provider_type, provider_name, provider_user_id, email)
			VALUES ($1, $2, $3, $4, $5)
		`, user.ID, providerType, providerName, providerUserID, email)
		return err
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to create user via SSO: %w", err)
	}

	return user, true, nil
}

// LinkSSOIdentity links an SSO identity to an existing dashboard user
func (s *DashboardAuthService) LinkSSOIdentity(ctx context.Context, userID uuid.UUID, provider, providerUserID, email string) error {
	// Split provider into type and name (format: "oauth:authelia" or "saml:okta")
	parts := strings.SplitN(provider, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid provider format (expected 'type:name'): %s", provider)
	}
	providerType := parts[0]
	providerName := parts[1]

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO platform.sso_identities (user_id, provider_type, provider_name, provider_user_id, email)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (provider_type, provider_name, provider_user_id) DO UPDATE SET email = EXCLUDED.email
		`, userID, providerType, providerName, providerUserID, email)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to link SSO identity: %w", err)
	}
	return nil
}

// LoginViaSSO logs in a dashboard user via SSO and returns tokens
func (s *DashboardAuthService) LoginViaSSO(ctx context.Context, user *DashboardUser, ipAddress net.IP, userAgent string) (*LoginResponse, error) {
	// Safe IP address string for logging
	var ipStr string
	if ipAddress != nil {
		ipStr = ipAddress.String()
	}

	// Check if account is locked
	if user.IsLocked {
		LogSecurityWarning(ctx, SecurityEvent{
			Type:      SecurityEventLoginFailed,
			UserID:    user.ID.String(),
			Email:     user.Email,
			IPAddress: ipStr,
			UserAgent: userAgent,
			Details:   map[string]interface{}{"reason": "account_locked", "dashboard": true, "sso": true},
		})
		return nil, ErrAccountLocked
	}

	// Check if account is active
	if !user.IsActive {
		LogSecurityWarning(ctx, SecurityEvent{
			Type:      SecurityEventLoginFailed,
			UserID:    user.ID.String(),
			Email:     user.Email,
			IPAddress: ipStr,
			UserAgent: userAgent,
			Details:   map[string]interface{}{"reason": "account_inactive", "dashboard": true, "sso": true},
		})
		return nil, errors.New("account is inactive")
	}

	// Log successful SSO login
	LogSecurityEvent(ctx, SecurityEvent{
		Type:      SecurityEventLoginSuccess,
		UserID:    user.ID.String(),
		Email:     user.Email,
		IPAddress: ipStr,
		UserAgent: userAgent,
		Details:   map[string]interface{}{"dashboard": true, "sso": true},
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

	// Generate JWT token pair with tenant context
	accessToken, refreshToken, sessionID, err := s.jwtManager.GenerateTokenPairWithTenant(user.ID.String(), user.Email, userRole, userMetadata, nil, tenantOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Hash the access token
	hash := sha256.Sum256([]byte(accessToken))
	tokenHash := hex.EncodeToString(hash[:])

	// Handle nil IP address
	var ipAddressStr interface{}
	if ipAddress != nil {
		ipAddressStr = ipAddress.String()
	}

	// Delete existing sessions and create new one
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `DELETE FROM platform.sessions WHERE user_id = $1`, user.ID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO platform.sessions (id, user_id, token, ip_address, user_agent, expires_at)
			VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '24 hours')
		`, sessionID, user.ID, tokenHash, ipAddressStr, userAgent)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Log activity
	s.logActivity(ctx, user.ID, "sso_login", "", "", ipAddress, userAgent, nil)

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(24 * 60 * 60),
	}, nil
}
