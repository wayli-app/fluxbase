package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage custom MCP tools and resources",
	Long:  `Create, deploy, and manage custom MCP (Model Context Protocol) tools and resources.`,
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
	mcpURI         string
	mcpMimeType    string
	mcpIsTemplate  bool
	mcpTestArgs    string
	mcpTestParams  string
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
Tool metadata is read from annotations in the file.

Annotations:
  // @fluxbase:mcp-tool
  // @fluxbase:name my_tool
  // @fluxbase:description My custom tool
  // @fluxbase:scopes read:tables,write:storage
  // @fluxbase:timeout 30
  // @fluxbase:memory 128
  // @fluxbase:allow-net
  // @fluxbase:allow-env

Examples:
  fluxbase mcp tools sync --dir ./mcp-tools
  fluxbase mcp tools sync --dir ./mcp-tools --namespace production
  fluxbase mcp tools sync --dir ./mcp-tools --dry-run`,
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

// Resources Commands

var mcpResourcesCmd = &cobra.Command{
	Use:     "resources",
	Aliases: []string{"resource", "res"},
	Short:   "Manage custom MCP resources",
	Long:    `Create, deploy, and manage custom MCP resources.`,
}

var mcpResourcesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all custom MCP resources",
	Long: `List all custom MCP resources.

Examples:
  fluxbase mcp resources list
  fluxbase mcp resources list --namespace production
  fluxbase mcp resources list -o json`,
	PreRunE: requireAuth,
	RunE:    runMCPResourcesList,
}

var mcpResourcesGetCmd = &cobra.Command{
	Use:   "get [uri]",
	Short: "Get custom MCP resource details",
	Long: `Get details of a specific custom MCP resource.

Examples:
  fluxbase mcp resources get fluxbase://custom/analytics
  fluxbase mcp resources get fluxbase://custom/analytics -o json`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runMCPResourcesGet,
}

var mcpResourcesCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new custom MCP resource",
	Long: `Create a new custom MCP resource.

Examples:
  fluxbase mcp resources create analytics --uri "fluxbase://custom/analytics" --code ./analytics.ts
  fluxbase mcp resources create analytics --uri "fluxbase://custom/users/{id}" --code ./user.ts --template`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runMCPResourcesCreate,
}

var mcpResourcesDeleteCmd = &cobra.Command{
	Use:     "delete [uri]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a custom MCP resource",
	Long: `Delete a custom MCP resource.

Examples:
  fluxbase mcp resources delete fluxbase://custom/analytics`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runMCPResourcesDelete,
}

var mcpResourcesSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync custom MCP resources from a directory",
	Long: `Sync custom MCP resources from a directory to the server.

Each .ts file in the directory will be synced as a custom resource.
Resource metadata is read from annotations in the file.

Annotations:
  // @fluxbase:mcp-resource
  // @fluxbase:uri fluxbase://custom/my-resource
  // @fluxbase:name My Resource
  // @fluxbase:description My custom resource
  // @fluxbase:mime-type application/json
  // @fluxbase:template (for parameterized URIs)
  // @fluxbase:scopes read:tables
  // @fluxbase:timeout 10
  // @fluxbase:cache-ttl 60

Examples:
  fluxbase mcp resources sync --dir ./mcp-resources
  fluxbase mcp resources sync --dir ./mcp-resources --namespace production
  fluxbase mcp resources sync --dir ./mcp-resources --dry-run`,
	PreRunE: requireAuth,
	RunE:    runMCPResourcesSync,
}

