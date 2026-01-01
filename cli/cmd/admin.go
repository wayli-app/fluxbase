package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var adminCmd = &cobra.Command{
	Use:     "admin",
	Aliases: []string{"adm"},
	Short:   "Manage admin users and dashboard access",
	Long: `Manage admin users, invitations, and sessions for the Fluxbase dashboard.

Admin users have access to the Fluxbase admin dashboard for managing
your database, users, functions, and other platform features.

Use subcommands to manage:
  - users: Admin user CRUD operations
  - invitations: Pending admin invitations
  - sessions: Active admin sessions
  - password-reset: Send password reset emails`,
}

var (
	adminResetEmail string
)

var adminPasswordResetCmd = &cobra.Command{
	Use:   "password-reset",
	Short: "Send a password reset email to an admin user",
	Long: `Send a password reset email to an admin user.

The admin will receive an email with a link to reset their password.

Examples:
  fluxbase admin password-reset --email admin@example.com`,
	PreRunE: requireAuth,
	RunE:    runAdminPasswordReset,
}

func init() {
	// Password reset flags
	adminPasswordResetCmd.Flags().StringVar(&adminResetEmail, "email", "", "Email address of the admin user")
	_ = adminPasswordResetCmd.MarkFlagRequired("email")

	// Add subcommands
	adminCmd.AddCommand(adminPasswordResetCmd)
	adminCmd.AddCommand(adminUsersCmd)
	adminCmd.AddCommand(adminInvitationsCmd)
	adminCmd.AddCommand(adminSessionsCmd)
}

func runAdminPasswordReset(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]any{
		"email": adminResetEmail,
	}

	if err := apiClient.DoPost(ctx, "/api/v1/admin/password-reset/request", body, nil); err != nil {
		return err
	}

	fmt.Printf("Password reset email sent to %s\n", adminResetEmail)
	return nil
}
