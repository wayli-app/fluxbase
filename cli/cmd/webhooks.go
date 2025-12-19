package cmd

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var webhooksCmd = &cobra.Command{
	Use:     "webhooks",
	Aliases: []string{"webhook", "wh"},
	Short:   "Manage webhooks",
	Long:    `Create, configure, and manage webhooks.`,
}

var (
	whURL     string
	whEvents  string
	whSecret  string
	whEnabled bool
)

var webhooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all webhooks",
	Long: `List all configured webhooks.

Examples:
  fluxbase webhooks list`,
	PreRunE: requireAuth,
	RunE:    runWebhooksList,
}

var webhooksGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get webhook details",
	Long: `Get details of a specific webhook.

Examples:
  fluxbase webhooks get abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runWebhooksGet,
}

var webhooksCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new webhook",
	Long: `Create a new webhook.

Examples:
  fluxbase webhooks create --url https://example.com/webhook --events "INSERT,UPDATE"`,
	PreRunE: requireAuth,
	RunE:    runWebhooksCreate,
}

var webhooksUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a webhook",
	Long: `Update an existing webhook.

Examples:
  fluxbase webhooks update abc123 --url https://new-url.com/webhook`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runWebhooksUpdate,
}

var webhooksDeleteCmd = &cobra.Command{
	Use:     "delete [id]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a webhook",
	Long: `Delete a webhook.

Examples:
  fluxbase webhooks delete abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runWebhooksDelete,
}

var webhooksTestCmd = &cobra.Command{
	Use:   "test [id]",
	Short: "Send a test webhook",
	Long: `Send a test webhook to verify configuration.

Examples:
  fluxbase webhooks test abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runWebhooksTest,
}

var webhooksDeliveriesCmd = &cobra.Command{
	Use:   "deliveries [id]",
	Short: "List webhook deliveries",
	Long: `List recent delivery attempts for a webhook.

Examples:
  fluxbase webhooks deliveries abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runWebhooksDeliveries,
}

func init() {
	// Create flags
	webhooksCreateCmd.Flags().StringVar(&whURL, "url", "", "Webhook URL (required)")
	webhooksCreateCmd.Flags().StringVar(&whEvents, "events", "", "Events to subscribe to (comma-separated)")
	webhooksCreateCmd.Flags().StringVar(&whSecret, "secret", "", "Webhook secret for signature verification")
	_ = webhooksCreateCmd.MarkFlagRequired("url")

	// Update flags
	webhooksUpdateCmd.Flags().StringVar(&whURL, "url", "", "Webhook URL")
	webhooksUpdateCmd.Flags().StringVar(&whEvents, "events", "", "Events to subscribe to")
	webhooksUpdateCmd.Flags().BoolVar(&whEnabled, "enabled", true, "Enable/disable webhook")

	webhooksCmd.AddCommand(webhooksListCmd)
	webhooksCmd.AddCommand(webhooksGetCmd)
	webhooksCmd.AddCommand(webhooksCreateCmd)
	webhooksCmd.AddCommand(webhooksUpdateCmd)
	webhooksCmd.AddCommand(webhooksDeleteCmd)
	webhooksCmd.AddCommand(webhooksTestCmd)
	webhooksCmd.AddCommand(webhooksDeliveriesCmd)
}

func runWebhooksList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var webhooks []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/webhooks", nil, &webhooks); err != nil {
		return err
	}

	if len(webhooks) == 0 {
		fmt.Println("No webhooks found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"ID", "URL", "EVENTS", "ENABLED"},
			Rows:    make([][]string, len(webhooks)),
		}

		for i, wh := range webhooks {
			id := getStringValue(wh, "id")
			whURL := getStringValue(wh, "url")
			events := getStringValue(wh, "events")
			enabled := fmt.Sprintf("%v", wh["enabled"])

			data.Rows[i] = []string{id, whURL, events, enabled}
		}

		formatter.PrintTable(data)
	} else {
		formatter.Print(webhooks)
	}

	return nil
}

func runWebhooksGet(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var webhook map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/webhooks/"+url.PathEscape(id), nil, &webhook); err != nil {
		return err
	}

	formatter := GetFormatter()
	return formatter.Print(webhook)
}

func runWebhooksCreate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"url":     whURL,
		"enabled": true,
	}

	if whEvents != "" {
		body["events"] = whEvents
	}
	if whSecret != "" {
		body["secret"] = whSecret
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/webhooks", body, &result); err != nil {
		return err
	}

	id := getStringValue(result, "id")
	fmt.Printf("Webhook created with ID: %s\n", id)
	return nil
}

func runWebhooksUpdate(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := make(map[string]interface{})

	if whURL != "" {
		body["url"] = whURL
	}
	if whEvents != "" {
		body["events"] = whEvents
	}
	if cmd.Flags().Changed("enabled") {
		body["enabled"] = whEnabled
	}

	if len(body) == 0 {
		return fmt.Errorf("no updates specified")
	}

	if err := apiClient.DoPatch(ctx, "/api/v1/webhooks/"+url.PathEscape(id), body, nil); err != nil {
		return err
	}

	fmt.Printf("Webhook '%s' updated.\n", id)
	return nil
}

func runWebhooksDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoDelete(ctx, "/api/v1/webhooks/"+url.PathEscape(id)); err != nil {
		return err
	}

	fmt.Printf("Webhook '%s' deleted.\n", id)
	return nil
}

func runWebhooksTest(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/webhooks/"+url.PathEscape(id)+"/test", nil, &result); err != nil {
		return err
	}

	success := result["success"]
	if success == true {
		fmt.Println("Test webhook sent successfully.")
	} else {
		errMsg := getStringValue(result, "error")
		fmt.Printf("Test webhook failed: %s\n", errMsg)
	}

	return nil
}

func runWebhooksDeliveries(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var deliveries []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/webhooks/"+url.PathEscape(id)+"/deliveries", nil, &deliveries); err != nil {
		return err
	}

	if len(deliveries) == 0 {
		fmt.Println("No deliveries found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"ID", "STATUS", "RESPONSE", "TIMESTAMP"},
			Rows:    make([][]string, len(deliveries)),
		}

		for i, d := range deliveries {
			did := getStringValue(d, "id")
			status := fmt.Sprintf("%d", getIntValue(d, "response_status"))
			response := getStringValue(d, "response_body")
			if len(response) > 50 {
				response = response[:50] + "..."
			}
			timestamp := getStringValue(d, "created_at")

			data.Rows[i] = []string{did, status, response, timestamp}
		}

		formatter.PrintTable(data)
	} else {
		formatter.Print(deliveries)
	}

	return nil
}