var mcpResourcesTestCmd = &cobra.Command{
	Use:   "test [uri]",
	Short: "Test a custom MCP resource",
	Long: `Test a custom MCP resource by reading it with sample parameters.

Examples:
  fluxbase mcp resources test fluxbase://custom/analytics
  fluxbase mcp resources test "fluxbase://custom/users/{id}" --params '{"id": "123"}'`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runMCPResourcesTest,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpToolsCmd)
	mcpCmd.AddCommand(mcpResourcesCmd)

	// Tools subcommands
	mcpToolsCmd.AddCommand(mcpToolsListCmd)
	mcpToolsCmd.AddCommand(mcpToolsGetCmd)
	mcpToolsCmd.AddCommand(mcpToolsCreateCmd)
	mcpToolsCmd.AddCommand(mcpToolsUpdateCmd)
	mcpToolsCmd.AddCommand(mcpToolsDeleteCmd)
	mcpToolsCmd.AddCommand(mcpToolsSyncCmd)
	mcpToolsCmd.AddCommand(mcpToolsTestCmd)

	// Resources subcommands
	mcpResourcesCmd.AddCommand(mcpResourcesListCmd)
	mcpResourcesCmd.AddCommand(mcpResourcesGetCmd)
	mcpResourcesCmd.AddCommand(mcpResourcesCreateCmd)
	mcpResourcesCmd.AddCommand(mcpResourcesDeleteCmd)
	mcpResourcesCmd.AddCommand(mcpResourcesSyncCmd)
	mcpResourcesCmd.AddCommand(mcpResourcesTestCmd)

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
	mcpToolsSyncCmd.Flags().StringVar(&mcpSyncDir, "dir", "", "Directory containing tool files (required)")
	mcpToolsSyncCmd.Flags().StringVar(&mcpNamespace, "namespace", "default", "Namespace")
	mcpToolsSyncCmd.Flags().BoolVar(&mcpDryRun, "dry-run", false, "Show what would be synced without making changes")
	_ = mcpToolsSyncCmd.MarkFlagRequired("dir")

	// Tools test flags
	mcpToolsTestCmd.Flags().StringVar(&mcpTestArgs, "args", "{}", "JSON arguments to pass to the tool")
	mcpToolsTestCmd.Flags().StringVar(&mcpNamespace, "namespace", "default", "Namespace")

	// Resources list flags
	mcpResourcesListCmd.Flags().StringVar(&mcpNamespace, "namespace", "", "Filter by namespace")

	// Resources create flags
	mcpResourcesCreateCmd.Flags().StringVar(&mcpURI, "uri", "", "Resource URI (required)")
	mcpResourcesCreateCmd.Flags().StringVar(&mcpCodeFile, "code", "", "Path to TypeScript code file (required)")
	mcpResourcesCreateCmd.Flags().StringVar(&mcpNamespace, "namespace", "default", "Namespace")
	mcpResourcesCreateCmd.Flags().StringVar(&mcpDescription, "description", "", "Resource description")
	mcpResourcesCreateCmd.Flags().StringVar(&mcpMimeType, "mime-type", "application/json", "MIME type")
	mcpResourcesCreateCmd.Flags().BoolVar(&mcpIsTemplate, "template", false, "URI contains parameters")
	mcpResourcesCreateCmd.Flags().IntVar(&mcpTimeout, "timeout", 10, "Execution timeout in seconds")
	_ = mcpResourcesCreateCmd.MarkFlagRequired("uri")
	_ = mcpResourcesCreateCmd.MarkFlagRequired("code")

	// Resources sync flags
	mcpResourcesSyncCmd.Flags().StringVar(&mcpSyncDir, "dir", "", "Directory containing resource files (required)")
	mcpResourcesSyncCmd.Flags().StringVar(&mcpNamespace, "namespace", "default", "Namespace")
	mcpResourcesSyncCmd.Flags().BoolVar(&mcpDryRun, "dry-run", false, "Show what would be synced without making changes")
	_ = mcpResourcesSyncCmd.MarkFlagRequired("dir")

	// Resources test flags
	mcpResourcesTestCmd.Flags().StringVar(&mcpTestParams, "params", "{}", "JSON parameters for template URIs")
	mcpResourcesTestCmd.Flags().StringVar(&mcpNamespace, "namespace", "default", "Namespace")
}

// Tool command implementations

func runMCPToolsList(cmd *cobra.Command, args []string) error {
	client, err := createAPIClient()
	if err != nil {
		return err
	}

	endpoint := "/api/v1/mcp/tools"
	if mcpNamespace != "" {
		endpoint += "?namespace=" + mcpNamespace
	}

	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return handleErrorResponse(resp)
	}

	var result struct {
		Tools []map[string]interface{} `json:"tools"`
		Count int                      `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if outputFormat == "json" {
		return output.JSON(result)
	}

	if len(result.Tools) == 0 {
		fmt.Println("No custom MCP tools found.")
		return nil
	}

	headers := []string{"NAME", "NAMESPACE", "ENABLED", "VERSION", "DESCRIPTION"}
	rows := make([][]string, len(result.Tools))
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
		rows[i] = []string{
			tool["name"].(string),
			tool["namespace"].(string),
			enabled,
			version,
			desc,
		}
	}

	output.Table(headers, rows)
	return nil
}

func runMCPToolsGet(cmd *cobra.Command, args []string) error {
	client, err := createAPIClient()
	if err != nil {
		return err
	}

	// First list to find by name
	endpoint := "/api/v1/mcp/tools?namespace=" + mcpNamespace
	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("failed to get tool: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return handleErrorResponse(resp)
	}

	var result struct {
		Tools []map[string]interface{} `json:"tools"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Find tool by name
	var tool map[string]interface{}
	for _, t := range result.Tools {
		if t["name"].(string) == args[0] {
			tool = t
			break
		}
	}

	if tool == nil {
		return fmt.Errorf("tool not found: %s", args[0])
	}

	if outputFormat == "json" {
		return output.JSON(tool)
	}

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

	return nil
}

