package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var rpcCmd = &cobra.Command{
	Use:   "rpc",
	Short: "Manage and invoke RPC procedures",
	Long:  `List, invoke, and manage stored procedures (RPC).`,
}

var (
	rpcNamespace  string
	rpcParams     string
	rpcFile       string
	rpcAsync      bool
	rpcSyncDir    string
	rpcDryRun     bool
	rpcDeleteMiss bool
)

var rpcListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all procedures",
	Long: `List all RPC procedures.

Examples:
  fluxbase rpc list
  fluxbase rpc list --namespace default`,
	PreRunE: requireAuth,
	RunE:    runRPCList,
}

var rpcGetCmd = &cobra.Command{
	Use:   "get [namespace/name]",
	Short: "Get procedure details",
	Long: `Get details of a specific procedure.

Examples:
  fluxbase rpc get default/calculate_totals`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runRPCGet,
}

var rpcInvokeCmd = &cobra.Command{
	Use:   "invoke [namespace/name]",
	Short: "Invoke a procedure",
	Long: `Invoke an RPC procedure.

Examples:
  fluxbase rpc invoke default/calculate_totals
  fluxbase rpc invoke default/process_order --params '{"order_id": 123}'
  fluxbase rpc invoke default/batch_update --file ./params.json --async`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runRPCInvoke,
}

var rpcSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync procedures from a directory",
	Long: `Sync RPC procedures from SQL files.

Examples:
  fluxbase rpc sync --dir ./rpc
  fluxbase rpc sync --dir ./rpc --namespace production`,
	PreRunE: requireAuth,
	RunE:    runRPCSync,
}

func init() {
	// List flags
	rpcListCmd.Flags().StringVar(&rpcNamespace, "namespace", "", "Filter by namespace")

	// Invoke flags
	rpcInvokeCmd.Flags().StringVar(&rpcParams, "params", "", "JSON parameters")
	rpcInvokeCmd.Flags().StringVar(&rpcFile, "file", "", "File containing JSON parameters")
	rpcInvokeCmd.Flags().BoolVar(&rpcAsync, "async", false, "Execute asynchronously")

	// Sync flags
	rpcSyncCmd.Flags().StringVar(&rpcSyncDir, "dir", "./rpc", "Directory containing RPC SQL files")
	rpcSyncCmd.Flags().StringVar(&rpcNamespace, "namespace", "default", "Target namespace")
	rpcSyncCmd.Flags().BoolVar(&rpcDryRun, "dry-run", false, "Preview changes without applying")
	rpcSyncCmd.Flags().BoolVar(&rpcDeleteMiss, "delete-missing", false, "Delete procedures not in directory")

	rpcCmd.AddCommand(rpcListCmd)
	rpcCmd.AddCommand(rpcGetCmd)
	rpcCmd.AddCommand(rpcInvokeCmd)
	rpcCmd.AddCommand(rpcSyncCmd)
}

func runRPCList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	if rpcNamespace != "" {
		query.Set("namespace", rpcNamespace)
	}

	var procedures []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/rpc/procedures", query, &procedures); err != nil {
		return err
	}

	if len(procedures) == 0 {
		fmt.Println("No procedures found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"NAMESPACE", "NAME", "ENABLED", "PUBLIC", "SCHEDULE"},
			Rows:    make([][]string, len(procedures)),
		}

		for i, proc := range procedures {
			namespace := getStringValue(proc, "namespace")
			name := getStringValue(proc, "name")
			enabled := fmt.Sprintf("%v", proc["enabled"])
			public := fmt.Sprintf("%v", proc["is_public"])
			schedule := getStringValue(proc, "schedule")
			if schedule == "" {
				schedule = "-"
			}

			data.Rows[i] = []string{namespace, name, enabled, public, schedule}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(procedures); err != nil {
			return err
		}
	}

	return nil
}

func runRPCGet(cmd *cobra.Command, args []string) error {
	// Parse namespace/name
	namespace, name, err := parseNamespacedName(args[0])
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("/api/v1/admin/rpc/procedures/%s/%s", url.PathEscape(namespace), url.PathEscape(name))

	var proc map[string]interface{}
	if err := apiClient.DoGet(ctx, path, nil, &proc); err != nil {
		return err
	}

	formatter := GetFormatter()
	return formatter.Print(proc)
}

