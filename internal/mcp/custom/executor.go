package custom

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/fluxbase-eu/fluxbase/internal/runtime"
	"github.com/fluxbase-eu/fluxbase/internal/secrets"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Executor handles execution of custom MCP tools and resources using the Deno runtime.
type Executor struct {
	runtime        *runtime.DenoRuntime
	secretsService secrets.Service
	publicURL      string
	jwtSecret      string
}

// NewExecutor creates a new Executor instance.
func NewExecutor(jwtSecret, publicURL string, secretsService secrets.Service) *Executor {
	// Create a dedicated Deno runtime for MCP tools
	r := runtime.NewRuntime(
		runtime.RuntimeTypeFunction,
		jwtSecret,
		publicURL,
		runtime.WithTimeout(30*time.Second),
		runtime.WithMemoryLimit(128),
		runtime.WithMaxOutputSize(10*1024*1024), // 10MB
	)

	return &Executor{
		runtime:        r,
		secretsService: secretsService,
		publicURL:      publicURL,
		jwtSecret:      jwtSecret,
	}
}

// ExecuteTool executes a custom MCP tool and returns the result.
func (e *Executor) ExecuteTool(
	ctx context.Context,
	tool *CustomTool,
	args map[string]any,
	authCtx *mcp.AuthContext,
) (*mcp.ToolResult, error) {
	start := time.Now()

	// Get secrets if needed
	var secretsMap map[string]string
	if tool.AllowEnv && e.secretsService != nil {
		var err error
		secretsMap, err = e.secretsService.GetAllDecrypted(ctx, tool.Namespace)
		if err != nil {
			log.Warn().Err(err).Str("tool", tool.Name).Msg("Failed to get secrets for custom tool")
		}
	}

	// Build execution context
	execCtx := map[string]any{
		"tool_name":  tool.Name,
		"namespace":  tool.Namespace,
		"user_id":    "",
		"user_email": "",
		"user_role":  "",
		"scopes":     []string{},
	}
	if authCtx != nil {
		execCtx["user_id"] = authCtx.UserID
		execCtx["user_email"] = authCtx.UserEmail
		execCtx["user_role"] = authCtx.UserRole
		execCtx["scopes"] = authCtx.Scopes
	}

	// Wrap the user code with MCP tool runtime bridge
	wrappedCode := e.wrapToolCode(tool.Code, args, execCtx)

	// Create execution request
	req := runtime.ExecutionRequest{
		ID:        uuid.New(),
		Name:      "mcp_tool_" + tool.Name,
		Namespace: tool.Namespace,
	}

	// Set permissions based on tool configuration
	perms := runtime.Permissions{
		AllowNet:      tool.AllowNet,
		AllowEnv:      tool.AllowEnv,
		AllowRead:     tool.AllowRead,
		AllowWrite:    tool.AllowWrite,
		MemoryLimitMB: tool.MemoryLimitMB,
	}

	// Execute with timeout
	timeout := time.Duration(tool.TimeoutSeconds) * time.Second
	result, err := e.runtime.Execute(ctx, wrappedCode, req, perms, nil, &timeout, secretsMap)

	duration := time.Since(start)

	if err != nil {
		log.Error().
			Err(err).
			Str("tool", tool.Name).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("Custom MCP tool execution failed")

		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Tool execution failed: %v", err))},
			IsError: true,
		}, nil
	}

	// Parse the result
	return e.parseToolResult(result)
}

// ExecuteResource executes a custom MCP resource handler and returns the contents.
func (e *Executor) ExecuteResource(
	ctx context.Context,
	resource *CustomResource,
	params map[string]string,
	authCtx *mcp.AuthContext,
) ([]mcp.Content, error) {
	start := time.Now()

	// Build execution context
	execCtx := map[string]any{
		"resource_uri":  resource.URI,
		"resource_name": resource.Name,
		"namespace":     resource.Namespace,
		"params":        params,
		"user_id":       "",
		"user_email":    "",
		"user_role":     "",
		"scopes":        []string{},
	}
	if authCtx != nil {
		execCtx["user_id"] = authCtx.UserID
		execCtx["user_email"] = authCtx.UserEmail
		execCtx["user_role"] = authCtx.UserRole
		execCtx["scopes"] = authCtx.Scopes
	}

	// Wrap the user code with MCP resource runtime bridge
	wrappedCode := e.wrapResourceCode(resource.Code, params, execCtx)

	// Create execution request
	req := runtime.ExecutionRequest{
		ID:        uuid.New(),
		Name:      "mcp_resource_" + resource.Name,
		Namespace: resource.Namespace,
	}

	// Resources typically need network access for database queries
	perms := runtime.Permissions{
		AllowNet:   true,
		AllowEnv:   false,
		AllowRead:  false,
		AllowWrite: false,
	}

	// Execute with timeout
	timeout := time.Duration(resource.TimeoutSeconds) * time.Second
	result, err := e.runtime.Execute(ctx, wrappedCode, req, perms, nil, &timeout, nil)

	duration := time.Since(start)

	if err != nil {
		log.Error().
			Err(err).
			Str("resource", resource.URI).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("Custom MCP resource execution failed")

		return nil, fmt.Errorf("resource execution failed: %w", err)
	}

	// Parse the result
	return e.parseResourceResult(result)
}

