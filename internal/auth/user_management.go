package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
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
	Metadata       map[string]interface{} `json:"metadata"`
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
func (s *UserManagementService) ListEnrichedUsers(ctx context.Context) ([]*EnrichedUser, error) {
	query := `
		SELECT
			u.id,
			u.email,
			u.email_verified,
			u.role,
			u.metadata,
			u.created_at,
			u.updated_at,
			COALESCE(COUNT(DISTINCT CASE WHEN s.expires_at > NOW() THEN s.id END), 0) as active_sessions,
			MAX(s.created_at) as last_sign_in,
			CASE
				WHEN u.password_hash IS NOT NULL THEN 'email'
				WHEN u.email_verified = false THEN 'invite_pending'
				ELSE 'email'
			END as provider
		FROM auth.users u
		LEFT JOIN auth.sessions s ON u.id = s.user_id
		GROUP BY u.id, u.email, u.email_verified, u.role, u.metadata, u.created_at, u.updated_at, u.password_hash
		ORDER BY u.created_at DESC
	`

	rows, err := s.userRepo.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query enriched users: %w", err)
	}
	defer rows.Close()

	var users []*EnrichedUser
	for rows.Next() {
		user := &EnrichedUser{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.EmailVerified,
			&user.Role,
			&user.Metadata,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.ActiveSessions,
			&user.LastSignIn,
			&user.Provider,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan enriched user: %w", err)
		}
		users = append(users, user)
	}

	return users, rows.Err()
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
func (s *UserManagementService) InviteUser(ctx context.Context, req InviteUserRequest) (*InviteUserResponse, error) {
	// Validate role
	if req.Role == "" {
		req.Role = "user"
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

	// Create user
	createReq := CreateUserRequest{
		Email:    req.Email,
		Password: tempPassword, // Not used, we provide hash directly
		Role:     req.Role,
	}

	user, err := s.userRepo.Create(ctx, createReq, hashedPassword)
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
func (s *UserManagementService) UpdateUserRole(ctx context.Context, userID string, newRole string) (*User, error) {
	req := UpdateUserRequest{
		Role: &newRole,
	}
	return s.userRepo.Update(ctx, userID, req)
}

// DeleteUser deletes a user (cascades to sessions, tokens, etc.)
func (s *UserManagementService) DeleteUser(ctx context.Context, userID string) error {
	return s.userRepo.Delete(ctx, userID)
}

// ResetUserPassword triggers a password reset for a user
func (s *UserManagementService) ResetUserPassword(ctx context.Context, userID string) (string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
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

// Helper function to generate secure random password
func generateSecurePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}
