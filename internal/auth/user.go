package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrUserAlreadyExists is returned when trying to create a user with existing email
	ErrUserAlreadyExists = errors.New("user with this email already exists")
	// ErrInvalidCredentials is returned when login credentials are invalid
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrAccountLocked is returned when an account is locked due to too many failed login attempts
	ErrAccountLocked = errors.New("account locked due to too many failed login attempts")
)

// User represents a user in the system
type User struct {
	ID                  string     `json:"id" db:"id"`
	Email               string     `json:"email" db:"email"`
	PasswordHash        string     `json:"-" db:"password_hash"` // Never expose in JSON
	EmailVerified       bool       `json:"email_verified" db:"email_verified"`
	Role                string     `json:"role,omitempty" db:"role"`
	UserMetadata        any        `json:"user_metadata,omitempty" db:"user_metadata"` // User-editable metadata
	AppMetadata         any        `json:"app_metadata,omitempty" db:"app_metadata"`   // Application/admin-only metadata
	FailedLoginAttempts int        `json:"-" db:"failed_login_attempts"`               // Track failed logins for lockout
	IsLocked            bool       `json:"-" db:"is_locked"`                           // Account locked status
	LockedUntil         *time.Time `json:"-" db:"locked_until"`                        // Lock expiry time
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	Role         string `json:"role,omitempty"`
	UserMetadata any    `json:"user_metadata,omitempty"` // User-editable metadata
	AppMetadata  any    `json:"app_metadata,omitempty"`  // Application/admin-only metadata
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	Email         *string `json:"email,omitempty"`
	EmailVerified *bool   `json:"email_verified,omitempty"`
	Role          *string `json:"role,omitempty"`
	UserMetadata  any     `json:"user_metadata,omitempty"` // User-editable metadata
	AppMetadata   any     `json:"app_metadata,omitempty"`  // Application/admin-only metadata (admin only)
}

// UserRepository handles database operations for users
type UserRepository struct {
	db *database.Connection
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.Connection) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, req CreateUserRequest, passwordHash string) (*User, error) {
	user := &User{
		ID:            uuid.New().String(),
		Email:         req.Email,
		PasswordHash:  passwordHash,
		EmailVerified: false,
		Role:          req.Role,
		UserMetadata:  req.UserMetadata,
		AppMetadata:   req.AppMetadata,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Set default role if not provided
	if user.Role == "" {
		user.Role = "authenticated"
	}

	query := `
		INSERT INTO auth.users (id, email, password_hash, email_verified, role, user_metadata, app_metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, email, email_verified, role, user_metadata, app_metadata, created_at, updated_at
	`

	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, query,
			user.ID,
			user.Email,
			user.PasswordHash,
			user.EmailVerified,
			user.Role,
			user.UserMetadata,
			user.AppMetadata,
			user.CreatedAt,
			user.UpdatedAt,
		)

		return row.Scan(
			&user.ID,
			&user.Email,
			&user.EmailVerified,
			&user.Role,
			&user.UserMetadata,
			&user.AppMetadata,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
	})

	if err != nil {
		// Check for unique constraint violation
		if database.IsUniqueViolation(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	return user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, email, password_hash, email_verified, role, user_metadata, app_metadata,
		       COALESCE(failed_login_attempts, 0), COALESCE(is_locked, false), locked_until,
		       created_at, updated_at
		FROM auth.users
		WHERE id = $1
	`

	user := &User{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, id).Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.EmailVerified,
			&user.Role,
			&user.UserMetadata,
			&user.AppMetadata,
			&user.FailedLoginAttempts,
			&user.IsLocked,
			&user.LockedUntil,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, email_verified, role, user_metadata, app_metadata,
		       COALESCE(failed_login_attempts, 0), COALESCE(is_locked, false), locked_until,
		       created_at, updated_at
		FROM auth.users
		WHERE email = $1
	`

	user := &User{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, email).Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.EmailVerified,
			&user.Role,
			&user.UserMetadata,
			&user.AppMetadata,
			&user.FailedLoginAttempts,
			&user.IsLocked,
			&user.LockedUntil,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// IncrementFailedLoginAttempts increments failed login attempts and locks account after threshold
// SECURITY FIX: Now sets locked_until to allow automatic unlock after 15 minutes
func (r *UserRepository) IncrementFailedLoginAttempts(ctx context.Context, userID string) error {
	query := `
		UPDATE auth.users
		SET failed_login_attempts = COALESCE(failed_login_attempts, 0) + 1,
		    is_locked = CASE WHEN COALESCE(failed_login_attempts, 0) >= 4 THEN true ELSE false END,
		    locked_until = CASE WHEN COALESCE(failed_login_attempts, 0) >= 4 THEN NOW() + interval '15 minutes' ELSE locked_until END,
		    updated_at = NOW()
		WHERE id = $1
	`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, userID)
		return err
	})
}

