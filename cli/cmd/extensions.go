package cmd

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/client"
	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var extensionsCmd = &cobra.Command{
	Use:     "extensions",
	Aliases: []string{"extension", "ext"},
	Short:   "Manage PostgreSQL extensions",
	Long:    `List, enable, and disable PostgreSQL extensions.`,
}

var (
	extSchema string
)

var extensionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all extensions",
	Long: `List all available PostgreSQL extensions.

Examples:
  fluxbase extensions list`,
	PreRunE: requireAuth,
	RunE:    runExtensionsList,
}

var extensionsStatusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Get extension status",
	Long: `Get the status of a specific extension.

Examples:
  fluxbase extensions status pgvector`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runExtensionsStatus,
}

var extensionsEnableCmd = &cobra.Command{
	Use:   "enable [name]",
	Short: "Enable an extension",
	Long: `Enable a PostgreSQL extension.

Examples:
  fluxbase extensions enable pgvector
  fluxbase extensions enable postgis --schema public`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runExtensionsEnable,
}

var extensionsDisableCmd = &cobra.Command{
	Use:   "disable [name]",
	Short: "Disable an extension",
	Long: `Disable a PostgreSQL extension.

Examples:
  fluxbase extensions disable pgvector`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runExtensionsDisable,
}

func init() {
	// Enable flags
	extensionsEnableCmd.Flags().StringVar(&extSchema, "schema", "", "Schema to install extension in")

	extensionsCmd.AddCommand(extensionsListCmd)
	extensionsCmd.AddCommand(extensionsStatusCmd)
	extensionsCmd.AddCommand(extensionsEnableCmd)
	extensionsCmd.AddCommand(extensionsDisableCmd)
}

func runExtensionsList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := apiClient.Get(ctx, "/api/v1/admin/extensions", nil)
	if err != nil {
		return err
	}

	var extensions []map[string]interface{}
	if err := client.DecodeResponse(resp, &extensions); err != nil {
		return err
	}

	if len(extensions) == 0 {
		fmt.Println("No extensions found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"NAME", "VERSION", "INSTALLED", "SCHEMA"},
			Rows:    make([][]string, len(extensions)),
		}

		for i, ext := range extensions {
			name := getStringValue(ext, "name")
			version := getStringValue(ext, "default_version")
			installed := fmt.Sprintf("%v", ext["installed"])
			schema := getStringValue(ext, "schema")
			if schema == "" {
				schema = "-"
			}

			data.Rows[i] = []string{name, version, installed, schema}
		}

		formatter.PrintTable(data)
	} else {
		formatter.Print(extensions)
	}

	return nil
}

func runExtensionsStatus(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := apiClient.Get(ctx, "/api/v1/admin/extensions/"+url.PathEscape(name)+"/status", nil)
	if err != nil {
		return err
	}

	var status map[string]interface{}
	if err := client.DecodeResponse(resp, &status); err != nil {
		return err
	}

	formatter := GetFormatter()
	return formatter.Print(status)
}

func runExtensionsEnable(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	body := make(map[string]interface{})
	if extSchema != "" {
		body["schema"] = extSchema
	}

	resp, err := apiClient.Post(ctx, "/api/v1/admin/extensions/"+url.PathEscape(name)+"/enable", body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return client.ParseError(resp)
	}
	resp.Body.Close()

	fmt.Printf("Extension '%s' enabled.\n", name)
	return nil
}

func runExtensionsDisable(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := apiClient.Post(ctx, "/api/v1/admin/extensions/"+url.PathEscape(name)+"/disable", nil)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return client.ParseError(resp)
	}
	resp.Body.Close()

	fmt.Printf("Extension '%s' disabled.\n", name)
	return nil
}
