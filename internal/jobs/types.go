package jobs

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrDuplicateJob is returned when a duplicate job is already pending or running
var ErrDuplicateJob = errors.New("duplicate job already pending or running")

// JobStatus represents the execution status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// WorkerStatus represents the status of a worker
type WorkerStatus string

const (
	WorkerStatusActive   WorkerStatus = "active"
	WorkerStatusDraining WorkerStatus = "draining"
	WorkerStatusStopped  WorkerStatus = "stopped"
)

// JobFunction represents a job function definition (template)
type JobFunction struct {
	ID                     uuid.UUID  `db:"id" json:"id"`
	Name                   string     `db:"name" json:"name"`
	Namespace              string     `db:"namespace" json:"namespace"`
	Description            *string    `db:"description" json:"description,omitempty"`
	Code                   *string    `db:"code" json:"code,omitempty"`
	OriginalCode           *string    `db:"original_code" json:"original_code,omitempty"`
	IsBundled              bool       `db:"is_bundled" json:"is_bundled"`
	BundleError            *string    `db:"bundle_error" json:"bundle_error,omitempty"`
	Enabled                bool       `db:"enabled" json:"enabled"`
	Schedule               *string    `db:"schedule" json:"schedule,omitempty"`
	TimeoutSeconds         int        `db:"timeout_seconds" json:"timeout_seconds"`
	MemoryLimitMB          int        `db:"memory_limit_mb" json:"memory_limit_mb"`
	MaxRetries             int        `db:"max_retries" json:"max_retries"`
	ProgressTimeoutSeconds int        `db:"progress_timeout_seconds" json:"progress_timeout_seconds"`
	AllowNet               bool       `db:"allow_net" json:"allow_net"`
	AllowEnv               bool       `db:"allow_env" json:"allow_env"`
	AllowRead              bool       `db:"allow_read" json:"allow_read"`
	AllowWrite             bool       `db:"allow_write" json:"allow_write"`
	RequireRoles           []string   `db:"require_roles" json:"require_roles,omitempty"` // Required roles: "admin", "authenticated", "anon", or custom roles. User needs ANY of the specified roles.
	DisableExecutionLogs   bool       `db:"disable_execution_logs" json:"disable_execution_logs"`
	Version                int        `db:"version" json:"version"`
	CreatedBy              *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
	Source                 string     `db:"source" json:"source"` // "filesystem" or "api"
	CreatedAt              time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time  `db:"updated_at" json:"updated_at"`
}

// JobFunctionSummary is a lightweight version of JobFunction for list responses (excludes code fields)
type JobFunctionSummary struct {
	ID                     uuid.UUID  `db:"id" json:"id"`
	Name                   string     `db:"name" json:"name"`
	Namespace              string     `db:"namespace" json:"namespace"`
	Description            *string    `db:"description" json:"description,omitempty"`
	IsBundled              bool       `db:"is_bundled" json:"is_bundled"`
	BundleError            *string    `db:"bundle_error" json:"bundle_error,omitempty"`
	Enabled                bool       `db:"enabled" json:"enabled"`
	Schedule               *string    `db:"schedule" json:"schedule,omitempty"`
	TimeoutSeconds         int        `db:"timeout_seconds" json:"timeout_seconds"`
	MemoryLimitMB          int        `db:"memory_limit_mb" json:"memory_limit_mb"`
	MaxRetries             int        `db:"max_retries" json:"max_retries"`
	ProgressTimeoutSeconds int        `db:"progress_timeout_seconds" json:"progress_timeout_seconds"`
	AllowNet               bool       `db:"allow_net" json:"allow_net"`
	AllowEnv               bool       `db:"allow_env" json:"allow_env"`
	AllowRead              bool       `db:"allow_read" json:"allow_read"`
	AllowWrite             bool       `db:"allow_write" json:"allow_write"`
	RequireRoles           []string   `db:"require_roles" json:"require_roles,omitempty"`
	DisableExecutionLogs   bool       `db:"disable_execution_logs" json:"disable_execution_logs"`
	Version                int        `db:"version" json:"version"`
	CreatedBy              *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
	Source                 string     `db:"source" json:"source"` // "filesystem" or "api"
	CreatedAt              time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time  `db:"updated_at" json:"updated_at"`
}

