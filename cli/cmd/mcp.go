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

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage custom MCP tools",
	Long:  `Create, deploy, and manage custom MCP (Model Context Protocol) tools.`,
}

var (
	mcpNamespace   string
	mcpCodeFile    string
	mcpDescription string
	mcpTimeout     int
	mcpMemory      int
	mcpAllowNet    bool
	mcpAllowEnv    bool
	mcpAllowRead   bool
	mcpAllowWrite  bool
	mcpSyncDir     string
	mcpDryRun      bool
	mcpTestArgs    string
)

// Tools Commands

var mcpToolsCmd = &cobra.Command{
	Use:     "tools",
	Aliases: []string{"tool"},
	Short:   "Manage custom MCP tools",
	Long:    `Create, deploy, and manage custom MCP tools.`,
}

var mcpToolsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all custom MCP tools",
	Long: `List all custom MCP tools.

Examples:
  fluxbase mcp tools list
  fluxbase mcp tools list --namespace production
  fluxbase mcp tools list -o json`,
	PreRunE: requireAuth,
	RunE:    runMCPToolsList,
}

var mcpToolsGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get custom MCP tool details",
	Long: `Get details of a specific custom MCP tool.

Examples:
  fluxbase mcp tools get weather_forecast
  fluxbase mcp tools get weather_forecast -o json`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runMCPToolsGet,
}

var mcpToolsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new custom MCP tool",
	Long: `Create a new custom MCP tool.

Examples:
  fluxbase mcp tools create weather_forecast --code ./weather.ts
  fluxbase mcp tools create weather_forecast --code ./weather.ts --description "Get weather forecast"`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runMCPToolsCreate,
}

var mcpToolsUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update an existing custom MCP tool",
	Long: `Update an existing custom MCP tool.

Examples:
  fluxbase mcp tools update weather_forecast --code ./weather.ts
  fluxbase mcp tools update weather_forecast --timeout 60`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runMCPToolsUpdate,
}

var mcpToolsDeleteCmd = &cobra.Command{
	Use:     "delete [name]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a custom MCP tool",
	Long: `Delete a custom MCP tool.

Examples:
  fluxbase mcp tools delete weather_forecast`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runMCPToolsDelete,
}

var mcpToolsSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync custom MCP tools from a directory",
	Long: `Sync custom MCP tools from a directory to the server.

Each .ts file in the directory will be synced as a custom tool.
Tool name defaults to filename. All annotations are optional.

Directory auto-detection (when --dir is not specified):
  1. ./fluxbase/mcp-tools/
  2. ./mcp-tools/

Annotations (all optional):
  // @fluxbase:name my_tool         (defaults to filename)
  // @fluxbase:namespace production (defaults to CLI --namespace flag or "default")
  // @fluxbase:description ...      (helpful for AI)
  // @fluxbase:scopes read:tables   (additional scopes beyond execute:custom)
  // @fluxbase:timeout 30           (defaults to 30s)
  // @fluxbase:memory 128           (defaults to 128MB)
  // @fluxbase:allow-net            (opt-in for network access)
  // @fluxbase:allow-env            (opt-in for secrets/env access)

Examples:
  fluxbase mcp tools sync                                # Auto-detect directory
  fluxbase mcp tools sync --dir ./mcp-tools
  fluxbase mcp tools sync --dir ./mcp-tools --namespace production
  fluxbase mcp tools sync --dry-run`,
	PreRunE: requireAuth,
	RunE:    runMCPToolsSync,
}

