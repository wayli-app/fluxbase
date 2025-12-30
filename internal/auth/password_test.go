package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestNewPasswordHasher(t *testing.T) {
	hasher := NewPasswordHasher()

	assert.NotNil(t, hasher)
	assert.Equal(t, DefaultBcryptCost, hasher.cost)
	assert.Equal(t, MinPasswordLength, hasher.minLength)
	// New secure defaults require upper, lower, and digit (but not symbol)
	assert.True(t, hasher.requireUpper)
	assert.True(t, hasher.requireLower)
	assert.True(t, hasher.requireDigit)
	assert.False(t, hasher.requireSymbol)
}

func TestNewPasswordHasherWithConfig(t *testing.T) {
	config := PasswordHasherConfig{
		Cost:          10,
		MinLength:     12,
		RequireUpper:  true,
		RequireLower:  true,
		RequireDigit:  true,
		RequireSymbol: true,
	}

	hasher := NewPasswordHasherWithConfig(config)

	assert.NotNil(t, hasher)
	assert.Equal(t, 10, hasher.cost)
	assert.Equal(t, 12, hasher.minLength)
	assert.True(t, hasher.requireUpper)
	assert.True(t, hasher.requireLower)
	assert.True(t, hasher.requireDigit)
	assert.True(t, hasher.requireSymbol)
}

func TestNewPasswordHasherWithConfig_Defaults(t *testing.T) {
	config := PasswordHasherConfig{} // All zero values

	hasher := NewPasswordHasherWithConfig(config)

	assert.Equal(t, DefaultBcryptCost, hasher.cost)
	assert.Equal(t, MinPasswordLength, hasher.minLength)
}

func TestHashPassword_Success(t *testing.T) {
	hasher := NewPasswordHasherWithConfig(PasswordHasherConfig{Cost: 4}) // Use low cost for faster tests
	password := "testpassword123"

	hash, err := hasher.HashPassword(password)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Verify it's a valid bcrypt hash
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	assert.NoError(t, err)
}

func TestHashPassword_DifferentHashesForSamePassword(t *testing.T) {
	hasher := NewPasswordHasherWithConfig(PasswordHasherConfig{Cost: 4}) // Use low cost for faster tests
	password := "testpassword123"

	hash1, err1 := hasher.HashPassword(password)
	hash2, err2 := hasher.HashPassword(password)

	require.NoError(t, err1)
	require.NoError(t, err2)

	// Hashes should be different due to salt
	assert.NotEqual(t, hash1, hash2)

	// But both should validate against the same password
	assert.NoError(t, hasher.ComparePassword(hash1, password))
	assert.NoError(t, hasher.ComparePassword(hash2, password))
}

func TestHashPassword_TooShort(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "short" // Less than 8 characters

	hash, err := hasher.HashPassword(password)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrWeakPassword)
	assert.Empty(t, hash)
}

func TestHashPassword_TooLong(t *testing.T) {
	hasher := NewPasswordHasher()
	password := strings.Repeat("a", MaxPasswordLength+1)

	hash, err := hasher.HashPassword(password)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrPasswordTooLong)
	assert.Empty(t, hash)
}

func TestComparePassword_Success(t *testing.T) {
	hasher := NewPasswordHasherWithConfig(PasswordHasherConfig{Cost: 4}) // Use low cost for faster tests
	password := "testpassword123"

	hash, err := hasher.HashPassword(password)
	require.NoError(t, err)

	err = hasher.ComparePassword(hash, password)
	assert.NoError(t, err)
}

func TestComparePassword_WrongPassword(t *testing.T) {
	hasher := NewPasswordHasherWithConfig(PasswordHasherConfig{Cost: 4}) // Use low cost for faster tests
	password := "testpassword123"

	hash, err := hasher.HashPassword(password)
	require.NoError(t, err)

	err = hasher.ComparePassword(hash, "wrongpassword")
	assert.Error(t, err)
}

func TestComparePassword_CaseSensitive(t *testing.T) {
	hasher := NewPasswordHasherWithConfig(PasswordHasherConfig{Cost: 4}) // Use low cost for faster tests
	password := "TestPassword123"

	hash, err := hasher.HashPassword(password)
	require.NoError(t, err)

	// Wrong case should fail
	err = hasher.ComparePassword(hash, "testpassword123")
	assert.Error(t, err)

	// Correct case should succeed
	err = hasher.ComparePassword(hash, password)
	assert.NoError(t, err)
}

