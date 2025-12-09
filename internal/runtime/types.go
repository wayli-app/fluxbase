package runtime

import "github.com/google/uuid"

// RuntimeType distinguishes between edge functions and job functions
type RuntimeType int

const (
	// RuntimeTypeFunction is for edge functions (HTTP-triggered, short timeout)
	RuntimeTypeFunction RuntimeType = iota
	// RuntimeTypeJob is for job functions (queue-triggered, long timeout)
	RuntimeTypeJob
)

// String returns the string representation of RuntimeType
func (t RuntimeType) String() string {
	switch t {
	case RuntimeTypeFunction:
		return "function"
	case RuntimeTypeJob:
		return "job"
	default:
		return "unknown"
	}
}

// ExecutionRequest represents a unified execution request for both functions and jobs
type ExecutionRequest struct {
	// Common fields
	ID        uuid.UUID `json:"id"`                  // execution_id or job_id
	Name      string    `json:"name"`                // function_name or job_name
	Namespace string    `json:"namespace,omitempty"` // namespace for multi-tenancy
	UserID    string    `json:"user_id,omitempty"`   // user who triggered the execution
	UserEmail string    `json:"user_email,omitempty"`
	UserRole  string    `json:"user_role,omitempty"`
	BaseURL   string    `json:"base_url,omitempty"` // base URL for constructing absolute URLs in runtime

	// HTTP context (functions)
	Method    string            `json:"method,omitempty"`
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
	Params    map[string]string `json:"params,omitempty"`
	SessionID string            `json:"session_id,omitempty"`

	// Job context (jobs)
	Payload    map[string]interface{} `json:"payload,omitempty"`
	RetryCount int                    `json:"retry_count,omitempty"`
}

// ExecutionResult represents the unified result of an execution
type ExecutionResult struct {
	// Common fields
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	Logs       string `json:"logs"`
	DurationMs int64  `json:"duration_ms"`

	// Function response format (HTTP)
	Status  int               `json:"status,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`

	// Job response format
	Result map[string]interface{} `json:"result,omitempty"`
}

// Progress represents a progress update from an execution
type Progress struct {
	Percent              int                    `json:"percent"`
	Message              string                 `json:"message,omitempty"`
	Data                 map[string]interface{} `json:"data,omitempty"`
	EstimatedSecondsLeft *int                   `json:"estimated_seconds_left,omitempty"` // Calculated by worker
}

// Permissions represents Deno security permissions
type Permissions struct {
	AllowNet      bool `json:"allow_net"`
	AllowEnv      bool `json:"allow_env"`
	AllowRead     bool `json:"allow_read"`
	AllowWrite    bool `json:"allow_write"`
	MemoryLimitMB int  `json:"memory_limit_mb,omitempty"` // V8 heap limit in MB
}

// DefaultPermissions returns safe default permissions
func DefaultPermissions() Permissions {
	return Permissions{
		AllowNet:   true,
		AllowEnv:   true,
		AllowRead:  false,
		AllowWrite: false,
	}
}

// DefaultFunctionPermissions returns default permissions for edge functions
func DefaultFunctionPermissions() Permissions {
	return Permissions{
		AllowNet:      true,
		AllowEnv:      true,
		AllowRead:     false,
		AllowWrite:    false,
		MemoryLimitMB: 512,
	}
}

// DefaultJobPermissions returns default permissions for job functions
func DefaultJobPermissions() Permissions {
	return Permissions{
		AllowNet:      true,
		AllowEnv:      true,
		AllowRead:     false,
		AllowWrite:    false,
		MemoryLimitMB: 512,
	}
}
