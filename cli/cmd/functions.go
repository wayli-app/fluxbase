package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/annotations"
	"github.com/fluxbase-eu/fluxbase/cli/bundler"
	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var functionsCmd = &cobra.Command{
	Use:     "functions",
	Aliases: []string{"fn", "function"},
	Short:   "Manage edge functions",
	Long:    `Create, deploy, and manage edge functions.`,
}

var (
	fnNamespace   string
	fnCodeFile    string
	fnDescription string
	fnTimeout     int
	fnMemory      int
	fnInvokeData  string
	fnInvokeFile  string
	fnAsync       bool
	fnTail        int
	fnFollow      bool
	fnSyncDir     string
	fnDryRun      bool
)

var functionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all functions",
	Long: `List all edge functions.

Examples:
  fluxbase functions list
  fluxbase functions list --namespace production
  fluxbase functions list -o json`,
	PreRunE: requireAuth,
	RunE:    runFunctionsList,
}

var functionsGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get function details",
	Long: `Get details of a specific function.

Examples:
  fluxbase functions get my-function
  fluxbase functions get my-function -o json`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runFunctionsGet,
}

var functionsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new function",
	Long: `Create a new edge function.

Examples:
  fluxbase functions create my-function --code ./function.ts
  fluxbase functions create my-function --code ./function.ts --description "My function"`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runFunctionsCreate,
}

var functionsUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update an existing function",
	Long: `Update an existing edge function.

Examples:
  fluxbase functions update my-function --code ./function.ts
  fluxbase functions update my-function --timeout 60`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runFunctionsUpdate,
}

var functionsDeleteCmd = &cobra.Command{
	Use:     "delete [name]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a function",
	Long: `Delete an edge function.

Examples:
  fluxbase functions delete my-function`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runFunctionsDelete,
}

var functionsInvokeCmd = &cobra.Command{
	Use:   "invoke [name]",
	Short: "Invoke a function",
	Long: `Invoke an edge function.

Examples:
  fluxbase functions invoke my-function
  fluxbase functions invoke my-function --data '{"key": "value"}'
  fluxbase functions invoke my-function --file ./payload.json`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runFunctionsInvoke,
}

var functionsLogsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "View function execution logs",
	Long: `View logs for function executions.

Examples:
  fluxbase functions logs my-function
  fluxbase functions logs my-function --tail 50`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runFunctionsLogs,
}

var functionsSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync functions from a directory",
	Long: `Sync edge functions from a directory to the server.

Examples:
  fluxbase functions sync --dir ./functions
  fluxbase functions sync --dir ./functions --namespace production
  fluxbase functions sync --dir ./functions --dry-run`,
	PreRunE: requireAuth,
	RunE:    runFunctionsSync,
}

func init() {
	// List flags
	functionsListCmd.Flags().StringVar(&fnNamespace, "namespace", "", "Filter by namespace")

	// Create flags
	functionsCreateCmd.Flags().StringVar(&fnCodeFile, "code", "", "Path to function code file (required)")
	functionsCreateCmd.Flags().StringVar(&fnDescription, "description", "", "Function description")
	functionsCreateCmd.Flags().IntVar(&fnTimeout, "timeout", 30, "Execution timeout in seconds")
	functionsCreateCmd.Flags().IntVar(&fnMemory, "memory", 128, "Memory limit in MB")
	_ = functionsCreateCmd.MarkFlagRequired("code")

	// Update flags
	functionsUpdateCmd.Flags().StringVar(&fnCodeFile, "code", "", "Path to function code file")
	functionsUpdateCmd.Flags().StringVar(&fnDescription, "description", "", "Function description")
	functionsUpdateCmd.Flags().IntVar(&fnTimeout, "timeout", 0, "Execution timeout in seconds")
	functionsUpdateCmd.Flags().IntVar(&fnMemory, "memory", 0, "Memory limit in MB")

	// Invoke flags
	functionsInvokeCmd.Flags().StringVar(&fnInvokeData, "data", "", "JSON data to send")
	functionsInvokeCmd.Flags().StringVar(&fnInvokeFile, "file", "", "File containing JSON data")
	functionsInvokeCmd.Flags().BoolVar(&fnAsync, "async", false, "Invoke asynchronously")

	// Logs flags
	functionsLogsCmd.Flags().IntVar(&fnTail, "tail", 20, "Number of log lines to show")
	functionsLogsCmd.Flags().BoolVar(&fnFollow, "follow", false, "Follow log output")

	// Sync flags
	functionsSyncCmd.Flags().StringVar(&fnSyncDir, "dir", "./functions", "Directory containing functions")
	functionsSyncCmd.Flags().StringVar(&fnNamespace, "namespace", "default", "Target namespace")
	functionsSyncCmd.Flags().BoolVar(&fnDryRun, "dry-run", false, "Preview changes without applying")

	functionsCmd.AddCommand(functionsListCmd)
	functionsCmd.AddCommand(functionsGetCmd)
	functionsCmd.AddCommand(functionsCreateCmd)
	functionsCmd.AddCommand(functionsUpdateCmd)
	functionsCmd.AddCommand(functionsDeleteCmd)
	functionsCmd.AddCommand(functionsInvokeCmd)
	functionsCmd.AddCommand(functionsLogsCmd)
	functionsCmd.AddCommand(functionsSyncCmd)
}

