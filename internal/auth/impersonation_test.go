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

// =============================================================================
// Additional Error Tests
// =============================================================================

func TestImpersonationErrors_ExactMessages(t *testing.T) {
	assert.Equal(t, "only dashboard admins can impersonate users", ErrNotAdmin.Error())
	assert.Equal(t, "cannot impersonate yourself", ErrSelfImpersonation.Error())
	assert.Equal(t, "no active impersonation session found", ErrNoActiveImpersonation.Error())
}

func TestImpersonationErrors_Distinct(t *testing.T) {
	errors := []error{
		ErrNotAdmin,
		ErrSelfImpersonation,
		ErrNoActiveImpersonation,
	}

	for i, err1 := range errors {
		for j, err2 := range errors {
			if i != j {
				assert.NotEqual(t, err1, err2)
			}
		}
	}
}

// =============================================================================
// Well-Known User ID Tests
// =============================================================================

func TestWellKnownUserIDs(t *testing.T) {
	t.Run("anon user ID is nil UUID", func(t *testing.T) {
		assert.Equal(t, "00000000-0000-0000-0000-000000000000", AnonUserID)
		assert.Len(t, AnonUserID, 36) // UUID format
	})

	t.Run("service user ID is valid UUID", func(t *testing.T) {
		assert.Equal(t, "00000000-0000-0000-0000-000000000001", ServiceUserID)
		assert.Len(t, ServiceUserID, 36) // UUID format
	})

	t.Run("well-known IDs are distinct", func(t *testing.T) {
		assert.NotEqual(t, AnonUserID, ServiceUserID)
	})
}

// =============================================================================
// ImpersonationType Tests
// =============================================================================

func TestImpersonationType_Values(t *testing.T) {
	types := []struct {
		impType  ImpersonationType
		expected string
	}{
		{ImpersonationTypeUser, "user"},
		{ImpersonationTypeAnon, "anon"},
		{ImpersonationTypeService, "service"},
	}

	for _, tt := range types {
		t.Run(string(tt.impType), func(t *testing.T) {
			assert.Equal(t, ImpersonationType(tt.expected), tt.impType)
		})
	}
}

func TestImpersonationType_StringConversion(t *testing.T) {
	assert.Equal(t, "user", string(ImpersonationTypeUser))
	assert.Equal(t, "anon", string(ImpersonationTypeAnon))
	assert.Equal(t, "service", string(ImpersonationTypeService))
}

// =============================================================================
// Session Duration Tests
// =============================================================================

func TestImpersonationSession_Duration(t *testing.T) {
	t.Run("active session has no end time", func(t *testing.T) {
		session := ImpersonationSession{
			StartedAt: time.Now(),
			EndedAt:   nil,
			IsActive:  true,
		}

		assert.Nil(t, session.EndedAt)
		assert.True(t, session.IsActive)
	})

	t.Run("ended session has duration", func(t *testing.T) {
		start := time.Now().Add(-time.Hour)
		end := time.Now()

		session := ImpersonationSession{
			StartedAt: start,
			EndedAt:   &end,
			IsActive:  false,
		}

		duration := session.EndedAt.Sub(session.StartedAt)
		assert.Equal(t, time.Hour, duration.Round(time.Second))
	})
}

// =============================================================================
// Request Validation Tests
// =============================================================================

func TestStartImpersonationRequest_EmptyFields(t *testing.T) {
	req := StartImpersonationRequest{}

	assert.Empty(t, req.TargetUserID)
	assert.Empty(t, req.Reason)
	assert.Empty(t, req.IPAddress)
	assert.Empty(t, req.UserAgent)
}

func TestStartImpersonationRequest_ReasonLength(t *testing.T) {
	shortReason := "Test"
	longReason := "This is a very detailed reason explaining why the admin needs to impersonate this specific user to investigate a reported issue with their account settings and permissions that has been escalated through support channels."

	reqShort := StartImpersonationRequest{
		TargetUserID: "user-123",
		Reason:       shortReason,
	}

	reqLong := StartImpersonationRequest{
		TargetUserID: "user-456",
		Reason:       longReason,
	}

	assert.Len(t, reqShort.Reason, 4)
	assert.Greater(t, len(reqLong.Reason), 100)
}

// =============================================================================
// Service Constructor Tests
// =============================================================================

func TestNewImpersonationService_WithDependencies(t *testing.T) {
	repo := NewImpersonationRepository(nil)
	userRepo := NewUserRepository(nil)

	svc := NewImpersonationService(repo, userRepo, nil, nil)

	assert.NotNil(t, svc)
	assert.Equal(t, repo, svc.repo)
	assert.Equal(t, userRepo, svc.userRepo)
}

