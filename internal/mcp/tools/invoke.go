package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/functions"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/fluxbase-eu/fluxbase/internal/rpc"
	"github.com/fluxbase-eu/fluxbase/internal/runtime"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// InvokeFunctionTool implements the invoke_function MCP tool
type InvokeFunctionTool struct {
	storage      *functions.Storage
	runtime      *runtime.DenoRuntime
	publicURL    string
	functionsDir string
}

// NewInvokeFunctionTool creates a new invoke_function tool
func NewInvokeFunctionTool(db *database.Connection, denoRuntime *runtime.DenoRuntime, publicURL, functionsDir string) *InvokeFunctionTool {
	return &InvokeFunctionTool{
		storage:      functions.NewStorage(db),
		runtime:      denoRuntime,
		publicURL:    publicURL,
		functionsDir: functionsDir,
	}
}

func (t *InvokeFunctionTool) Name() string {
	return "invoke_function"
}

func (t *InvokeFunctionTool) Description() string {
	return "Invoke an edge function (JavaScript/TypeScript) by name. Returns the function's response."
}

func (t *InvokeFunctionTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The name of the edge function to invoke",
			},
			"namespace": map[string]any{
				"type":        "string",
				"description": "Optional namespace for the function",
			},
			"body": map[string]any{
				"type":        "object",
				"description": "Request body to pass to the function (will be JSON-encoded)",
			},
			"method": map[string]any{
				"type":        "string",
				"description": "HTTP method to simulate (default: POST)",
				"enum":        []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
				"default":     "POST",
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "Additional headers to pass to the function",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
		},
		"required": []string{"name"},
	}
}

func (t *InvokeFunctionTool) RequiredScopes() []string {
	return []string{mcp.ScopeExecuteFunctions}
}

func (t *InvokeFunctionTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	// Parse arguments
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("function name is required")
	}

	namespace := ""
	if ns, ok := args["namespace"].(string); ok {
		namespace = ns
	}

	method := "POST"
	if m, ok := args["method"].(string); ok {
		method = m
	}

	// Get function
	var fn *functions.EdgeFunction
	var err error
	if namespace != "" {
		fn, err = t.storage.GetFunctionByNamespace(ctx, name, namespace)
	} else {
		fn, err = t.storage.GetFunction(ctx, name)
	}
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Function not found: %s", name))},
			IsError: true,
		}, nil
	}

	// Check if enabled
	if !fn.Enabled {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Function is disabled")},
			IsError: true,
		}, nil
	}

	// Build request body
	var bodyStr string
	if body, ok := args["body"].(map[string]any); ok {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize body: %w", err)
		}
		bodyStr = string(bodyBytes)
	}

	// Build headers
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	if headerMap, ok := args["headers"].(map[string]any); ok {
		for k, v := range headerMap {
			if vs, ok := v.(string); ok {
				headers[k] = vs
			}
		}
	}

	// Generate execution ID
	executionID := uuid.New()

	// Build execution request
	req := runtime.ExecutionRequest{
		ID:        executionID,
		Name:      fn.Name,
		Namespace: fn.Namespace,
		Method:    method,
		URL:       t.publicURL + "/functions/v1/" + fn.Name + "/invoke",
		BaseURL:   t.publicURL,
		Headers:   headers,
		Body:      bodyStr,
		Params:    make(map[string]string),
	}

	// Add user context from MCP auth
	if authCtx.UserID != nil {
		req.UserID = *authCtx.UserID
	}
	if authCtx.UserEmail != "" {
		req.UserEmail = authCtx.UserEmail
	}
	req.UserRole = authCtx.UserRole

	log.Debug().
		Str("function", fn.Name).
		Str("namespace", fn.Namespace).
		Str("method", method).
		Str("execution_id", executionID.String()).
		Msg("MCP: Invoking function")

	// Set up default permissions for MCP function invocation
	perms := runtime.DefaultPermissions()

	// Execute function with default timeout and no secrets for MCP calls
	resp, err := t.runtime.Execute(ctx, fn.Code, req, perms, nil, nil, nil)
	if err != nil {
		log.Error().Err(err).Str("function", fn.Name).Msg("MCP: Function execution failed")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Function execution failed: %v", err))},
			IsError: true,
		}, nil
	}

	// Build result
	result := map[string]any{
		"status":       resp.Status,
		"headers":      resp.Headers,
		"body":         resp.Body,
		"execution_id": executionID.String(),
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to serialize result: %v", err))},
			IsError: true,
		}, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}

