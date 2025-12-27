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

var secretsCmd = &cobra.Command{
	Use:     "secrets",
	Aliases: []string{"secret"},
	Short:   "Manage secrets for edge functions",
	Long:    `Create, update, delete, and manage secrets that are injected into edge functions at runtime.`,
}

var (
	secretScope       string
	secretNamespace   string
	secretDescription string
	secretExpires     string
)

var secretsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets",
	Long: `List all secrets (values are never shown).

Examples:
  fluxbase secrets list
  fluxbase secrets list --scope global
  fluxbase secrets list --namespace my-namespace`,
	PreRunE: requireAuth,
	RunE:    runSecretsList,
}

var secretsSetCmd = &cobra.Command{
	Use:   "set [name] [value]",
	Short: "Create or update a secret",
	Long: `Create or update a secret.

The value is encrypted at rest and injected as FLUXBASE_SECRET_<NAME> environment variable.

Examples:
  fluxbase secrets set API_KEY "my-secret-key"
  fluxbase secrets set DATABASE_URL "postgres://..." --scope namespace --namespace my-ns
  fluxbase secrets set TEMP_KEY "value" --expires 30d`,
	Args:    cobra.ExactArgs(2),
	PreRunE: requireAuth,
	RunE:    runSecretsSet,
}

var secretsGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get secret metadata (not the value)",
	Long: `Get metadata for a specific secret (the value is never returned).

Examples:
  fluxbase secrets get API_KEY
  fluxbase secrets get DATABASE_URL --namespace my-namespace`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runSecretsGet,
}

var secretsDeleteCmd = &cobra.Command{
	Use:     "delete [name]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a secret",
	Long: `Delete a secret permanently.

Examples:
  fluxbase secrets delete API_KEY
  fluxbase secrets delete DATABASE_URL --namespace my-namespace`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runSecretsDelete,
}

var secretsHistoryCmd = &cobra.Command{
	Use:   "history [name]",
	Short: "Show version history for a secret",
	Long: `Show version history for a secret (values are never shown).

Examples:
  fluxbase secrets history API_KEY
  fluxbase secrets history DATABASE_URL --namespace my-namespace`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runSecretsHistory,
}

var secretsRollbackCmd = &cobra.Command{
	Use:   "rollback [name] [version]",
	Short: "Rollback a secret to a previous version",
	Long: `Rollback a secret to a previous version.

Examples:
  fluxbase secrets rollback API_KEY 2
  fluxbase secrets rollback DATABASE_URL 1 --namespace my-namespace`,
	Args:    cobra.ExactArgs(2),
	PreRunE: requireAuth,
	RunE:    runSecretsRollback,
}

func init() {
	// List flags
	secretsListCmd.Flags().StringVar(&secretScope, "scope", "", "Filter by scope (global, namespace)")
	secretsListCmd.Flags().StringVar(&secretNamespace, "namespace", "", "Filter by namespace")

	// Set flags
	secretsSetCmd.Flags().StringVar(&secretScope, "scope", "global", "Secret scope (global, namespace)")
	secretsSetCmd.Flags().StringVar(&secretNamespace, "namespace", "", "Namespace for namespace-scoped secrets")
	secretsSetCmd.Flags().StringVar(&secretDescription, "description", "", "Description of the secret")
	secretsSetCmd.Flags().StringVar(&secretExpires, "expires", "", "Expiration duration (e.g., 30d, 1y)")

	// Get flags
	secretsGetCmd.Flags().StringVar(&secretNamespace, "namespace", "", "Namespace for namespace-scoped secrets")

	// Delete flags
	secretsDeleteCmd.Flags().StringVar(&secretNamespace, "namespace", "", "Namespace for namespace-scoped secrets")

	// History flags
	secretsHistoryCmd.Flags().StringVar(&secretNamespace, "namespace", "", "Namespace for namespace-scoped secrets")

	// Rollback flags
	secretsRollbackCmd.Flags().StringVar(&secretNamespace, "namespace", "", "Namespace for namespace-scoped secrets")

	secretsCmd.AddCommand(secretsListCmd)
	secretsCmd.AddCommand(secretsSetCmd)
	secretsCmd.AddCommand(secretsGetCmd)
	secretsCmd.AddCommand(secretsDeleteCmd)
	secretsCmd.AddCommand(secretsHistoryCmd)
	secretsCmd.AddCommand(secretsRollbackCmd)
}