// wrapToolCode wraps user tool code with the MCP runtime bridge.
func (e *Executor) wrapToolCode(code string, args map[string]any, execCtx map[string]any) string {
	argsJSON, _ := json.Marshal(args)
	ctxJSON, _ := json.Marshal(execCtx)

	return fmt.Sprintf(`
// MCP Tool Runtime Bridge
const __MCP_ARGS__ = %s;
const __MCP_CONTEXT__ = %s;
const __FLUXBASE_URL__ = Deno.env.get("FLUXBASE_PUBLIC_URL") || "";
const __FLUXBASE_TOKEN__ = Deno.env.get("FLUXBASE_USER_TOKEN") || "";

// Create Fluxbase SDK context for tools
const context = {
	args: __MCP_ARGS__,
	...__MCP_CONTEXT__,

	// Secrets accessor
	secrets: {
		get: (name) => Deno.env.get("FLUXBASE_SECRET_" + name.toUpperCase()),
	},

	// Simple fetch wrapper for Fluxbase API
	fluxbase: {
		url: __FLUXBASE_URL__,
		token: __FLUXBASE_TOKEN__,
		async fetch(path, options = {}) {
			const url = this.url + path;
			const headers = {
				"Content-Type": "application/json",
				...(this.token ? { "Authorization": "Bearer " + this.token } : {}),
				...(options.headers || {}),
			};
			const response = await fetch(url, { ...options, headers });
			if (!response.ok) {
				throw new Error("Fluxbase API error: " + response.status + " " + response.statusText);
			}
			return response.json();
		},
		async from(table) {
			return {
				_table: table,
				_select: "*",
				_filters: [],
				select(columns) {
					this._select = columns;
					return this;
				},
				eq(column, value) {
					this._filters.push(column + "=eq." + encodeURIComponent(value));
					return this;
				},
				async execute() {
					let path = "/rest/v1/" + this._table + "?select=" + encodeURIComponent(this._select);
					if (this._filters.length > 0) {
						path += "&" + this._filters.join("&");
					}
					return context.fluxbase.fetch(path);
				}
			};
		}
	}
};

// User-defined tool handler
%s

// Execute the handler
(async () => {
	try {
		// Look for default export or handler function
		const handlerFn = typeof handler === "function" ? handler :
			(typeof default_export === "function" ? default_export : null);

		if (!handlerFn) {
			throw new Error("Tool must export a 'handler' or default function");
		}

		const result = await handlerFn(__MCP_ARGS__, context);

		// Normalize result to MCP format
		let mcpResult;
		if (result && result.content) {
			// Already in MCP format
			mcpResult = result;
		} else if (typeof result === "string") {
			mcpResult = { content: [{ type: "text", text: result }] };
		} else if (result === null || result === undefined) {
			mcpResult = { content: [{ type: "text", text: "OK" }] };
		} else {
			mcpResult = { content: [{ type: "text", text: JSON.stringify(result, null, 2) }] };
		}

		console.log("__RESULT__::" + JSON.stringify({
			status: 200,
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify(mcpResult)
		}));
	} catch (error) {
		console.log("__RESULT__::" + JSON.stringify({
			status: 500,
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				content: [{ type: "text", text: error.message || String(error) }],
				isError: true
			})
		}));
	}
})();
`, string(argsJSON), string(ctxJSON), code)
}

