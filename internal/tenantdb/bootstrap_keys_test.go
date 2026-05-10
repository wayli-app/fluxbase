package tenantdb

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"

	"github.com/nimbleflux/fluxbase/internal/keys"
)

func TestDefaultScopes(t *testing.T) {
	tests := []struct {
		name     string
		keyType  string
		expected []string
	}{
		{
			name:     "tenant service key returns wildcard scope",
			keyType:  keys.KeyTypeTenantService,
			expected: []string{"*"},
		},
		{
			name:     "anon key returns read scope",
			keyType:  keys.KeyTypeAnon,
			expected: []string{"read"},
		},
		{
			name:     "unknown key type returns empty scopes",
			keyType:  "unknown",
			expected: []string{},
		},
		{
			name:     "empty key type returns empty scopes",
			keyType:  "",
			expected: []string{},
		},
		{
			name:     "global service key type returns empty scopes",
			keyType:  keys.KeyTypeGlobalService,
			expected: []string{},
		},
		{
			name:     "publishable key type returns empty scopes",
			keyType:  keys.KeyTypePublishable,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultScopes(tt.keyType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultScopes_ReturnsNewSliceEachCall(t *testing.T) {
	s1 := defaultScopes(keys.KeyTypeTenantService)
	s2 := defaultScopes(keys.KeyTypeTenantService)
	s1[0] = "modified"
	assert.Equal(t, "*", s2[0])
}

func TestDefaultRateLimit(t *testing.T) {
	tests := []struct {
		name     string
		keyType  string
		expected int
	}{
		{
			name:     "tenant service key returns 10000",
			keyType:  keys.KeyTypeTenantService,
			expected: 10000,
		},
		{
			name:     "anon key returns 60",
			keyType:  keys.KeyTypeAnon,
			expected: 60,
		},
		{
			name:     "unknown key type returns 60",
			keyType:  "unknown",
			expected: 60,
		},
		{
			name:     "empty key type returns 60",
			keyType:  "",
			expected: 60,
		},
		{
			name:     "global service key type returns 60",
			keyType:  keys.KeyTypeGlobalService,
			expected: 60,
		},
		{
			name:     "publishable key type returns 60",
			keyType:  keys.KeyTypePublishable,
			expected: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultRateLimit(tt.keyType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNoRowsError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "pgx ErrNoRows returns true",
			err:      pgx.ErrNoRows,
			expected: true,
		},
		{
			name:     "wrapped pgx ErrNoRows returns true",
			err:      fmt.Errorf("query failed: %w", pgx.ErrNoRows),
			expected: true,
		},
		{
			name:     "nil error returns false",
			err:      nil,
			expected: false,
		},
		{
			name:     "other error returns false",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "context canceled returns false",
			err:      errors.New("context canceled"),
			expected: false,
		},
		{
			name:     "connection error returns false",
			err:      errors.New("connection refused"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNoRowsError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnsureDefaultTenantAndKeys_SkipWithoutDB(t *testing.T) {
	t.Skip("EnsureDefaultTenantAndKeys requires a real database pool - covered by integration tests")
}

func TestGetDefaultTenantID_SkipWithoutDB(t *testing.T) {
	t.Skip("GetDefaultTenantID requires a real database pool - covered by integration tests")
}
