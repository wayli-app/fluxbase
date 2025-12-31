package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/fluxbase-eu/fluxbase/internal/rpc"
)

// RPCResource provides RPC procedures information
type RPCResource struct {
	storage *rpc.Storage
}

// NewRPCResource creates a new RPC resource
func NewRPCResource(storage *rpc.Storage) *RPCResource {
	return &RPCResource{
		storage: storage,
	}
}

func (r *RPCResource) URI() string {
	return "fluxbase://rpc"
}

func (r *RPCResource) Name() string {
	return "RPC Procedures"
}

func (r *RPCResource) Description() string {
	return "List of available RPC procedures with their input/output schemas"
}

func (r *RPCResource) MimeType() string {
	return "application/json"
}

func (r *RPCResource) RequiredScopes() []string {
	return []string{mcp.ScopeExecuteRPC}
}

func (r *RPCResource) Read(ctx context.Context, authCtx *mcp.AuthContext) ([]mcp.Content, error) {
	if r.storage == nil {
		return nil, fmt.Errorf("RPC storage not available")
	}

	// List all procedures
	procs, err := r.storage.ListProcedures(ctx, 1000, 0, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list procedures: %w", err)
	}

	// Build response
	procList := make([]map[string]any, 0, len(procs))
	for _, proc := range procs {
		procInfo := map[string]any{
			"name":        proc.Name,
			"namespace":   proc.Namespace,
			"enabled":     proc.Enabled,
			"description": proc.Description,
			"created_at":  proc.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}

		// Include input schema if available
		if proc.InputSchema != nil {
			var inputSchema any
			if err := json.Unmarshal(proc.InputSchema, &inputSchema); err == nil {
				procInfo["input_schema"] = inputSchema
			}
		}

		// Include output schema if available
		if proc.OutputSchema != nil {
			var outputSchema any
			if err := json.Unmarshal(proc.OutputSchema, &outputSchema); err == nil {
				procInfo["output_schema"] = outputSchema
			}
		}

		if proc.MaxExecutionTimeSeconds > 0 {
			procInfo["max_execution_time_seconds"] = proc.MaxExecutionTimeSeconds
		}

		procList = append(procList, procInfo)
	}

	result := map[string]any{
		"procedures": procList,
		"count":      len(procList),
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize procedures: %w", err)
	}

	return []mcp.Content{mcp.TextContent(string(data))}, nil
}
