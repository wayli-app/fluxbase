package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wayli-app/fluxbase/internal/database"
)

var (
	// ErrInvitationNotFound is returned when an invitation token is not found
	ErrInvitationNotFound = errors.New("invitation not found")
	// ErrInvitationExpired is returned when an invitation token has expired
	ErrInvitationExpired = errors.New("invitation has expired")
	// ErrInvitationAlreadyAccepted is returned when an invitation has already been accepted
	ErrInvitationAlreadyAccepted = errors.New("invitation has already been accepted")
)

// InvitationToken represents an invitation for a new user
type InvitationToken struct {
	ID         uuid.UUID  `json:"id"`
	Email      string     `json:"email"`
	Token      string     `json:"token"`
	Role       string     `json:"role"`
	InvitedBy  *uuid.UUID `json:"invited_by,omitempty"`
	ExpiresAt  time.Time  `json:"expires_at"`
	Accepted   bool       `json:"accepted"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// InvitationService handles user invitation operations
type InvitationService struct {
	db *database.Connection
}

// NewInvitationService creates a new invitation service
func NewInvitationService(db *database.Connection) *InvitationService {
	return &InvitationService{db: db}
}

// GenerateToken generates a cryptographically secure random token
func (s *InvitationService) GenerateToken() (string, error) {
	// Generate 32 bytes of random data
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode to URL-safe base64
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CreateInvitation creates a new invitation token
func (s *InvitationService) CreateInvitation(ctx context.Context, email, role string, invitedBy *uuid.UUID, expiryDuration time.Duration) (*InvitationToken, error) {
	// Generate secure token
	token, err := s.GenerateToken()
	if err != nil {
		return nil, err
	}

	// Calculate expiration time (default 7 days)
	if expiryDuration == 0 {
		expiryDuration = 7 * 24 * time.Hour
	}
	expiresAt := time.Now().Add(expiryDuration)

	// Insert invitation token
	invitation := &InvitationToken{
		ID:        uuid.New(),
		Email:     email,
		Token:     token,
		Role:      role,
		InvitedBy: invitedBy,
		ExpiresAt: expiresAt,
		Accepted:  false,
		CreatedAt: time.Now(),
	}

	err = s.db.QueryRow(ctx, `
		INSERT INTO dashboard.invitation_tokens (id, email, token, role, invited_by, expires_at, accepted, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, email, token, role, invited_by, expires_at, accepted, created_at
	`,
		invitation.ID,
		invitation.Email,
		invitation.Token,
		invitation.Role,
		invitation.InvitedBy,
		invitation.ExpiresAt,
		invitation.Accepted,
		invitation.CreatedAt,
	).Scan(
		&invitation.ID,
		&invitation.Email,
		&invitation.Token,
		&invitation.Role,
		&invitation.InvitedBy,
		&invitation.ExpiresAt,
		&invitation.Accepted,
		&invitation.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return invitation, nil
}

// ValidateToken validates an invitation token and returns the invitation
func (s *InvitationService) ValidateToken(ctx context.Context, token string) (*InvitationToken, error) {
	invitation := &InvitationToken{}

	err := s.db.QueryRow(ctx, `
		SELECT id, email, token, role, invited_by, expires_at, accepted, accepted_at, created_at
		FROM dashboard.invitation_tokens
		WHERE token = $1
	`, token).Scan(
		&invitation.ID,
		&invitation.Email,
		&invitation.Token,
		&invitation.Role,
		&invitation.InvitedBy,
		&invitation.ExpiresAt,
		&invitation.Accepted,
		&invitation.AcceptedAt,
		&invitation.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}

	// Check if already accepted
	if invitation.Accepted {
		return nil, ErrInvitationAlreadyAccepted
	}

	// Check if expired
	if time.Now().After(invitation.ExpiresAt) {
		return nil, ErrInvitationExpired
	}

	return invitation, nil
}

// AcceptInvitation marks an invitation as accepted
func (s *InvitationService) AcceptInvitation(ctx context.Context, token string) error {
	now := time.Now()

	result, err := s.db.Exec(ctx, `
		UPDATE dashboard.invitation_tokens
		SET accepted = true, accepted_at = $1
		WHERE token = $2 AND accepted = false AND expires_at > $1
	`, now, token)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		// Token either doesn't exist, already accepted, or expired
		// Need to query to determine which
		_, err := s.ValidateToken(ctx, token)
		return err
	}

	return nil
}

// RevokeInvitation revokes (deletes) an invitation token
func (s *InvitationService) RevokeInvitation(ctx context.Context, token string) error {
	result, err := s.db.Exec(ctx, `
		DELETE FROM dashboard.invitation_tokens WHERE token = $1
	`, token)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrInvitationNotFound
	}

	return nil
}

// GetInvitationByEmail retrieves pending invitations for an email
func (s *InvitationService) GetInvitationByEmail(ctx context.Context, email string) ([]InvitationToken, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, email, token, role, invited_by, expires_at, accepted, accepted_at, created_at
		FROM dashboard.invitation_tokens
		WHERE email = $1 AND accepted = false AND expires_at > NOW()
		ORDER BY created_at DESC
	`, email)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []InvitationToken
	for rows.Next() {
		var inv InvitationToken
		err := rows.Scan(
			&inv.ID,
			&inv.Email,
			&inv.Token,
			&inv.Role,
			&inv.InvitedBy,
			&inv.ExpiresAt,
			&inv.Accepted,
			&inv.AcceptedAt,
			&inv.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		invitations = append(invitations, inv)
	}

	return invitations, rows.Err()
}

// ListInvitations retrieves all invitations (for admin panel)
func (s *InvitationService) ListInvitations(ctx context.Context, includeAccepted, includeExpired bool) ([]InvitationToken, error) {
	query := `
		SELECT id, email, token, role, invited_by, expires_at, accepted, accepted_at, created_at
		FROM dashboard.invitation_tokens
		WHERE 1=1
	`

	args := []interface{}{}

	if !includeAccepted {
		query += " AND accepted = false"
	}

	if !includeExpired {
		query += " AND expires_at > NOW()"
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []InvitationToken
	for rows.Next() {
		var inv InvitationToken
		err := rows.Scan(
			&inv.ID,
			&inv.Email,
			&inv.Token,
			&inv.Role,
			&inv.InvitedBy,
			&inv.ExpiresAt,
			&inv.Accepted,
			&inv.AcceptedAt,
			&inv.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		invitations = append(invitations, inv)
	}

	return invitations, rows.Err()
}

// CleanupExpiredInvitations removes expired invitation tokens
func (s *InvitationService) CleanupExpiredInvitations(ctx context.Context) (int64, error) {
	result, err := s.db.Exec(ctx, `
		DELETE FROM dashboard.invitation_tokens
		WHERE expires_at < NOW() AND accepted = false
	`)

	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}
