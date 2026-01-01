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

var usersCmd = &cobra.Command{
	Use:     "users",
	Aliases: []string{"user"},
	Short:   "Manage application users",
	Long: `Manage application users (end users of your application).

Use subcommands to list, view, invite, and delete app users.
For admin/dashboard users, use the 'fluxbase admin users' command instead.`,
}

var (
	appUserEmail string
	appUserForce bool
)

var usersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all app users",
	Long: `List all application users.

Examples:
  fluxbase users list
  fluxbase users list -o json
  fluxbase users list --search john`,
	PreRunE: requireAuth,
	RunE:    runUsersList,
}

var usersGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get app user details",
	Long: `Get details of a specific application user.

Examples:
  fluxbase users get 550e8400-e29b-41d4-a716-446655440000`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runUsersGet,
}

var usersInviteCmd = &cobra.Command{
	Use:   "invite",
	Short: "Invite a new app user",
	Long: `Invite a new application user via email.

The invited user will receive an email with a link to set up their account.

Examples:
  fluxbase users invite --email user@example.com`,
	PreRunE: requireAuth,
	RunE:    runUsersInvite,
}

var usersDeleteCmd = &cobra.Command{
	Use:     "delete [id]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete an app user",
	Long: `Delete an application user from the system.

This action is irreversible.

Examples:
  fluxbase users delete 550e8400-e29b-41d4-a716-446655440000
  fluxbase users delete 550e8400-e29b-41d4-a716-446655440000 --force`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runUsersDelete,
}

var (
	usersSearchQuery string
)

func init() {
	// List flags
	usersListCmd.Flags().StringVar(&usersSearchQuery, "search", "", "Search users by email")

	// Invite flags
	usersInviteCmd.Flags().StringVar(&appUserEmail, "email", "", "Email address to invite")
	_ = usersInviteCmd.MarkFlagRequired("email")

	// Delete flags
	usersDeleteCmd.Flags().BoolVarP(&appUserForce, "force", "f", false, "Skip confirmation prompt")

	// Add subcommands
	usersCmd.AddCommand(usersListCmd)
	usersCmd.AddCommand(usersGetCmd)
	usersCmd.AddCommand(usersInviteCmd)
	usersCmd.AddCommand(usersDeleteCmd)
}

// AppUser represents an application user
type AppUser struct {
	ID             string                 `json:"id"`
	Email          string                 `json:"email"`
	Phone          *string                `json:"phone"`
	Role           string                 `json:"role"`
	EmailVerified  bool                   `json:"email_verified"`
	PhoneVerified  bool                   `json:"phone_verified"`
	IsActive       bool                   `json:"is_active"`
	IsBanned       bool                   `json:"is_banned"`
	LastSignInAt   *time.Time             `json:"last_sign_in_at"`
	ActiveSessions int                    `json:"active_sessions"`
	UserMetadata   map[string]interface{} `json:"user_metadata"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

func runUsersList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	query.Set("type", "app")
	query.Set("limit", "100")
	if usersSearchQuery != "" {
		query.Set("search", usersSearchQuery)
	}

	var result struct {
		Users []*AppUser `json:"users"`
		Total int        `json:"total"`
	}

	if err := apiClient.DoGet(ctx, "/api/v1/admin/users", query, &result); err != nil {
		return err
	}

	if len(result.Users) == 0 {
		fmt.Println("No users found")
		return nil
	}

	if formatter.Format != output.FormatTable {
		return formatter.Print(result.Users)
	}

	data := output.TableData{
		Headers: []string{"ID", "EMAIL", "ROLE", "VERIFIED", "ACTIVE", "LAST LOGIN"},
		Rows:    make([][]string, 0, len(result.Users)),
	}

	for _, user := range result.Users {
		lastLogin := "-"
		if user.LastSignInAt != nil {
			lastLogin = formatTime(*user.LastSignInAt)
		}

		verified := "no"
		if user.EmailVerified {
			verified = "yes"
		}

		active := "yes"
		if !user.IsActive {
			active = "no"
		}
		if user.IsBanned {
			active = "banned"
		}

		data.Rows = append(data.Rows, []string{
			user.ID,
			user.Email,
			user.Role,
			verified,
			active,
			lastLogin,
		})
	}

	formatter.PrintTable(data)
	fmt.Printf("\nTotal: %d users\n", result.Total)

	return nil
}

func runUsersGet(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userID := args[0]
	query := url.Values{}
	query.Set("type", "app")

	var user AppUser
	if err := apiClient.DoGet(ctx, "/api/v1/admin/users/"+url.PathEscape(userID), query, &user); err != nil {
		return err
	}

	if formatter.Format != output.FormatTable {
		return formatter.Print(user)
	}

	fmt.Printf("ID:              %s\n", user.ID)
	fmt.Printf("Email:           %s\n", user.Email)
	if user.Phone != nil {
		fmt.Printf("Phone:           %s\n", *user.Phone)
	}
	fmt.Printf("Role:            %s\n", user.Role)
	fmt.Printf("Email Verified:  %v\n", user.EmailVerified)
	fmt.Printf("Phone Verified:  %v\n", user.PhoneVerified)
	fmt.Printf("Active:          %v\n", user.IsActive)
	fmt.Printf("Banned:          %v\n", user.IsBanned)
	fmt.Printf("Active Sessions: %d\n", user.ActiveSessions)
	if user.LastSignInAt != nil {
		fmt.Printf("Last Sign In:    %s\n", formatTime(*user.LastSignInAt))
	}
	fmt.Printf("Created:         %s\n", formatTime(user.CreatedAt))
	fmt.Printf("Updated:         %s\n", formatTime(user.UpdatedAt))

	return nil
}

func runUsersInvite(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]any{
		"email": appUserEmail,
	}

	query := url.Values{}
	query.Set("type", "app")

	var result struct {
		Message string `json:"message"`
	}

	// Use POST with query params
	if err := apiClient.DoPost(ctx, "/api/v1/admin/users/invite?"+query.Encode(), body, &result); err != nil {
		return err
	}

	fmt.Printf("Invitation sent to %s\n", appUserEmail)

	return nil
}

func runUsersDelete(cmd *cobra.Command, args []string) error {
	userID := args[0]

	if !appUserForce {
		fmt.Printf("Are you sure you want to delete user '%s'?\n", userID)
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
	path := "/api/v1/admin/users/" + url.PathEscape(userID) + "?type=app"

	if err := apiClient.DoDelete(ctx, path); err != nil {
		return err
	}

	fmt.Printf("User '%s' deleted successfully\n", userID)
	return nil
}
