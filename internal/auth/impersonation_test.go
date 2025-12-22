package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestImpersonationConstants(t *testing.T) {
	t.Run("user ID constants are defined", func(t *testing.T) {
		assert.Equal(t, "00000000-0000-0000-0000-000000000000", AnonUserID)
		assert.Equal(t, "00000000-0000-0000-0000-000000000001", ServiceUserID)
	})
}

func TestImpersonationErrors(t *testing.T) {
	t.Run("error types are defined", func(t *testing.T) {
		assert.NotNil(t, ErrNotAdmin)
		assert.NotNil(t, ErrSelfImpersonation)
		assert.NotNil(t, ErrNoActiveImpersonation)
	})

	t.Run("error messages are meaningful", func(t *testing.T) {
		assert.Contains(t, ErrNotAdmin.Error(), "admin")
		assert.Contains(t, ErrSelfImpersonation.Error(), "impersonate yourself")
		assert.Contains(t, ErrNoActiveImpersonation.Error(), "no active")
	})
}

func TestImpersonationType_Constants(t *testing.T) {
	assert.Equal(t, ImpersonationType("user"), ImpersonationTypeUser)
	assert.Equal(t, ImpersonationType("anon"), ImpersonationTypeAnon)
	assert.Equal(t, ImpersonationType("service"), ImpersonationTypeService)
}

func TestImpersonationSession_Struct(t *testing.T) {
	t.Run("creates user impersonation session", func(t *testing.T) {
		now := time.Now()
		targetUserID := "target-user-123"
		targetRole := "authenticated"

		session := ImpersonationSession{
			ID:                "session-123",
			AdminUserID:       "admin-456",
			TargetUserID:      &targetUserID,
			ImpersonationType: ImpersonationTypeUser,
			TargetRole:        &targetRole,
			Reason:            "Debugging user issue",
			StartedAt:         now,
			IPAddress:         "192.168.1.1",
			UserAgent:         "Mozilla/5.0",
			IsActive:          true,
		}

		assert.Equal(t, "session-123", session.ID)
		assert.Equal(t, "admin-456", session.AdminUserID)
		assert.Equal(t, "target-user-123", *session.TargetUserID)
		assert.Equal(t, ImpersonationTypeUser, session.ImpersonationType)
		assert.Equal(t, "authenticated", *session.TargetRole)
		assert.Equal(t, "Debugging user issue", session.Reason)
		assert.True(t, session.IsActive)
		assert.Nil(t, session.EndedAt)
	})

	t.Run("creates anon impersonation session", func(t *testing.T) {
		anonRole := "anon"

		session := ImpersonationSession{
			ID:                "session-456",
			AdminUserID:       "admin-789",
			TargetUserID:      nil, // No target user for anon
			ImpersonationType: ImpersonationTypeAnon,
			TargetRole:        &anonRole,
			Reason:            "Testing anonymous access",
			IsActive:          true,
		}

		assert.Nil(t, session.TargetUserID)
		assert.Equal(t, ImpersonationTypeAnon, session.ImpersonationType)
		assert.Equal(t, "anon", *session.TargetRole)
	})

	t.Run("creates service impersonation session", func(t *testing.T) {
		serviceRole := "service_role"

		session := ImpersonationSession{
			ID:                "session-789",
			AdminUserID:       "admin-101",
			TargetUserID:      nil,
			ImpersonationType: ImpersonationTypeService,
			TargetRole:        &serviceRole,
			Reason:            "Testing service role access",
			IsActive:          true,
		}

		assert.Nil(t, session.TargetUserID)
		assert.Equal(t, ImpersonationTypeService, session.ImpersonationType)
		assert.Equal(t, "service_role", *session.TargetRole)
	})

	t.Run("handles ended session", func(t *testing.T) {
		now := time.Now()
		endedAt := now.Add(time.Hour)
		targetUserID := "user-123"

		session := ImpersonationSession{
			ID:                "session-ended",
			AdminUserID:       "admin-123",
			TargetUserID:      &targetUserID,
			ImpersonationType: ImpersonationTypeUser,
			StartedAt:         now,
			EndedAt:           &endedAt,
			IsActive:          false,
		}

		assert.False(t, session.IsActive)
		assert.NotNil(t, session.EndedAt)
	})
}

func TestStartImpersonationRequest_Struct(t *testing.T) {
	req := StartImpersonationRequest{
		TargetUserID: "user-to-impersonate",
		Reason:       "Investigating reported bug",
		IPAddress:    "10.0.0.1",
		UserAgent:    "AdminDashboard/1.0",
	}

	assert.Equal(t, "user-to-impersonate", req.TargetUserID)
	assert.Equal(t, "Investigating reported bug", req.Reason)
	assert.Equal(t, "10.0.0.1", req.IPAddress)
	assert.Equal(t, "AdminDashboard/1.0", req.UserAgent)
}

func TestStartImpersonationResponse_Struct(t *testing.T) {
	session := &ImpersonationSession{
		ID:          "session-123",
		AdminUserID: "admin-123",
		IsActive:    true,
	}
	targetUser := &User{
		ID:    "user-456",
		Email: "user@example.com",
		Role:  "authenticated",
	}

	resp := StartImpersonationResponse{
		Session:      session,
		TargetUser:   targetUser,
		AccessToken:  "access-token-jwt",
		RefreshToken: "refresh-token-jwt",
		ExpiresIn:    3600,
	}

	assert.Equal(t, session, resp.Session)
	assert.Equal(t, targetUser, resp.TargetUser)
	assert.Equal(t, "access-token-jwt", resp.AccessToken)
	assert.Equal(t, "refresh-token-jwt", resp.RefreshToken)
	assert.Equal(t, int64(3600), resp.ExpiresIn)
}

func TestNewImpersonationRepository(t *testing.T) {
	repo := NewImpersonationRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewImpersonationService(t *testing.T) {
	svc := NewImpersonationService(nil, nil, nil, nil)
	assert.NotNil(t, svc)
}
