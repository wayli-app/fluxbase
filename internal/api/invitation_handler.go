package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/email"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// InvitationHandler handles invitation-related API endpoints
type InvitationHandler struct {
	invitationService *auth.InvitationService
	dashboardAuth     *auth.DashboardAuthService
	emailService      email.Service
	baseURL           string
}

// NewInvitationHandler creates a new invitation handler
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

// CreateInvitationRequest represents a request to create an invitation
type CreateInvitationRequest struct {
	Email          string `json:"email" validate:"required,email"`
	Role           string `json:"role" validate:"required,oneof=dashboard_admin dashboard_user"`
	ExpiryDuration int64  `json:"expiry_duration,omitempty"` // Duration in seconds, default 7 days
}

// CreateInvitationResponse represents the invitation creation response
type CreateInvitationResponse struct {
	Invitation  *auth.InvitationToken `json:"invitation"`
	InviteLink  string                `json:"invite_link"`
	EmailSent   bool                  `json:"email_sent"`
	EmailStatus string                `json:"email_status,omitempty"`
}

// ValidateInvitationResponse represents the token validation response
type ValidateInvitationResponse struct {
	Valid      bool                  `json:"valid"`
	Invitation *auth.InvitationToken `json:"invitation,omitempty"`
	Error      string                `json:"error,omitempty"`
}

// AcceptInvitationRequest represents a request to accept an invitation
type AcceptInvitationRequest struct {
	Password string `json:"password" validate:"required,min=12"`
	Name     string `json:"name" validate:"required,min=2"`
}

// AcceptInvitationResponse represents the invitation acceptance response
type AcceptInvitationResponse struct {
	User         *auth.DashboardUser `json:"user"`
	AccessToken  string              `json:"access_token"`
	RefreshToken string              `json:"refresh_token"`
	ExpiresIn    int64               `json:"expires_in"`
}

// CreateInvitation generates a new invitation token
// POST /api/v1/admin/invitations
func (h *InvitationHandler) CreateInvitation(c *fiber.Ctx) error {
	ctx := context.Background()

	// Get inviter info from context (set by auth middleware)
	inviterID, ok := c.Locals("user_id").(string)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}

	inviterUUID, err := uuid.Parse(inviterID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Parse request
	var req CreateInvitationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate role
	if req.Role != "dashboard_admin" && req.Role != "dashboard_user" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role. Must be 'dashboard_admin' or 'dashboard_user'",
		})
	}

	// Calculate expiry duration (default 7 days)
	expiryDuration := 7 * 24 * time.Hour
	if req.ExpiryDuration > 0 {
		expiryDuration = time.Duration(req.ExpiryDuration) * time.Second
	}

	// Create invitation
	invitation, err := h.invitationService.CreateInvitation(ctx, req.Email, req.Role, &inviterUUID, expiryDuration)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create invitation: %v", err),
		})
	}

	// Generate invite link using base URL from config
	inviteLink := fmt.Sprintf("%s/invite/%s", h.baseURL, invitation.Token)

	// Send email notification
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
		Invitation:  invitation,
		InviteLink:  inviteLink,
		EmailSent:   emailSent,
		EmailStatus: emailStatus,
	})
}

// ValidateInvitation validates an invitation token
// GET /api/v1/invitations/:token/validate
func (h *InvitationHandler) ValidateInvitation(c *fiber.Ctx) error {
	ctx := context.Background()

	token := c.Params("token")
	if token == "" {
		return c.Status(http.StatusBadRequest).JSON(ValidateInvitationResponse{
			Valid: false,
			Error: "Token is required",
		})
	}

	invitation, err := h.invitationService.ValidateToken(ctx, token)
	if err != nil {
		errorMessage := "Invalid token"
		if err == auth.ErrInvitationExpired {
			errorMessage = "Invitation has expired"
		} else if err == auth.ErrInvitationAlreadyAccepted {
			errorMessage = "Invitation has already been accepted"
		} else if err == auth.ErrInvitationNotFound {
			errorMessage = "Invitation not found"
		}

		return c.JSON(ValidateInvitationResponse{
			Valid: false,
			Error: errorMessage,
		})
	}

	// Don't expose the token in the response
	invitation.Token = ""

	return c.JSON(ValidateInvitationResponse{
		Valid:      true,
		Invitation: invitation,
	})
}

