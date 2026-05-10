package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// EnrichedUser represents a user with additional metadata for admin view
type EnrichedUser struct {
	ID                string                 `json:"id"`
	Email             string                 `json:"email"`
	EmailVerified     bool                   `json:"email_verified"`
	Role              string                 `json:"role"`
	Provider          string                 `json:"provider"` // "email", "invite_pending", "magic_link"
	ActiveSessions    int                    `json:"active_sessions"`
	LastSignIn        *time.Time             `json:"last_sign_in"`
	IsLocked          bool                   `json:"is_locked"`
	UserMetadata      map[string]interface{} `json:"user_metadata"`
	AppMetadata       map[string]interface{} `json:"app_metadata"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	TenantAssignments []TenantAssignment     `json:"tenant_assignments,omitempty"`
}

// TenantAssignment represents a user's assignment to a tenant
type TenantAssignment struct {
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	TenantSlug string `json:"tenant_slug"`
}

// UserManagementService provides admin operations for user management
type UserManagementService struct {
	userRepo       *UserRepository
	sessionRepo    *SessionRepository
	passwordHasher *PasswordHasher
	emailService   EmailService
	baseURL        string
}

// NewUserManagementService creates a new user management service
func NewUserManagementService(
	userRepo *UserRepository,
	sessionRepo *SessionRepository,
	passwordHasher *PasswordHasher,
	emailService EmailService,
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
// userType can be "app" for auth.users or "platform" for platform.users
// tenantID is optional and filters app users by tenant membership
func (s *UserManagementService) ListEnrichedUsers(ctx context.Context, userType string, tenantID string) ([]*EnrichedUser, error) {
	// Default to app users if not specified
	if userType == "" {
		userType = "app"
	}

	// Determine which table to query
	usersTable := "auth.users"
	sessionsTable := "auth.sessions"
	tenantAssignmentsSelect := ""
	tenantAssignmentsJoin := ""

	if userType == "platform" {
		usersTable = "platform.users"
		sessionsTable = "platform.sessions"
		// Join to get tenant assignments for platform users
		tenantAssignmentsJoin = `
			LEFT JOIN LATERAL (
				SELECT COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'tenant_id', t.id,
							'tenant_name', t.name,
							'tenant_slug', t.slug
						)
					),
					'[]'::jsonb
				) as assignments
				FROM platform.tenant_admin_assignments taa
				JOIN platform.tenants t ON t.id = taa.tenant_id
				WHERE taa.user_id = u.id
			) ta ON true`
		tenantAssignmentsSelect = ", ta.assignments as tenant_assignments"
	} else {
		// App users: join tenant memberships to populate tenant_assignments
		tenantAssignmentsJoin = `
			LEFT JOIN LATERAL (
				SELECT COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'tenant_id', t.id,
							'tenant_name', t.name,
							'tenant_slug', t.slug
						)
					),
					'[]'::jsonb
				) as assignments
				FROM platform.tenant_memberships tm
				JOIN platform.tenants t ON t.id = tm.tenant_id
				WHERE tm.user_id = u.id
			) ta ON true`
		tenantAssignmentsSelect = ", ta.assignments as tenant_assignments"
	}

	// Build GROUP BY clause - include tenant assignments for all user types
	groupByClause := "u.id, u.email, u.email_verified, u.role, u.user_metadata, u.app_metadata, u.created_at, u.updated_at, u.password_hash, u.is_locked, ta.assignments"

	// Build WHERE clause for tenant filtering
	var whereClause string
	var args []interface{}
	if userType == "app" && tenantID != "" {
		whereClause = `
			WHERE u.id IN (
				SELECT tm.user_id FROM platform.tenant_memberships tm
				WHERE tm.tenant_id = $1
			)`
		args = append(args, tenantID)
	} else if userType == "platform" && tenantID != "" {
		whereClause = `
			WHERE u.id IN (
				SELECT taa.user_id FROM platform.tenant_admin_assignments taa
				WHERE taa.tenant_id = $1
			)`
		args = append(args, tenantID)
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
			%s
		FROM %s u
		LEFT JOIN %s s ON u.id = s.user_id
		%s
		%s
		GROUP BY %s
		ORDER BY u.created_at DESC
	`, tenantAssignmentsSelect, usersTable, sessionsTable, tenantAssignmentsJoin, whereClause, groupByClause)

	var users []*EnrichedUser
	err := database.WrapWithServiceRole(ctx, s.userRepo.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to query enriched users: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			user := &EnrichedUser{}
			var tenantAssignmentsJSON []byte

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
				&tenantAssignmentsJSON,
			)
			if err != nil {
				return fmt.Errorf("failed to scan enriched user: %w", err)
			}
			// Parse tenant assignments from JSON
			if len(tenantAssignmentsJSON) > 0 && string(tenantAssignmentsJSON) != "null" {
				if err := json.Unmarshal(tenantAssignmentsJSON, &user.TenantAssignments); err != nil {
					user.TenantAssignments = nil
				}
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
// userType can be "app" for auth.users or "platform" for platform.users
func (s *UserManagementService) GetEnrichedUserByID(ctx context.Context, userID string, userType string) (*EnrichedUser, error) {
	// Default to app users if not specified
	if userType == "" {
		userType = "app"
	}

	// Determine which table to query
	usersTable := "auth.users"
	sessionsTable := "auth.sessions"
	if userType == "platform" {
		usersTable = "platform.users"
		sessionsTable = "platform.sessions"
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
	Email     string `json:"email"`
	Role      string `json:"role"`
	Password  string `json:"password,omitempty"`   // Optional: if provided, use this instead of generating
	SkipEmail bool   `json:"skip_email,omitempty"` // Optional: if true, don't send invitation email
	TenantID  string `json:"tenant_id,omitempty"`  // Optional: tenant to add the user to (for app users)
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
	// Validate role - for platform/dashboard users, default to instance_admin
	if req.Role == "" {
		if userType == "platform" {
			req.Role = "instance_admin"
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

	// Add user to tenant if tenant_id is provided (for app users only)
	if userType == "app" && req.TenantID != "" && s.userRepo.db != nil {
		_, err := s.userRepo.db.Exec(
			ctx,
			`INSERT INTO platform.tenant_memberships (tenant_id, user_id, role)
			 VALUES ($1::uuid, $2::uuid, 'tenant_member')
			 ON CONFLICT (tenant_id, user_id) DO NOTHING`,
			req.TenantID, user.ID,
		)
		if err != nil {
			// Log error but don't fail - user was created successfully
			// The admin can manually add them to the tenant later
		}
	}

	// Try to send invitation email if email service is available and not skipped
	emailSent := false
	message := ""

	if req.SkipEmail {
		message = "User created. Copy the temporary password below (it will not be shown again)"
		return &InviteUserResponse{
			User:              user,
			TemporaryPassword: tempPassword,
			EmailSent:         false,
			Message:           message,
		}, nil
	}

	if s.emailService != nil {
		inviteLink := fmt.Sprintf("%s/sign-in", s.baseURL)
		err := s.emailService.SendInvitationEmail(ctx, req.Email, "An administrator", inviteLink)
		if err != nil {
			// Log error but don't fail - user was created successfully
			message = "User created. Failed to send invitation email - share the temporary password manually."
		} else {
			emailSent = true
			message = fmt.Sprintf("Invitation email sent to %s", req.Email)
		}
	}

	if !emailSent {
		if message == "" {
			message = "User created. Copy the temporary password below (it will not be shown again)"
		}
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
	if userType == "platform" {
		usersTable = "platform.users"
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

// UpdateUserRequest for admin user updates
type UpdateAdminUserRequest struct {
	Email        *string                `json:"email,omitempty"`
	Role         *string                `json:"role,omitempty"`
	Password     *string                `json:"password,omitempty"`
	UserMetadata map[string]interface{} `json:"user_metadata,omitempty"`
}

// UpdateUser updates a user's information
func (s *UserManagementService) UpdateUser(ctx context.Context, userID string, req UpdateAdminUserRequest, userType string) (*EnrichedUser, error) {
	// Build update request
	updateReq := UpdateUserRequest{}

	if req.Email != nil {
		updateReq.Email = req.Email
	}
	if req.Role != nil {
		updateReq.Role = req.Role
	}
	if req.UserMetadata != nil {
		updateReq.UserMetadata = req.UserMetadata
	}

	// Update user in the appropriate table
	_, err := s.userRepo.UpdateInTable(ctx, userID, updateReq, userType)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// If password is provided, update it
	if req.Password != nil && *req.Password != "" {
		hashedPassword, err := s.passwordHasher.HashPassword(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		err = s.userRepo.UpdatePasswordInTable(ctx, userID, hashedPassword, userType)
		if err != nil {
			return nil, fmt.Errorf("failed to update password: %w", err)
		}
	}

	// Return the updated user
	return s.GetEnrichedUserByID(ctx, userID, userType)
}

// Helper function to generate secure random password
func generateSecurePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}