func runFunctionsList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	if fnNamespace != "" {
		query.Set("namespace", fnNamespace)
	}

	var functions []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/functions/", query, &functions); err != nil {
		return err
	}

	if len(functions) == 0 {
		fmt.Println("No functions found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"NAME", "NAMESPACE", "ENABLED", "TIMEOUT", "MEMORY"},
			Rows:    make([][]string, len(functions)),
		}

		for i, fn := range functions {
			name := getStringValue(fn, "name")
			namespace := getStringValue(fn, "namespace")
			enabled := fmt.Sprintf("%v", fn["enabled"])
			timeout := fmt.Sprintf("%vs", getIntValue(fn, "timeout_seconds"))
			memory := fmt.Sprintf("%vMB", getIntValue(fn, "memory_limit_mb"))

			data.Rows[i] = []string{name, namespace, enabled, timeout, memory}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(functions); err != nil {
			return err
		}
	}

	return nil
}

func runFunctionsGet(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var fn map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/functions/"+url.PathEscape(name), nil, &fn); err != nil {
		return err
	}

	formatter := GetFormatter()
	return formatter.Print(fn)
}

func runFunctionsCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Read code file
	code, err := os.ReadFile(fnCodeFile) //nolint:gosec // CLI tool reads user-provided file path
	if err != nil {
		return fmt.Errorf("failed to read code file: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"name":            name,
		"code":            string(code),
		"timeout_seconds": fnTimeout,
		"memory_limit_mb": fnMemory,
	}

	if fnDescription != "" {
		body["description"] = fnDescription
	}

	var fn map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/functions/", body, &fn); err != nil {
		return err
	}

	fmt.Printf("Function '%s' created successfully.\n", name)
	return nil
}

func runFunctionsUpdate(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	body := make(map[string]interface{})

	if fnCodeFile != "" {
		code, err := os.ReadFile(fnCodeFile) //nolint:gosec // CLI tool reads user-provided file path
		if err != nil {
			return fmt.Errorf("failed to read code file: %w", err)
		}
		body["code"] = string(code)
	}

	if fnDescription != "" {
		body["description"] = fnDescription
	}

	if fnTimeout > 0 {
		body["timeout_seconds"] = fnTimeout
	}

	if fnMemory > 0 {
		body["memory_limit_mb"] = fnMemory
	}

	if len(body) == 0 {
		return fmt.Errorf("no updates specified")
	}

	if err := apiClient.DoPut(ctx, "/api/v1/functions/"+url.PathEscape(name), body, nil); err != nil {
		return err
	}

	fmt.Printf("Function '%s' updated successfully.\n", name)
	return nil
}

func runFunctionsDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoDelete(ctx, "/api/v1/functions/"+url.PathEscape(name)); err != nil {
		return err
	}

	fmt.Printf("Function '%s' deleted successfully.\n", name)
	return nil
}

