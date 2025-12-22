package auth

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MockUserRepository is an in-memory implementation of UserRepositoryInterface for testing.
type MockUserRepository struct {
	mu       sync.RWMutex
	users    map[string]*User
	byEmail  map[string]*User
	CreateFn func(ctx context.Context, req CreateUserRequest, passwordHash string) (*User, error) // Optional override
}

// NewMockUserRepository creates a new mock user repository.
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:   make(map[string]*User),
		byEmail: make(map[string]*User),
	}
}

func (m *MockUserRepository) Create(ctx context.Context, req CreateUserRequest, passwordHash string) (*User, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, req, passwordHash)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.byEmail[req.Email]; exists {
		return nil, ErrUserAlreadyExists
	}

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

	m.users[user.ID] = user
	m.byEmail[user.Email] = user
	return user, nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.byEmail[email]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (m *MockUserRepository) List(ctx context.Context, limit, offset int) ([]*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]*User, 0, len(m.users))
	for _, u := range m.users {
		users = append(users, u)
	}

	// Apply offset and limit
	if offset >= len(users) {
		return []*User{}, nil
	}
	users = users[offset:]
	if limit > 0 && limit < len(users) {
		users = users[:limit]
	}
	return users, nil
}

func (m *MockUserRepository) Update(ctx context.Context, id string, req UpdateUserRequest) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}

	if req.Email != nil {
		delete(m.byEmail, user.Email)
		user.Email = *req.Email
		m.byEmail[user.Email] = user
	}
	if req.EmailVerified != nil {
		user.EmailVerified = *req.EmailVerified
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.UserMetadata != nil {
		user.UserMetadata = req.UserMetadata
	}
	if req.AppMetadata != nil {
		user.AppMetadata = req.AppMetadata
	}
	user.UpdatedAt = time.Now()

	return user, nil
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, id string, newPasswordHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return ErrUserNotFound
	}
	user.PasswordHash = newPasswordHash
	user.UpdatedAt = time.Now()
	return nil
}

func (m *MockUserRepository) VerifyEmail(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return ErrUserNotFound
	}
	user.EmailVerified = true
	user.UpdatedAt = time.Now()
	return nil
}

func (m *MockUserRepository) IncrementFailedLoginAttempts(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[userID]
	if !exists {
		return ErrUserNotFound
	}
	user.FailedLoginAttempts++
	user.UpdatedAt = time.Now()
	return nil
}

func (m *MockUserRepository) ResetFailedLoginAttempts(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[userID]
	if !exists {
		return ErrUserNotFound
	}
	user.FailedLoginAttempts = 0
	user.UpdatedAt = time.Now()
	return nil
}

func (m *MockUserRepository) UnlockUser(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[userID]
	if !exists {
		return ErrUserNotFound
	}
	user.IsLocked = false
	user.LockedUntil = nil
	user.FailedLoginAttempts = 0
	user.UpdatedAt = time.Now()
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return ErrUserNotFound
	}
	delete(m.byEmail, user.Email)
	delete(m.users, id)
	return nil
}

func (m *MockUserRepository) Count(ctx context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users), nil
}

// MockSessionRepository is an in-memory implementation of SessionRepositoryInterface for testing.
type MockSessionRepository struct {
	mu             sync.RWMutex
	sessions       map[string]*Session
	byAccessToken  map[string]*Session
	byRefreshToken map[string]*Session
	byUserID       map[string][]*Session
}

// NewMockSessionRepository creates a new mock session repository.
func NewMockSessionRepository() *MockSessionRepository {
	return &MockSessionRepository{
		sessions:       make(map[string]*Session),
		byAccessToken:  make(map[string]*Session),
		byRefreshToken: make(map[string]*Session),
		byUserID:       make(map[string][]*Session),
	}
}

func (m *MockSessionRepository) Create(ctx context.Context, userID, accessToken, refreshToken string, expiresAt time.Time) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := &Session{
		ID:           uuid.New().String(),
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
	}

	m.sessions[session.ID] = session
	m.byAccessToken[accessToken] = session
	if refreshToken != "" {
		m.byRefreshToken[refreshToken] = session
	}
	m.byUserID[userID] = append(m.byUserID[userID], session)

	return session, nil
}

