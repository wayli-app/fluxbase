package cmd

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/cobra"

	"github.com/nimbleflux/fluxbase/cli/output"
)

var aiProvidersCmd = &cobra.Command{
	Use:     "providers",
	Aliases: []string{"provider"},
	Short:   "Manage AI providers",
	Long:    `Create and manage AI providers (OpenAI, Azure, Ollama).`,
}

var (
	provType        string
	provDisplayName string
	provAPIKey      string
	provBaseURL     string
	provModel       string
	provIsDefault   bool
	provEnabled     bool
	provEnabledSet  bool
	provEmbedModel  string
)

var aiProvidersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List AI providers",
	Long: `List all configured AI providers.

Examples:
  fluxbase ai providers list
  fluxbase ai providers list -o json`,
	PreRunE: requireAuth,
	RunE:    runAIProvidersList,
}

var aiProvidersGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get AI provider details",
	Long: `Get details of a specific AI provider.

Examples:
  fluxbase ai providers get abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAIProvidersGet,
}

var aiProvidersCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create an AI provider",
	Long: `Create a new AI provider.

Examples:
  fluxbase ai providers create my-openai --type openai --api-key sk-xxx --model gpt-4
  fluxbase ai providers create my-ollama --type ollama --base-url http://localhost:11434`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAIProvidersCreate,
}

var aiProvidersUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update an AI provider",
	Long: `Update an existing AI provider.

Examples:
  fluxbase ai providers update abc123 --model gpt-4o
  fluxbase ai providers update abc123 --enabled=false`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAIProvidersUpdate,
}

var aiProvidersDeleteCmd = &cobra.Command{
	Use:     "delete [id]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete an AI provider",
	Long: `Delete an AI provider.

Examples:
  fluxbase ai providers delete abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAIProvidersDelete,
}

var aiProvidersSetDefaultCmd = &cobra.Command{
	Use:   "set-default [id]",
	Short: "Set a provider as the default",
	Long: `Set an AI provider as the default provider for chat completion.

Examples:
  fluxbase ai providers set-default abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAIProvidersSetDefault,
}

var aiProvidersSetEmbeddingCmd = &cobra.Command{
	Use:   "set-embedding [id]",
	Short: "Set a provider as the embedding provider",
	Long: `Set an AI provider as the dedicated embedding provider.

Examples:
  fluxbase ai providers set-embedding abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAIProvidersSetEmbedding,
}

var aiProvidersClearEmbeddingCmd = &cobra.Command{
	Use:   "clear-embedding [id]",
	Short: "Clear the dedicated embedding provider",
	Long: `Remove the dedicated embedding provider setting.

Examples:
  fluxbase ai providers clear-embedding abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runAIProvidersClearEmbedding,
}

func init() {
	aiProvidersCreateCmd.Flags().StringVar(&provType, "type", "openai", "Provider type: openai, azure, ollama")
	aiProvidersCreateCmd.Flags().StringVar(&provDisplayName, "display-name", "", "Display name")
	aiProvidersCreateCmd.Flags().StringVar(&provAPIKey, "api-key", "", "API key")
	aiProvidersCreateCmd.Flags().StringVar(&provBaseURL, "base-url", "", "Base URL (for Ollama/custom)")
	aiProvidersCreateCmd.Flags().StringVar(&provModel, "model", "", "Default model")
	aiProvidersCreateCmd.Flags().BoolVar(&provIsDefault, "default", false, "Set as default provider")
	aiProvidersCreateCmd.Flags().BoolVar(&provEnabled, "enabled", true, "Enable provider")

	aiProvidersUpdateCmd.Flags().StringVar(&provDisplayName, "display-name", "", "Display name")
	aiProvidersUpdateCmd.Flags().StringVar(&provAPIKey, "api-key", "", "API key")
	aiProvidersUpdateCmd.Flags().StringVar(&provBaseURL, "base-url", "", "Base URL")
	aiProvidersUpdateCmd.Flags().StringVar(&provModel, "model", "", "Default model")
	aiProvidersUpdateCmd.Flags().StringVar(&provEmbedModel, "embedding-model", "", "Embedding model")
	aiProvidersUpdateCmd.Flags().BoolVar(&provEnabled, "enabled", true, "Enable provider")
	aiProvidersUpdateCmd.Flags().BoolVar(&provEnabledSet, "set-enabled", false, "Explicitly set enabled flag")

	aiProvidersCmd.AddCommand(aiProvidersListCmd)
	aiProvidersCmd.AddCommand(aiProvidersGetCmd)
	aiProvidersCmd.AddCommand(aiProvidersCreateCmd)
	aiProvidersCmd.AddCommand(aiProvidersUpdateCmd)
	aiProvidersCmd.AddCommand(aiProvidersDeleteCmd)
	aiProvidersCmd.AddCommand(aiProvidersSetDefaultCmd)
	aiProvidersCmd.AddCommand(aiProvidersSetEmbeddingCmd)
	aiProvidersCmd.AddCommand(aiProvidersClearEmbeddingCmd)
}