var mcpToolsTestCmd = &cobra.Command{
	Use:   "test [name]",
	Short: "Test a custom MCP tool",
	Long: `Test a custom MCP tool by invoking it with sample arguments.

Examples:
  fluxbase mcp tools test weather_forecast --args '{"location": "New York"}'`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runMCPToolsTest,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpToolsCmd)

	// Tools subcommands
	mcpToolsCmd.AddCommand(mcpToolsListCmd)
	mcpToolsCmd.AddCommand(mcpToolsGetCmd)
	mcpToolsCmd.AddCommand(mcpToolsCreateCmd)
	mcpToolsCmd.AddCommand(mcpToolsUpdateCmd)
	mcpToolsCmd.AddCommand(mcpToolsDeleteCmd)
	mcpToolsCmd.AddCommand(mcpToolsSyncCmd)
	mcpToolsCmd.AddCommand(mcpToolsTestCmd)

	// Tools list flags
	mcpToolsListCmd.Flags().StringVar(&mcpNamespace, "namespace", "", "Filter by namespace")

	// Tools create flags
	mcpToolsCreateCmd.Flags().StringVar(&mcpCodeFile, "code", "", "Path to TypeScript code file (required)")
	mcpToolsCreateCmd.Flags().StringVar(&mcpNamespace, "namespace", "default", "Namespace")
	mcpToolsCreateCmd.Flags().StringVar(&mcpDescription, "description", "", "Tool description")
	mcpToolsCreateCmd.Flags().IntVar(&mcpTimeout, "timeout", 30, "Execution timeout in seconds")
	mcpToolsCreateCmd.Flags().IntVar(&mcpMemory, "memory", 128, "Memory limit in MB")
	mcpToolsCreateCmd.Flags().BoolVar(&mcpAllowNet, "allow-net", true, "Allow network access")
	mcpToolsCreateCmd.Flags().BoolVar(&mcpAllowEnv, "allow-env", false, "Allow environment variable access")
	mcpToolsCreateCmd.Flags().BoolVar(&mcpAllowRead, "allow-read", false, "Allow file read access")
	mcpToolsCreateCmd.Flags().BoolVar(&mcpAllowWrite, "allow-write", false, "Allow file write access")
	_ = mcpToolsCreateCmd.MarkFlagRequired("code")

	// Tools update flags
	mcpToolsUpdateCmd.Flags().StringVar(&mcpCodeFile, "code", "", "Path to TypeScript code file")
	mcpToolsUpdateCmd.Flags().StringVar(&mcpNamespace, "namespace", "default", "Namespace")
	mcpToolsUpdateCmd.Flags().StringVar(&mcpDescription, "description", "", "Tool description")
	mcpToolsUpdateCmd.Flags().IntVar(&mcpTimeout, "timeout", 0, "Execution timeout in seconds")
	mcpToolsUpdateCmd.Flags().IntVar(&mcpMemory, "memory", 0, "Memory limit in MB")

	// Tools sync flags
	mcpToolsSyncCmd.Flags().StringVar(&mcpSyncDir, "dir", "", "Directory containing tool files (auto-detects ./fluxbase/mcp-tools/ or ./mcp-tools/)")
	mcpToolsSyncCmd.Flags().StringVar(&mcpNamespace, "namespace", "default", "Namespace")
	mcpToolsSyncCmd.Flags().BoolVar(&mcpDryRun, "dry-run", false, "Show what would be synced without making changes")

	// Tools test flags
	mcpToolsTestCmd.Flags().StringVar(&mcpTestArgs, "args", "{}", "JSON arguments to pass to the tool")
	mcpToolsTestCmd.Flags().StringVar(&mcpNamespace, "namespace", "default", "Namespace")
}

// Tool command implementations

func runMCPToolsList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	if mcpNamespace != "" {
		query.Set("namespace", mcpNamespace)
	}

	var result struct {
		Tools []map[string]interface{} `json:"tools"`
		Count int                      `json:"count"`
	}
	if err := apiClient.DoGet(ctx, "/api/v1/mcp/tools", query, &result); err != nil {
		return err
	}

	if len(result.Tools) == 0 {
		fmt.Println("No custom MCP tools found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"NAME", "NAMESPACE", "ENABLED", "VERSION", "DESCRIPTION"},
			Rows:    make([][]string, len(result.Tools)),
		}

		for i, tool := range result.Tools {
			enabled := "true"
			if e, ok := tool["enabled"].(bool); ok && !e {
				enabled = "false"
			}
			version := "1"
			if v, ok := tool["version"].(float64); ok {
				version = fmt.Sprintf("%d", int(v))
			}
			desc := ""
			if d, ok := tool["description"].(string); ok {
				desc = truncate(d, 40)
			}
			data.Rows[i] = []string{
				getStringValue(tool, "name"),
				getStringValue(tool, "namespace"),
				enabled,
				version,
				desc,
			}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(result); err != nil {
			return err
		}
	}

	return nil
}

