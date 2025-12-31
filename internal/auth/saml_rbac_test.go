package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateGroupMembership_NoRules(t *testing.T) {
	// Create a service with empty configuration - we're only testing validation logic
	service := &SAMLService{
		providers: make(map[string]*SAMLProvider),
	}

	provider := &SAMLProvider{
		Name:              "test-provider",
		RequiredGroups:    nil,
		RequiredGroupsAll: nil,
		DeniedGroups:      nil,
	}

	// No rules configured, should always pass
	err := service.ValidateGroupMembership(provider, []string{"group1", "group2"})
	require.NoError(t, err)

	err = service.ValidateGroupMembership(provider, []string{})
	require.NoError(t, err)

	err = service.ValidateGroupMembership(provider, nil)
	require.NoError(t, err)
}

func TestValidateGroupMembership_RequiredGroups_OR_Logic(t *testing.T) {
	service := &SAMLService{
		providers: make(map[string]*SAMLProvider),
	}

	provider := &SAMLProvider{
		Name:           "test-provider",
		RequiredGroups: []string{"admins", "editors"},
	}

	// User has at least one required group - should pass
	err := service.ValidateGroupMembership(provider, []string{"admins", "viewers"})
	require.NoError(t, err)

	err = service.ValidateGroupMembership(provider, []string{"editors"})
	require.NoError(t, err)

	err = service.ValidateGroupMembership(provider, []string{"admins", "editors"})
	require.NoError(t, err)

	// User has none of the required groups - should fail
	err = service.ValidateGroupMembership(provider, []string{"viewers", "guests"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be member of one of")

	err = service.ValidateGroupMembership(provider, []string{})
	require.Error(t, err)

	err = service.ValidateGroupMembership(provider, nil)
	require.Error(t, err)
}

func TestValidateGroupMembership_RequiredGroupsAll_AND_Logic(t *testing.T) {
	service := &SAMLService{
		providers: make(map[string]*SAMLProvider),
	}

	provider := &SAMLProvider{
		Name:              "test-provider",
		RequiredGroupsAll: []string{"admins", "verified"},
	}

	// User has all required groups - should pass
	err := service.ValidateGroupMembership(provider, []string{"admins", "verified", "editors"})
	require.NoError(t, err)

	err = service.ValidateGroupMembership(provider, []string{"admins", "verified"})
	require.NoError(t, err)

	// User missing one required group - should fail
	err = service.ValidateGroupMembership(provider, []string{"admins"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required group")

	err = service.ValidateGroupMembership(provider, []string{"verified"})
	require.Error(t, err)

	err = service.ValidateGroupMembership(provider, []string{"other-group"})
	require.Error(t, err)

	err = service.ValidateGroupMembership(provider, []string{})
	require.Error(t, err)
}

func TestValidateGroupMembership_DeniedGroups(t *testing.T) {
	service := &SAMLService{
		providers: make(map[string]*SAMLProvider),
	}

	provider := &SAMLProvider{
		Name:         "test-provider",
		DeniedGroups: []string{"contractors", "guests"},
	}

	// User not in denied groups - should pass
	err := service.ValidateGroupMembership(provider, []string{"admins", "editors"})
	require.NoError(t, err)

	err = service.ValidateGroupMembership(provider, []string{})
	require.NoError(t, err)

	// User in a denied group - should fail
	err = service.ValidateGroupMembership(provider, []string{"admins", "contractors"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "restricted group")
	assert.Contains(t, err.Error(), "contractors")

	err = service.ValidateGroupMembership(provider, []string{"guests"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "guests")
}

func TestValidateGroupMembership_DeniedGroupsTakePrecedence(t *testing.T) {
	service := &SAMLService{
		providers: make(map[string]*SAMLProvider),
	}

	provider := &SAMLProvider{
		Name:           "test-provider",
		RequiredGroups: []string{"admins"},
		DeniedGroups:   []string{"contractors"},
	}

	// User has required group but also in denied group - denied takes precedence
	err := service.ValidateGroupMembership(provider, []string{"admins", "contractors"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "restricted group")
	assert.Contains(t, err.Error(), "contractors")
}

func TestValidateGroupMembership_CombinedRules(t *testing.T) {
	service := &SAMLService{
		providers: make(map[string]*SAMLProvider),
	}

	provider := &SAMLProvider{
		Name:              "test-provider",
		RequiredGroups:    []string{"admins", "editors"},
		RequiredGroupsAll: []string{"verified", "active"},
		DeniedGroups:      []string{"suspended"},
	}

	// Valid: has at least one from RequiredGroups, all from RequiredGroupsAll, none from DeniedGroups
	err := service.ValidateGroupMembership(provider, []string{"admins", "verified", "active"})
	require.NoError(t, err)

	err = service.ValidateGroupMembership(provider, []string{"editors", "verified", "active", "other"})
	require.NoError(t, err)

	// Invalid: missing from RequiredGroupsAll
	err = service.ValidateGroupMembership(provider, []string{"admins", "verified"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required group")

	// Invalid: missing from RequiredGroups
	err = service.ValidateGroupMembership(provider, []string{"verified", "active"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be member of one of")

	// Invalid: in denied group (takes precedence)
	err = service.ValidateGroupMembership(provider, []string{"admins", "verified", "active", "suspended"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "restricted group")
}

func TestValidateGroupMembership_CaseSensitivity(t *testing.T) {
	service := &SAMLService{
		providers: make(map[string]*SAMLProvider),
	}

	provider := &SAMLProvider{
		Name:           "test-provider",
		RequiredGroups: []string{"Admins"},
	}

	// Groups are case-sensitive
	err := service.ValidateGroupMembership(provider, []string{"Admins"})
	require.NoError(t, err)

	err = service.ValidateGroupMembership(provider, []string{"admins"})
	require.Error(t, err)

	err = service.ValidateGroupMembership(provider, []string{"ADMINS"})
	require.Error(t, err)
}

func TestExtractGroups_DefaultAttribute(t *testing.T) {
	service := &SAMLService{
		providers: make(map[string]*SAMLProvider),
	}

	provider := &SAMLProvider{
		Name:           "test-provider",
		GroupAttribute: "", // Should default to "groups"
	}

	service.providers["test-provider"] = provider

	assertion := &SAMLAssertion{
		Attributes: map[string][]string{
			"groups": {"admin", "editor"},
		},
	}

	groups := service.ExtractGroups("test-provider", assertion)
	assert.Equal(t, []string{"admin", "editor"}, groups)
}

func TestExtractGroups_CustomAttribute(t *testing.T) {
	service := &SAMLService{
		providers: make(map[string]*SAMLProvider),
	}

	provider := &SAMLProvider{
		Name:           "test-provider",
		GroupAttribute: "memberOf",
	}

	service.providers["test-provider"] = provider

	assertion := &SAMLAssertion{
		Attributes: map[string][]string{
			"memberOf": {"CN=Admins,OU=Groups,DC=example,DC=com", "CN=Users,OU=Groups,DC=example,DC=com"},
			"groups":   {"should-not-use-this"},
		},
	}

	groups := service.ExtractGroups("test-provider", assertion)
	assert.Equal(t, []string{"CN=Admins,OU=Groups,DC=example,DC=com", "CN=Users,OU=Groups,DC=example,DC=com"}, groups)
}

func TestExtractGroups_FallbackToCommonAttributes(t *testing.T) {
	service := &SAMLService{
		providers: make(map[string]*SAMLProvider),
	}

	provider := &SAMLProvider{
		Name:           "test-provider",
		GroupAttribute: "customGroups", // Not present in assertion
	}

	service.providers["test-provider"] = provider

	// Should fallback to common group attributes
	assertion := &SAMLAssertion{
		Attributes: map[string][]string{
			"memberOf": {"group1", "group2"},
		},
	}

	groups := service.ExtractGroups("test-provider", assertion)
	assert.Equal(t, []string{"group1", "group2"}, groups)
}

func TestExtractGroups_MicrosoftClaimFormat(t *testing.T) {
	service := &SAMLService{
		providers: make(map[string]*SAMLProvider),
	}

	provider := &SAMLProvider{
		Name:           "test-provider",
		GroupAttribute: "",
	}

	service.providers["test-provider"] = provider

	// Azure AD uses long claim URIs
	assertion := &SAMLAssertion{
		Attributes: map[string][]string{
			"http://schemas.microsoft.com/ws/2008/06/identity/claims/groups": {"group-uuid-1", "group-uuid-2"},
		},
	}

	groups := service.ExtractGroups("test-provider", assertion)
	assert.Equal(t, []string{"group-uuid-1", "group-uuid-2"}, groups)
}

func TestExtractGroups_NoGroupsFound(t *testing.T) {
	service := &SAMLService{
		providers: make(map[string]*SAMLProvider),
	}

	provider := &SAMLProvider{
		Name:           "test-provider",
		GroupAttribute: "groups",
	}

	service.providers["test-provider"] = provider

	assertion := &SAMLAssertion{
		Attributes: map[string][]string{
			"email": {"user@example.com"},
			"name":  {"Test User"},
		},
	}

	groups := service.ExtractGroups("test-provider", assertion)
	assert.Empty(t, groups)
}

func TestExtractGroups_ProviderNotFound(t *testing.T) {
	service := &SAMLService{
		providers: make(map[string]*SAMLProvider),
	}

	assertion := &SAMLAssertion{
		Attributes: map[string][]string{
			"groups": {"admin"},
		},
	}

	groups := service.ExtractGroups("nonexistent-provider", assertion)
	assert.Empty(t, groups)
}