func runRPCInvoke(cmd *cobra.Command, args []string) error {
	// Parse namespace/name
	namespace, name, err := parseNamespacedName(args[0])
	if err != nil {
		return err
	}

	// Get parameters
	var params interface{}
	if rpcFile != "" {
		content, err := os.ReadFile(rpcFile) //nolint:gosec // CLI tool reads user-provided file path
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if err := json.Unmarshal(content, &params); err != nil {
			return fmt.Errorf("invalid JSON in file: %w", err)
		}
	} else if rpcParams != "" {
		if err := json.Unmarshal([]byte(rpcParams), &params); err != nil {
			return fmt.Errorf("invalid JSON params: %w", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	path := fmt.Sprintf("/rpc/%s/%s", url.PathEscape(namespace), url.PathEscape(name))

	body := map[string]interface{}{}
	if params != nil {
		body["params"] = params
	}
	if rpcAsync {
		body["async"] = true
	}

	var result interface{}
	if err := apiClient.DoPost(ctx, path, body, &result); err != nil {
		return err
	}

	formatter := GetFormatter()
	return formatter.Print(result)
}

func runRPCSync(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Auto-detect directory if not explicitly specified
	dir, err := detectResourceDir("rpc", rpcSyncDir, "./rpc")
	if err != nil {
		return err
	}
	rpcSyncDir = dir

	// Read SQL files from directory
	entries, err := os.ReadDir(rpcSyncDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var procedures []map[string]interface{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// Read file
		content, err := os.ReadFile(filepath.Join(rpcSyncDir, name)) //nolint:gosec // CLI tool reads user-provided file path
		if err != nil {
			fmt.Printf("Warning: failed to read %s: %v\n", name, err)
			continue
		}

		// Remove .sql extension for procedure name
		procName := strings.TrimSuffix(name, ".sql")

		procedures = append(procedures, map[string]interface{}{
			"name": procName,
			"code": string(content),
		})
	}

	if len(procedures) == 0 {
		fmt.Println("No SQL files found in directory.")
		return nil
	}

	if rpcDryRun {
		fmt.Println("Dry run - would sync the following procedures:")
		for _, proc := range procedures {
			fmt.Printf("  - %s\n", proc["name"])
		}
		return nil
	}

	// Build sync request body
	body := map[string]interface{}{
		"namespace":  rpcNamespace,
		"procedures": procedures,
		"options": map[string]interface{}{
			"delete_missing": rpcDeleteMiss,
			"dry_run":        false,
		},
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/rpc/sync", body, &result); err != nil {
		return err
	}

	// Parse and display result
	if summary, ok := result["summary"].(map[string]interface{}); ok {
		created := getIntValue(summary, "created")
		updated := getIntValue(summary, "updated")
		deleted := getIntValue(summary, "deleted")
		unchanged := getIntValue(summary, "unchanged")
		errors := getIntValue(summary, "errors")

		fmt.Printf("Synced %d procedures to namespace '%s':\n", len(procedures), rpcNamespace)
		fmt.Printf("  Created: %d\n", created)
		fmt.Printf("  Updated: %d\n", updated)
		fmt.Printf("  Deleted: %d\n", deleted)
		fmt.Printf("  Unchanged: %d\n", unchanged)
		if errors > 0 {
			fmt.Printf("  Errors: %d\n", errors)
		}
	} else {
		fmt.Printf("Synced %d procedures to namespace '%s'.\n", len(procedures), rpcNamespace)
	}

	// Print any errors
	if errs, ok := result["errors"].([]interface{}); ok && len(errs) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range errs {
			if errMap, ok := e.(map[string]interface{}); ok {
				proc := getStringValue(errMap, "procedure")
				errMsg := getStringValue(errMap, "error")
				fmt.Printf("  - %s: %s\n", proc, errMsg)
			}
		}
	}

	return nil
}

// parseNamespacedName parses "namespace/name" format
func parseNamespacedName(s string) (namespace, name string, err error) {
	parts := splitNamespace(s)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid format: expected 'namespace/name', got '%s'", s)
	}
	return parts[0], parts[1], nil
}

func splitNamespace(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
