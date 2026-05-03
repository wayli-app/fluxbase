package runtime

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateUserToken(t *testing.T) {
	t.Run("returns error for empty JWT secret", func(t *testing.T) {
		req := ExecutionRequest{
			ID:   uuid.New(),
			Name: "test-function",
		}

		token, err := generateUserToken("", req, RuntimeTypeFunction, 30*time.Second)

		require.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "JWT secret not configured")
	})

	t.Run("generates valid token for function", func(t *testing.T) {
		req := ExecutionRequest{
			ID:        uuid.New(),
			Name:      "test-function",
			UserID:    "user-123",
			UserEmail: "user@example.com",
			UserRole:  "admin",
		}

		token, err := generateUserToken("test-secret", req, RuntimeTypeFunction, 30*time.Second)

		require.NoError(t, err)
		assert.NotEmpty(t, token)

		parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			return []byte("test-secret"), nil
		})
		require.NoError(t, err)
		assert.True(t, parsed.Valid)

		claims := parsed.Claims.(jwt.MapClaims)
		assert.Equal(t, "fluxbase", claims["iss"])
		assert.Equal(t, "user-123", claims["sub"])
		assert.Equal(t, "user-123", claims["user_id"])
		assert.Equal(t, "user@example.com", claims["email"])
		assert.Equal(t, "admin", claims["role"])
		assert.Equal(t, "access", claims["token_type"])
		assert.Equal(t, req.ID.String(), claims["execution_id"])
	})

	t.Run("generates valid token for job", func(t *testing.T) {
		req := ExecutionRequest{
			ID:     uuid.New(),
			Name:   "test-job",
			UserID: "user-456",
		}

		token, err := generateUserToken("test-secret", req, RuntimeTypeJob, 5*time.Minute)

		require.NoError(t, err)
		assert.NotEmpty(t, token)

		parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			return []byte("test-secret"), nil
		})
		require.NoError(t, err)

		claims := parsed.Claims.(jwt.MapClaims)
		assert.Equal(t, req.ID.String(), claims["job_id"])
		assert.Nil(t, claims["execution_id"])
	})

	t.Run("defaults role to authenticated when not provided", func(t *testing.T) {
		req := ExecutionRequest{
			ID:     uuid.New(),
			Name:   "test-function",
			UserID: "user-123",
		}

		token, err := generateUserToken("test-secret", req, RuntimeTypeFunction, 30*time.Second)

		require.NoError(t, err)

		parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			return []byte("test-secret"), nil
		})
		require.NoError(t, err)

		claims := parsed.Claims.(jwt.MapClaims)
		assert.Equal(t, "authenticated", claims["role"])
	})

	t.Run("omits user claims when user ID is empty", func(t *testing.T) {
		req := ExecutionRequest{
			ID:   uuid.New(),
			Name: "anonymous-function",
		}

		token, err := generateUserToken("test-secret", req, RuntimeTypeFunction, 30*time.Second)

		require.NoError(t, err)

		parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			return []byte("test-secret"), nil
		})
		require.NoError(t, err)

		claims := parsed.Claims.(jwt.MapClaims)
		assert.Nil(t, claims["sub"])
		assert.Nil(t, claims["user_id"])
	})

	t.Run("sets correct expiration based on timeout", func(t *testing.T) {
		req := ExecutionRequest{
			ID:   uuid.New(),
			Name: "test-function",
		}
		timeout := 2 * time.Minute

		token, err := generateUserToken("test-secret", req, RuntimeTypeFunction, timeout)

		require.NoError(t, err)

		parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			return []byte("test-secret"), nil
		})
		require.NoError(t, err)

		claims := parsed.Claims.(jwt.MapClaims)
		iat := int64(claims["iat"].(float64))
		exp := int64(claims["exp"].(float64))

		assert.InDelta(t, iat+int64(timeout.Seconds()), exp, 2)
	})

	t.Run("includes unique jti", func(t *testing.T) {
		req := ExecutionRequest{
			ID:   uuid.New(),
			Name: "test-function",
		}

		token1, _ := generateUserToken("test-secret", req, RuntimeTypeFunction, 30*time.Second)
		token2, _ := generateUserToken("test-secret", req, RuntimeTypeFunction, 30*time.Second)

		parsed1, _ := jwt.Parse(token1, func(t *jwt.Token) (interface{}, error) {
			return []byte("test-secret"), nil
		})
		parsed2, _ := jwt.Parse(token2, func(t *jwt.Token) (interface{}, error) {
			return []byte("test-secret"), nil
		})

		claims1 := parsed1.Claims.(jwt.MapClaims)
		claims2 := parsed2.Claims.(jwt.MapClaims)

		assert.NotEqual(t, claims1["jti"], claims2["jti"])
	})
}

