package auth

import (
	"context"
	"testing"
	"time"
)

func TestMockUserRepository_CreateAndGet(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	req := CreateUserRequest{
		Email: "test@example.com",
		Role:  "authenticated",
	}

	user, err := repo.Create(ctx, req, "hashed_password")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if user.ID == "" {
		t.Error("Expected user ID to be set")
	}
	if user.Email != req.Email {
		t.Errorf("Expected email %s, got %s", req.Email, user.Email)
	}

	// Get by ID
	fetched, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, fetched.Email)
	}

	// Get by email
	fetched, err = repo.GetByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}
	if fetched.ID != user.ID {
		t.Errorf("Expected ID %s, got %s", user.ID, fetched.ID)
	}
}

func TestMockUserRepository_DuplicateEmail(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	req := CreateUserRequest{
		Email: "duplicate@example.com",
	}

	_, err := repo.Create(ctx, req, "hash1")
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	_, err = repo.Create(ctx, req, "hash2")
	if err != ErrUserAlreadyExists {
		t.Errorf("Expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestMockUserRepository_Update(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	req := CreateUserRequest{
		Email: "original@example.com",
		Role:  "authenticated",
	}

	user, _ := repo.Create(ctx, req, "hash")

	newEmail := "updated@example.com"
	verified := true
	updated, err := repo.Update(ctx, user.ID, UpdateUserRequest{
		Email:         &newEmail,
		EmailVerified: &verified,
	})

	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Email != newEmail {
		t.Errorf("Expected email %s, got %s", newEmail, updated.Email)
	}
	if !updated.EmailVerified {
		t.Error("Expected EmailVerified to be true")
	}

	// Old email should not find user
	_, err = repo.GetByEmail(ctx, "original@example.com")
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound for old email, got %v", err)
	}
}

func TestMockUserRepository_Delete(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	req := CreateUserRequest{Email: "delete@example.com"}
	user, _ := repo.Create(ctx, req, "hash")

	err := repo.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = repo.GetByID(ctx, user.ID)
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound after delete, got %v", err)
	}
}

func TestMockUserRepository_Count(t *testing.T) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	count, _ := repo.Count(ctx)
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	repo.Create(ctx, CreateUserRequest{Email: "user1@example.com"}, "hash")
	repo.Create(ctx, CreateUserRequest{Email: "user2@example.com"}, "hash")

	count, _ = repo.Count(ctx)
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestMockSessionRepository_CreateAndGet(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	userID := "user-123"
	accessToken := "access-token"
	refreshToken := "refresh-token"
	expiresAt := time.Now().Add(time.Hour)

	session, err := repo.Create(ctx, userID, accessToken, refreshToken, expiresAt)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if session.ID == "" {
		t.Error("Expected session ID to be set")
	}
	if session.UserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, session.UserID)
	}

	// Get by access token
	fetched, err := repo.GetByAccessToken(ctx, accessToken)
	if err != nil {
		t.Fatalf("GetByAccessToken failed: %v", err)
	}
	if fetched.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, fetched.ID)
	}

	// Get by refresh token
	fetched, err = repo.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		t.Fatalf("GetByRefreshToken failed: %v", err)
	}
	if fetched.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, fetched.ID)
	}

	// Get by user ID
	sessions, err := repo.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetByUserID failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}
}

func TestMockSessionRepository_UpdateTokens(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	session, _ := repo.Create(ctx, "user-123", "old-access", "old-refresh", time.Now().Add(time.Hour))

	newAccess := "new-access"
	newRefresh := "new-refresh"
	newExpiry := time.Now().Add(2 * time.Hour)

	err := repo.UpdateTokens(ctx, session.ID, newAccess, newRefresh, newExpiry)
	if err != nil {
		t.Fatalf("UpdateTokens failed: %v", err)
	}

	// Old tokens should not work
	_, err = repo.GetByAccessToken(ctx, "old-access")
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound for old token, got %v", err)
	}

	// New tokens should work
	fetched, err := repo.GetByAccessToken(ctx, newAccess)
	if err != nil {
		t.Fatalf("GetByAccessToken with new token failed: %v", err)
	}
	if fetched.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, fetched.ID)
	}
}

