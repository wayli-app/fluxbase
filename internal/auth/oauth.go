package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

var (
	// ErrInvalidProvider is returned when an OAuth provider is not supported
	ErrInvalidProvider = errors.New("invalid OAuth provider")
	// ErrInvalidState is returned when OAuth state doesn't match
	ErrInvalidState = errors.New("invalid OAuth state")
)

// OAuthProvider represents different OAuth providers
type OAuthProvider string

const (
	// ProviderGoogle represents Google OAuth
	ProviderGoogle OAuthProvider = "google"
	// ProviderGithub represents GitHub OAuth
	ProviderGithub OAuthProvider = "github"
	// ProviderMicrosoft represents Microsoft OAuth
	ProviderMicrosoft OAuthProvider = "microsoft"
	// ProviderApple represents Apple OAuth
	ProviderApple OAuthProvider = "apple"
	// ProviderFacebook represents Facebook OAuth
	ProviderFacebook OAuthProvider = "facebook"
	// ProviderTwitter represents Twitter OAuth
	ProviderTwitter OAuthProvider = "twitter"
	// ProviderLinkedIn represents LinkedIn OAuth
	ProviderLinkedIn OAuthProvider = "linkedin"
	// ProviderGitLab represents GitLab OAuth
	ProviderGitLab OAuthProvider = "gitlab"
	// ProviderBitbucket represents Bitbucket OAuth
	ProviderBitbucket OAuthProvider = "bitbucket"
)

// OAuthConfig holds OAuth provider configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// OAuthManager handles OAuth authentication flows
type OAuthManager struct {
	configs map[OAuthProvider]*oauth2.Config
}

// NewOAuthManager creates a new OAuth manager
func NewOAuthManager() *OAuthManager {
	return &OAuthManager{
		configs: make(map[OAuthProvider]*oauth2.Config),
	}
}

// RegisterProvider registers an OAuth provider
func (m *OAuthManager) RegisterProvider(provider OAuthProvider, config OAuthConfig) error {
	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       config.Scopes,
		Endpoint:     m.GetEndpoint(provider),
	}

	m.configs[provider] = oauth2Config
	return nil
}

// GetEndpoint returns the OAuth2 endpoint for a provider
func (m *OAuthManager) GetEndpoint(provider OAuthProvider) oauth2.Endpoint {
	switch provider {
	case ProviderGoogle:
		return google.Endpoint
	case ProviderGithub:
		return github.Endpoint
	case ProviderMicrosoft:
		return oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		}
	case ProviderApple:
		return oauth2.Endpoint{
			AuthURL:  "https://appleid.apple.com/auth/authorize",
			TokenURL: "https://appleid.apple.com/auth/token",
		}
	case ProviderFacebook:
		return oauth2.Endpoint{
			AuthURL:  "https://www.facebook.com/v12.0/dialog/oauth",
			TokenURL: "https://graph.facebook.com/v12.0/oauth/access_token",
		}
	case ProviderTwitter:
		return oauth2.Endpoint{
			AuthURL:  "https://twitter.com/i/oauth2/authorize",
			TokenURL: "https://api.twitter.com/2/oauth2/token",
		}
	case ProviderLinkedIn:
		return oauth2.Endpoint{
			AuthURL:  "https://www.linkedin.com/oauth/v2/authorization",
			TokenURL: "https://www.linkedin.com/oauth/v2/accessToken",
		}
	case ProviderGitLab:
		return oauth2.Endpoint{
			AuthURL:  "https://gitlab.com/oauth/authorize",
			TokenURL: "https://gitlab.com/oauth/token",
		}
	case ProviderBitbucket:
		return oauth2.Endpoint{
			AuthURL:  "https://bitbucket.org/site/oauth2/authorize",
			TokenURL: "https://bitbucket.org/site/oauth2/access_token",
		}
	default:
		return oauth2.Endpoint{}
	}
}

