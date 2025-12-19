package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/output"
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
)

var chatbotsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all chatbots",
	Long: `List all configured chatbots.

Examples:
  fluxbase chatbots list
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

var chatbotsChatCmd = &cobra.Command{
	Use:   "chat [id]",
	Short: "Start an interactive chat session",
	Long: `Start an interactive chat session with a chatbot.

Examples:
  fluxbase chatbots chat abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runChatbotsChat,
}

func init() {
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

	chatbotsCmd.AddCommand(chatbotsListCmd)
	chatbotsCmd.AddCommand(chatbotsGetCmd)
	chatbotsCmd.AddCommand(chatbotsCreateCmd)
	chatbotsCmd.AddCommand(chatbotsUpdateCmd)
	chatbotsCmd.AddCommand(chatbotsDeleteCmd)
	chatbotsCmd.AddCommand(chatbotsChatCmd)
}

func runChatbotsList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var chatbots []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/ai/chatbots", nil, &chatbots); err != nil {
		return err
	}

	if len(chatbots) == 0 {
		fmt.Println("No chatbots found.")
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

func runChatbotsChat(cmd *cobra.Command, args []string) error {
	chatbotID := args[0]

	fmt.Println("Starting chat session. Type 'exit' or 'quit' to end.")
	fmt.Println("---")

	reader := bufio.NewReader(os.Stdin)

	// Create a conversation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var convResult map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/ai/conversations", map[string]interface{}{
		"chatbot_id": chatbotID,
	}, &convResult); err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}

	conversationID := getStringValue(convResult, "id")

	for {
		fmt.Print("You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		// Send message
		msgCtx, msgCancel := context.WithTimeout(context.Background(), 60*time.Second)

		var msgResult map[string]interface{}
		err = apiClient.DoPost(msgCtx, "/api/v1/ai/conversations/"+url.PathEscape(conversationID)+"/messages", map[string]interface{}{
			"content": input,
		}, &msgResult)
		msgCancel()

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Extract assistant response
		if response, ok := msgResult["response"].(map[string]interface{}); ok {
			content := getStringValue(response, "content")
			fmt.Printf("Assistant: %s\n", content)
		} else if content := getStringValue(msgResult, "content"); content != "" {
			fmt.Printf("Assistant: %s\n", content)
		}

		fmt.Println()
	}

	return nil
}
