package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTManager(t *testing.T) {
	secretKey := "test-secret-key"
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	manager := NewJWTManager(secretKey, accessTTL, refreshTTL)

	assert.NotNil(t, manager)
	assert.Equal(t, []byte(secretKey), manager.secretKey)
	assert.Equal(t, accessTTL, manager.accessTokenTTL)
	assert.Equal(t, refreshTTL, manager.refreshTokenTTL)
	assert.Equal(t, "fluxbase", manager.issuer)
}

func TestGenerateAccessToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	userID := "user123"
	email := "test@example.com"
	role := "user"

	token, claims, err := manager.GenerateAccessToken(userID, email, role, nil, nil)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotNil(t, claims)

	// Verify claims
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, "access", claims.TokenType)
	assert.NotEmpty(t, claims.SessionID)
	assert.Equal(t, "fluxbase", claims.Issuer)
	assert.Equal(t, userID, claims.Subject)
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotEmpty(t, claims.ID)

	// Verify expiry is approximately 15 minutes from now
	expectedExpiry := time.Now().Add(15 * time.Minute)
	assert.WithinDuration(t, expectedExpiry, claims.ExpiresAt.Time, 5*time.Second)
}

func TestGenerateRefreshToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	userID := "user123"
	email := "test@example.com"
	sessionID := "session123"

	token, claims, err := manager.GenerateRefreshToken(userID, email, sessionID, nil, nil)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotNil(t, claims)

	// Verify claims
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, sessionID, claims.SessionID)
	assert.Equal(t, "refresh", claims.TokenType)
	assert.Equal(t, "fluxbase", claims.Issuer)

	// Verify expiry is approximately 7 days from now
	expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
	assert.WithinDuration(t, expectedExpiry, claims.ExpiresAt.Time, 5*time.Second)
}

func TestGenerateTokenPair(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	userID := "user123"
	email := "test@example.com"
	role := "admin"

	accessToken, refreshToken, sessionID, err := manager.GenerateTokenPair(userID, email, role, nil, nil)

	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.NotEmpty(t, sessionID)

	// Validate both tokens
	accessClaims, err := manager.ValidateAccessToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, sessionID, accessClaims.SessionID)
	assert.Equal(t, userID, accessClaims.UserID)
	assert.Equal(t, role, accessClaims.Role)

	refreshClaims, err := manager.ValidateRefreshToken(refreshToken)
	require.NoError(t, err)
	assert.Equal(t, sessionID, refreshClaims.SessionID)
	assert.Equal(t, userID, refreshClaims.UserID)
}

