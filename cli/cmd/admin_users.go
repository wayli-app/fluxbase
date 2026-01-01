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

var adminUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage admin users",
	Long:  `List, view, invite, and delete admin users.`,
}

var (
	adminUserRole  string
	adminUserEmail string
	adminUserForce bool
)

var adminUsersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all admin users",
	Long: `List all admin/dashboard users.

Examples:
  fluxbase admin users list
  fluxbase admin users list -o json`,
	PreRunE: requireAuth,
	RunE:    runAdminUsersList,
}

var adminUsersGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get admin user details",
	Long: `Get details of a specific admin user.

Examples:
  fluxbase admin users get 550e8400-e29b-41d4-a716-446655440000`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAdminUsersGet,
}

var adminUsersInviteCmd = &cobra.Command{
	Use:   "invite",
	Short: "Invite a new admin user",
	Long: `Invite a new admin user via email.

The invited user will receive an email with a link to set up their account.

Examples:
  fluxbase admin users invite --email admin@example.com
  fluxbase admin users invite --email admin@example.com --role dashboard_admin`,
	PreRunE: requireAuth,
	RunE:    runAdminUsersInvite,
}

var adminUsersDeleteCmd = &cobra.Command{
	Use:     "delete [id]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete an admin user",
	Long: `Delete an admin user from the system.

This action is irreversible.

Examples:
  fluxbase admin users delete 550e8400-e29b-41d4-a716-446655440000
  fluxbase admin users delete 550e8400-e29b-41d4-a716-446655440000 --force`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAdminUsersDelete,
}

func init() {
	// Invite flags
	adminUsersInviteCmd.Flags().StringVar(&adminUserEmail, "email", "", "Email address to invite")
	adminUsersInviteCmd.Flags().StringVar(&adminUserRole, "role", "dashboard_user", "Role for the new user (dashboard_user or dashboard_admin)")
	_ = adminUsersInviteCmd.MarkFlagRequired("email")

	// Delete flags
	adminUsersDeleteCmd.Flags().BoolVarP(&adminUserForce, "force", "f", false, "Skip confirmation prompt")

	// Add subcommands
	adminUsersCmd.AddCommand(adminUsersListCmd)
	adminUsersCmd.AddCommand(adminUsersGetCmd)
	adminUsersCmd.AddCommand(adminUsersInviteCmd)
	adminUsersCmd.AddCommand(adminUsersDeleteCmd)
}

// AdminUser represents an admin/dashboard user
type AdminUser struct {
	ID             string     `json:"id"`
	Email          string     `json:"email"`
	FullName       string     `json:"full_name"`
	Role           string     `json:"role"`
	EmailVerified  bool       `json:"email_verified"`
	IsActive       bool       `json:"is_active"`
	IsLocked       bool       `json:"is_locked"`
	LastSignInAt   *time.Time `json:"last_sign_in_at"`
	ActiveSessions int        `json:"active_sessions"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func runAdminUsersList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	query.Set("type", "dashboard")
	query.Set("limit", "100")

	var result struct {
		Users []*AdminUser `json:"users"`
		Total int          `json:"total"`
	}

	if err := apiClient.DoGet(ctx, "/api/v1/admin/users", query, &result); err != nil {
		return err
	}

	if len(result.Users) == 0 {
		fmt.Println("No admin users found")
		return nil
	}

	if formatter.Format != output.FormatTable {
		return formatter.Print(result.Users)
	}

	data := output.TableData{
		Headers: []string{"ID", "EMAIL", "NAME", "ROLE", "ACTIVE", "LAST LOGIN"},
		Rows:    make([][]string, 0, len(result.Users)),
	}

	for _, user := range result.Users {
		lastLogin := "-"
		if user.LastSignInAt != nil {
			lastLogin = formatTime(*user.LastSignInAt)
		}

		active := "yes"
		if !user.IsActive {
			active = "no"
		}
		if user.IsLocked {
			active = "locked"
		}

		data.Rows = append(data.Rows, []string{
			user.ID,
			user.Email,
			user.FullName,
			user.Role,
			active,
			lastLogin,
		})
	}

	formatter.PrintTable(data)
	fmt.Printf("\nTotal: %d admin users\n", result.Total)

	return nil
}

func runAdminUsersGet(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID := args[0]
	query := url.Values{}
	query.Set("type", "dashboard")

	var user AdminUser
	if err := apiClient.DoGet(ctx, "/api/v1/admin/users/"+url.PathEscape(userID), query, &user); err != nil {
		return err
	}

	if formatter.Format != output.FormatTable {
		return formatter.Print(user)
	}

	fmt.Printf("ID:              %s\n", user.ID)
	fmt.Printf("Email:           %s\n", user.Email)
	fmt.Printf("Name:            %s\n", user.FullName)
	fmt.Printf("Role:            %s\n", user.Role)
	fmt.Printf("Email Verified:  %v\n", user.EmailVerified)
	fmt.Printf("Active:          %v\n", user.IsActive)
	fmt.Printf("Locked:          %v\n", user.IsLocked)
	fmt.Printf("Active Sessions: %d\n", user.ActiveSessions)
	if user.LastSignInAt != nil {
		fmt.Printf("Last Sign In:    %s\n", formatTime(*user.LastSignInAt))
	}
	fmt.Printf("Created:         %s\n", formatTime(user.CreatedAt))
	fmt.Printf("Updated:         %s\n", formatTime(user.UpdatedAt))

	return nil
}

func runAdminUsersInvite(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]any{
		"email": adminUserEmail,
		"role":  adminUserRole,
	}

	var result struct {
		Token     string    `json:"token"`
		Email     string    `json:"email"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	if err := apiClient.DoPost(ctx, "/api/v1/admin/invitations", body, &result); err != nil {
		return err
	}

	if formatter.Format != output.FormatTable {
		return formatter.Print(result)
	}

	fmt.Printf("Invitation sent to %s\n", result.Email)
	fmt.Printf("Expires: %s\n", formatTime(result.ExpiresAt))

	return nil
}

func runAdminUsersDelete(cmd *cobra.Command, args []string) error {
	userID := args[0]

	if !adminUserForce {
		fmt.Printf("Are you sure you want to delete admin user '%s'?\n", userID)
		fmt.Printf("This action is irreversible.\n")
		fmt.Printf("Type 'yes' to confirm: ")

		var confirm string
		_, _ = fmt.Scanln(&confirm)

		if strings.ToLower(confirm) != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Append query param to path since DoDelete doesn't support query params
	path := "/api/v1/admin/users/" + url.PathEscape(userID) + "?type=dashboard"

	if err := apiClient.DoDelete(ctx, path); err != nil {
		return err
	}

	fmt.Printf("Admin user '%s' deleted successfully\n", userID)
	return nil
}
