package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wayli-app/fluxbase/internal/database"
)

var (
	// ErrNotAdmin is returned when a non-admin tries to impersonate
	ErrNotAdmin = errors.New("only admins can impersonate users")
	// ErrSelfImpersonation is returned when trying to impersonate yourself
	ErrSelfImpersonation = errors.New("cannot impersonate yourself")
	// ErrNoActiveImpersonation is returned when trying to stop non-existent impersonation
	ErrNoActiveImpersonation = errors.New("no active impersonation session found")
)

// ImpersonationSession represents an admin impersonation session
type ImpersonationSession struct {
	ID           string     `json:"id" db:"id"`
	AdminUserID  string     `json:"admin_user_id" db:"admin_user_id"`
	TargetUserID string     `json:"target_user_id" db:"target_user_id"`
	Reason       string     `json:"reason,omitempty" db:"reason"`
	StartedAt    time.Time  `json:"started_at" db:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty" db:"ended_at"`
	IPAddress    string     `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent    string     `json:"user_agent,omitempty" db:"user_agent"`
	IsActive     bool       `json:"is_active" db:"is_active"`
}

// ImpersonationRepository handles database operations for impersonation sessions
type ImpersonationRepository struct {
	db *database.Connection
}

// NewImpersonationRepository creates a new impersonation repository
func NewImpersonationRepository(db *database.Connection) *ImpersonationRepository {
	return &ImpersonationRepository{db: db}
}

// Create creates a new impersonation session
func (r *ImpersonationRepository) Create(ctx context.Context, session *ImpersonationSession) (*ImpersonationSession, error) {
	query := `
		INSERT INTO auth.impersonation_sessions
		(id, admin_user_id, target_user_id, reason, started_at, ip_address, user_agent, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, admin_user_id, target_user_id, reason, started_at, ended_at, ip_address, user_agent, is_active
	`

	row := r.db.QueryRow(ctx, query,
		session.ID,
		session.AdminUserID,
		session.TargetUserID,
		session.Reason,
		session.StartedAt,
		session.IPAddress,
		session.UserAgent,
		session.IsActive,
	)

	result := &ImpersonationSession{}
	err := row.Scan(
		&result.ID,
		&result.AdminUserID,
		&result.TargetUserID,
		&result.Reason,
		&result.StartedAt,
		&result.EndedAt,
		&result.IPAddress,
		&result.UserAgent,
		&result.IsActive,
	)

	if err != nil {
		return nil, err
	}

	return result, nil
}

// EndSession marks an impersonation session as ended
func (r *ImpersonationRepository) EndSession(ctx context.Context, sessionID string) error {
	query := `
		UPDATE auth.impersonation_sessions
		SET ended_at = NOW(), is_active = false
		WHERE id = $1 AND is_active = true
	`

	result, err := r.db.Exec(ctx, query, sessionID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrNoActiveImpersonation
	}

	return nil
}

// GetActiveByAdmin gets the active impersonation session for an admin
func (r *ImpersonationRepository) GetActiveByAdmin(ctx context.Context, adminUserID string) (*ImpersonationSession, error) {
	query := `
		SELECT id, admin_user_id, target_user_id, reason, started_at, ended_at, ip_address, user_agent, is_active
		FROM auth.impersonation_sessions
		WHERE admin_user_id = $1 AND is_active = true
		ORDER BY started_at DESC
		LIMIT 1
	`

	session := &ImpersonationSession{}
	err := r.db.QueryRow(ctx, query, adminUserID).Scan(
		&session.ID,
		&session.AdminUserID,
		&session.TargetUserID,
		&session.Reason,
		&session.StartedAt,
		&session.EndedAt,
		&session.IPAddress,
		&session.UserAgent,
		&session.IsActive,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoActiveImpersonation
		}
		return nil, err
	}

	return session, nil
}

// ListByAdmin lists all impersonation sessions for an admin (audit trail)
func (r *ImpersonationRepository) ListByAdmin(ctx context.Context, adminUserID string, limit, offset int) ([]*ImpersonationSession, error) {
	query := `
		SELECT id, admin_user_id, target_user_id, reason, started_at, ended_at, ip_address, user_agent, is_active
		FROM auth.impersonation_sessions
		WHERE admin_user_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, adminUserID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := []*ImpersonationSession{}
	for rows.Next() {
		session := &ImpersonationSession{}
		err := rows.Scan(
			&session.ID,
			&session.AdminUserID,
			&session.TargetUserID,
			&session.Reason,
			&session.StartedAt,
			&session.EndedAt,
			&session.IPAddress,
			&session.UserAgent,
			&session.IsActive,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// ImpersonationService provides business logic for admin impersonation
type ImpersonationService struct {
	repo       *ImpersonationRepository
	userRepo   *UserRepository
	jwtManager *JWTManager
}

// NewImpersonationService creates a new impersonation service
func NewImpersonationService(
	repo *ImpersonationRepository,
	userRepo *UserRepository,
	jwtManager *JWTManager,
) *ImpersonationService {
	return &ImpersonationService{
		repo:       repo,
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

// StartImpersonationRequest represents a request to start impersonating a user
type StartImpersonationRequest struct {
	TargetUserID string `json:"target_user_id"`
	Reason       string `json:"reason"`
	IPAddress    string `json:"-"` // Set from request context
	UserAgent    string `json:"-"` // Set from request context
}

// StartImpersonationResponse represents the response when starting impersonation
type StartImpersonationResponse struct {
	Session      *ImpersonationSession `json:"session"`
	TargetUser   *User                 `json:"target_user"`
	AccessToken  string                `json:"access_token"`
	RefreshToken string                `json:"refresh_token"`
	ExpiresIn    int64                 `json:"expires_in"`
}

// StartImpersonation starts an impersonation session
func (s *ImpersonationService) StartImpersonation(
	ctx context.Context,
	adminUserID string,
	req StartImpersonationRequest,
) (*StartImpersonationResponse, error) {
	// Verify admin user exists and is admin
	adminUser, err := s.userRepo.GetByID(ctx, adminUserID)
	if err != nil {
		return nil, fmt.Errorf("admin user not found: %w", err)
	}

	if adminUser.Role != "admin" {
		return nil, ErrNotAdmin
	}

	// Verify target user exists
	targetUser, err := s.userRepo.GetByID(ctx, req.TargetUserID)
	if err != nil {
		return nil, fmt.Errorf("target user not found: %w", err)
	}

	// Prevent self-impersonation
	if adminUserID == req.TargetUserID {
		return nil, ErrSelfImpersonation
	}

	// Check if admin already has an active impersonation session
	existingSession, err := s.repo.GetActiveByAdmin(ctx, adminUserID)
	if err == nil && existingSession != nil {
		// End the existing session first
		if err := s.repo.EndSession(ctx, existingSession.ID); err != nil {
			return nil, fmt.Errorf("failed to end existing session: %w", err)
		}
	}

	// Create new impersonation session
	session := &ImpersonationSession{
		ID:           uuid.New().String(),
		AdminUserID:  adminUserID,
		TargetUserID: req.TargetUserID,
		Reason:       req.Reason,
		StartedAt:    time.Now(),
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		IsActive:     true,
	}

	createdSession, err := s.repo.Create(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create impersonation session: %w", err)
	}

	// Generate JWT tokens for the target user
	// Note: The JWT contains the target user's info, but we track admin in the session
	accessToken, _, err := s.jwtManager.GenerateAccessToken(targetUser.ID, targetUser.Email, targetUser.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, _, err := s.jwtManager.GenerateRefreshToken(targetUser.ID, targetUser.Email, "")
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &StartImpersonationResponse{
		Session:      createdSession,
		TargetUser:   targetUser,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtManager.accessTokenTTL.Seconds()),
	}, nil
}

// StopImpersonation stops the active impersonation session for an admin
func (s *ImpersonationService) StopImpersonation(ctx context.Context, adminUserID string) error {
	// Get active session
	session, err := s.repo.GetActiveByAdmin(ctx, adminUserID)
	if err != nil {
		return err
	}

	// End the session
	return s.repo.EndSession(ctx, session.ID)
}

// GetActiveSession gets the active impersonation session for an admin
func (s *ImpersonationService) GetActiveSession(ctx context.Context, adminUserID string) (*ImpersonationSession, error) {
	return s.repo.GetActiveByAdmin(ctx, adminUserID)
}

// ListSessions lists impersonation sessions for audit purposes
func (s *ImpersonationService) ListSessions(ctx context.Context, adminUserID string, limit, offset int) ([]*ImpersonationSession, error) {
	return s.repo.ListByAdmin(ctx, adminUserID, limit, offset)
}
