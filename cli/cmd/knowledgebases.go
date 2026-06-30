package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	// Entity/graph/documents flags
	kbEntityType        string
	kbEntitySearch      string
	kbEntityLimit       int
	kbDocLimit          int
	kbDocTitle          string
	kbDocTag            []string
	kbExtractNoEntities bool
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

var kbEntitiesCmd = &cobra.Command{
	Use:   "entities [kb-id]",
	Short: "List or search entities in a knowledge base",
	Long: `List entities extracted from a knowledge base's documents, or search them by name.

Examples:
  fluxbase kb entities abc123
  fluxbase kb entities abc123 --type person --limit 20
  fluxbase kb entities abc123 --search "Acme"`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runKBEntities,
}

var kbGraphCmd = &cobra.Command{
	Use:   "graph [kb-id]",
	Short: "Show the knowledge graph (entities + relationships)",
	Long: `Display a summary of the knowledge graph for a knowledge base.

Examples:
  fluxbase kb graph abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runKBGraph,
}

var kbChatbotsCmd = &cobra.Command{
	Use:   "chatbots [kb-id]",
	Short: "List chatbots linked to a knowledge base",
	Long: `List all chatbots that have a knowledge base linked for RAG.

Examples:
  fluxbase kb chatbots abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runKBChatbots,
}

var kbDocumentsCmd = &cobra.Command{
	Use:     "documents [kb-id]",
	Aliases: []string{"docs"},
	Short:   "List documents in a knowledge base",
	Long: `List documents in a knowledge base.

Examples:
  fluxbase kb documents abc123 --limit 50`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runKBDocuments,
}

var kbUploadCmd = &cobra.Command{
	Use:   "upload [kb-id] [file]",
	Short: "Upload a file as a document",
	Long: `Upload a file (PDF, text, markdown, etc.) to a knowledge base.

Examples:
  fluxbase kb upload abc123 ./docs/guide.md
  fluxbase kb upload abc123 report.pdf --title "Q4 Report" --tag finance`,
	Args:    cobra.ExactArgs(2),
	PreRunE: requireAuth,
	RunE:    runKBUpload,
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
	kbUpdateCmd.Flags().BoolVar(&kbExtractNoEntities, "entity-extraction", true, "Enable/disable rule-based entity extraction for this KB")

	// Entity/graph/documents flags
	kbEntitiesCmd.Flags().StringVar(&kbEntityType, "type", "", "Filter entities by type (person, organization, location, etc.)")
	kbEntitiesCmd.Flags().StringVar(&kbEntitySearch, "search", "", "Search entities by name (substring match)")
	kbEntitiesCmd.Flags().IntVar(&kbEntityLimit, "limit", 50, "Maximum number of entities to return")

	kbDocumentsCmd.Flags().IntVar(&kbDocLimit, "limit", 50, "Maximum number of documents to return")

	kbUploadCmd.Flags().StringVar(&kbDocTitle, "title", "", "Document title (defaults to filename)")
	kbUploadCmd.Flags().StringArrayVar(&kbDocTag, "tag", nil, "Tag to attach (can be repeated)")

	kbCmd.AddCommand(kbListCmd)
	kbCmd.AddCommand(kbGetCmd)
	kbCmd.AddCommand(kbCreateCmd)
	kbCmd.AddCommand(kbUpdateCmd)
	kbCmd.AddCommand(kbDeleteCmd)
	kbCmd.AddCommand(kbEntitiesCmd)
	kbCmd.AddCommand(kbGraphCmd)
	kbCmd.AddCommand(kbChatbotsCmd)
	kbCmd.AddCommand(kbDocumentsCmd)
	kbCmd.AddCommand(kbUploadCmd)
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
	if cmd.Flags().Changed("entity-extraction") {
		body["entity_extraction_enabled"] = kbExtractNoEntities
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

// runKBEntities lists or searches entities in a knowledge base.
func runKBEntities(cmd *cobra.Command, args []string) error {
	kbID := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response map[string]interface{}

	if kbEntitySearch != "" {
		path := fmt.Sprintf("/api/v1/admin/ai/knowledge-bases/%s/entities/search?q=%s&limit=%d",
			url.PathEscape(kbID), url.QueryEscape(kbEntitySearch), kbEntityLimit)
		if err := apiClient.DoGet(ctx, path, nil, &response); err != nil {
			return err
		}
	} else {
		params := url.Values{}
		params.Set("limit", fmt.Sprintf("%d", kbEntityLimit))
		if kbEntityType != "" {
			params.Set("type", kbEntityType)
		}
		path := "/api/v1/admin/ai/knowledge-bases/" + url.PathEscape(kbID) + "/entities?" + params.Encode()
		if err := apiClient.DoGet(ctx, path, nil, &response); err != nil {
			return err
		}
	}

	entities := getSlice(response, "entities")
	if len(entities) == 0 {
		fmt.Println("No entities found.")
		return nil
	}

	formatter := GetFormatter()
	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"ID", "TYPE", "NAME", "CANONICAL"},
			Rows:    make([][]string, len(entities)),
		}
		for i, e := range entities {
			data.Rows[i] = []string{
				getStringValue(e, "id"),
				getStringValue(e, "entity_type"),
				getStringValue(e, "name"),
				getStringValue(e, "canonical_name"),
			}
		}
		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(entities); err != nil {
			return err
		}
	}
	return nil
}

