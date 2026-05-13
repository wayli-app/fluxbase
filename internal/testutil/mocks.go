// Package testutil provides shared test utilities and mocks for unit testing.
package testutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/nimbleflux/fluxbase/internal/storage"
)

// ErrMockObjectNotFound is returned when an object is not found in mock storage
var ErrMockObjectNotFound = errors.New("object not found")

// MockStorageProvider implements storage.Provider for testing
type MockStorageProvider struct {
	mu      sync.RWMutex
	objects map[string]map[string][]byte // bucket -> key -> data
	buckets map[string]bool

	// Callbacks for custom behavior
	OnUpload   func(ctx context.Context, bucket, key string, data io.Reader, size int64) error
	OnDownload func(ctx context.Context, bucket, key string) (io.ReadCloser, *storage.Object, error)
	OnDelete   func(ctx context.Context, bucket, key string) error
}

// NewMockStorageProvider creates a new mock storage provider
func NewMockStorageProvider() *MockStorageProvider {
	return &MockStorageProvider{
		objects: make(map[string]map[string][]byte),
		buckets: make(map[string]bool),
	}
}

func (m *MockStorageProvider) Name() string {
	return "mock"
}

func (m *MockStorageProvider) Health(ctx context.Context) error {
	return nil
}

func (m *MockStorageProvider) Upload(ctx context.Context, bucket, key string, data io.Reader, size int64, opts *storage.UploadOptions) (*storage.Object, error) {
	if m.OnUpload != nil {
		if err := m.OnUpload(ctx, bucket, key, data, size); err != nil {
			return nil, err
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.objects[bucket]; !exists {
		m.objects[bucket] = make(map[string][]byte)
	}

	content, _ := io.ReadAll(data)
	m.objects[bucket][key] = content

	return &storage.Object{
		Key:          key,
		Size:         int64(len(content)),
		LastModified: time.Now(),
	}, nil
}

func (m *MockStorageProvider) Download(ctx context.Context, bucket, key string, opts *storage.DownloadOptions) (io.ReadCloser, *storage.Object, error) {
	if m.OnDownload != nil {
		return m.OnDownload(ctx, bucket, key)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if bucketData, exists := m.objects[bucket]; exists {
		if data, exists := bucketData[key]; exists {
			return io.NopCloser(newBytesReader(data)), &storage.Object{Key: key, Size: int64(len(data))}, nil
		}
	}
	return nil, nil, ErrMockObjectNotFound
}

func (m *MockStorageProvider) Delete(ctx context.Context, bucket, key string) error {
	if m.OnDelete != nil {
		return m.OnDelete(ctx, bucket, key)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if bucketData, exists := m.objects[bucket]; exists {
		delete(bucketData, key)
	}
	return nil
}

func (m *MockStorageProvider) Exists(ctx context.Context, bucket, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if bucketData, exists := m.objects[bucket]; exists {
		_, exists := bucketData[key]
		return exists, nil
	}
	return false, nil
}

func (m *MockStorageProvider) GetObject(ctx context.Context, bucket, key string) (*storage.Object, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if bucketData, exists := m.objects[bucket]; exists {
		if data, exists := bucketData[key]; exists {
			return &storage.Object{Key: key, Size: int64(len(data))}, nil
		}
	}
	return nil, ErrMockObjectNotFound
}

func (m *MockStorageProvider) List(ctx context.Context, bucket string, opts *storage.ListOptions) (*storage.ListResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var objects []storage.Object
	if bucketData, exists := m.objects[bucket]; exists {
		for key, data := range bucketData {
			objects = append(objects, storage.Object{Key: key, Size: int64(len(data))})
		}
	}
	return &storage.ListResult{Objects: objects}, nil
}

func (m *MockStorageProvider) CreateBucket(ctx context.Context, bucket string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.buckets[bucket] = true
	m.objects[bucket] = make(map[string][]byte)
	return nil
}

func (m *MockStorageProvider) DeleteBucket(ctx context.Context, bucket string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.buckets, bucket)
	delete(m.objects, bucket)
	return nil
}

func (m *MockStorageProvider) BucketExists(ctx context.Context, bucket string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.buckets[bucket], nil
}

func (m *MockStorageProvider) ListBuckets(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var buckets []string
	for bucket := range m.buckets {
		buckets = append(buckets, bucket)
	}
	return buckets, nil
}

func (m *MockStorageProvider) GenerateSignedURL(ctx context.Context, bucket, key string, opts *storage.SignedURLOptions) (string, error) {
	return "https://mock-storage.example.com/" + bucket + "/" + key + "?signed=true", nil
}

func (m *MockStorageProvider) CopyObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if srcData, exists := m.objects[srcBucket]; exists {
		if data, exists := srcData[srcKey]; exists {
			if _, exists := m.objects[destBucket]; !exists {
				m.objects[destBucket] = make(map[string][]byte)
			}
			m.objects[destBucket][destKey] = data
			return nil
		}
	}
	return ErrMockObjectNotFound
}

func (m *MockStorageProvider) MoveObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	if err := m.CopyObject(ctx, srcBucket, srcKey, destBucket, destKey); err != nil {
		return err
	}
	return m.Delete(ctx, srcBucket, srcKey)
}

// bytesReader wraps []byte to implement io.Reader
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// MockPubSub implements pubsub.PubSub for testing
type MockPubSub struct {
	mu            sync.RWMutex
	subscriptions map[string][]chan []byte
	published     []PublishedMessage
}

// PublishedMessage records a published message for testing
type PublishedMessage struct {
	Channel string
	Payload []byte
}

// NewMockPubSub creates a new mock pubsub
func NewMockPubSub() *MockPubSub {
	return &MockPubSub{
		subscriptions: make(map[string][]chan []byte),
	}
}

func (m *MockPubSub) Name() string {
	return "mock"
}

func (m *MockPubSub) Publish(ctx context.Context, channel string, payload []byte) error {
	m.mu.Lock()
	m.published = append(m.published, PublishedMessage{Channel: channel, Payload: payload})
	subs := m.subscriptions[channel]
	m.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- payload:
		default:
		}
	}
	return nil
}

