package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit tests for validatePlatformPasswordLength, the length-only policy
// extracted from DashboardAuthService.ChangePassword and ResetPassword
// (issue #313 option C remainder). Previously the only "validation test"
// (TestDashboardAuthService_ChangePassword_Validation) compared len() values
// directly and never invoked the real policy — so the bounds and error
// messages had zero coverage.

func TestValidatePlatformPasswordLength(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		password string
		wantErr  string // substring; "" = no error
	}{
		{"empty rejected", "", "at least"},
		{"just under min", strings.Repeat("a", MinPasswordLength-1), "at least"},
		{"exactly min accepted", strings.Repeat("a", MinPasswordLength), ""},
		{"typical strong accepted", "ValidPassword123!", ""},
		{"exactly max accepted", strings.Repeat("a", MaxPasswordLength), ""},
		{"over max rejected", strings.Repeat("a", MaxPasswordLength+1), "at most"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validatePlatformPasswordLength(tt.password)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidatePlatformPasswordLength_LimitsAreConsistent(t *testing.T) {
	t.Parallel()
	// Lock in the documented policy bounds so a drift (e.g. raising min without
	// updating callers) is caught here rather than only via integration tests.
	assert.Equal(t, 12, MinPasswordLength, "MinPasswordLength documented as 12")
	assert.Equal(t, 72, MaxPasswordLength, "MaxPasswordLength is the bcrypt limit")
	assert.Less(t, MinPasswordLength, MaxPasswordLength, "min must be below max")
}

func TestValidatePlatformPasswordLength_NoComplexityRules(t *testing.T) {
	t.Parallel()
	// CRITICAL behavior pin: platform password validation is LENGTH-ONLY. Unlike
	// PasswordHasher.ValidatePassword, it must NOT require upper/lower/digit/
	// symbol characters. A lowercase-only password of sufficient length passes.
	// This test exists so that a future "consolidation" onto ValidatePassword
	// (which would add complexity rules) is caught as a behavior change, not
	// silently shipped.
	err := validatePlatformPasswordLength(strings.Repeat("a", MinPasswordLength))
	assert.NoError(t, err, "length-only policy: all-lowercase password must pass")
}
