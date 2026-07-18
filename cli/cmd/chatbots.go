package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nimbleflux/fluxbase/cli/output"
)

var chatbotsCmd = &cobra.Command{
	Use:     "chatbots",
	Aliases: []string{"chatbot", "cb"},
	Short:   "Manage AI chatbots",
	Long:    `Create, configure, and manage AI chatbots.`,
}

var (
	cbSystemPrompt  string
	cbModel         string
	cbTemperature   float64
	cbMaxTokens     int
	cbKnowledgeBase string
	// cbListNamespace is the optional --namespace filter for `chatbots list`.
	// Empty (default) means list across all namespaces. Kept separate from
	// cbNamespace (used by sync as a target with default "default") so the
	// two commands don't fight over the same var's initial value.
	cbListNamespace string
)

var chatbotsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all chatbots",
	Long: `List all configured chatbots.

Examples:
  fluxbase chatbots list
  fluxbase chatbots list --namespace wayli
  fluxbase chatbots list -o json`,
	PreRunE: requireAuth,
	RunE:    runChatbotsList,
}

var chatbotsGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get chatbot details",
	Long: `Get details of a specific chatbot.

Examples:
  fluxbase chatbots get abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runChatbotsGet,
}

var chatbotsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new chatbot",
	Long: `Create a new AI chatbot.

Examples:
  fluxbase chatbots create support-bot --system-prompt "You are a helpful support assistant"
  fluxbase chatbots create support-bot --model gpt-4 --temperature 0.7`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runChatbotsCreate,
}

var chatbotsUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a chatbot",
	Long: `Update an existing chatbot.

Examples:
  fluxbase chatbots update abc123 --system-prompt "Updated prompt"`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runChatbotsUpdate,
}

var chatbotsDeleteCmd = &cobra.Command{
	Use:     "delete [id]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a chatbot",
	Long: `Delete a chatbot.

Examples:
  fluxbase chatbots delete abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runChatbotsDelete,
}

var chatbotsSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync chatbots from a directory",
	Long: `Sync AI chatbots from TypeScript files.

Examples:
  fluxbase chatbots sync --dir ./chatbots
  fluxbase chatbots sync --dir ./chatbots --namespace production`,
	PreRunE: requireAuth,
	RunE:    runChatbotsSync,
}

var (
	cbSyncDir       string
	cbNamespace     string
	cbDryRun        bool
	cbDeleteMissing bool
)

func init() {
	// List flags
	chatbotsListCmd.Flags().StringVar(&cbListNamespace, "namespace", "", "Filter by namespace")

	// Create flags
	chatbotsCreateCmd.Flags().StringVar(&cbSystemPrompt, "system-prompt", "", "System prompt for the chatbot")
	chatbotsCreateCmd.Flags().StringVar(&cbModel, "model", "gpt-3.5-turbo", "AI model to use")
	chatbotsCreateCmd.Flags().Float64Var(&cbTemperature, "temperature", 0.7, "Temperature (0.0-1.0)")
	chatbotsCreateCmd.Flags().IntVar(&cbMaxTokens, "max-tokens", 1024, "Maximum tokens in response")
	chatbotsCreateCmd.Flags().StringVar(&cbKnowledgeBase, "knowledge-base", "", "Knowledge base ID to attach")

	// Update flags
	chatbotsUpdateCmd.Flags().StringVar(&cbSystemPrompt, "system-prompt", "", "System prompt for the chatbot")
	chatbotsUpdateCmd.Flags().StringVar(&cbModel, "model", "", "AI model to use")
	chatbotsUpdateCmd.Flags().Float64Var(&cbTemperature, "temperature", 0, "Temperature (0.0-1.0)")
	chatbotsUpdateCmd.Flags().IntVar(&cbMaxTokens, "max-tokens", 0, "Maximum tokens in response")

	// Sync flags
	chatbotsSyncCmd.Flags().StringVar(&cbSyncDir, "dir", "./chatbots", "Directory containing chatbot .ts files or folders with index.ts")
	chatbotsSyncCmd.Flags().StringVar(&cbNamespace, "namespace", "default", "Target namespace")
	chatbotsSyncCmd.Flags().BoolVar(&cbDryRun, "dry-run", false, "Preview changes without applying")
	chatbotsSyncCmd.Flags().BoolVar(&cbDeleteMissing, "delete-missing", false, "Delete chatbots not in directory")

	chatbotsCmd.AddCommand(chatbotsListCmd)
	chatbotsCmd.AddCommand(chatbotsGetCmd)
	chatbotsCmd.AddCommand(chatbotsCreateCmd)
	chatbotsCmd.AddCommand(chatbotsUpdateCmd)
	chatbotsCmd.AddCommand(chatbotsDeleteCmd)
	chatbotsCmd.AddCommand(chatbotsSyncCmd)
	chatbotsCmd.AddCommand(chatbotsToggleCmd)
	chatbotsCmd.AddCommand(chatbotsKBCmd)
}