func TestMockSessionRepository_Delete(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	session, _ := repo.Create(ctx, "user-123", "access", "refresh", time.Now().Add(time.Hour))

	err := repo.Delete(ctx, session.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = repo.GetByAccessToken(ctx, "access")
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestMockSessionRepository_DeleteByUserID(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	userID := "user-123"
	repo.Create(ctx, userID, "access1", "refresh1", time.Now().Add(time.Hour))
	repo.Create(ctx, userID, "access2", "refresh2", time.Now().Add(time.Hour))

	sessions, _ := repo.GetByUserID(ctx, userID)
	if len(sessions) != 2 {
		t.Fatalf("Expected 2 sessions, got %d", len(sessions))
	}

	err := repo.DeleteByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("DeleteByUserID failed: %v", err)
	}

	sessions, _ = repo.GetByUserID(ctx, userID)
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions after delete, got %d", len(sessions))
	}
}

func TestMockSessionRepository_DeleteExpired(t *testing.T) {
	repo := NewMockSessionRepository()
	ctx := context.Background()

	// Create one expired and one valid session
	repo.Create(ctx, "user-123", "expired", "refresh1", time.Now().Add(-time.Hour))
	repo.Create(ctx, "user-456", "valid", "refresh2", time.Now().Add(time.Hour))

	deleted, err := repo.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("Expected 1 deleted session, got %d", deleted)
	}

	count, _ := repo.Count(ctx)
	if count != 1 {
		t.Errorf("Expected 1 remaining session, got %d", count)
	}
}

func TestMockTokenBlacklistRepository_AddAndCheck(t *testing.T) {
	repo := NewMockTokenBlacklistRepository()
	ctx := context.Background()

	jti := "token-jti-123"
	userID := "user-123"

	err := repo.Add(ctx, jti, userID, "logout", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	isBlacklisted, err := repo.IsBlacklisted(ctx, jti)
	if err != nil {
		t.Fatalf("IsBlacklisted failed: %v", err)
	}
	if !isBlacklisted {
		t.Error("Expected token to be blacklisted")
	}

	// Non-blacklisted token
	isBlacklisted, err = repo.IsBlacklisted(ctx, "unknown-jti")
	if err != nil {
		t.Fatalf("IsBlacklisted failed: %v", err)
	}
	if isBlacklisted {
		t.Error("Expected unknown token to not be blacklisted")
	}
}

func TestMockTokenBlacklistRepository_ExpiredNotBlacklisted(t *testing.T) {
	repo := NewMockTokenBlacklistRepository()
	ctx := context.Background()

	jti := "expired-jti"
	err := repo.Add(ctx, jti, "user-123", "logout", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	isBlacklisted, err := repo.IsBlacklisted(ctx, jti)
	if err != nil {
		t.Fatalf("IsBlacklisted failed: %v", err)
	}
	if isBlacklisted {
		t.Error("Expected expired token to not be blacklisted")
	}
}

func TestMockTokenBlacklistRepository_DeleteExpired(t *testing.T) {
	repo := NewMockTokenBlacklistRepository()
	ctx := context.Background()

	repo.Add(ctx, "expired", "user-1", "logout", time.Now().Add(-time.Hour))
	repo.Add(ctx, "valid", "user-2", "logout", time.Now().Add(time.Hour))

	deleted, err := repo.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("Expected 1 deleted entry, got %d", deleted)
	}

	// Valid entry should still exist
	entry, err := repo.GetByJTI(ctx, "valid")
	if err != nil {
		t.Fatalf("GetByJTI failed: %v", err)
	}
	if entry == nil {
		t.Error("Expected valid entry to exist")
	}
}
