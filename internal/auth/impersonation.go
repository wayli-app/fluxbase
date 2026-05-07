package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

const (
	AnonUserID          = "00000000-0000-0000-0000-000000000000"
	ServiceUserID       = "00000000-0000-0000-0000-000000000001"
	TenantServiceUserID = "00000000-0000-0000-0000-000000000002"
)

var (
	ErrNotAdmin              = errors.New("only dashboard admins can impersonate users")
	ErrSelfImpersonation     = errors.New("cannot impersonate yourself")
	ErrNoActiveImpersonation = errors.New("no active impersonation session found")
	ErrTenantRequired        = errors.New("tenant context is required for impersonation")
	ErrTargetUserNotInTenant = errors.New("target user does not belong to the current tenant")
	ErrNotTenantAdmin        = errors.New("user is not an admin for the specified tenant")
)

type ImpersonationType string

const (
	ImpersonationTypeUser    ImpersonationType = "user"
	ImpersonationTypeAnon    ImpersonationType = "anon"
	ImpersonationTypeService ImpersonationType = "service"
)

type ImpersonationSession struct {
	ID                string            `json:"id" db:"id"`
	AdminUserID       string            `json:"admin_user_id" db:"admin_user_id"`
	TargetUserID      *string           `json:"target_user_id,omitempty" db:"target_user_id"`
	ImpersonationType ImpersonationType `json:"impersonation_type" db:"impersonation_type"`
	TargetRole        *string           `json:"target_role,omitempty" db:"target_role"`
	Reason            string            `json:"reason,omitempty" db:"reason"`
	StartedAt         time.Time         `json:"started_at" db:"started_at"`
	EndedAt           *time.Time        `json:"ended_at,omitempty" db:"ended_at"`
	IPAddress         string            `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent         string            `json:"user_agent,omitempty" db:"user_agent"`
	IsActive          bool              `json:"is_active" db:"is_active"`
	AccessTokenJTI    string            `json:"access_token_jti,omitempty" db:"access_token_jti"`
	RefreshTokenJTI   string            `json:"refresh_token_jti,omitempty" db:"refresh_token_jti"`
	TenantID          *string           `json:"tenant_id,omitempty" db:"tenant_id"`
}

type ImpersonationRepository struct {
	db *database.Connection
}

func NewImpersonationRepository(db *database.Connection) *ImpersonationRepository {
	return &ImpersonationRepository{db: db}
}

var sessionColumns = `id, admin_user_id, target_user_id, impersonation_type, target_role, reason, started_at, ended_at, ip_address, user_agent, is_active, access_token_jti, refresh_token_jti, tenant_id`

func scanSession(
	scanner interface {
		Scan(dest ...interface{}) error
	},
	s *ImpersonationSession,
) error {
	return scanner.Scan(
		&s.ID,
		&s.AdminUserID,
		&s.TargetUserID,
		&s.ImpersonationType,
		&s.TargetRole,
		&s.Reason,
		&s.StartedAt,
		&s.EndedAt,
		&s.IPAddress,
		&s.UserAgent,
		&s.IsActive,
		&s.AccessTokenJTI,
		&s.RefreshTokenJTI,
		&s.TenantID,
	)
}

func (r *ImpersonationRepository) Create(ctx context.Context, session *ImpersonationSession) (*ImpersonationSession, error) {
	query := fmt.Sprintf(`
		INSERT INTO auth.impersonation_sessions
		(id, admin_user_id, target_user_id, impersonation_type, target_role, reason, started_at, ip_address, user_agent, is_active, access_token_jti, refresh_token_jti, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING %s
	`, sessionColumns)

	result := &ImpersonationSession{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			session.ID,
			session.AdminUserID,
			session.TargetUserID,
			session.ImpersonationType,
			session.TargetRole,
			session.Reason,
			session.StartedAt,
			session.IPAddress,
			session.UserAgent,
			session.IsActive,
			session.AccessTokenJTI,
			session.RefreshTokenJTI,
			session.TenantID,
		).Scan(
			&result.ID,
			&result.AdminUserID,
			&result.TargetUserID,
			&result.ImpersonationType,
			&result.TargetRole,
			&result.Reason,
			&result.StartedAt,
			&result.EndedAt,
			&result.IPAddress,
			&result.UserAgent,
			&result.IsActive,
			&result.AccessTokenJTI,
			&result.RefreshTokenJTI,
			&result.TenantID,
		)
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *ImpersonationRepository) EndSession(ctx context.Context, sessionID string) error {
	query := `
		UPDATE auth.impersonation_sessions
		SET ended_at = NOW(), is_active = false
		WHERE id = $1 AND is_active = true
	`

	return database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, sessionID)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return ErrNoActiveImpersonation
		}

		return nil
	})
}

func (r *ImpersonationRepository) GetActiveByAdmin(ctx context.Context, adminUserID string) (*ImpersonationSession, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM auth.impersonation_sessions
		WHERE admin_user_id = $1 AND is_active = true
		ORDER BY started_at DESC
		LIMIT 1
	`, sessionColumns)

	session := &ImpersonationSession{}
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		return scanSession(tx.QueryRow(ctx, query, adminUserID), session)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoActiveImpersonation
		}
		return nil, err
	}

	return session, nil
}