func runMCPToolsCreate(cmd *cobra.Command, args []string) error {
	code, err := os.ReadFile(mcpCodeFile)
	if err != nil {
		return fmt.Errorf("failed to read code file: %w", err)
	}

	client, err := createAPIClient()
	if err != nil {
		return err
	}

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

	resp, err := client.Post("/api/v1/mcp/tools", payload)
	if err != nil {
		return fmt.Errorf("failed to create tool: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return handleErrorResponse(resp)
	}

	var tool map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tool); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if outputFormat == "json" {
		return output.JSON(tool)
	}

	fmt.Printf("Created custom MCP tool: %s\n", args[0])
	return nil
}

func runMCPToolsUpdate(cmd *cobra.Command, args []string) error {
	client, err := createAPIClient()
	if err != nil {
		return err
	}

	// First, find the tool by name to get its ID
	listResp, err := client.Get("/api/v1/mcp/tools?namespace=" + mcpNamespace)
	if err != nil {
		return fmt.Errorf("failed to find tool: %w", err)
	}
	defer listResp.Body.Close()

	var listResult struct {
		Tools []map[string]interface{} `json:"tools"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listResult); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	var toolID string
	for _, t := range listResult.Tools {
		if t["name"].(string) == args[0] {
			toolID = t["id"].(string)
			break
		}
	}

	if toolID == "" {
		return fmt.Errorf("tool not found: %s", args[0])
	}

	payload := make(map[string]interface{})
	if mcpCodeFile != "" {
		code, err := os.ReadFile(mcpCodeFile)
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

	resp, err := client.Put("/api/v1/mcp/tools/"+toolID, payload)
	if err != nil {
		return fmt.Errorf("failed to update tool: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return handleErrorResponse(resp)
	}

	fmt.Printf("Updated custom MCP tool: %s\n", args[0])
	return nil
}

func runMCPToolsDelete(cmd *cobra.Command, args []string) error {
	client, err := createAPIClient()
	if err != nil {
		return err
	}

	// First, find the tool by name to get its ID
	listResp, err := client.Get("/api/v1/mcp/tools?namespace=" + mcpNamespace)
	if err != nil {
		return fmt.Errorf("failed to find tool: %w", err)
	}
	defer listResp.Body.Close()

	var listResult struct {
		Tools []map[string]interface{} `json:"tools"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listResult); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	var toolID string
	for _, t := range listResult.Tools {
		if t["name"].(string) == args[0] {
			toolID = t["id"].(string)
			break
		}
	}

	if toolID == "" {
		return fmt.Errorf("tool not found: %s", args[0])
	}

	resp, err := client.Delete("/api/v1/mcp/tools/" + toolID)
	if err != nil {
		return fmt.Errorf("failed to delete tool: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return handleErrorResponse(resp)
	}

	fmt.Printf("Deleted custom MCP tool: %s\n", args[0])
	return nil
}

func runMCPToolsSync(cmd *cobra.Command, args []string) error {
	files, err := filepath.Glob(filepath.Join(mcpSyncDir, "*.ts"))
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No .ts files found in directory")
		return nil
	}

	client, err := createAPIClient()
	if err != nil {
		return err
	}

	for _, file := range files {
		code, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", file, err)
			continue
		}

		// Parse annotations from code
		name, annotations := parseMCPAnnotations(string(code), filepath.Base(file))

		// Skip if not marked as MCP tool
		if _, isTool := annotations["tool"]; !isTool {
			continue
		}

		if mcpDryRun {
			fmt.Printf("Would sync tool: %s (from %s)\n", name, filepath.Base(file))
			continue
		}

		payload := map[string]interface{}{
			"name":      name,
			"namespace": mcpNamespace,
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

		resp, err := client.Post("/api/v1/mcp/tools/sync", payload)
		if err != nil {
			fmt.Printf("Error syncing %s: %v\n", name, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == 200 || resp.StatusCode == 201 {
			fmt.Printf("Synced tool: %s\n", name)
		} else {
			fmt.Printf("Failed to sync %s: %d\n", name, resp.StatusCode)
		}
	}

	return nil
}

func runMCPToolsTest(cmd *cobra.Command, args []string) error {
	client, err := createAPIClient()
	if err != nil {
		return err
	}

	// First, find the tool by name to get its ID
	listResp, err := client.Get("/api/v1/mcp/tools?namespace=" + mcpNamespace)
	if err != nil {
		return fmt.Errorf("failed to find tool: %w", err)
	}
	defer listResp.Body.Close()

	var listResult struct {
		Tools []map[string]interface{} `json:"tools"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listResult); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	var toolID string
	for _, t := range listResult.Tools {
		if t["name"].(string) == args[0] {
			toolID = t["id"].(string)
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

	resp, err := client.Post("/api/v1/mcp/tools/"+toolID+"/test", payload)
	if err != nil {
		return fmt.Errorf("failed to test tool: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if outputFormat == "json" {
		return output.JSON(result)
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

// Resource command implementations

func runMCPResourcesList(cmd *cobra.Command, args []string) error {
	client, err := createAPIClient()
	if err != nil {
		return err
	}

	endpoint := "/api/v1/mcp/resources"
	if mcpNamespace != "" {
		endpoint += "?namespace=" + mcpNamespace
	}

	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return handleErrorResponse(resp)
	}

	var result struct {
		Resources []map[string]interface{} `json:"resources"`
		Count     int                      `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if outputFormat == "json" {
		return output.JSON(result)
	}

	if len(result.Resources) == 0 {
		fmt.Println("No custom MCP resources found.")
		return nil
	}

	headers := []string{"URI", "NAME", "TEMPLATE", "ENABLED"}
	rows := make([][]string, len(result.Resources))
	for i, res := range result.Resources {
		isTemplate := "no"
		if t, ok := res["is_template"].(bool); ok && t {
			isTemplate = "yes"
		}
		enabled := "true"
		if e, ok := res["enabled"].(bool); ok && !e {
			enabled = "false"
		}
		rows[i] = []string{
			res["uri"].(string),
			res["name"].(string),
			isTemplate,
			enabled,
		}
	}

	output.Table(headers, rows)
	return nil
}

func runMCPResourcesGet(cmd *cobra.Command, args []string) error {
	client, err := createAPIClient()
	if err != nil {
		return err
	}

	// List to find by URI
	endpoint := "/api/v1/mcp/resources?namespace=" + mcpNamespace
	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return handleErrorResponse(resp)
	}

	var result struct {
		Resources []map[string]interface{} `json:"resources"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	var resource map[string]interface{}
	for _, r := range result.Resources {
		if r["uri"].(string) == args[0] {
			resource = r
			break
		}
	}

	if resource == nil {
		return fmt.Errorf("resource not found: %s", args[0])
	}

	if outputFormat == "json" {
		return output.JSON(resource)
	}

	fmt.Printf("URI:         %s\n", resource["uri"])
	fmt.Printf("Name:        %s\n", resource["name"])
	fmt.Printf("Namespace:   %s\n", resource["namespace"])
	fmt.Printf("MIME Type:   %s\n", resource["mime_type"])
	fmt.Printf("Template:    %v\n", resource["is_template"])
	fmt.Printf("Enabled:     %v\n", resource["enabled"])
	if code, ok := resource["code"].(string); ok {
		fmt.Printf("\nCode:\n%s\n", code)
	}

	return nil
}

func runMCPResourcesCreate(cmd *cobra.Command, args []string) error {
	code, err := os.ReadFile(mcpCodeFile)
	if err != nil {
		return fmt.Errorf("failed to read code file: %w", err)
	}

	client, err := createAPIClient()
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"uri":             mcpURI,
		"name":            args[0],
		"namespace":       mcpNamespace,
		"code":            string(code),
		"description":     mcpDescription,
		"mime_type":       mcpMimeType,
		"is_template":     mcpIsTemplate,
		"timeout_seconds": mcpTimeout,
	}

	resp, err := client.Post("/api/v1/mcp/resources", payload)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return handleErrorResponse(resp)
	}

	var resource map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&resource); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if outputFormat == "json" {
		return output.JSON(resource)
	}

	fmt.Printf("Created custom MCP resource: %s\n", mcpURI)
	return nil
}

func runMCPResourcesDelete(cmd *cobra.Command, args []string) error {
	client, err := createAPIClient()
	if err != nil {
		return err
	}

	// First, find the resource by URI to get its ID
	listResp, err := client.Get("/api/v1/mcp/resources?namespace=" + mcpNamespace)
	if err != nil {
		return fmt.Errorf("failed to find resource: %w", err)
	}
	defer listResp.Body.Close()

	var listResult struct {
		Resources []map[string]interface{} `json:"resources"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listResult); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	var resourceID string
	for _, r := range listResult.Resources {
		if r["uri"].(string) == args[0] {
			resourceID = r["id"].(string)
			break
		}
	}

	if resourceID == "" {
		return fmt.Errorf("resource not found: %s", args[0])
	}

	resp, err := client.Delete("/api/v1/mcp/resources/" + resourceID)
	if err != nil {
		return fmt.Errorf("failed to delete resource: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return handleErrorResponse(resp)
	}

	fmt.Printf("Deleted custom MCP resource: %s\n", args[0])
	return nil
}

func runMCPResourcesSync(cmd *cobra.Command, args []string) error {
	files, err := filepath.Glob(filepath.Join(mcpSyncDir, "*.ts"))
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No .ts files found in directory")
		return nil
	}

	client, err := createAPIClient()
	if err != nil {
		return err
	}

	for _, file := range files {
		code, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", file, err)
			continue
		}

		// Parse annotations from code
		name, annotations := parseMCPAnnotations(string(code), filepath.Base(file))

		// Skip if not marked as MCP resource
		if _, isResource := annotations["resource"]; !isResource {
			continue
		}

		uri, ok := annotations["uri"]
		if !ok {
			fmt.Printf("Skipping %s: missing @fluxbase:uri annotation\n", file)
			continue
		}

		if mcpDryRun {
			fmt.Printf("Would sync resource: %s (from %s)\n", uri, filepath.Base(file))
			continue
		}

		payload := map[string]interface{}{
			"uri":       uri,
			"name":      name,
			"namespace": mcpNamespace,
			"code":      string(code),
			"upsert":    true,
		}

		// Apply annotations
		if desc, ok := annotations["description"]; ok {
			payload["description"] = desc
		}
		if mimeType, ok := annotations["mime-type"]; ok {
			payload["mime_type"] = mimeType
		}
		if _, ok := annotations["template"]; ok {
			payload["is_template"] = true
		}
		if timeout, ok := annotations["timeout"]; ok {
			payload["timeout_seconds"] = timeout
		}

		resp, err := client.Post("/api/v1/mcp/resources/sync", payload)
		if err != nil {
			fmt.Printf("Error syncing %s: %v\n", uri, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == 200 || resp.StatusCode == 201 {
			fmt.Printf("Synced resource: %s\n", uri)
		} else {
			fmt.Printf("Failed to sync %s: %d\n", uri, resp.StatusCode)
		}
	}

	return nil
}

func runMCPResourcesTest(cmd *cobra.Command, args []string) error {
	client, err := createAPIClient()
	if err != nil {
		return err
	}

	// First, find the resource by URI to get its ID
	listResp, err := client.Get("/api/v1/mcp/resources?namespace=" + mcpNamespace)
	if err != nil {
		return fmt.Errorf("failed to find resource: %w", err)
	}
	defer listResp.Body.Close()

	var listResult struct {
		Resources []map[string]interface{} `json:"resources"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listResult); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	var resourceID string
	for _, r := range listResult.Resources {
		if r["uri"].(string) == args[0] {
			resourceID = r["id"].(string)
			break
		}
	}

	if resourceID == "" {
		return fmt.Errorf("resource not found: %s", args[0])
	}

	var testParams map[string]string
	if err := json.Unmarshal([]byte(mcpTestParams), &testParams); err != nil {
		return fmt.Errorf("invalid params JSON: %w", err)
	}

	payload := map[string]interface{}{
		"params": testParams,
	}

	resp, err := client.Post("/api/v1/mcp/resources/"+resourceID+"/test", payload)
	if err != nil {
		return fmt.Errorf("failed to test resource: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if outputFormat == "json" {
		return output.JSON(result)
	}

	if success, ok := result["success"].(bool); ok && success {
		fmt.Println("Resource read successfully!")
	} else {
		fmt.Println("Resource read failed!")
	}

	if contents, ok := result["contents"].([]interface{}); ok {
		fmt.Println("\nContents:")
		for _, c := range contents {
			if cm, ok := c.(map[string]interface{}); ok {
				if text, ok := cm["text"].(string); ok {
					fmt.Println(text)
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

		// Handle special mcp-tool and mcp-resource type markers
		if line == "mcp-tool" || strings.HasPrefix(line, "mcp-tool ") {
			annotations["tool"] = true
			continue
		}
		if line == "mcp-resource" || strings.HasPrefix(line, "mcp-resource ") {
			annotations["resource"] = true
			continue
		}

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
