package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/email"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

type InvitationHandler struct {
	invitationService *auth.InvitationService
	dashboardAuth     *auth.DashboardAuthService
	emailService      email.Service
	baseURL           string
}

func NewInvitationHandler(
	invitationService *auth.InvitationService,
	dashboardAuth *auth.DashboardAuthService,
	emailService email.Service,
	baseURL string,
) *InvitationHandler {
	return &InvitationHandler{
		invitationService: invitationService,
		dashboardAuth:     dashboardAuth,
		emailService:      emailService,
		baseURL:           baseURL,
	}
}

func (h *InvitationHandler) requireInvitationService(c fiber.Ctx) error {
	if h.invitationService == nil {
		return SendNotInitialized(c, "Invitation service")
	}
	return nil
}

func (h *InvitationHandler) requireDashboardAuth(c fiber.Ctx) error {
	if h.dashboardAuth == nil {
		return SendNotInitialized(c, "Dashboard auth service")
	}
	return nil
}

func invitationErrorDetails(err error) string {
	switch {
	case errors.Is(err, auth.ErrInvitationExpired):
		return "Invitation has expired"
	case errors.Is(err, auth.ErrInvitationAlreadyAccepted):
		return "Invitation has already been accepted"
	case errors.Is(err, auth.ErrInvitationNotFound):
		return "Invitation not found"
	default:
		return "Invalid token"
	}
}

func invitationErrorStatus(err error) int {
	switch {
	case errors.Is(err, auth.ErrInvitationExpired):
		return 410
	case errors.Is(err, auth.ErrInvitationAlreadyAccepted):
		return 409
	case errors.Is(err, auth.ErrInvitationNotFound):
		return 404
	default:
		return 400
	}
}

type CreateInvitationRequest struct {
	Email          string `json:"email"`
	Role           string `json:"role"`
	ExpiryDuration int64  `json:"expiry_duration,omitempty"`
}

type CreateInvitationResponse struct {
	Invitation  *auth.InvitationToken `json:"invitation"`
	InviteLink  string                `json:"invite_link"`
	EmailSent   bool                  `json:"email_sent"`
	EmailStatus string                `json:"email_status,omitempty"`
}

type ValidateInvitationResponse struct {
	Valid      bool                  `json:"valid"`
	Invitation *auth.InvitationToken `json:"invitation,omitempty"`
	Error      string                `json:"error,omitempty"`
}

type AcceptInvitationRequest struct {
	Password string `json:"password"`
	Name     string `json:"name"`
}

type AcceptInvitationResponse struct {
	User         *auth.DashboardUser `json:"user"`
	AccessToken  string              `json:"access_token"`
	RefreshToken string              `json:"refresh_token"`
	ExpiresIn    int64               `json:"expires_in"`
}

func (h *InvitationHandler) CreateInvitation(c fiber.Ctx) error {
	ctx := c.Context()

	inviterID := middleware.GetUserID(c)
	if inviterID == "" {
		return SendUnauthorized(c, "User not authenticated", ErrCodeAuthRequired)
	}

	inviterUUID, err := uuid.Parse(inviterID)
	if err != nil {
		return SendInternalError(c, "Invalid user ID")
	}

	var req CreateInvitationRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := auth.ValidateDashboardRole(req.Role); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeInvalidRole)
	}

	inviterRole, ok := c.Locals("user_role").(string)
	if !ok {
		return SendUnauthorized(c, "User role not found", ErrCodeAuthRequired)
	}

	if req.Role == "tenant_admin" && inviterRole != "instance_admin" {
		return SendForbidden(c, "Only instance_admin can invite tenant_admin users", ErrCodeInsufficientPermissions)
	}

	if err := h.requireInvitationService(c); err != nil {
		return err
	}

	expiryDuration := 7 * 24 * time.Hour
	if req.ExpiryDuration > 0 {
		expiryDuration = time.Duration(req.ExpiryDuration) * time.Second
	}

	invitation, err := h.invitationService.CreateInvitation(ctx, req.Email, req.Role, &inviterUUID, expiryDuration)
	if err != nil {
		return SendInternalError(c, "Failed to create invitation")
	}

	inviteLink := fmt.Sprintf("%s/invite/%s", h.baseURL, invitation.PlaintextToken)

	emailSent := false
	emailStatus := ""

	if h.emailService != nil {
		inviterName := "An administrator"
		if err := h.emailService.SendInvitationEmail(ctx, req.Email, inviterName, inviteLink); err != nil {
			log.Warn().Err(err).Str("email", req.Email).Msg("Failed to send invitation email")
			emailStatus = fmt.Sprintf("Failed to send email: %v. Share the invite link manually.", err)
		} else {
			emailSent = true
			emailStatus = "Invitation email sent successfully"
		}
	} else {
		emailStatus = "Email service not configured. Share the invite link manually."
	}

	return c.Status(http.StatusCreated).JSON(CreateInvitationResponse{
		Invitation:  invitation.InvitationToken,
		InviteLink:  inviteLink,
		EmailSent:   emailSent,
		EmailStatus: emailStatus,
	})
}