// runKBGraph shows a summary of the knowledge graph for a KB.
func runKBGraph(cmd *cobra.Command, args []string) error {
	kbID := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var graph map[string]interface{}
	path := "/api/v1/admin/ai/knowledge-bases/" + url.PathEscape(kbID) + "/graph"
	if err := apiClient.DoGet(ctx, path, nil, &graph); err != nil {
		return err
	}

	formatter := GetFormatter()
	if formatter.Format == output.FormatTable {
		fmt.Printf("Entities:     %d\n", getIntValue(graph, "entity_count"))
		fmt.Printf("Relationships: %d\n", getIntValue(graph, "relationship_count"))
		fmt.Println()

		entities := getSlice(graph, "entities")
		if len(entities) > 0 {
			fmt.Println("Top entities:")
			data := output.TableData{
				Headers: []string{"TYPE", "NAME", "DOCUMENTS"},
				Rows:    make([][]string, 0, len(entities)),
			}
			for _, e := range entities {
				docCount := 0
				if meta, ok := e["metadata"].(map[string]interface{}); ok {
					docCount = getIntValue(meta, "document_count")
				}
				data.Rows = append(data.Rows, []string{
					getStringValue(e, "entity_type"),
					getStringValue(e, "canonical_name"),
					fmt.Sprintf("%d", docCount),
				})
			}
			formatter.PrintTable(data)
		}
	} else {
		if err := formatter.Print(graph); err != nil {
			return err
		}
	}
	return nil
}

// runKBChatbots lists chatbots linked to a KB.
func runKBChatbots(cmd *cobra.Command, args []string) error {
	kbID := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response map[string]interface{}
	path := "/api/v1/admin/ai/knowledge-bases/" + url.PathEscape(kbID) + "/chatbots"
	if err := apiClient.DoGet(ctx, path, nil, &response); err != nil {
		return err
	}

	chatbots := getSlice(response, "chatbots")
	if len(chatbots) == 0 {
		fmt.Println("No chatbots linked to this knowledge base.")
		return nil
	}

	formatter := GetFormatter()
	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"ID", "NAME", "NAMESPACE", "ENABLED"},
			Rows:    make([][]string, len(chatbots)),
		}
		for i, cb := range chatbots {
			data.Rows[i] = []string{
				getStringValue(cb, "id"),
				getStringValue(cb, "name"),
				getStringValue(cb, "namespace"),
				fmt.Sprintf("%v", cb["enabled"]),
			}
		}
		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(chatbots); err != nil {
			return err
		}
	}
	return nil
}

// runKBDocuments lists documents in a KB.
func runKBDocuments(cmd *cobra.Command, args []string) error {
	kbID := args[0]
	limit := kbDocLimit
	if limit <= 0 {
		limit = 50
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response map[string]interface{}
	path := fmt.Sprintf("/api/v1/admin/ai/knowledge-bases/%s/documents?limit=%d",
		url.PathEscape(kbID), limit)
	if err := apiClient.DoGet(ctx, path, nil, &response); err != nil {
		return err
	}

	docs := getSlice(response, "documents")
	if len(docs) == 0 {
		fmt.Println("No documents found.")
		return nil
	}

	formatter := GetFormatter()
	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"ID", "TITLE", "STATUS", "CHUNKS"},
			Rows:    make([][]string, len(docs)),
		}
		for i, d := range docs {
			data.Rows[i] = []string{
				getStringValue(d, "id"),
				getStringValue(d, "title"),
				getStringValue(d, "status"),
				fmt.Sprintf("%d", getIntValue(d, "total_chunks")),
			}
		}
		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(docs); err != nil {
			return err
		}
	}
	return nil
}

// runKBUpload uploads a file as a document to a KB via multipart form.
func runKBUpload(cmd *cobra.Command, args []string) error {
	kbID := args[0]
	localFile := args[1]

	fileData, err := os.ReadFile(localFile) //nolint:gosec // CLI tool reads user-provided file path
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	title := kbDocTitle
	if title == "" {
		title = filepath.Base(localFile)
	}

	// Build multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file part
	part, err := writer.CreateFormFile("file", filepath.Base(localFile))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(fileData); err != nil {
		return fmt.Errorf("failed to write file part: %w", err)
	}

	// Add title field
	if err := writer.WriteField("title", title); err != nil {
		return fmt.Errorf("failed to write title field: %w", err)
	}
	// Add tags
	for _, tag := range kbDocTag {
		if err := writer.WriteField("tags", tag); err != nil {
			return fmt.Errorf("failed to write tag field: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	uploadURL := apiClient.BaseURL + "/api/v1/admin/ai/knowledge-bases/" + url.PathEscape(kbID) + "/documents/upload"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Auth
	creds, err := apiClient.CredentialManager.GetCredentials(apiClient.Profile.Name)
	if err != nil {
		return err
	}
	if creds != nil {
		if creds.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
		} else if creds.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+creds.APIKey)
		}
	}

	resp, err := apiClient.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("upload failed (%d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Uploaded '%s' to knowledge base '%s'.\n", localFile, kbID)
	return nil
}

// getSlice returns a slice of maps from a response map, or nil if missing.
func getSlice(m map[string]interface{}, key string) []map[string]interface{} {
	v, ok := m[key]
	if !ok {
		return nil
	}
	s, ok := v.([]interface{})
	if !ok {
		return nil
	}
	result := make([]map[string]interface{}, 0, len(s))
	for _, item := range s {
		if mp, ok := item.(map[string]interface{}); ok {
			result = append(result, mp)
		}
	}
	return result
}