func runSecretsList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := url.Values{}
	if secretScope != "" {
		params.Set("scope", secretScope)
	}
	if secretNamespace != "" {
		params.Set("namespace", secretNamespace)
	}

	var secrets []map[string]interface{}
	path := "/api/v1/secrets"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	if err := apiClient.DoGet(ctx, path, nil, &secrets); err != nil {
		return err
	}

	if len(secrets) == 0 {
		fmt.Println("No secrets found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"NAME", "SCOPE", "NAMESPACE", "VERSION", "EXPIRES", "UPDATED"},
			Rows:    make([][]string, len(secrets)),
		}

		for i, secret := range secrets {
			name := getStringValue(secret, "name")
			scope := getStringValue(secret, "scope")
			namespace := getStringValue(secret, "namespace")
			if namespace == "" {
				namespace = "-"
			}
			version := fmt.Sprintf("%v", secret["version"])
			expiresAt := getStringValue(secret, "expires_at")
			if expiresAt == "" {
				expiresAt = "never"
			} else if secret["is_expired"] == true {
				expiresAt = expiresAt + " (expired)"
			}
			updatedAt := getStringValue(secret, "updated_at")

			data.Rows[i] = []string{name, scope, namespace, version, expiresAt, updatedAt}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(secrets); err != nil {
			return err
		}
	}

	return nil
}

func runSecretsSet(cmd *cobra.Command, args []string) error {
	name := args[0]
	value := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First, try to find if the secret already exists
	findParams := url.Values{}
	findParams.Set("name", name)
	if secretNamespace != "" {
		findParams.Set("namespace", secretNamespace)
	} else if secretScope == "global" {
		findParams.Set("scope", "global")
	}

	var existingSecrets []map[string]interface{}
	findPath := "/api/v1/secrets?" + findParams.Encode()
	_ = apiClient.DoGet(ctx, findPath, nil, &existingSecrets)

	// Find exact match
	var existingID string
	for _, s := range existingSecrets {
		sName := getStringValue(s, "name")
		sNamespace := getStringValue(s, "namespace")
		sScope := getStringValue(s, "scope")

		// Match by name and scope/namespace
		if sName == name {
			if secretNamespace != "" && sNamespace == secretNamespace {
				existingID = getStringValue(s, "id")
				break
			} else if secretNamespace == "" && sScope == "global" && sNamespace == "" {
				existingID = getStringValue(s, "id")
				break
			}
		}
	}

	if existingID != "" {
		// Update existing secret
		body := map[string]interface{}{
			"value": value,
		}
		if secretDescription != "" {
			body["description"] = secretDescription
		}

		if err := apiClient.DoPut(ctx, "/api/v1/secrets/"+url.PathEscape(existingID), body, nil); err != nil {
			return err
		}

		fmt.Printf("Secret '%s' updated.\n", name)
	} else {
		// Create new secret
		body := map[string]interface{}{
			"name":  name,
			"value": value,
			"scope": secretScope,
		}

		if secretScope == "namespace" {
			if secretNamespace == "" {
				return fmt.Errorf("--namespace is required when scope is 'namespace'")
			}
			body["namespace"] = secretNamespace
		}

		if secretDescription != "" {
			body["description"] = secretDescription
		}

		if secretExpires != "" {
			// Parse duration and convert to timestamp
			duration, err := parseDuration(secretExpires)
			if err != nil {
				return fmt.Errorf("invalid expiration format: %w", err)
			}
			expiresAt := time.Now().Add(duration)
			body["expires_at"] = expiresAt.Format(time.RFC3339)
		}

		if err := apiClient.DoPost(ctx, "/api/v1/secrets", body, nil); err != nil {
			return err
		}

		fmt.Printf("Secret '%s' created.\n", name)
	}

	fmt.Printf("The secret will be available as FLUXBASE_SECRET_%s in edge functions.\n", strings.ToUpper(name))

	return nil
}

func runSecretsGet(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find the secret by name
	params := url.Values{}
	params.Set("name", name)
	if secretNamespace != "" {
		params.Set("namespace", secretNamespace)
	}

	var secrets []map[string]interface{}
	path := "/api/v1/secrets?" + params.Encode()

	if err := apiClient.DoGet(ctx, path, nil, &secrets); err != nil {
		return err
	}

	// Find exact match
	var secret map[string]interface{}
	for _, s := range secrets {
		sName := getStringValue(s, "name")
		sNamespace := getStringValue(s, "namespace")

		if sName == name {
			if secretNamespace != "" && sNamespace == secretNamespace {
				secret = s
				break
			} else if secretNamespace == "" && sNamespace == "" {
				secret = s
				break
			}
		}
	}

	if secret == nil {
		return fmt.Errorf("secret '%s' not found", name)
	}

	formatter := GetFormatter()
	return formatter.Print(secret)
}

func runSecretsDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find the secret by name first
	params := url.Values{}
	if secretNamespace != "" {
		params.Set("namespace", secretNamespace)
	}

	var secrets []map[string]interface{}
	path := "/api/v1/secrets"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	if err := apiClient.DoGet(ctx, path, nil, &secrets); err != nil {
		return err
	}

	// Find exact match
	var secretID string
	for _, s := range secrets {
		sName := getStringValue(s, "name")
		sNamespace := getStringValue(s, "namespace")

		if sName == name {
			if secretNamespace != "" && sNamespace == secretNamespace {
				secretID = getStringValue(s, "id")
				break
			} else if secretNamespace == "" && sNamespace == "" {
				secretID = getStringValue(s, "id")
				break
			}
		}
	}

	if secretID == "" {
		return fmt.Errorf("secret '%s' not found", name)
	}

	if err := apiClient.DoDelete(ctx, "/api/v1/secrets/"+url.PathEscape(secretID)); err != nil {
		return err
	}

	fmt.Printf("Secret '%s' deleted.\n", name)
	return nil
}

func runSecretsHistory(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find the secret by name first
	params := url.Values{}
	if secretNamespace != "" {
		params.Set("namespace", secretNamespace)
	}

	var secrets []map[string]interface{}
	listPath := "/api/v1/secrets"
	if len(params) > 0 {
		listPath += "?" + params.Encode()
	}

	if err := apiClient.DoGet(ctx, listPath, nil, &secrets); err != nil {
		return err
	}

	// Find exact match
	var secretID string
	for _, s := range secrets {
		sName := getStringValue(s, "name")
		sNamespace := getStringValue(s, "namespace")

		if sName == name {
			if secretNamespace != "" && sNamespace == secretNamespace {
				secretID = getStringValue(s, "id")
				break
			} else if secretNamespace == "" && sNamespace == "" {
				secretID = getStringValue(s, "id")
				break
			}
		}
	}

	if secretID == "" {
		return fmt.Errorf("secret '%s' not found", name)
	}

	// Get version history
	var versions []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/secrets/"+url.PathEscape(secretID)+"/versions", nil, &versions); err != nil {
		return err
	}

	if len(versions) == 0 {
		fmt.Println("No version history found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"VERSION", "CREATED AT", "CREATED BY"},
			Rows:    make([][]string, len(versions)),
		}

		for i, v := range versions {
			version := fmt.Sprintf("%v", v["version"])
			createdAt := getStringValue(v, "created_at")
			createdBy := getStringValue(v, "created_by")
			if createdBy == "" {
				createdBy = "-"
			}

			data.Rows[i] = []string{version, createdAt, createdBy}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(versions); err != nil {
			return err
		}
	}

	return nil
}

func runSecretsRollback(cmd *cobra.Command, args []string) error {
	name := args[0]
	version := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Find the secret by name first
	params := url.Values{}
	if secretNamespace != "" {
		params.Set("namespace", secretNamespace)
	}

	var secrets []map[string]interface{}
	listPath := "/api/v1/secrets"
	if len(params) > 0 {
		listPath += "?" + params.Encode()
	}

	if err := apiClient.DoGet(ctx, listPath, nil, &secrets); err != nil {
		return err
	}

	// Find exact match
	var secretID string
	for _, s := range secrets {
		sName := getStringValue(s, "name")
		sNamespace := getStringValue(s, "namespace")

		if sName == name {
			if secretNamespace != "" && sNamespace == secretNamespace {
				secretID = getStringValue(s, "id")
				break
			} else if secretNamespace == "" && sNamespace == "" {
				secretID = getStringValue(s, "id")
				break
			}
		}
	}

	if secretID == "" {
		return fmt.Errorf("secret '%s' not found", name)
	}

	// Rollback to version
	if err := apiClient.DoPost(ctx, "/api/v1/secrets/"+url.PathEscape(secretID)+"/rollback/"+url.PathEscape(version), nil, nil); err != nil {
		return err
	}

	fmt.Printf("Secret '%s' rolled back to version %s.\n", name, version)
	return nil
}

// parseDuration parses duration strings like "30d", "1y", "24h"
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]

	var value int
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value: %s", s)
	}

	switch unit {
	case 's':
		return time.Duration(value) * time.Second, nil
	case 'm':
		return time.Duration(value) * time.Minute, nil
	case 'h':
		return time.Duration(value) * time.Hour, nil
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case 'y':
		return time.Duration(value) * 365 * 24 * time.Hour, nil
	default:
		// Try standard Go duration parsing
		return time.ParseDuration(s)
	}
}
