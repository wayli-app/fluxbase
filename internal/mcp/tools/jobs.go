package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/jobs"
	"github.com/fluxbase-eu/fluxbase/internal/mcp"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// SubmitJobTool implements the submit_job MCP tool
type SubmitJobTool struct {
	storage *jobs.Storage
}

// NewSubmitJobTool creates a new submit_job tool
func NewSubmitJobTool(storage *jobs.Storage) *SubmitJobTool {
	return &SubmitJobTool{
		storage: storage,
	}
}

func (t *SubmitJobTool) Name() string {
	return "submit_job"
}

func (t *SubmitJobTool) Description() string {
	return "Submit a background job for async processing. Returns the job ID for tracking."
}

func (t *SubmitJobTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"job_name": map[string]any{
				"type":        "string",
				"description": "The name of the job function to execute",
			},
			"namespace": map[string]any{
				"type":        "string",
				"description": "Optional namespace for the job function",
			},
			"payload": map[string]any{
				"type":        "object",
				"description": "Data to pass to the job function",
			},
			"priority": map[string]any{
				"type":        "integer",
				"description": "Job priority (higher = more urgent, default: 0)",
				"default":     0,
			},
			"scheduled_at": map[string]any{
				"type":        "string",
				"description": "ISO 8601 datetime to schedule job execution (e.g., '2024-01-15T10:00:00Z')",
			},
		},
		"required": []string{"job_name"},
	}
}

func (t *SubmitJobTool) RequiredScopes() []string {
	return []string{mcp.ScopeExecuteJobs}
}

func (t *SubmitJobTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	// Parse arguments
	jobName, ok := args["job_name"].(string)
	if !ok || jobName == "" {
		return nil, fmt.Errorf("job_name is required")
	}

	namespace := ""
	if ns, ok := args["namespace"].(string); ok {
		namespace = ns
	}

	// Get job function
	var fn *jobs.JobFunction
	var err error
	if namespace != "" {
		fn, err = t.storage.GetJobFunctionByNamespace(ctx, jobName, namespace)
	} else {
		fn, err = t.storage.GetJobFunctionByName(ctx, jobName)
	}
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Job function not found: %s", jobName))},
			IsError: true,
		}, nil
	}

	// Check if enabled
	if !fn.Enabled {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent("Job function is disabled")},
			IsError: true,
		}, nil
	}

	// Parse payload
	var payload map[string]any
	if p, ok := args["payload"].(map[string]any); ok {
		payload = p
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize payload: %w", err)
	}

	// Parse priority
	priority := 0
	if p, ok := args["priority"].(float64); ok {
		priority = int(p)
	}

	// Parse scheduled time
	var scheduledAt *time.Time
	if s, ok := args["scheduled_at"].(string); ok && s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("invalid scheduled_at format (expected ISO 8601): %w", err)
		}
		scheduledAt = &t
	}

	// Get user info
	var userID *uuid.UUID
	if authCtx.UserID != nil {
		parsed, err := uuid.Parse(*authCtx.UserID)
		if err == nil {
			userID = &parsed
		}
	}

	userRole := authCtx.UserRole
	userEmail := authCtx.UserEmail

	log.Debug().
		Str("job_name", jobName).
		Str("namespace", namespace).
		Int("priority", priority).
		Interface("scheduled_at", scheduledAt).
		Msg("MCP: Submitting job")

	// Create job
	job := &jobs.Job{
		ID:            uuid.New(),
		JobFunctionID: fn.ID,
		Status:        jobs.StatusPending,
		Priority:      priority,
		Payload:       payloadBytes,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		ScheduledAt:   scheduledAt,
	}

	if userID != nil {
		job.UserID = userID
	}
	if userRole != "" {
		job.UserRole = &userRole
	}
	if userEmail != "" {
		job.UserEmail = &userEmail
	}

	if err := t.storage.CreateJob(ctx, job); err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Failed to create job: %v", err))},
			IsError: true,
		}, nil
	}

	result := map[string]any{
		"job_id":     job.ID.String(),
		"job_name":   jobName,
		"status":     string(job.Status),
		"priority":   priority,
		"created_at": job.CreatedAt.Format(time.RFC3339),
	}
	if scheduledAt != nil {
		result["scheduled_at"] = scheduledAt.Format(time.RFC3339)
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

// GetJobStatusTool implements the get_job_status MCP tool
type GetJobStatusTool struct {
	storage *jobs.Storage
}

// NewGetJobStatusTool creates a new get_job_status tool
func NewGetJobStatusTool(storage *jobs.Storage) *GetJobStatusTool {
	return &GetJobStatusTool{
		storage: storage,
	}
}

func (t *GetJobStatusTool) Name() string {
	return "get_job_status"
}

func (t *GetJobStatusTool) Description() string {
	return "Get the status of a submitted background job."
}

func (t *GetJobStatusTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"job_id": map[string]any{
				"type":        "string",
				"description": "The job ID returned from submit_job",
			},
		},
		"required": []string{"job_id"},
	}
}

func (t *GetJobStatusTool) RequiredScopes() []string {
	return []string{mcp.ScopeExecuteJobs}
}

func (t *GetJobStatusTool) Execute(ctx context.Context, args map[string]any, authCtx *mcp.AuthContext) (*mcp.ToolResult, error) {
	jobIDStr, ok := args["job_id"].(string)
	if !ok || jobIDStr == "" {
		return nil, fmt.Errorf("job_id is required")
	}

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid job_id format: %w", err)
	}

	log.Debug().
		Str("job_id", jobIDStr).
		Msg("MCP: Getting job status")

	job, err := t.storage.GetJob(ctx, jobID)
	if err != nil {
		return &mcp.ToolResult{
			Content: []mcp.Content{mcp.ErrorContent(fmt.Sprintf("Job not found: %s", jobIDStr))},
			IsError: true,
		}, nil
	}

	result := map[string]any{
		"job_id":     job.ID.String(),
		"status":     string(job.Status),
		"priority":   job.Priority,
		"attempts":   job.Attempts,
		"created_at": job.CreatedAt.Format(time.RFC3339),
		"updated_at": job.UpdatedAt.Format(time.RFC3339),
	}

	if job.ScheduledAt != nil {
		result["scheduled_at"] = job.ScheduledAt.Format(time.RFC3339)
	}
	if job.StartedAt != nil {
		result["started_at"] = job.StartedAt.Format(time.RFC3339)
	}
	if job.CompletedAt != nil {
		result["completed_at"] = job.CompletedAt.Format(time.RFC3339)
	}
	if job.Error != nil {
		result["error"] = *job.Error
	}
	if job.Result != nil {
		var resultData any
		if err := json.Unmarshal(job.Result, &resultData); err == nil {
			result["result"] = resultData
		}
	}
	if job.Progress != nil {
		result["progress"] = *job.Progress
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
