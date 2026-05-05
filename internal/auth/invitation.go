package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

var (
	// ErrInvitationNotFound is returned when an invitation token is not found
	ErrInvitationNotFound = errors.New("invitation not found")
	// ErrInvitationExpired is returned when an invitation token has expired
	ErrInvitationExpired = errors.New("invitation has expired")
	// ErrInvitationAlreadyAccepted is returned when an invitation has already been accepted
	ErrInvitationAlreadyAccepted = errors.New("invitation has already been accepted")
)

// hashInvitationToken creates a SHA-256 hash of a token and returns it as hex.
func hashInvitationToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// InvitationToken represents an invitation for a new user
type InvitationToken struct {
	ID         uuid.UUID  `json:"id"`
	Email      string     `json:"email"`
	Token      string     `json:"-" db:"token"`
	TokenHash  *string    `json:"-" db:"token_hash"`
	Role       string     `json:"role"`
	TenantID   *uuid.UUID `json:"tenant_id,omitempty"`
	InvitedBy  *uuid.UUID `json:"invited_by,omitempty"`
	ExpiresAt  time.Time  `json:"expires_at"`
	Accepted   bool       `json:"accepted"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// InvitationTokenWithPlaintext wraps InvitationToken with the plaintext token
// for one-time use (e.g., building invitation links). The plaintext is never
// exposed in JSON responses.
type InvitationTokenWithPlaintext struct {
	*InvitationToken
	PlaintextToken string `json:"token"`
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
func (s *InvitationService) CreateInvitation(ctx context.Context, email, role string, invitedBy *uuid.UUID, expiryDuration time.Duration) (*InvitationTokenWithPlaintext, error) {
	return s.CreateInvitationWithTenant(ctx, email, role, nil, invitedBy, expiryDuration)
}

// CreateInvitationWithTenant creates a new invitation token with an optional tenant context
func (s *InvitationService) CreateInvitationWithTenant(ctx context.Context, email, role string, tenantID *uuid.UUID, invitedBy *uuid.UUID, expiryDuration time.Duration) (*InvitationTokenWithPlaintext, error) {
	token, err := s.GenerateToken()
	if err != nil {
		return nil, err
	}

	if expiryDuration == 0 {
		expiryDuration = 7 * 24 * time.Hour
	}
	expiresAt := time.Now().Add(expiryDuration)

	tokenHash := hashInvitationToken(token)

	invitation := &InvitationToken{
		ID:        uuid.New(),
		Email:     email,
		Token:     token,
		TokenHash: &tokenHash,
		Role:      role,
		TenantID:  tenantID,
		InvitedBy: invitedBy,
		ExpiresAt: expiresAt,
		Accepted:  false,
		CreatedAt: time.Now(),
	}

	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, `
			INSERT INTO platform.invitation_tokens (id, email, token, token_hash, role, tenant_id, invited_by, expires_at, accepted, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id, email, token, token_hash, role, tenant_id, invited_by, expires_at, accepted, created_at
		`,
			invitation.ID,
			invitation.Email,
			invitation.Token,
			invitation.TokenHash,
			invitation.Role,
			invitation.TenantID,
			invitation.InvitedBy,
			invitation.ExpiresAt,
			invitation.Accepted,
			invitation.CreatedAt,
		).Scan(
			&invitation.ID,
			&invitation.Email,
			&invitation.Token,
			&invitation.TokenHash,
			&invitation.Role,
			&invitation.TenantID,
			&invitation.InvitedBy,
			&invitation.ExpiresAt,
			&invitation.Accepted,
			&invitation.CreatedAt,
		)
	})
	if err != nil {
		return nil, err
	}

	return &InvitationTokenWithPlaintext{InvitationToken: invitation, PlaintextToken: token}, nil
}

func validateInvitation(invitation *InvitationToken) error {
	if invitation.Accepted {
		return ErrInvitationAlreadyAccepted
	}
	if time.Now().After(invitation.ExpiresAt) {
		return ErrInvitationExpired
	}
	return nil
}

// ValidateToken validates an invitation token and returns the invitation
func (s *InvitationService) ValidateToken(ctx context.Context, token string) (*InvitationToken, error) {
	tokenHash := hashInvitationToken(token)
	invitation := &InvitationToken{}

	// Try hash-based lookup first
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, email, token, token_hash, role, tenant_id, invited_by, expires_at, accepted, accepted_at, created_at
			FROM platform.invitation_tokens
			WHERE token_hash = $1
		`, tokenHash).Scan(
			&invitation.ID,
			&invitation.Email,
			&invitation.Token,
			&invitation.TokenHash,
			&invitation.Role,
			&invitation.TenantID,
			&invitation.InvitedBy,
			&invitation.ExpiresAt,
			&invitation.Accepted,
			&invitation.AcceptedAt,
			&invitation.CreatedAt,
		)
	})
	if err == nil {
		return invitation, validateInvitation(invitation)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Fallback: plaintext lookup for legacy tokens
	err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT id, email, token, token_hash, role, tenant_id, invited_by, expires_at, accepted, accepted_at, created_at
			FROM platform.invitation_tokens
			WHERE token = $1
		`, token).Scan(
			&invitation.ID,
			&invitation.Email,
			&invitation.Token,
			&invitation.TokenHash,
			&invitation.Role,
			&invitation.TenantID,
			&invitation.InvitedBy,
			&invitation.ExpiresAt,
			&invitation.Accepted,
			&invitation.AcceptedAt,
			&invitation.CreatedAt,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}

	// Lazy migration: backfill hash for this legacy token
	if invitation.TokenHash == nil || *invitation.TokenHash == "" {
		if migrateErr := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
			_, execErr := tx.Exec(ctx, `UPDATE platform.invitation_tokens SET token_hash = $1 WHERE id = $2`, tokenHash, invitation.ID)
			return execErr
		}); migrateErr != nil {
			log.Debug().Err(migrateErr).Str("invitation_id", invitation.ID.String()).Msg("Failed to lazy-migrate invitation token hash")
		}
		invitation.TokenHash = &tokenHash
	}

	return invitation, validateInvitation(invitation)
}

// AcceptInvitation marks an invitation as accepted
func (s *InvitationService) AcceptInvitation(ctx context.Context, token string) error {
	now := time.Now()
	tokenHash := hashInvitationToken(token)

	// Try hash-based lookup first
	var result pgconn.CommandTag
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		var err error
		result, err = tx.Exec(ctx, `
			UPDATE platform.invitation_tokens
			SET accepted = true, accepted_at = $1
			WHERE token_hash = $2 AND accepted = false AND expires_at > $1
		`, now, tokenHash)
		return err
	})
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		// Fallback: try plaintext lookup for legacy tokens
		err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
			var err error
			result, err = tx.Exec(ctx, `
				UPDATE platform.invitation_tokens
				SET accepted = true, accepted_at = $1, token_hash = $3
				WHERE token = $2 AND accepted = false AND expires_at > $1
			`, now, token, tokenHash)
			return err
		})
		if err != nil {
			return err
		}
	}

	if result.RowsAffected() == 0 {
		_, err := s.ValidateToken(ctx, token)
		return err
	}

	return nil
}

// RevokeInvitation revokes (deletes) an invitation token
func (s *InvitationService) RevokeInvitation(ctx context.Context, token string) error {
	tokenHash := hashInvitationToken(token)

	var result pgconn.CommandTag
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		var err error
		result, err = tx.Exec(ctx, `
			DELETE FROM platform.invitation_tokens WHERE token_hash = $1
		`, tokenHash)
		return err
	})
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		// Fallback: try plaintext lookup
		err = database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
			var err error
			result, err = tx.Exec(ctx, `
				DELETE FROM platform.invitation_tokens WHERE token = $1
			`, token)
			return err
		})
		if err != nil {
			return err
		}
	}

	if result.RowsAffected() == 0 {
		return ErrInvitationNotFound
	}

	return nil
}

// GetInvitationByEmail retrieves pending invitations for an email
func (s *InvitationService) GetInvitationByEmail(ctx context.Context, email string) ([]InvitationToken, error) {
	var invitations []InvitationToken
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, email, token, token_hash, role, tenant_id, invited_by, expires_at, accepted, accepted_at, created_at
			FROM platform.invitation_tokens
			WHERE email = $1 AND accepted = false AND expires_at > NOW()
			ORDER BY created_at DESC
		`, email)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var inv InvitationToken
			if err := rows.Scan(
				&inv.ID,
				&inv.Email,
				&inv.Token,
				&inv.TokenHash,
				&inv.Role,
				&inv.TenantID,
				&inv.InvitedBy,
				&inv.ExpiresAt,
				&inv.Accepted,
				&inv.AcceptedAt,
				&inv.CreatedAt,
			); err != nil {
				return err
			}
			invitations = append(invitations, inv)
		}
		return rows.Err()
	})

	return invitations, err
}