// AcceptInvitation accepts an invitation and creates a new user
// POST /api/v1/invitations/:token/accept
func (h *InvitationHandler) AcceptInvitation(c *fiber.Ctx) error {
	ctx := context.Background()

	token := c.Params("token")
	if token == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Token is required",
		})
	}

	// Parse request
	var req AcceptInvitationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate password strength
	if len(req.Password) < 12 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 12 characters long",
		})
	}

	// Validate invitation token
	invitation, err := h.invitationService.ValidateToken(ctx, token)
	if err != nil {
		errorMessage := "Invalid token"
		statusCode := http.StatusBadRequest
		if err == auth.ErrInvitationExpired {
			errorMessage = "Invitation has expired"
			statusCode = http.StatusGone
		} else if err == auth.ErrInvitationAlreadyAccepted {
			errorMessage = "Invitation has already been accepted"
			statusCode = http.StatusConflict
		} else if err == auth.ErrInvitationNotFound {
			errorMessage = "Invitation not found"
			statusCode = http.StatusNotFound
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"error": errorMessage,
		})
	}

	// Set the invitation token in the database session
	// This allows the RLS policy to verify the invitation
	_, err = h.dashboardAuth.GetDB().Exec(ctx, "SET LOCAL app.invitation_token = $1", token)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to set session context",
		})
	}

	// Create the user in dashboard.users
	user, err := h.dashboardAuth.CreateUser(ctx, invitation.Email, req.Password, req.Name)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create user: %v", err),
		})
	}

	// Update user role and verify email
	_, err = h.dashboardAuth.GetDB().Exec(ctx, `
		UPDATE dashboard.users
		SET role = $1, email_verified = true
		WHERE id = $2
	`, invitation.Role, user.ID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to set user role and verify email",
		})
	}

	// Mark invitation as accepted
	if err := h.invitationService.AcceptInvitation(ctx, token); err != nil {
		// Log the error but don't fail the request - user was created successfully
		log.Warn().
			Err(err).
			Str("token", token).
			Str("email", invitation.Email).
			Msg("Failed to mark invitation as accepted")
	}

	// Log in the user to get access token
	loggedInUser, loginResp, err := h.dashboardAuth.Login(ctx, invitation.Email, req.Password, nil, c.Get("User-Agent"))
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "User created but failed to generate access token",
		})
	}

	return c.Status(http.StatusCreated).JSON(AcceptInvitationResponse{
		User:         loggedInUser,
		AccessToken:  loginResp.AccessToken,
		RefreshToken: loginResp.RefreshToken,
		ExpiresIn:    loginResp.ExpiresIn,
	})
}

// ListInvitations retrieves all invitations (admin only)
// GET /api/v1/admin/invitations
func (h *InvitationHandler) ListInvitations(c *fiber.Ctx) error {
	ctx := context.Background()

	// Parse query parameters
	includeAccepted := c.Query("include_accepted", "false") == "true"
	includeExpired := c.Query("include_expired", "false") == "true"

	invitations, err := h.invitationService.ListInvitations(ctx, includeAccepted, includeExpired)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list invitations",
		})
	}

	// Don't expose tokens in the list
	for i := range invitations {
		invitations[i].Token = ""
	}

	return c.JSON(fiber.Map{
		"invitations": invitations,
	})
}

// RevokeInvitation revokes an invitation token (admin only)
// DELETE /api/v1/admin/invitations/:token
func (h *InvitationHandler) RevokeInvitation(c *fiber.Ctx) error {
	ctx := context.Background()

	token := c.Params("token")
	if token == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Token is required",
		})
	}

	if err := h.invitationService.RevokeInvitation(ctx, token); err != nil {
		if err == auth.ErrInvitationNotFound {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "Invitation not found",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to revoke invitation",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Invitation revoked successfully",
	})
}
