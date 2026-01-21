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
	secretsService *secrets.Storage
	publicURL      string
	jwtSecret      string
}

// NewExecutor creates a new Executor instance.
func NewExecutor(jwtSecret, publicURL string, secretsService *secrets.Storage) *Executor {
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
		secretsMap, err = e.secretsService.GetSecretsForNamespace(ctx, tool.Namespace)
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
		if authCtx.UserID != nil {
			execCtx["user_id"] = *authCtx.UserID
		}
		execCtx["user_email"] = authCtx.UserEmail
		execCtx["user_role"] = authCtx.UserRole
		execCtx["scopes"] = authCtx.Scopes
	}

	// Wrap the user code with MCP tool runtime bridge
	wrappedCode := e.wrapToolCode(tool.Code, args, execCtx)

	// Create execution request with user context for token generation
	req := runtime.ExecutionRequest{
		ID:        uuid.New(),
		Name:      "mcp_tool_" + tool.Name,
		Namespace: tool.Namespace,
	}
	// Pass user context to execution request for proper token generation
	if authCtx != nil {
		if authCtx.UserID != nil {
			req.UserID = *authCtx.UserID
		}
		req.UserEmail = authCtx.UserEmail
		req.UserRole = authCtx.UserRole
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
		if authCtx.UserID != nil {
			execCtx["user_id"] = *authCtx.UserID
		}
		execCtx["user_email"] = authCtx.UserEmail
		execCtx["user_role"] = authCtx.UserRole
		execCtx["scopes"] = authCtx.Scopes
	}

	// Wrap the user code with MCP resource runtime bridge
	wrappedCode := e.wrapResourceCode(resource.Code, params, execCtx)

	// Create execution request with user context
	req := runtime.ExecutionRequest{
		ID:        uuid.New(),
		Name:      "mcp_resource_" + resource.Name,
		Namespace: resource.Namespace,
	}
	if authCtx != nil {
		if authCtx.UserID != nil {
			req.UserID = *authCtx.UserID
		}
		req.UserEmail = authCtx.UserEmail
		req.UserRole = authCtx.UserRole
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

// fluxbaseSDKCode returns the JavaScript code for the Fluxbase SDK clients.
// This provides both user-scoped (fluxbase) and service-scoped (fluxbaseService) clients.
func fluxbaseSDKCode() string {
	return `
// Fluxbase SDK - Query Builder
class QueryBuilder {
	constructor(client, table) {
		this._client = client;
		this._table = table;
		this._select = "*";
		this._filters = [];
		this._order = [];
		this._limit = null;
		this._offset = null;
		this._single = false;
	}

	select(columns) {
		this._select = columns;
		return this;
	}

	eq(column, value) {
		this._filters.push(column + "=eq." + encodeURIComponent(String(value)));
		return this;
	}

	neq(column, value) {
		this._filters.push(column + "=neq." + encodeURIComponent(String(value)));
		return this;
	}

	gt(column, value) {
		this._filters.push(column + "=gt." + encodeURIComponent(String(value)));
		return this;
	}

	gte(column, value) {
		this._filters.push(column + "=gte." + encodeURIComponent(String(value)));
		return this;
	}

	lt(column, value) {
		this._filters.push(column + "=lt." + encodeURIComponent(String(value)));
		return this;
	}

	lte(column, value) {
		this._filters.push(column + "=lte." + encodeURIComponent(String(value)));
		return this;
	}

	like(column, pattern) {
		this._filters.push(column + "=like." + encodeURIComponent(pattern));
		return this;
	}

	ilike(column, pattern) {
		this._filters.push(column + "=ilike." + encodeURIComponent(pattern));
		return this;
	}

	is(column, value) {
		this._filters.push(column + "=is." + encodeURIComponent(String(value)));
		return this;
	}

	in(column, values) {
		this._filters.push(column + "=in.(" + values.map(v => encodeURIComponent(String(v))).join(",") + ")");
		return this;
	}

	contains(column, value) {
		this._filters.push(column + "=cs." + encodeURIComponent(JSON.stringify(value)));
		return this;
	}

	containedBy(column, value) {
		this._filters.push(column + "=cd." + encodeURIComponent(JSON.stringify(value)));
		return this;
	}

	order(column, options = {}) {
		const dir = options.ascending === false ? "desc" : "asc";
		const nulls = options.nullsFirst ? ".nullsfirst" : "";
		this._order.push(column + "." + dir + nulls);
		return this;
	}

	limit(count) {
		this._limit = count;
		return this;
	}

	offset(count) {
		this._offset = count;
		return this;
	}

	single() {
		this._single = true;
		this._limit = 1;
		return this;
	}

	maybeSingle() {
		this._single = true;
		this._limit = 1;
		return this;
	}

	async execute() {
		let path = "/rest/v1/" + this._table + "?select=" + encodeURIComponent(this._select);

		if (this._filters.length > 0) {
			path += "&" + this._filters.join("&");
		}

		if (this._order.length > 0) {
			path += "&order=" + this._order.join(",");
		}

		if (this._limit !== null) {
			path += "&limit=" + this._limit;
		}

		if (this._offset !== null) {
			path += "&offset=" + this._offset;
		}

		const result = await this._client._fetch(path);

		if (this._single) {
			return { data: result[0] || null, error: null };
		}

		return { data: result, error: null };
	}
}

// Fluxbase SDK - Insert Builder
class InsertBuilder {
	constructor(client, table, data) {
		this._client = client;
		this._table = table;
		this._data = data;
		this._select = null;
		this._onConflict = null;
	}

	select(columns) {
		this._select = columns;
		return this;
	}

	onConflict(columns) {
		this._onConflict = columns;
		return this;
	}

	async execute() {
		let path = "/rest/v1/" + this._table;
		const params = [];

		if (this._select) {
			params.push("select=" + encodeURIComponent(this._select));
		}

		if (this._onConflict) {
			params.push("on_conflict=" + encodeURIComponent(this._onConflict));
		}

		if (params.length > 0) {
			path += "?" + params.join("&");
		}

		const result = await this._client._fetch(path, {
			method: "POST",
			body: JSON.stringify(this._data),
			headers: { "Prefer": "return=representation" }
		});

		return { data: result, error: null };
	}
}

// Fluxbase SDK - Update Builder
class UpdateBuilder {
	constructor(client, table, data) {
		this._client = client;
		this._table = table;
		this._data = data;
		this._filters = [];
		this._select = null;
	}

	eq(column, value) {
		this._filters.push(column + "=eq." + encodeURIComponent(String(value)));
		return this;
	}

	neq(column, value) {
		this._filters.push(column + "=neq." + encodeURIComponent(String(value)));
		return this;
	}

	select(columns) {
		this._select = columns;
		return this;
	}

	async execute() {
		let path = "/rest/v1/" + this._table;
		const params = [...this._filters];

		if (this._select) {
			params.push("select=" + encodeURIComponent(this._select));
		}

		if (params.length > 0) {
			path += "?" + params.join("&");
		}

		const result = await this._client._fetch(path, {
			method: "PATCH",
			body: JSON.stringify(this._data),
			headers: { "Prefer": "return=representation" }
		});

		return { data: result, error: null };
	}
}

// Fluxbase SDK - Delete Builder
class DeleteBuilder {
	constructor(client, table) {
		this._client = client;
		this._table = table;
		this._filters = [];
		this._select = null;
	}

	eq(column, value) {
		this._filters.push(column + "=eq." + encodeURIComponent(String(value)));
		return this;
	}

	neq(column, value) {
		this._filters.push(column + "=neq." + encodeURIComponent(String(value)));
		return this;
	}

	select(columns) {
		this._select = columns;
		return this;
	}

	async execute() {
		let path = "/rest/v1/" + this._table;
		const params = [...this._filters];

		if (this._select) {
			params.push("select=" + encodeURIComponent(this._select));
		}

		if (params.length > 0) {
			path += "?" + params.join("&");
		}

		const result = await this._client._fetch(path, {
			method: "DELETE",
			headers: { "Prefer": "return=representation" }
		});

		return { data: result, error: null };
	}
}

// Fluxbase SDK - Client
class FluxbaseClient {
	constructor(url, token) {
		this.url = url;
		this.token = token;
	}

	async _fetch(path, options = {}) {
		const url = this.url + path;
		const headers = {
			"Content-Type": "application/json",
			...(this.token ? { "Authorization": "Bearer " + this.token } : {}),
			...(options.headers || {}),
		};

		const response = await fetch(url, { ...options, headers });

		if (!response.ok) {
			const errorBody = await response.text();
			throw new Error("Fluxbase API error: " + response.status + " " + response.statusText + " - " + errorBody);
		}

		const text = await response.text();
		return text ? JSON.parse(text) : null;
	}

	from(table) {
		return new QueryBuilder(this, table);
	}

	insert(table, data) {
		return new InsertBuilder(this, table, data);
	}

	update(table, data) {
		return new UpdateBuilder(this, table, data);
	}

	delete(table) {
		return new DeleteBuilder(this, table);
	}

	async rpc(functionName, params = {}) {
		const path = "/rest/v1/rpc/" + functionName;
		const result = await this._fetch(path, {
			method: "POST",
			body: JSON.stringify(params),
		});
		return { data: result, error: null };
	}

	// Storage operations
	storage = {
		_client: this,

		async list(bucket, options = {}) {
			const params = new URLSearchParams();
			if (options.prefix) params.set("prefix", options.prefix);
			if (options.limit) params.set("limit", String(options.limit));
			if (options.offset) params.set("offset", String(options.offset));

			const path = "/storage/v1/object/list/" + bucket + (params.toString() ? "?" + params : "");
			return this._client._fetch(path);
		},

		async download(bucket, path) {
			const url = this._client.url + "/storage/v1/object/" + bucket + "/" + path;
			const response = await fetch(url, {
				headers: this._client.token ? { "Authorization": "Bearer " + this._client.token } : {}
			});
			if (!response.ok) throw new Error("Storage download error: " + response.status);
			return response;
		},

		async upload(bucket, path, file, options = {}) {
			const url = this._client.url + "/storage/v1/object/" + bucket + "/" + path;
			const headers = {
				...(this._client.token ? { "Authorization": "Bearer " + this._client.token } : {}),
				...(options.contentType ? { "Content-Type": options.contentType } : {}),
			};

			const response = await fetch(url, {
				method: "POST",
				headers,
				body: file,
			});

			if (!response.ok) throw new Error("Storage upload error: " + response.status);
			return response.json();
		},

		async remove(bucket, paths) {
			const pathList = Array.isArray(paths) ? paths : [paths];
			const url = this._client.url + "/storage/v1/object/" + bucket;
			return this._client._fetch(url, {
				method: "DELETE",
				body: JSON.stringify({ prefixes: pathList }),
			});
		},

		getPublicUrl(bucket, path) {
			return this._client.url + "/storage/v1/object/public/" + bucket + "/" + path;
		}
	};

	// Functions invocation
	functions = {
		_client: this,

		async invoke(functionName, options = {}) {
			const path = "/functions/v1/" + functionName;
			const result = await this._client._fetch(path, {
				method: "POST",
				body: options.body ? JSON.stringify(options.body) : undefined,
				headers: options.headers || {},
			});
			return { data: result, error: null };
		}
	};
}
`
}

// wrapToolCode wraps user tool code with the MCP runtime bridge.
func (e *Executor) wrapToolCode(code string, args map[string]any, execCtx map[string]any) string {
	argsJSON, _ := json.Marshal(args)
	ctxJSON, _ := json.Marshal(execCtx)

	return fmt.Sprintf(`
// MCP Tool Runtime Bridge
%s

const __MCP_ARGS__ = %s;
const __MCP_CONTEXT__ = %s;
const __FLUXBASE_URL__ = Deno.env.get("FLUXBASE_PUBLIC_URL") || "";
const __FLUXBASE_USER_TOKEN__ = Deno.env.get("FLUXBASE_USER_TOKEN") || "";
const __FLUXBASE_SERVICE_TOKEN__ = Deno.env.get("FLUXBASE_SERVICE_TOKEN") || "";

// Create Fluxbase SDK clients
// fluxbase: User-scoped client that respects RLS policies
const fluxbase = new FluxbaseClient(__FLUXBASE_URL__, __FLUXBASE_USER_TOKEN__);

// fluxbaseService: Service-scoped client that bypasses RLS (use with caution!)
const fluxbaseService = new FluxbaseClient(__FLUXBASE_URL__, __FLUXBASE_SERVICE_TOKEN__);

// Tool utilities object - metadata and helpers (same pattern as edge functions)
const toolUtils = {
	...__MCP_CONTEXT__,

	// Secrets accessor (requires allow_env permission)
	secrets: {
		get: (name) => Deno.env.get("FLUXBASE_SECRET_" + name.toUpperCase()),
	},

	// Environment info
	env: {
		get: (name) => Deno.env.get(name),
	},

	// AI capabilities - allows tools to use AI completions and embeddings
	ai: {
		// Chat completion with an AI provider
		// Usage: const response = await utils.ai.chat({ messages: [...], provider: "openai", model: "gpt-4" });
		async chat(options) {
			const url = __FLUXBASE_URL__ + "/api/v1/internal/ai/chat";
			const body = {
				messages: options.messages || [],
				provider: options.provider,
				model: options.model,
				max_tokens: options.maxTokens || options.max_tokens,
				temperature: options.temperature,
			};
			const response = await fetch(url, {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					"Authorization": "Bearer " + __FLUXBASE_SERVICE_TOKEN__,
				},
				body: JSON.stringify(body),
			});
			if (!response.ok) {
				const errorText = await response.text();
				throw new Error("AI chat failed: " + response.status + " " + errorText);
			}
			return response.json();
		},

		// Generate embeddings for text
		// Usage: const { embedding } = await utils.ai.embed({ text: "Hello world" });
		async embed(options) {
			const url = __FLUXBASE_URL__ + "/api/v1/internal/ai/embed";
			const body = {
				text: options.text,
				provider: options.provider,
			};
			const response = await fetch(url, {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					"Authorization": "Bearer " + __FLUXBASE_SERVICE_TOKEN__,
				},
				body: JSON.stringify(body),
			});
			if (!response.ok) {
				const errorText = await response.text();
				throw new Error("AI embed failed: " + response.status + " " + errorText);
			}
			return response.json();
		},

		// List available AI providers
		// Usage: const { providers, default: defaultProvider } = await utils.ai.listProviders();
		async listProviders() {
			const url = __FLUXBASE_URL__ + "/api/v1/internal/ai/providers";
			const response = await fetch(url, {
				method: "GET",
				headers: {
					"Authorization": "Bearer " + __FLUXBASE_SERVICE_TOKEN__,
				},
			});
			if (!response.ok) {
				const errorText = await response.text();
				throw new Error("AI listProviders failed: " + response.status + " " + errorText);
			}
			return response.json();
		},
	},
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

		// Call with same signature as edge functions: handler(args, fluxbase, fluxbaseService, utils)
		const result = await handlerFn(__MCP_ARGS__, fluxbase, fluxbaseService, toolUtils);

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
`, fluxbaseSDKCode(), string(argsJSON), string(ctxJSON), code)
}

// wrapResourceCode wraps user resource code with the MCP runtime bridge.
func (e *Executor) wrapResourceCode(code string, params map[string]string, execCtx map[string]any) string {
	paramsJSON, _ := json.Marshal(params)
	ctxJSON, _ := json.Marshal(execCtx)

	return fmt.Sprintf(`
// MCP Resource Runtime Bridge
%s

const __MCP_PARAMS__ = %s;
const __MCP_CONTEXT__ = %s;
const __FLUXBASE_URL__ = Deno.env.get("FLUXBASE_PUBLIC_URL") || "";
const __FLUXBASE_USER_TOKEN__ = Deno.env.get("FLUXBASE_USER_TOKEN") || "";
const __FLUXBASE_SERVICE_TOKEN__ = Deno.env.get("FLUXBASE_SERVICE_TOKEN") || "";

// Create Fluxbase SDK clients
// fluxbase: User-scoped client that respects RLS policies
const fluxbase = new FluxbaseClient(__FLUXBASE_URL__, __FLUXBASE_USER_TOKEN__);

// fluxbaseService: Service-scoped client that bypasses RLS (use with caution!)
const fluxbaseService = new FluxbaseClient(__FLUXBASE_URL__, __FLUXBASE_SERVICE_TOKEN__);

// Resource utilities object - metadata and helpers
const resourceUtils = {
	...__MCP_CONTEXT__,
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

		// Call with same signature as edge functions: handler(params, fluxbase, fluxbaseService, utils)
		const result = await handlerFn(__MCP_PARAMS__, fluxbase, fluxbaseService, resourceUtils);

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
`, fluxbaseSDKCode(), string(paramsJSON), string(ctxJSON), code)
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
