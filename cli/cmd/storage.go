package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/output"
	"github.com/fluxbase-eu/fluxbase/cli/util"
)

var storageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Manage file storage",
	Long:  `Manage storage buckets and objects.`,
}

var storageBucketsCmd = &cobra.Command{
	Use:   "buckets",
	Short: "Manage storage buckets",
	Long:  `List, create, and delete storage buckets.`,
}

var storageObjectsCmd = &cobra.Command{
	Use:   "objects",
	Short: "Manage storage objects",
	Long:  `Upload, download, and manage files in storage.`,
}

var (
	bucketPublic      bool
	bucketMaxSize     int64
	objectPrefix      string
	objectContentType string
	urlExpires        int
)

// Bucket commands
var storageBucketsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all buckets",
	Long: `List all storage buckets.

Examples:
  fluxbase storage buckets list`,
	PreRunE: requireAuth,
	RunE:    runBucketsList,
}

var storageBucketsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a bucket",
	Long: `Create a new storage bucket.

Examples:
  fluxbase storage buckets create my-bucket
  fluxbase storage buckets create my-bucket --public`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runBucketsCreate,
}

var storageBucketsDeleteCmd = &cobra.Command{
	Use:     "delete [name]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a bucket",
	Long: `Delete a storage bucket.

Examples:
  fluxbase storage buckets delete my-bucket`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runBucketsDelete,
}

// Object commands
var storageObjectsListCmd = &cobra.Command{
	Use:   "list [bucket]",
	Short: "List objects in a bucket",
	Long: `List all objects in a storage bucket.

Examples:
  fluxbase storage objects list my-bucket
  fluxbase storage objects list my-bucket --prefix images/`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runObjectsList,
}

var storageObjectsUploadCmd = &cobra.Command{
	Use:   "upload [bucket] [path] [local-file]",
	Short: "Upload a file",
	Long: `Upload a file to storage.

Examples:
  fluxbase storage objects upload my-bucket images/photo.jpg ./photo.jpg
  fluxbase storage objects upload my-bucket report.pdf ./report.pdf --content-type application/pdf`,
	Args:    cobra.ExactArgs(3),
	PreRunE: requireAuth,
	RunE:    runObjectsUpload,
}

var storageObjectsDownloadCmd = &cobra.Command{
	Use:   "download [bucket] [path] [local-file]",
	Short: "Download a file",
	Long: `Download a file from storage.

Examples:
  fluxbase storage objects download my-bucket images/photo.jpg ./photo.jpg`,
	Args:    cobra.MinimumNArgs(2),
	PreRunE: requireAuth,
	RunE:    runObjectsDownload,
}

var storageObjectsDeleteCmd = &cobra.Command{
	Use:     "delete [bucket] [path]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete an object",
	Long: `Delete an object from storage.

Examples:
  fluxbase storage objects delete my-bucket images/photo.jpg`,
	Args:    cobra.ExactArgs(2),
	PreRunE: requireAuth,
	RunE:    runObjectsDelete,
}

var storageObjectsURLCmd = &cobra.Command{
	Use:   "url [bucket] [path]",
	Short: "Get a signed URL for an object",
	Long: `Generate a signed URL for accessing an object.

Examples:
  fluxbase storage objects url my-bucket images/photo.jpg
  fluxbase storage objects url my-bucket images/photo.jpg --expires 7200`,
	Args:    cobra.ExactArgs(2),
	PreRunE: requireAuth,
	RunE:    runObjectsURL,
}

func init() {
	// Bucket create flags
	storageBucketsCreateCmd.Flags().BoolVar(&bucketPublic, "public", false, "Make bucket publicly accessible")
	storageBucketsCreateCmd.Flags().Int64Var(&bucketMaxSize, "max-size", 0, "Maximum file size in bytes (0 = no limit)")

	// Object list flags
	storageObjectsListCmd.Flags().StringVar(&objectPrefix, "prefix", "", "Filter by path prefix")

	// Object upload flags
	storageObjectsUploadCmd.Flags().StringVar(&objectContentType, "content-type", "", "Content type (auto-detected if not specified)")

	// URL flags
	storageObjectsURLCmd.Flags().IntVar(&urlExpires, "expires", 3600, "URL expiration time in seconds")

	// Add bucket subcommands
	storageBucketsCmd.AddCommand(storageBucketsListCmd)
	storageBucketsCmd.AddCommand(storageBucketsCreateCmd)
	storageBucketsCmd.AddCommand(storageBucketsDeleteCmd)

	// Add object subcommands
	storageObjectsCmd.AddCommand(storageObjectsListCmd)
	storageObjectsCmd.AddCommand(storageObjectsUploadCmd)
	storageObjectsCmd.AddCommand(storageObjectsDownloadCmd)
	storageObjectsCmd.AddCommand(storageObjectsDeleteCmd)
	storageObjectsCmd.AddCommand(storageObjectsURLCmd)

	// Add to storage command
	storageCmd.AddCommand(storageBucketsCmd)
	storageCmd.AddCommand(storageObjectsCmd)
}

