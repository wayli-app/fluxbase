package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wayli-app/fluxbase/internal/database"
)

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrUserAlreadyExists is returned when trying to create a user with existing email
	ErrUserAlreadyExists = errors.New("user with this email already exists")
	// ErrInvalidCredentials is returned when login credentials are invalid
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// User represents a user in the system
type User struct {
	ID            string    `json:"id" db:"id"`
	Email         string    `json:"email" db:"email"`
	PasswordHash  string    `json:"-" db:"password_hash"` // Never expose in JSON
	EmailVerified bool      `json:"email_verified" db:"email_verified"`
	Role          string    `json:"role,omitempty" db:"role"`
	Metadata      any       `json:"metadata,omitempty" db:"metadata"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
	Metadata any    `json:"metadata,omitempty"`
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	Email         *string `json:"email,omitempty"`
	EmailVerified *bool   `json:"email_verified,omitempty"`
	Role          *string `json:"role,omitempty"`
	Metadata      any     `json:"metadata,omitempty"`
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
		Metadata:      req.Metadata,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Set default role if not provided
	if user.Role == "" {
		user.Role = "authenticated"
	}

	query := `
		INSERT INTO auth.users (id, email, password_hash, email_verified, role, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, email, email_verified, role, metadata, created_at, updated_at
	`

	row := r.db.QueryRow(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.EmailVerified,
		user.Role,
		user.Metadata,
		user.CreatedAt,
		user.UpdatedAt,
	)

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.EmailVerified,
		&user.Role,
		&user.Metadata,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

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
		SELECT id, email, password_hash, email_verified, role, metadata, created_at, updated_at
		FROM auth.users
		WHERE id = $1
	`

	user := &User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.EmailVerified,
		&user.Role,
		&user.Metadata,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

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
		SELECT id, email, password_hash, email_verified, role, metadata, created_at, updated_at
		FROM auth.users
		WHERE email = $1
	`

	user := &User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.EmailVerified,
		&user.Role,
		&user.Metadata,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
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

	if req.Metadata != nil {
		updates = append(updates, formatPlaceholder("metadata", argCount))
		args = append(args, req.Metadata)
		// argCount++ // Not used after this point
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
		RETURNING id, email, email_verified, role, metadata, created_at, updated_at
	`

	user := &User{}
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&user.ID,
		&user.Email,
		&user.EmailVerified,
		&user.Role,
		&user.Metadata,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

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

	result, err := r.db.Exec(ctx, query, id, newPasswordHash)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// VerifyEmail marks a user's email as verified
func (r *UserRepository) VerifyEmail(ctx context.Context, id string) error {
	query := `
		UPDATE auth.users
		SET email_verified = true, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM auth.users WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// List retrieves users with pagination
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*User, error) {
	query := `
		SELECT id, email, email_verified, role, metadata, created_at, updated_at
		FROM auth.users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*User{}
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.EmailVerified,
			&user.Role,
			&user.Metadata,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// Count returns the total number of users
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM auth.users`

	var count int
	err := r.db.QueryRow(ctx, query).Scan(&count)
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