func runFunctionsInvoke(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var payload interface{}

	if fnInvokeFile != "" {
		data, err := os.ReadFile(fnInvokeFile) //nolint:gosec // CLI tool reads user-provided file path
		if err != nil {
			return fmt.Errorf("failed to read payload file: %w", err)
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("invalid JSON in payload file: %w", err)
		}
	} else if fnInvokeData != "" {
		if err := json.Unmarshal([]byte(fnInvokeData), &payload); err != nil {
			return fmt.Errorf("invalid JSON data: %w", err)
		}
	}

	resp, err := apiClient.Post(ctx, "/api/v1/functions/"+url.PathEscape(name)+"/invoke", payload)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("function invocation failed: %s", string(body))
	}

	// Try to pretty-print if JSON
	var jsonResult interface{}
	if err := json.Unmarshal(body, &jsonResult); err == nil {
		formatter := GetFormatter()
		return formatter.Print(jsonResult)
	}

	// Otherwise print raw
	fmt.Println(string(body))
	return nil
}

func runFunctionsLogs(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	query.Set("limit", fmt.Sprintf("%d", fnTail))

	var executions []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/functions/"+url.PathEscape(name)+"/executions", query, &executions); err != nil {
		return err
	}

	if len(executions) == 0 {
		fmt.Println("No executions found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"ID", "STATUS", "DURATION", "STARTED"},
			Rows:    make([][]string, len(executions)),
		}

		for i, exec := range executions {
			id := getStringValue(exec, "id")
			status := getStringValue(exec, "status")
			duration := fmt.Sprintf("%vms", getIntValue(exec, "duration_ms"))
			started := getStringValue(exec, "started_at")

			data.Rows[i] = []string{id, status, duration, started}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(executions); err != nil {
			return err
		}
	}

	return nil
}

