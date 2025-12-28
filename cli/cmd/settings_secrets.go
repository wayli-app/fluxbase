package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var (
	settingsSecretUser        bool
	settingsSecretDescription string
)

var settingsSecretsCmd = &cobra.Command{
	Use:     "secrets",
	Aliases: []string{"secret"},
	Short:   "Manage encrypted settings secrets",
	Long: `Manage encrypted application secrets stored in settings.

Unlike edge function secrets (fluxbase secrets), these are application-level secrets
stored in the settings system with encryption. They support user-specific secrets
that only the owning user can access.

Use --user flag to work with user-specific secrets (encrypted with user-derived key).
Without --user, operates on system-level secrets (requires admin privileges).`,
}

var settingsSecretsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all settings secrets (metadata only)",
	Long: `List all encrypted settings secrets (values are never shown).

Examples:
  # List system secrets (admin only)
  fluxbase settings secrets list

  # List user-specific secrets
  fluxbase settings secrets list --user`,
	PreRunE: requireAuth,
	RunE:    runSettingsSecretsList,
}

var settingsSecretsSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Create or update a settings secret",
	Long: `Create or update an encrypted settings secret.

The value is encrypted at rest and can only be decrypted server-side.
SDK and API endpoints only return metadata, never the actual value.

Examples:
  # Set a system secret (admin only)
  fluxbase settings secrets set stripe_api_key "sk-live-xxx" --description "Stripe production key"

  # Set a user-specific secret
  fluxbase settings secrets set openai_api_key "sk-xxx" --user --description "My OpenAI key"`,
	Args:    cobra.ExactArgs(2),
	PreRunE: requireAuth,
	RunE:    runSettingsSecretsSet,
}

var settingsSecretsGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get secret metadata (not the value)",
	Long: `Get metadata for a specific settings secret (the value is never returned).

Examples:
  # Get system secret metadata
  fluxbase settings secrets get stripe_api_key

  # Get user-specific secret metadata
  fluxbase settings secrets get openai_api_key --user`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runSettingsSecretsGet,
}

var settingsSecretsDeleteCmd = &cobra.Command{
	Use:     "delete [key]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a settings secret",
	Long: `Delete a settings secret permanently.

Examples:
  # Delete a system secret (admin only)
  fluxbase settings secrets delete stripe_api_key

  # Delete a user-specific secret
  fluxbase settings secrets delete openai_api_key --user`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runSettingsSecretsDelete,
}

func init() {
	// List flags
	settingsSecretsListCmd.Flags().BoolVar(&settingsSecretUser, "user", false, "Operate on user-specific secrets")

	// Set flags
	settingsSecretsSetCmd.Flags().BoolVar(&settingsSecretUser, "user", false, "Create a user-specific secret")
	settingsSecretsSetCmd.Flags().StringVar(&settingsSecretDescription, "description", "", "Description of the secret")

	// Get flags
	settingsSecretsGetCmd.Flags().BoolVar(&settingsSecretUser, "user", false, "Get a user-specific secret")

	// Delete flags
	settingsSecretsDeleteCmd.Flags().BoolVar(&settingsSecretUser, "user", false, "Delete a user-specific secret")

	settingsSecretsCmd.AddCommand(settingsSecretsListCmd)
	settingsSecretsCmd.AddCommand(settingsSecretsSetCmd)
	settingsSecretsCmd.AddCommand(settingsSecretsGetCmd)
	settingsSecretsCmd.AddCommand(settingsSecretsDeleteCmd)

	// Register under settings command
	settingsCmd.AddCommand(settingsSecretsCmd)
}

// SecretMetadata represents the metadata returned for a secret
type SecretMetadata struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Description string    `json:"description,omitempty"`
	UserID      *string   `json:"user_id,omitempty"`
	CreatedBy   *string   `json:"created_by,omitempty"`
	UpdatedBy   *string   `json:"updated_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func runSettingsSecretsList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var endpoint string
	if settingsSecretUser {
		endpoint = "/api/v1/settings/secrets"
	} else {
		endpoint = "/api/v1/admin/settings/custom/secrets"
	}

	var secrets []SecretMetadata
	if err := apiClient.DoGet(ctx, endpoint, nil, &secrets); err != nil {
		return err
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		var rows [][]string
		for _, s := range secrets {
			desc := s.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}
			scope := "system"
			if s.UserID != nil {
				scope = "user"
			}
			rows = append(rows, []string{
				s.Key,
				scope,
				desc,
				s.UpdatedAt.Format("2006-01-02 15:04"),
			})
		}

		data := output.TableData{
			Headers: []string{"KEY", "SCOPE", "DESCRIPTION", "UPDATED"},
			Rows:    rows,
		}

		if len(rows) == 0 {
			if settingsSecretUser {
				fmt.Println("No user secrets found.")
			} else {
				fmt.Println("No system secrets found.")
			}
		} else {
			formatter.PrintTable(data)
		}
	} else {
		if err := formatter.Print(secrets); err != nil {
			return err
		}
	}

	return nil
}

func runSettingsSecretsSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var endpoint string
	if settingsSecretUser {
		endpoint = "/api/v1/settings/secret"
	} else {
		endpoint = "/api/v1/admin/settings/custom/secret"
	}

	body := map[string]interface{}{
		"key":   key,
		"value": value,
	}
	if settingsSecretDescription != "" {
		body["description"] = settingsSecretDescription
	}

	var result SecretMetadata
	if err := apiClient.DoPost(ctx, endpoint, body, &result); err != nil {
		// Try PUT if it already exists (upsert behavior)
		putEndpoint := endpoint + "/" + key
		if err := apiClient.DoPut(ctx, putEndpoint, map[string]interface{}{
			"value":       value,
			"description": settingsSecretDescription,
		}, &result); err != nil {
			return err
		}
	}

	scope := "system"
	if settingsSecretUser {
		scope = "user"
	}
	fmt.Printf("Secret '%s' (%s) created/updated successfully.\n", key, scope)
	return nil
}

func runSettingsSecretsGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var endpoint string
	if settingsSecretUser {
		endpoint = "/api/v1/settings/secret/" + key
	} else {
		endpoint = "/api/v1/admin/settings/custom/secret/" + key
	}

	var secret SecretMetadata
	if err := apiClient.DoGet(ctx, endpoint, nil, &secret); err != nil {
		return err
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		scope := "system"
		if secret.UserID != nil {
			scope = "user"
		}

		fmt.Printf("Key:         %s\n", secret.Key)
		fmt.Printf("Scope:       %s\n", scope)
		if secret.Description != "" {
			fmt.Printf("Description: %s\n", secret.Description)
		}
		fmt.Printf("Created:     %s\n", secret.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated:     %s\n", secret.UpdatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println("\nNote: Secret values are never returned. They can only be accessed server-side.")
	} else {
		if err := formatter.Print(secret); err != nil {
			return err
		}
	}

	return nil
}

func runSettingsSecretsDelete(cmd *cobra.Command, args []string) error {
	key := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var endpoint string
	if settingsSecretUser {
		endpoint = "/api/v1/settings/secret/" + key
	} else {
		endpoint = "/api/v1/admin/settings/custom/secret/" + key
	}

	if err := apiClient.DoDelete(ctx, endpoint); err != nil {
		return err
	}

	scope := "system"
	if settingsSecretUser {
		scope = "user"
	}
	fmt.Printf("Secret '%s' (%s) deleted successfully.\n", key, scope)
	return nil
}
