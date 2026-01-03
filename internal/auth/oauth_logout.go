package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrOAuthLogoutStateNotFound is returned when a logout state is not found
	ErrOAuthLogoutStateNotFound = errors.New("oauth logout state not found")
	// ErrOAuthLogoutStateExpired is returned when a logout state has expired
	ErrOAuthLogoutStateExpired = errors.New("oauth logout state expired")
	// ErrOAuthTokenNotFound is returned when no OAuth token is found for the user/provider
	ErrOAuthTokenNotFound = errors.New("oauth token not found")
	// ErrOAuthProviderNoSLO is returned when the provider doesn't support logout
	ErrOAuthProviderNoSLO = errors.New("oauth provider does not support single logout")
)

// OAuthLogoutState represents a stored logout state for CSRF protection
type OAuthLogoutState struct {
	ID                    string
	UserID                string
	Provider              string
	State                 string
	PostLogoutRedirectURI string
	CreatedAt             time.Time
	ExpiresAt             time.Time
}

// StoredOAuthToken represents an OAuth token stored in the database
type StoredOAuthToken struct {
	ID           string
	UserID       string
	Provider     string
	AccessToken  string
	RefreshToken string
	IDToken      string
	TokenExpiry  time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// OAuthLogoutResult contains the result of an OAuth logout operation
type OAuthLogoutResult struct {
	// LocalLogoutComplete indicates JWT tokens were revoked
	LocalLogoutComplete bool `json:"local_tokens_revoked"`
	// ProviderTokenRevoked indicates the token was revoked at the provider
	ProviderTokenRevoked bool `json:"provider_token_revoked"`
	// RequiresRedirect indicates the user should be redirected for OIDC logout
	RequiresRedirect bool `json:"requires_redirect,omitempty"`
	// RedirectURL is the URL to redirect to for OIDC logout
	RedirectURL string `json:"redirect_url,omitempty"`
	// Provider is the name of the OAuth provider
	Provider string `json:"provider"`
	// Warning contains any warning message
	Warning string `json:"warning,omitempty"`
}

// OAuthLogoutService handles OAuth Single Logout operations
type OAuthLogoutService struct {
	db            *pgxpool.Pool
	encryptionKey string
	httpClient    *http.Client
}

// NewOAuthLogoutService creates a new OAuth logout service
func NewOAuthLogoutService(db *pgxpool.Pool, encryptionKey string) *OAuthLogoutService {
	return &OAuthLogoutService{
		db:            db,
		encryptionKey: encryptionKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GenerateLogoutState generates a random state for CSRF protection during logout
func GenerateLogoutState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// StoreLogoutState stores a logout state for CSRF protection
func (s *OAuthLogoutService) StoreLogoutState(ctx context.Context, userID, provider, state, postLogoutRedirectURI string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO auth.oauth_logout_states (user_id, provider, state, post_logout_redirect_uri)
		VALUES ($1, $2, $3, $4)
	`, userID, provider, state, postLogoutRedirectURI)
	return err
}

// ValidateLogoutState validates and consumes a logout state
func (s *OAuthLogoutService) ValidateLogoutState(ctx context.Context, state string) (*OAuthLogoutState, error) {
	var logoutState OAuthLogoutState

	// Use a transaction to atomically read and delete
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx, `
		SELECT id, user_id, provider, state, post_logout_redirect_uri, created_at, expires_at
		FROM auth.oauth_logout_states
		WHERE state = $1
	`, state).Scan(
		&logoutState.ID,
		&logoutState.UserID,
		&logoutState.Provider,
		&logoutState.State,
		&logoutState.PostLogoutRedirectURI,
		&logoutState.CreatedAt,
		&logoutState.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOAuthLogoutStateNotFound
		}
		return nil, err
	}

	// Delete the state (one-time use)
	_, err = tx.Exec(ctx, `DELETE FROM auth.oauth_logout_states WHERE id = $1`, logoutState.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Check if expired
	if time.Now().After(logoutState.ExpiresAt) {
		return nil, ErrOAuthLogoutStateExpired
	}

	return &logoutState, nil
}

// GetUserOAuthToken retrieves the user's stored OAuth token for a provider
func (s *OAuthLogoutService) GetUserOAuthToken(ctx context.Context, userID, provider string) (*StoredOAuthToken, error) {
	var token StoredOAuthToken
	var refreshToken, idToken *string
	var tokenExpiry *time.Time

	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, provider, access_token, refresh_token, id_token, token_expiry, created_at, updated_at
		FROM auth.oauth_tokens
		WHERE user_id = $1 AND provider = $2
	`, userID, provider).Scan(
		&token.ID,
		&token.UserID,
		&token.Provider,
		&token.AccessToken,
		&refreshToken,
		&idToken,
		&tokenExpiry,
		&token.CreatedAt,
		&token.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOAuthTokenNotFound
		}
		return nil, err
	}

	if refreshToken != nil {
		token.RefreshToken = *refreshToken
	}
	if idToken != nil {
		token.IDToken = *idToken
	}
	if tokenExpiry != nil {
		token.TokenExpiry = *tokenExpiry
	}

	return &token, nil
}

// DeleteUserOAuthToken removes the user's OAuth token after logout
func (s *OAuthLogoutService) DeleteUserOAuthToken(ctx context.Context, userID, provider string) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM auth.oauth_tokens
		WHERE user_id = $1 AND provider = $2
	`, userID, provider)
	return err
}

// CleanupExpiredLogoutStates removes expired logout states
func (s *OAuthLogoutService) CleanupExpiredLogoutStates(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM auth.oauth_logout_states
		WHERE expires_at < NOW()
	`)
	return err
}