func runChatbotsList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build optional query params. Empty namespace = list across all.
	var query url.Values
	if cbListNamespace != "" {
		query = url.Values{}
		query.Set("namespace", cbListNamespace)
	}

	var response struct {
		Chatbots []map[string]interface{} `json:"chatbots"`
		Count    int                      `json:"count"`
	}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/ai/chatbots", query, &response); err != nil {
		return err
	}
	chatbots := response.Chatbots

	if len(chatbots) == 0 {
		if cbListNamespace != "" {
			fmt.Printf("No chatbots found in namespace '%s'.\n", cbListNamespace)
		} else {
			fmt.Println("No chatbots found.")
		}
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"ID", "NAME", "MODEL", "ENABLED"},
			Rows:    make([][]string, len(chatbots)),
		}

		for i, cb := range chatbots {
			id := getStringValue(cb, "id")
			name := getStringValue(cb, "name")
			model := getStringValue(cb, "model")
			enabled := fmt.Sprintf("%v", cb["enabled"])

			data.Rows[i] = []string{id, name, model, enabled}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(chatbots); err != nil {
			return err
		}
	}

	return nil
}

func runChatbotsGet(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var chatbot map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/ai/chatbots/"+url.PathEscape(id), nil, &chatbot); err != nil {
		return err
	}

	formatter := GetFormatter()
	return formatter.Print(chatbot)
}

func runChatbotsCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"name":        name,
		"model":       cbModel,
		"temperature": cbTemperature,
		"max_tokens":  cbMaxTokens,
		"enabled":     true,
	}

	if cbSystemPrompt != "" {
		body["system_prompt"] = cbSystemPrompt
	}

	if cbKnowledgeBase != "" {
		body["knowledge_base_ids"] = []string{cbKnowledgeBase}
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/ai/chatbots", body, &result); err != nil {
		return err
	}

	id := getStringValue(result, "id")
	fmt.Printf("Chatbot '%s' created with ID: %s\n", name, id)
	return nil
}

func runChatbotsUpdate(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := make(map[string]interface{})

	if cbSystemPrompt != "" {
		body["system_prompt"] = cbSystemPrompt
	}
	if cbModel != "" {
		body["model"] = cbModel
	}
	if cbTemperature > 0 {
		body["temperature"] = cbTemperature
	}
	if cbMaxTokens > 0 {
		body["max_tokens"] = cbMaxTokens
	}

	if len(body) == 0 {
		return fmt.Errorf("no updates specified")
	}

	if err := apiClient.DoPut(ctx, "/api/v1/admin/ai/chatbots/"+url.PathEscape(id), body, nil); err != nil {
		return err
	}

	fmt.Printf("Chatbot '%s' updated.\n", id)
	return nil
}

func runChatbotsDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoDelete(ctx, "/api/v1/admin/ai/chatbots/"+url.PathEscape(id)); err != nil {
		return err
	}

	fmt.Printf("Chatbot '%s' deleted.\n", id)
	return nil
}

func runChatbotsSync(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Auto-detect directory if not explicitly specified
	dir, err := detectResourceDir("chatbots", cbSyncDir, "./chatbots")
	if err != nil {
		return err
	}
	cbSyncDir = dir

	// Read YAML files from directory
	entries, err := os.ReadDir(cbSyncDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var chatbots []map[string]interface{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Only support .ts files
		if !strings.HasSuffix(name, ".ts") {
			continue
		}
		cbName := strings.TrimSuffix(name, ".ts")

		// Read file
		content, err := os.ReadFile(filepath.Join(cbSyncDir, name)) //nolint:gosec // CLI tool reads user-provided file path
		if err != nil {
			fmt.Printf("Warning: failed to read %s: %v\n", name, err)
			continue
		}

		chatbots = append(chatbots, map[string]interface{}{
			"name": cbName,
			"code": string(content),
		})
	}

	if len(chatbots) == 0 {
		fmt.Println("No chatbot TypeScript files (.ts) found in directory.")
		return nil
	}

	if cbDryRun {
		fmt.Println("Dry run - would sync the following chatbots:")
		for _, cb := range chatbots {
			fmt.Printf("  - %s\n", cb["name"])
		}
		return nil
	}

	// Build sync request body
	body := map[string]interface{}{
		"namespace": cbNamespace,
		"chatbots":  chatbots,
		"options": map[string]interface{}{
			"delete_missing": cbDeleteMissing,
			"dry_run":        false,
		},
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/ai/chatbots/sync", body, &result); err != nil {
		return err
	}

	// Parse and display result
	if summary, ok := result["summary"].(map[string]interface{}); ok {
		created := getIntValue(summary, "created")
		updated := getIntValue(summary, "updated")
		deleted := getIntValue(summary, "deleted")
		unchanged := getIntValue(summary, "unchanged")
		errorCount := getIntValue(summary, "errors")

		fmt.Printf("Synced %d chatbots to namespace '%s':\n", len(chatbots), cbNamespace)
		fmt.Printf("  Created: %d\n", created)
		fmt.Printf("  Updated: %d\n", updated)
		fmt.Printf("  Deleted: %d\n", deleted)
		fmt.Printf("  Unchanged: %d\n", unchanged)
		if errorCount > 0 {
			fmt.Printf("  Errors: %d\n", errorCount)
		}
	} else {
		fmt.Printf("Synced %d chatbots to namespace '%s'.\n", len(chatbots), cbNamespace)
	}

	// Display errors if any
	if errors, ok := result["errors"].([]interface{}); ok && len(errors) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range errors {
			if errMap, ok := e.(map[string]interface{}); ok {
				name := getStringValue(errMap, "name")
				errMsg := getStringValue(errMap, "error")
				action := getStringValue(errMap, "action")
				if action != "" {
					fmt.Printf("  - %s (%s): %s\n", name, action, errMsg)
				} else {
					fmt.Printf("  - %s: %s\n", name, errMsg)
				}
			}
		}
	}

	return nil
}

var chatbotsToggleCmd = &cobra.Command{
	Use:   "toggle [id]",
	Short: "Toggle a chatbot enabled/disabled",
	Long: `Toggle a chatbot's enabled state.

Examples:
  fluxbase chatbots toggle abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runChatbotsToggle,
}

func runChatbotsToggle(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/ai/chatbots/"+url.PathEscape(id)+"/toggle", nil, &result); err != nil {
		return err
	}

	enabled := false
	if e, ok := result["enabled"].(bool); ok {
		enabled = e
	}
	fmt.Printf("Chatbot '%s' %s.\n", id, map[bool]string{true: "enabled", false: "disabled"}[enabled])
	return nil
}

var chatbotsKBCmd = &cobra.Command{
	Use:   "kb",
	Short: "Manage chatbot knowledge base links",
	Long:  `Link, unlink, and list knowledge bases attached to chatbots.`,
}

var chatbotsKBListCmd = &cobra.Command{
	Use:   "list [chatbot-id]",
	Short: "List linked knowledge bases",
	Long: `List knowledge bases linked to a chatbot.

Examples:
  fluxbase chatbots kb list abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runChatbotsKBList,
}

