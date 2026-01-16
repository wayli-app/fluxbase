package branching

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Error Variables Tests
// =============================================================================

func TestBranchingErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{
			name:     "ErrBranchNotFound",
			err:      ErrBranchNotFound,
			contains: "branch not found",
		},
		{
			name:     "ErrBranchExists",
			err:      ErrBranchExists,
			contains: "branch already exists",
		},
		{
			name:     "ErrCannotDeleteMainBranch",
			err:      ErrCannotDeleteMainBranch,
			contains: "cannot delete main branch",
		},
		{
			name:     "ErrBranchNotReady",
			err:      ErrBranchNotReady,
			contains: "branch is not ready",
		},
		{
			name:     "ErrMaxBranchesReached",
			err:      ErrMaxBranchesReached,
			contains: "maximum number of branches reached",
		},
		{
			name:     "ErrInvalidSlug",
			err:      ErrInvalidSlug,
			contains: "invalid branch slug",
		},
		{
			name:     "ErrGitHubConfigNotFound",
			err:      ErrGitHubConfigNotFound,
			contains: "github config not found",
		},
		{
			name:     "ErrBranchingDisabled",
			err:      ErrBranchingDisabled,
			contains: "database branching is disabled",
		},
		{
			name:     "ErrAccessDenied",
			err:      ErrAccessDenied,
			contains: "access denied to branch",
		},
		{
			name:     "ErrDatabaseOperationFailed",
			err:      ErrDatabaseOperationFailed,
			contains: "database operation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, tt.err)
			assert.Contains(t, tt.err.Error(), tt.contains)
		})
	}
}

func TestBranchingErrors_NotNil(t *testing.T) {
	allErrors := []error{
		ErrBranchNotFound,
		ErrBranchExists,
		ErrCannotDeleteMainBranch,
		ErrBranchNotReady,
		ErrMaxBranchesReached,
		ErrInvalidSlug,
		ErrGitHubConfigNotFound,
		ErrBranchingDisabled,
		ErrAccessDenied,
		ErrDatabaseOperationFailed,
	}

	for _, err := range allErrors {
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	}
}

func TestBranchingErrors_AreDistinct(t *testing.T) {
	allErrors := []error{
		ErrBranchNotFound,
		ErrBranchExists,
		ErrCannotDeleteMainBranch,
		ErrBranchNotReady,
		ErrMaxBranchesReached,
		ErrInvalidSlug,
		ErrGitHubConfigNotFound,
		ErrBranchingDisabled,
		ErrAccessDenied,
		ErrDatabaseOperationFailed,
	}

	// Each error should be distinct from all others
	for i, err1 := range allErrors {
		for j, err2 := range allErrors {
			if i != j {
				assert.NotEqual(t, err1, err2, "Errors at index %d and %d should be different", i, j)
			}
		}
	}
}

func TestBranchingErrors_CanBeWrapped(t *testing.T) {
	wrappedErr := errors.Join(ErrBranchNotFound, errors.New("additional context"))

	assert.True(t, errors.Is(wrappedErr, ErrBranchNotFound))
}

func TestBranchingErrors_ErrorsIs(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		match  bool
	}{
		{"ErrBranchNotFound matches itself", ErrBranchNotFound, ErrBranchNotFound, true},
		{"ErrBranchNotFound doesn't match ErrBranchExists", ErrBranchNotFound, ErrBranchExists, false},
		{"Wrapped error matches original", errors.Join(ErrAccessDenied, errors.New("context")), ErrAccessDenied, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.Is(tt.err, tt.target)
			assert.Equal(t, tt.match, result)
		})
	}
}

// =============================================================================
// Error Categories Tests
// =============================================================================

func TestBranchingErrors_Categories(t *testing.T) {
	t.Run("not found errors", func(t *testing.T) {
		notFoundErrors := []error{
			ErrBranchNotFound,
			ErrGitHubConfigNotFound,
		}

		for _, err := range notFoundErrors {
			assert.Contains(t, err.Error(), "not found")
		}
	})

	t.Run("access errors", func(t *testing.T) {
		accessErrors := []error{
			ErrAccessDenied,
			ErrCannotDeleteMainBranch,
		}

		for _, err := range accessErrors {
			// These are access-related errors
			assert.NotEmpty(t, err.Error())
		}
	})

	t.Run("state errors", func(t *testing.T) {
		stateErrors := []error{
			ErrBranchNotReady,
			ErrBranchExists,
			ErrMaxBranchesReached,
		}

		for _, err := range stateErrors {
			// These are state-related errors
			assert.NotEmpty(t, err.Error())
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		validationErrors := []error{
			ErrInvalidSlug,
		}

		for _, err := range validationErrors {
			assert.Contains(t, err.Error(), "invalid")
		}
	})

	t.Run("configuration errors", func(t *testing.T) {
		configErrors := []error{
			ErrBranchingDisabled,
		}

		for _, err := range configErrors {
			assert.Contains(t, err.Error(), "disabled")
		}
	})

	t.Run("operation errors", func(t *testing.T) {
		operationErrors := []error{
			ErrDatabaseOperationFailed,
		}

		for _, err := range operationErrors {
			assert.Contains(t, err.Error(), "failed")
		}
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkError_String(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ErrBranchNotFound.Error()
	}
}

func BenchmarkErrors_Is(b *testing.B) {
	wrappedErr := errors.Join(ErrBranchNotFound, errors.New("context"))

	for i := 0; i < b.N; i++ {
		_ = errors.Is(wrappedErr, ErrBranchNotFound)
	}
}
