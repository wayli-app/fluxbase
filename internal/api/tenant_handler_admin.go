package api

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"
)

type TenantAdminAssignment struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	UserID     string    `json:"user_id"`
	AssignedAt time.Time `json:"assigned_at"`
}

type AssignAdminRequest struct {
	UserID string `json:"user_id"`
}

func (h *TenantHandler) ListAdmins(c fiber.Ctx) error {
	ctx := c.Context()
	tenantID := c.Params("id")
	userID := middleware.GetUserID(c)
	isInstanceAdmin, _ := c.Locals("is_instance_admin").(bool)

	if !isInstanceAdmin {
		hasAccess, err := h.Storage.IsUserAssignedToTenant(ctx, userID, tenantID)
		if err != nil || !hasAccess {
			return SendForbidden(c, "Access denied to this tenant", ErrCodeAccessDenied)
		}
	}

	rows, err := h.DB.Pool().Query(ctx, `
		SELECT ta.id, ta.tenant_id, ta.user_id, ta.assigned_at,
		       du.email, du.role as dashboard_role
		FROM platform.tenant_admin_assignments ta
		INNER JOIN platform.users du ON du.id = ta.user_id
		WHERE ta.tenant_id = $1::uuid
		ORDER BY ta.assigned_at ASC
	`, tenantID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list admins")
		return SendInternalError(c, "Failed to list admins")
	}
	defer rows.Close()

	type AdminWithUser struct {
		TenantAdminAssignment
		Email         string `json:"email"`
		DashboardRole string `json:"dashboard_role"`
	}

	var admins []AdminWithUser
	for rows.Next() {
		var m AdminWithUser
		err := rows.Scan(
			&m.ID, &m.TenantID, &m.UserID, &m.AssignedAt,
			&m.Email, &m.DashboardRole,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan admin")
			continue
		}
		admins = append(admins, m)
	}

	if admins == nil {
		admins = []AdminWithUser{}
	}

	return c.JSON(admins)
}

func (h *TenantHandler) AssignAdmin(c fiber.Ctx) error {
	ctx := c.Context()
	tenantID := c.Params("id")

	var req AssignAdminRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	var userExists bool
	err := h.DB.Pool().QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM platform.users WHERE id = $1::uuid AND deleted_at IS NULL)`,
		req.UserID,
	).Scan(&userExists)
	if err != nil || !userExists {
		return SendBadRequest(c, "User not found", ErrCodeNotFound)
	}

	var assignment TenantAdminAssignment
	err = h.DB.Pool().QueryRow(ctx, `
		INSERT INTO platform.tenant_admin_assignments (tenant_id, user_id)
		VALUES ($1::uuid, $2::uuid)
		ON CONFLICT (tenant_id, user_id) DO NOTHING
		RETURNING id, tenant_id, user_id, assigned_at
	`, tenantID, req.UserID).Scan(
		&assignment.ID, &assignment.TenantID, &assignment.UserID, &assignment.AssignedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err := h.DB.Pool().QueryRow(ctx, `
				SELECT id, tenant_id, user_id, assigned_at
				FROM platform.tenant_admin_assignments
				WHERE tenant_id = $1::uuid AND user_id = $2::uuid
			`, tenantID, req.UserID).Scan(
				&assignment.ID, &assignment.TenantID, &assignment.UserID, &assignment.AssignedAt,
			)
			if err != nil {
				log.Error().Err(err).Msg("Failed to get existing assignment")
				return SendInternalError(c, "Failed to assign admin")
			}
		} else {
			log.Error().Err(err).Msg("Failed to assign admin")
			return SendInternalError(c, "Failed to assign admin")
		}
	}

	log.Info().
		Str("tenant_id", tenantID).
		Str("user_id", req.UserID).
		Msg("Admin assigned to tenant")

	return c.Status(fiber.StatusCreated).JSON(assignment)
}

func (h *TenantHandler) RemoveAdmin(c fiber.Ctx) error {
	ctx := c.Context()
	tenantID := c.Params("id")
	userID := c.Params("user_id")

	result, err := h.DB.Pool().Exec(ctx, `
		DELETE FROM platform.tenant_admin_assignments
		WHERE tenant_id = $1::uuid AND user_id = $2::uuid
	`, tenantID, userID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to remove admin")
		return SendInternalError(c, "Failed to remove admin")
	}

	if result.RowsAffected() == 0 {
		return SendNotFound(c, "Admin assignment not found")
	}

	log.Info().
		Str("tenant_id", tenantID).
		Str("user_id", userID).
		Msg("Admin removed from tenant")

	return c.SendStatus(fiber.StatusNoContent)
}
