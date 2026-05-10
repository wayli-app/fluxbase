package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecretKey = "test-secret-key-must-be-32-characters!"

func TestNewJWTManager(t *testing.T) {
	accessTTL := 15 * time.Minute
	refreshTTL := 7 * 24 * time.Hour

	manager, err := NewJWTManager(testSecretKey, accessTTL, refreshTTL)
	require.NoError(t, err)

	assert.NotNil(t, manager)
	assert.Equal(t, []byte(testSecretKey), manager.secretKey)
	assert.Equal(t, accessTTL, manager.accessTokenTTL)
	assert.Equal(t, refreshTTL, manager.refreshTokenTTL)
	assert.Equal(t, "fluxbase", manager.issuer)
}

func TestNewJWTManager_SecretKeyValidation(t *testing.T) {
	tests := []struct {
		name      string
		secretKey string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid 32 character key",
			secretKey: "12345678901234567890123456789012",
			wantErr:   false,
		},
		{
			name:      "valid 64 character key",
			secretKey: "1234567890123456789012345678901212345678901234567890123456789012",
			wantErr:   false,
		},
		{
			name:      "too short - 31 characters",
			secretKey: "1234567890123456789012345678901",
			wantErr:   true,
			errMsg:    "JWT secret key must be at least 32 characters",
		},
		{
			name:      "too short - 1 character",
			secretKey: "x",
			wantErr:   true,
			errMsg:    "JWT secret key must be at least 32 characters",
		},
		{
			name:      "empty key",
			secretKey: "",
			wantErr:   true,
			errMsg:    "JWT secret key must be at least 32 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewJWTManager(tt.secretKey, 15*time.Minute, 7*24*time.Hour)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
			}
		})
	}
}

func TestGenerateAccessToken(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	userID := "user123"
	email := "test@example.com"
	role := "user"

	token, claims, err := manager.GenerateAccessToken(userID, email, role, nil, nil)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotNil(t, claims)

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

	expectedExpiry := time.Now().Add(15 * time.Minute)
	assert.WithinDuration(t, expectedExpiry, claims.ExpiresAt.Time, 5*time.Second)
}

func TestGenerateRefreshToken(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	userID := "user123"
	email := "test@example.com"
	role := "authenticated"
	sessionID := "session123"

	token, claims, err := manager.GenerateRefreshToken(userID, email, role, sessionID, nil, nil)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotNil(t, claims)

	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, sessionID, claims.SessionID)
	assert.Equal(t, "refresh", claims.TokenType)
	assert.Equal(t, "fluxbase", claims.Issuer)

	expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
	assert.WithinDuration(t, expectedExpiry, claims.ExpiresAt.Time, 5*time.Second)
}

