package cmd

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/cobra"

	"github.com/nimbleflux/fluxbase/cli/output"
)

var kbCmd = &cobra.Command{
	Use:     "kb",
	Aliases: []string{"knowledge-bases", "knowledge-base"},
	Short:   "Manage knowledge bases",
	Long:    `Create and manage knowledge bases for AI chatbots.`,
}

var (
	kbNamespace      string
	kbDescription    string
	kbChunkSize      int
	kbChunkOverlap   int
	kbEmbeddingModel string
	kbEmbeddingDims  int
	kbChunkStrategy  string
	kbVisibility     string
	kbName           string
	kbEnabled        bool
	kbEnabledSet     bool
)

var kbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List knowledge bases",
	Long: `List all knowledge bases.

Examples:
  fluxbase kb list
  fluxbase kb list --namespace production -o json`,
	PreRunE: requireAuth,
	RunE:    runKBList,
}

var kbGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get knowledge base details",
	Long: `Get details of a specific knowledge base.

Examples:
  fluxbase kb get abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runKBGet,
}

var kbCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a knowledge base",
	Long: `Create a new knowledge base for AI chatbots.

Examples:
  fluxbase kb create my-kb --namespace default --embedding-model text-embedding-3-small
  fluxbase kb create docs-kb --chunk-size 500 --visibility shared -o json`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runKBCreate,
}

var kbUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a knowledge base",
	Long: `Update an existing knowledge base.

Examples:
  fluxbase kb update abc123 --description "Updated description"
  fluxbase kb update abc123 --enabled=false --chunk-size 1000`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runKBUpdate,
}

var kbDeleteCmd = &cobra.Command{
	Use:     "delete [id]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a knowledge base",
	Long: `Delete a knowledge base and all its documents.

Examples:
  fluxbase kb delete abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runKBDelete,
}

func init() {
	kbListCmd.Flags().StringVar(&kbNamespace, "namespace", "", "Filter by namespace")

	kbCreateCmd.Flags().StringVar(&kbNamespace, "namespace", "default", "Namespace")
	kbCreateCmd.Flags().StringVar(&kbDescription, "description", "", "Description")
	kbCreateCmd.Flags().IntVar(&kbChunkSize, "chunk-size", 0, "Chunk size in tokens (default: 512)")
	kbCreateCmd.Flags().IntVar(&kbChunkOverlap, "chunk-overlap", 0, "Chunk overlap in tokens (default: 50)")
	kbCreateCmd.Flags().StringVar(&kbEmbeddingModel, "embedding-model", "", "Embedding model (default: text-embedding-3-small)")
	kbCreateCmd.Flags().IntVar(&kbEmbeddingDims, "embedding-dimensions", 0, "Embedding dimensions (must be 1536)")
	kbCreateCmd.Flags().StringVar(&kbChunkStrategy, "chunk-strategy", "", "Chunking strategy: recursive, sentence, paragraph, fixed")
	kbCreateCmd.Flags().StringVar(&kbVisibility, "visibility", "", "Visibility: private, shared, public (default: private)")

	kbUpdateCmd.Flags().StringVar(&kbName, "name", "", "New name")
	kbUpdateCmd.Flags().StringVar(&kbDescription, "description", "", "Description")
	kbUpdateCmd.Flags().IntVar(&kbChunkSize, "chunk-size", 0, "Chunk size in tokens")
	kbUpdateCmd.Flags().IntVar(&kbChunkOverlap, "chunk-overlap", 0, "Chunk overlap in tokens")
	kbUpdateCmd.Flags().StringVar(&kbEmbeddingModel, "embedding-model", "", "Embedding model")
	kbUpdateCmd.Flags().StringVar(&kbChunkStrategy, "chunk-strategy", "", "Chunking strategy: recursive, sentence, paragraph, fixed")
	kbUpdateCmd.Flags().StringVar(&kbVisibility, "visibility", "", "Visibility: private, shared, public")
	kbUpdateCmd.Flags().BoolVar(&kbEnabled, "enabled", true, "Enable/disable the knowledge base")
	kbUpdateCmd.Flags().BoolVar(&kbEnabledSet, "set-enabled", false, "Explicitly set enabled flag (use --enabled=true/false with this)")

	kbCmd.AddCommand(kbListCmd)
	kbCmd.AddCommand(kbGetCmd)
	kbCmd.AddCommand(kbCreateCmd)
	kbCmd.AddCommand(kbUpdateCmd)
	kbCmd.AddCommand(kbDeleteCmd)
}