// InvokeRPCTool implements the invoke_rpc MCP tool
type InvokeRPCTool struct {
	executor *rpc.Executor
	storage  *rpc.Storage
}

// NewInvokeRPCTool creates a new invoke_rpc tool
func NewInvokeRPCTool(executor *rpc.Executor, storage *rpc.Storage) *InvokeRPCTool {
	return &InvokeRPCTool{
		executor: executor,
		storage:  storage,
	}
}

func (t *InvokeRPCTool) Name() string {
	return "invoke_rpc"
}

func (t *InvokeRPCTool) Description() string {
	return "Invoke a database RPC procedure (stored SQL with parameters). Returns the query result."
}

func (t *InvokeRPCTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The name of the RPC procedure to invoke",
			},
			"namespace": map[string]any{
				"type":        "string",
				"description": "Optional namespace for the procedure",
			},
			"params": map[string]any{
				"type":        "object",
				"description": "Parameters to pass to the procedure",
			},
		},
		"required": []string{"name"},
	}
}

func (t *InvokeRPCTool) RequiredScopes() []string {
	return []string{mcp.ScopeExecuteRPC}
}

func (t *InvokeRPCTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	// Parse arguments
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("procedure name is required")
	}

	namespace := ""
	if ns, ok := args["namespace"].(string); ok {
		namespace = ns
	}

	// Get procedure
	var proc *rpc.Procedure
	var err error
	if namespace != "" {
		proc, err = t.storage.GetProcedureByNamespace(ctx, name, namespace)
	} else {
		proc, err = t.storage.GetProcedureByName(ctx, name)
	}
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Procedure not found: %s", name))},
			IsError: true,
		}, nil
	}

	// Check if enabled
	if !proc.Enabled {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Procedure is disabled")},
			IsError: true,
		}, nil
	}

	// Parse params
	params := make(map[string]interface{})
	if p, ok := args["params"].(map[string]any); ok {
		params = p
	}

	// Build execution context
	userID := ""
	if authCtx.UserID != nil {
		userID = *authCtx.UserID
	}

	execCtx := &rpc.ExecuteContext{
		Procedure:            proc,
		Params:               params,
		UserID:               userID,
		UserRole:             authCtx.UserRole,
		UserEmail:            authCtx.UserEmail,
		IsAsync:              false,
		DisableExecutionLogs: true, // Don't create execution logs for MCP calls
	}

	log.Debug().
		Str("procedure", proc.Name).
		Str("namespace", proc.Namespace).
		Interface("params", params).
		Msg("MCP: Invoking RPC procedure")

	// Execute with timeout
	timeout := time.Duration(proc.MaxExecutionTimeSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	execCtx2, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := t.executor.Execute(execCtx2, execCtx)
	if err != nil {
		log.Error().Err(err).Str("procedure", proc.Name).Msg("MCP: RPC execution failed")
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("RPC execution failed: %v", err))},
			IsError: true,
		}, nil
	}

	// Check for execution error
	if result.Status == rpc.StatusFailed || result.Error != nil {
		errorMsg := "Execution failed"
		if result.Error != nil {
			errorMsg = *result.Error
		}
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(errorMsg)},
			IsError: true,
		}, nil
	}

	// Build response
	response := map[string]any{
		"status":       result.Status,
		"execution_id": result.ExecutionID,
	}

	if result.RowsReturned != nil {
		response["rows_returned"] = *result.RowsReturned
	}
	if result.DurationMs != nil {
		response["duration_ms"] = *result.DurationMs
	}

	// Parse and include result data
	if result.Result != nil {
		var data any
		if err := json.Unmarshal(result.Result, &data); err == nil {
			response["data"] = data
		}
	}

	resultJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to serialize result: %v", err))},
			IsError: true,
		}, nil
	}

	return &mcp.ToolResult{
		Content: []mcp.Content{mcp.TextContent(string(resultJSON))},
	}, nil
}
