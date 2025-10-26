package database

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL error codes
const (
	// ErrCodeUniqueViolation is the PostgreSQL error code for unique constraint violations
	ErrCodeUniqueViolation = "23505"
	// ErrCodeForeignKeyViolation is the PostgreSQL error code for foreign key violations
	ErrCodeForeignKeyViolation = "23503"
	// ErrCodeCheckViolation is the PostgreSQL error code for check constraint violations
	ErrCodeCheckViolation = "23514"
)

// IsUniqueViolation checks if an error is a unique constraint violation
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == ErrCodeUniqueViolation
	}
	return false
}

// IsForeignKeyViolation checks if an error is a foreign key violation
func IsForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == ErrCodeForeignKeyViolation
	}
	return false
}

// IsCheckViolation checks if an error is a check constraint violation
func IsCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == ErrCodeCheckViolation
	}
	return false
}

// GetConstraintName returns the constraint name from a PostgreSQL error
func GetConstraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}
