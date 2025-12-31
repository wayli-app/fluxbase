package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOAuthClaims_NoRules(t *testing.T) {
	provider := &OAuthProviderRBAC{
		Name:           "google",
		RequiredClaims: nil,
		DeniedClaims:   nil,
	}

	idTokenClaims := map[string]interface{}{
		"sub":   "user123",
		"email": "user@example.com",
		"roles": []string{"user", "editor"},
	}

	// No rules configured, should always pass
	err := ValidateOAuthClaims(provider, idTokenClaims)
	require.NoError(t, err)

	err = ValidateOAuthClaims(provider, map[string]interface{}{})
	require.NoError(t, err)
}

func TestValidateOAuthClaims_RequiredClaims_Present(t *testing.T) {
	provider := &OAuthProviderRBAC{
		Name: "google",
		RequiredClaims: map[string][]string{
			"roles": {"admin", "editor"},
		},
	}

	// User has at least one required claim value - should pass
	idTokenClaims := map[string]interface{}{
		"sub":   "user123",
		"email": "user@example.com",
		"roles": []string{"editor", "viewer"},
	}

	err := ValidateOAuthClaims(provider, idTokenClaims)
	require.NoError(t, err)

	// Single string claim value (not array)
	idTokenClaims = map[string]interface{}{
		"sub":   "user123",
		"email": "user@example.com",
		"roles": "admin",
	}

	err = ValidateOAuthClaims(provider, idTokenClaims)
	require.NoError(t, err)
}

