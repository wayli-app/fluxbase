package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit tests for computeOIDCUserUpdate, the pure decision core extracted from
// Service.updateUserFromOIDCClaims (issue #313 option C). These cover the
// policy that governs whether/how an OIDC IdP's claims overwrite local user
// data on each sign-in — security-relevant because it controls propagation of
// email-verified status and profile fields.

func TestComputeOIDCUserUpdate_NoChange(t *testing.T) {
	t.Parallel()
	// Everything already matches; no update needed.
	user := &User{
		EmailVerified: true,
		UserMetadata: map[string]interface{}{
			"name":    "Alice",
			"picture": "https://example.com/a.png",
		},
	}
	claims := &IDTokenClaims{
		EmailVerified: true,
		Name:          "Alice",
		Picture:       "https://example.com/a.png",
	}

	req, needsUpdate := computeOIDCUserUpdate(user, claims)
	assert.False(t, needsUpdate, "identical state must not trigger an update")
	_ = req
}

func TestComputeOIDCUserUpdate_PromotesEmailVerified(t *testing.T) {
	t.Parallel()
	// IdP says verified, local says not → must promote and flag update.
	user := &User{EmailVerified: false}
	claims := &IDTokenClaims{EmailVerified: true}

	req, needsUpdate := computeOIDCUserUpdate(user, claims)
	require.True(t, needsUpdate)
	require.NotNil(t, req.EmailVerified)
	assert.Equal(t, true, *req.EmailVerified)
}

func TestComputeOIDCUserUpdate_DoesNotDemoteEmailVerified(t *testing.T) {
	t.Parallel()
	// Local already verified; IdP says not verified → must NOT demote (one-way ratchet).
	user := &User{EmailVerified: true}
	claims := &IDTokenClaims{EmailVerified: false}

	_, needsUpdate := computeOIDCUserUpdate(user, claims)
	assert.False(t, needsUpdate, "must not demote an already-verified email")
}

func TestComputeOIDCUserUpdate_SyncsNameAndPicture(t *testing.T) {
	t.Parallel()
	user := &User{
		UserMetadata: map[string]interface{}{
			"name":    "Old",
			"picture": "https://old.png",
			"extra":   "kept", // unrelated metadata must be preserved
		},
	}
	claims := &IDTokenClaims{
		Name:    "New",
		Picture: "https://new.png",
	}

	req, needsUpdate := computeOIDCUserUpdate(user, claims)
	require.True(t, needsUpdate)

	md, ok := req.UserMetadata.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "New", md["name"])
	assert.Equal(t, "https://new.png", md["picture"])
	assert.Equal(t, "kept", md["extra"], "existing metadata keys must be preserved")
}

func TestComputeOIDCUserUpdate_EmptyClaimsDoNotOverwrite(t *testing.T) {
	t.Parallel()
	// Empty Name/Picture in claims must not clear existing values.
	user := &User{
		UserMetadata: map[string]interface{}{
			"name": "Alice",
		},
	}
	claims := &IDTokenClaims{} // both empty

	_, needsUpdate := computeOIDCUserUpdate(user, claims)
	assert.False(t, needsUpdate, "empty claim fields must not trigger an update")
}

func TestComputeOIDCUserUpdate_NilMetadata(t *testing.T) {
	t.Parallel()
	// User with nil metadata + a Name claim → must not panic, must update.
	user := &User{UserMetadata: nil}
	claims := &IDTokenClaims{Name: "Alice"}

	req, needsUpdate := computeOIDCUserUpdate(user, claims)
	require.True(t, needsUpdate)
	md, ok := req.UserMetadata.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Alice", md["name"])
}

func TestComputeOIDCUserUpdate_NonMapMetadata(t *testing.T) {
	t.Parallel()
	// If UserMetadata is a non-map (e.g. a JSON string), the code treats it as
	// empty and builds fresh metadata. Pin this so a future change to the
	// metadata representation is deliberate.
	user := &User{UserMetadata: "not-a-map"}
	claims := &IDTokenClaims{Name: "Alice"}

	req, needsUpdate := computeOIDCUserUpdate(user, claims)
	require.True(t, needsUpdate)
	md, ok := req.UserMetadata.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Alice", md["name"], "non-map metadata is replaced, not merged")
}
