package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

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
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// MagicLinkRepository handles database operations for magic links
type MagicLinkRepository struct {
	db *database.Connection
}

// NewMagicLinkRepository creates a new magic link repository
func NewMagicLinkRepository(db *database.Connection) *MagicLinkRepository {
	return &MagicLinkRepository{db: db}
}

// Create creates a new magic link
func (r *MagicLinkRepository) Create(ctx context.Context, email string, expiryDuration time.Duration) (*MagicLink, error) {
	token, err := GenerateMagicLinkToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	magicLink := &MagicLink{
		ID:        uuid.New().String(),
		Email:     email,
		Token:     token,
		ExpiresAt: time.Now().Add(expiryDuration),
		CreatedAt: time.Now(),
	}

	query := `
		INSERT INTO auth.magic_links (id, email, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, token, expires_at, used_at, created_at
	`

	err = r.db.QueryRow(ctx, query,
		magicLink.ID,
		magicLink.Email,
		magicLink.Token,
		magicLink.ExpiresAt,
		magicLink.CreatedAt,
	).Scan(
		&magicLink.ID,
		&magicLink.Email,
		&magicLink.Token,
		&magicLink.ExpiresAt,
		&magicLink.UsedAt,
		&magicLink.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return magicLink, nil
}

// GetByToken retrieves a magic link by token
func (r *MagicLinkRepository) GetByToken(ctx context.Context, token string) (*MagicLink, error) {
	query := `
		SELECT id, email, token, expires_at, used_at, created_at
		FROM auth.magic_links
		WHERE token = $1
	`

	magicLink := &MagicLink{}
	err := r.db.QueryRow(ctx, query, token).Scan(
		&magicLink.ID,
		&magicLink.Email,
		&magicLink.Token,
		&magicLink.ExpiresAt,
		&magicLink.UsedAt,
		&magicLink.CreatedAt,
	)

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

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrMagicLinkNotFound
	}

	return nil
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

	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

// DeleteByEmail deletes all magic links for an email
func (r *MagicLinkRepository) DeleteByEmail(ctx context.Context, email string) error {
	query := `DELETE FROM auth.magic_links WHERE email = $1`

	_, err := r.db.Exec(ctx, query, email)
	return err
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
	repo          *MagicLinkRepository
	userRepo      *UserRepository
	emailSender   EmailSender
	linkDuration  time.Duration
	baseURL       string
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

	// Create new magic link
	magicLink, err := s.repo.Create(ctx, email, s.linkDuration)
	if err != nil {
		return fmt.Errorf("failed to create magic link: %w", err)
	}

	// Generate the full link URL
	link := fmt.Sprintf("%s/auth/verify?token=%s", s.baseURL, magicLink.Token)

	// Send email
	if err := s.emailSender.SendMagicLink(ctx, email, magicLink.Token, link); err != nil {
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
