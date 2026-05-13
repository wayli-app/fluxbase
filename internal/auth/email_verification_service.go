package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/nimbleflux/fluxbase/internal/settings"
)

type EmailVerificationService struct {
	repo                    *EmailVerificationRepository
	userRepo                *UserRepository
	settingsCache           *settings.SettingsCache
	emailService            EmailService
	baseURL                 string
	emailVerificationExpiry time.Duration
}

func NewEmailVerificationService(
	repo *EmailVerificationRepository,
	userRepo *UserRepository,
	settingsCache *settings.SettingsCache,
	emailService EmailService,
	baseURL string,
	emailVerificationExpiry time.Duration,
) *EmailVerificationService {
	return &EmailVerificationService{
		repo:                    repo,
		userRepo:                userRepo,
		settingsCache:           settingsCache,
		emailService:            emailService,
		baseURL:                 baseURL,
		emailVerificationExpiry: emailVerificationExpiry,
	}
}

func (s *EmailVerificationService) IsEmailVerificationRequired(ctx context.Context) bool {
	required := s.settingsCache.GetBool(ctx, "app.auth.require_email_verification", false)
	if !required {
		return false
	}

	if s.emailService == nil {
		return false
	}
	return s.emailService.IsConfigured()
}

func (s *EmailVerificationService) SendEmailVerification(ctx context.Context, userID, email string) error {
	if s.emailService == nil || !s.emailService.IsConfigured() {
		return fmt.Errorf("email service is not configured")
	}

	_ = s.repo.DeleteByUserID(ctx, userID)

	tokenWithPlaintext, err := s.repo.Create(ctx, userID, s.emailVerificationExpiry)
	if err != nil {
		return fmt.Errorf("failed to create verification token: %w", err)
	}

	link := fmt.Sprintf("%s/auth/verify-email?token=%s", s.baseURL, tokenWithPlaintext.PlaintextToken)

	if err := s.emailService.SendVerificationEmail(ctx, email, tokenWithPlaintext.PlaintextToken, link); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}

func (s *EmailVerificationService) VerifyEmailToken(ctx context.Context, token string) (*User, error) {
	emailToken, err := s.repo.Validate(ctx, token)
	if err != nil {
		return nil, err
	}

	if err := s.repo.MarkAsUsed(ctx, emailToken.ID); err != nil {
		return nil, fmt.Errorf("failed to mark token as used: %w", err)
	}

	if err := s.userRepo.VerifyEmail(ctx, emailToken.UserID); err != nil {
		return nil, fmt.Errorf("failed to verify email: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, emailToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}