func runMCPToolsGet(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	query.Set("namespace", mcpNamespace)

	var result struct {
		Tools []map[string]interface{} `json:"tools"`
	}
	if err := apiClient.DoGet(ctx, "/api/v1/mcp/tools", query, &result); err != nil {
		return err
	}

	// Find tool by name
	var tool map[string]interface{}
	for _, t := range result.Tools {
		if getStringValue(t, "name") == args[0] {
			tool = t
			break
		}
	}

	if tool == nil {
		return fmt.Errorf("tool not found: %s", args[0])
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		// Pretty print tool details
		fmt.Printf("Name:        %s\n", tool["name"])
		fmt.Printf("Namespace:   %s\n", tool["namespace"])
		fmt.Printf("Enabled:     %v\n", tool["enabled"])
		fmt.Printf("Version:     %v\n", tool["version"])
		if desc, ok := tool["description"].(string); ok && desc != "" {
			fmt.Printf("Description: %s\n", desc)
		}
		fmt.Printf("Timeout:     %vs\n", tool["timeout_seconds"])
		fmt.Printf("Memory:      %vMB\n", tool["memory_limit_mb"])
		fmt.Printf("Allow Net:   %v\n", tool["allow_net"])
		fmt.Printf("Allow Env:   %v\n", tool["allow_env"])
		if code, ok := tool["code"].(string); ok {
			fmt.Printf("\nCode:\n%s\n", code)
		}
	} else {
		return formatter.Print(tool)
	}

	return nil
}

func runMCPToolsCreate(cmd *cobra.Command, args []string) error {
	code, err := os.ReadFile(mcpCodeFile) //nolint:gosec // CLI tool reads user-provided file path
	if err != nil {
		return fmt.Errorf("failed to read code file: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	payload := map[string]interface{}{
		"name":            args[0],
		"namespace":       mcpNamespace,
		"code":            string(code),
		"description":     mcpDescription,
		"timeout_seconds": mcpTimeout,
		"memory_limit_mb": mcpMemory,
		"allow_net":       mcpAllowNet,
		"allow_env":       mcpAllowEnv,
		"allow_read":      mcpAllowRead,
		"allow_write":     mcpAllowWrite,
	}

	var tool map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/mcp/tools", payload, &tool); err != nil {
		return err
	}

	formatter := GetFormatter()
	if formatter.Format != output.FormatTable {
		return formatter.Print(tool)
	}

	fmt.Printf("Created custom MCP tool: %s\n", args[0])
	return nil
}

func runMCPToolsUpdate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First, find the tool by name to get its ID
	query := url.Values{}
	query.Set("namespace", mcpNamespace)

	var listResult struct {
		Tools []map[string]interface{} `json:"tools"`
	}
	if err := apiClient.DoGet(ctx, "/api/v1/mcp/tools", query, &listResult); err != nil {
		return err
	}

	var toolID string
	for _, t := range listResult.Tools {
		if getStringValue(t, "name") == args[0] {
			toolID = getStringValue(t, "id")
			break
		}
	}

	if toolID == "" {
		return fmt.Errorf("tool not found: %s", args[0])
	}

	payload := make(map[string]interface{})
	if mcpCodeFile != "" {
		code, err := os.ReadFile(mcpCodeFile) //nolint:gosec // CLI tool reads user-provided file path
		if err != nil {
			return fmt.Errorf("failed to read code file: %w", err)
		}
		payload["code"] = string(code)
	}
	if mcpDescription != "" {
		payload["description"] = mcpDescription
	}
	if mcpTimeout > 0 {
		payload["timeout_seconds"] = mcpTimeout
	}
	if mcpMemory > 0 {
		payload["memory_limit_mb"] = mcpMemory
	}

	if len(payload) == 0 {
		return fmt.Errorf("no updates specified")
	}

	if err := apiClient.DoPut(ctx, "/api/v1/mcp/tools/"+toolID, payload, nil); err != nil {
		return err
	}

	fmt.Printf("Updated custom MCP tool: %s\n", args[0])
	return nil
}

func runMCPToolsDelete(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First, find the tool by name to get its ID
	query := url.Values{}
	query.Set("namespace", mcpNamespace)

	var listResult struct {
		Tools []map[string]interface{} `json:"tools"`
	}
	if err := apiClient.DoGet(ctx, "/api/v1/mcp/tools", query, &listResult); err != nil {
		return err
	}

	var toolID string
	for _, t := range listResult.Tools {
		if getStringValue(t, "name") == args[0] {
			toolID = getStringValue(t, "id")
			break
		}
	}

	if toolID == "" {
		return fmt.Errorf("tool not found: %s", args[0])
	}

	if err := apiClient.DoDelete(ctx, "/api/v1/mcp/tools/"+toolID); err != nil {
		return err
	}

	fmt.Printf("Deleted custom MCP tool: %s\n", args[0])
	return nil
}

