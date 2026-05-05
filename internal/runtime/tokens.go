package runtime

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// generateUserToken generates a JWT token for the execution's user context
// This token respects RLS policies based on the user who triggered the execution
func generateUserToken(jwtSecret string, req ExecutionRequest, runtimeType RuntimeType, timeout time.Duration) (string, error) {
	if jwtSecret == "" {
		return "", fmt.Errorf("JWT secret not configured")
	}

	now := time.Now()

	// Build claims matching the auth.TokenClaims format
	claims := jwt.MapClaims{
		"iss":        "fluxbase",
		"iat":        now.Unix(),
		"exp":        now.Add(timeout).Unix(),
		"nbf":        now.Unix(),
		"jti":        uuid.New().String(),
		"token_type": "access",
	}

	// Add execution ID for audit trail
	switch runtimeType {
	case RuntimeTypeFunction:
		claims["execution_id"] = req.ID.String()
	case RuntimeTypeJob:
		claims["job_id"] = req.ID.String()
	}

	// Add user context if available
	if req.UserID != "" {
		claims["sub"] = req.UserID
		claims["user_id"] = req.UserID
	}
	if req.UserEmail != "" {
		claims["email"] = req.UserEmail
	}
	if req.UserRole != "" {
		claims["role"] = req.UserRole
	} else {
		claims["role"] = "authenticated"
	}

	// Add multi-tenancy context
	if req.TenantID != "" {
		claims["tenant_id"] = req.TenantID
	}
	if req.TenantRole != "" {
		claims["tenant_role"] = req.TenantRole
	}
	if req.IsInstanceAdmin {
		claims["is_instance_admin"] = true
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// generateServiceToken generates a JWT token scoped to the execution's tenant.
// Uses tenant_service role which enforces RLS, ensuring tenant isolation.
// The TenantID is required — executions without tenant context are rejected.
func generateServiceToken(jwtSecret string, req ExecutionRequest, runtimeType RuntimeType, timeout time.Duration) (string, error) {
	if jwtSecret == "" {
		return "", fmt.Errorf("JWT secret not configured")
	}

	if req.TenantID == "" {
		return "", fmt.Errorf("tenant context required for service token generation")
	}

	now := time.Now()

	claims := jwt.MapClaims{
		"iss":        "fluxbase",
		"sub":        "tenant_service",
		"role":       "tenant_service",
		"iat":        now.Unix(),
		"exp":        now.Add(timeout).Unix(),
		"nbf":        now.Unix(),
		"jti":        uuid.New().String(),
		"token_type": "access",
		"tenant_id":  req.TenantID,
	}

	switch runtimeType {
	case RuntimeTypeFunction:
		claims["execution_id"] = req.ID.String()
	case RuntimeTypeJob:
		claims["job_id"] = req.ID.String()
	}

	if req.IsInstanceAdmin {
		claims["is_instance_admin"] = true
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}