func TestValidateOAuthClaims_RequiredClaims_Missing(t *testing.T) {
	provider := &OAuthProviderRBAC{
		Name: "google",
		RequiredClaims: map[string][]string{
			"roles": {"admin"},
		},
	}

	// Claim not present at all
	idTokenClaims := map[string]interface{}{
		"sub":   "user123",
		"email": "user@example.com",
	}

	err := ValidateOAuthClaims(provider, idTokenClaims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required claim")
	assert.Contains(t, err.Error(), "roles")
}

func TestValidateOAuthClaims_RequiredClaims_WrongValue(t *testing.T) {
	provider := &OAuthProviderRBAC{
		Name: "google",
		RequiredClaims: map[string][]string{
			"roles": {"admin", "superuser"},
		},
	}

	// Claim present but with wrong value
	idTokenClaims := map[string]interface{}{
		"sub":   "user123",
		"email": "user@example.com",
		"roles": []string{"viewer", "editor"},
	}

	err := ValidateOAuthClaims(provider, idTokenClaims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
	assert.Contains(t, err.Error(), "must have one of")
}

func TestValidateOAuthClaims_DeniedClaims(t *testing.T) {
	provider := &OAuthProviderRBAC{
		Name: "google",
		DeniedClaims: map[string][]string{
			"status": {"suspended", "blocked"},
		},
	}

	// User not in denied claims - should pass
	idTokenClaims := map[string]interface{}{
		"sub":    "user123",
		"email":  "user@example.com",
		"status": "active",
	}

	err := ValidateOAuthClaims(provider, idTokenClaims)
	require.NoError(t, err)

	// Claim not present - should pass
	idTokenClaims = map[string]interface{}{
		"sub":   "user123",
		"email": "user@example.com",
	}

	err = ValidateOAuthClaims(provider, idTokenClaims)
	require.NoError(t, err)

	// User in denied claim - should fail
	idTokenClaims = map[string]interface{}{
		"sub":    "user123",
		"email":  "user@example.com",
		"status": "suspended",
	}

	err = ValidateOAuthClaims(provider, idTokenClaims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "restricted value")
	assert.Contains(t, err.Error(), "suspended")
}

func TestValidateOAuthClaims_DeniedClaimsTakePrecedence(t *testing.T) {
	provider := &OAuthProviderRBAC{
		Name: "google",
		RequiredClaims: map[string][]string{
			"roles": {"admin"},
		},
		DeniedClaims: map[string][]string{
			"status": {"suspended"},
		},
	}

	// User has required claim but also in denied claim - denied takes precedence
	idTokenClaims := map[string]interface{}{
		"sub":    "user123",
		"email":  "user@example.com",
		"roles":  "admin",
		"status": "suspended",
	}

	err := ValidateOAuthClaims(provider, idTokenClaims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "restricted value")
	assert.Contains(t, err.Error(), "suspended")
}

func TestValidateOAuthClaims_MultipleClaims(t *testing.T) {
	provider := &OAuthProviderRBAC{
		Name: "google",
		RequiredClaims: map[string][]string{
			"roles":      {"admin", "editor"},
			"department": {"IT", "Engineering"},
		},
		DeniedClaims: map[string][]string{
			"status": {"suspended", "inactive"},
		},
	}

	// Valid: has required claims, not in denied
	idTokenClaims := map[string]interface{}{
		"sub":        "user123",
		"email":      "user@example.com",
		"roles":      []string{"editor"},
		"department": "Engineering",
		"status":     "active",
	}

	err := ValidateOAuthClaims(provider, idTokenClaims)
	require.NoError(t, err)

	// Invalid: missing one required claim
	idTokenClaims = map[string]interface{}{
		"sub":        "user123",
		"email":      "user@example.com",
		"department": "Engineering",
		"status":     "active",
	}

	err = ValidateOAuthClaims(provider, idTokenClaims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required claim")

	// Invalid: has required claims but wrong department value
	idTokenClaims = map[string]interface{}{
		"sub":        "user123",
		"email":      "user@example.com",
		"roles":      "admin",
		"department": "Sales",
		"status":     "active",
	}

	err = ValidateOAuthClaims(provider, idTokenClaims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must have one of")
}

func TestValidateOAuthClaims_ArrayClaims(t *testing.T) {
	provider := &OAuthProviderRBAC{
		Name: "google",
		RequiredClaims: map[string][]string{
			"groups": {"FluxbaseAdmins"},
		},
	}

	// Claim is array of strings
	idTokenClaims := map[string]interface{}{
		"sub":    "user123",
		"email":  "user@example.com",
		"groups": []string{"AllUsers", "FluxbaseAdmins", "Developers"},
	}

	err := ValidateOAuthClaims(provider, idTokenClaims)
	require.NoError(t, err)

	// Claim is array but doesn't contain required value
	idTokenClaims = map[string]interface{}{
		"sub":    "user123",
		"email":  "user@example.com",
		"groups": []string{"AllUsers", "Developers"},
	}

	err = ValidateOAuthClaims(provider, idTokenClaims)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must have one of")
}

func TestValidateOAuthClaims_InterfaceArrayClaims(t *testing.T) {
	provider := &OAuthProviderRBAC{
		Name: "azure_ad",
		RequiredClaims: map[string][]string{
			"groups": {"admin-group-uuid"},
		},
	}

	// Claim is []interface{} (common from JSON unmarshaling)
	idTokenClaims := map[string]interface{}{
		"sub":   "user123",
		"email": "user@example.com",
		"groups": []interface{}{
			"user-group-uuid",
			"admin-group-uuid",
			"editor-group-uuid",
		},
	}

	err := ValidateOAuthClaims(provider, idTokenClaims)
	require.NoError(t, err)
}

func TestValidateOAuthClaims_CaseSensitivity(t *testing.T) {
	provider := &OAuthProviderRBAC{
		Name: "google",
		RequiredClaims: map[string][]string{
			"roles": {"Admin"},
		},
	}

	// Claim values are case-sensitive
	idTokenClaims := map[string]interface{}{
		"sub":   "user123",
		"email": "user@example.com",
		"roles": "Admin",
	}

	err := ValidateOAuthClaims(provider, idTokenClaims)
	require.NoError(t, err)

	idTokenClaims = map[string]interface{}{
		"sub":   "user123",
		"email": "user@example.com",
		"roles": "admin",
	}

	err = ValidateOAuthClaims(provider, idTokenClaims)
	require.Error(t, err)
}

func TestValidateOAuthClaims_NonStringClaimValue(t *testing.T) {
	provider := &OAuthProviderRBAC{
		Name: "custom",
		RequiredClaims: map[string][]string{
			"level": {"5"},
		},
	}

	// Claim is a number (will be converted to string)
	idTokenClaims := map[string]interface{}{
		"sub":   "user123",
		"email": "user@example.com",
		"level": 5,
	}

	err := ValidateOAuthClaims(provider, idTokenClaims)
	require.NoError(t, err)
}

func TestNormalizeClaimToStringArray_String(t *testing.T) {
	result := normalizeClaimToStringArray("single-value")
	assert.Equal(t, []string{"single-value"}, result)
}

func TestNormalizeClaimToStringArray_StringSlice(t *testing.T) {
	result := normalizeClaimToStringArray([]string{"val1", "val2", "val3"})
	assert.Equal(t, []string{"val1", "val2", "val3"}, result)
}

func TestNormalizeClaimToStringArray_InterfaceSlice(t *testing.T) {
	result := normalizeClaimToStringArray([]interface{}{"val1", "val2", "val3"})
	assert.Equal(t, []string{"val1", "val2", "val3"}, result)
}

func TestNormalizeClaimToStringArray_MixedInterfaceSlice(t *testing.T) {
	// Only string items are included
	result := normalizeClaimToStringArray([]interface{}{"val1", 123, "val2", true})
	assert.Equal(t, []string{"val1", "val2"}, result)
}

func TestNormalizeClaimToStringArray_Number(t *testing.T) {
	result := normalizeClaimToStringArray(42)
	assert.Equal(t, []string{"42"}, result)
}

func TestNormalizeClaimToStringArray_Boolean(t *testing.T) {
	result := normalizeClaimToStringArray(true)
	assert.Equal(t, []string{"true"}, result)
}
