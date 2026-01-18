package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/output"
	"github.com/fluxbase-eu/fluxbase/cli/util"
)

var kbCmd = &cobra.Command{
	Use:     "kb",
	Aliases: []string{"knowledge-bases", "knowledge-base"},
	Short:   "Manage knowledge bases",
	Long:    `Create and manage knowledge bases for AI chatbots.`,
}

var (
	kbDescription     string
	kbEmbeddingModel  string
	kbChunkSize       int
	kbDocTitle        string
	kbDocMetadata     string
	kbSearchLimit     int
	kbSearchThreshold float64
)

var kbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all knowledge bases",
	Long: `List all knowledge bases.

Examples:
  fluxbase kb list
  fluxbase kb list -o json`,
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
	Short: "Create a new knowledge base",
	Long: `Create a new knowledge base.

Examples:
  fluxbase kb create docs --description "Product documentation"
  fluxbase kb create docs --embedding-model text-embedding-ada-002`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runKBCreate,
}

var kbUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a knowledge base",
	Long: `Update an existing knowledge base.

Examples:
  fluxbase kb update abc123 --description "Updated description"`,
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

var kbUploadCmd = &cobra.Command{
	Use:   "upload [id] [file]",
	Short: "Upload a document to a knowledge base",
	Long: `Upload a document to a knowledge base for indexing.

Supported formats: PDF, DOCX, TXT, MD, images (with OCR)

Examples:
  fluxbase kb upload abc123 ./docs/manual.pdf
  fluxbase kb upload abc123 ./docs/guide.md --title "User Guide"`,
	Args:    cobra.ExactArgs(2),
	PreRunE: requireAuth,
	RunE:    runKBUpload,
}