func runAIProvidersList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response struct {
		Providers []map[string]interface{} `json:"providers"`
		Count     int                      `json:"count"`
	}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/ai/providers", nil, &response); err != nil {
		return err
	}

	providers := response.Providers
	if len(providers) == 0 {
		fmt.Println("No AI providers found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"ID", "NAME", "TYPE", "DEFAULT", "EMBEDDING", "ENABLED"},
			Rows:    make([][]string, len(providers)),
		}

		for i, p := range providers {
			isDefault := ""
			if d, _ := p["is_default"].(bool); d {
				isDefault = "*"
			}
			isEmbed := ""
			if e := p["use_for_embeddings"]; e != nil {
				if b, _ := e.(bool); b {
					isEmbed = "*"
				}
			}
			data.Rows[i] = []string{
				getStringValue(p, "id"),
				getStringValue(p, "name"),
				getStringValue(p, "provider_type"),
				isDefault,
				isEmbed,
				fmt.Sprintf("%v", p["enabled"]),
			}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(providers); err != nil {
			return err
		}
	}

	return nil
}

func runAIProvidersGet(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var provider map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/ai/providers/"+url.PathEscape(id), nil, &provider); err != nil {
		return err
	}

	formatter := GetFormatter()
	return formatter.Print(provider)
}

func runAIProvidersCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"name":          name,
		"provider_type": provType,
		"enabled":       provEnabled,
		"is_default":    provIsDefault,
	}

	if provDisplayName != "" {
		body["display_name"] = provDisplayName
	}

	config := make(map[string]interface{})
	if provAPIKey != "" {
		config["api_key"] = provAPIKey
	}
	if provBaseURL != "" {
		config["base_url"] = provBaseURL
	}
	if provModel != "" {
		config["model"] = provModel
	}
	if len(config) > 0 {
		body["config"] = config
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/ai/providers", body, &result); err != nil {
		return err
	}

	id := getStringValue(result, "id")
	fmt.Printf("AI provider '%s' created with ID: %s\n", name, id)
	return nil
}

func runAIProvidersUpdate(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := make(map[string]interface{})

	if provDisplayName != "" {
		body["display_name"] = provDisplayName
	}
	if provEnabledSet || cmd.Flags().Changed("enabled") {
		body["enabled"] = provEnabled
	}
	if provEmbedModel != "" {
		body["embedding_model"] = provEmbedModel
	}

	config := make(map[string]interface{})
	if cmd.Flags().Changed("api-key") {
		config["api_key"] = provAPIKey
	}
	if cmd.Flags().Changed("base-url") {
		config["base_url"] = provBaseURL
	}
	if cmd.Flags().Changed("model") {
		config["model"] = provModel
	}
	if len(config) > 0 {
		body["config"] = config
	}

	if len(body) == 0 {
		return fmt.Errorf("no updates specified")
	}

	if err := apiClient.DoPut(ctx, "/api/v1/admin/ai/providers/"+url.PathEscape(id), body, nil); err != nil {
		return err
	}

	fmt.Printf("AI provider '%s' updated.\n", id)
	return nil
}

func runAIProvidersDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoDelete(ctx, "/api/v1/admin/ai/providers/"+url.PathEscape(id)); err != nil {
		return err
	}

	fmt.Printf("AI provider '%s' deleted.\n", id)
	return nil
}

func runAIProvidersSetDefault(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoPut(ctx, "/api/v1/admin/ai/providers/"+url.PathEscape(id)+"/default", nil, nil); err != nil {
		return err
	}

	fmt.Printf("AI provider '%s' set as default.\n", id)
	return nil
}

func runAIProvidersSetEmbedding(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoPut(ctx, "/api/v1/admin/ai/providers/"+url.PathEscape(id)+"/embedding", nil, nil); err != nil {
		return err
	}

	fmt.Printf("AI provider '%s' set as embedding provider.\n", id)
	return nil
}

func runAIProvidersClearEmbedding(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoDelete(ctx, "/api/v1/admin/ai/providers/"+url.PathEscape(id)+"/embedding"); err != nil {
		return err
	}

	fmt.Printf("Embedding provider cleared for '%s'.\n", id)
	return nil
}
