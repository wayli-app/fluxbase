package auth

import (
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var jwtLengthWarnOnce sync.Once

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
	UserID       string                 `json:"user_id"`
	Email        string                 `json:"email,omitempty"` // Empty for anonymous users
	Name         string                 `json:"name,omitempty"`  // Display name of the user
	Role         string                 `json:"role,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`    // Empty for anonymous users (no session)
	TokenType    string                 `json:"token_type"`              // "access" or "refresh"
	IsAnonymous  bool                   `json:"is_anonymous,omitempty"`  // True for anonymous users
	UserMetadata any                    `json:"user_metadata,omitempty"` // User-editable metadata
	AppMetadata  any                    `json:"app_metadata,omitempty"`  // Application/admin-only metadata
	RawClaims    map[string]interface{} `json:"-"`                       // Full claims map for RLS (not serialized)
	jwt.RegisteredClaims

	// Multi-tenancy fields
	TenantID        *string `json:"tenant_id,omitempty"`         // Current tenant ID
	TenantRole      string  `json:"tenant_role,omitempty"`       // User's role in current tenant (tenant_admin, tenant_member)
	IsInstanceAdmin bool    `json:"is_instance_admin,omitempty"` // True for instance-level admins

	// Impersonation tracking - for security audit and revocation
	ImpersonatedBy string `json:"impersonated_by,omitempty"` // Admin user ID who issued this impersonation token
}

// TokenOption is a functional option for token generation
type TokenOption func(*tokenOptions)

// tokenOptions holds options for token generation
type tokenOptions struct {
	impersonatedBy  string
	tenantID        *string
	tenantRole      string
	isInstanceAdmin bool
}

// WithImpersonatedBy sets the admin user ID who is impersonating
func WithImpersonatedBy(adminID string) TokenOption {
	return func(o *tokenOptions) {
		o.impersonatedBy = adminID
	}
}