var chatbotsKBLinkCmd = &cobra.Command{
	Use:   "link [chatbot-id] [kb-id]",
	Short: "Link a knowledge base to a chatbot",
	Long: `Link a knowledge base to a chatbot for RAG retrieval.

Examples:
  fluxbase chatbots kb link abc123 kb456`,
	Args:    cobra.ExactArgs(2),
	PreRunE: requireAuth,
	RunE:    runChatbotsKBLink,
}

var chatbotsKBUnlinkCmd = &cobra.Command{
	Use:   "unlink [chatbot-id] [kb-id]",
	Short: "Unlink a knowledge base from a chatbot",
	Long: `Remove a knowledge base link from a chatbot.

Examples:
  fluxbase chatbots kb unlink abc123 kb456`,
	Args:    cobra.ExactArgs(2),
	PreRunE: requireAuth,
	RunE:    runChatbotsKBUnlink,
}

func init() {
	chatbotsKBCmd.AddCommand(chatbotsKBListCmd)
	chatbotsKBCmd.AddCommand(chatbotsKBLinkCmd)
	chatbotsKBCmd.AddCommand(chatbotsKBUnlinkCmd)
}

func runChatbotsKBList(cmd *cobra.Command, args []string) error {
	chatbotID := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response struct {
		Links []map[string]interface{} `json:"links"`
		Count int                      `json:"count"`
	}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/ai/chatbots/"+url.PathEscape(chatbotID)+"/knowledge-bases", nil, &response); err != nil {
		return err
	}

	links := response.Links
	if len(links) == 0 {
		fmt.Println("No knowledge bases linked to this chatbot.")
		return nil
	}

	formatter := GetFormatter()
	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"KB ID", "ENABLED", "MAX CHUNKS", "PRIORITY"},
			Rows:    make([][]string, len(links)),
		}
		for i, l := range links {
			maxChunks := "-"
			if mc := getIntValue(l, "max_chunks"); mc > 0 {
				maxChunks = fmt.Sprintf("%d", mc)
			}
			priority := "-"
			if p := getIntValue(l, "priority"); p > 0 {
				priority = fmt.Sprintf("%d", p)
			}
			data.Rows[i] = []string{
				getStringValue(l, "knowledge_base_id"),
				fmt.Sprintf("%v", l["enabled"]),
				maxChunks,
				priority,
			}
		}
		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(links); err != nil {
			return err
		}
	}

	return nil
}

func runChatbotsKBLink(cmd *cobra.Command, args []string) error {
	chatbotID := args[0]
	kbID := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"knowledge_base_id": kbID,
	}

	if err := apiClient.DoPost(ctx, "/api/v1/admin/ai/chatbots/"+url.PathEscape(chatbotID)+"/knowledge-bases", body, nil); err != nil {
		return err
	}

	fmt.Printf("Knowledge base '%s' linked to chatbot '%s'.\n", kbID, chatbotID)
	return nil
}

func runChatbotsKBUnlink(cmd *cobra.Command, args []string) error {
	chatbotID := args[0]
	kbID := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoDelete(ctx, "/api/v1/admin/ai/chatbots/"+url.PathEscape(chatbotID)+"/knowledge-bases/"+url.PathEscape(kbID)); err != nil {
		return err
	}

	fmt.Printf("Knowledge base '%s' unlinked from chatbot '%s'.\n", kbID, chatbotID)
	return nil
}