func runBucketsList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var buckets []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/storage/buckets", nil, &buckets); err != nil {
		return err
	}

	if len(buckets) == 0 {
		fmt.Println("No buckets found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"NAME", "PUBLIC", "CREATED"},
			Rows:    make([][]string, len(buckets)),
		}

		for i, bucket := range buckets {
			name := getStringValue(bucket, "name")
			public := fmt.Sprintf("%v", bucket["public"])
			created := getStringValue(bucket, "created_at")

			data.Rows[i] = []string{name, public, created}
		}

		formatter.PrintTable(data)
	} else {
		formatter.Print(buckets)
	}

	return nil
}

func runBucketsCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"name":   name,
		"public": bucketPublic,
	}

	if bucketMaxSize > 0 {
		body["max_file_size"] = bucketMaxSize
	}

	if err := apiClient.DoPost(ctx, "/api/v1/storage/buckets/"+url.PathEscape(name), body, nil); err != nil {
		return err
	}

	fmt.Printf("Bucket '%s' created successfully.\n", name)
	return nil
}

func runBucketsDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoDelete(ctx, "/api/v1/storage/buckets/"+url.PathEscape(name)); err != nil {
		return err
	}

	fmt.Printf("Bucket '%s' deleted.\n", name)
	return nil
}

func runObjectsList(cmd *cobra.Command, args []string) error {
	bucket := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	if objectPrefix != "" {
		query.Set("prefix", objectPrefix)
	}

	var objects []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/storage/"+url.PathEscape(bucket), query, &objects); err != nil {
		return err
	}

	if len(objects) == 0 {
		fmt.Println("No objects found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"NAME", "SIZE", "CONTENT-TYPE", "MODIFIED"},
			Rows:    make([][]string, len(objects)),
		}

		for i, obj := range objects {
			name := getStringValue(obj, "name")
			size := util.FormatBytes(int64(getIntValue(obj, "size")))
			contentType := getStringValue(obj, "content_type")
			modified := getStringValue(obj, "updated_at")

			data.Rows[i] = []string{name, size, contentType, modified}
		}

		formatter.PrintTable(data)
	} else {
		formatter.Print(objects)
	}

	return nil
}

func runObjectsUpload(cmd *cobra.Command, args []string) error {
	bucket := args[0]
	remotePath := args[1]
	localFile := args[2]

	// Read file
	data, err := os.ReadFile(localFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Determine content type
	contentType := objectContentType
	if contentType == "" {
		ext := filepath.Ext(localFile)
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Build URL
	uploadPath := fmt.Sprintf("/api/v1/storage/%s/%s", url.PathEscape(bucket), remotePath)

	// Create request manually for file upload
	u, err := url.Parse(apiClient.BaseURL)
	if err != nil {
		return err
	}
	u.Path = uploadPath

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", contentType)

	// Get credentials and add auth
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

	fmt.Printf("Uploaded '%s' to '%s/%s' (%s)\n", localFile, bucket, remotePath, util.FormatBytes(int64(len(data))))
	return nil
}

func runObjectsDownload(cmd *cobra.Command, args []string) error {
	bucket := args[0]
	remotePath := args[1]
	localFile := remotePath
	if len(args) > 2 {
		localFile = args[2]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	downloadPath := fmt.Sprintf("/api/v1/storage/%s/%s", url.PathEscape(bucket), remotePath)

	resp, err := apiClient.Get(ctx, downloadPath, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed: %s", string(body))
	}

	// Create output file
	out, err := os.Create(localFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = out.Close() }()

	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Downloaded '%s/%s' to '%s' (%s)\n", bucket, remotePath, localFile, util.FormatBytes(n))
	return nil
}

func runObjectsDelete(cmd *cobra.Command, args []string) error {
	bucket := args[0]
	remotePath := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deletePath := fmt.Sprintf("/api/v1/storage/%s/%s", url.PathEscape(bucket), remotePath)

	if err := apiClient.DoDelete(ctx, deletePath); err != nil {
		return err
	}

	fmt.Printf("Deleted '%s/%s'\n", bucket, remotePath)
	return nil
}

func runObjectsURL(cmd *cobra.Command, args []string) error {
	bucket := args[0]
	remotePath := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	signPath := fmt.Sprintf("/api/v1/storage/%s/sign/%s", url.PathEscape(bucket), remotePath)

	body := map[string]interface{}{
		"expires_in": urlExpires,
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, signPath, body, &result); err != nil {
		return err
	}

	signedURL := getStringValue(result, "url")
	fmt.Println(signedURL)

	return nil
}
