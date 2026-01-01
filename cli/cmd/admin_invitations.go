package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var adminInvitationsCmd = &cobra.Command{
	Use:     "invitations",
	Aliases: []string{"invitation", "inv"},
	Short:   "Manage admin invitations",
	Long:    `List and revoke pending admin user invitations.`,
}

var (
	invIncludeAccepted bool
	invIncludeExpired  bool
	invForce           bool
)

var adminInvitationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List admin invitations",
	Long: `List pending and accepted admin invitations.

Examples:
  fluxbase admin invitations list
  fluxbase admin invitations list --include-accepted
  fluxbase admin invitations list --include-expired`,
	PreRunE: requireAuth,
	RunE:    runAdminInvitationsList,
}

var adminInvitationsRevokeCmd = &cobra.Command{
	Use:     "revoke [token]",
	Aliases: []string{"rm", "delete", "remove"},
	Short:   "Revoke an invitation",
	Long: `Revoke a pending admin invitation.

The invitation token can be found using the list command.

Examples:
  fluxbase admin invitations revoke abc123def456
  fluxbase admin invitations revoke abc123def456 --force`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAdminInvitationsRevoke,
}

func init() {
	// List flags
	adminInvitationsListCmd.Flags().BoolVar(&invIncludeAccepted, "include-accepted", false, "Include accepted invitations")
	adminInvitationsListCmd.Flags().BoolVar(&invIncludeExpired, "include-expired", false, "Include expired invitations")

	// Revoke flags
	adminInvitationsRevokeCmd.Flags().BoolVarP(&invForce, "force", "f", false, "Skip confirmation prompt")

	// Add subcommands
	adminInvitationsCmd.AddCommand(adminInvitationsListCmd)
	adminInvitationsCmd.AddCommand(adminInvitationsRevokeCmd)
}

// AdminInvitation represents an admin invitation
type AdminInvitation struct {
	Token      string     `json:"token"`
	Email      string     `json:"email"`
	Role       string     `json:"role"`
	InvitedBy  *string    `json:"invited_by"`
	AcceptedAt *time.Time `json:"accepted_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

func runAdminInvitationsList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	if invIncludeAccepted {
		query.Set("include_accepted", "true")
	}
	if invIncludeExpired {
		query.Set("include_expired", "true")
	}

	var result struct {
		Invitations []*AdminInvitation `json:"invitations"`
	}

	if err := apiClient.DoGet(ctx, "/api/v1/admin/invitations", query, &result); err != nil {
		return err
	}

	if len(result.Invitations) == 0 {
		fmt.Println("No invitations found")
		return nil
	}

	if formatter.Format != output.FormatTable {
		return formatter.Print(result.Invitations)
	}

	data := output.TableData{
		Headers: []string{"TOKEN", "EMAIL", "ROLE", "STATUS", "EXPIRES"},
		Rows:    make([][]string, 0, len(result.Invitations)),
	}

	for _, inv := range result.Invitations {
		status := "pending"
		if inv.AcceptedAt != nil {
			status = "accepted"
		} else if time.Now().After(inv.ExpiresAt) {
			status = "expired"
		}

		// Show only first 12 chars of token for display
		token := inv.Token
		if len(token) > 12 {
			token = token[:12] + "..."
		}

		data.Rows = append(data.Rows, []string{
			token,
			inv.Email,
			inv.Role,
			status,
			formatTime(inv.ExpiresAt),
		})
	}

	formatter.PrintTable(data)

	return nil
}

func runAdminInvitationsRevoke(cmd *cobra.Command, args []string) error {
	token := args[0]

	if !invForce {
		fmt.Printf("Are you sure you want to revoke invitation '%s'?\n", token)
		fmt.Printf("Type 'yes' to confirm: ")

		var confirm string
		_, _ = fmt.Scanln(&confirm)

		if strings.ToLower(confirm) != "yes" {
			fmt.Println("Revocation cancelled")
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoDelete(ctx, "/api/v1/admin/invitations/"+url.PathEscape(token)); err != nil {
		return err
	}

	fmt.Printf("Invitation '%s' revoked successfully\n", token)
	return nil
}
