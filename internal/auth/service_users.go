package auth

import (
	"context"
	"errors"
	"fmt"
	"time"
)

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
	// Validate email if provided
	if req.Email != nil {
		if err := ValidateEmail(*req.Email); err != nil {
			return nil, fmt.Errorf("invalid email: %w", err)
		}
	}
	return s.userRepo.Update(ctx, userID, req)
}

// SendMagicLink sends a magic link to the specified email
func (s *Service) SendMagicLink(ctx context.Context, email string) error {
	// Check if magic link is enabled from database settings (with fallback to config)
	enableMagicLink := s.settingsCache.GetBool(ctx, "app.auth.magic_link_enabled", s.config.MagicLinkEnabled)
	if !enableMagicLink {
		return fmt.Errorf("magic link authentication is disabled")
	}

	return s.magicLinkService.SendMagicLink(ctx, email)
}

// VerifyMagicLink verifies a magic link and returns tokens
func (s *Service) VerifyMagicLink(ctx context.Context, token string) (*SignInResponse, error) {
	// Check if magic link is enabled from database settings (with fallback to config)
	enableMagicLink := s.settingsCache.GetBool(ctx, "app.auth.magic_link_enabled", s.config.MagicLinkEnabled)
	if !enableMagicLink {
		return nil, fmt.Errorf("magic link authentication is disabled")
	}

	// Verify the magic link
	email, err := s.magicLinkService.VerifyMagicLink(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to verify magic link: %w", err)
	}

	// Get existing user - auto-creation is disabled for security
	// Users must register via signup endpoint first
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, fmt.Errorf("no account found for this email - please sign up first")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Generate tokens with metadata
	accessToken, refreshToken, _, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
		user.UserMetadata,
		user.AppMetadata,
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

// GetUserByEmail retrieves a user by email
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// CreateUser creates a new user with email and optional password
func (s *Service) CreateUser(ctx context.Context, email, password string) (*User, error) {
	// Validate email format and length
	if err := ValidateEmail(email); err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	// If password is empty, create user without password (for OTP/OAuth flows)
	hashedPassword := ""
	if password != "" {
		hash, err := s.passwordHasher.HashPassword(password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		hashedPassword = hash
	}

	req := CreateUserRequest{
		Email:    email,
		Password: password,
		Role:     "user",
	}
	return s.userRepo.Create(ctx, req, hashedPassword)
}

// IsEmailVerificationRequired checks if email verification is required based on settings and email configuration
func (s *Service) IsEmailVerificationRequired(ctx context.Context) bool {
	return s.emailVerificationService.IsEmailVerificationRequired(ctx)
}

// SendOTP sends an OTP code via email
func (s *Service) SendOTP(ctx context.Context, email, purpose string) error {
	if s.otpService == nil {
		return fmt.Errorf("OTP service not initialized")
	}
	return s.otpService.SendEmailOTP(ctx, email, purpose)
}

// SignInWithIDToken signs in a user with an OAuth ID token (Google, Apple, Microsoft, or custom OIDC)
func (s *Service) SignInWithIDToken(ctx context.Context, provider, idToken, nonce string) (*SignInResponse, error) {
	// Check if the provider is configured
	if !s.oidcVerifier.IsProviderConfigured(provider) {
		return nil, fmt.Errorf("OIDC provider not configured: %s", provider)
	}

	// Verify the ID token and extract claims
	claims, err := s.oidcVerifier.Verify(ctx, provider, idToken, nonce)
	if err != nil {
		return nil, fmt.Errorf("invalid ID token: %w", err)
	}

	// Require email for user lookup/creation
	if claims.Email == "" {
		return nil, fmt.Errorf("ID token does not contain email claim")
	}

	// Look up existing user by email
	user, err := s.userRepo.GetByEmail(ctx, claims.Email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}

	if user == nil {
		// Create new user from OIDC claims
		user, err = s.createOIDCUser(ctx, provider, claims)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	} else {
		// Update user info from OIDC claims if changed
		if err := s.updateUserFromOIDCClaims(ctx, user, claims); err != nil {
			// Log but don't fail the sign-in
			fmt.Printf("warning: failed to update user from OIDC claims: %v\n", err)
		}
	}

	// Generate JWT tokens
	accessToken, refreshToken, _, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Email,
		user.Role,
		user.UserMetadata,
		user.AppMetadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &SignInResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.JWTExpiry.Seconds()),
	}, nil
}

