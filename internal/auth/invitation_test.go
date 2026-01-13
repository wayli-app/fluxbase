package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Error Variable Tests
// =============================================================================

func TestInvitationErrors_Defined(t *testing.T) {
	assert.NotNil(t, ErrInvitationNotFound)
	assert.NotNil(t, ErrInvitationExpired)
	assert.NotNil(t, ErrInvitationAlreadyAccepted)
}

func TestInvitationErrors_Messages(t *testing.T) {
	assert.Equal(t, "invitation not found", ErrInvitationNotFound.Error())
	assert.Equal(t, "invitation has expired", ErrInvitationExpired.Error())
	assert.Equal(t, "invitation has already been accepted", ErrInvitationAlreadyAccepted.Error())
}

func TestInvitationErrors_Distinct(t *testing.T) {
	errors := []error{
		ErrInvitationNotFound,
		ErrInvitationExpired,
		ErrInvitationAlreadyAccepted,
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
// InvitationToken Struct Tests
// =============================================================================

func TestInvitationToken_Fields(t *testing.T) {
	now := time.Now()
	invitedBy := uuid.New()
	acceptedAt := now.Add(time.Hour)

	token := InvitationToken{
		ID:         uuid.New(),
		Email:      "newuser@example.com",
		Token:      "abc123xyz789",
		Role:       "authenticated",
		InvitedBy:  &invitedBy,
		ExpiresAt:  now.Add(7 * 24 * time.Hour),
		Accepted:   true,
		AcceptedAt: &acceptedAt,
		CreatedAt:  now,
	}

	assert.NotEqual(t, uuid.Nil, token.ID)
	assert.Equal(t, "newuser@example.com", token.Email)
	assert.Equal(t, "abc123xyz789", token.Token)
	assert.Equal(t, "authenticated", token.Role)
	assert.NotNil(t, token.InvitedBy)
	assert.Equal(t, invitedBy, *token.InvitedBy)
	assert.True(t, token.Accepted)
	assert.NotNil(t, token.AcceptedAt)
}

func TestInvitationToken_NullableFields(t *testing.T) {
	token := InvitationToken{
		ID:         uuid.New(),
		Email:      "user@example.com",
		Token:      "token123",
		Role:       "viewer",
		InvitedBy:  nil,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		Accepted:   false,
		AcceptedAt: nil,
		CreatedAt:  time.Now(),
	}

	assert.Nil(t, token.InvitedBy)
	assert.Nil(t, token.AcceptedAt)
	assert.False(t, token.Accepted)
}

func TestInvitationToken_Roles(t *testing.T) {
	roles := []string{
		"authenticated",
		"admin",
		"viewer",
		"editor",
		"moderator",
		"service_role",
	}

	for _, role := range roles {
		t.Run("role_"+role, func(t *testing.T) {
			token := InvitationToken{
				ID:    uuid.New(),
				Email: "user@example.com",
				Token: "token123",
				Role:  role,
			}

			assert.Equal(t, role, token.Role)
		})
	}
}

func TestInvitationToken_ExpirationState(t *testing.T) {
	t.Run("not expired", func(t *testing.T) {
		future := time.Now().Add(7 * 24 * time.Hour)
		token := InvitationToken{
			ExpiresAt: future,
		}

		assert.True(t, token.ExpiresAt.After(time.Now()))
	})

	t.Run("expired", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		token := InvitationToken{
			ExpiresAt: past,
		}

		assert.True(t, token.ExpiresAt.Before(time.Now()))
	})

	t.Run("just expired", func(t *testing.T) {
		justPast := time.Now().Add(-1 * time.Second)
		token := InvitationToken{
			ExpiresAt: justPast,
		}

		assert.True(t, time.Now().After(token.ExpiresAt))
	})
}

func TestInvitationToken_AcceptedState(t *testing.T) {
	t.Run("pending invitation", func(t *testing.T) {
		token := InvitationToken{
			Accepted:   false,
			AcceptedAt: nil,
		}

		assert.False(t, token.Accepted)
		assert.Nil(t, token.AcceptedAt)
	})

	t.Run("accepted invitation", func(t *testing.T) {
		now := time.Now()
		token := InvitationToken{
			Accepted:   true,
			AcceptedAt: &now,
		}

		assert.True(t, token.Accepted)
		assert.NotNil(t, token.AcceptedAt)
	})
}

// =============================================================================
// InvitationService Tests
// =============================================================================

func TestNewInvitationService(t *testing.T) {
	svc := NewInvitationService(nil)

	require.NotNil(t, svc)
	assert.Nil(t, svc.db)
}

func TestInvitationService_GenerateToken(t *testing.T) {
	svc := NewInvitationService(nil)

	token, err := svc.GenerateToken()

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestInvitationService_GenerateToken_Uniqueness(t *testing.T) {
	svc := NewInvitationService(nil)
	tokens := make(map[string]bool)

	for i := 0; i < 100; i++ {
		token, err := svc.GenerateToken()
		require.NoError(t, err)

		assert.False(t, tokens[token], "Token collision detected")
		tokens[token] = true
	}

	assert.Len(t, tokens, 100)
}

func TestInvitationService_GenerateToken_Base64URLEncoded(t *testing.T) {
	svc := NewInvitationService(nil)

	token, err := svc.GenerateToken()
	require.NoError(t, err)

	// Base64 URL safe characters: A-Z, a-z, 0-9, -, _, =
	for _, c := range token {
		isAlphaNum := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
		isUrlSafe := c == '-' || c == '_' || c == '='
		assert.True(t, isAlphaNum || isUrlSafe, "Invalid character in token: %c", c)
	}
}

func TestInvitationService_GenerateToken_Length(t *testing.T) {
	svc := NewInvitationService(nil)

	token, err := svc.GenerateToken()
	require.NoError(t, err)

	// 32 bytes base64 encoded should be ~44 characters
	assert.GreaterOrEqual(t, len(token), 40)
	assert.LessOrEqual(t, len(token), 48)
}

// =============================================================================
// Validation Logic Tests (without DB)
// =============================================================================

func TestInvitationToken_ValidationLogic(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		token       InvitationToken
		expectedErr error
	}{
		{
			name: "valid invitation",
			token: InvitationToken{
				Email:     "user@example.com",
				Token:     "valid-token",
				Accepted:  false,
				ExpiresAt: now.Add(24 * time.Hour),
			},
			expectedErr: nil,
		},
		{
			name: "already accepted",
			token: InvitationToken{
				Email:      "user@example.com",
				Token:      "accepted-token",
				Accepted:   true,
				AcceptedAt: &now,
				ExpiresAt:  now.Add(24 * time.Hour),
			},
			expectedErr: ErrInvitationAlreadyAccepted,
		},
		{
			name: "expired invitation",
			token: InvitationToken{
				Email:     "user@example.com",
				Token:     "expired-token",
				Accepted:  false,
				ExpiresAt: now.Add(-1 * time.Hour),
			},
			expectedErr: ErrInvitationExpired,
		},
		{
			name: "accepted and expired",
			token: InvitationToken{
				Email:      "user@example.com",
				Token:      "bad-token",
				Accepted:   true,
				AcceptedAt: &now,
				ExpiresAt:  now.Add(-1 * time.Hour),
			},
			expectedErr: ErrInvitationAlreadyAccepted, // Already accepted is checked first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate validation logic from ValidateToken
			var err error

			// Check if already accepted (checked first)
			if tt.token.Accepted {
				err = ErrInvitationAlreadyAccepted
			} else if time.Now().After(tt.token.ExpiresAt) {
				err = ErrInvitationExpired
			}

			if tt.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

// =============================================================================
// Email Validation Tests
// =============================================================================

func TestInvitationToken_EmailFormats(t *testing.T) {
	emails := []string{
		"user@example.com",
		"user.name@example.com",
		"user+tag@example.com",
		"user@subdomain.example.com",
		"user@example.co.uk",
		"user123@example.com",
	}

	for _, email := range emails {
		t.Run(email, func(t *testing.T) {
			token := InvitationToken{
				ID:        uuid.New(),
				Email:     email,
				Token:     "token123",
				Role:      "authenticated",
				ExpiresAt: time.Now().Add(24 * time.Hour),
			}

			assert.Equal(t, email, token.Email)
		})
	}
}

// =============================================================================
// Expiry Duration Tests
// =============================================================================

func TestInvitationToken_ExpiryDurations(t *testing.T) {
	durations := []struct {
		name     string
		duration time.Duration
	}{
		{"1 hour", time.Hour},
		{"24 hours", 24 * time.Hour},
		{"7 days (default)", 7 * 24 * time.Hour},
		{"30 days", 30 * 24 * time.Hour},
	}

	for _, d := range durations {
		t.Run(d.name, func(t *testing.T) {
			now := time.Now()
			expiresAt := now.Add(d.duration)

			token := InvitationToken{
				CreatedAt: now,
				ExpiresAt: expiresAt,
			}

			expectedDuration := token.ExpiresAt.Sub(token.CreatedAt)
			assert.Equal(t, d.duration, expectedDuration)
		})
	}
}

// =============================================================================
// InvitedBy Tests
// =============================================================================

func TestInvitationToken_InvitedBy(t *testing.T) {
	t.Run("with inviter", func(t *testing.T) {
		inviterID := uuid.New()
		token := InvitationToken{
			InvitedBy: &inviterID,
		}

		assert.NotNil(t, token.InvitedBy)
		assert.Equal(t, inviterID, *token.InvitedBy)
	})

	t.Run("without inviter (system invitation)", func(t *testing.T) {
		token := InvitationToken{
			InvitedBy: nil,
		}

		assert.Nil(t, token.InvitedBy)
	})
}

// =============================================================================
// Token Lifecycle Tests
// =============================================================================

func TestInvitationToken_Lifecycle(t *testing.T) {
	t.Run("full lifecycle - created to accepted", func(t *testing.T) {
		createdAt := time.Now()
		inviterID := uuid.New()

		// 1. Created state
		token := InvitationToken{
			ID:        uuid.New(),
			Email:     "newuser@example.com",
			Token:     "lifecycle-token",
			Role:      "authenticated",
			InvitedBy: &inviterID,
			ExpiresAt: createdAt.Add(7 * 24 * time.Hour),
			Accepted:  false,
			CreatedAt: createdAt,
		}

		assert.False(t, token.Accepted)
		assert.Nil(t, token.AcceptedAt)
		assert.True(t, token.ExpiresAt.After(time.Now()))

		// 2. Accepted state
		acceptedAt := time.Now()
		token.Accepted = true
		token.AcceptedAt = &acceptedAt

		assert.True(t, token.Accepted)
		assert.NotNil(t, token.AcceptedAt)
		assert.True(t, token.AcceptedAt.After(token.CreatedAt))
	})
}

// =============================================================================
// ListInvitations Filter Logic Tests
// =============================================================================

func TestListInvitations_FilterLogic(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	// Sample invitations for testing filter logic
	invitations := []InvitationToken{
		{ID: uuid.New(), Accepted: false, ExpiresAt: future},  // Active
		{ID: uuid.New(), Accepted: true, ExpiresAt: future},   // Accepted, not expired
		{ID: uuid.New(), Accepted: false, ExpiresAt: past},    // Not accepted, expired
		{ID: uuid.New(), Accepted: true, ExpiresAt: past},     // Accepted and expired
	}

	t.Run("filter active only", func(t *testing.T) {
		includeAccepted := false
		includeExpired := false

		var filtered []InvitationToken
		for _, inv := range invitations {
			if !includeAccepted && inv.Accepted {
				continue
			}
			if !includeExpired && inv.ExpiresAt.Before(now) {
				continue
			}
			filtered = append(filtered, inv)
		}

		assert.Len(t, filtered, 1) // Only the active one
	})

	t.Run("include accepted", func(t *testing.T) {
		includeAccepted := true
		includeExpired := false

		var filtered []InvitationToken
		for _, inv := range invitations {
			if !includeAccepted && inv.Accepted {
				continue
			}
			if !includeExpired && inv.ExpiresAt.Before(now) {
				continue
			}
			filtered = append(filtered, inv)
		}

		assert.Len(t, filtered, 2) // Active + Accepted not expired
	})

	t.Run("include expired", func(t *testing.T) {
		includeAccepted := false
		includeExpired := true

		var filtered []InvitationToken
		for _, inv := range invitations {
			if !includeAccepted && inv.Accepted {
				continue
			}
			if !includeExpired && inv.ExpiresAt.Before(now) {
				continue
			}
			filtered = append(filtered, inv)
		}

		assert.Len(t, filtered, 2) // Active + Not accepted but expired
	})

	t.Run("include all", func(t *testing.T) {
		includeAccepted := true
		includeExpired := true

		var filtered []InvitationToken
		for _, inv := range invitations {
			if !includeAccepted && inv.Accepted {
				continue
			}
			if !includeExpired && inv.ExpiresAt.Before(now) {
				continue
			}
			filtered = append(filtered, inv)
		}

		assert.Len(t, filtered, 4) // All invitations
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkInvitationService_GenerateToken(b *testing.B) {
	svc := NewInvitationService(nil)

	for i := 0; i < b.N; i++ {
		_, _ = svc.GenerateToken()
	}
}

func BenchmarkInvitationToken_Creation(b *testing.B) {
	inviterID := uuid.New()
	now := time.Now()

	for i := 0; i < b.N; i++ {
		_ = InvitationToken{
			ID:        uuid.New(),
			Email:     "user@example.com",
			Token:     "token123",
			Role:      "authenticated",
			InvitedBy: &inviterID,
			ExpiresAt: now.Add(7 * 24 * time.Hour),
			Accepted:  false,
			CreatedAt: now,
		}
	}
}

func BenchmarkNewInvitationService(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewInvitationService(nil)
	}
}

// =============================================================================
// Concurrent Token Generation Tests
// =============================================================================

func TestInvitationService_ConcurrentTokenGeneration(t *testing.T) {
	svc := NewInvitationService(nil)
	tokens := make(chan string, 100)
	done := make(chan bool)

	// Generate tokens concurrently
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				token, err := svc.GenerateToken()
				require.NoError(t, err)
				tokens <- token
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	close(tokens)

	// Collect and verify uniqueness
	seen := make(map[string]bool)
	for token := range tokens {
		assert.False(t, seen[token], "Token collision detected")
		seen[token] = true
	}

	assert.Len(t, seen, 100)
}