func (m *MockPubSub) Subscribe(ctx context.Context, channel string) (<-chan []byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan []byte, 100)
	m.subscriptions[channel] = append(m.subscriptions[channel], ch)
	return ch, nil
}

func (m *MockPubSub) Unsubscribe(ctx context.Context, channel string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.subscriptions, channel)
	return nil
}

func (m *MockPubSub) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, subs := range m.subscriptions {
		for _, ch := range subs {
			close(ch)
		}
	}
	m.subscriptions = make(map[string][]chan []byte)
	return nil
}

// GetPublishedMessages returns all published messages for testing
func (m *MockPubSub) GetPublishedMessages() []PublishedMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]PublishedMessage{}, m.published...)
}

// MockSettingsCache provides a mock for settings.SettingsCache
type MockSettingsCache struct {
	mu       sync.RWMutex
	boolVals map[string]bool
	intVals  map[string]int
	strVals  map[string]string
}

// NewMockSettingsCache creates a new mock settings cache
func NewMockSettingsCache() *MockSettingsCache {
	return &MockSettingsCache{
		boolVals: make(map[string]bool),
		intVals:  make(map[string]int),
		strVals:  make(map[string]string),
	}
}

// SetBool sets a boolean value for testing
func (m *MockSettingsCache) SetBool(key string, value bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.boolVals[key] = value
}

// GetBool retrieves a boolean value (mimics SettingsCache.GetBool interface)
func (m *MockSettingsCache) GetBool(ctx context.Context, key string, defaultValue bool) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, exists := m.boolVals[key]; exists {
		return val
	}
	return defaultValue
}

