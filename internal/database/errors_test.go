package database

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestErrorCodeConstants(t *testing.T) {
	assert.Equal(t, "23505", ErrCodeUniqueViolation)
	assert.Equal(t, "23503", ErrCodeForeignKeyViolation)
	assert.Equal(t, "23514", ErrCodeCheckViolation)
}

func TestIsUniqueViolation(t *testing.T) {
	t.Run("returns true for unique violation error", func(t *testing.T) {
		err := &pgconn.PgError{Code: ErrCodeUniqueViolation}
		assert.True(t, IsUniqueViolation(err))
	})

	t.Run("returns false for other pg errors", func(t *testing.T) {
		err := &pgconn.PgError{Code: ErrCodeForeignKeyViolation}
		assert.False(t, IsUniqueViolation(err))
	})

	t.Run("returns false for non-pg error", func(t *testing.T) {
		err := errors.New("generic error")
		assert.False(t, IsUniqueViolation(err))
	})

	t.Run("returns false for nil error", func(t *testing.T) {
		assert.False(t, IsUniqueViolation(nil))
	})

	t.Run("returns false for wrapped non-pg error", func(t *testing.T) {
		wrappedErr := errors.New("wrapped generic error")
		assert.False(t, IsUniqueViolation(wrappedErr))
	})
}

func TestIsForeignKeyViolation(t *testing.T) {
	t.Run("returns true for foreign key violation error", func(t *testing.T) {
		err := &pgconn.PgError{Code: ErrCodeForeignKeyViolation}
		assert.True(t, IsForeignKeyViolation(err))
	})

	t.Run("returns false for other pg errors", func(t *testing.T) {
		err := &pgconn.PgError{Code: ErrCodeUniqueViolation}
		assert.False(t, IsForeignKeyViolation(err))
	})

	t.Run("returns false for non-pg error", func(t *testing.T) {
		err := errors.New("generic error")
		assert.False(t, IsForeignKeyViolation(err))
	})

	t.Run("returns false for nil error", func(t *testing.T) {
		assert.False(t, IsForeignKeyViolation(nil))
	})
}

func TestIsCheckViolation(t *testing.T) {
	t.Run("returns true for check violation error", func(t *testing.T) {
		err := &pgconn.PgError{Code: ErrCodeCheckViolation}
		assert.True(t, IsCheckViolation(err))
	})

	t.Run("returns false for other pg errors", func(t *testing.T) {
		err := &pgconn.PgError{Code: ErrCodeUniqueViolation}
		assert.False(t, IsCheckViolation(err))
	})

	t.Run("returns false for non-pg error", func(t *testing.T) {
		err := errors.New("generic error")
		assert.False(t, IsCheckViolation(err))
	})

	t.Run("returns false for nil error", func(t *testing.T) {
		assert.False(t, IsCheckViolation(nil))
	})
}

func TestGetConstraintName(t *testing.T) {
	t.Run("returns constraint name from pg error", func(t *testing.T) {
		err := &pgconn.PgError{
			Code:           ErrCodeUniqueViolation,
			ConstraintName: "users_email_key",
		}
		assert.Equal(t, "users_email_key", GetConstraintName(err))
	})

	t.Run("returns empty string for non-pg error", func(t *testing.T) {
		err := errors.New("generic error")
		assert.Equal(t, "", GetConstraintName(err))
	})

	t.Run("returns empty string for nil error", func(t *testing.T) {
		assert.Equal(t, "", GetConstraintName(nil))
	})

	t.Run("returns empty string when no constraint name set", func(t *testing.T) {
		err := &pgconn.PgError{Code: ErrCodeCheckViolation}
		assert.Equal(t, "", GetConstraintName(err))
	})
}