func TestValidatePassword_MinLength(t *testing.T) {
	hasher := NewPasswordHasherWithConfig(PasswordHasherConfig{
		MinLength: 10,
	})

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"too short", "short", true},
		{"exactly min length", "1234567890", false},
		{"longer than min", "12345678901", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hasher.ValidatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePassword_RequireUpper(t *testing.T) {
	hasher := NewPasswordHasherWithConfig(PasswordHasherConfig{
		MinLength:    8,
		RequireUpper: true,
	})

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"no uppercase", "password123", true},
		{"has uppercase", "Password123", false},
		{"multiple uppercase", "PASSWORD123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hasher.ValidatePassword(tt.password)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrWeakPassword)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePassword_RequireLower(t *testing.T) {
	hasher := NewPasswordHasherWithConfig(PasswordHasherConfig{
		MinLength:    8,
		RequireLower: true,
	})

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"no lowercase", "PASSWORD123", true},
		{"has lowercase", "Password123", false},
		{"multiple lowercase", "password123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hasher.ValidatePassword(tt.password)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrWeakPassword)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePassword_RequireDigit(t *testing.T) {
	hasher := NewPasswordHasherWithConfig(PasswordHasherConfig{
		MinLength:    8,
		RequireDigit: true,
	})

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"no digit", "password", true},
		{"has digit", "password1", false},
		{"multiple digits", "password123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hasher.ValidatePassword(tt.password)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrWeakPassword)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePassword_RequireSymbol(t *testing.T) {
	hasher := NewPasswordHasherWithConfig(PasswordHasherConfig{
		MinLength:     8,
		RequireSymbol: true,
	})

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"no symbol", "password123", true},
		{"has punctuation", "password!", false},
		{"has symbol", "password@123", false},
		{"multiple symbols", "p@ssw0rd!#", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hasher.ValidatePassword(tt.password)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrWeakPassword)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePassword_AllRequirements(t *testing.T) {
	hasher := NewPasswordHasherWithConfig(PasswordHasherConfig{
		MinLength:     12,
		RequireUpper:  true,
		RequireLower:  true,
		RequireDigit:  true,
		RequireSymbol: true,
	})

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"missing all", "password", true},
		{"missing upper", "password123!", true},
		{"missing lower", "PASSWORD123!", true},
		{"missing digit", "Password!", true},
		{"missing symbol", "Password123", true},
		{"too short", "Pass123!", true},
		{"valid password", "ValidPass123!", false},
		{"valid complex", "MyP@ssw0rd123!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hasher.ValidatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNeedsRehash(t *testing.T) {
	// Create hash with cost 4 (fast for testing)
	hasher4 := NewPasswordHasherWithConfig(PasswordHasherConfig{Cost: 4})
	hash4, err := hasher4.HashPassword("testpassword")
	require.NoError(t, err)

	// Create hasher with cost 5
	hasher5 := NewPasswordHasherWithConfig(PasswordHasherConfig{Cost: 5})

	// Should need rehash because cost is different
	assert.True(t, hasher5.NeedsRehash(hash4))

	// Hash with cost 5
	hash5, err := hasher5.HashPassword("testpassword")
	require.NoError(t, err)

	// Should not need rehash because cost matches
	assert.False(t, hasher5.NeedsRehash(hash5))
}

func TestNeedsRehash_InvalidHash(t *testing.T) {
	hasher := NewPasswordHasher()

	// Invalid hash should need rehash
	assert.True(t, hasher.NeedsRehash("invalid-hash"))
	assert.True(t, hasher.NeedsRehash(""))
}

func TestPasswordHasher_RealWorldUsage(t *testing.T) {
	// Simulate real-world user registration and login flow
	hasher := NewPasswordHasherWithConfig(PasswordHasherConfig{
		Cost:          4, // Use low cost for faster tests
		MinLength:     8,
		RequireUpper:  true,
		RequireLower:  true,
		RequireDigit:  true,
		RequireSymbol: false,
	})

	// User registers with a password
	password := "MyPassword123"
	hash, err := hasher.HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	// Simulate storing hash in database (not actually storing, just validating)
	storedHash := hash

	// User attempts to login with correct password
	err = hasher.ComparePassword(storedHash, password)
	assert.NoError(t, err, "Login should succeed with correct password")

	// User attempts to login with wrong password
	err = hasher.ComparePassword(storedHash, "WrongPassword123")
	assert.Error(t, err, "Login should fail with wrong password")

	// Check if rehash is needed (e.g., security policy changed)
	if hasher.NeedsRehash(storedHash) {
		newHash, err := hasher.HashPassword(password)
		require.NoError(t, err)
		storedHash = newHash
	}

	// User should still be able to login after rehash
	err = hasher.ComparePassword(storedHash, password)
	assert.NoError(t, err, "Login should still work after rehash")
}

func TestConcurrentHashing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent hashing test in short mode")
	}

	// Use lower cost for faster testing while still testing concurrency
	hasher := NewPasswordHasherWithConfig(PasswordHasherConfig{Cost: 4})

	// Hash passwords concurrently
	// Reduce to 10 goroutines to avoid extremely long test times with race detector
	const numGoroutines = 10
	results := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			hash, err := hasher.HashPassword("testpassword123")
			require.NoError(t, err)
			results <- hash
		}()
	}

	// Collect all hashes
	hashes := make(map[string]bool)
	for i := 0; i < numGoroutines; i++ {
		hash := <-results
		hashes[hash] = true
	}

	// All hashes should be unique (due to random salt)
	assert.Len(t, hashes, numGoroutines)

	// All hashes should validate against the same password
	for hash := range hashes {
		err := hasher.ComparePassword(hash, "testpassword123")
		assert.NoError(t, err)
	}
}

func BenchmarkHashPassword(b *testing.B) {
	hasher := NewPasswordHasher()
	password := "testpassword123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hasher.HashPassword(password)
	}
}

func BenchmarkComparePassword(b *testing.B) {
	hasher := NewPasswordHasher()
	password := "testpassword123"
	hash, _ := hasher.HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hasher.ComparePassword(hash, password)
	}
}