// RevokeTokenAtProvider revokes the OAuth token at the provider using RFC 7009
func (s *OAuthLogoutService) RevokeTokenAtProvider(ctx context.Context, revocationEndpoint, token, tokenTypeHint, clientID, clientSecret string) error {
	if revocationEndpoint == "" {
		return ErrOAuthProviderNoSLO
	}

	// Build form data for revocation request
	data := url.Values{}
	data.Set("token", token)
	if tokenTypeHint != "" {
		data.Set("token_type_hint", tokenTypeHint)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, revocationEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create revocation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add client credentials (Basic auth or form-based)
	if clientID != "" && clientSecret != "" {
		req.SetBasicAuth(clientID, clientSecret)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send revocation request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// RFC 7009: The authorization server responds with HTTP status code 200 if the token
	// has been revoked successfully or if the client submitted an invalid token.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token revocation failed with status %d", resp.StatusCode)
	}

	return nil
}

// GenerateOIDCLogoutURL generates the OIDC RP-Initiated Logout URL
func (s *OAuthLogoutService) GenerateOIDCLogoutURL(endSessionEndpoint, idToken, postLogoutRedirectURI, state string) (string, error) {
	if endSessionEndpoint == "" {
		return "", ErrOAuthProviderNoSLO
	}

	logoutURL, err := url.Parse(endSessionEndpoint)
	if err != nil {
		return "", fmt.Errorf("invalid end_session_endpoint: %w", err)
	}

	q := logoutURL.Query()

	// Add id_token_hint if we have an ID token
	if idToken != "" {
		q.Set("id_token_hint", idToken)
	}

	// Add post_logout_redirect_uri if specified
	if postLogoutRedirectURI != "" {
		q.Set("post_logout_redirect_uri", postLogoutRedirectURI)
	}

	// Add state for CSRF protection
	if state != "" {
		q.Set("state", state)
	}

	logoutURL.RawQuery = q.Encode()
	return logoutURL.String(), nil
}

// GetDefaultRevocationEndpoint returns the default revocation endpoint for known providers
func GetDefaultRevocationEndpoint(provider OAuthProvider) string {
	switch provider {
	case ProviderGoogle:
		return "https://oauth2.googleapis.com/revoke"
	case ProviderMicrosoft:
		// Microsoft uses the same logout endpoint for token revocation
		return ""
	case ProviderApple:
		return "https://appleid.apple.com/auth/revoke"
	case ProviderGitLab:
		return "https://gitlab.com/oauth/revoke"
	case ProviderGithub:
		// GitHub doesn't support token revocation via standard endpoint
		return ""
	case ProviderFacebook:
		// Facebook uses a different mechanism
		return ""
	case ProviderTwitter:
		return "https://api.twitter.com/2/oauth2/revoke"
	case ProviderLinkedIn:
		// LinkedIn doesn't support standard token revocation
		return ""
	case ProviderBitbucket:
		// Bitbucket doesn't support standard token revocation
		return ""
	default:
		return ""
	}
}

// GetDefaultEndSessionEndpoint returns the default OIDC end_session_endpoint for known providers
func GetDefaultEndSessionEndpoint(provider OAuthProvider) string {
	switch provider {
	case ProviderGoogle:
		return "https://accounts.google.com/o/oauth2/logout"
	case ProviderMicrosoft:
		return "https://login.microsoftonline.com/common/oauth2/v2.0/logout"
	case ProviderApple:
		// Apple doesn't support OIDC RP-Initiated Logout
		return ""
	case ProviderGitLab:
		// GitLab supports OIDC logout
		return "https://gitlab.com/oauth/logout"
	case ProviderGithub:
		// GitHub doesn't support OIDC logout
		return ""
	case ProviderFacebook:
		// Facebook doesn't support OIDC logout
		return ""
	case ProviderTwitter:
		// Twitter doesn't support OIDC logout
		return ""
	case ProviderLinkedIn:
		// LinkedIn doesn't support OIDC logout
		return ""
	case ProviderBitbucket:
		// Bitbucket doesn't support OIDC logout
		return ""
	default:
		return ""
	}
}