// ListInvitations retrieves all invitations (for admin panel)
func (s *InvitationService) ListInvitations(ctx context.Context, includeAccepted, includeExpired bool) ([]InvitationToken, error) {
	query := `
		SELECT id, email, token, token_hash, role, tenant_id, invited_by, expires_at, accepted, accepted_at, created_at
		FROM platform.invitation_tokens
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

	var invitations []InvitationToken
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var inv InvitationToken
			if err := rows.Scan(
				&inv.ID,
				&inv.Email,
				&inv.Token,
				&inv.TokenHash,
				&inv.Role,
				&inv.TenantID,
				&inv.InvitedBy,
				&inv.ExpiresAt,
				&inv.Accepted,
				&inv.AcceptedAt,
				&inv.CreatedAt,
			); err != nil {
				return err
			}
			invitations = append(invitations, inv)
		}
		return rows.Err()
	})

	return invitations, err
}

// CleanupExpiredInvitations removes expired invitation tokens
func (s *InvitationService) CleanupExpiredInvitations(ctx context.Context) (int64, error) {
	var result pgconn.CommandTag
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		var err error
		result, err = tx.Exec(ctx, `
			DELETE FROM platform.invitation_tokens
			WHERE expires_at < NOW() AND accepted = false
		`)
		return err
	})
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}
