package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Error Variable Tests
// =============================================================================

func TestSAMLErrors_Defined(t *testing.T) {
	errors := []error{
		ErrSAMLProviderNotFound,
		ErrSAMLProviderDisabled,
		ErrSAMLMetadataFetchFailed,
		ErrSAMLMetadataParseFailed,
		ErrSAMLMetadataInsecureURL,
		ErrSAMLAssertionInvalid,
		ErrSAMLAssertionExpired,
		ErrSAMLAssertionReplayed,
		ErrSAMLAudienceMismatch,
		ErrSAMLMissingEmail,
		ErrSAMLUserCreationDisabled,
		ErrSAMLInvalidRelayState,
		ErrSAMLSLONotSupported,
		ErrSAMLSessionNotFound,
		ErrSAMLLogoutFailed,
		ErrSAMLInvalidLogoutRequest,
		ErrSAMLInvalidLogoutResponse,
		ErrSAMLSigningKeyMissing,
	}

	for _, err := range errors {
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	}
}

func TestSAMLErrors_Messages(t *testing.T) {
	tests := []struct {
		err     error
		message string
	}{
		{ErrSAMLProviderNotFound, "SAML provider not found"},
		{ErrSAMLProviderDisabled, "SAML provider is disabled"},
		{ErrSAMLMetadataFetchFailed, "failed to fetch IdP metadata"},
		{ErrSAMLMetadataParseFailed, "failed to parse IdP metadata"},
		{ErrSAMLMetadataInsecureURL, "SAML metadata URL must use HTTPS"},
		{ErrSAMLAssertionInvalid, "invalid SAML assertion"},
		{ErrSAMLAssertionExpired, "SAML assertion has expired"},
		{ErrSAMLAssertionReplayed, "SAML assertion has already been used"},
		{ErrSAMLAudienceMismatch, "SAML assertion audience does not match service provider"},
		{ErrSAMLMissingEmail, "email attribute not found in SAML assertion"},
		{ErrSAMLUserCreationDisabled, "automatic user creation is disabled for this provider"},
		{ErrSAMLInvalidRelayState, "invalid RelayState redirect URL"},
		{ErrSAMLSLONotSupported, "SAML SLO not supported by this provider"},
		{ErrSAMLSessionNotFound, "SAML session not found"},
		{ErrSAMLLogoutFailed, "SAML logout failed"},
		{ErrSAMLInvalidLogoutRequest, "invalid SAML LogoutRequest"},
		{ErrSAMLInvalidLogoutResponse, "invalid SAML LogoutResponse"},
		{ErrSAMLSigningKeyMissing, "SP signing key not configured"},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			assert.Equal(t, tt.message, tt.err.Error())
		})
	}
}

func TestSAMLErrors_Distinct(t *testing.T) {
	errors := []error{
		ErrSAMLProviderNotFound,
		ErrSAMLProviderDisabled,
		ErrSAMLMetadataFetchFailed,
		ErrSAMLMetadataParseFailed,
		ErrSAMLAssertionInvalid,
		ErrSAMLAssertionExpired,
		ErrSAMLAudienceMismatch,
	}

	for i, err1 := range errors {
		for j, err2 := range errors {
			if i != j {
				assert.NotEqual(t, err1, err2)
			}
		}
	}
}

// =============================================================================
// SAMLProvider Struct Tests
// =============================================================================