// GetAuthURL returns the OAuth authorization URL
func (m *OAuthManager) GetAuthURL(provider OAuthProvider, state string) (string, error) {
	config, ok := m.configs[provider]
	if !ok {
		return "", ErrInvalidProvider
	}

	return config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// ExchangeCode exchanges an authorization code for tokens
func (m *OAuthManager) ExchangeCode(ctx context.Context, provider OAuthProvider, code string) (*oauth2.Token, error) {
	config, ok := m.configs[provider]
	if !ok {
		return nil, ErrInvalidProvider
	}

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return token, nil
}

// GetUserInfo retrieves user information from the OAuth provider
func (m *OAuthManager) GetUserInfo(ctx context.Context, provider OAuthProvider, token *oauth2.Token) (map[string]interface{}, error) {
	config, ok := m.configs[provider]
	if !ok {
		return nil, ErrInvalidProvider
	}

	client := config.Client(ctx, token)

	// Get user info from provider-specific endpoint
	userInfoURL := m.GetUserInfoURL(provider)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return userInfo, nil
}

// GetUserInfoURL returns the user info endpoint for a provider
func (m *OAuthManager) GetUserInfoURL(provider OAuthProvider) string {
	switch provider {
	case ProviderGoogle:
		return "https://www.googleapis.com/oauth2/v2/userinfo"
	case ProviderGithub:
		return "https://api.github.com/user"
	case ProviderMicrosoft:
		return "https://graph.microsoft.com/v1.0/me"
	case ProviderApple:
		return "https://appleid.apple.com/auth/keys"
	case ProviderFacebook:
		return "https://graph.facebook.com/me?fields=id,name,email,picture"
	case ProviderTwitter:
		return "https://api.twitter.com/2/users/me"
	case ProviderLinkedIn:
		return "https://api.linkedin.com/v2/me"
	case ProviderGitLab:
		return "https://gitlab.com/api/v4/user"
	case ProviderBitbucket:
		return "https://api.bitbucket.org/2.0/user"
	default:
		return ""
	}
}

// GenerateState generates a random state parameter for CSRF protection
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// StateMetadata holds metadata associated with an OAuth state
type StateMetadata struct {
	Expiry       time.Time
	RedirectURI  string // Optional custom redirect URI for this OAuth flow
	Provider     string // OAuth provider name
	CodeVerifier string // PKCE code verifier
	Nonce        string // OpenID Connect nonce
}

// StateStore manages OAuth state tokens for CSRF protection
// Uses a mutex to protect concurrent access from multiple goroutines
type StateStore struct {
	mu     sync.RWMutex
	states map[string]*StateMetadata
}

// NewStateStore creates a new state store
func NewStateStore() *StateStore {
	return &StateStore{
		states: make(map[string]*StateMetadata),
	}
}

// Set stores a state token with optional redirect URI
func (s *StateStore) Set(state string, redirectURI ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	metadata := &StateMetadata{
		Expiry: time.Now().Add(10 * time.Minute),
	}

	// Store redirect URI if provided
	if len(redirectURI) > 0 && redirectURI[0] != "" {
		metadata.RedirectURI = redirectURI[0]
	}

	s.states[state] = metadata
}

// Validate checks if a state token is valid and removes it
// Uses a full lock since we both read and delete atomically
func (s *StateStore) Validate(state string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	metadata, exists := s.states[state]
	if !exists {
		return false
	}

	delete(s.states, state)

	// Check if expired
	return time.Now().Before(metadata.Expiry)
}

// GetAndValidate checks if a state token is valid, removes it, and returns the metadata
// Returns the metadata and a boolean indicating if the state was valid
func (s *StateStore) GetAndValidate(state string) (*StateMetadata, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	metadata, exists := s.states[state]
	if !exists {
		return nil, false
	}

	delete(s.states, state)

	// Check if expired
	if time.Now().After(metadata.Expiry) {
		return nil, false
	}

	return metadata, true
}

// Cleanup removes expired state tokens
func (s *StateStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for state, metadata := range s.states {
		if now.After(metadata.Expiry) {
			delete(s.states, state)
		}
	}
}

// SetWithMetadata stores a state token with full metadata (implements StateStorer)
func (s *StateStore) SetWithMetadata(ctx context.Context, state string, metadata StateMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if metadata.Expiry.IsZero() {
		metadata.Expiry = time.Now().Add(10 * time.Minute)
	}

	s.states[state] = &metadata
	return nil
}

// ValidateWithContext checks if a state token is valid (implements StateStorer)
func (s *StateStore) ValidateWithContext(ctx context.Context, state string) bool {
	return s.Validate(state)
}

// GetAndValidateWithContext validates and returns metadata (implements StateStorer)
func (s *StateStore) GetAndValidateWithContext(ctx context.Context, state string) (*StateMetadata, bool) {
	return s.GetAndValidate(state)
}

// CleanupWithContext removes expired state tokens (implements StateStorer)
func (s *StateStore) CleanupWithContext(ctx context.Context) error {
	s.Cleanup()
	return nil
}

// StateStorer is the interface for OAuth state storage
// Implementations can be in-memory (StateStore) or database-backed (DBStateStore)
type StateStorer interface {
	// Set stores a state token with optional metadata
	Set(ctx context.Context, state string, metadata StateMetadata) error
	// Validate checks if a state token is valid and removes it
	Validate(ctx context.Context, state string) bool
	// GetAndValidate validates and returns metadata, removing the state
	GetAndValidate(ctx context.Context, state string) (*StateMetadata, bool)
	// Cleanup removes expired state tokens
	Cleanup(ctx context.Context) error
}

// DBStateStoreConfig holds configuration for database-backed state storage
type DBStateStoreConfig struct {
	// DefaultTTL is the default time-to-live for state tokens (default: 10 minutes)
	DefaultTTL time.Duration
	// CleanupInterval is how often to run cleanup (default: 5 minutes)
	CleanupInterval time.Duration
}

// DefaultDBStateStoreConfig returns the default configuration
func DefaultDBStateStoreConfig() DBStateStoreConfig {
	return DBStateStoreConfig{
		DefaultTTL:      10 * time.Minute,
		CleanupInterval: 5 * time.Minute,
	}
}

// DBPool is the interface for database operations (subset of pgxpool.Pool)
type DBPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (interface{ RowsAffected() int64 }, error)
	QueryRow(ctx context.Context, sql string, args ...any) interface {
		Scan(dest ...any) error
	}
}