func (r *ImpersonationRepository) ListByAdmin(ctx context.Context, adminUserID string, limit, offset int) ([]*ImpersonationSession, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM auth.impersonation_sessions
		WHERE admin_user_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3
	`, sessionColumns)

	var sessions []*ImpersonationSession
	err := database.WrapWithServiceRole(ctx, r.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, adminUserID, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			session := &ImpersonationSession{}
			if err := scanSession(rows, session); err != nil {
				return err
			}
			sessions = append(sessions, session)
		}

		return rows.Err()
	})

	return sessions, err
}

type ImpersonationService struct {
	repo                  *ImpersonationRepository
	userRepo              *UserRepository
	jwtManager            *JWTManager
	db                    *database.Connection
	tokenBlacklistService *TokenBlacklistService
}

func NewImpersonationService(
	repo *ImpersonationRepository,
	userRepo *UserRepository,
	jwtManager *JWTManager,
	db *database.Connection,
) *ImpersonationService {
	return &ImpersonationService{
		repo:       repo,
		userRepo:   userRepo,
		jwtManager: jwtManager,
		db:         db,
	}
}

func (s *ImpersonationService) SetTokenBlacklistService(service *TokenBlacklistService) {
	s.tokenBlacklistService = service
}

// verifyAdminOrTenantAdmin checks if the user is authorized to impersonate.
// If tenantID is provided, it also accepts tenant_admin assignments for that tenant.
// If tenantID is empty, only instance_admin users are accepted.
func (s *ImpersonationService) verifyAdminOrTenantAdmin(ctx context.Context, adminUserID, tenantID string) error {
	var count int
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM platform.users
			WHERE id = $1 AND deleted_at IS NULL AND is_active = true
		`, adminUserID).Scan(&count)
	})

	if err != nil {
		log.Debug().Err(err).Str("admin_user_id", adminUserID).Msg("Failed to check platform.users")
	} else if count > 0 {
		if tenantID == "" {
			return nil
		}

		var role string
		err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, `
				SELECT role FROM platform.users WHERE id = $1
			`, adminUserID).Scan(&role)
		})
		if err == nil && role == "instance_admin" {
			return nil
		}

		return s.verifyTenantAssignment(ctx, adminUserID, tenantID)
	}

	adminUser, err := s.userRepo.GetByID(ctx, adminUserID)
	if err != nil {
		return fmt.Errorf("admin user not found: %w", err)
	}

	if adminUser.Role == "instance_admin" {
		return nil
	}

	if tenantID != "" {
		return s.verifyTenantAssignment(ctx, adminUserID, tenantID)
	}

	return ErrNotAdmin
}

func (s *ImpersonationService) verifyTenantAssignment(ctx context.Context, adminUserID, tenantID string) error {
	var count int
	err := database.WrapWithServiceRoleAndTenant(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM platform.tenant_admin_assignments
			WHERE user_id = $1 AND tenant_id = $2
		`, adminUserID, tenantID).Scan(&count)
	})
	if err != nil {
		return fmt.Errorf("failed to verify tenant assignment: %w", err)
	}
	if count == 0 {
		return ErrNotTenantAdmin
	}
	return nil
}

func (s *ImpersonationService) verifyTargetUserInTenant(ctx context.Context, targetUserID, tenantID string) error {
	var count int
	err := database.WrapWithServiceRoleAndTenant(ctx, s.db, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM auth.users
			WHERE id = $1 AND tenant_id = $2
		`, targetUserID, tenantID).Scan(&count)
	})
	if err != nil {
		return fmt.Errorf("failed to verify target user tenant: %w", err)
	}
	if count == 0 {
		return ErrTargetUserNotInTenant
	}
	return nil
}

type StartImpersonationRequest struct {
	TargetUserID string `json:"target_user_id"`
	Reason       string `json:"reason"`
	IPAddress    string `json:"-"`
	UserAgent    string `json:"-"`
}