// Job represents a job execution instance
type Job struct {
	ID                     uuid.UUID  `db:"id" json:"id"`
	Namespace              string     `db:"namespace" json:"namespace"`
	JobFunctionID          *uuid.UUID `db:"function_id" json:"job_function_id,omitempty"`
	JobName                string     `db:"job_name" json:"job_name"`
	Status                 JobStatus  `db:"status" json:"status"`
	Payload                *string    `db:"payload" json:"payload,omitempty"`   // JSONB as string
	Result                 *string    `db:"result" json:"result,omitempty"`     // JSONB as string
	Progress               *string    `db:"progress" json:"progress,omitempty"` // JSONB as string
	Priority               int        `db:"priority" json:"priority"`
	MaxDurationSeconds     *int       `db:"max_duration_seconds" json:"max_duration_seconds,omitempty"`
	ProgressTimeoutSeconds *int       `db:"progress_timeout_seconds" json:"progress_timeout_seconds,omitempty"`
	MaxRetries             int        `db:"max_retries" json:"max_retries"`
	RetryCount             int        `db:"retry_count" json:"retry_count"`
	ErrorMessage           *string    `db:"error_message" json:"error_message,omitempty"`
	WorkerID               *uuid.UUID `db:"worker_id" json:"worker_id,omitempty"`
	CreatedBy              *uuid.UUID `db:"created_by" json:"created_by,omitempty"`
	UserRole               *string    `db:"user_role" json:"user_role,omitempty"`   // Role of user who submitted job
	UserEmail              *string    `db:"user_email" json:"user_email,omitempty"` // Email of user who submitted job
	UserName               *string    `db:"user_name" json:"user_name,omitempty"`   // Display name of user who submitted job
	CreatedAt              time.Time  `db:"created_at" json:"created_at"`
	ScheduledAt            *time.Time `db:"scheduled_at" json:"scheduled_at,omitempty"`
	StartedAt              *time.Time `db:"started_at" json:"started_at,omitempty"`
	LastProgressAt         *time.Time `db:"last_progress_at" json:"last_progress_at,omitempty"`
	CompletedAt            *time.Time `db:"completed_at" json:"completed_at,omitempty"`

	// Computed fields (not stored in DB, calculated on-the-fly)
	EstimatedCompletionAt *time.Time `db:"-" json:"estimated_completion_at,omitempty"`
	EstimatedSecondsLeft  *int       `db:"-" json:"estimated_seconds_left,omitempty"`

	// Flattened progress fields for frontend consumption (computed from Progress JSON)
	ProgressPercent *int                   `db:"-" json:"progress_percent,omitempty"`
	ProgressMessage *string                `db:"-" json:"progress_message,omitempty"`
	ProgressData    map[string]interface{} `db:"-" json:"progress_data,omitempty"`

	// DeduplicationKey (optional) - if set, prevents duplicate pending/running jobs with same key
	// Calculated from namespace + job_name + payload hash
	DeduplicationKey *string `db:"-" json:"deduplication_key,omitempty"`
}

// FlattenProgress parses the Progress JSON string and populates the flattened progress fields
func (j *Job) FlattenProgress() {
	if j.Progress == nil || *j.Progress == "" {
		return
	}

	var progress Progress
	if err := json.Unmarshal([]byte(*j.Progress), &progress); err != nil {
		return
	}

	j.ProgressPercent = &progress.Percent
	if progress.Message != "" {
		j.ProgressMessage = &progress.Message
	}
	if len(progress.Data) > 0 {
		j.ProgressData = progress.Data
	}
}

