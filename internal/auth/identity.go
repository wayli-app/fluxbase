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
	// ErrIdentityNotFound is returned when an identity is not found
	ErrIdentityNotFound = errors.New("identity not found")
	// ErrIdentityAlreadyLinked is returned when trying to link an identity that's already linked
	ErrIdentityAlreadyLinked = errors.New("identity is already linked to another user")
)

// UserIdentity represents a linked OAuth identity
type UserIdentity struct {
	ID             string                 `json:"id" db:"id"`
	UserID         string                 `json:"user_id" db:"user_id"`
	Provider       string                 `json:"provider" db:"provider"`
	ProviderUserID string                 `json:"provider_user_id" db:"provider_user_id"`
	Email          *string                `json:"email,omitempty" db:"email"`
	IdentityData   map[string]interface{} `json:"identity_data,omitempty" db:"metadata"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// IdentityRepository handles database operations for user identities
type IdentityRepository struct {
	db *database.Connection
}

// NewIdentityRepository creates a new identity repository
func NewIdentityRepository(db *database.Connection) *IdentityRepository {
	return &IdentityRepository{db: db}
}

// GetByUserID retrieves all identities for a user
func (r *IdentityRepository) GetByUserID(ctx context.Context, userID string) ([]UserIdentity, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, email, metadata, created_at, updated_at
		FROM auth.oauth_links
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	var identities []UserIdentity
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, userID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var identity UserIdentity
			err := rows.Scan(
				&identity.ID,
				&identity.UserID,
				&identity.Provider,
				&identity.ProviderUserID,
				&identity.Email,
				&identity.IdentityData,
				&identity.CreatedAt,
				&identity.UpdatedAt,
			)
			if err != nil {
				return err
			}
			identities = append(identities, identity)
		}

		return rows.Err()
	})

	return identities, err
}

// GetByID retrieves an identity by ID
func (r *IdentityRepository) GetByID(ctx context.Context, id string) (*UserIdentity, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, email, metadata, created_at, updated_at
		FROM auth.oauth_links
		WHERE id = $1
	`

	var identity UserIdentity
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, id).Scan(
			&identity.ID,
			&identity.UserID,
			&identity.Provider,
			&identity.ProviderUserID,
			&identity.Email,
			&identity.IdentityData,
			&identity.CreatedAt,
			&identity.UpdatedAt,
		)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrIdentityNotFound
		}
		return nil, err
	}

	return &identity, nil
}