// ResetFailedLoginAttempts resets failed login attempts after successful login
func (r *UserRepository) ResetFailedLoginAttempts(ctx context.Context, userID string) error {
	query := `
		UPDATE auth.users
		SET failed_login_attempts = 0,
		    is_locked = false,
		    locked_until = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, userID)
		return err
	})
}

// UnlockUser unlocks a user account (admin operation)
func (r *UserRepository) UnlockUser(ctx context.Context, userID string) error {
	return r.ResetFailedLoginAttempts(ctx, userID)
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, id string, req UpdateUserRequest) (*User, error) {
	// Build dynamic update query
	updates := []string{}
	args := []interface{}{id}
	argCount := 2

	if req.Email != nil {
		updates = append(updates, formatPlaceholder("email", argCount))
		args = append(args, *req.Email)
		argCount++
	}

	if req.EmailVerified != nil {
		updates = append(updates, formatPlaceholder("email_verified", argCount))
		args = append(args, *req.EmailVerified)
		argCount++
	}

	if req.Role != nil {
		updates = append(updates, formatPlaceholder("role", argCount))
		args = append(args, *req.Role)
		argCount++
	}

	if req.UserMetadata != nil {
		updates = append(updates, formatPlaceholder("user_metadata", argCount))
		args = append(args, req.UserMetadata)
		argCount++
	}

	if req.AppMetadata != nil {
		updates = append(updates, formatPlaceholder("app_metadata", argCount))
		args = append(args, req.AppMetadata)
	}

	if len(updates) == 0 {
		// No updates, just return current user
		return r.GetByID(ctx, id)
	}

	// Always update updated_at
	updates = append(updates, "updated_at = NOW()")

	query := `
		UPDATE auth.users
		SET ` + joinStrings(updates, ", ") + `
		WHERE id = $1
		RETURNING id, email, email_verified, role, user_metadata, app_metadata, created_at, updated_at
	`

	user := &User{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, args...).Scan(
			&user.ID,
			&user.Email,
			&user.EmailVerified,
			&user.Role,
			&user.UserMetadata,
			&user.AppMetadata,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		if database.IsUniqueViolation(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	return user, nil
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(ctx context.Context, id string, newPasswordHash string) error {
	query := `
		UPDATE auth.users
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1
	`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, id, newPasswordHash)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return ErrUserNotFound
		}

		return nil
	})
}

// VerifyEmail marks a user's email as verified
func (r *UserRepository) VerifyEmail(ctx context.Context, id string) error {
	query := `
		UPDATE auth.users
		SET email_verified = true, updated_at = NOW()
		WHERE id = $1
	`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, id)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return ErrUserNotFound
		}

		return nil
	})
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM auth.users WHERE id = $1`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, id)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return ErrUserNotFound
		}

		return nil
	})
}

// List retrieves users with pagination
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*User, error) {
	query := `
		SELECT id, email, email_verified, role, user_metadata, app_metadata, created_at, updated_at
		FROM auth.users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	users := []*User{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			user := &User{}
			err := rows.Scan(
				&user.ID,
				&user.Email,
				&user.EmailVerified,
				&user.Role,
				&user.UserMetadata,
				&user.AppMetadata,
				&user.CreatedAt,
				&user.UpdatedAt,
			)
			if err != nil {
				return err
			}
			users = append(users, user)
		}

		return rows.Err()
	})

	return users, err
}

// Count returns the total number of users
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM auth.users`

	var count int
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query).Scan(&count)
	})
	return count, err
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// Helper function to format SQL placeholder
func formatPlaceholder(column string, argNum int) string {
	return fmt.Sprintf("%s = $%d", column, argNum)
}