// wrapResourceCode wraps user resource code with the MCP runtime bridge.
func (e *Executor) wrapResourceCode(code string, params map[string]string, execCtx map[string]any) string {
	paramsJSON, _ := json.Marshal(params)
	ctxJSON, _ := json.Marshal(execCtx)

	return fmt.Sprintf(`
// MCP Resource Runtime Bridge
const __MCP_PARAMS__ = %s;
const __MCP_CONTEXT__ = %s;
const __FLUXBASE_URL__ = Deno.env.get("FLUXBASE_PUBLIC_URL") || "";
const __FLUXBASE_TOKEN__ = Deno.env.get("FLUXBASE_USER_TOKEN") || "";

// Create Fluxbase SDK context for resources
const context = {
	params: __MCP_PARAMS__,
	...__MCP_CONTEXT__,

	// Simple fetch wrapper for Fluxbase API
	fluxbase: {
		url: __FLUXBASE_URL__,
		token: __FLUXBASE_TOKEN__,
		async fetch(path, options = {}) {
			const url = this.url + path;
			const headers = {
				"Content-Type": "application/json",
				...(this.token ? { "Authorization": "Bearer " + this.token } : {}),
				...(options.headers || {}),
			};
			const response = await fetch(url, { ...options, headers });
			if (!response.ok) {
				throw new Error("Fluxbase API error: " + response.status + " " + response.statusText);
			}
			return response.json();
		},
		async from(table) {
			return {
				_table: table,
				_select: "*",
				_filters: [],
				select(columns) {
					this._select = columns;
					return this;
				},
				eq(column, value) {
					this._filters.push(column + "=eq." + encodeURIComponent(value));
					return this;
				},
				async execute() {
					let path = "/rest/v1/" + this._table + "?select=" + encodeURIComponent(this._select);
					if (this._filters.length > 0) {
						path += "&" + this._filters.join("&");
					}
					return context.fluxbase.fetch(path);
				}
			};
		}
	}
};

// User-defined resource handler
%s

// Execute the handler
(async () => {
	try {
		// Look for default export or handler function
		const handlerFn = typeof handler === "function" ? handler :
			(typeof default_export === "function" ? default_export : null);

		if (!handlerFn) {
			throw new Error("Resource must export a 'handler' or default function");
		}

		const result = await handlerFn(__MCP_PARAMS__, context);

		// Normalize result to MCP content array
		let contents;
		if (Array.isArray(result)) {
			contents = result;
		} else if (typeof result === "string") {
			contents = [{ type: "text", text: result }];
		} else if (result === null || result === undefined) {
			contents = [{ type: "text", text: "" }];
		} else {
			contents = [{ type: "text", text: JSON.stringify(result, null, 2) }];
		}

		console.log("__RESULT__::" + JSON.stringify({
			status: 200,
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ contents })
		}));
	} catch (error) {
		console.log("__RESULT__::" + JSON.stringify({
			status: 500,
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ error: error.message || String(error) })
		}));
	}
})();
`, string(paramsJSON), string(ctxJSON), code)
}

// parseToolResult parses the Deno execution result into an MCP ToolResult.
func (e *Executor) parseToolResult(result *runtime.ExecutionResult) (*mcp.ToolResult, error) {
	if !result.Success {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(result.Error)},
			IsError: true,
		}, nil
	}

	// Parse the body as MCP result
	var mcpResult struct {
		Content []mcp.Content `json:"content"`
		IsError bool          `json:"isError"`
	}

	if err := json.Unmarshal([]byte(result.Body), &mcpResult); err != nil {
		// If parsing fails, treat the body as plain text
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.TextContent(result.Body)},
		}, nil
	}

	return &mcp.ToolResult{
		Content: mcpResult.Content,
		IsError: mcpResult.IsError,
	}, nil
}

// parseResourceResult parses the Deno execution result into MCP Content array.
func (e *Executor) parseResourceResult(result *runtime.ExecutionResult) ([]mcp.Content, error) {
	if !result.Success {
		return nil, fmt.Errorf("resource execution failed: %s", result.Error)
	}

	// Parse the body
	var response struct {
		Contents []mcp.Content `json:"contents"`
		Error    string        `json:"error"`
	}

	if err := json.Unmarshal([]byte(result.Body), &response); err != nil {
		// If parsing fails, treat the body as plain text content
		return []mcp.Content{mcp.TextContent(result.Body)}, nil
	}

	if response.Error != "" {
		return nil, fmt.Errorf("resource error: %s", response.Error)
	}

	return response.Contents, nil
}

// ValidateToolCode performs basic validation on tool code.
func ValidateToolCode(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("code cannot be empty")
	}

	// Check for handler export
	if !strings.Contains(code, "handler") &&
		!strings.Contains(code, "export default") &&
		!strings.Contains(code, "export async function") {
		return fmt.Errorf("code must export a 'handler' function or use default export")
	}

	return nil
}

// ValidateResourceCode performs basic validation on resource code.
func ValidateResourceCode(code string) error {
	return ValidateToolCode(code) // Same validation rules
}