// WorkerRecord represents a worker node record in the database
type WorkerRecord struct {
	ID                uuid.UUID    `db:"id" json:"id"`
	Name              *string      `db:"name" json:"name,omitempty"`
	Hostname          *string      `db:"hostname" json:"hostname,omitempty"`
	Status            WorkerStatus `db:"status" json:"status"`
	MaxConcurrentJobs int          `db:"max_concurrent_jobs" json:"max_concurrent_jobs"`
	CurrentJobCount   int          `db:"current_job_count" json:"current_job_count"`
	LastHeartbeatAt   time.Time    `db:"last_heartbeat_at" json:"last_heartbeat_at"`
	StartedAt         time.Time    `db:"started_at" json:"started_at"`
	Metadata          *string      `db:"metadata" json:"metadata,omitempty"` // JSONB as string
}

// JobFunctionFile represents a supporting file for a multi-file job function
type JobFunctionFile struct {
	ID            uuid.UUID `db:"id" json:"id"`
	JobFunctionID uuid.UUID `db:"function_id" json:"job_function_id"`
	FilePath      string    `db:"file_path" json:"file_path"`
	Content       string    `db:"content" json:"content"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// Progress represents job execution progress
type Progress struct {
	Percent              int                    `json:"percent"`
	Message              string                 `json:"message,omitempty"`
	EstimatedSecondsLeft *int                   `json:"estimated_seconds_left,omitempty"`
	Data                 map[string]interface{} `json:"data,omitempty"`
}

// Note: ExecutionLog is now in the central logging schema (logging.entries)

// CalculateETA computes the estimated completion time for a running job
// using linear extrapolation based on current progress and elapsed time.
// The computed fields EstimatedCompletionAt and EstimatedSecondsLeft are
// only populated for running jobs with progress > 0.
func (j *Job) CalculateETA() {
	// Only calculate for running jobs
	if j.Status != JobStatusRunning || j.StartedAt == nil {
		return
	}

	// Parse progress to get percent
	if j.Progress == nil || *j.Progress == "" {
		return
	}

	var progress Progress
	if err := json.Unmarshal([]byte(*j.Progress), &progress); err != nil {
		return
	}

	// Can't calculate ETA with 0% or 100% progress
	if progress.Percent <= 0 || progress.Percent >= 100 {
		return
	}

	// Calculate ETA using linear extrapolation
	elapsed := time.Since(*j.StartedAt)
	if elapsed.Seconds() <= 0 {
		return
	}

	rate := float64(progress.Percent) / elapsed.Seconds()
	if rate <= 0 {
		return
	}

	remainingPercent := float64(100 - progress.Percent)
	remainingSeconds := int(remainingPercent / rate)

	eta := time.Now().Add(time.Duration(remainingSeconds) * time.Second)
	j.EstimatedCompletionAt = &eta
	j.EstimatedSecondsLeft = &remainingSeconds
}

// JobFilters represents filters for querying jobs
type JobFilters struct {
	Status        *JobStatus
	JobName       *string
	Namespace     *string
	CreatedBy     *uuid.UUID
	WorkerID      *uuid.UUID
	Limit         *int
	Offset        *int
	IncludeResult *bool // Include result field in response (excluded by default for performance)
}

// JobStats represents aggregate statistics about jobs
type JobStats struct {
	TotalJobs          int                `json:"total_jobs"`
	PendingJobs        int                `json:"pending_jobs"`
	RunningJobs        int                `json:"running_jobs"`
	CompletedJobs      int                `json:"completed_jobs"`
	FailedJobs         int                `json:"failed_jobs"`
	CancelledJobs      int                `json:"cancelled_jobs"`
	AvgDurationSeconds float64            `json:"avg_duration_seconds"`
	JobsByStatus       []JobStatusCount   `json:"jobs_by_status"`
	JobsByDay          []JobDayCount      `json:"jobs_by_day"`
	JobsByFunction     []JobFunctionCount `json:"jobs_by_function"`
}

// JobStatusCount represents count by status
type JobStatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// JobDayCount represents count by day
type JobDayCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// JobFunctionCount represents count by function name
type JobFunctionCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Permissions represents execution permissions for a job
type Permissions struct {
	AllowNet      bool
	AllowEnv      bool
	AllowRead     bool
	AllowWrite    bool
	MemoryLimitMB int // V8 heap memory limit in MB (0 = use default)
}