var kbDocumentsCmd = &cobra.Command{
	Use:   "documents [id]",
	Short: "List documents in a knowledge base",
	Long: `List all documents in a knowledge base.

Examples:
  fluxbase kb documents abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runKBDocuments,
}

var kbDocumentDeleteCmd = &cobra.Command{
	Use:   "delete [kb-id] [doc-id]",
	Short: "Delete a document from a knowledge base",
	Long: `Delete a specific document from a knowledge base.

Examples:
  fluxbase kb documents delete abc123 doc456`,
	Args:    cobra.ExactArgs(2),
	PreRunE: requireAuth,
	RunE:    runKBDocumentDelete,
}

var kbSearchCmd = &cobra.Command{
	Use:   "search [id] [query]",
	Short: "Search a knowledge base",
	Long: `Search a knowledge base using semantic similarity.

Examples:
  fluxbase kb search abc123 "how to reset password"
  fluxbase kb search abc123 "pricing plans" --limit 5`,
	Args:    cobra.ExactArgs(2),
	PreRunE: requireAuth,
	RunE:    runKBSearch,
}

func init() {
	// Create flags
	kbCreateCmd.Flags().StringVar(&kbDescription, "description", "", "Knowledge base description")
	kbCreateCmd.Flags().StringVar(&kbEmbeddingModel, "embedding-model", "", "Embedding model to use")
	kbCreateCmd.Flags().IntVar(&kbChunkSize, "chunk-size", 512, "Document chunk size")

	// Update flags
	kbUpdateCmd.Flags().StringVar(&kbDescription, "description", "", "Knowledge base description")

	// Upload flags
	kbUploadCmd.Flags().StringVar(&kbDocTitle, "title", "", "Document title")
	kbUploadCmd.Flags().StringVar(&kbDocMetadata, "metadata", "", "Document metadata (JSON)")

	// Search flags
	kbSearchCmd.Flags().IntVar(&kbSearchLimit, "limit", 10, "Maximum results to return")
	kbSearchCmd.Flags().Float64Var(&kbSearchThreshold, "threshold", 0.7, "Similarity threshold (0.0-1.0)")

	// Add document delete as subcommand
	kbDocumentsCmd.AddCommand(kbDocumentDeleteCmd)

	kbCmd.AddCommand(kbListCmd)
	kbCmd.AddCommand(kbGetCmd)
	kbCmd.AddCommand(kbCreateCmd)
	kbCmd.AddCommand(kbUpdateCmd)
	kbCmd.AddCommand(kbDeleteCmd)
	kbCmd.AddCommand(kbUploadCmd)
	kbCmd.AddCommand(kbDocumentsCmd)
	kbCmd.AddCommand(kbSearchCmd)
}

func runKBList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var response struct {
		KnowledgeBases []map[string]interface{} `json:"knowledge_bases"`
		Count          int                      `json:"count"`
	}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/ai/knowledge-bases", nil, &response); err != nil {
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
			Headers: []string{"ID", "NAME", "DOCUMENTS", "CREATED"},
			Rows:    make([][]string, len(kbs)),
		}

		for i, kb := range kbs {
			id := getStringValue(kb, "id")
			name := getStringValue(kb, "name")
			docs := fmt.Sprintf("%d", getIntValue(kb, "document_count"))
			created := getStringValue(kb, "created_at")

			data.Rows[i] = []string{id, name, docs, created}
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
		"name":       name,
		"chunk_size": kbChunkSize,
	}

	if kbDescription != "" {
		body["description"] = kbDescription
	}
	if kbEmbeddingModel != "" {
		body["embedding_model"] = kbEmbeddingModel
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

	if kbDescription != "" {
		body["description"] = kbDescription
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

func runKBUpload(cmd *cobra.Command, args []string) error {
	kbID := args[0]
	filePath := args[1]

	// Read file
	file, err := os.Open(filePath) //nolint:gosec // CLI tool reads user-provided file path
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Add title if specified
	if kbDocTitle != "" {
		if err := writer.WriteField("title", kbDocTitle); err != nil {
			return err
		}
	}

	// Add metadata if specified
	if kbDocMetadata != "" {
		if err := writer.WriteField("metadata", kbDocMetadata); err != nil {
			return err
		}
	}

	if err := writer.Close(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Build request - use /upload endpoint for multipart uploads
	uploadURL := apiClient.BaseURL + "/api/v1/admin/ai/knowledge-bases/" + url.PathEscape(kbID) + "/documents/upload"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Add auth
	creds, err := apiClient.CredentialManager.GetCredentials(apiClient.Profile.Name)
	if err != nil {
		return err
	}
	if creds != nil && creds.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
	} else if creds != nil && creds.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+creds.APIKey)
	}

	resp, err := apiClient.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Non-JSON response is OK
		fmt.Printf("Uploaded '%s' to knowledge base '%s' (%s)\n", filepath.Base(filePath), kbID, util.FormatBytes(fileInfo.Size()))
		return nil
	}

	docID := getStringValue(result, "id")
	fmt.Printf("Uploaded '%s' to knowledge base '%s' (Document ID: %s, %s)\n", filepath.Base(filePath), kbID, docID, util.FormatBytes(fileInfo.Size()))
	return nil
}

func runKBDocuments(cmd *cobra.Command, args []string) error {
	kbID := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// API returns wrapped response: {"documents": [...], "count": N}
	var response struct {
		Documents []map[string]interface{} `json:"documents"`
		Count     int                      `json:"count"`
	}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/ai/knowledge-bases/"+url.PathEscape(kbID)+"/documents", nil, &response); err != nil {
		return err
	}
	docs := response.Documents

	if len(docs) == 0 {
		fmt.Println("No documents found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"ID", "TITLE", "TYPE", "CHUNKS", "STATUS"},
			Rows:    make([][]string, len(docs)),
		}

		for i, doc := range docs {
			id := getStringValue(doc, "id")
			title := getStringValue(doc, "title")
			if title == "" {
				title = getStringValue(doc, "filename")
			}
			docType := getStringValue(doc, "file_type")
			if docType == "" {
				docType = getStringValue(doc, "content_type")
			}
			chunks := fmt.Sprintf("%d", getIntValue(doc, "chunk_count"))
			status := getStringValue(doc, "status")

			data.Rows[i] = []string{id, title, docType, chunks, status}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(docs); err != nil {
			return err
		}
	}

	return nil
}

func runKBDocumentDelete(cmd *cobra.Command, args []string) error {
	kbID := args[0]
	docID := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deletePath := fmt.Sprintf("/api/v1/admin/ai/knowledge-bases/%s/documents/%s", url.PathEscape(kbID), url.PathEscape(docID))

	if err := apiClient.DoDelete(ctx, deletePath); err != nil {
		return err
	}

	fmt.Printf("Document '%s' deleted from knowledge base '%s'.\n", docID, kbID)
	return nil
}

func runKBSearch(cmd *cobra.Command, args []string) error {
	kbID := args[0]
	query := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"query":     query,
		"limit":     kbSearchLimit,
		"threshold": kbSearchThreshold,
	}

	var results []map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/ai/knowledge-bases/"+url.PathEscape(kbID)+"/search", body, &results); err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		for i, result := range results {
			score := result["score"]
			content := getStringValue(result, "content")
			docTitle := getStringValue(result, "document_title")

			fmt.Printf("%d. [%.2f] %s\n", i+1, score, docTitle)
			// Truncate content for display
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			fmt.Printf("   %s\n\n", content)
		}
	} else {
		if err := formatter.Print(results); err != nil {
			return err
		}
	}

	return nil
}