// DBStateStore provides database-backed OAuth state storage
// Supports multi-instance deployments where OAuth callback may hit different instance
type DBStateStore struct {
	db          DBPool
	config      DBStateStoreConfig
	stopCleanup chan struct{}
}

// NewDBStateStore creates a new database-backed state store
func NewDBStateStore(db DBPool, config DBStateStoreConfig) *DBStateStore {
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 10 * time.Minute
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}

	store := &DBStateStore{
		db:          db,
		config:      config,
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	go store.cleanupLoop()

	return store
}

// Stop stops the cleanup goroutine
func (s *DBStateStore) Stop() {
	close(s.stopCleanup)
}

// cleanupLoop periodically removes expired state tokens
func (s *DBStateStore) cleanupLoop() {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_ = s.Cleanup(ctx)
			cancel()
		case <-s.stopCleanup:
			return
		}
	}
}

// Set stores a state token with metadata in the database
func (s *DBStateStore) Set(ctx context.Context, state string, metadata StateMetadata) error {
	expiresAt := metadata.Expiry
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(s.config.DefaultTTL)
	}

	_, err := s.db.Exec(ctx, `
		INSERT INTO auth.oauth_states (state, provider, redirect_uri, code_verifier, nonce, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (state) DO UPDATE SET
			provider = EXCLUDED.provider,
			redirect_uri = EXCLUDED.redirect_uri,
			code_verifier = EXCLUDED.code_verifier,
			nonce = EXCLUDED.nonce,
			expires_at = EXCLUDED.expires_at
	`, state, metadata.Provider, metadata.RedirectURI, metadata.CodeVerifier, metadata.Nonce, expiresAt)

	return err
}

// Validate checks if a state token is valid and removes it
func (s *DBStateStore) Validate(ctx context.Context, state string) bool {
	result, err := s.db.Exec(ctx, `
		DELETE FROM auth.oauth_states
		WHERE state = $1 AND expires_at > NOW()
	`, state)

	if err != nil {
		return false
	}

	return result.RowsAffected() > 0
}

// GetAndValidate validates a state token, removes it, and returns the metadata
func (s *DBStateStore) GetAndValidate(ctx context.Context, state string) (*StateMetadata, bool) {
	var metadata StateMetadata
	var expiresAt time.Time

	// Use a transaction to atomically read and delete
	err := s.db.QueryRow(ctx, `
		DELETE FROM auth.oauth_states
		WHERE state = $1 AND expires_at > NOW()
		RETURNING provider, redirect_uri, code_verifier, nonce, expires_at
	`, state).Scan(&metadata.Provider, &metadata.RedirectURI, &metadata.CodeVerifier, &metadata.Nonce, &expiresAt)

	if err != nil {
		return nil, false
	}

	metadata.Expiry = expiresAt
	return &metadata, true
}

// Cleanup removes expired state tokens from the database
func (s *DBStateStore) Cleanup(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM auth.oauth_states
		WHERE expires_at < NOW()
	`)
	return err
}