func runMCPToolsSync(cmd *cobra.Command, args []string) error {
	// Auto-detect directory if not specified
	dir := mcpSyncDir
	if dir == "" {
		var err error
		dir, err = detectResourceDir("mcp-tools", "", "")
		if err != nil {
			return fmt.Errorf("no MCP tools directory found (looked for ./fluxbase/mcp-tools/ and ./mcp-tools/). Use --dir to specify")
		}
		fmt.Printf("Auto-detected MCP tools directory: %s\n", dir)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.ts"))
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No .ts files found in directory")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	for _, file := range files {
		code, err := os.ReadFile(file) //nolint:gosec // CLI tool reads user-provided file path
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", file, err)
			continue
		}

		// Parse annotations from code
		name, annotations := parseMCPAnnotations(string(code), filepath.Base(file))

		// Use namespace from annotation if specified, otherwise use CLI flag
		namespace := mcpNamespace
		if ns, ok := annotations["namespace"]; ok {
			namespace = ns.(string)
		}

		if mcpDryRun {
			fmt.Printf("Would sync tool: %s (namespace: %s, from %s)\n", name, namespace, filepath.Base(file))
			continue
		}

		payload := map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"code":      string(code),
			"upsert":    true,
		}

		// Apply annotations
		if desc, ok := annotations["description"]; ok {
			payload["description"] = desc
		}
		if timeout, ok := annotations["timeout"]; ok {
			payload["timeout_seconds"] = timeout
		}
		if memory, ok := annotations["memory"]; ok {
			payload["memory_limit_mb"] = memory
		}
		if _, ok := annotations["allow-net"]; ok {
			payload["allow_net"] = true
		}
		if _, ok := annotations["allow-env"]; ok {
			payload["allow_env"] = true
		}

		var result map[string]interface{}
		err = apiClient.DoPost(ctx, "/api/v1/mcp/tools/sync", payload, &result)
		if err != nil {
			fmt.Printf("Error syncing %s: %v\n", name, err)
			continue
		}

		fmt.Printf("Synced tool: %s\n", name)
	}

	return nil
}

func runMCPToolsTest(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First, find the tool by name to get its ID
	query := url.Values{}
	query.Set("namespace", mcpNamespace)

	var listResult struct {
		Tools []map[string]interface{} `json:"tools"`
	}
	if err := apiClient.DoGet(ctx, "/api/v1/mcp/tools", query, &listResult); err != nil {
		return err
	}

	var toolID string
	for _, t := range listResult.Tools {
		if getStringValue(t, "name") == args[0] {
			toolID = getStringValue(t, "id")
			break
		}
	}

	if toolID == "" {
		return fmt.Errorf("tool not found: %s", args[0])
	}

	var testArgs map[string]interface{}
	if err := json.Unmarshal([]byte(mcpTestArgs), &testArgs); err != nil {
		return fmt.Errorf("invalid args JSON: %w", err)
	}

	payload := map[string]interface{}{
		"args": testArgs,
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/mcp/tools/"+toolID+"/test", payload, &result); err != nil {
		return err
	}

	formatter := GetFormatter()
	if formatter.Format != output.FormatTable {
		return formatter.Print(result)
	}

	if success, ok := result["success"].(bool); ok && success {
		fmt.Println("Tool executed successfully!")
	} else {
		fmt.Println("Tool execution failed!")
	}

	if res, ok := result["result"].(map[string]interface{}); ok {
		if content, ok := res["content"].([]interface{}); ok {
			fmt.Println("\nResult:")
			for _, c := range content {
				if cm, ok := c.(map[string]interface{}); ok {
					if text, ok := cm["text"].(string); ok {
						fmt.Println(text)
					}
				}
			}
		}
	}

	return nil
}

// Helper functions

func parseMCPAnnotations(code, filename string) (name string, annotations map[string]interface{}) {
	annotations = make(map[string]interface{})

	// Default name from filename
	name = strings.TrimSuffix(filename, ".ts")
	name = strings.ReplaceAll(name, "-", "_")

	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "//") {
			continue
		}

		line = strings.TrimPrefix(line, "//")
		line = strings.TrimSpace(line)

		// Support @fluxbase: annotations (consistent with edge functions and jobs)
		if !strings.HasPrefix(line, "@fluxbase:") {
			continue
		}

		line = strings.TrimPrefix(line, "@fluxbase:")
		parts := strings.SplitN(line, " ", 2)

		key := parts[0]
		var value interface{} = true
		if len(parts) > 1 {
			value = strings.TrimSpace(parts[1])
		}

		annotations[key] = value

		// Override name if specified
		if key == "name" {
			name = value.(string)
		}
	}

	return name, annotations
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