func (h *InvitationHandler) ValidateInvitation(c fiber.Ctx) error {
	ctx := c.Context()

	token := c.Params("token")
	if token == "" {
		return c.Status(http.StatusBadRequest).JSON(ValidateInvitationResponse{
			Valid: false,
			Error: "Token is required",
		})
	}

	if err := h.requireInvitationService(c); err != nil {
		return err
	}

	invitation, err := h.invitationService.ValidateToken(ctx, token)
	if err != nil {
		return c.JSON(ValidateInvitationResponse{
			Valid: false,
			Error: invitationErrorDetails(err),
		})
	}

	return c.JSON(ValidateInvitationResponse{
		Valid:      true,
		Invitation: invitation,
	})
}

func (h *InvitationHandler) AcceptInvitation(c fiber.Ctx) error {
	ctx := c.Context()

	token := c.Params("token")
	if token == "" {
		return SendMissingField(c, "Token")
	}

	var req AcceptInvitationRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if err := auth.ValidateDashboardPassword(req.Password); err != nil {
		return SendBadRequest(c, err.Error(), ErrCodeValidationFailed)
	}

	if err := h.requireInvitationService(c); err != nil {
		return err
	}

	if err := h.requireDashboardAuth(c); err != nil {
		return err
	}

	invitation, err := h.invitationService.ValidateToken(ctx, token)
	if err != nil {
		return SendError(c, invitationErrorStatus(err), invitationErrorDetails(err))
	}

	_, err = h.dashboardAuth.GetDB().Exec(ctx, "SET LOCAL app.invitation_token = $1", token)
	if err != nil {
		return SendInternalError(c, "Failed to set session context")
	}

	user, err := h.dashboardAuth.CreateUser(ctx, invitation.Email, req.Password, req.Name)
	if err != nil {
		return SendInternalError(c, "Failed to create user")
	}

	_, err = h.dashboardAuth.GetDB().Exec(ctx, `
		UPDATE platform.users
		SET role = $1, email_verified = true
		WHERE id = $2
	`, invitation.Role, user.ID)
	if err != nil {
		return SendInternalError(c, "Failed to set user role and verify email")
	}

	if err := h.invitationService.AcceptInvitation(ctx, token); err != nil {
		log.Warn().
			Err(err).
			Str("invitation_id", invitation.ID.String()).
			Str("email", invitation.Email).
			Msg("Failed to mark invitation as accepted")
	}

	loggedInUser, loginResp, err := h.dashboardAuth.Login(ctx, invitation.Email, req.Password, nil, c.Get("User-Agent"))
	if err != nil {
		return SendInternalError(c, "User created but failed to generate access token")
	}

	return c.Status(http.StatusCreated).JSON(AcceptInvitationResponse{
		User:         loggedInUser,
		AccessToken:  loginResp.AccessToken,
		RefreshToken: loginResp.RefreshToken,
		ExpiresIn:    loginResp.ExpiresIn,
	})
}

func (h *InvitationHandler) ListInvitations(c fiber.Ctx) error {
	ctx := c.Context()

	includeAccepted := c.Query("include_accepted", "false") == "true"
	includeExpired := c.Query("include_expired", "false") == "true"

	if err := h.requireInvitationService(c); err != nil {
		return err
	}

	invitations, err := h.invitationService.ListInvitations(ctx, includeAccepted, includeExpired)
	if err != nil {
		return SendInternalError(c, "Failed to list invitations")
	}

	return c.JSON(fiber.Map{
		"invitations": invitations,
	})
}

func (h *InvitationHandler) RevokeInvitation(c fiber.Ctx) error {
	ctx := c.Context()

	token := c.Params("token")
	if token == "" {
		return SendMissingField(c, "Token")
	}

	if err := h.requireInvitationService(c); err != nil {
		return err
	}

	if err := h.invitationService.RevokeInvitation(ctx, token); err != nil {
		if errors.Is(err, auth.ErrInvitationNotFound) {
			return SendNotFound(c, "Invitation not found")
		}
		return SendInternalError(c, "Failed to revoke invitation")
	}

	return apperrors.SendSuccess(c, "Invitation revoked successfully")
}
