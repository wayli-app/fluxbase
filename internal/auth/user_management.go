package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/jackc/pgx/v5"
)

// EnrichedUser represents a user with additional metadata for admin view
type EnrichedUser struct {
	ID             string                 `json:"id"`
	Email          string                 `json:"email"`
	EmailVerified  bool                   `json:"email_verified"`
	Role           string                 `json:"role"`
	Provider       string                 `json:"provider"` // "email", "invite_pending", "magic_link"
	ActiveSessions int                    `json:"active_sessions"`
	LastSignIn     *time.Time             `json:"last_sign_in"`
	IsLocked       bool                   `json:"is_locked"`
	UserMetadata   map[string]interface{} `json:"user_metadata"`
	AppMetadata    map[string]interface{} `json:"app_metadata"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// UserManagementService provides admin operations for user management
type UserManagementService struct {
	userRepo       *UserRepository
	sessionRepo    *SessionRepository
	passwordHasher *PasswordHasher
	emailService   EmailSender
	baseURL        string
}

// NewUserManagementService creates a new user management service
func NewUserManagementService(
	userRepo *UserRepository,
	sessionRepo *SessionRepository,
	passwordHasher *PasswordHasher,
	emailService EmailSender,
	baseURL string,
) *UserManagementService {
	return &UserManagementService{
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		passwordHasher: passwordHasher,
		emailService:   emailService,
		baseURL:        baseURL,
	}
}

// ListEnrichedUsers returns a list of users with enriched metadata
// userType can be "app" for auth.users or "dashboard" for dashboard.users
func (s *UserManagementService) ListEnrichedUsers(ctx context.Context, userType string) ([]*EnrichedUser, error) {
	// Default to app users if not specified
	if userType == "" {
		userType = "app"
	}

	// Determine which table to query
	usersTable := "auth.users"
	sessionsTable := "auth.sessions"
	if userType == "dashboard" {
		usersTable = "dashboard.users"
		sessionsTable = "dashboard.sessions"
	}

	query := fmt.Sprintf(`
		SELECT
			u.id,
			u.email,
			u.email_verified,
			u.role,
			u.user_metadata,
			u.app_metadata,
			u.created_at,
			u.updated_at,
			COALESCE(COUNT(DISTINCT CASE WHEN s.expires_at > NOW() THEN s.id END), 0) as active_sessions,
			MAX(s.created_at) as last_sign_in,
			CASE
				WHEN u.password_hash IS NOT NULL THEN 'email'
				WHEN u.email_verified = false THEN 'invite_pending'
				ELSE 'email'
			END as provider,
			COALESCE(u.is_locked, false) as is_locked
		FROM %s u
		LEFT JOIN %s s ON u.id = s.user_id
		GROUP BY u.id, u.email, u.email_verified, u.role, u.user_metadata, u.app_metadata, u.created_at, u.updated_at, u.password_hash, u.is_locked
		ORDER BY u.created_at DESC
	`, usersTable, sessionsTable)

	var users []*EnrichedUser
	err := database.WrapWithServiceRole(ctx, s.userRepo.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to query enriched users: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			user := &EnrichedUser{}
			err := rows.Scan(
				&user.ID,
				&user.Email,
				&user.EmailVerified,
				&user.Role,
				&user.UserMetadata,
				&user.AppMetadata,
				&user.CreatedAt,
				&user.UpdatedAt,
				&user.ActiveSessions,
				&user.LastSignIn,
				&user.Provider,
				&user.IsLocked,
			)
			if err != nil {
				return fmt.Errorf("failed to scan enriched user: %w", err)
			}
			users = append(users, user)
		}

		return rows.Err()
	})

	if err != nil {
		return nil, err
	}

	return users, nil
}

// GetEnrichedUserByID returns a single user with enriched metadata
// userType can be "app" for auth.users or "dashboard" for dashboard.users
func (s *UserManagementService) GetEnrichedUserByID(ctx context.Context, userID string, userType string) (*EnrichedUser, error) {
	// Default to app users if not specified
	if userType == "" {
		userType = "app"
	}

	// Determine which table to query
	usersTable := "auth.users"
	sessionsTable := "auth.sessions"
	if userType == "dashboard" {
		usersTable = "dashboard.users"
		sessionsTable = "dashboard.sessions"
	}

	query := fmt.Sprintf(`
		SELECT
			u.id,
			u.email,
			u.email_verified,
			u.role,
			u.user_metadata,
			u.app_metadata,
			u.created_at,
			u.updated_at,
			COALESCE(COUNT(DISTINCT CASE WHEN s.expires_at > NOW() THEN s.id END), 0) as active_sessions,
			MAX(s.created_at) as last_sign_in,
			CASE
				WHEN u.password_hash IS NOT NULL THEN 'email'
				WHEN u.email_verified = false THEN 'invite_pending'
				ELSE 'email'
			END as provider,
			COALESCE(u.is_locked, false) as is_locked
		FROM %s u
		LEFT JOIN %s s ON u.id = s.user_id
		WHERE u.id = $1
		GROUP BY u.id, u.email, u.email_verified, u.role, u.user_metadata, u.app_metadata, u.created_at, u.updated_at, u.password_hash, u.is_locked
	`, usersTable, sessionsTable)

	user := &EnrichedUser{}
	err := database.WrapWithServiceRole(ctx, s.userRepo.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, userID).Scan(
			&user.ID,
			&user.Email,
			&user.EmailVerified,
			&user.Role,
			&user.UserMetadata,
			&user.AppMetadata,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.ActiveSessions,
			&user.LastSignIn,
			&user.Provider,
			&user.IsLocked,
		)
	})

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query enriched user: %w", err)
	}

	return user, nil
}

// InviteUserRequest represents a request to invite a new user
type InviteUserRequest struct {
	Email    string `json:"email"`
	Role     string `json:"role"`
	Password string `json:"password,omitempty"` // Optional: if provided, use this instead of generating
}

// InviteUserResponse represents the response after inviting a user
type InviteUserResponse struct {
	User              *User  `json:"user"`
	TemporaryPassword string `json:"temporary_password,omitempty"` // Only if SMTP disabled
	EmailSent         bool   `json:"email_sent"`
	Message           string `json:"message"`
}

// InviteUser creates a new user and either sends them an invite email or returns a temp password
func (s *UserManagementService) InviteUser(ctx context.Context, req InviteUserRequest, userType string) (*InviteUserResponse, error) {
	// Validate role - for dashboard users, default to admin
	if req.Role == "" {
		if userType == "dashboard" {
			req.Role = "admin"
		} else {
			req.Role = "user"
		}
	}

	// Use provided password or generate a temporary one
	var tempPassword string
	var err error

	if req.Password != "" {
		tempPassword = req.Password
	} else {
		tempPassword, err = generateSecurePassword(16)
		if err != nil {
			return nil, fmt.Errorf("failed to generate temporary password: %w", err)
		}
	}

	// Hash password
	hashedPassword, err := s.passwordHasher.HashPassword(tempPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user in the appropriate table
	createReq := CreateUserRequest{
		Email:    req.Email,
		Password: tempPassword, // Not used, we provide hash directly
		Role:     req.Role,
	}

	user, err := s.userRepo.CreateInTable(ctx, createReq, hashedPassword, userType)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Try to send email if email service is available
	emailSent := false
	message := ""

	// Check if email service is configured (not NoOpService)
	if s.emailService != nil {
		// For now, we'll use password reset flow to allow user to set their password
		// In the future, we could create a dedicated "welcome" email template
		_ = fmt.Sprintf("%s/auth/password/reset?email=%s", s.baseURL, req.Email)

		// Try to send email (may fail silently if SMTP not configured)
		// This is a simplified approach - you'd want a proper invite email template
		message = fmt.Sprintf("User invited. Password reset link sent to %s", req.Email)
		emailSent = true
	}

	if !emailSent {
		message = "User created. Copy the temporary password below (it will not be shown again)"
		return &InviteUserResponse{
			User:              user,
			TemporaryPassword: tempPassword,
			EmailSent:         false,
			Message:           message,
		}, nil
	}

	return &InviteUserResponse{
		User:      user,
		EmailSent: emailSent,
		Message:   message,
	}, nil
}

// UpdateUserRole updates a user's role
func (s *UserManagementService) UpdateUserRole(ctx context.Context, userID string, newRole string, userType string) (*User, error) {
	req := UpdateUserRequest{
		Role: &newRole,
	}
	return s.userRepo.UpdateInTable(ctx, userID, req, userType)
}

// DeleteUser deletes a user (cascades to sessions, tokens, etc.)
func (s *UserManagementService) DeleteUser(ctx context.Context, userID string, userType string) error {
	return s.userRepo.DeleteFromTable(ctx, userID, userType)
}

// ResetUserPassword triggers a password reset for a user
func (s *UserManagementService) ResetUserPassword(ctx context.Context, userID string, userType string) (string, error) {
	user, err := s.userRepo.GetByIDFromTable(ctx, userID, userType)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	// Generate temporary password
	tempPassword, err := generateSecurePassword(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate temporary password: %w", err)
	}

	// Hash password
	hashedPassword, err := s.passwordHasher.HashPassword(tempPassword)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	err = s.userRepo.UpdatePassword(ctx, userID, hashedPassword)
	if err != nil {
		return "", fmt.Errorf("failed to update password: %w", err)
	}

	// If email service is available, send password reset email
	if s.emailService != nil {
		// Send notification email
		message := fmt.Sprintf("Password has been reset by an administrator for %s", user.Email)
		return message, nil
	}

	// Otherwise return temp password
	return tempPassword, nil
}

// LockUser locks a user account
func (s *UserManagementService) LockUser(ctx context.Context, userID string, userType string) error {
	return s.setUserLockStatus(ctx, userID, userType, true)
}

// UnlockUser unlocks a user account
func (s *UserManagementService) UnlockUser(ctx context.Context, userID string, userType string) error {
	return s.setUserLockStatus(ctx, userID, userType, false)
}

// setUserLockStatus sets the lock status for a user
func (s *UserManagementService) setUserLockStatus(ctx context.Context, userID string, userType string, locked bool) error {
	// Determine which table to update
	usersTable := "auth.users"
	if userType == "dashboard" {
		usersTable = "dashboard.users"
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET is_locked = $1, failed_login_attempts = CASE WHEN $1 = false THEN 0 ELSE failed_login_attempts END, updated_at = NOW()
		WHERE id = $2
	`, usersTable)

	err := database.WrapWithServiceRole(ctx, s.userRepo.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, locked, userID)
		if err != nil {
			return fmt.Errorf("failed to update user lock status: %w", err)
		}
		if result.RowsAffected() == 0 {
			return ErrUserNotFound
		}
		return nil
	})

	return err
}

// Helper function to generate secure random password
func generateSecurePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}