func runFunctionsSync(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Increase HTTP client timeout for sync operations (large bundled payloads)
	originalTimeout := apiClient.HTTPClient.Timeout
	apiClient.HTTPClient.Timeout = 5 * time.Minute
	defer func() { apiClient.HTTPClient.Timeout = originalTimeout }()

	// Auto-detect directory if not explicitly specified
	dir, err := detectResourceDir("functions", fnSyncDir, "./functions")
	if err != nil {
		return err
	}
	fnSyncDir = dir

	// Check for _shared directory and sync shared modules first
	sharedDir := filepath.Join(fnSyncDir, "_shared")
	if info, err := os.Stat(sharedDir); err == nil && info.IsDir() {
		sharedEntries, err := os.ReadDir(sharedDir)
		if err == nil {
			sharedCount := 0
			for _, entry := range sharedEntries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				if !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".js") {
					continue
				}

				content, err := os.ReadFile(filepath.Join(sharedDir, name)) //nolint:gosec // CLI tool reads user-provided file path
				if err != nil {
					fmt.Printf("Warning: failed to read shared module %s: %v\n", name, err)
					continue
				}

				modulePath := "_shared/" + name

				// Create or update the shared module via API
				moduleBody := map[string]interface{}{
					"module_path": modulePath,
					"content":     string(content),
				}

				// Try PUT first (update), fall back to POST (create)
				err = apiClient.DoPut(ctx, "/api/v1/functions/shared/"+url.PathEscape(modulePath), moduleBody, nil)
				if err != nil {
					err = apiClient.DoPost(ctx, "/api/v1/functions/shared/", moduleBody, nil)
					if err != nil {
						fmt.Printf("Warning: failed to sync shared module %s: %v\n", modulePath, err)
						continue
					}
				}
				sharedCount++
			}
			if sharedCount > 0 {
				fmt.Printf("Synced %d shared modules.\n", sharedCount)
			}
		}
	}

	// Read functions from directory
	entries, err := os.ReadDir(fnSyncDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var functions []map[string]interface{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".js") {
			continue
		}

		// Read file
		content, err := os.ReadFile(filepath.Join(fnSyncDir, name)) //nolint:gosec // CLI tool reads user-provided file path
		if err != nil {
			fmt.Printf("Warning: failed to read %s: %v\n", name, err)
			continue
		}

		// Remove extension for function name
		fnName := strings.TrimSuffix(strings.TrimSuffix(name, ".ts"), ".js")

		// Parse @fluxbase: annotations BEFORE bundling (esbuild strips comments)
		fn := map[string]interface{}{
			"name": fnName,
			"code": string(content),
		}
		config := annotations.ParseFunctionAnnotations(string(content))
		annotations.ApplyFunctionConfig(fn, config)

		functions = append(functions, fn)
	}

	if len(functions) == 0 {
		fmt.Println("No functions found in directory.")
		return nil
	}

	if fnDryRun {
		fmt.Println("Dry run - would sync the following functions:")
		for _, fn := range functions {
			fmt.Printf("  - %s\n", fn["name"])
		}
		return nil
	}

	// Read shared modules for bundling
	sharedModulesMap := make(map[string]string)
	if info, err := os.Stat(sharedDir); err == nil && info.IsDir() {
		sharedEntries, err := os.ReadDir(sharedDir)
		if err == nil {
			for _, entry := range sharedEntries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				if !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".js") {
					continue
				}
				content, err := os.ReadFile(filepath.Join(sharedDir, name)) //nolint:gosec // CLI tool reads user-provided file path
				if err == nil {
					sharedModulesMap["_shared/"+name] = string(content)
				}
			}
		}
	}

	// Check if any function needs bundling
	needsBundling := false
	for _, fn := range functions {
		code := fn["code"].(string)
		// Simple check for imports
		if strings.Contains(code, "import ") {
			needsBundling = true
			break
		}
	}

	// Bundle functions that have imports (if Deno is available)
	if needsBundling {
		b, err := bundler.NewBundler(fnSyncDir)
		if err != nil {
			// Deno not available - send unbundled code, server will handle it
			fmt.Println("Note: Deno not available for local bundling. Server will bundle functions.")
		} else {
			for i, fn := range functions {
				code := fn["code"].(string)
				fnName := fn["name"].(string)

				if !b.NeedsBundle(code) {
					continue // No imports, skip bundling
				}

				// Validate imports first
				if err := b.ValidateImports(code); err != nil {
					return fmt.Errorf("function %s: %w", fnName, err)
				}

				fmt.Printf("Bundling %s...", fnName)

				result, err := b.Bundle(ctx, code, sharedModulesMap)
				if err != nil {
					fmt.Println() // Complete the line
					return fmt.Errorf("failed to bundle %s: %w", fnName, err)
				}

				// Print size info
				originalSize := len(code)
				bundledSize := len(result.BundledCode)
				fmt.Printf(" %s → %s\n", formatBytes(originalSize), formatBytes(bundledSize))

				// Replace code with bundled code
				functions[i]["code"] = result.BundledCode
				functions[i]["is_bundled"] = true
			}
		}
	}

	// Call sync API
	body := map[string]interface{}{
		"namespace": fnNamespace,
		"functions": functions,
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/functions/sync", body, &result); err != nil {
		return err
	}

	// Parse the nested summary response
	summary, ok := result["summary"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format from server")
	}

	created := getIntValue(summary, "created")
	updated := getIntValue(summary, "updated")
	deleted := getIntValue(summary, "deleted")
	errors := getIntValue(summary, "errors")

	fmt.Printf("Synced functions to namespace '%s': %d created, %d updated, %d deleted.\n",
		fnNamespace, created, updated, deleted)
	if errors > 0 {
		fmt.Printf("Warning: %d errors occurred during sync.\n", errors)
		// Print error details if available
		if errorList, ok := result["errors"].([]interface{}); ok {
			for _, e := range errorList {
				if errMap, ok := e.(map[string]interface{}); ok {
					fmt.Printf("  - %s: %v\n", errMap["function"], errMap["error"])
				}
			}
		}
	}

	return nil
}

// Helper functions

func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getIntValue(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case int64:
			return int(n)
		}
	}
	return 0
}

func formatBytes(bytes int) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)
	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
