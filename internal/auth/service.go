package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
)

// Service provides a high-level authentication API
type Service struct {
	userRepo        *UserRepository
	sessionRepo     *SessionRepository
	magicLinkRepo   *MagicLinkRepository
	jwtManager      *JWTManager
	passwordHasher  *PasswordHasher
	oauthManager    *OAuthManager
	magicLinkService *MagicLinkService
	config          *config.AuthConfig
}

// NewService creates a new authentication service
func NewService(
	db *database.Connection,
	cfg *config.AuthConfig,
	emailService EmailSender,
	baseURL string,
) *Service {
	userRepo := NewUserRepository(db)
	sessionRepo := NewSessionRepository(db)
	magicLinkRepo := NewMagicLinkRepository(db)

	jwtManager := NewJWTManager(cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry)
	passwordHasher := NewPasswordHasherWithConfig(PasswordHasherConfig{MinLength: cfg.PasswordMinLen, Cost: cfg.BcryptCost})
	oauthManager := NewOAuthManager()

	magicLinkService := NewMagicLinkService(
		magicLinkRepo,
		userRepo,
		emailService,
		15*time.Minute, // TODO: Get from config
		baseURL,
	)

	return &Service{
		userRepo:         userRepo,
		sessionRepo:      sessionRepo,
		magicLinkRepo:    magicLinkRepo,
		jwtManager:       jwtManager,
		passwordHasher:   passwordHasher,
		oauthManager:     oauthManager,
		magicLinkService: magicLinkService,
		config:           cfg,
	}
}

// SignUpRequest represents a user registration request
type SignUpRequest struct {
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SignUpResponse represents a successful registration response
type SignUpResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// SignUp registers a new user with email and password
func (s *Service) SignUp(ctx context.Context, req SignUpRequest) (*SignUpResponse, error) {
	if !s.config.EnableSignup {
		return nil, fmt.Errorf("signup is disabled")
	}

	// Validate password
	if err := s.passwordHasher.ValidatePassword(req.Password); err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	// Hash password
	hashedPassword, err := s.passwordHasher.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user, err := s.userRepo.Create(ctx, CreateUserRequest{
		Email:    req.Email,
		Metadata: req.Metadata,
	}, hashedPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	accessToken, refreshToken, _, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session
	expiresAt := time.Now().Add(s.config.RefreshExpiry)
	_, err = s.sessionRepo.Create(ctx, user.ID, accessToken, refreshToken, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &SignUpResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// SignInRequest represents a login request
type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SignInResponse represents a successful login response
type SignInResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// SignIn authenticates a user with email and password
func (s *Service) SignIn(ctx context.Context, req SignInRequest) (*SignInResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, fmt.Errorf("invalid email or password")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	if err := s.passwordHasher.ComparePassword(user.PasswordHash, req.Password); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Generate tokens
	accessToken, refreshToken, _, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session
	expiresAt := time.Now().Add(s.config.RefreshExpiry)
	_, err = s.sessionRepo.Create(ctx, user.ID, accessToken, refreshToken, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &SignInResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// SignOut logs out a user by invalidating their session
func (s *Service) SignOut(ctx context.Context, accessToken string) error {
	// Get session by access token
	session, err := s.sessionRepo.GetByAccessToken(ctx, accessToken)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			// Already signed out or invalid token
			return nil
		}
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Delete session
	if err := s.sessionRepo.Delete(ctx, session.ID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse represents a successful token refresh
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// RefreshToken generates a new access token using a refresh token
func (s *Service) RefreshToken(ctx context.Context, req RefreshTokenRequest) (*RefreshTokenResponse, error) {
	// Validate refresh token
	claims, err := s.jwtManager.ValidateToken(req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("invalid token type")
	}

	// Get session by refresh token
	session, err := s.sessionRepo.GetByRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, fmt.Errorf("session not found or expired")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// Generate new access token
	newAccessToken, err := s.jwtManager.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update session with new access token
	if err := s.sessionRepo.UpdateAccessToken(ctx, session.ID, newAccessToken); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return &RefreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: req.RefreshToken, // Refresh token stays the same
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// GetUser retrieves the current user by access token
func (s *Service) GetUser(ctx context.Context, accessToken string) (*User, error) {
	// Validate token
	claims, err := s.jwtManager.ValidateToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Verify session still exists (not signed out)
	_, err = s.sessionRepo.GetByAccessToken(ctx, accessToken)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, fmt.Errorf("session not found or expired")
		}
		return nil, fmt.Errorf("failed to verify session: %w", err)
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUser updates user information
func (s *Service) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (*User, error) {
	return s.userRepo.Update(ctx, userID, req)
}

// SendMagicLink sends a magic link to the specified email
func (s *Service) SendMagicLink(ctx context.Context, email string) error {
	if !s.config.EnableMagicLink {
		return fmt.Errorf("magic link authentication is disabled")
	}

	return s.magicLinkService.SendMagicLink(ctx, email)
}

// VerifyMagicLink verifies a magic link and returns tokens
func (s *Service) VerifyMagicLink(ctx context.Context, token string) (*SignInResponse, error) {
	if !s.config.EnableMagicLink {
		return nil, fmt.Errorf("magic link authentication is disabled")
	}

	// Verify the magic link
	email, err := s.magicLinkService.VerifyMagicLink(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to verify magic link: %w", err)
	}

	// Get or create user
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Create new user for magic link
			user, err = s.userRepo.Create(ctx, CreateUserRequest{
				Email:    email,
				Metadata: nil,
			}, "") // No password for magic link users
			if err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
	}

	// Generate tokens
	accessToken, refreshToken, _, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session
	expiresAt := time.Now().Add(s.config.RefreshExpiry)
	_, err = s.sessionRepo.Create(ctx, user.ID, accessToken, refreshToken, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &SignInResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// ValidateToken validates an access token and returns the claims
func (s *Service) ValidateToken(token string) (*TokenClaims, error) {
	return s.jwtManager.ValidateToken(token)
}

// GetOAuthManager returns the OAuth manager for configuring providers
func (s *Service) GetOAuthManager() *OAuthManager {
	return s.oauthManager
}