// SetInt sets an integer value for testing
func (m *MockSettingsCache) SetInt(key string, value int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.intVals[key] = value
}

// GetInt retrieves an integer value
func (m *MockSettingsCache) GetInt(ctx context.Context, key string, defaultValue int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, exists := m.intVals[key]; exists {
		return val
	}
	return defaultValue
}

// SetString sets a string value for testing
func (m *MockSettingsCache) SetString(key string, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strVals[key] = value
}

// GetString retrieves a string value
func (m *MockSettingsCache) GetString(ctx context.Context, key string, defaultValue string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, exists := m.strVals[key]; exists {
		return val
	}
	return defaultValue
}

// MockSubscriptionDB implements realtime.SubscriptionDB for testing.
// It allows configuring which tables are enabled for realtime and
// controlling RLS/ownership check results.
type MockSubscriptionDB struct {
	mu sync.RWMutex

	// EnabledTables maps "schema.table" to enabled status
	EnabledTables map[string]bool

	// RLSResults maps "schema.table.recordID" to access result
	RLSResults map[string]bool

	// OwnershipResults maps execution ID to (isOwner, exists)
	OwnershipResults map[uuid.UUID]struct {
		IsOwner bool
		Exists  bool
	}
}

// NewMockSubscriptionDB creates a new mock subscription database
func NewMockSubscriptionDB() *MockSubscriptionDB {
	return &MockSubscriptionDB{
		EnabledTables: make(map[string]bool),
		RLSResults:    make(map[string]bool),
		OwnershipResults: make(map[uuid.UUID]struct {
			IsOwner bool
			Exists  bool
		}),
	}
}

// EnableTable marks a table as enabled for realtime
func (m *MockSubscriptionDB) EnableTable(schema, table string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EnabledTables[schema+"."+table] = true
}

// IsTableRealtimeEnabled implements SubscriptionDB
func (m *MockSubscriptionDB) IsTableRealtimeEnabled(ctx context.Context, schema, table string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.EnabledTables[schema+"."+table], nil
}

// CheckRLSAccess implements SubscriptionDB
func (m *MockSubscriptionDB) CheckRLSAccess(ctx context.Context, schema, table, role string, claims map[string]interface{}, recordID interface{}) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := schema + "." + table + "." + fmt.Sprintf("%v", recordID)
	if result, exists := m.RLSResults[key]; exists {
		return result, nil
	}
	// Default: allow access
	return true, nil
}

// CheckRPCOwnership implements SubscriptionDB
func (m *MockSubscriptionDB) CheckRPCOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if result, exists := m.OwnershipResults[execID]; exists {
		return result.IsOwner, result.Exists, nil
	}
	return false, false, nil
}

// CheckJobOwnership implements SubscriptionDB
func (m *MockSubscriptionDB) CheckJobOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if result, exists := m.OwnershipResults[execID]; exists {
		return result.IsOwner, result.Exists, nil
	}
	return false, false, nil
}

// CheckFunctionOwnership implements SubscriptionDB
func (m *MockSubscriptionDB) CheckFunctionOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if result, exists := m.OwnershipResults[execID]; exists {
		return result.IsOwner, result.Exists, nil
	}
	return false, false, nil
}

// MockSAMLService provides a mock for auth.SAMLService for testing
type MockSAMLService struct {
	mu sync.RWMutex

	// Callbacks for custom behavior
	OnInitiateLogin  func(ctx context.Context, providerName, relayState string) (redirectURL string, err error)
	OnHandleCallback func(ctx context.Context, samlResponse, relayState string, providerName string) (nameID, email string, attributes map[string][]string, err error)
	OnGetProviders   func(ctx context.Context) ([]interface{}, error)
	OnLogout         func(ctx context.Context, sessionID string) error

	// State tracking for assertions
	Providers      map[string]bool   // provider name -> enabled
	Sessions       map[string]string // session ID -> user ID
	UsedAssertions map[string]bool   // assertion ID -> used (for replay detection)
}