// createOIDCUser creates a new user from OIDC claims
func (s *Service) createOIDCUser(ctx context.Context, provider string, claims *IDTokenClaims) (*User, error) {
	req := CreateUserRequest{
		Email:    claims.Email,
		Password: "", // No password for OIDC users
		Role:     "authenticated",
		UserMetadata: map[string]interface{}{
			"name":    claims.Name,
			"picture": claims.Picture,
		},
		AppMetadata: map[string]interface{}{
			"provider":         provider,
			"provider_user_id": claims.Subject,
		},
	}

	// Create user with empty password hash (OIDC users don't have passwords)
	user, err := s.userRepo.Create(ctx, req, "")
	if err != nil {
		return nil, err
	}

	// Update email_verified if the OIDC provider verified it
	if claims.EmailVerified {
		emailVerified := true
		_, err = s.userRepo.Update(ctx, user.ID, UpdateUserRequest{
			EmailVerified: &emailVerified,
		})
		if err != nil {
			// Log but don't fail - user was created
			fmt.Printf("warning: failed to update email_verified: %v\n", err)
		}
		user.EmailVerified = true
	}

	return user, nil
}

// computeOIDCUserUpdate is the pure decision core of updateUserFromOIDCClaims:
// it computes what (if anything) should change given the current user and the
// fresh OIDC claims. Returns the update request and whether any field changed;
// the caller persists it via the repository. Extracted so the policy
// (verified-email promotion, name/picture sync) is unit-testable without a DB.
func computeOIDCUserUpdate(user *User, claims *IDTokenClaims) (UpdateUserRequest, bool) {
	updateReq := UpdateUserRequest{}
	needsUpdate := false

	// Update email verification status if changed
	if claims.EmailVerified && !user.EmailVerified {
		emailVerified := true
		updateReq.EmailVerified = &emailVerified
		needsUpdate = true
	}

	// Update user metadata if name or picture changed
	currentMetadata, _ := user.UserMetadata.(map[string]interface{})
	if currentMetadata == nil {
		currentMetadata = make(map[string]interface{})
	}

	newMetadata := make(map[string]interface{})
	for k, v := range currentMetadata {
		newMetadata[k] = v
	}

	if claims.Name != "" {
		if currentName, _ := currentMetadata["name"].(string); currentName != claims.Name {
			newMetadata["name"] = claims.Name
			needsUpdate = true
		}
	}

	if claims.Picture != "" {
		if currentPic, _ := currentMetadata["picture"].(string); currentPic != claims.Picture {
			newMetadata["picture"] = claims.Picture
			needsUpdate = true
		}
	}

	if needsUpdate {
		updateReq.UserMetadata = newMetadata
	}
	return updateReq, needsUpdate
}

// updateUserFromOIDCClaims updates user info from OIDC claims if changed
func (s *Service) updateUserFromOIDCClaims(ctx context.Context, user *User, claims *IDTokenClaims) error {
	updateReq, needsUpdate := computeOIDCUserUpdate(user, claims)
	if !needsUpdate {
		return nil
	}
	_, err := s.userRepo.Update(ctx, user.ID, updateReq)
	return err
}

// =============================================================================
// SAML SSO Methods
// =============================================================================

// CreateSAMLUser creates a new user from a SAML assertion
func (s *Service) CreateSAMLUser(ctx context.Context, email, name, provider, nameID string, attrs map[string][]string) (*User, error) {
	// Validate email format
	if err := ValidateEmail(email); err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	// Build user metadata with name if provided
	userMetadata := make(map[string]interface{})
	if name != "" {
		userMetadata["full_name"] = name
	}

	// Create user without password (SAML users authenticate via IdP)
	req := CreateUserRequest{
		Email:        email,
		Password:     "",
		Role:         "authenticated",
		UserMetadata: userMetadata,
	}

	user, err := s.userRepo.Create(ctx, req, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Link SAML identity using identity service
	if err := s.LinkSAMLIdentity(ctx, user.ID, provider, nameID, attrs); err != nil {
		// Log warning but don't fail - user was created successfully
		_ = err // Ignore error, user is still valid
	}

	// Refresh user to get updated data
	user, err = s.userRepo.GetByID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created user: %w", err)
	}

	return user, nil
}

// LinkSAMLIdentity links or updates a SAML identity for a user
func (s *Service) LinkSAMLIdentity(ctx context.Context, userID, provider, nameID string, attrs map[string][]string) error {
	// Create identity data that includes SAML-specific fields
	identityData := map[string]interface{}{
		"saml_name_id":    nameID,
		"saml_attributes": attrs,
	}

	// Extract email from SAML attributes if present
	var email *string
	if emails, ok := attrs["email"]; ok && len(emails) > 0 {
		email = &emails[0]
	}

	// Use the identity service to link the SAML identity
	// Provider format: "saml:{provider_name}"
	_, err := s.identityService.LinkIdentity(ctx, userID, "saml:"+provider, nameID, email, identityData)
	return err
}