// GetByProviderAndUserID retrieves an identity by provider and provider user ID
func (r *IdentityRepository) GetByProviderAndUserID(ctx context.Context, provider, providerUserID string) (*UserIdentity, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, email, metadata, created_at, updated_at
		FROM auth.oauth_links
		WHERE provider = $1 AND provider_user_id = $2
	`

	var identity UserIdentity
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, provider, providerUserID).Scan(
			&identity.ID,
			&identity.UserID,
			&identity.Provider,
			&identity.ProviderUserID,
			&identity.Email,
			&identity.IdentityData,
			&identity.CreatedAt,
			&identity.UpdatedAt,
		)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrIdentityNotFound
		}
		return nil, err
	}

	return &identity, nil
}

// Create creates a new identity link
func (r *IdentityRepository) Create(ctx context.Context, userID, provider, providerUserID string, email *string, metadata map[string]interface{}) (*UserIdentity, error) {
	identity := &UserIdentity{
		ID:             uuid.New().String(),
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
		Email:          email,
		IdentityData:   metadata,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	query := `
		INSERT INTO auth.oauth_links (id, user_id, provider, provider_user_id, email, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, provider, provider_user_id, email, metadata, created_at, updated_at
	`

	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			identity.ID,
			identity.UserID,
			identity.Provider,
			identity.ProviderUserID,
			identity.Email,
			identity.IdentityData,
			identity.CreatedAt,
			identity.UpdatedAt,
		).Scan(
			&identity.ID,
			&identity.UserID,
			&identity.Provider,
			&identity.ProviderUserID,
			&identity.Email,
			&identity.IdentityData,
			&identity.CreatedAt,
			&identity.UpdatedAt,
		)
	})

	if err != nil {
		return nil, err
	}

	return identity, nil
}

// Delete deletes an identity by ID
func (r *IdentityRepository) Delete(ctx context.Context, id, userID string) error {
	query := `
		DELETE FROM auth.oauth_links
		WHERE id = $1 AND user_id = $2
	`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, id, userID)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return ErrIdentityNotFound
		}

		return nil
	})
}

// DeleteByProvider deletes all identities for a user and provider
func (r *IdentityRepository) DeleteByProvider(ctx context.Context, userID, provider string) error {
	query := `
		DELETE FROM auth.oauth_links
		WHERE user_id = $1 AND provider = $2
	`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, userID, provider)
		return err
	})
}

// IdentityService provides identity management functionality
type IdentityService struct {
	repo         *IdentityRepository
	oauthManager *OAuthManager
	stateStore   *StateStore
}

// NewIdentityService creates a new identity service
func NewIdentityService(
	repo *IdentityRepository,
	oauthManager *OAuthManager,
	stateStore *StateStore,
) *IdentityService {
	return &IdentityService{
		repo:         repo,
		oauthManager: oauthManager,
		stateStore:   stateStore,
	}
}

// GetUserIdentities retrieves all OAuth identities linked to a user
func (s *IdentityService) GetUserIdentities(ctx context.Context, userID string) ([]UserIdentity, error) {
	identities, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user identities: %w", err)
	}

	return identities, nil
}

// LinkIdentityProvider initiates OAuth flow to link a new provider
func (s *IdentityService) LinkIdentityProvider(ctx context.Context, userID string, provider string) (string, string, error) {
	// Generate OAuth state for CSRF protection
	state, err := GenerateState()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state with user ID embedded (you might want to use a proper state store with user context)
	s.stateStore.Set(state)

	// Get OAuth authorization URL
	authURL, err := s.oauthManager.GetAuthURL(OAuthProvider(provider), state)
	if err != nil {
		return "", "", fmt.Errorf("failed to get auth URL: %w", err)
	}

	return authURL, state, nil
}

// LinkIdentityCallback handles the OAuth callback to complete identity linking
func (s *IdentityService) LinkIdentityCallback(ctx context.Context, userID, provider, code, state string) (*UserIdentity, error) {
	// Validate state
	if !s.stateStore.Validate(state) {
		return nil, ErrInvalidState
	}

	// Exchange code for token
	token, err := s.oauthManager.ExchangeCode(ctx, OAuthProvider(provider), code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info from provider
	userInfo, err := s.oauthManager.GetUserInfo(ctx, OAuthProvider(provider), token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Extract provider user ID and email
	providerUserID := fmt.Sprintf("%v", userInfo["id"])
	var email *string
	if emailVal, ok := userInfo["email"].(string); ok && emailVal != "" {
		email = &emailVal
	}

	// Check if this provider identity is already linked to another user
	existingIdentity, err := s.repo.GetByProviderAndUserID(ctx, provider, providerUserID)
	if err != nil && !errors.Is(err, ErrIdentityNotFound) {
		return nil, err
	}

	if existingIdentity != nil && existingIdentity.UserID != userID {
		return nil, ErrIdentityAlreadyLinked
	}

	// If already linked to this user, return existing identity
	if existingIdentity != nil {
		return existingIdentity, nil
	}

	// Create new identity link
	identity, err := s.repo.Create(ctx, userID, provider, providerUserID, email, userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity link: %w", err)
	}

	return identity, nil
}

// UnlinkIdentity removes an OAuth identity from a user
func (s *IdentityService) UnlinkIdentity(ctx context.Context, userID, identityID string) error {
	// Verify the identity belongs to the user and delete it
	err := s.repo.Delete(ctx, identityID, userID)
	if err != nil {
		if errors.Is(err, ErrIdentityNotFound) {
			return ErrIdentityNotFound
		}
		return fmt.Errorf("failed to unlink identity: %w", err)
	}

	return nil
}