// NewMockSAMLService creates a new mock SAML service
func NewMockSAMLService() *MockSAMLService {
	return &MockSAMLService{
		Providers:      make(map[string]bool),
		Sessions:       make(map[string]string),
		UsedAssertions: make(map[string]bool),
	}
}

// InitiateLogin mocks SAML login initiation
func (m *MockSAMLService) InitiateLogin(ctx context.Context, providerName, relayState string) (string, error) {
	if m.OnInitiateLogin != nil {
		return m.OnInitiateLogin(ctx, providerName, relayState)
	}

	m.mu.RLock()
	enabled := m.Providers[providerName]
	m.mu.RUnlock()

	if !enabled {
		return "", errors.New("provider not found or disabled")
	}

	// Return a mock redirect URL
	return "https://idp.example.com/sso?SAMLRequest=mock", nil
}

// HandleCallback mocks SAML callback processing
func (m *MockSAMLService) HandleCallback(ctx context.Context, samlResponse, relayState, providerName string) (nameID, email string, attributes map[string][]string, err error) {
	if m.OnHandleCallback != nil {
		return m.OnHandleCallback(ctx, samlResponse, relayState, providerName)
	}

	// Check for replay attack
	m.mu.Lock()
	if m.UsedAssertions[samlResponse] {
		m.mu.Unlock()
		return "", "", nil, errors.New("assertion already used")
	}
	m.UsedAssertions[samlResponse] = true
	m.mu.Unlock()

	// Return mock user data
	return "test-name-id", "test@example.com", map[string][]string{
		"email": {"test@example.com"},
		"name":  {"Test User"},
	}, nil
}

// AddProvider adds a provider to the mock for testing
func (m *MockSAMLService) AddProvider(name string, enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Providers[name] = enabled
}

// MarkAssertionUsed marks an assertion as used for replay detection testing
func (m *MockSAMLService) MarkAssertionUsed(assertionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UsedAssertions[assertionID] = true
}

// =============================================================================
// Mock OAuth Provider
// =============================================================================

// MockOAuthProvider implements OAuth provider functionality for testing
type MockOAuthProvider struct {
	// Callbacks for custom behavior
	AuthURLFunc  func(state string) string
	ExchangeFunc func(token string) (*OAuthToken, error)
	GetUserFunc  func(token *OAuthToken) (*OAuthUser, error)

	// State tracking
	StateToken   string // Expected state for validation
	UserInfo     *OAuthUser
	TokenInfo    *OAuthToken
	ShouldError  bool
	ErrorMessage string
}

// OAuthToken represents an OAuth token for testing
type OAuthToken struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
	TokenType    string
}

// OAuthUser represents an OAuth user for testing
type OAuthUser struct {
	ID       string
	Email    string
	Name     string
	Picture  string
	Provider string
}

// NewMockOAuthProvider creates a new mock OAuth provider
func NewMockOAuthProvider() *MockOAuthProvider {
	return &MockOAuthProvider{
		UserInfo: &OAuthUser{
			ID:       "test-oauth-id",
			Email:    "oauth@example.com",
			Name:     "OAuth Test User",
			Provider: "test",
		},
		TokenInfo: &OAuthToken{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			Expiry:       time.Now().Add(time.Hour),
			TokenType:    "Bearer",
		},
	}
}

// AuthURL generates an authorization URL for testing
func (m *MockOAuthProvider) AuthURL(state string) string {
	if m.AuthURLFunc != nil {
		return m.AuthURLFunc(state)
	}
	return "https://oauth.example.com/authorize?state=" + state
}

// Exchange exchanges an authorization code for a token
func (m *MockOAuthProvider) Exchange(code string) (*OAuthToken, error) {
	if m.ExchangeFunc != nil {
		return m.ExchangeFunc(code)
	}
	if m.ShouldError {
		return nil, errors.New(m.ErrorMessage)
	}
	return m.TokenInfo, nil
}