func (m *MockSessionRepository) GetByAccessToken(ctx context.Context, accessToken string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.byAccessToken[accessToken]
	if !exists {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func (m *MockSessionRepository) GetByRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.byRefreshToken[refreshToken]
	if !exists {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func (m *MockSessionRepository) GetByUserID(ctx context.Context, userID string) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := m.byUserID[userID]
	if sessions == nil {
		return []*Session{}, nil
	}
	return sessions, nil
}

func (m *MockSessionRepository) UpdateTokens(ctx context.Context, id, accessToken, refreshToken string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[id]
	if !exists {
		return ErrSessionNotFound
	}

	// Remove old token mappings
	delete(m.byAccessToken, session.AccessToken)
	delete(m.byRefreshToken, session.RefreshToken)

	// Update session
	session.AccessToken = accessToken
	session.RefreshToken = refreshToken
	session.ExpiresAt = expiresAt

	// Add new token mappings
	m.byAccessToken[accessToken] = session
	if refreshToken != "" {
		m.byRefreshToken[refreshToken] = session
	}

	return nil
}

func (m *MockSessionRepository) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[id]
	if !exists {
		return ErrSessionNotFound
	}

	delete(m.byAccessToken, session.AccessToken)
	delete(m.byRefreshToken, session.RefreshToken)
	delete(m.sessions, id)

	// Remove from user's sessions
	userSessions := m.byUserID[session.UserID]
	for i, s := range userSessions {
		if s.ID == id {
			m.byUserID[session.UserID] = append(userSessions[:i], userSessions[i+1:]...)
			break
		}
	}

	return nil
}

func (m *MockSessionRepository) DeleteByAccessToken(ctx context.Context, accessToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.byAccessToken[accessToken]
	if !exists {
		return ErrSessionNotFound
	}

	delete(m.byAccessToken, session.AccessToken)
	delete(m.byRefreshToken, session.RefreshToken)
	delete(m.sessions, session.ID)

	// Remove from user's sessions
	userSessions := m.byUserID[session.UserID]
	for i, s := range userSessions {
		if s.ID == session.ID {
			m.byUserID[session.UserID] = append(userSessions[:i], userSessions[i+1:]...)
			break
		}
	}

	return nil
}

func (m *MockSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessions := m.byUserID[userID]
	for _, session := range sessions {
		delete(m.byAccessToken, session.AccessToken)
		delete(m.byRefreshToken, session.RefreshToken)
		delete(m.sessions, session.ID)
	}
	delete(m.byUserID, userID)

	return nil
}

func (m *MockSessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	now := time.Now()

	for id, session := range m.sessions {
		if session.ExpiresAt.Before(now) {
			delete(m.byAccessToken, session.AccessToken)
			delete(m.byRefreshToken, session.RefreshToken)
			delete(m.sessions, id)

			userSessions := m.byUserID[session.UserID]
			for i, s := range userSessions {
				if s.ID == id {
					m.byUserID[session.UserID] = append(userSessions[:i], userSessions[i+1:]...)
					break
				}
			}
			count++
		}
	}

	return count, nil
}

func (m *MockSessionRepository) Count(ctx context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions), nil
}

// MockTokenBlacklistRepository is an in-memory implementation of TokenBlacklistRepositoryInterface for testing.
type MockTokenBlacklistRepository struct {
	mu      sync.RWMutex
	entries map[string]*TokenBlacklistEntry
}

// NewMockTokenBlacklistRepository creates a new mock token blacklist repository.
func NewMockTokenBlacklistRepository() *MockTokenBlacklistRepository {
	return &MockTokenBlacklistRepository{
		entries: make(map[string]*TokenBlacklistEntry),
	}
}

func (m *MockTokenBlacklistRepository) Add(ctx context.Context, jti, userID, reason string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries[jti] = &TokenBlacklistEntry{
		ID:        uuid.New().String(),
		TokenJTI:  jti,
		UserID:    userID,
		Reason:    reason,
		ExpiresAt: expiresAt,
		RevokedAt: time.Now(),
	}
	return nil
}

func (m *MockTokenBlacklistRepository) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.entries[jti]
	if !exists {
		return false, nil
	}
	// Check if expired
	if entry.ExpiresAt.Before(time.Now()) {
		return false, nil
	}
	return true, nil
}

func (m *MockTokenBlacklistRepository) GetByJTI(ctx context.Context, jti string) (*TokenBlacklistEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.entries[jti]
	if !exists {
		return nil, errors.New("token not found in blacklist")
	}
	return entry, nil
}

func (m *MockTokenBlacklistRepository) RevokeAllUserTokens(ctx context.Context, userID, reason string) error {
	// In a real implementation, this would query active sessions and blacklist their JTIs
	// For mock purposes, we just record that this was called
	return nil
}

func (m *MockTokenBlacklistRepository) DeleteExpired(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	now := time.Now()

	for jti, entry := range m.entries {
		if entry.ExpiresAt.Before(now) {
			delete(m.entries, jti)
			count++
		}
	}

	return count, nil
}

func (m *MockTokenBlacklistRepository) DeleteByUser(ctx context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for jti, entry := range m.entries {
		if entry.UserID == userID {
			delete(m.entries, jti)
		}
	}
	return nil
}

// Ensure mocks implement interfaces.
var (
	_ UserRepositoryInterface           = (*MockUserRepository)(nil)
	_ SessionRepositoryInterface        = (*MockSessionRepository)(nil)
	_ TokenBlacklistRepositoryInterface = (*MockTokenBlacklistRepository)(nil)
)