func TestNewImpersonationService_AllNil(t *testing.T) {
	svc := NewImpersonationService(nil, nil, nil, nil)

	assert.NotNil(t, svc)
	assert.Nil(t, svc.repo)
	assert.Nil(t, svc.userRepo)
	assert.Nil(t, svc.jwtManager)
	assert.Nil(t, svc.db)
}

// =============================================================================
// Response Structure Tests
// =============================================================================

func TestStartImpersonationResponse_AnonUser(t *testing.T) {
	anonRole := "anon"
	session := &ImpersonationSession{
		ID:                "session-anon-123",
		AdminUserID:       "admin-123",
		ImpersonationType: ImpersonationTypeAnon,
		TargetRole:        &anonRole,
		IsActive:          true,
	}

	targetUser := &User{
		ID:    AnonUserID,
		Email: "anonymous@fluxbase.local",
		Role:  "anon",
	}

	resp := StartImpersonationResponse{
		Session:      session,
		TargetUser:   targetUser,
		AccessToken:  "anon-access-token",
		RefreshToken: "anon-refresh-token",
		ExpiresIn:    3600,
	}

	assert.Equal(t, AnonUserID, resp.TargetUser.ID)
	assert.Equal(t, "anon", resp.TargetUser.Role)
	assert.Equal(t, ImpersonationTypeAnon, resp.Session.ImpersonationType)
}

func TestStartImpersonationResponse_ServiceUser(t *testing.T) {
	serviceRole := "service_role"
	session := &ImpersonationSession{
		ID:                "session-service-123",
		AdminUserID:       "admin-123",
		ImpersonationType: ImpersonationTypeService,
		TargetRole:        &serviceRole,
		IsActive:          true,
	}

	targetUser := &User{
		ID:    ServiceUserID,
		Email: "service@fluxbase.local",
		Role:  "service_role",
	}

	resp := StartImpersonationResponse{
		Session:      session,
		TargetUser:   targetUser,
		AccessToken:  "service-access-token",
		RefreshToken: "service-refresh-token",
		ExpiresIn:    3600,
	}

	assert.Equal(t, ServiceUserID, resp.TargetUser.ID)
	assert.Equal(t, "service_role", resp.TargetUser.Role)
	assert.Equal(t, ImpersonationTypeService, resp.Session.ImpersonationType)
}

// =============================================================================
// Audit Trail Tests
// =============================================================================

func TestImpersonationSession_AuditFields(t *testing.T) {
	now := time.Now()
	targetUserID := "user-123"
	targetRole := "authenticated"

	session := ImpersonationSession{
		ID:                "session-audit-123",
		AdminUserID:       "admin-456",
		TargetUserID:      &targetUserID,
		ImpersonationType: ImpersonationTypeUser,
		TargetRole:        &targetRole,
		Reason:            "Investigating bug report #12345",
		StartedAt:         now,
		IPAddress:         "203.0.113.42",
		UserAgent:         "AdminDashboard/2.1 (Windows NT 10.0)",
		IsActive:          true,
	}

	// Verify all audit fields are populated
	assert.NotEmpty(t, session.ID)
	assert.NotEmpty(t, session.AdminUserID)
	assert.NotNil(t, session.TargetUserID)
	assert.NotEmpty(t, session.Reason)
	assert.NotEmpty(t, session.IPAddress)
	assert.NotEmpty(t, session.UserAgent)
	assert.False(t, session.StartedAt.IsZero())
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkImpersonationSession_Creation(b *testing.B) {
	targetUserID := "user-123"
	targetRole := "authenticated"

	for i := 0; i < b.N; i++ {
		_ = ImpersonationSession{
			ID:                "session-123",
			AdminUserID:       "admin-456",
			TargetUserID:      &targetUserID,
			ImpersonationType: ImpersonationTypeUser,
			TargetRole:        &targetRole,
			Reason:            "Test reason",
			StartedAt:         time.Now(),
			IPAddress:         "192.168.1.1",
			UserAgent:         "Mozilla/5.0",
			IsActive:          true,
		}
	}
}

func BenchmarkStartImpersonationRequest_Creation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = StartImpersonationRequest{
			TargetUserID: "user-to-impersonate",
			Reason:       "Investigating reported bug",
			IPAddress:    "10.0.0.1",
			UserAgent:    "AdminDashboard/1.0",
		}
	}
}

func BenchmarkNewImpersonationService(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewImpersonationService(nil, nil, nil, nil)
	}
}

func BenchmarkNewImpersonationRepository(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewImpersonationRepository(nil)
	}
}