func runKBList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	path := "/api/v1/admin/ai/knowledge-bases"
	params := url.Values{}
	if kbNamespace != "" {
		params.Set("namespace", kbNamespace)
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var response struct {
		KnowledgeBases []map[string]interface{} `json:"knowledge_bases"`
		Count          int                      `json:"count"`
	}
	if err := apiClient.DoGet(ctx, path, nil, &response); err != nil {
		return err
	}

	kbs := response.KnowledgeBases
	if len(kbs) == 0 {
		fmt.Println("No knowledge bases found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"ID", "NAME", "NAMESPACE", "DOCUMENTS", "CHUNKS", "ENABLED"},
			Rows:    make([][]string, len(kbs)),
		}

		for i, kb := range kbs {
			data.Rows[i] = []string{
				getStringValue(kb, "id"),
				getStringValue(kb, "name"),
				getStringValue(kb, "namespace"),
				fmt.Sprintf("%d", getIntValue(kb, "document_count")),
				fmt.Sprintf("%d", getIntValue(kb, "total_chunks")),
				fmt.Sprintf("%v", kb["enabled"]),
			}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(kbs); err != nil {
			return err
		}
	}

	return nil
}

func runKBGet(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var kb map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/ai/knowledge-bases/"+url.PathEscape(id), nil, &kb); err != nil {
		return err
	}

	formatter := GetFormatter()
	return formatter.Print(kb)
}

func runKBCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"name": name,
	}

	if cmd.Flags().Changed("namespace") {
		body["namespace"] = kbNamespace
	}
	if kbDescription != "" {
		body["description"] = kbDescription
	}
	if kbChunkSize > 0 {
		body["chunk_size"] = kbChunkSize
	}
	if kbChunkOverlap > 0 {
		body["chunk_overlap"] = kbChunkOverlap
	}
	if kbEmbeddingModel != "" {
		body["embedding_model"] = kbEmbeddingModel
	}
	if kbEmbeddingDims > 0 {
		body["embedding_dimensions"] = kbEmbeddingDims
	}
	if kbChunkStrategy != "" {
		body["chunk_strategy"] = kbChunkStrategy
	}
	if kbVisibility != "" {
		body["visibility"] = kbVisibility
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/ai/knowledge-bases", body, &result); err != nil {
		return err
	}

	id := getStringValue(result, "id")
	fmt.Printf("Knowledge base '%s' created with ID: %s\n", name, id)
	return nil
}

func runKBUpdate(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := make(map[string]interface{})

	if kbName != "" {
		body["name"] = kbName
	}
	if kbDescription != "" {
		body["description"] = kbDescription
	}
	if cmd.Flags().Changed("chunk-size") {
		body["chunk_size"] = kbChunkSize
	}
	if cmd.Flags().Changed("chunk-overlap") {
		body["chunk_overlap"] = kbChunkOverlap
	}
	if kbEmbeddingModel != "" {
		body["embedding_model"] = kbEmbeddingModel
	}
	if kbChunkStrategy != "" {
		body["chunk_strategy"] = kbChunkStrategy
	}
	if kbVisibility != "" {
		body["visibility"] = kbVisibility
	}
	if kbEnabledSet || cmd.Flags().Changed("enabled") {
		body["enabled"] = kbEnabled
	}

	if len(body) == 0 {
		return fmt.Errorf("no updates specified")
	}

	if err := apiClient.DoPut(ctx, "/api/v1/admin/ai/knowledge-bases/"+url.PathEscape(id), body, nil); err != nil {
		return err
	}

	fmt.Printf("Knowledge base '%s' updated.\n", id)
	return nil
}

func runKBDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoDelete(ctx, "/api/v1/admin/ai/knowledge-bases/"+url.PathEscape(id)); err != nil {
		return err
	}

	fmt.Printf("Knowledge base '%s' deleted.\n", id)
	return nil
}