func TestValidateToken_Success(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	userID := "user123"
	email := "test@example.com"
	role := "user"

	token, originalClaims, err := manager.GenerateAccessToken(userID, email, role, nil, nil)
	require.NoError(t, err)

	// Validate the token
	claims, err := manager.ValidateToken(token)

	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, originalClaims.UserID, claims.UserID)
	assert.Equal(t, originalClaims.Email, claims.Email)
	assert.Equal(t, originalClaims.Role, claims.Role)
	assert.Equal(t, originalClaims.SessionID, claims.SessionID)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"malformed token", "not.a.valid.token"},
		{"random string", "random-string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateToken(tt.token)

			assert.Error(t, err)
			assert.Nil(t, claims)
		})
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	manager1 := NewJWTManager("secret1", 15*time.Minute, 7*24*time.Hour)
	manager2 := NewJWTManager("secret2", 15*time.Minute, 7*24*time.Hour)

	token, _, err := manager1.GenerateAccessToken("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	// Try to validate with wrong secret
	claims, err := manager2.ValidateToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Create manager with very short TTL
	manager := NewJWTManager("test-secret", 1*time.Millisecond, 1*time.Millisecond)

	token, _, err := manager.GenerateAccessToken("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	claims, err := manager.ValidateToken(token)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrExpiredToken)
	assert.Nil(t, claims)
}

func TestValidateAccessToken_Success(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	token, _, err := manager.GenerateAccessToken("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	claims, err := manager.ValidateAccessToken(token)

	require.NoError(t, err)
	assert.Equal(t, "access", claims.TokenType)
}

func TestValidateAccessToken_RefreshTokenFails(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	token, _, err := manager.GenerateRefreshToken("user123", "test@example.com", "session123", nil, nil)
	require.NoError(t, err)

	claims, err := manager.ValidateAccessToken(token)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
	assert.Nil(t, claims)
}

func TestValidateRefreshToken_Success(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	token, _, err := manager.GenerateRefreshToken("user123", "test@example.com", "session123", nil, nil)
	require.NoError(t, err)

	claims, err := manager.ValidateRefreshToken(token)

	require.NoError(t, err)
	assert.Equal(t, "refresh", claims.TokenType)
}

func TestValidateRefreshToken_AccessTokenFails(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	token, _, err := manager.GenerateAccessToken("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	claims, err := manager.ValidateRefreshToken(token)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
	assert.Nil(t, claims)
}

func TestRefreshAccessToken_Success(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	// Generate initial token pair
	_, refreshToken, sessionID, err := manager.GenerateTokenPair("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	// Refresh the access token
	newAccessToken, err := manager.RefreshAccessToken(refreshToken)

	require.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)

	// Validate the new access token
	claims, err := manager.ValidateAccessToken(newAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
	// Note: Session ID will be different as we generate a new one
	assert.NotEmpty(t, claims.SessionID)
	// The original session ID should not match since we create a new session
	_ = sessionID
}

func TestRefreshAccessToken_InvalidRefreshToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	newToken, err := manager.RefreshAccessToken("invalid-token")

	assert.Error(t, err)
	assert.Empty(t, newToken)
}

func TestRefreshAccessToken_AccessTokenFails(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	accessToken, _, _, err := manager.GenerateTokenPair("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	// Try to refresh using access token (should fail)
	newToken, err := manager.RefreshAccessToken(accessToken)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
	assert.Empty(t, newToken)
}

func TestExtractUserID_Success(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	userID := "user123"
	token, _, err := manager.GenerateAccessToken(userID, "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	extractedUserID, err := manager.ExtractUserID(token)

	require.NoError(t, err)
	assert.Equal(t, userID, extractedUserID)
}

func TestExtractUserID_InvalidToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	userID, err := manager.ExtractUserID("invalid-token")

	assert.Error(t, err)
	assert.Empty(t, userID)
}

func TestGetTokenExpiry_Success(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	token, claims, err := manager.GenerateAccessToken("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	expiry, err := manager.GetTokenExpiry(token)

	require.NoError(t, err)
	assert.Equal(t, claims.ExpiresAt.Time, expiry)
	assert.True(t, expiry.After(time.Now()))
}

func TestGetTokenExpiry_InvalidToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	expiry, err := manager.GetTokenExpiry("invalid-token")

	assert.Error(t, err)
	assert.True(t, expiry.IsZero())
}

func TestTokenClaims_StandardCompliance(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	token, claims, err := manager.GenerateAccessToken("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	// Parse token to verify standard JWT compliance
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return manager.secretKey, nil
	})

	require.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	// Verify all standard claims are present
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.NotBefore)
	assert.NotEmpty(t, claims.ID)
	assert.NotEmpty(t, claims.Issuer)
	assert.NotEmpty(t, claims.Subject)
}

func TestConcurrentTokenGeneration(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	// Generate tokens concurrently
	const numGoroutines = 100
	results := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			token, _, err := manager.GenerateAccessToken(
				"user123",
				"test@example.com",
				"user",
				nil,
				nil,
			)
			require.NoError(t, err)
			results <- token
		}(i)
	}

	// Collect all tokens
	tokens := make(map[string]bool)
	for i := 0; i < numGoroutines; i++ {
		token := <-results
		tokens[token] = true
	}

	// All tokens should be unique
	assert.Len(t, tokens, numGoroutines)

	// All tokens should be valid
	for token := range tokens {
		claims, err := manager.ValidateAccessToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user123", claims.UserID)
	}
}

// Tests for Supabase-compatible service role JWT support

func TestGenerateServiceRoleToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	token, err := manager.GenerateServiceRoleToken()

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate the token
	claims, err := manager.ValidateServiceRoleToken(token)
	require.NoError(t, err)
	assert.Equal(t, "service_role", claims.Role)
	assert.Equal(t, "fluxbase", claims.Issuer)
	assert.Empty(t, claims.UserID) // Service role tokens have no user
}

func TestGenerateAnonToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	token, err := manager.GenerateAnonToken()

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate the token
	claims, err := manager.ValidateServiceRoleToken(token)
	require.NoError(t, err)
	assert.Equal(t, "anon", claims.Role)
	assert.Equal(t, "fluxbase", claims.Issuer)
	assert.Empty(t, claims.UserID)
}

func TestValidateServiceRoleToken_Success(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	tests := []struct {
		name     string
		role     string
		generate func() (string, error)
	}{
		{
			name: "service_role token",
			role: "service_role",
			generate: func() (string, error) {
				return manager.GenerateServiceRoleToken()
			},
		},
		{
			name: "anon token",
			role: "anon",
			generate: func() (string, error) {
				return manager.GenerateAnonToken()
			},
		},
		{
			name: "authenticated user token",
			role: "authenticated",
			generate: func() (string, error) {
				token, _, err := manager.GenerateAccessToken("user123", "test@example.com", "authenticated", nil, nil)
				return token, err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := tt.generate()
			require.NoError(t, err)

			claims, err := manager.ValidateServiceRoleToken(token)
			require.NoError(t, err)
			assert.Equal(t, tt.role, claims.Role)
		})
	}
}

func TestValidateServiceRoleToken_InvalidRole(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour)

	// Generate a token with an invalid role (e.g., "admin" is not in allowed list)
	token, _, err := manager.GenerateAccessToken("user123", "test@example.com", "admin", nil, nil)
	require.NoError(t, err)

	claims, err := manager.ValidateServiceRoleToken(token)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
	assert.Nil(t, claims)
}

func TestValidateServiceRoleToken_WrongSecret(t *testing.T) {
	manager1 := NewJWTManager("secret1", 15*time.Minute, 7*24*time.Hour)
	manager2 := NewJWTManager("secret2", 15*time.Minute, 7*24*time.Hour)

	token, err := manager1.GenerateServiceRoleToken()
	require.NoError(t, err)

	// Try to validate with wrong secret
	claims, err := manager2.ValidateServiceRoleToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateServiceRoleToken_SupabaseFormat(t *testing.T) {
	// Use a consistent secret for Supabase-style tokens
	secret := "super-secret-jwt-token-with-at-least-32-characters-long"
	manager := NewJWTManager(secret, 15*time.Minute, 7*24*time.Hour)

	tests := []struct {
		name   string
		role   string
		issuer string
	}{
		{"supabase-demo issuer with service_role", "service_role", "supabase-demo"},
		{"supabase-demo issuer with anon", "anon", "supabase-demo"},
		{"supabase issuer with service_role", "service_role", "supabase"},
		{"fluxbase issuer with service_role", "service_role", "fluxbase"},
		{"empty issuer with service_role", "service_role", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a Supabase-style JWT manually
			now := time.Now()
			claims := jwt.MapClaims{
				"role": tt.role,
				"iat":  now.Unix(),
				"exp":  now.Add(time.Hour).Unix(),
			}
			if tt.issuer != "" {
				claims["iss"] = tt.issuer
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, err := token.SignedString([]byte(secret))
			require.NoError(t, err)

			// Validate using our function
			parsedClaims, err := manager.ValidateServiceRoleToken(tokenString)
			require.NoError(t, err)
			assert.Equal(t, tt.role, parsedClaims.Role)
		})
	}
}

func TestValidateServiceRoleToken_InvalidIssuer(t *testing.T) {
	secret := "test-secret"
	manager := NewJWTManager(secret, 15*time.Minute, 7*24*time.Hour)

	// Create a token with an unknown issuer
	now := time.Now()
	claims := jwt.MapClaims{
		"role": "service_role",
		"iss":  "unknown-issuer",
		"iat":  now.Unix(),
		"exp":  now.Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	parsedClaims, err := manager.ValidateServiceRoleToken(tokenString)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
	assert.Nil(t, parsedClaims)
}

func TestValidateServiceRoleToken_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	manager := NewJWTManager(secret, 15*time.Minute, 7*24*time.Hour)

	// Create an expired token
	now := time.Now()
	claims := jwt.MapClaims{
		"role": "service_role",
		"iss":  "supabase-demo",
		"iat":  now.Add(-2 * time.Hour).Unix(),
		"exp":  now.Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	parsedClaims, err := manager.ValidateServiceRoleToken(tokenString)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrExpiredToken)
	assert.Nil(t, parsedClaims)
}