type StartImpersonationResponse struct {
	Session      *ImpersonationSession `json:"session"`
	TargetUser   *User                 `json:"target_user"`
	AccessToken  string                `json:"access_token"`
	RefreshToken string                `json:"refresh_token"`
	ExpiresIn    int64                 `json:"expires_in"`
}

func tenantIDPtr(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}

func (s *ImpersonationService) endExistingSession(ctx context.Context, adminUserID string) error {
	existingSession, err := s.repo.GetActiveByAdmin(ctx, adminUserID)
	if err == nil && existingSession != nil {
		if err := s.repo.EndSession(ctx, existingSession.ID); err != nil {
			return fmt.Errorf("failed to end existing session: %w", err)
		}
	}
	return nil
}

func (s *ImpersonationService) StartImpersonation(
	ctx context.Context,
	adminUserID string,
	tenantID string,
	req StartImpersonationRequest,
) (*StartImpersonationResponse, error) {
	if err := s.verifyAdminOrTenantAdmin(ctx, adminUserID, tenantID); err != nil {
		return nil, err
	}

	targetUser, err := s.userRepo.GetByID(ctx, req.TargetUserID)
	if err != nil {
		return nil, fmt.Errorf("target user not found: %w", err)
	}

	if adminUserID == req.TargetUserID {
		return nil, ErrSelfImpersonation
	}

	if tenantID != "" {
		if err := s.verifyTargetUserInTenant(ctx, req.TargetUserID, tenantID); err != nil {
			return nil, err
		}
	}

	if err := s.endExistingSession(ctx, adminUserID); err != nil {
		return nil, err
	}

	targetUserID := targetUser.ID
	session := &ImpersonationSession{
		ID:                uuid.New().String(),
		AdminUserID:       adminUserID,
		TargetUserID:      &targetUserID,
		ImpersonationType: ImpersonationTypeUser,
		TargetRole:        &targetUser.Role,
		Reason:            req.Reason,
		StartedAt:         time.Now(),
		IPAddress:         req.IPAddress,
		UserAgent:         req.UserAgent,
		IsActive:          true,
		TenantID:          tenantIDPtr(tenantID),
	}

	createdSession, err := s.repo.Create(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create impersonation session: %w", err)
	}

	var opts []TokenOption
	opts = append(opts, WithImpersonatedBy(adminUserID))
	if tenantID != "" {
		opts = append(opts, WithTenantContext(tenantID, "", false))
	}

	accessToken, accessClaims, err := s.jwtManager.GenerateAccessToken(
		targetUser.ID, targetUser.Email, targetUser.Role,
		targetUser.UserMetadata, targetUser.AppMetadata, opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshClaims, err := s.jwtManager.GenerateRefreshToken(
		targetUser.ID, targetUser.Email, targetUser.Role, "",
		targetUser.UserMetadata, targetUser.AppMetadata, opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	createdSession.AccessTokenJTI = accessClaims.ID
	createdSession.RefreshTokenJTI = refreshClaims.ID

	return &StartImpersonationResponse{
		Session:      createdSession,
		TargetUser:   targetUser,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtManager.accessTokenTTL.Seconds()),
	}, nil
}

func (s *ImpersonationService) StartAnonImpersonation(
	ctx context.Context,
	adminUserID string,
	tenantID string,
	reason string,
	ipAddress string,
	userAgent string,
) (*StartImpersonationResponse, error) {
	if err := s.verifyAdminOrTenantAdmin(ctx, adminUserID, tenantID); err != nil {
		return nil, err
	}

	if err := s.endExistingSession(ctx, adminUserID); err != nil {
		return nil, err
	}

	anonRole := "anon"
	session := &ImpersonationSession{
		ID:                uuid.New().String(),
		AdminUserID:       adminUserID,
		TargetUserID:      nil,
		ImpersonationType: ImpersonationTypeAnon,
		TargetRole:        &anonRole,
		Reason:            reason,
		StartedAt:         time.Now(),
		IPAddress:         ipAddress,
		UserAgent:         userAgent,
		IsActive:          true,
		TenantID:          tenantIDPtr(tenantID),
	}

	createdSession, err := s.repo.Create(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create impersonation session: %w", err)
	}

	var opts []TokenOption
	opts = append(opts, WithImpersonatedBy(adminUserID))
	if tenantID != "" {
		opts = append(opts, WithTenantContext(tenantID, "", false))
	}

	accessToken, accessClaims, err := s.jwtManager.GenerateAccessToken(
		AnonUserID, "anonymous@fluxbase.local", "anon", nil, nil, opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshClaims, err := s.jwtManager.GenerateRefreshToken(
		AnonUserID, "anonymous@fluxbase.local", "anon", "", nil, nil, opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	createdSession.AccessTokenJTI = accessClaims.ID
	createdSession.RefreshTokenJTI = refreshClaims.ID

	targetUser := &User{
		ID:    AnonUserID,
		Email: "anonymous@fluxbase.local",
		Role:  "anon",
	}

	return &StartImpersonationResponse{
		Session:      createdSession,
		TargetUser:   targetUser,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtManager.accessTokenTTL.Seconds()),
	}, nil
}

func (s *ImpersonationService) StartServiceImpersonation(
	ctx context.Context,
	adminUserID string,
	tenantID string,
	reason string,
	ipAddress string,
	userAgent string,
) (*StartImpersonationResponse, error) {
	if err := s.verifyAdminOrTenantAdmin(ctx, adminUserID, tenantID); err != nil {
		return nil, err
	}

	if err := s.endExistingSession(ctx, adminUserID); err != nil {
		return nil, err
	}

	var jwtRole, targetRole, email, userID string
	var tenantOpts []TokenOption

	if tenantID != "" {
		userID = TenantServiceUserID
		email = "tenant-service@fluxbase.local"
		jwtRole = "tenant_service"
		targetRole = "tenant_service"
		tenantOpts = append(tenantOpts, WithTenantContext(tenantID, "tenant_service", false))
	} else {
		userID = ServiceUserID
		email = "service@fluxbase.local"
		jwtRole = "service_role"
		targetRole = "service"
	}

	session := &ImpersonationSession{
		ID:                uuid.New().String(),
		AdminUserID:       adminUserID,
		TargetUserID:      nil,
		ImpersonationType: ImpersonationTypeService,
		TargetRole:        &targetRole,
		Reason:            reason,
		StartedAt:         time.Now(),
		IPAddress:         ipAddress,
		UserAgent:         userAgent,
		IsActive:          true,
		TenantID:          tenantIDPtr(tenantID),
	}

	createdSession, err := s.repo.Create(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create impersonation session: %w", err)
	}

	var opts []TokenOption
	opts = append(opts, WithImpersonatedBy(adminUserID))
	opts = append(opts, tenantOpts...)

	accessToken, accessClaims, err := s.jwtManager.GenerateAccessToken(
		userID, email, jwtRole, nil, nil, opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshClaims, err := s.jwtManager.GenerateRefreshToken(
		userID, email, jwtRole, "", nil, nil, opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	createdSession.AccessTokenJTI = accessClaims.ID
	createdSession.RefreshTokenJTI = refreshClaims.ID

	targetUser := &User{
		ID:    userID,
		Email: email,
		Role:  jwtRole,
	}

	return &StartImpersonationResponse{
		Session:      createdSession,
		TargetUser:   targetUser,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtManager.accessTokenTTL.Seconds()),
	}, nil
}

func (s *ImpersonationService) StopImpersonation(ctx context.Context, adminUserID string) error {
	session, err := s.repo.GetActiveByAdmin(ctx, adminUserID)
	if err != nil {
		return err
	}

	if session.AccessTokenJTI != "" && s.tokenBlacklistService != nil {
		expiresAt := time.Now().Add(1 * time.Hour)

		if err := s.tokenBlacklistService.repo.Add(ctx, session.AccessTokenJTI, &adminUserID, "impersonation_stopped", expiresAt); err != nil {
			log.Error().Err(err).Str("jti", session.AccessTokenJTI).Msg("Failed to blacklist access token during impersonation stop")
		} else {
			log.Info().Str("jti", session.AccessTokenJTI).Str("admin_user_id", adminUserID).Msg("Blacklisted impersonation access token")
		}
	}

	if session.RefreshTokenJTI != "" && s.tokenBlacklistService != nil {
		expiresAt := time.Now().Add(24 * time.Hour)

		if err := s.tokenBlacklistService.repo.Add(ctx, session.RefreshTokenJTI, &adminUserID, "impersonation_stopped", expiresAt); err != nil {
			log.Error().Err(err).Str("jti", session.RefreshTokenJTI).Msg("Failed to blacklist refresh token during impersonation stop")
		} else {
			log.Info().Str("jti", session.RefreshTokenJTI).Str("admin_user_id", adminUserID).Msg("Blacklisted impersonation refresh token")
		}
	}

	return s.repo.EndSession(ctx, session.ID)
}

func (s *ImpersonationService) GetActiveSession(ctx context.Context, adminUserID string) (*ImpersonationSession, error) {
	return s.repo.GetActiveByAdmin(ctx, adminUserID)
}

func (s *ImpersonationService) ListSessions(ctx context.Context, adminUserID string, limit, offset int) ([]*ImpersonationSession, error) {
	return s.repo.ListByAdmin(ctx, adminUserID, limit, offset)
}