// CreateInTable creates a new user in the specified table (auth.users or dashboard.users)
func (r *UserRepository) CreateInTable(ctx context.Context, req CreateUserRequest, passwordHash string, userType string) (*User, error) {
	user := &User{
		ID:            uuid.New().String(),
		Email:         req.Email,
		PasswordHash:  passwordHash,
		EmailVerified: false,
		Role:          req.Role,
		UserMetadata:  req.UserMetadata,
		AppMetadata:   req.AppMetadata,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Set default role if not provided
	if user.Role == "" {
		if userType == "dashboard" {
			user.Role = "admin"
		} else {
			user.Role = "authenticated"
		}
	}

	// Determine which table to use
	tableName := "auth.users"
	if userType == "dashboard" {
		tableName = "dashboard.users"
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (id, email, password_hash, email_verified, role, user_metadata, app_metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, email, email_verified, role, user_metadata, app_metadata, created_at, updated_at
	`, tableName)

	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			query,
			user.ID,
			user.Email,
			user.PasswordHash,
			user.EmailVerified,
			user.Role,
			user.UserMetadata,
			user.AppMetadata,
			user.CreatedAt,
			user.UpdatedAt,
		).Scan(
			&user.ID,
			&user.Email,
			&user.EmailVerified,
			&user.Role,
			&user.UserMetadata,
			&user.AppMetadata,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
	})

	if err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateInTable updates a user in the specified table
func (r *UserRepository) UpdateInTable(ctx context.Context, id string, req UpdateUserRequest, userType string) (*User, error) {
	// Determine which table to use
	tableName := "auth.users"
	if userType == "dashboard" {
		tableName = "dashboard.users"
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{id}
	argNum := 2

	if req.Email != nil {
		updates = append(updates, formatPlaceholder("email", argNum))
		args = append(args, *req.Email)
		argNum++
	}

	if req.EmailVerified != nil {
		updates = append(updates, formatPlaceholder("email_verified", argNum))
		args = append(args, *req.EmailVerified)
		argNum++
	}

	if req.Role != nil {
		updates = append(updates, formatPlaceholder("role", argNum))
		args = append(args, *req.Role)
		argNum++
	}

	if req.UserMetadata != nil {
		updates = append(updates, formatPlaceholder("user_metadata", argNum))
		args = append(args, req.UserMetadata)
		argNum++
	}

	if req.AppMetadata != nil {
		updates = append(updates, formatPlaceholder("app_metadata", argNum))
		args = append(args, req.AppMetadata)
		argNum++
	}

	// Always update updated_at
	updates = append(updates, formatPlaceholder("updated_at", argNum))
	args = append(args, time.Now())

	if len(updates) == 1 { // Only updated_at
		return nil, fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET %s
		WHERE id = $1
		RETURNING id, email, email_verified, role, user_metadata, app_metadata, created_at, updated_at
	`, tableName, joinStrings(updates, ", "))

	user := &User{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, args...).Scan(
			&user.ID,
			&user.Email,
			&user.EmailVerified,
			&user.Role,
			&user.UserMetadata,
			&user.AppMetadata,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
	})

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// UpdatePasswordInTable updates a user's password in the specified table
func (r *UserRepository) UpdatePasswordInTable(ctx context.Context, id string, newPasswordHash string, userType string) error {
	// Determine which table to use
	tableName := "auth.users"
	if userType == "dashboard" {
		tableName = "dashboard.users"
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1
	`, tableName)

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, id, newPasswordHash)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return ErrUserNotFound
		}

		return nil
	})
}

// DeleteFromTable deletes a user from the specified table
func (r *UserRepository) DeleteFromTable(ctx context.Context, id string, userType string) error {
	// Determine which table to use
	tableName := "auth.users"
	if userType == "dashboard" {
		tableName = "dashboard.users"
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, tableName)

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, id)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return ErrUserNotFound
		}

		return nil
	})
}

// GetByIDFromTable retrieves a user by ID from the specified table
func (r *UserRepository) GetByIDFromTable(ctx context.Context, id string, userType string) (*User, error) {
	// Determine which table to use
	tableName := "auth.users"
	if userType == "dashboard" {
		tableName = "dashboard.users"
	}

	query := fmt.Sprintf(`
		SELECT id, email, email_verified, role, user_metadata, app_metadata, created_at, updated_at
		FROM %s
		WHERE id = $1
	`, tableName)

	user := &User{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, id).Scan(
			&user.ID,
			&user.Email,
			&user.EmailVerified,
			&user.Role,
			&user.UserMetadata,
			&user.AppMetadata,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
	})

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}