// GetUser gets user information from the OAuth provider
func (m *MockOAuthProvider) GetUser(token *OAuthToken) (*OAuthUser, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(token)
	}
	if m.ShouldError {
		return nil, errors.New(m.ErrorMessage)
	}
	return m.UserInfo, nil
}

// =============================================================================
// Mock TOTP Validator
// =============================================================================

// MockTOTPValidator implements TOTP validation for testing
type MockTOTPValidator struct {
	mu sync.RWMutex

	// Callbacks for custom behavior
	ValidateFunc func(secret, code string) bool
	GenerateFunc func() (secret string, qrCode []byte, err error)

	// State tracking
	ValidCodes   map[string][]string // secret -> valid codes
	Secret       string
	ShouldError  bool
	ErrorMessage string
	CodeIsValid  bool // Override for validation result
}

// NewMockTOTPValidator creates a new mock TOTP validator
func NewMockTOTPValidator() *MockTOTPValidator {
	return &MockTOTPValidator{
		ValidCodes:  make(map[string][]string),
		CodeIsValid: true,
	}
}

// Generate generates a new TOTP secret for testing
func (m *MockTOTPValidator) Generate() (string, []byte, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc()
	}
	if m.ShouldError {
		return "", nil, errors.New(m.ErrorMessage)
	}
	//nolint:gosec // Test mock data, not real credentials
	secret := "JBSWY3DPEHPK3PXP"
	qrCode := []byte("mock-qr-code-data")
	return secret, qrCode, nil
}

// Validate validates a TOTP code for testing
func (m *MockTOTPValidator) Validate(secret, code string) bool {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(secret, code)
	}
	if m.ShouldError {
		return false
	}
	return m.CodeIsValid
}

// AddValidCode adds a valid code for a secret (for testing)
func (m *MockTOTPValidator) AddValidCode(secret, code string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ValidCodes[secret] == nil {
		m.ValidCodes[secret] = []string{}
	}
	m.ValidCodes[secret] = append(m.ValidCodes[secret], code)
}

// =============================================================================
// Mock Rate Limiter
// =============================================================================

// MockRateLimiter implements rate limiting for testing
type MockRateLimiter struct {
	mu sync.RWMutex

	// Callbacks for custom behavior
	CheckFunc  func(key string) (bool, time.Duration, error)
	RecordFunc func(key string) error
	ResetFunc  func(key string) error

	// State tracking
	Attempts     map[string]int
	Locked       map[string]bool
	Limit        int
	Window       time.Duration
	ShouldError  bool
	ErrorMessage string
}

// NewMockRateLimiter creates a new mock rate limiter
func NewMockRateLimiter(limit int, window time.Duration) *MockRateLimiter {
	return &MockRateLimiter{
		Attempts: make(map[string]int),
		Locked:   make(map[string]bool),
		Limit:    limit,
		Window:   window,
	}
}

