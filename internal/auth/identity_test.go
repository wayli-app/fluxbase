package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIdentityErrors(t *testing.T) {
	t.Run("error types are defined", func(t *testing.T) {
		assert.NotNil(t, ErrIdentityNotFound)
		assert.NotNil(t, ErrIdentityAlreadyLinked)
	})

	t.Run("error messages are meaningful", func(t *testing.T) {
		assert.Contains(t, ErrIdentityNotFound.Error(), "not found")
		assert.Contains(t, ErrIdentityAlreadyLinked.Error(), "already linked")
	})
}

func TestUserIdentity_Struct(t *testing.T) {
	t.Run("creates identity with all fields", func(t *testing.T) {
		now := time.Now()
		email := "user@example.com"

		identity := UserIdentity{
			ID:             "identity-123",
			UserID:         "user-456",
			Provider:       "google",
			ProviderUserID: "google-user-789",
			Email:          &email,
			IdentityData: map[string]interface{}{
				"name":    "Test User",
				"picture": "https://example.com/avatar.jpg",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		assert.Equal(t, "identity-123", identity.ID)
		assert.Equal(t, "user-456", identity.UserID)
		assert.Equal(t, "google", identity.Provider)
		assert.Equal(t, "google-user-789", identity.ProviderUserID)
		assert.Equal(t, "user@example.com", *identity.Email)
		assert.Equal(t, "Test User", identity.IdentityData["name"])
	})

	t.Run("handles nil optional fields", func(t *testing.T) {
		identity := UserIdentity{
			ID:             "identity-123",
			UserID:         "user-456",
			Provider:       "github",
			ProviderUserID: "github-user-789",
		}

		assert.Nil(t, identity.Email)
		assert.Nil(t, identity.IdentityData)
	})
}

func TestNewIdentityRepository(t *testing.T) {
	// Test that it doesn't panic with nil db
	repo := NewIdentityRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewIdentityService(t *testing.T) {
	// Test that it doesn't panic with nil dependencies
	svc := NewIdentityService(nil, nil, nil)
	assert.NotNil(t, svc)
}
