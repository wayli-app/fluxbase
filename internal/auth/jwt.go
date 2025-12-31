package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	// ErrInvalidToken is returned when a token is invalid
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when a token has expired
	ErrExpiredToken = errors.New("token has expired")
	// ErrInvalidSignature is returned when token signature is invalid
	ErrInvalidSignature = errors.New("invalid token signature")
)

// TokenClaims represents the JWT claims
type TokenClaims struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email,omitempty"` // Empty for anonymous users
	Name         string `json:"name,omitempty"`  // Display name of the user
	Role         string `json:"role,omitempty"`
	SessionID    string `json:"session_id,omitempty"`    // Empty for anonymous users (no session)
	TokenType    string `json:"token_type"`              // "access" or "refresh"
	IsAnonymous  bool   `json:"is_anonymous,omitempty"`  // True for anonymous users
	UserMetadata any    `json:"user_metadata,omitempty"` // User-editable metadata
	AppMetadata  any    `json:"app_metadata,omitempty"`  // Application/admin-only metadata
	jwt.RegisteredClaims
}

// JWTManager handles JWT token operations
type JWTManager struct {
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	serviceRoleTTL  time.Duration
	anonTTL         time.Duration
	issuer          string
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:       []byte(secretKey),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
		serviceRoleTTL:  24 * time.Hour, // Default 24 hours
		anonTTL:         24 * time.Hour, // Default 24 hours
		issuer:          "fluxbase",
	}
}

// NewJWTManagerWithConfig creates a new JWT manager with full configuration
func NewJWTManagerWithConfig(secretKey string, accessTTL, refreshTTL, serviceRoleTTL, anonTTL time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:       []byte(secretKey),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
		serviceRoleTTL:  serviceRoleTTL,
		anonTTL:         anonTTL,
		issuer:          "fluxbase",
	}
}

// GenerateAccessToken generates a new access token
func (m *JWTManager) GenerateAccessToken(userID, email, role string, userMetadata, appMetadata any) (string, *TokenClaims, error) {
	now := time.Now()
	sessionID := uuid.New().String()

	claims := &TokenClaims{
		UserID:       userID,
		Email:        email,
		Role:         role,
		SessionID:    sessionID,
		TokenType:    "access",
		UserMetadata: userMetadata,
		AppMetadata:  appMetadata,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", nil, err
	}

	return tokenString, claims, nil
}

// GenerateRefreshToken generates a new refresh token
func (m *JWTManager) GenerateRefreshToken(userID, email, role, sessionID string, userMetadata, appMetadata any) (string, *TokenClaims, error) {
	now := time.Now()

	claims := &TokenClaims{
		UserID:       userID,
		Email:        email,
		Role:         role,
		SessionID:    sessionID,
		TokenType:    "refresh",
		UserMetadata: userMetadata,
		AppMetadata:  appMetadata,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", nil, err
	}

	return tokenString, claims, nil
}

// GenerateTokenPair generates both access and refresh tokens
func (m *JWTManager) GenerateTokenPair(userID, email, role string, userMetadata, appMetadata any) (accessToken, refreshToken string, sessionID string, err error) {
	// Generate access token
	accessToken, claims, err := m.GenerateAccessToken(userID, email, role, userMetadata, appMetadata)
	if err != nil {
		return "", "", "", err
	}

	sessionID = claims.SessionID

	// Generate refresh token with the same session ID and role
	refreshToken, _, err = m.GenerateRefreshToken(userID, email, role, sessionID, userMetadata, appMetadata)
	if err != nil {
		return "", "", "", err
	}

	return accessToken, refreshToken, sessionID, nil
}

// ValidateToken validates and parses a JWT token
func (m *JWTManager) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSignature
		}
		return m.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateAccessToken validates an access token specifically
func (m *JWTManager) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "access" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token specifically
func (m *JWTManager) ValidateRefreshToken(tokenString string) (*TokenClaims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshAccessToken generates a new access token from a refresh token
func (m *JWTManager) RefreshAccessToken(refreshTokenString string) (string, error) {
	// Validate refresh token
	claims, err := m.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return "", err
	}

	// Generate new access token with the same session ID and metadata
	accessToken, _, err := m.GenerateAccessToken(claims.UserID, claims.Email, claims.Role, claims.UserMetadata, claims.AppMetadata)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

// ExtractUserID extracts the user ID from a token
func (m *JWTManager) ExtractUserID(tokenString string) (string, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

// GetTokenExpiry returns when a token expires
func (m *JWTManager) GetTokenExpiry(tokenString string) (time.Time, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return time.Time{}, err
	}
	return claims.ExpiresAt.Time, nil
}

// GenerateAnonymousAccessToken generates an access token for an anonymous user
func (m *JWTManager) GenerateAnonymousAccessToken(userID string) (string, error) {
	now := time.Now()

	claims := &TokenClaims{
		UserID:      userID,
		Email:       "",     // No email for anonymous users
		Role:        "anon", // Anonymous role
		SessionID:   "",     // No session for anonymous users
		TokenType:   "access",
		IsAnonymous: true,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GenerateAnonymousRefreshToken generates a refresh token for an anonymous user
func (m *JWTManager) GenerateAnonymousRefreshToken(userID string) (string, error) {
	now := time.Now()

	claims := &TokenClaims{
		UserID:      userID,
		Email:       "",     // No email for anonymous users
		Role:        "anon", // Anonymous role
		SessionID:   "",     // No session for anonymous users
		TokenType:   "refresh",
		IsAnonymous: true,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateServiceRoleToken validates a JWT that contains a role claim (anon, service_role, authenticated)
// This is used for Supabase-compatible client keys which are JWTs with role claims.
// Unlike user tokens, these don't require user lookup or revocation checks.
// Accepts issuers: "fluxbase", "supabase-demo", "supabase"
func (m *JWTManager) ValidateServiceRoleToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSignature
		}
		return m.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Validate issuer - accept tokens from known issuers or no issuer
	issuer := claims.Issuer
	if issuer != "" && issuer != "fluxbase" && issuer != "supabase-demo" && issuer != "supabase" {
		return nil, ErrInvalidToken
	}

	// Validate role is one of the expected service roles
	role := claims.Role
	if role != "anon" && role != "service_role" && role != "authenticated" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GenerateServiceRoleToken generates a JWT with service_role that bypasses RLS
func (m *JWTManager) GenerateServiceRoleToken() (string, error) {
	now := time.Now()

	claims := &TokenClaims{
		UserID:    "",             // No user for service role
		Role:      "service_role", // Service role bypasses RLS
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.serviceRoleTTL)), // Configurable, default 24h
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// GenerateAnonToken generates a JWT with anon role for anonymous access
func (m *JWTManager) GenerateAnonToken() (string, error) {
	now := time.Now()

	claims := &TokenClaims{
		UserID:    "",     // No user for anon
		Role:      "anon", // Anonymous role
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.anonTTL)), // Configurable, default 24h
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}
