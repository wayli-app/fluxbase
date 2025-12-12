package rpc

import (
	"encoding/json"
	"time"
)

// ExecutionStatus represents the status of an RPC execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
	StatusTimeout   ExecutionStatus = "timeout"
)

// Procedure represents an RPC procedure definition
type Procedure struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Description  string `json:"description,omitempty"`
	SQLQuery     string `json:"sql_query"`
	OriginalCode string `json:"original_code,omitempty"`

	// Parsed from annotations (nullable for schemaless)
	InputSchema  json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`

	// Access control from annotations
	AllowedTables  []string `json:"allowed_tables"`
	AllowedSchemas []string `json:"allowed_schemas"`

	// Execution config
	MaxExecutionTimeSeconds int     `json:"max_execution_time_seconds"`
	RequireRole             *string `json:"require_role,omitempty"`
	IsPublic                bool    `json:"is_public"`

	// Runtime config
	Enabled   bool      `json:"enabled"`
	Version   int       `json:"version"`
	Source    string    `json:"source"` // "filesystem", "api", or "sdk"
	CreatedBy *string   `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProcedureSummary is a lightweight version for listings
type ProcedureSummary struct {
	ID                      string    `json:"id"`
	Name                    string    `json:"name"`
	Namespace               string    `json:"namespace"`
	Description             string    `json:"description,omitempty"`
	AllowedTables           []string  `json:"allowed_tables"`
	AllowedSchemas          []string  `json:"allowed_schemas"`
	MaxExecutionTimeSeconds int       `json:"max_execution_time_seconds"`
	RequireRole             *string   `json:"require_role,omitempty"`
	IsPublic                bool      `json:"is_public"`
	Enabled                 bool      `json:"enabled"`
	Version                 int       `json:"version"`
	Source                  string    `json:"source"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// ToSummary converts a Procedure to a ProcedureSummary
func (p *Procedure) ToSummary() ProcedureSummary {
	return ProcedureSummary{
		ID:                      p.ID,
		Name:                    p.Name,
		Namespace:               p.Namespace,
		Description:             p.Description,
		AllowedTables:           p.AllowedTables,
		AllowedSchemas:          p.AllowedSchemas,
		MaxExecutionTimeSeconds: p.MaxExecutionTimeSeconds,
		RequireRole:             p.RequireRole,
		IsPublic:                p.IsPublic,
		Enabled:                 p.Enabled,
		Version:                 p.Version,
		Source:                  p.Source,
		CreatedAt:               p.CreatedAt,
		UpdatedAt:               p.UpdatedAt,
	}
}

// Execution represents an RPC execution instance
type Execution struct {
	ID            string          `json:"id"`
	ProcedureID   *string         `json:"procedure_id,omitempty"`
	ProcedureName string          `json:"procedure_name"`
	Namespace     string          `json:"namespace"`
	Status        ExecutionStatus `json:"status"`

	// Input/Output
	InputParams  json.RawMessage `json:"input_params,omitempty"`
	Result       json.RawMessage `json:"result,omitempty"`
	ErrorMessage *string         `json:"error_message,omitempty"`
	RowsReturned *int            `json:"rows_returned,omitempty"`

	// Performance
	DurationMs *int `json:"duration_ms,omitempty"`

	// User context
	UserID    *string `json:"user_id,omitempty"`
	UserRole  *string `json:"user_role,omitempty"`
	UserEmail *string `json:"user_email,omitempty"`

	// Execution mode
	IsAsync bool `json:"is_async"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// ExecutionLog represents a single log line from an execution
type ExecutionLog struct {
	ID          int64     `json:"id"`
	ExecutionID string    `json:"execution_id"`
	LineNumber  int       `json:"line_number"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	CreatedAt   time.Time `json:"created_at"`
}

// CallerContext represents the context of the RPC caller
type CallerContext struct {
	UserID   string                 `json:"user_id"`
	Role     string                 `json:"role"`
	Email    string                 `json:"email,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// InvokeRequest represents a request to invoke an RPC procedure
type InvokeRequest struct {
	Params map[string]interface{} `json:"params,omitempty"`
	Async  bool                   `json:"async,omitempty"`
}

// InvokeResponse represents the response from an RPC invocation
type InvokeResponse struct {
	ExecutionID  string          `json:"execution_id"`
	Status       ExecutionStatus `json:"status"`
	Result       json.RawMessage `json:"result,omitempty"`
	RowsReturned *int            `json:"rows_returned,omitempty"`
	DurationMs   *int            `json:"duration_ms,omitempty"`
	Error        *string         `json:"error,omitempty"`
}

// Annotations represents the parsed annotations from an RPC SQL file
type Annotations struct {
	Name             string            `json:"name,omitempty"`
	Description      string            `json:"description,omitempty"`
	InputSchema      map[string]string `json:"input,omitempty"`
	OutputSchema     map[string]string `json:"output,omitempty"`
	AllowedTables    []string          `json:"allowed_tables,omitempty"`
	AllowedSchemas   []string          `json:"allowed_schemas,omitempty"`
	MaxExecutionTime time.Duration     `json:"max_execution_time,omitempty"`
	RequireRole      string            `json:"require_role,omitempty"`
	IsPublic         bool              `json:"is_public,omitempty"`
	Version          int               `json:"version,omitempty"`
}

// ProcedureSpec is used for syncing procedures from filesystem or SDK
type ProcedureSpec struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled,omitempty"`
}

// SyncRequest represents a request to sync RPC procedures
type SyncRequest struct {
	Namespace  string          `json:"namespace,omitempty"`
	Procedures []ProcedureSpec `json:"procedures,omitempty"`
	Options    SyncOptions     `json:"options,omitempty"`
}

// SyncOptions contains options for syncing procedures
type SyncOptions struct {
	DeleteMissing bool `json:"delete_missing,omitempty"`
	DryRun        bool `json:"dry_run,omitempty"`
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	Message   string      `json:"message"`
	Namespace string      `json:"namespace"`
	Summary   SyncSummary `json:"summary"`
	Details   SyncDetails `json:"details"`
	Errors    []SyncError `json:"errors,omitempty"`
	DryRun    bool        `json:"dry_run"`
}

// SyncSummary contains counts for a sync operation
type SyncSummary struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Deleted   int `json:"deleted"`
	Unchanged int `json:"unchanged"`
	Errors    int `json:"errors"`
}

// SyncDetails contains lists of affected procedure names
type SyncDetails struct {
	Created   []string `json:"created"`
	Updated   []string `json:"updated"`
	Deleted   []string `json:"deleted"`
	Unchanged []string `json:"unchanged"`
}

// SyncError represents an error during sync
type SyncError struct {
	Procedure string `json:"procedure"`
	Error     string `json:"error"`
}

// ListExecutionsOptions represents options for listing executions
type ListExecutionsOptions struct {
	Namespace     string
	ProcedureName string
	Status        ExecutionStatus
	UserID        string
	Limit         int
	Offset        int
}

// DefaultAnnotations returns the default configuration for annotations
func DefaultAnnotations() *Annotations {
	return &Annotations{
		AllowedSchemas:   []string{"public"},
		AllowedTables:    []string{},
		MaxExecutionTime: 30 * time.Second,
		IsPublic:         false,
		Version:          1,
	}
}