// WithTenantContext sets tenant context for the generated token
func WithTenantContext(tenantID, tenantRole string, isInstanceAdmin bool) TokenOption {
	return func(o *tokenOptions) {
		o.tenantID = &tenantID
		o.tenantRole = tenantRole
		o.isInstanceAdmin = isInstanceAdmin
	}
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
func NewJWTManager(secretKey string, accessTTL, refreshTTL time.Duration) (*JWTManager, error) {
	if len(secretKey) < 32 {
		return nil, errors.New("JWT secret key must be at least 32 characters (256 bits)")
	}

	if len(secretKey) < 64 {
		jwtLengthWarnOnce.Do(func() {
			log.Warn().
				Int("length", len(secretKey)).
				Msg("JWT secret key is shorter than recommended 64 characters")
		})
	}

	return &JWTManager{
		secretKey:       []byte(secretKey),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
		serviceRoleTTL:  24 * time.Hour,
		anonTTL:         24 * time.Hour,
		issuer:          "fluxbase",
	}, nil
}

// NewJWTManagerWithConfig creates a new JWT manager with full configuration
func NewJWTManagerWithConfig(secretKey string, accessTTL, refreshTTL, serviceRoleTTL, anonTTL time.Duration) (*JWTManager, error) {
	if len(secretKey) < 32 {
		return nil, errors.New("JWT secret key must be at least 32 characters (256 bits)")
	}

	if len(secretKey) < 64 {
		jwtLengthWarnOnce.Do(func() {
			log.Warn().
				Int("length", len(secretKey)).
				Msg("JWT secret key is shorter than recommended 64 characters")
		})
	}

	return &JWTManager{
		secretKey:       []byte(secretKey),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
		serviceRoleTTL:  serviceRoleTTL,
		anonTTL:         anonTTL,
		issuer:          "fluxbase",
	}, nil
}

// GenerateAccessToken generates a new access token
func (m *JWTManager) GenerateAccessToken(userID, email, role string, userMetadata, appMetadata any, opts ...TokenOption) (string, *TokenClaims, error) {
	now := time.Now()
	sessionID := uuid.New().String()

	// Apply token options
	options := &tokenOptions{}
	for _, opt := range opts {
		opt(options)
	}

	claims := &TokenClaims{
		UserID:          userID,
		Email:           email,
		Role:            role,
		SessionID:       sessionID,
		TokenType:       "access",
		UserMetadata:    userMetadata,
		AppMetadata:     appMetadata,
		ImpersonatedBy:  options.impersonatedBy,
		TenantID:        options.tenantID,
		TenantRole:      options.tenantRole,
		IsInstanceAdmin: options.isInstanceAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  []string{"fluxbase"},
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
func (m *JWTManager) GenerateRefreshToken(userID, email, role, sessionID string, userMetadata, appMetadata any, opts ...TokenOption) (string, *TokenClaims, error) {
	now := time.Now()

	// Apply token options
	options := &tokenOptions{}
	for _, opt := range opts {
		opt(options)
	}

	claims := &TokenClaims{
		UserID:          userID,
		Email:           email,
		Role:            role,
		SessionID:       sessionID,
		TokenType:       "refresh",
		UserMetadata:    userMetadata,
		AppMetadata:     appMetadata,
		ImpersonatedBy:  options.impersonatedBy,
		TenantID:        options.tenantID,
		TenantRole:      options.tenantRole,
		IsInstanceAdmin: options.isInstanceAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  []string{"fluxbase"},
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

// TenantTokenOptions contains options for generating tenant-aware tokens
type TenantTokenOptions struct {
	TenantID        *string
	TenantRole      string
	IsInstanceAdmin bool
}

// GenerateAccessTokenWithTenant generates a new access token with tenant context
func (m *JWTManager) GenerateAccessTokenWithTenant(userID, email, role string, userMetadata, appMetadata any, tenantOpts TenantTokenOptions) (string, *TokenClaims, error) {
	now := time.Now()
	sessionID := uuid.New().String()

	claims := &TokenClaims{
		UserID:          userID,
		Email:           email,
		Role:            role,
		SessionID:       sessionID,
		TokenType:       "access",
		UserMetadata:    userMetadata,
		AppMetadata:     appMetadata,
		TenantID:        tenantOpts.TenantID,
		TenantRole:      tenantOpts.TenantRole,
		IsInstanceAdmin: tenantOpts.IsInstanceAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  []string{"fluxbase"},
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

// GenerateTokenPairWithTenant generates both access and refresh tokens with tenant context
func (m *JWTManager) GenerateTokenPairWithTenant(userID, email, role string, userMetadata, appMetadata any, tenantOpts TenantTokenOptions) (accessToken, refreshToken string, sessionID string, err error) {
	// Generate access token with tenant context
	accessToken, claims, err := m.GenerateAccessTokenWithTenant(userID, email, role, userMetadata, appMetadata, tenantOpts)
	if err != nil {
		return "", "", "", err
	}

	sessionID = claims.SessionID

	// Generate refresh token with the same session ID, role, and tenant context
	now := time.Now()
	refreshClaims := &TokenClaims{
		UserID:          userID,
		Email:           email,
		Role:            role,
		SessionID:       sessionID,
		TokenType:       "refresh",
		UserMetadata:    userMetadata,
		AppMetadata:     appMetadata,
		TenantID:        tenantOpts.TenantID,
		TenantRole:      tenantOpts.TenantRole,
		IsInstanceAdmin: tenantOpts.IsInstanceAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  []string{"fluxbase"},
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = token.SignedString(m.secretKey)
	if err != nil {
		return "", "", "", err
	}

	return accessToken, refreshToken, sessionID, nil
}

// RefreshAccessTokenWithTenant generates a new access token from a refresh token, preserving tenant context
func (m *JWTManager) RefreshAccessTokenWithTenant(refreshTokenString string, tenantOpts *TenantTokenOptions) (string, error) {
	// Validate refresh token
	claims, err := m.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return "", err
	}

	// Use provided tenant options or preserve existing from refresh token
	opts := TenantTokenOptions{
		TenantID:        claims.TenantID,
		TenantRole:      claims.TenantRole,
		IsInstanceAdmin: claims.IsInstanceAdmin,
	}
	if tenantOpts != nil {
		if tenantOpts.TenantID != nil {
			opts.TenantID = tenantOpts.TenantID
		}
		if tenantOpts.TenantRole != "" {
			opts.TenantRole = tenantOpts.TenantRole
		}
		if tenantOpts.IsInstanceAdmin {
			opts.IsInstanceAdmin = true
		}
	}

	// Generate new access token with tenant context
	accessToken, _, err := m.GenerateAccessTokenWithTenant(claims.UserID, claims.Email, claims.Role, claims.UserMetadata, claims.AppMetadata, opts)
	if err != nil {
		return "", err
	}

	return accessToken, nil
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

	// Also parse the raw claims to capture custom claims not in TokenClaims struct
	// This is needed for RLS policies that use custom claims (e.g., meeting_id, player_id)
	rawToken, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return m.secretKey, nil
	})
	if rawToken != nil {
		if mapClaims, ok := rawToken.Claims.(jwt.MapClaims); ok {
			claims.RawClaims = make(map[string]interface{})
			for k, v := range mapClaims {
				claims.RawClaims[k] = v
			}
		}
	}

	return claims, nil
}

// ValidateTokenWithSecret validates and parses a JWT token using a specific secret key
// This is used for multi-tenant scenarios where each tenant may have a different JWT secret
func (m *JWTManager) ValidateTokenWithSecret(tokenString, secretKey string) (*TokenClaims, error) {
	if len(secretKey) < 32 {
		return nil, errors.New("JWT secret key must be at least 32 characters")
	}

	secret := []byte(secretKey)
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSignature
		}
		return secret, nil
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

	// Also parse the raw claims to capture custom claims not in TokenClaims struct
	// This is needed for RLS policies that use custom claims (e.g., meeting_id, player_id)
	rawToken, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if rawToken != nil {
		if mapClaims, ok := rawToken.Claims.(jwt.MapClaims); ok {
			claims.RawClaims = make(map[string]interface{})
			for k, v := range mapClaims {
				claims.RawClaims[k] = v
			}
		}
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
			Audience:  []string{"fluxbase"},
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
			Audience:  []string{"fluxbase"},
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
// Unlike user tokens, these don't require user lookup or revocation checks.
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
	if issuer != "" && issuer != "fluxbase" {
		return nil, ErrInvalidToken
	}

	// Validate role is one of the expected service roles
	role := claims.Role
	if role != "anon" && role != "service_role" && role != "authenticated" {
		return nil, ErrInvalidToken
	}

	// Also parse the raw claims to capture custom claims not in TokenClaims struct
	// This is needed for RLS policies that use custom claims (e.g., meeting_id, player_id)
	rawToken, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return m.secretKey, nil
	})
	if rawToken != nil {
		if mapClaims, ok := rawToken.Claims.(jwt.MapClaims); ok {
			claims.RawClaims = make(map[string]interface{})
			for k, v := range mapClaims {
				claims.RawClaims[k] = v
			}
		}
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
			Audience:  []string{"fluxbase"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.serviceRoleTTL)), // Configurable, default 24h
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// GenerateServiceRoleTokenWithTenant generates a JWT with service_role and tenant context
func (m *JWTManager) GenerateServiceRoleTokenWithTenant(tenantID *string) (string, error) {
	now := time.Now()

	claims := &TokenClaims{
		UserID:    "",             // No user for service role
		Role:      "service_role", // Service role bypasses RLS
		TokenType: "access",
		TenantID:  tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  []string{"fluxbase"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.serviceRoleTTL)),
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
			Audience:  []string{"fluxbase"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.anonTTL)), // Configurable, default 24h
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// GenerateAnonTokenWithTenant generates a JWT with anon role and tenant context
func (m *JWTManager) GenerateAnonTokenWithTenant(tenantID *string) (string, error) {
	now := time.Now()

	claims := &TokenClaims{
		UserID:    "",     // No user for anon
		Role:      "anon", // Anonymous role
		TokenType: "access",
		TenantID:  tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  []string{"fluxbase"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.anonTTL)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// GetMaxTokenTTL returns the maximum TTL among all token types
// This is used to determine how long a user-wide revocation marker should persist
func (m *JWTManager) GetMaxTokenTTL() time.Duration {
	maxTTL := m.accessTokenTTL
	if m.refreshTokenTTL > maxTTL {
		maxTTL = m.refreshTokenTTL
	}
	if m.serviceRoleTTL > maxTTL {
		maxTTL = m.serviceRoleTTL
	}
	if m.anonTTL > maxTTL {
		maxTTL = m.anonTTL
	}
	// Fallback to 7 days if all TTls are somehow zero or very small
	if maxTTL < 7*24*time.Hour {
		maxTTL = 7 * 24 * time.Hour
	}
	return maxTTL
}

// getMaxTokenTTL returns the maximum TTL among all token types
// This is used to determine how long a user-wide revocation marker should persist
func (m *JWTManager) getMaxTokenTTL() time.Duration {
	return m.GetMaxTokenTTL()
}
