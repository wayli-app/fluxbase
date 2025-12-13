package auth

import (
	"context"
	"errors"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	// ErrSessionNotFound is returned when a session is not found
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionExpired is returned when a session has expired
	ErrSessionExpired = errors.New("session has expired")
)

// Session represents a user session
type Session struct {
	ID           string    `json:"id" db:"id"`
	UserID       string    `json:"user_id" db:"user_id"`
	AccessToken  string    `json:"access_token" db:"access_token"`
	RefreshToken string    `json:"refresh_token" db:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// SessionRepository handles database operations for sessions
type SessionRepository struct {
	db *database.Connection
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *database.Connection) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session
func (r *SessionRepository) Create(ctx context.Context, userID, accessToken, refreshToken string, expiresAt time.Time) (*Session, error) {
	session := &Session{
		ID:           uuid.New().String(),
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
	}

	query := `
		INSERT INTO auth.sessions (id, user_id, access_token, refresh_token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, access_token, refresh_token, expires_at, created_at
	`

	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			session.ID,
			session.UserID,
			session.AccessToken,
			session.RefreshToken,
			session.ExpiresAt,
			session.CreatedAt,
		).Scan(
			&session.ID,
			&session.UserID,
			&session.AccessToken,
			&session.RefreshToken,
			&session.ExpiresAt,
			&session.CreatedAt,
		)
	})

	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetByAccessToken retrieves a session by access token
func (r *SessionRepository) GetByAccessToken(ctx context.Context, accessToken string) (*Session, error) {
	query := `
		SELECT id, user_id, access_token, refresh_token, expires_at, created_at
		FROM auth.sessions
		WHERE access_token = $1
	`

	session := &Session{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, accessToken).Scan(
			&session.ID,
			&session.UserID,
			&session.AccessToken,
			&session.RefreshToken,
			&session.ExpiresAt,
			&session.CreatedAt,
		)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	return session, nil
}

// GetByRefreshToken retrieves a session by refresh token
func (r *SessionRepository) GetByRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	query := `
		SELECT id, user_id, access_token, refresh_token, expires_at, created_at
		FROM auth.sessions
		WHERE refresh_token = $1
	`

	session := &Session{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, refreshToken).Scan(
			&session.ID,
			&session.UserID,
			&session.AccessToken,
			&session.RefreshToken,
			&session.ExpiresAt,
			&session.CreatedAt,
		)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	return session, nil
}

// GetByUserID retrieves all sessions for a user
func (r *SessionRepository) GetByUserID(ctx context.Context, userID string) ([]*Session, error) {
	query := `
		SELECT id, user_id, access_token, refresh_token, expires_at, created_at
		FROM auth.sessions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	sessions := []*Session{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, userID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			session := &Session{}
			err := rows.Scan(
				&session.ID,
				&session.UserID,
				&session.AccessToken,
				&session.RefreshToken,
				&session.ExpiresAt,
				&session.CreatedAt,
			)
			if err != nil {
				return err
			}

			// Skip expired sessions
			if time.Now().Before(session.ExpiresAt) {
				sessions = append(sessions, session)
			}
		}

		return rows.Err()
	})

	return sessions, err
}

// UpdateTokens updates the tokens for a session
func (r *SessionRepository) UpdateTokens(ctx context.Context, id, accessToken, refreshToken string, expiresAt time.Time) error {
	query := `
		UPDATE auth.sessions
		SET access_token = $2, refresh_token = $3, expires_at = $4
		WHERE id = $1
	`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, id, accessToken, refreshToken, expiresAt)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return ErrSessionNotFound
		}

		return nil
	})
}

// UpdateAccessToken updates only the access token
func (r *SessionRepository) UpdateAccessToken(ctx context.Context, id, accessToken string) error {
	query := `
		UPDATE auth.sessions
		SET access_token = $2
		WHERE id = $1
	`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, id, accessToken)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return ErrSessionNotFound
		}

		return nil
	})
}

// Delete deletes a session by ID
func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM auth.sessions WHERE id = $1`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, id)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return ErrSessionNotFound
		}

		return nil
	})
}

// DeleteByAccessToken deletes a session by access token
func (r *SessionRepository) DeleteByAccessToken(ctx context.Context, accessToken string) error {
	query := `DELETE FROM auth.sessions WHERE access_token = $1`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, accessToken)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return ErrSessionNotFound
		}

		return nil
	})
}

// DeleteByUserID deletes all sessions for a user
func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	query := `DELETE FROM auth.sessions WHERE user_id = $1`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, userID)
		return err
	})
}

// DeleteExpired deletes all expired sessions
func (r *SessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM auth.sessions WHERE expires_at < NOW()`

	var rowsAffected int64
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query)
		if err != nil {
			return err
		}

		rowsAffected = result.RowsAffected()
		return nil
	})

	return rowsAffected, err
}

// Count returns the total number of active sessions
func (r *SessionRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM auth.sessions WHERE expires_at > NOW()`

	var count int
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query).Scan(&count)
	})
	return count, err
}

// CountByUserID returns the number of active sessions for a user
func (r *SessionRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM auth.sessions WHERE user_id = $1 AND expires_at > NOW()`

	var count int
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, userID).Scan(&count)
	})
	return count, err
}

// SessionWithUser represents a session with user info for admin views
type SessionWithUser struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UserEmail *string   `json:"user_email,omitempty"`
}

// ListAll returns all sessions with user info for admin views
func (r *SessionRepository) ListAll(ctx context.Context, includeExpired bool) ([]SessionWithUser, error) {
	query := `
		SELECT s.id, s.user_id, s.expires_at, s.created_at, u.email
		FROM auth.sessions s
		LEFT JOIN auth.users u ON s.user_id = u.id
	`
	if !includeExpired {
		query += ` WHERE s.expires_at > NOW()`
	}
	query += ` ORDER BY s.created_at DESC`

	sessions := []SessionWithUser{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var session SessionWithUser
			err := rows.Scan(
				&session.ID,
				&session.UserID,
				&session.ExpiresAt,
				&session.CreatedAt,
				&session.UserEmail,
			)
			if err != nil {
				return err
			}
			sessions = append(sessions, session)
		}

		return rows.Err()
	})

	return sessions, err
}

// ListAllPaginated returns paginated sessions with user info for admin views
func (r *SessionRepository) ListAllPaginated(ctx context.Context, includeExpired bool, limit, offset int) ([]SessionWithUser, int, error) {
	// Build the WHERE clause
	whereClause := ""
	if !includeExpired {
		whereClause = " WHERE s.expires_at > NOW()"
	}

	// Count total
	countQuery := `SELECT COUNT(*) FROM auth.sessions s` + whereClause
	var total int

	// Query with pagination
	query := `
		SELECT s.id, s.user_id, s.expires_at, s.created_at, u.email
		FROM auth.sessions s
		LEFT JOIN auth.users u ON s.user_id = u.id
	` + whereClause + ` ORDER BY s.created_at DESC LIMIT $1 OFFSET $2`

	sessions := []SessionWithUser{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		// Get total count
		if err := tx.QueryRow(ctx, countQuery).Scan(&total); err != nil {
			return err
		}

		// Get paginated results
		rows, err := tx.Query(ctx, query, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var session SessionWithUser
			err := rows.Scan(
				&session.ID,
				&session.UserID,
				&session.ExpiresAt,
				&session.CreatedAt,
				&session.UserEmail,
			)
			if err != nil {
				return err
			}
			sessions = append(sessions, session)
		}

		return rows.Err()
	})

	return sessions, total, err
}
