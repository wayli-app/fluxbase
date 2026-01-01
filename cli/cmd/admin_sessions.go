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

var adminSessionsCmd = &cobra.Command{
	Use:     "sessions",
	Aliases: []string{"session"},
	Short:   "Manage admin sessions",
	Long:    `List and revoke active admin sessions.`,
}

var (
	sessionForce bool
)

var adminSessionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active admin sessions",
	Long: `List all active admin sessions.

Examples:
  fluxbase admin sessions list
  fluxbase admin sessions list -o json`,
	PreRunE: requireAuth,
	RunE:    runAdminSessionsList,
}

var adminSessionsRevokeCmd = &cobra.Command{
	Use:     "revoke [session-id]",
	Aliases: []string{"rm", "delete"},
	Short:   "Revoke a specific session",
	Long: `Revoke a specific admin session by ID.

Examples:
  fluxbase admin sessions revoke 550e8400-e29b-41d4-a716-446655440000`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAdminSessionsRevoke,
}

var adminSessionsRevokeAllCmd = &cobra.Command{
	Use:   "revoke-all [user-id]",
	Short: "Revoke all sessions for a user",
	Long: `Revoke all active sessions for a specific admin user.

Examples:
  fluxbase admin sessions revoke-all 550e8400-e29b-41d4-a716-446655440000
  fluxbase admin sessions revoke-all 550e8400-e29b-41d4-a716-446655440000 --force`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAdminSessionsRevokeAll,
}

func init() {
	// Revoke flags
	adminSessionsRevokeCmd.Flags().BoolVarP(&sessionForce, "force", "f", false, "Skip confirmation prompt")
	adminSessionsRevokeAllCmd.Flags().BoolVarP(&sessionForce, "force", "f", false, "Skip confirmation prompt")

	// Add subcommands
	adminSessionsCmd.AddCommand(adminSessionsListCmd)
	adminSessionsCmd.AddCommand(adminSessionsRevokeCmd)
	adminSessionsCmd.AddCommand(adminSessionsRevokeAllCmd)
}

// AdminSession represents an admin session
type AdminSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	UserEmail string    `json:"user_email"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func runAdminSessionsList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	query.Set("limit", "100")

	var result struct {
		Sessions []*AdminSession `json:"sessions"`
		Total    int             `json:"total"`
	}

	if err := apiClient.DoGet(ctx, "/api/v1/admin/auth/sessions", query, &result); err != nil {
		return err
	}

	if len(result.Sessions) == 0 {
		fmt.Println("No active sessions found")
		return nil
	}

	if formatter.Format != output.FormatTable {
		return formatter.Print(result.Sessions)
	}

	data := output.TableData{
		Headers: []string{"ID", "USER", "IP ADDRESS", "CREATED", "EXPIRES"},
		Rows:    make([][]string, 0, len(result.Sessions)),
	}

	for _, session := range result.Sessions {
		// Truncate session ID for display
		sessionID := session.ID
		if len(sessionID) > 12 {
			sessionID = sessionID[:12] + "..."
		}

		data.Rows = append(data.Rows, []string{
			sessionID,
			session.UserEmail,
			session.IPAddress,
			formatTime(session.CreatedAt),
			formatTime(session.ExpiresAt),
		})
	}

	formatter.PrintTable(data)
	fmt.Printf("\nTotal: %d active sessions\n", result.Total)

	return nil
}

func runAdminSessionsRevoke(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	if !sessionForce {
		fmt.Printf("Are you sure you want to revoke session '%s'?\n", sessionID)
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

	if err := apiClient.DoDelete(ctx, "/api/v1/admin/auth/sessions/"+url.PathEscape(sessionID)); err != nil {
		return err
	}

	fmt.Printf("Session '%s' revoked successfully\n", sessionID)
	return nil
}

func runAdminSessionsRevokeAll(cmd *cobra.Command, args []string) error {
	userID := args[0]

	if !sessionForce {
		fmt.Printf("Are you sure you want to revoke all sessions for user '%s'?\n", userID)
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

	if err := apiClient.DoDelete(ctx, "/api/v1/admin/auth/sessions/user/"+url.PathEscape(userID)); err != nil {
		return err
	}

	fmt.Printf("All sessions for user '%s' revoked successfully\n", userID)
	return nil
}
