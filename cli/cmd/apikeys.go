package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/output"
	"github.com/fluxbase-eu/fluxbase/cli/util"
)

var apikeysCmd = &cobra.Command{
	Use:     "apikeys",
	Aliases: []string{"apikey", "keys"},
	Short:   "Manage API keys",
	Long:    `Create, revoke, and manage API keys.`,
}

var (
	akName      string
	akScopes    string
	akRateLimit int
	akExpires   string
)

var apikeysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all API keys",
	Long: `List all API keys.

Examples:
  fluxbase apikeys list`,
	PreRunE: requireAuth,
	RunE:    runAPIKeysList,
}

var apikeysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API key",
	Long: `Create a new API key.

Examples:
  fluxbase apikeys create --name "Production API" --scopes "read:tables,write:tables"
  fluxbase apikeys create --name "Read Only" --scopes "read:*" --rate-limit 100`,
	PreRunE: requireAuth,
	RunE:    runAPIKeysCreate,
}

var apikeysGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get API key details",
	Long: `Get details of a specific API key.

Examples:
  fluxbase apikeys get abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAPIKeysGet,
}

var apikeysRevokeCmd = &cobra.Command{
	Use:   "revoke [id]",
	Short: "Revoke an API key",
	Long: `Revoke an API key (disables it but keeps the record).

Examples:
  fluxbase apikeys revoke abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAPIKeysRevoke,
}

var apikeysDeleteCmd = &cobra.Command{
	Use:     "delete [id]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete an API key",
	Long: `Delete an API key permanently.

Examples:
  fluxbase apikeys delete abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAPIKeysDelete,
}

func init() {
	// Create flags
	apikeysCreateCmd.Flags().StringVar(&akName, "name", "", "API key name (required)")
	apikeysCreateCmd.Flags().StringVar(&akScopes, "scopes", "", "Comma-separated scopes")
	apikeysCreateCmd.Flags().IntVar(&akRateLimit, "rate-limit", 0, "Requests per minute (0 = no limit)")
	apikeysCreateCmd.Flags().StringVar(&akExpires, "expires", "", "Expiration duration (e.g., 30d, 1y)")
	_ = apikeysCreateCmd.MarkFlagRequired("name")

	apikeysCmd.AddCommand(apikeysListCmd)
	apikeysCmd.AddCommand(apikeysCreateCmd)
	apikeysCmd.AddCommand(apikeysGetCmd)
	apikeysCmd.AddCommand(apikeysRevokeCmd)
	apikeysCmd.AddCommand(apikeysDeleteCmd)
}

func runAPIKeysList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var keys []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/api-keys", nil, &keys); err != nil {
		return err
	}

	if len(keys) == 0 {
		fmt.Println("No API keys found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"ID", "NAME", "SCOPES", "STATUS", "CREATED"},
			Rows:    make([][]string, len(keys)),
		}

		for i, key := range keys {
			id := getStringValue(key, "id")
			name := getStringValue(key, "name")
			scopes := getStringValue(key, "scopes")
			if len(scopes) > 30 {
				scopes = scopes[:30] + "..."
			}
			revoked := key["revoked"] == true
			status := "active"
			if revoked {
				status = "revoked"
			}
			created := getStringValue(key, "created_at")

			data.Rows[i] = []string{id, name, scopes, status, created}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(keys); err != nil {
			return err
		}
	}

	return nil
}

func runAPIKeysCreate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"name": akName,
	}

	if akScopes != "" {
		scopeList := strings.Split(akScopes, ",")
		for i, s := range scopeList {
			scopeList[i] = strings.TrimSpace(s)
		}
		body["scopes"] = scopeList
	}

	if akRateLimit > 0 {
		body["rate_limit"] = akRateLimit
	}

	if akExpires != "" {
		// Parse duration
		body["expires_in"] = akExpires
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/api-keys", body, &result); err != nil {
		return err
	}

	id := getStringValue(result, "id")
	key := getStringValue(result, "key")

	fmt.Printf("API key created:\n")
	fmt.Printf("  ID: %s\n", id)
	fmt.Printf("  Key: %s\n", key)
	fmt.Println()
	fmt.Println("IMPORTANT: Save this key now. You won't be able to see it again!")

	return nil
}

func runAPIKeysGet(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var key map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/api-keys/"+url.PathEscape(id), nil, &key); err != nil {
		return err
	}

	// Mask the key if present
	if k, ok := key["key"].(string); ok && k != "" {
		key["key"] = util.MaskToken(k)
	}

	formatter := GetFormatter()
	return formatter.Print(key)
}

func runAPIKeysRevoke(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoPost(ctx, "/api/v1/api-keys/"+url.PathEscape(id)+"/revoke", nil, nil); err != nil {
		return err
	}

	fmt.Printf("API key '%s' revoked.\n", id)
	return nil
}

func runAPIKeysDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoDelete(ctx, "/api/v1/api-keys/"+url.PathEscape(id)); err != nil {
		return err
	}

	fmt.Printf("API key '%s' deleted.\n", id)
	return nil
}