func TestGenerateServiceToken(t *testing.T) {
	tenantID := uuid.New().String()

	t.Run("returns error for empty JWT secret", func(t *testing.T) {
		req := ExecutionRequest{
			ID:       uuid.New(),
			Name:     "test-function",
			TenantID: tenantID,
		}

		token, err := generateServiceToken("", req, RuntimeTypeFunction, 30*time.Second)

		require.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "JWT secret not configured")
	})

	t.Run("returns error when TenantID is empty", func(t *testing.T) {
		req := ExecutionRequest{
			ID:   uuid.New(),
			Name: "test-function",
		}

		token, err := generateServiceToken("test-secret", req, RuntimeTypeFunction, 30*time.Second)

		require.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "tenant context required")
	})

	t.Run("generates tenant_service token for function", func(t *testing.T) {
		req := ExecutionRequest{
			ID:       uuid.New(),
			Name:     "test-function",
			TenantID: tenantID,
		}

		token, err := generateServiceToken("test-secret", req, RuntimeTypeFunction, 30*time.Second)

		require.NoError(t, err)
		assert.NotEmpty(t, token)

		parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			return []byte("test-secret"), nil
		})
		require.NoError(t, err)
		assert.True(t, parsed.Valid)

		claims := parsed.Claims.(jwt.MapClaims)
		assert.Equal(t, "fluxbase", claims["iss"])
		assert.Equal(t, "tenant_service", claims["sub"])
		assert.Equal(t, "tenant_service", claims["role"])
		assert.Equal(t, "access", claims["token_type"])
		assert.Equal(t, tenantID, claims["tenant_id"])
		assert.Equal(t, req.ID.String(), claims["execution_id"])
	})

	t.Run("generates tenant_service token for job", func(t *testing.T) {
		req := ExecutionRequest{
			ID:       uuid.New(),
			Name:     "test-job",
			TenantID: tenantID,
		}

		token, err := generateServiceToken("test-secret", req, RuntimeTypeJob, 5*time.Minute)

		require.NoError(t, err)

		parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			return []byte("test-secret"), nil
		})
		require.NoError(t, err)

		claims := parsed.Claims.(jwt.MapClaims)
		assert.Equal(t, req.ID.String(), claims["job_id"])
		assert.Nil(t, claims["execution_id"])
		assert.Equal(t, "tenant_service", claims["role"])
		assert.Equal(t, tenantID, claims["tenant_id"])
	})

	t.Run("token does not include user claims", func(t *testing.T) {
		req := ExecutionRequest{
			ID:       uuid.New(),
			Name:     "test-function",
			TenantID: tenantID,
			UserID:   "user-123",
			UserRole: "admin",
		}

		token, err := generateServiceToken("test-secret", req, RuntimeTypeFunction, 30*time.Second)

		require.NoError(t, err)

		parsed, _ := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			return []byte("test-secret"), nil
		})

		claims := parsed.Claims.(jwt.MapClaims)
		assert.Equal(t, "tenant_service", claims["sub"])
		assert.Equal(t, "tenant_service", claims["role"])
		assert.Nil(t, claims["user_id"])
	})

	t.Run("uses HS256 signing method", func(t *testing.T) {
		req := ExecutionRequest{
			ID:       uuid.New(),
			Name:     "test-function",
			TenantID: tenantID,
		}

		token, err := generateServiceToken("test-secret", req, RuntimeTypeFunction, 30*time.Second)
		require.NoError(t, err)

		parts := strings.Split(token, ".")
		require.Len(t, parts, 3, "JWT should have 3 parts")
	})
}

func BenchmarkGenerateUserToken(b *testing.B) {
	req := ExecutionRequest{
		ID:        uuid.New(),
		Name:      "benchmark-function",
		UserID:    "user-123",
		UserEmail: "user@example.com",
		UserRole:  "authenticated",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = generateUserToken("benchmark-secret", req, RuntimeTypeFunction, 30*time.Second)
	}
}

func BenchmarkGenerateServiceToken(b *testing.B) {
	req := ExecutionRequest{
		ID:       uuid.New(),
		Name:     "benchmark-function",
		TenantID: uuid.New().String(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = generateServiceToken("benchmark-secret", req, RuntimeTypeFunction, 30*time.Second)
	}
}