func TestSAMLProvider_Fields(t *testing.T) {
	now := time.Now()
	provider := SAMLProvider{
		ID:                       "provider-123",
		Name:                     "okta",
		Enabled:                  true,
		EntityID:                 "https://app.example.com/saml",
		AcsURL:                   "https://app.example.com/auth/saml/acs",
		SloURL:                   "https://app.example.com/auth/saml/slo",
		SsoURL:                   "https://idp.okta.com/sso",
		IdPSloURL:                "https://idp.okta.com/slo",
		Certificate:              "MIIC...",
		AttributeMapping:         map[string]string{"email": "email", "name": "displayName"},
		AutoCreateUsers:          true,
		DefaultRole:              "authenticated",
		AllowIDPInitiated:        false,
		AllowedRedirectHosts:     []string{"example.com", "app.example.com"},
		AllowInsecureMetadataURL: false,
		AllowDashboardLogin:      true,
		AllowAppLogin:            true,
		RequiredGroups:           []string{"users", "admins"},
		RequiredGroupsAll:        []string{"verified"},
		DeniedGroups:             []string{"blocked"},
		GroupAttribute:           "groups",
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	assert.Equal(t, "provider-123", provider.ID)
	assert.Equal(t, "okta", provider.Name)
	assert.True(t, provider.Enabled)
	assert.Equal(t, "https://app.example.com/saml", provider.EntityID)
	assert.True(t, provider.AllowDashboardLogin)
	assert.True(t, provider.AllowAppLogin)
	assert.Len(t, provider.RequiredGroups, 2)
	assert.Len(t, provider.AllowedRedirectHosts, 2)
}

func TestSAMLProvider_Defaults(t *testing.T) {
	provider := SAMLProvider{}

	assert.Empty(t, provider.ID)
	assert.Empty(t, provider.Name)
	assert.False(t, provider.Enabled)
	assert.Nil(t, provider.AttributeMapping)
	assert.Nil(t, provider.RequiredGroups)
	assert.Nil(t, provider.AllowedRedirectHosts)
}

// =============================================================================
// SAMLSession Struct Tests
// =============================================================================

func TestSAMLSession_Fields(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(8 * time.Hour)

	session := SAMLSession{
		ID:           "session-123",
		UserID:       "user-456",
		ProviderID:   "provider-789",
		ProviderName: "okta",
		NameID:       "user@example.com",
		NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
		SessionIndex: "_abc123",
		Attributes:   map[string]interface{}{"email": "user@example.com", "groups": []string{"users"}},
		ExpiresAt:    &expiresAt,
		CreatedAt:    now,
	}

	assert.Equal(t, "session-123", session.ID)
	assert.Equal(t, "user-456", session.UserID)
	assert.Equal(t, "okta", session.ProviderName)
	assert.Equal(t, "user@example.com", session.NameID)
	assert.NotNil(t, session.ExpiresAt)
	assert.NotNil(t, session.Attributes)
}

func TestSAMLSession_NullableFields(t *testing.T) {
	session := SAMLSession{
		ID:        "session-123",
		UserID:    "user-456",
		ExpiresAt: nil,
	}

	assert.Nil(t, session.ExpiresAt)
	assert.Nil(t, session.Attributes)
}

// =============================================================================
// SAMLAssertion Struct Tests
// =============================================================================

func TestSAMLAssertion_Fields(t *testing.T) {
	now := time.Now()
	assertion := SAMLAssertion{
		ID:           "_assertion-123",
		NameID:       "user@example.com",
		NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
		SessionIndex: "_session-idx",
		Attributes: map[string][]string{
			"email":  {"user@example.com"},
			"groups": {"admin", "users"},
		},
		IssueInstant: now,
		NotBefore:    now.Add(-5 * time.Minute),
		NotOnOrAfter: now.Add(5 * time.Minute),
	}

	assert.Equal(t, "_assertion-123", assertion.ID)
	assert.Equal(t, "user@example.com", assertion.NameID)
	assert.Contains(t, assertion.Attributes, "email")
	assert.Contains(t, assertion.Attributes, "groups")
	assert.Len(t, assertion.Attributes["groups"], 2)
}

// =============================================================================
// LogoutRequestResult Struct Tests
// =============================================================================

func TestLogoutRequestResult_Fields(t *testing.T) {
	result := LogoutRequestResult{
		RedirectURL: "https://idp.example.com/slo?SAMLRequest=...",
		RequestID:   "id-abc123",
		Binding:     "redirect",
	}

	assert.Contains(t, result.RedirectURL, "SAMLRequest")
	assert.Equal(t, "id-abc123", result.RequestID)
	assert.Equal(t, "redirect", result.Binding)
}

// =============================================================================
// ParsedLogoutRequest Struct Tests
// =============================================================================

func TestParsedLogoutRequest_Fields(t *testing.T) {
	req := ParsedLogoutRequest{
		ID:           "id-logout-123",
		NameID:       "user@example.com",
		NameIDFormat: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
		SessionIndex: "_session-abc",
		Issuer:       "https://idp.example.com",
		Destination:  "https://sp.example.com/slo",
		RelayState:   "/dashboard",
	}

	assert.Equal(t, "id-logout-123", req.ID)
	assert.Equal(t, "user@example.com", req.NameID)
	assert.Equal(t, "https://idp.example.com", req.Issuer)
}

// =============================================================================
// ParsedLogoutResponse Struct Tests
// =============================================================================

func TestParsedLogoutResponse_Fields(t *testing.T) {
	resp := ParsedLogoutResponse{
		InResponseTo:  "id-logout-123",
		Status:        "urn:oasis:names:tc:SAML:2.0:status:Success",
		StatusMessage: "Logout completed",
		Issuer:        "https://idp.example.com",
	}

	assert.Equal(t, "id-logout-123", resp.InResponseTo)
	assert.Contains(t, resp.Status, "Success")
	assert.Equal(t, "Logout completed", resp.StatusMessage)
}

// =============================================================================
// ValidateRelayState Tests
// =============================================================================

func TestValidateRelayState_EmptyString(t *testing.T) {
	result, err := ValidateRelayState("", nil)

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestValidateRelayState_RelativeURL(t *testing.T) {
	tests := []string{
		"/dashboard",
		"/auth/callback",
		"/settings/profile",
		"dashboard",
		"path/to/page",
	}

	for _, relayState := range tests {
		t.Run(relayState, func(t *testing.T) {
			result, err := ValidateRelayState(relayState, nil)

			require.NoError(t, err)
			assert.Equal(t, relayState, result)
		})
	}
}

func TestValidateRelayState_ProtocolRelativeBlocked(t *testing.T) {
	protocolRelative := "//evil.com/path"

	result, err := ValidateRelayState(protocolRelative, []string{"example.com"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "protocol-relative")
	assert.Empty(t, result)
}

func TestValidateRelayState_AbsoluteURLNoAllowedHosts(t *testing.T) {
	absoluteURL := "https://example.com/dashboard"

	result, err := ValidateRelayState(absoluteURL, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "allowed_redirect_hosts")
	assert.Empty(t, result)
}

func TestValidateRelayState_AbsoluteURLAllowed(t *testing.T) {
	tests := []struct {
		relayState   string
		allowedHosts []string
		shouldPass   bool
	}{
		{"https://example.com/dashboard", []string{"example.com"}, true},
		{"https://app.example.com/settings", []string{"example.com"}, true}, // subdomain
		{"https://evil.com/phishing", []string{"example.com"}, false},
		{"https://example.com.evil.com/path", []string{"example.com"}, false},
		{"https://api.app.example.com/data", []string{"app.example.com"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.relayState, func(t *testing.T) {
			result, err := ValidateRelayState(tt.relayState, tt.allowedHosts)

			if tt.shouldPass {
				require.NoError(t, err)
				assert.Equal(t, tt.relayState, result)
			} else {
				assert.Error(t, err)
				assert.Empty(t, result)
			}
		})
	}
}

func TestValidateRelayState_InvalidURL(t *testing.T) {
	invalidURL := "://invalid"

	result, err := ValidateRelayState(invalidURL, []string{"example.com"})

	assert.Error(t, err)
	assert.Empty(t, result)
}

// =============================================================================
// SanitizeSAMLAttribute Tests
// =============================================================================

func TestSanitizeSAMLAttribute_NormalInput(t *testing.T) {
	inputs := []string{
		"John Doe",
		"user@example.com",
		"Admin User",
		"Test User Name",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			result := SanitizeSAMLAttribute(input)
			assert.Equal(t, input, result)
		})
	}
}

func TestSanitizeSAMLAttribute_RemovesNullBytes(t *testing.T) {
	input := "John\x00Doe"
	result := SanitizeSAMLAttribute(input)

	assert.Equal(t, "JohnDoe", result)
	assert.NotContains(t, result, "\x00")
}

func TestSanitizeSAMLAttribute_RemovesControlCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"bell character", "Test\x07User", "TestUser"},
		{"backspace", "Test\x08User", "TestUser"},
		{"escape", "Test\x1bUser", "TestUser"},
		{"DEL character", "Test\x7fUser", "TestUser"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSAMLAttribute(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeSAMLAttribute_PreservesWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"tab", "Test\tUser", "Test\tUser"},
		{"newline", "Test\nUser", "Test\nUser"},
		{"carriage return", "Test\rUser", "Test\rUser"},
		{"space", "Test User", "Test User"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSAMLAttribute(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeSAMLAttribute_TruncatesLongStrings(t *testing.T) {
	// Create a string longer than 1024 characters
	longString := ""
	for i := 0; i < 2000; i++ {
		longString += "a"
	}

	result := SanitizeSAMLAttribute(longString)

	assert.Len(t, result, 1024)
}

func TestSanitizeSAMLAttribute_TrimsWhitespace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  John Doe  ", "John Doe"},
		{"\t\nJohn Doe\t\n", "John Doe"},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeSAMLAttribute(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeSAMLAttribute_PreservesUnicode(t *testing.T) {
	unicodeStrings := []string{
		"日本語名前",
		"Müller",
		"Москва",
		"北京",
		"José García",
	}

	for _, input := range unicodeStrings {
		t.Run(input, func(t *testing.T) {
			result := SanitizeSAMLAttribute(input)
			assert.Equal(t, input, result)
		})
	}
}

func TestSanitizeSAMLAttribute_EmptyString(t *testing.T) {
	result := SanitizeSAMLAttribute("")
	assert.Empty(t, result)
}

// =============================================================================
// ValidateGroupMembership Tests
// =============================================================================

func TestValidateGroupMembership_NoRestrictions(t *testing.T) {
	provider := &SAMLProvider{
		RequiredGroups:    nil,
		RequiredGroupsAll: nil,
		DeniedGroups:      nil,
	}

	err := (&SAMLService{}).ValidateGroupMembership(provider, []string{"users", "admins"})

	assert.NoError(t, err)
}

func TestValidateGroupMembership_DeniedGroupsBlocks(t *testing.T) {
	provider := &SAMLProvider{
		DeniedGroups: []string{"blocked", "banned"},
	}

	tests := []struct {
		groups      []string
		shouldBlock bool
	}{
		{[]string{"users", "admins"}, false},
		{[]string{"users", "blocked"}, true},
		{[]string{"banned"}, true},
		{[]string{}, false},
	}

	for _, tt := range tests {
		err := (&SAMLService{}).ValidateGroupMembership(provider, tt.groups)

		if tt.shouldBlock {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "access denied")
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestValidateGroupMembership_RequiredGroupsAny(t *testing.T) {
	provider := &SAMLProvider{
		RequiredGroups: []string{"editors", "admins"},
	}

	tests := []struct {
		groups     []string
		shouldPass bool
	}{
		{[]string{"users", "editors"}, true},
		{[]string{"admins"}, true},
		{[]string{"users", "viewers"}, false},
		{[]string{}, false},
	}

	for _, tt := range tests {
		err := (&SAMLService{}).ValidateGroupMembership(provider, tt.groups)

		if tt.shouldPass {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "must be member of one of")
		}
	}
}

func TestValidateGroupMembership_RequiredGroupsAll(t *testing.T) {
	provider := &SAMLProvider{
		RequiredGroupsAll: []string{"verified", "active"},
	}

	tests := []struct {
		groups     []string
		shouldPass bool
	}{
		{[]string{"verified", "active", "users"}, true},
		{[]string{"verified", "active"}, true},
		{[]string{"verified"}, false},
		{[]string{"active"}, false},
		{[]string{"users"}, false},
		{[]string{}, false},
	}

	for _, tt := range tests {
		err := (&SAMLService{}).ValidateGroupMembership(provider, tt.groups)

		if tt.shouldPass {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "missing required group")
		}
	}
}

func TestValidateGroupMembership_CombinedRules(t *testing.T) {
	provider := &SAMLProvider{
		RequiredGroups:    []string{"editors", "admins"},
		RequiredGroupsAll: []string{"verified"},
		DeniedGroups:      []string{"blocked"},
	}

	tests := []struct {
		name       string
		groups     []string
		shouldPass bool
		errContain string
	}{
		{"all requirements met", []string{"verified", "admins"}, true, ""},
		{"missing verified", []string{"admins"}, false, "missing required group"},
		{"missing required group (any)", []string{"verified", "users"}, false, "must be member of one of"},
		{"in denied group", []string{"verified", "admins", "blocked"}, false, "access denied"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (&SAMLService{}).ValidateGroupMembership(provider, tt.groups)

			if tt.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			}
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkValidateRelayState_RelativeURL(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ValidateRelayState("/dashboard/settings", nil)
	}
}

func BenchmarkValidateRelayState_AbsoluteURL(b *testing.B) {
	allowedHosts := []string{"example.com", "app.example.com"}

	for i := 0; i < b.N; i++ {
		_, _ = ValidateRelayState("https://app.example.com/dashboard", allowedHosts)
	}
}

func BenchmarkSanitizeSAMLAttribute_Short(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = SanitizeSAMLAttribute("John Doe")
	}
}

func BenchmarkSanitizeSAMLAttribute_WithControlChars(b *testing.B) {
	input := "John\x00\x07\x1bDoe"

	for i := 0; i < b.N; i++ {
		_ = SanitizeSAMLAttribute(input)
	}
}

func BenchmarkSanitizeSAMLAttribute_Long(b *testing.B) {
	longString := ""
	for i := 0; i < 2000; i++ {
		longString += "a"
	}

	for i := 0; i < b.N; i++ {
		_ = SanitizeSAMLAttribute(longString)
	}
}

func BenchmarkValidateGroupMembership_Simple(b *testing.B) {
	provider := &SAMLProvider{
		RequiredGroups: []string{"users", "admins"},
	}
	groups := []string{"users", "editors"}
	svc := &SAMLService{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = svc.ValidateGroupMembership(provider, groups)
	}
}

func BenchmarkValidateGroupMembership_Complex(b *testing.B) {
	provider := &SAMLProvider{
		RequiredGroups:    []string{"editors", "admins", "superusers"},
		RequiredGroupsAll: []string{"verified", "active"},
		DeniedGroups:      []string{"blocked", "banned", "suspended"},
	}
	groups := []string{"verified", "active", "users", "editors", "readers"}
	svc := &SAMLService{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = svc.ValidateGroupMembership(provider, groups)
	}
}

// =============================================================================
// SAMLProvider Login Target Tests
// =============================================================================

func TestSAMLProvider_LoginTargets(t *testing.T) {
	t.Run("dashboard only", func(t *testing.T) {
		provider := SAMLProvider{
			AllowDashboardLogin: true,
			AllowAppLogin:       false,
		}

		assert.True(t, provider.AllowDashboardLogin)
		assert.False(t, provider.AllowAppLogin)
	})

	t.Run("app only", func(t *testing.T) {
		provider := SAMLProvider{
			AllowDashboardLogin: false,
			AllowAppLogin:       true,
		}

		assert.False(t, provider.AllowDashboardLogin)
		assert.True(t, provider.AllowAppLogin)
	})

	t.Run("both", func(t *testing.T) {
		provider := SAMLProvider{
			AllowDashboardLogin: true,
			AllowAppLogin:       true,
		}

		assert.True(t, provider.AllowDashboardLogin)
		assert.True(t, provider.AllowAppLogin)
	})
}

// =============================================================================
// AttributeMapping Tests
// =============================================================================

func TestSAMLProvider_AttributeMapping(t *testing.T) {
	t.Run("standard mapping", func(t *testing.T) {
		provider := SAMLProvider{
			AttributeMapping: map[string]string{
				"email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
				"name":  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
			},
		}

		assert.Contains(t, provider.AttributeMapping, "email")
		assert.Contains(t, provider.AttributeMapping, "name")
	})

	t.Run("custom mapping", func(t *testing.T) {
		provider := SAMLProvider{
			AttributeMapping: map[string]string{
				"email":      "userEmail",
				"name":       "displayName",
				"department": "dept",
				"manager":    "managerEmail",
			},
		}

		assert.Len(t, provider.AttributeMapping, 4)
		assert.Equal(t, "userEmail", provider.AttributeMapping["email"])
	})

	t.Run("nil mapping", func(t *testing.T) {
		provider := SAMLProvider{
			AttributeMapping: nil,
		}

		assert.Nil(t, provider.AttributeMapping)
	})
}
