package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// hashMagicLinkToken creates a SHA-256 hash of a token and returns it as base64.
// SECURITY: Magic link tokens are stored as hashes to prevent exposure if the database is breached.
func hashMagicLinkToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(hash[:])
}

var (
	// ErrMagicLinkNotFound is returned when a magic link is not found
	ErrMagicLinkNotFound = errors.New("magic link not found")
	// ErrMagicLinkExpired is returned when a magic link has expired
	ErrMagicLinkExpired = errors.New("magic link has expired")
	// ErrMagicLinkUsed is returned when a magic link has already been used
	ErrMagicLinkUsed = errors.New("magic link has already been used")
)

// MagicLink represents a passwordless authentication link
type MagicLink struct {
	ID        string     `json:"id" db:"id"`
	Email     string     `json:"email" db:"email"`
	TokenHash string     `json:"-" db:"token_hash"` // SECURITY: Only hash is stored, never exposed in JSON
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// MagicLinkRepository handles database operations for magic links
type MagicLinkRepository struct {
	db *database.Connection
}

// NewMagicLinkRepository creates a new magic link repository
func NewMagicLinkRepository(db *database.Connection) *MagicLinkRepository {
	return &MagicLinkRepository{db: db}
}

// MagicLinkWithToken is returned from Create with the plaintext token for sending via email.
// The plaintext token is never stored in the database.
type MagicLinkWithToken struct {
	MagicLink
	PlaintextToken string // The plaintext token to send to the user (not stored)
}

// Create creates a new magic link
// SECURITY: The plaintext token is returned only once for sending via email. Only the hash is stored.
func (r *MagicLinkRepository) Create(ctx context.Context, email string, expiryDuration time.Duration) (*MagicLinkWithToken, error) {
	plaintextToken, err := GenerateMagicLinkToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Hash the token before storing
	tokenHash := hashMagicLinkToken(plaintextToken)

	magicLink := &MagicLinkWithToken{
		MagicLink: MagicLink{
			ID:        uuid.New().String(),
			Email:     email,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(expiryDuration),
			CreatedAt: time.Now(),
		},
		PlaintextToken: plaintextToken,
	}

	query := `
		INSERT INTO auth.magic_links (id, email, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, token_hash, expires_at, used_at, created_at
	`

	err = database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			magicLink.ID,
			magicLink.Email,
			magicLink.TokenHash,
			magicLink.ExpiresAt,
			magicLink.CreatedAt,
		).Scan(
			&magicLink.ID,
			&magicLink.Email,
			&magicLink.TokenHash,
			&magicLink.ExpiresAt,
			&magicLink.UsedAt,
			&magicLink.CreatedAt,
		)
	})

	if err != nil {
		return nil, err
	}

	return magicLink, nil
}

// GetByToken retrieves a magic link by token
// SECURITY: The incoming plaintext token is hashed before lookup
func (r *MagicLinkRepository) GetByToken(ctx context.Context, token string) (*MagicLink, error) {
	// Hash the incoming token for comparison
	tokenHash := hashMagicLinkToken(token)

	query := `
		SELECT id, email, token_hash, expires_at, used_at, created_at
		FROM auth.magic_links
		WHERE token_hash = $1
	`

	magicLink := &MagicLink{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, tokenHash).Scan(
			&magicLink.ID,
			&magicLink.Email,
			&magicLink.TokenHash,
			&magicLink.ExpiresAt,
			&magicLink.UsedAt,
			&magicLink.CreatedAt,
		)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMagicLinkNotFound
		}
		return nil, err
	}

	return magicLink, nil
}

// MarkAsUsed marks a magic link as used
func (r *MagicLinkRepository) MarkAsUsed(ctx context.Context, id string) error {
	query := `
		UPDATE auth.magic_links
		SET used_at = NOW()
		WHERE id = $1
	`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, id)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return ErrMagicLinkNotFound
		}

		return nil
	})
}

// Validate validates a magic link token
func (r *MagicLinkRepository) Validate(ctx context.Context, token string) (*MagicLink, error) {
	magicLink, err := r.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check if already used
	if magicLink.UsedAt != nil {
		return nil, ErrMagicLinkUsed
	}

	// Check if expired
	if time.Now().After(magicLink.ExpiresAt) {
		return nil, ErrMagicLinkExpired
	}

	return magicLink, nil
}

// DeleteExpired deletes all expired magic links
func (r *MagicLinkRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM auth.magic_links WHERE expires_at < NOW()`

	var rowsAffected int64
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query)
		if err != nil {
			return err
		}
		rowsAffected = result.RowsAffected()
		return nil
	})

	return rowsAffected, err
}

// DeleteByEmail deletes all magic links for an email
func (r *MagicLinkRepository) DeleteByEmail(ctx context.Context, email string) error {
	query := `DELETE FROM auth.magic_links WHERE email = $1`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, email)
		return err
	})
}

// GenerateMagicLinkToken generates a secure random token for magic links
func GenerateMagicLinkToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// MagicLinkService provides magic link functionality
type MagicLinkService struct {
	repo         *MagicLinkRepository
	userRepo     *UserRepository
	emailSender  EmailSender
	linkDuration time.Duration
	baseURL      string
}

// EmailSender defines the interface for sending emails
type EmailSender interface {
	SendMagicLink(ctx context.Context, to, token, link string) error
	SendPasswordReset(ctx context.Context, to, token, link string) error
}

// NewMagicLinkService creates a new magic link service
func NewMagicLinkService(
	repo *MagicLinkRepository,
	userRepo *UserRepository,
	emailSender EmailSender,
	linkDuration time.Duration,
	baseURL string,
) *MagicLinkService {
	return &MagicLinkService{
		repo:         repo,
		userRepo:     userRepo,
		emailSender:  emailSender,
		linkDuration: linkDuration,
		baseURL:      baseURL,
	}
}

// SendMagicLink sends a magic link to the specified email
func (s *MagicLinkService) SendMagicLink(ctx context.Context, email string) error {
	// Check if user exists (optional - you might want to create user on magic link verification)
	_, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return err
	}

	// Invalidate old magic links for this email
	_ = s.repo.DeleteByEmail(ctx, email)

	// Create new magic link (returns MagicLinkWithToken containing the plaintext token)
	magicLink, err := s.repo.Create(ctx, email, s.linkDuration)
	if err != nil {
		return fmt.Errorf("failed to create magic link: %w", err)
	}

	// Generate the full link URL using the plaintext token (only available at creation time)
	link := fmt.Sprintf("%s/auth/verify?token=%s", s.baseURL, magicLink.PlaintextToken)

	// Send email with plaintext token
	if err := s.emailSender.SendMagicLink(ctx, email, magicLink.PlaintextToken, link); err != nil {
		return fmt.Errorf("failed to send magic link email: %w", err)
	}

	return nil
}

// VerifyMagicLink verifies a magic link and returns the email
func (s *MagicLinkService) VerifyMagicLink(ctx context.Context, token string) (string, error) {
	// Validate the token
	magicLink, err := s.repo.Validate(ctx, token)
	if err != nil {
		return "", err
	}

	// Mark as used
	if err := s.repo.MarkAsUsed(ctx, magicLink.ID); err != nil {
		return "", fmt.Errorf("failed to mark magic link as used: %w", err)
	}

	return magicLink.Email, nil
}
