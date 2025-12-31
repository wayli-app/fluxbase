package auth

import (
	"fmt"
)

// OAuthProviderRBAC represents OAuth provider configuration with RBAC fields
type OAuthProviderRBAC struct {
	Name           string
	RequiredClaims map[string][]string
	DeniedClaims   map[string][]string
}

// ValidateOAuthClaims validates user's OAuth ID token claims against provider's RBAC rules
func ValidateOAuthClaims(provider *OAuthProviderRBAC, idTokenClaims map[string]interface{}) error {
	// Check denied claims first (highest priority)
	for claimName, deniedValues := range provider.DeniedClaims {
		userClaimValue, exists := idTokenClaims[claimName]
		if !exists {
			continue
		}

		// Normalize claim value to string array
		userValues := normalizeClaimToStringArray(userClaimValue)
		for _, deniedVal := range deniedValues {
			for _, userVal := range userValues {
				if userVal == deniedVal {
					return fmt.Errorf("access denied: claim '%s' has restricted value '%s'", claimName, deniedVal)
				}
			}
		}
	}

	// Check required claims (must have at least one matching value)
	for claimName, requiredValues := range provider.RequiredClaims {
		userClaimValue, exists := idTokenClaims[claimName]
		if !exists {
			return fmt.Errorf("access denied: missing required claim '%s'", claimName)
		}

		userValues := normalizeClaimToStringArray(userClaimValue)
		hasRequiredValue := false
		for _, requiredVal := range requiredValues {
			for _, userVal := range userValues {
				if userVal == requiredVal {
					hasRequiredValue = true
					break
				}
			}
			if hasRequiredValue {
				break
			}
		}
		if !hasRequiredValue {
			return fmt.Errorf("access denied: claim '%s' must have one of: %v", claimName, requiredValues)
		}
	}

	return nil
}

// normalizeClaimToStringArray converts various claim value types to []string
// Handles: string, []string, []interface{} containing strings
func normalizeClaimToStringArray(value interface{}) []string {
	switch v := value.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		// Try to convert to string as fallback
		return []string{fmt.Sprintf("%v", v)}
	}
}