// Check checks if a key is rate limited
func (m *MockRateLimiter) Check(key string) (bool, time.Duration, error) {
	if m.CheckFunc != nil {
		return m.CheckFunc(key)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.ShouldError {
		return false, 0, errors.New(m.ErrorMessage)
	}

	if m.Locked[key] {
		return false, m.Window, nil
	}

	if m.Attempts[key] >= m.Limit {
		m.Locked[key] = true
		return false, m.Window, nil
	}

	return true, 0, nil
}

// Record records an attempt for a key
func (m *MockRateLimiter) Record(key string) error {
	if m.RecordFunc != nil {
		return m.RecordFunc(key)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Attempts[key]++
	return nil
}

// Reset resets rate limiting for a key
func (m *MockRateLimiter) Reset(key string) error {
	if m.ResetFunc != nil {
		return m.ResetFunc(key)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.Attempts, key)
	delete(m.Locked, key)
	return nil
}

// =============================================================================
// Mock Vector Database
// =============================================================================

// MockVectorDatabase implements vector database operations for testing
type MockVectorDatabase struct {
	mu sync.RWMutex

	// Callbacks for custom behavior
	InsertFunc func(vectors []Vector) error
	SearchFunc func(query Vector, limit int, filters map[string]string) ([]VectorSearchResult, error)
	DeleteFunc func(ids []string) error

	// State tracking
	Vectors      map[string]Vector // id -> vector
	ShouldError  bool
	ErrorMessage string
}

// Vector represents a vector with metadata
type Vector struct {
	ID       string
	Values   []float32
	Metadata map[string]interface{}
}

// VectorSearchResult represents a vector search result
type VectorSearchResult struct {
	Vector     Vector
	Similarity float32
}

// NewMockVectorDatabase creates a new mock vector database
func NewMockVectorDatabase() *MockVectorDatabase {
	return &MockVectorDatabase{
		Vectors: make(map[string]Vector),
	}
}

// Insert inserts vectors into the mock database
func (m *MockVectorDatabase) Insert(vectors []Vector) error {
	if m.InsertFunc != nil {
		return m.InsertFunc(vectors)
	}
	if m.ShouldError {
		return errors.New(m.ErrorMessage)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, v := range vectors {
		m.Vectors[v.ID] = v
	}
	return nil
}

// Search searches for similar vectors
func (m *MockVectorDatabase) Search(query Vector, limit int, filters map[string]string) ([]VectorSearchResult, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(query, limit, filters)
	}
	if m.ShouldError {
		return nil, errors.New(m.ErrorMessage)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []VectorSearchResult
	for _, v := range m.Vectors {
		// Simple cosine similarity (mock implementation)
		similarity := float32(0.9)
		results = append(results, VectorSearchResult{
			Vector:     v,
			Similarity: similarity,
		})
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

// Delete deletes vectors by IDs
func (m *MockVectorDatabase) Delete(ids []string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ids)
	}
	if m.ShouldError {
		return errors.New(m.ErrorMessage)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, id := range ids {
		delete(m.Vectors, id)
	}
	return nil
}

// =============================================================================
// Mock AI Providers
// =============================================================================

// MockOpenAIClient implements OpenAI client for testing
type MockOpenAIClient struct {
	// Callbacks for custom behavior
	ChatCompletionFunc func(ctx context.Context, messages []interface{}, opts map[string]interface{}) (string, error)
	EmbeddingFunc      func(ctx context.Context, texts []string) ([][]float32, error)

	// State tracking
	Response     string
	Embeddings   [][]float32
	ShouldError  bool
	ErrorMessage string
}

// NewMockOpenAIClient creates a new mock OpenAI client
func NewMockOpenAIClient() *MockOpenAIClient {
	return &MockOpenAIClient{
		Response: "Mock AI response",
		Embeddings: [][]float32{
			{0.1, 0.2, 0.3, 0.4, 0.5},
		},
	}
}

// ChatCompletion performs a chat completion
func (m *MockOpenAIClient) ChatCompletion(ctx context.Context, messages []interface{}, opts map[string]interface{}) (string, error) {
	if m.ChatCompletionFunc != nil {
		return m.ChatCompletionFunc(ctx, messages, opts)
	}
	if m.ShouldError {
		return "", errors.New(m.ErrorMessage)
	}
	return m.Response, nil
}

// Embedding generates embeddings for texts
func (m *MockOpenAIClient) Embedding(ctx context.Context, texts []string) ([][]float32, error) {
	if m.EmbeddingFunc != nil {
		return m.EmbeddingFunc(ctx, texts)
	}
	if m.ShouldError {
		return nil, errors.New(m.ErrorMessage)
	}
	result := make([][]float32, len(texts))
	for i := range result {
		result[i] = m.Embeddings[0]
	}
	return result, nil
}

// MockAzureClient is similar to MockOpenAIClient for Azure OpenAI
type MockAzureClient struct {
	*MockOpenAIClient
}

// NewMockAzureClient creates a new mock Azure client
func NewMockAzureClient() *MockAzureClient {
	return &MockAzureClient{
		MockOpenAIClient: NewMockOpenAIClient(),
	}
}

// MockOllamaClient is similar to MockOpenAIClient for Ollama
type MockOllamaClient struct {
	*MockOpenAIClient
}

// NewMockOllamaClient creates a new mock Ollama client
func NewMockOllamaClient() *MockOllamaClient {
	return &MockOllamaClient{
		MockOpenAIClient: NewMockOpenAIClient(),
	}
}

// =============================================================================
// Mock Runtime
// =============================================================================

// MockRuntime implements Deno runtime for testing
type MockRuntime struct {
	// Callbacks for custom behavior
	ExecuteFunc func(code string, env map[string]string) (string, error)
	BundleFunc  func(entryPoint string) (string, []byte, error)

	// State tracking
	Output       string
	BundledCode  string
	SourceMap    []byte
	ShouldError  bool
	ErrorMessage string
}

// NewMockRuntime creates a new mock runtime
func NewMockRuntime() *MockRuntime {
	return &MockRuntime{
		Output:      `{"status":"success","data":"mock output"}`,
		BundledCode: "bundled-code",
		SourceMap:   []byte("mock-source-map"),
	}
}

// Execute executes Deno code
func (m *MockRuntime) Execute(code string, env map[string]string) (string, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(code, env)
	}
	if m.ShouldError {
		return "", errors.New(m.ErrorMessage)
	}
	return m.Output, nil
}

// Bundle bundles Deno code
func (m *MockRuntime) Bundle(entryPoint string) (string, []byte, error) {
	if m.BundleFunc != nil {
		return m.BundleFunc(entryPoint)
	}
	if m.ShouldError {
		return "", nil, errors.New(m.ErrorMessage)
	}
	return m.BundledCode, m.SourceMap, nil
}

// =============================================================================
// Mock OCR Provider
// =============================================================================

// MockOCRProvider implements OCR for testing
type MockOCRProvider struct {
	// Callbacks for custom behavior
	ExtractPDFFunc   func(ctx context.Context, data []byte, languages []string) (string, error)
	ExtractImageFunc func(ctx context.Context, data []byte, languages []string) (string, error)
	IsAvailableFunc  func() bool

	// State tracking
	Text         string
	IsAvailable  bool
	ShouldError  bool
	ErrorMessage string
}

// NewMockOCRProvider creates a new mock OCR provider
func NewMockOCRProvider() *MockOCRProvider {
	return &MockOCRProvider{
		Text:        "Extracted text from document",
		IsAvailable: true,
	}
}

// ExtractPDF extracts text from a PDF
func (m *MockOCRProvider) ExtractPDF(ctx context.Context, data []byte, languages []string) (string, error) {
	if m.ExtractPDFFunc != nil {
		return m.ExtractPDFFunc(ctx, data, languages)
	}
	if m.ShouldError {
		return "", errors.New(m.ErrorMessage)
	}
	return m.Text, nil
}

// ExtractImage extracts text from an image
func (m *MockOCRProvider) ExtractImage(ctx context.Context, data []byte, languages []string) (string, error) {
	if m.ExtractImageFunc != nil {
		return m.ExtractImageFunc(ctx, data, languages)
	}
	if m.ShouldError {
		return "", errors.New(m.ErrorMessage)
	}
	return m.Text, nil
}

// Available returns whether the OCR provider is available
func (m *MockOCRProvider) Available() bool {
	if m.IsAvailableFunc != nil {
		return m.IsAvailableFunc()
	}
	return m.IsAvailable
}
