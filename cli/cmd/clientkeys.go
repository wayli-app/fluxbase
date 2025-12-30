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

var clientkeysCmd = &cobra.Command{
	Use:     "clientkeys",
	Aliases: []string{"clientkey", "keys"},
	Short:   "Manage client keys",
	Long:    `Create, revoke, and manage client keys.`,
}

var (
	ckName      string
	ckScopes    string
	ckRateLimit int
	ckExpires   string
)

var clientkeysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all client keys",
	Long: `List all client keys.

Examples:
  fluxbase clientkeys list`,
	PreRunE: requireAuth,
	RunE:    runClientKeysList,
}

var clientkeysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new client key",
	Long: `Create a new client key.

Examples:
  fluxbase clientkeys create --name "Production API" --scopes "read:tables,write:tables"
  fluxbase clientkeys create --name "Read Only" --scopes "read:*" --rate-limit 100`,
	PreRunE: requireAuth,
	RunE:    runClientKeysCreate,
}

var clientkeysGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get client key details",
	Long: `Get details of a specific client key.

Examples:
  fluxbase clientkeys get abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runClientKeysGet,
}

var clientkeysRevokeCmd = &cobra.Command{
	Use:   "revoke [id]",
	Short: "Revoke a client key",
	Long: `Revoke a client key (disables it but keeps the record).

Examples:
  fluxbase clientkeys revoke abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runClientKeysRevoke,
}

var clientkeysDeleteCmd = &cobra.Command{
	Use:     "delete [id]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a client key",
	Long: `Delete a client key permanently.

Examples:
  fluxbase clientkeys delete abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runClientKeysDelete,
}

func init() {
	// Create flags
	clientkeysCreateCmd.Flags().StringVar(&ckName, "name", "", "Client key name (required)")
	clientkeysCreateCmd.Flags().StringVar(&ckScopes, "scopes", "", "Comma-separated scopes")
	clientkeysCreateCmd.Flags().IntVar(&ckRateLimit, "rate-limit", 0, "Requests per minute (0 = no limit)")
	clientkeysCreateCmd.Flags().StringVar(&ckExpires, "expires", "", "Expiration duration (e.g., 30d, 1y)")
	_ = clientkeysCreateCmd.MarkFlagRequired("name")

	clientkeysCmd.AddCommand(clientkeysListCmd)
	clientkeysCmd.AddCommand(clientkeysCreateCmd)
	clientkeysCmd.AddCommand(clientkeysGetCmd)
	clientkeysCmd.AddCommand(clientkeysRevokeCmd)
	clientkeysCmd.AddCommand(clientkeysDeleteCmd)
}

func runClientKeysList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var keys []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/client-keys", nil, &keys); err != nil {
		return err
	}

	if len(keys) == 0 {
		fmt.Println("No client keys found.")
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

func runClientKeysCreate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"name": ckName,
	}

	if ckScopes != "" {
		scopeList := strings.Split(ckScopes, ",")
		for i, s := range scopeList {
			scopeList[i] = strings.TrimSpace(s)
		}
		body["scopes"] = scopeList
	}

	if ckRateLimit > 0 {
		body["rate_limit"] = ckRateLimit
	}

	if ckExpires != "" {
		// Parse duration
		body["expires_in"] = ckExpires
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/client-keys", body, &result); err != nil {
		return err
	}

	id := getStringValue(result, "id")
	key := getStringValue(result, "key")

	fmt.Printf("Client key created:\n")
	fmt.Printf("  ID: %s\n", id)
	fmt.Printf("  Key: %s\n", key)
	fmt.Println()
	fmt.Println("IMPORTANT: Save this key now. You won't be able to see it again!")

	return nil
}

func runClientKeysGet(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var key map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/client-keys/"+url.PathEscape(id), nil, &key); err != nil {
		return err
	}

	// Mask the key if present
	if k, ok := key["key"].(string); ok && k != "" {
		key["key"] = util.MaskToken(k)
	}

	formatter := GetFormatter()
	return formatter.Print(key)
}

func runClientKeysRevoke(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoPost(ctx, "/api/v1/client-keys/"+url.PathEscape(id)+"/revoke", nil, nil); err != nil {
		return err
	}

	fmt.Printf("Client key '%s' revoked.\n", id)
	return nil
}

func runClientKeysDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoDelete(ctx, "/api/v1/client-keys/"+url.PathEscape(id)); err != nil {
		return err
	}

	fmt.Printf("Client key '%s' deleted.\n", id)
	return nil
}
