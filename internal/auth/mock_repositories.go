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

	// Set default role if not provided
	if user.Role == "" {
		user.Role = "authenticated"
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

	// Lock account after 5 failed attempts
	if user.FailedLoginAttempts >= 5 {
		user.IsLocked = true
	}

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
	user.IsLocked = false
	user.LockedUntil = nil
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

// UpdateEmail updates a user's email
func (m *MockUserRepository) UpdateEmail(ctx context.Context, id, newEmail string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[id]
	if !exists {
		return ErrUserNotFound
	}

	// Delete old email mapping
	delete(m.byEmail, user.Email)

	// Update email
	user.Email = newEmail
	user.UpdatedAt = time.Now()

	// Add new email mapping
	m.byEmail[newEmail] = user

	return nil
}

// LockUser locks a user account for a specified duration
func (m *MockUserRepository) LockUser(ctx context.Context, userID string, lockDuration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[userID]
	if !exists {
		return ErrUserNotFound
	}

	user.IsLocked = true
	lockedUntil := time.Now().Add(lockDuration)
	user.LockedUntil = &lockedUntil
	user.UpdatedAt = time.Now()

	return nil
}

// UpdateRole updates a user's role
func (m *MockUserRepository) UpdateRole(ctx context.Context, userID, role string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[userID]
	if !exists {
		return ErrUserNotFound
	}

	user.Role = role
	user.UpdatedAt = time.Now()

	return nil
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
	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
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
	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
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

	// Filter out expired sessions
	now := time.Now()
	activeSessions := make([]*Session, 0)
	for _, s := range sessions {
		if s.ExpiresAt.After(now) {
			activeSessions = append(activeSessions, s)
		}
	}
	return activeSessions, nil
}

func (m *MockSessionRepository) UpdateTokens(ctx context.Context, id, expectedRefreshToken, accessToken, refreshToken string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[id]
	if !exists {
		return ErrSessionNotFound
	}

	// Compare-and-swap: reject a refresh token that was already rotated.
	if session.RefreshToken != expectedRefreshToken {
		return ErrRefreshTokenRotated
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

	// Count only active (non-expired) sessions
	now := time.Now()
	count := 0
	for _, s := range m.sessions {
		if s.ExpiresAt.After(now) {
			count++
		}
	}
	return count, nil
}

// UpdateAccessToken updates only the access token for a session
func (m *MockSessionRepository) UpdateAccessToken(ctx context.Context, id, accessToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[id]
	if !exists {
		return ErrSessionNotFound
	}

	// Delete old access token mapping
	delete(m.byAccessToken, session.AccessToken)

	// Update access token
	session.AccessToken = accessToken

	// Add new access token mapping
	m.byAccessToken[accessToken] = session

	return nil
}

// CountByUserID returns the number of active sessions for a user
func (m *MockSessionRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	count := 0
	for _, s := range m.sessions {
		if s.UserID == userID && s.ExpiresAt.After(now) {
			count++
		}
	}
	return count, nil
}

// ListAll returns all sessions with user info for admin views
func (m *MockSessionRepository) ListAll(ctx context.Context, includeExpired bool) ([]SessionWithUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []SessionWithUser
	now := time.Now()
	for _, s := range m.sessions {
		if includeExpired || s.ExpiresAt.After(now) {
			result = append(result, SessionWithUser{
				ID:        s.ID,
				UserID:    s.UserID,
				ExpiresAt: s.ExpiresAt,
				CreatedAt: s.CreatedAt,
			})
		}
	}
	return result, nil
}

// ListAllPaginated returns paginated sessions with user info
func (m *MockSessionRepository) ListAllPaginated(ctx context.Context, includeExpired bool, limit, offset int) ([]SessionWithUser, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allSessions []SessionWithUser
	now := time.Now()
	for _, s := range m.sessions {
		if includeExpired || s.ExpiresAt.After(now) {
			allSessions = append(allSessions, SessionWithUser{
				ID:        s.ID,
				UserID:    s.UserID,
				ExpiresAt: s.ExpiresAt,
				CreatedAt: s.CreatedAt,
			})
		}
	}

	total := len(allSessions)

	// Apply pagination
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	if start >= end {
		return []SessionWithUser{}, total, nil
	}

	return allSessions[start:end], total, nil
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

func (m *MockTokenBlacklistRepository) Add(ctx context.Context, jti string, revokedBy *string, reason string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	revokedByValue := ""
	if revokedBy != nil {
		revokedByValue = *revokedBy
	}

	m.entries[jti] = &TokenBlacklistEntry{
		ID:        uuid.New().String(),
		TokenJTI:  jti,
		RevokedBy: revokedByValue,
		Reason:    reason,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
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

func (m *MockTokenBlacklistRepository) RevokeAllUserTokens(ctx context.Context, userID, reason string, expiry time.Duration) error {
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
		if entry.RevokedBy == userID {
			delete(m.entries, jti)
		}
	}
	return nil
}

// MockPasswordResetRepository is an in-memory implementation of password reset repository for testing
type MockPasswordResetRepository struct {
	mu               sync.RWMutex
	tokens           map[string]*PasswordResetToken
	byUserID         map[string]*PasswordResetToken
	ValidateFn       func(ctx context.Context, token string) (*PasswordResetToken, error)
	CreateFn         func(ctx context.Context, userID string, expiryDuration time.Duration) (*PasswordResetTokenWithPlaintext, error)
	MarkAsUsedFn     func(ctx context.Context, id string) error
	GetLatestFn      func(ctx context.Context, userID string) (*PasswordResetToken, error)
	DeleteByUserIDFn func(ctx context.Context, userID string) error
}

// NewMockPasswordResetRepository creates a new mock password reset repository
func NewMockPasswordResetRepository() *MockPasswordResetRepository {
	return &MockPasswordResetRepository{
		tokens:   make(map[string]*PasswordResetToken),
		byUserID: make(map[string]*PasswordResetToken),
	}
}

func (m *MockPasswordResetRepository) Create(ctx context.Context, userID string, expiryDuration time.Duration) (*PasswordResetTokenWithPlaintext, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, userID, expiryDuration)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	plaintextToken, err := GeneratePasswordResetToken()
	if err != nil {
		return nil, err
	}

	tokenHash := hashPasswordResetToken(plaintextToken)
	token := &PasswordResetToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(expiryDuration),
		CreatedAt: time.Now(),
	}

	m.tokens[token.ID] = token
	m.byUserID[userID] = token

	return &PasswordResetTokenWithPlaintext{
		PasswordResetToken: *token,
		PlaintextToken:     plaintextToken,
	}, nil
}

func (m *MockPasswordResetRepository) GetByToken(ctx context.Context, token string) (*PasswordResetToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tokenHash := hashPasswordResetToken(token)
	for _, t := range m.tokens {
		if t.TokenHash == tokenHash {
			return t, nil
		}
	}
	return nil, ErrPasswordResetTokenNotFound
}

func (m *MockPasswordResetRepository) GetLatestByUserID(ctx context.Context, userID string) (*PasswordResetToken, error) {
	if m.GetLatestFn != nil {
		return m.GetLatestFn(ctx, userID)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	token, exists := m.byUserID[userID]
	if !exists {
		return nil, ErrPasswordResetTokenNotFound
	}
	return token, nil
}

func (m *MockPasswordResetRepository) MarkAsUsed(ctx context.Context, id string) error {
	if m.MarkAsUsedFn != nil {
		return m.MarkAsUsedFn(ctx, id)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	token, exists := m.tokens[id]
	if !exists {
		return ErrPasswordResetTokenNotFound
	}

	now := time.Now()
	token.UsedAt = &now
	return nil
}

func (m *MockPasswordResetRepository) Validate(ctx context.Context, token string) (*PasswordResetToken, error) {
	if m.ValidateFn != nil {
		return m.ValidateFn(ctx, token)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	tokenHash := hashPasswordResetToken(token)
	for _, t := range m.tokens {
		if t.TokenHash == tokenHash {
			// Check if already used
			if t.UsedAt != nil {
				return nil, ErrPasswordResetTokenUsed
			}
			// Check if expired
			if time.Now().After(t.ExpiresAt) {
				return nil, ErrPasswordResetTokenExpired
			}
			return t, nil
		}
	}
	return nil, ErrPasswordResetTokenNotFound
}

func (m *MockPasswordResetRepository) DeleteByUserID(ctx context.Context, userID string) error {
	if m.DeleteByUserIDFn != nil {
		return m.DeleteByUserIDFn(ctx, userID)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Find and delete token for user
	if token, exists := m.byUserID[userID]; exists {
		delete(m.tokens, token.ID)
		delete(m.byUserID, userID)
	}
	return nil
}

func (m *MockPasswordResetRepository) DeleteExpired(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	now := time.Now()
	for id, token := range m.tokens {
		if token.ExpiresAt.Before(now) {
			delete(m.tokens, id)
			if token.UserID != "" {
				delete(m.byUserID, token.UserID)
			}
			count++
		}
	}
	return count, nil
}

// MockTOTPRateLimiter is a mock TOTP rate limiter for testing
type MockTOTPRateLimiter struct {
	mu              sync.RWMutex
	failedAttempts  map[string]int       // userID -> attempt count
	blockedUsers    map[string]time.Time // userID -> blocked until
	CheckLimitFn    func(ctx context.Context, userID string) error
	RecordAttemptFn func(ctx context.Context, userID string, success bool, ipAddress, userAgent string) error
	maxAttempts     int
	windowDuration  time.Duration
	lockoutDuration time.Duration
}

// NewMockTOTPRateLimiter creates a new mock TOTP rate limiter
func NewMockTOTPRateLimiter() *MockTOTPRateLimiter {
	return &MockTOTPRateLimiter{
		failedAttempts:  make(map[string]int),
		blockedUsers:    make(map[string]time.Time),
		maxAttempts:     5,
		windowDuration:  5 * time.Minute,
		lockoutDuration: 15 * time.Minute,
	}
}

// SetMaxAttempts sets the maximum attempts before rate limiting kicks in
func (m *MockTOTPRateLimiter) SetMaxAttempts(max int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxAttempts = max
}

// Reset clears all rate limiting state for a user
func (m *MockTOTPRateLimiter) Reset(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.failedAttempts, userID)
	delete(m.blockedUsers, userID)
}

// SetBlockedUntil marks a user as blocked until a specific time
func (m *MockTOTPRateLimiter) SetBlockedUntil(userID string, until time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blockedUsers[userID] = until
}

// CheckRateLimit checks if the user has exceeded the TOTP attempt limit
func (m *MockTOTPRateLimiter) CheckRateLimit(ctx context.Context, userID string) error {
	if m.CheckLimitFn != nil {
		return m.CheckLimitFn(ctx, userID)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if user is blocked
	if blockedUntil, blocked := m.blockedUsers[userID]; blocked {
		if time.Now().Before(blockedUntil) {
			return ErrTOTPRateLimitExceeded
		}
		// Block period expired, remove block
		m.mu.RUnlock()
		m.mu.Lock()
		delete(m.blockedUsers, userID)
		m.mu.Unlock()
		m.mu.RLock()
	}

	// Check failed attempt count
	if m.failedAttempts[userID] >= m.maxAttempts {
		return ErrTOTPRateLimitExceeded
	}

	return nil
}

// RecordAttempt records a TOTP verification attempt (success or failure)
func (m *MockTOTPRateLimiter) RecordAttempt(ctx context.Context, userID string, success bool, ipAddress, userAgent string) error {
	if m.RecordAttemptFn != nil {
		return m.RecordAttemptFn(ctx, userID, success, ipAddress, userAgent)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if success {
		// Reset failed attempts on success
		delete(m.failedAttempts, userID)
		delete(m.blockedUsers, userID)
	} else {
		// Increment failed attempts
		m.failedAttempts[userID]++

		// Block user if max attempts reached
		if m.failedAttempts[userID] >= m.maxAttempts {
			m.blockedUsers[userID] = time.Now().Add(m.lockoutDuration)
		}
	}

	return nil
}

// GetFailedAttempts returns the current failed attempt count for a user
func (m *MockTOTPRateLimiter) GetFailedAttempts(userID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.failedAttempts[userID]
}

// MockOAuthManager is a mock OAuth manager for testing
type MockOAuthManager struct {
	mu             sync.RWMutex
	providers      map[string]bool
	AuthURLFn      func(provider, state string) (string, error)
	ExchangeCodeFn func(ctx context.Context, provider, code string) (string, map[string]interface{}, error)
	GetUserInfoFn  func(ctx context.Context, provider string, tokenStr string) (map[string]interface{}, error)
}

// NewMockOAuthManager creates a new mock OAuth manager
func NewMockOAuthManager() *MockOAuthManager {
	return &MockOAuthManager{
		providers: make(map[string]bool),
	}
}

// RegisterProvider registers an OAuth provider
func (m *MockOAuthManager) RegisterProvider(provider string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[provider] = true
}

// GetAuthURL generates an OAuth authorization URL
func (m *MockOAuthManager) GetAuthURL(provider, state string) (string, error) {
	if m.AuthURLFn != nil {
		return m.AuthURLFn(provider, state)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.providers[provider] {
		return "", ErrInvalidProvider
	}

	return "https://" + provider + ".example.com/oauth/authorize?state=" + state, nil
}

// ExchangeCode exchanges an authorization code for tokens
func (m *MockOAuthManager) ExchangeCode(ctx context.Context, provider, code string) (string, map[string]interface{}, error) {
	if m.ExchangeCodeFn != nil {
		return m.ExchangeCodeFn(ctx, provider, code)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.providers[provider] {
		return "", nil, ErrInvalidProvider
	}

	// Return mock token and user info
	token := "mock_access_token_" + provider
	userInfo := map[string]interface{}{
		"id":    "provider_user_123",
		"email": "user@" + provider + ".com",
		"name":  "Test User",
	}

	return token, userInfo, nil
}

// GetUserInfo retrieves user information from the OAuth provider
func (m *MockOAuthManager) GetUserInfo(ctx context.Context, provider string, tokenStr string) (map[string]interface{}, error) {
	if m.GetUserInfoFn != nil {
		return m.GetUserInfoFn(ctx, provider, tokenStr)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.providers[provider] {
		return nil, ErrInvalidProvider
	}

	return map[string]interface{}{
		"id":    "provider_user_123",
		"email": "user@" + provider + ".com",
		"name":  "Test User",
	}, nil
}

// Ensure mocks implement interfaces.
var (
	_ UserRepositoryInterface           = (*MockUserRepository)(nil)
	_ SessionRepositoryInterface        = (*MockSessionRepository)(nil)
	_ TokenBlacklistRepositoryInterface = (*MockTokenBlacklistRepository)(nil)
	_ PasswordResetRepositoryInterface  = (*MockPasswordResetRepository)(nil)
)

var (
	_ UserRepositoryInterface           = (*MockUserRepository)(nil)
	_ SessionRepositoryInterface        = (*MockSessionRepository)(nil)
	_ TokenBlacklistRepositoryInterface = (*MockTokenBlacklistRepository)(nil)
)