func TestGenerateTokenPair(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	userID := "user123"
	email := "test@example.com"
	role := "admin"

	accessToken, refreshToken, sessionID, err := manager.GenerateTokenPair(userID, email, role, nil, nil)

	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.NotEmpty(t, sessionID)

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
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	userID := "user123"
	email := "test@example.com"
	role := "user"

	token, originalClaims, err := manager.GenerateAccessToken(userID, email, role, nil, nil)
	require.NoError(t, err)

	claims, err := manager.ValidateToken(token)

	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, originalClaims.UserID, claims.UserID)
	assert.Equal(t, originalClaims.Email, claims.Email)
	assert.Equal(t, originalClaims.Role, claims.Role)
	assert.Equal(t, originalClaims.SessionID, claims.SessionID)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

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
	manager1, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)
	manager2, err := NewJWTManager("different-secret-key-must-be-32-chars!", 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	token, _, err := manager1.GenerateAccessToken("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	claims, err := manager2.ValidateToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 1*time.Millisecond, 1*time.Millisecond)
	require.NoError(t, err)

	token, _, err := manager.GenerateAccessToken("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	claims, err := manager.ValidateToken(token)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrExpiredToken)
	assert.Nil(t, claims)
}

func TestValidateAccessToken_Success(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	token, _, err := manager.GenerateAccessToken("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	claims, err := manager.ValidateAccessToken(token)

	require.NoError(t, err)
	assert.Equal(t, "access", claims.TokenType)
}

func TestValidateAccessToken_RefreshTokenFails(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	token, _, err := manager.GenerateRefreshToken("user123", "test@example.com", "authenticated", "session123", nil, nil)
	require.NoError(t, err)

	claims, err := manager.ValidateAccessToken(token)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
	assert.Nil(t, claims)
}

func TestValidateRefreshToken_Success(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	token, _, err := manager.GenerateRefreshToken("user123", "test@example.com", "authenticated", "session123", nil, nil)
	require.NoError(t, err)

	claims, err := manager.ValidateRefreshToken(token)

	require.NoError(t, err)
	assert.Equal(t, "refresh", claims.TokenType)
}

func TestValidateRefreshToken_AccessTokenFails(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	token, _, err := manager.GenerateAccessToken("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	claims, err := manager.ValidateRefreshToken(token)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
	assert.Nil(t, claims)
}

func TestRefreshAccessToken_Success(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	_, refreshToken, sessionID, err := manager.GenerateTokenPair("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	newAccessToken, err := manager.RefreshAccessToken(refreshToken)

	require.NoError(t, err)
	assert.NotEmpty(t, newAccessToken)

	claims, err := manager.ValidateAccessToken(newAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.NotEmpty(t, claims.SessionID)
	_ = sessionID
}

func TestRefreshAccessToken_InvalidRefreshToken(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	newToken, err := manager.RefreshAccessToken("invalid-token")

	assert.Error(t, err)
	assert.Empty(t, newToken)
}

func TestRefreshAccessToken_AccessTokenFails(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	accessToken, _, _, err := manager.GenerateTokenPair("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	newToken, err := manager.RefreshAccessToken(accessToken)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
	assert.Empty(t, newToken)
}

func TestExtractUserID_Success(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	userID := "user123"
	token, _, err := manager.GenerateAccessToken(userID, "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	extractedUserID, err := manager.ExtractUserID(token)

	require.NoError(t, err)
	assert.Equal(t, userID, extractedUserID)
}

func TestExtractUserID_InvalidToken(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	userID, err := manager.ExtractUserID("invalid-token")

	assert.Error(t, err)
	assert.Empty(t, userID)
}

func TestGetTokenExpiry_Success(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	token, claims, err := manager.GenerateAccessToken("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	expiry, err := manager.GetTokenExpiry(token)

	require.NoError(t, err)
	assert.Equal(t, claims.ExpiresAt.Time, expiry)
	assert.True(t, expiry.After(time.Now()))
}

func TestGetTokenExpiry_InvalidToken(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	expiry, err := manager.GetTokenExpiry("invalid-token")

	assert.Error(t, err)
	assert.True(t, expiry.IsZero())
}

func TestTokenClaims_StandardCompliance(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	token, claims, err := manager.GenerateAccessToken("user123", "test@example.com", "user", nil, nil)
	require.NoError(t, err)

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return manager.secretKey, nil
	})

	require.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.NotBefore)
	assert.NotEmpty(t, claims.ID)
	assert.NotEmpty(t, claims.Issuer)
	assert.NotEmpty(t, claims.Subject)
}

func TestConcurrentTokenGeneration(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

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

	tokens := make(map[string]bool)
	for i := 0; i < numGoroutines; i++ {
		token := <-results
		tokens[token] = true
	}

	assert.Len(t, tokens, numGoroutines)

	for token := range tokens {
		claims, err := manager.ValidateAccessToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user123", claims.UserID)
	}
}

func TestGenerateServiceRoleToken(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	token, err := manager.GenerateServiceRoleToken()

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := manager.ValidateServiceRoleToken(token)
	require.NoError(t, err)
	assert.Equal(t, "service_role", claims.Role)
	assert.Equal(t, "fluxbase", claims.Issuer)
	assert.Empty(t, claims.UserID)
}

func TestGenerateAnonToken(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	token, err := manager.GenerateAnonToken()

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := manager.ValidateServiceRoleToken(token)
	require.NoError(t, err)
	assert.Equal(t, "anon", claims.Role)
	assert.Equal(t, "fluxbase", claims.Issuer)
	assert.Empty(t, claims.UserID)
}

func TestValidateServiceRoleToken_Success(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

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
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	token, _, err := manager.GenerateAccessToken("user123", "test@example.com", "admin", nil, nil)
	require.NoError(t, err)

	claims, err := manager.ValidateServiceRoleToken(token)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
	assert.Nil(t, claims)
}

func TestValidateServiceRoleToken_WrongSecret(t *testing.T) {
	manager1, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)
	manager2, err := NewJWTManager("different-secret-key-must-be-32-chars!", 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	token, err := manager1.GenerateServiceRoleToken()
	require.NoError(t, err)

	claims, err := manager2.ValidateServiceRoleToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateServiceRoleToken_ValidIssuers(t *testing.T) {
	secret := "super-secret-jwt-token-with-at-least-32-characters-long"
	manager, err := NewJWTManager(secret, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	tests := []struct {
		name   string
		role   string
		issuer string
	}{
		{"fluxbase issuer with service_role", "service_role", "fluxbase"},
		{"fluxbase issuer with anon", "anon", "fluxbase"},
		{"empty issuer with service_role", "service_role", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			parsedClaims, err := manager.ValidateServiceRoleToken(tokenString)
			require.NoError(t, err)
			assert.Equal(t, tt.role, parsedClaims.Role)
		})
	}
}

func TestValidateServiceRoleToken_InvalidIssuer(t *testing.T) {
	secret := "test-secret-key-must-be-32-characters!"
	manager, err := NewJWTManager(secret, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

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
	secret := "test-secret-key-must-be-32-characters!"
	manager, err := NewJWTManager(secret, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	now := time.Now()
	claims := jwt.MapClaims{
		"role": "service_role",
		"iss":  "fluxbase",
		"iat":  now.Add(-2 * time.Hour).Unix(),
		"exp":  now.Add(-1 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	require.NoError(t, err)

	parsedClaims, err := manager.ValidateServiceRoleToken(tokenString)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrExpiredToken)
	assert.Nil(t, parsedClaims)
}

func TestGenerateAnonymousAccessToken(t *testing.T) {
	accessTTL := 15 * time.Minute
	manager, err := NewJWTManager(testSecretKey, accessTTL, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("generates valid anonymous access token", func(t *testing.T) {
		userID := "anon-user-123"
		tokenString, err := manager.GenerateAnonymousAccessToken(userID)

		require.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		claims, err := manager.ValidateToken(tokenString)
		require.NoError(t, err)

		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, "anon", claims.Role)
		assert.Equal(t, "access", claims.TokenType)
		assert.True(t, claims.IsAnonymous)
		assert.Empty(t, claims.Email)
		assert.Empty(t, claims.SessionID)
	})

	t.Run("token has correct expiry", func(t *testing.T) {
		userID := "anon-user-456"
		tokenString, err := manager.GenerateAnonymousAccessToken(userID)
		require.NoError(t, err)

		expiry, err := manager.GetTokenExpiry(tokenString)
		require.NoError(t, err)

		expectedExpiry := time.Now().Add(accessTTL)
		assert.WithinDuration(t, expectedExpiry, expiry, 5*time.Second)
	})

	t.Run("generates unique tokens for same user", func(t *testing.T) {
		userID := "anon-user-789"
		token1, err := manager.GenerateAnonymousAccessToken(userID)
		require.NoError(t, err)

		token2, err := manager.GenerateAnonymousAccessToken(userID)
		require.NoError(t, err)

		assert.NotEqual(t, token1, token2)
	})

	t.Run("empty user ID is allowed", func(t *testing.T) {
		tokenString, err := manager.GenerateAnonymousAccessToken("")
		require.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		claims, err := manager.ValidateToken(tokenString)
		require.NoError(t, err)
		assert.Empty(t, claims.UserID)
		assert.True(t, claims.IsAnonymous)
	})
}

func TestGenerateAnonymousRefreshToken(t *testing.T) {
	refreshTTL := 7 * 24 * time.Hour
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, refreshTTL)
	require.NoError(t, err)

	t.Run("generates valid anonymous refresh token", func(t *testing.T) {
		userID := "anon-user-123"
		tokenString, err := manager.GenerateAnonymousRefreshToken(userID)

		require.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		claims, err := manager.ValidateToken(tokenString)
		require.NoError(t, err)

		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, "anon", claims.Role)
		assert.Equal(t, "refresh", claims.TokenType)
		assert.True(t, claims.IsAnonymous)
		assert.Empty(t, claims.Email)
		assert.Empty(t, claims.SessionID)
	})

	t.Run("refresh token has longer expiry than access token", func(t *testing.T) {
		userID := "anon-user-456"

		accessToken, err := manager.GenerateAnonymousAccessToken(userID)
		require.NoError(t, err)

		refreshToken, err := manager.GenerateAnonymousRefreshToken(userID)
		require.NoError(t, err)

		accessExpiry, err := manager.GetTokenExpiry(accessToken)
		require.NoError(t, err)

		refreshExpiry, err := manager.GetTokenExpiry(refreshToken)
		require.NoError(t, err)

		assert.True(t, refreshExpiry.After(accessExpiry))
	})

	t.Run("generates unique tokens for same user", func(t *testing.T) {
		userID := "anon-user-789"
		token1, err := manager.GenerateAnonymousRefreshToken(userID)
		require.NoError(t, err)

		token2, err := manager.GenerateAnonymousRefreshToken(userID)
		require.NoError(t, err)

		assert.NotEqual(t, token1, token2)
	})
}

func TestAnonymousTokenValidation(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("anonymous access token passes ValidateAccessToken", func(t *testing.T) {
		userID := "anon-user"
		tokenString, err := manager.GenerateAnonymousAccessToken(userID)
		require.NoError(t, err)

		claims, err := manager.ValidateAccessToken(tokenString)
		require.NoError(t, err)
		assert.True(t, claims.IsAnonymous)
	})

	t.Run("anonymous refresh token passes ValidateRefreshToken", func(t *testing.T) {
		userID := "anon-user"
		tokenString, err := manager.GenerateAnonymousRefreshToken(userID)
		require.NoError(t, err)

		claims, err := manager.ValidateRefreshToken(tokenString)
		require.NoError(t, err)
		assert.True(t, claims.IsAnonymous)
	})

	t.Run("anonymous access token fails ValidateRefreshToken", func(t *testing.T) {
		userID := "anon-user"
		tokenString, err := manager.GenerateAnonymousAccessToken(userID)
		require.NoError(t, err)

		_, err = manager.ValidateRefreshToken(tokenString)
		assert.Error(t, err)
	})

	t.Run("anonymous refresh token fails ValidateAccessToken", func(t *testing.T) {
		userID := "anon-user"
		tokenString, err := manager.GenerateAnonymousRefreshToken(userID)
		require.NoError(t, err)

		_, err = manager.ValidateAccessToken(tokenString)
		assert.Error(t, err)
	})

	t.Run("wrong secret fails validation", func(t *testing.T) {
		tokenString, err := manager.GenerateAnonymousAccessToken("user")
		require.NoError(t, err)

		wrongManager, err := NewJWTManager("different-secret-key-must-be-32-chars!", 15*time.Minute, 7*24*time.Hour)
		require.NoError(t, err)
		_, err = wrongManager.ValidateToken(tokenString)
		assert.Error(t, err)
	})

	t.Run("ExtractUserID works for anonymous tokens", func(t *testing.T) {
		expectedUserID := "anon-user-extract"
		tokenString, err := manager.GenerateAnonymousAccessToken(expectedUserID)
		require.NoError(t, err)

		userID, err := manager.ExtractUserID(tokenString)
		require.NoError(t, err)
		assert.Equal(t, expectedUserID, userID)
	})
}

// =============================================================================
// Impersonation Token Security Tests
// =============================================================================

func TestGenerateAccessToken_WithImpersonation(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("token with impersonated_by claim", func(t *testing.T) {
		userID := "user123"
		email := "test@example.com"
		role := "authenticated"
		adminID := "admin-456"

		token, claims, err := manager.GenerateAccessToken(userID, email, role, nil, nil, WithImpersonatedBy(adminID))

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotNil(t, claims)

		// Verify standard claims
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
		assert.Equal(t, "access", claims.TokenType)

		// Verify impersonation claim
		assert.Equal(t, adminID, claims.ImpersonatedBy)
	})

	t.Run("token without impersonated_by defaults to empty", func(t *testing.T) {
		userID := "user123"
		email := "test@example.com"
		role := "authenticated"

		token, claims, err := manager.GenerateAccessToken(userID, email, role, nil, nil)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotNil(t, claims)

		// ImpersonatedBy should be empty (not set)
		assert.Empty(t, claims.ImpersonatedBy)
	})
}

func TestGenerateRefreshToken_WithImpersonation(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("refresh token with impersonated_by claim", func(t *testing.T) {
		userID := "user123"
		email := "test@example.com"
		role := "authenticated"
		sessionID := "session-abc"
		adminID := "admin-789"

		token, claims, err := manager.GenerateRefreshToken(userID, email, role, sessionID, nil, nil, WithImpersonatedBy(adminID))

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotNil(t, claims)

		// Verify standard claims
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
		assert.Equal(t, "refresh", claims.TokenType)
		assert.Equal(t, sessionID, claims.SessionID)

		// Verify impersonation claim
		assert.Equal(t, adminID, claims.ImpersonatedBy)
	})

	t.Run("refresh token without impersonated_by defaults to empty", func(t *testing.T) {
		userID := "user123"
		email := "test@example.com"
		role := "authenticated"
		sessionID := "session-xyz"

		token, claims, err := manager.GenerateRefreshToken(userID, email, role, sessionID, nil, nil)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotNil(t, claims)

		// ImpersonatedBy should be empty (not set)
		assert.Empty(t, claims.ImpersonatedBy)
	})
}

func TestValidateToken_ImpersonationBackwardCompatibility(t *testing.T) {
	manager, err := NewJWTManager(testSecretKey, 15*time.Minute, 7*24*time.Hour)
	require.NoError(t, err)

	t.Run("old tokens without impersonated_by claim still validate", func(t *testing.T) {
		userID := "user-retro"
		email := "retro@example.com"
		role := "authenticated"

		// Generate token without impersonation claim
		token, _, err := manager.GenerateAccessToken(userID, email, role, nil, nil)
		require.NoError(t, err)

		// Token should still validate
		claims, err := manager.ValidateToken(token)
		require.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, userID, claims.UserID)
		assert.Empty(t, claims.ImpersonatedBy, "Old tokens should have empty impersonated_by")
	})
}

func TestTokenClaims_ImpersonatedByField(t *testing.T) {
	t.Run("ImpersonatedBy field is present in claims struct", func(t *testing.T) {
		claims := TokenClaims{
			UserID:         "user-123",
			Email:          "user@example.com",
			Role:           "authenticated",
			ImpersonatedBy: "admin-456",
			TokenType:      "access",
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        "jti-123",
				Subject:   "user-123",
				Issuer:    "fluxbase",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				NotBefore: jwt.NewNumericDate(time.Now()),
			},
		}

		assert.Equal(t, "admin-456", claims.ImpersonatedBy)
	})

	t.Run("ImpersonatedBy defaults to empty string", func(t *testing.T) {
		claims := TokenClaims{
			UserID:    "user-789",
			Role:      "authenticated",
			TokenType: "access",
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        "jti-456",
				Subject:   "user-789",
				Issuer:    "fluxbase",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				NotBefore: jwt.NewNumericDate(time.Now()),
			},
		}

		assert.Empty(t, claims.ImpersonatedBy)
	})
}
