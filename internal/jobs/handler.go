package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

/*
JOB PROGRESS REPORTING API DOCUMENTATION

When writing job functions, use the global `Fluxbase` API to report progress:

1. REPORT PROGRESS

   Fluxbase.reportProgress(percent, message, data)

   Parameters:
   - percent (number): Progress percentage from 0 to 100
   - message (string): Human-readable status message
   - data (object, optional): Additional structured data

   Examples:

   // Simple percentage update
   Fluxbase.reportProgress(25, "Processing batch 1 of 4");

   // With absolute progress
   Fluxbase.reportProgress(50, "Processed 500 of 1000 records", {
     processed: 500,
     total: 1000,
     errors: 3
   });

   // Step-based progress
   Fluxbase.reportProgress(33, "Step 1: Validating data", {
     step: "validation",
     itemsValidated: 150
   });

2. GET JOB PAYLOAD

   const payload = Fluxbase.getJobPayload()

   Returns the job's input payload as an object.

3. CHECK CANCELLATION

   if (Fluxbase.checkCancellation()) {
     // Job was cancelled by user, clean up and exit
     return { success: false, error: "Job cancelled by user" };
   }

4. GET JOB CONTEXT

   const context = Fluxbase.getJobContext()

   Returns full job context including:
   - job_id: UUID of the job
   - job_name: Name of the job function
   - namespace: Job namespace
   - retry_count: Current retry attempt
   - user_id: User who submitted the job (if any)
   - payload: Job input data

BEST PRACTICES:

1. Report progress frequently (every 5-10% or after significant steps)
2. Use descriptive messages that help users understand what's happening
3. Include absolute progress in data field when applicable (e.g., "processed: 50, total: 100")
4. Check for cancellation in long-running loops
5. Return structured results:

   // Success
   return {
     success: true,
     result: { recordsProcessed: 1000, errors: 2 }
   };

   // Failure
   return {
     success: false,
     error: "Failed to connect to external API",
     result: { partialData: [...] }
   };

EXAMPLE JOB FUNCTION:

```typescript
// @fluxbase:timeout 600
// @fluxbase:max-retries 3
// @fluxbase:progress-timeout 60

export async function handler(request: Request) {
  const { items } = Fluxbase.getJobPayload();
  const total = items.length;
  let processed = 0;
  const results = [];

  Fluxbase.reportProgress(0, "Starting processing");

  for (const item of items) {
    // Check for cancellation
    if (Fluxbase.checkCancellation()) {
      return {
        success: false,
        error: "Job cancelled",
        result: { processed, results }
      };
    }

    // Process item
    const result = await processItem(item);
    results.push(result);
    processed++;

    // Report progress
    const percent = Math.round((processed / total) * 100);
    Fluxbase.reportProgress(percent, `Processed ${processed} of ${total}`, {
      processed,
      total,
      lastItem: item.id
    });
  }

  return {
    success: true,
    result: {
      totalProcessed: processed,
      results
    }
  };
}
```

ANNOTATIONS:

Configure job behavior using @fluxbase: annotations in code comments:

- @fluxbase:timeout 600               // Max duration in seconds (default: 300)
- @fluxbase:memory 512                // Memory limit in MB (default: 256)
- @fluxbase:max-retries 3             // Max retry attempts (default: 0)
- @fluxbase:progress-timeout 60       // Kill job if no progress for N seconds (default: 60)
- @fluxbase:enabled false             // Disable job function (default: true)
- @fluxbase:allow-read true           // Allow filesystem read (default: false)
- @fluxbase:allow-write true          // Allow filesystem write (default: false)
- @fluxbase:allow-net false           // Disallow network access (default: true)
- @fluxbase:allow-env false           // Disallow env var access (default: true)
- @fluxbase:schedule 0 2 * * *        // Cron schedule (optional)
- @fluxbase:schedule-params {"key": "value"}  // Parameters passed when scheduled (optional)

*/

// Handler manages HTTP endpoints for jobs
type Handler struct {
	storage     *Storage
	loader      *Loader
	manager     *Manager
	scheduler   *Scheduler
	config      *config.JobsConfig
	authService *auth.Service
}

// SetScheduler sets the scheduler for the handler
func (h *Handler) SetScheduler(scheduler *Scheduler) {
	h.scheduler = scheduler
}

// roleSatisfiesRequirement checks if the user's role satisfies the required role
// using a hierarchy where: admin > authenticated > anon
func roleSatisfiesRequirement(userRole, requiredRole string) bool {
	// Define role hierarchy levels (higher number = more privileged)
	roleLevel := map[string]int{
		"anon":          0,
		"authenticated": 1,
		"admin":         2,
	}

	userLevel, userOk := roleLevel[userRole]
	requiredLevel, requiredOk := roleLevel[requiredRole]

	// If the required role is not in the hierarchy, require exact match
	if !requiredOk {
		return userRole == requiredRole
	}

	// If user role is not in hierarchy, it's treated as authenticated level
	// (e.g., custom roles like "moderator", "editor" are at authenticated level)
	if !userOk {
		userLevel = roleLevel["authenticated"]
	}

	return userLevel >= requiredLevel
}

// NewHandler creates a new jobs handler
func NewHandler(db *database.Connection, cfg *config.JobsConfig, manager *Manager, authService *auth.Service) (*Handler, error) {
	storage := NewStorage(db)
	loader, err := NewLoader(storage, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create loader: %w", err)
	}

	return &Handler{
		storage:     storage,
		loader:      loader,
		manager:     manager,
		config:      cfg,
		authService: authService,
	}, nil
}

// RegisterRoutes registers all job routes
func (h *Handler) RegisterRoutes(app *fiber.App, authService *auth.Service, apiKeyService *auth.APIKeyService, db *pgxpool.Pool, jwtManager *auth.JWTManager) {
	// Apply authentication middleware
	authMiddleware := middleware.RequireAuthOrServiceKey(authService, apiKeyService, db, jwtManager)

	// Apply feature flag middleware to all jobs routes
	jobs := app.Group("/api/v1/jobs",
		middleware.RequireJobsEnabled(authService.GetSettingsCache()),
	)

	// User endpoints - require authentication, RLS enforced
	jobs.Post("/submit", authMiddleware, h.SubmitJob)
	jobs.Get("/:id", authMiddleware, h.GetJob)
	jobs.Get("/", authMiddleware, h.ListJobs)
	jobs.Post("/:id/cancel", authMiddleware, h.CancelJob)
	jobs.Post("/:id/retry", authMiddleware, h.RetryJob)
}

// RegisterAdminRoutes registers admin-only routes
func (h *Handler) RegisterAdminRoutes(app *fiber.App) {
	admin := app.Group("/api/v1/admin/jobs")

	// Admin endpoints
	admin.Post("/sync", h.SyncJobs)
	admin.Get("/functions", h.ListJobFunctions)
	admin.Get("/functions/:namespace/:name", h.GetJobFunction)
	admin.Put("/functions/:namespace/:name", h.UpdateJobFunction)
	admin.Delete("/functions/:namespace/:name", h.DeleteJobFunction)
	admin.Get("/stats", h.GetJobStats)
	admin.Get("/workers", h.ListWorkers)

	// Queue operations - admin can see and manage all jobs across users
	admin.Get("/queue", h.ListAllJobs)
	admin.Get("/queue/:id/logs", h.GetJobLogs) // More specific routes must come first
	admin.Post("/queue/:id/terminate", h.TerminateJob)
	admin.Post("/queue/:id/cancel", h.CancelJobAdmin)
	admin.Post("/queue/:id/retry", h.RetryJobAdmin)
	admin.Post("/queue/:id/resubmit", h.ResubmitJobAdmin)
	admin.Get("/queue/:id", h.GetJobAdmin) // Less specific route comes last
}

// SubmitJob submits a new job to the queue
func (h *Handler) SubmitJob(c *fiber.Ctx) error {
	var req struct {
		JobName   string                 `json:"job_name"`
		Namespace string                 `json:"namespace"`
		Payload   map[string]interface{} `json:"payload"`
		Priority  *int                   `json:"priority"`
		Scheduled *time.Time             `json:"scheduled_at"`
		// OnBehalfOf allows service_role to submit jobs as a specific user
		OnBehalfOf *struct {
			UserID    string  `json:"user_id"`
			UserEmail *string `json:"user_email"`
			UserRole  *string `json:"user_role"`
		} `json:"on_behalf_of"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validation
	if req.JobName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "job_name is required"})
	}

	// Get user context from locals
	var userID *uuid.UUID
	var userRole, userEmail *string

	// Check if on_behalf_of is being used
	if req.OnBehalfOf != nil {
		// Only service_role can use on_behalf_of
		callerRole := c.Locals("user_role")
		if callerRole == nil || callerRole.(string) != "service_role" {
			return c.Status(403).JSON(fiber.Map{
				"error": "on_behalf_of requires service_role",
			})
		}

		// Parse and validate the target user ID
		parsed, err := uuid.Parse(req.OnBehalfOf.UserID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": "Invalid user_id in on_behalf_of",
			})
		}

		// Verify user exists in auth.users
		var exists bool
		checkQuery := "SELECT EXISTS(SELECT 1 FROM auth.users WHERE id = $1)"
		if err := h.storage.conn.Pool().QueryRow(c.Context(), checkQuery, parsed).Scan(&exists); err != nil || !exists {
			return c.Status(400).JSON(fiber.Map{
				"error": "User not found in on_behalf_of.user_id",
			})
		}

		userID = &parsed
		userEmail = req.OnBehalfOf.UserEmail
		userRole = req.OnBehalfOf.UserRole

		// Default role to "authenticated" if not specified
		if userRole == nil {
			defaultRole := "authenticated"
			userRole = &defaultRole
		}

		log.Info().
			Str("target_user_id", parsed.String()).
			Str("caller", "service_role").
			Msg("Job submitted on behalf of user")
	} else if impersonationToken := c.Get("X-Impersonation-Token"); impersonationToken != "" && h.authService != nil {
		// Check for impersonation token - allows admin to submit jobs as another user
		impersonationClaims, err := h.authService.ValidateToken(impersonationToken)
		if err != nil {
			log.Warn().Err(err).Msg("Invalid impersonation token in job submission")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid impersonation token",
			})
		}

		// Override user context with impersonated user
		parsed, err := uuid.Parse(impersonationClaims.UserID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid user_id in impersonation token",
			})
		}
		userID = &parsed
		userEmail = &impersonationClaims.Email
		userRole = &impersonationClaims.Role

		log.Info().
			Str("target_user_id", parsed.String()).
			Str("impersonated_role", impersonationClaims.Role).
			Msg("Job submitted with impersonation")
	} else {
		// Standard flow: use caller's identity
		if uid := c.Locals("user_id"); uid != nil {
			if uidStr, ok := uid.(string); ok {
				parsed, err := uuid.Parse(uidStr)
				if err == nil {
					// Verify user exists in auth.users before setting created_by
					// Dashboard admins are in dashboard.users, not auth.users
					var exists bool
					checkQuery := "SELECT EXISTS(SELECT 1 FROM auth.users WHERE id = $1)"
					if err := h.storage.conn.Pool().QueryRow(c.Context(), checkQuery, parsed).Scan(&exists); err == nil && exists {
						userID = &parsed
					}
					// If user doesn't exist in auth.users, leave userID as nil
					// Job will be created without created_by (allowed by nullable FK)
				}
			}
		}

		if role := c.Locals("user_role"); role != nil {
			if roleStr, ok := role.(string); ok {
				userRole = &roleStr
			}
		}

		if email := c.Locals("user_email"); email != nil {
			if emailStr, ok := email.(string); ok {
				userEmail = &emailStr
			}
		}
	}

	// Get job function to validate it exists and is enabled
	// If namespace is provided, look up by namespace+name; otherwise find first match by name
	var jobFunction *JobFunction
	var err error
	if req.Namespace != "" {
		jobFunction, err = h.storage.GetJobFunction(c.Context(), req.Namespace, req.JobName)
	} else {
		jobFunction, err = h.storage.GetJobFunctionByName(c.Context(), req.JobName)
	}
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Job function not found",
			"job":   req.JobName,
		})
	}

	if !jobFunction.Enabled {
		return c.Status(403).JSON(fiber.Map{"error": "Job function is disabled"})
	}

	// Check role-based permissions
	if jobFunction.RequireRole != nil && *jobFunction.RequireRole != "" {
		if userRole == nil {
			return c.Status(403).JSON(fiber.Map{
				"error":         "Authentication required",
				"required_role": *jobFunction.RequireRole,
			})
		}

		// Check if user's role satisfies the required role using hierarchy
		// (admin > authenticated > anon)
		if !roleSatisfiesRequirement(*userRole, *jobFunction.RequireRole) {
			return c.Status(403).JSON(fiber.Map{
				"error":         "Insufficient permissions",
				"required_role": *jobFunction.RequireRole,
				"user_role":     *userRole,
			})
		}
	}

	// Serialize payload
	var payloadJSON *string
	if req.Payload != nil {
		payloadBytes, err := json.Marshal(req.Payload)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid payload"})
		}
		payloadStr := string(payloadBytes)
		payloadJSON = &payloadStr
	}

	// Create job
	job := &Job{
		ID:                     uuid.New(),
		Namespace:              jobFunction.Namespace,
		JobFunctionID:          &jobFunction.ID,
		JobName:                req.JobName,
		Status:                 JobStatusPending,
		Payload:                payloadJSON,
		Priority:               valueOr(req.Priority, 0),
		MaxDurationSeconds:     &jobFunction.TimeoutSeconds,
		ProgressTimeoutSeconds: &jobFunction.ProgressTimeoutSeconds,
		MaxRetries:             jobFunction.MaxRetries,
		RetryCount:             0,
		CreatedBy:              userID,
		UserRole:               userRole,
		UserEmail:              userEmail,
		ScheduledAt:            req.Scheduled,
	}

	if err := h.storage.CreateJob(c.Context(), job); err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("job_name", req.JobName).
			Str("request_id", reqID).
			Msg("Failed to create job")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to submit job",
			"request_id": reqID,
		})
	}

	log.Info().
		Str("job_id", job.ID.String()).
		Str("job_name", req.JobName).
		Str("user_id", toString(userID)).
		Msg("Job submitted")

	return c.Status(201).JSON(job)
}

// GetJob gets a job by ID (RLS enforced)
func (h *Handler) GetJob(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	job, err := h.storage.GetJob(c.Context(), jobID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
	}

	// Calculate ETA and flatten progress for running jobs
	job.CalculateETA()
	job.FlattenProgress()

	return c.JSON(job)
}

// GetJobAdmin gets a job by ID (admin access, bypasses RLS)
func (h *Handler) GetJobAdmin(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	job, err := h.storage.GetJobByIDAdmin(c.Context(), jobID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
	}

	// Calculate ETA and flatten progress for running jobs
	job.CalculateETA()
	job.FlattenProgress()

	return c.JSON(job)
}

// GetJobLogs gets execution logs for a job (admin access)
func (h *Handler) GetJobLogs(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	log.Debug().
		Str("job_id", jobID.String()).
		Msg("GetJobLogs called")

	// Parse optional after parameter for pagination
	var afterLine *int
	if after := c.QueryInt("after", -1); after >= 0 {
		afterLine = &after
	}

	logs, err := h.storage.GetExecutionLogs(c.Context(), jobID, afterLine)
	if err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("job_id", jobID.String()).
			Str("request_id", reqID).
			Msg("Failed to get job logs")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to get job logs",
			"request_id": reqID,
		})
	}

	log.Debug().
		Str("job_id", jobID.String()).
		Int("log_count", len(logs)).
		Msg("Returning execution logs")

	return c.JSON(fiber.Map{
		"logs": logs,
	})
}

// ListJobs lists jobs for the authenticated user (RLS enforced)
func (h *Handler) ListJobs(c *fiber.Ctx) error {
	// Parse filters
	filters := &JobFilters{}

	if status := c.Query("status"); status != "" {
		s := JobStatus(status)
		filters.Status = &s
	}

	if jobName := c.Query("job_name"); jobName != "" {
		filters.JobName = &jobName
	}

	if namespace := c.Query("namespace"); namespace != "" {
		filters.Namespace = &namespace
	}

	if c.Query("include_result") == "true" {
		includeResult := true
		filters.IncludeResult = &includeResult
	}

	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	filters.Limit = &limit
	filters.Offset = &offset

	jobs, err := h.storage.ListJobs(c.Context(), filters)
	if err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("request_id", reqID).
			Msg("Failed to list jobs")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to list jobs",
			"request_id": reqID,
		})
	}

	// Calculate ETA for running jobs
	for i := range jobs {
		jobs[i].CalculateETA()
	}

	return c.JSON(fiber.Map{
		"jobs":   jobs,
		"limit":  limit,
		"offset": offset,
	})
}

// CancelJob cancels a pending or running job (RLS enforced)
func (h *Handler) CancelJob(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	// Get job to check status
	job, err := h.storage.GetJob(c.Context(), jobID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
	}

	// Can only cancel pending or running jobs
	if job.Status != JobStatusPending && job.Status != JobStatusRunning {
		return c.Status(400).JSON(fiber.Map{
			"error":  "Job cannot be cancelled",
			"status": job.Status,
		})
	}

	if err := h.storage.CancelJob(c.Context(), jobID); err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("job_id", jobID.String()).
			Str("request_id", reqID).
			Msg("Failed to cancel job")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to cancel job",
			"request_id": reqID,
		})
	}

	// Signal the worker to kill the job process immediately
	if h.manager != nil {
		h.manager.CancelJob(jobID)
	}

	log.Info().Str("job_id", jobID.String()).Msg("Job cancelled by user")

	return c.JSON(fiber.Map{"message": "Job cancelled"})
}

// RetryJob retries a failed job (RLS enforced)
func (h *Handler) RetryJob(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	// Get job to check status
	job, err := h.storage.GetJob(c.Context(), jobID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
	}

	// Can only retry failed jobs
	if job.Status != JobStatusFailed {
		return c.Status(400).JSON(fiber.Map{
			"error":  "Only failed jobs can be retried",
			"status": job.Status,
		})
	}

	if err := h.storage.RequeueJob(c.Context(), jobID); err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("job_id", jobID.String()).
			Str("request_id", reqID).
			Msg("Failed to retry job")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to retry job",
			"request_id": reqID,
		})
	}

	log.Info().Str("job_id", jobID.String()).Msg("Job requeued for retry")

	return c.JSON(fiber.Map{"message": "Job requeued for retry"})
}

// CancelJobAdmin cancels a pending or running job (admin access, bypasses RLS)
func (h *Handler) CancelJobAdmin(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	// Get job to check status (admin access)
	job, err := h.storage.GetJobByIDAdmin(c.Context(), jobID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
	}

	// Can only cancel pending or running jobs
	if job.Status != JobStatusPending && job.Status != JobStatusRunning {
		return c.Status(400).JSON(fiber.Map{
			"error":  "Job cannot be cancelled",
			"status": job.Status,
		})
	}

	if err := h.storage.CancelJob(c.Context(), jobID); err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("job_id", jobID.String()).
			Str("request_id", reqID).
			Msg("Failed to cancel job")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to cancel job",
			"request_id": reqID,
		})
	}

	// Signal the worker to kill the job process immediately
	if h.manager != nil {
		h.manager.CancelJob(jobID)
	}

	log.Info().Str("job_id", jobID.String()).Msg("Job cancelled by admin")

	return c.JSON(fiber.Map{"message": "Job cancelled"})
}

// RetryJobAdmin retries a failed job (admin access, bypasses RLS)
func (h *Handler) RetryJobAdmin(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	// Get job to check status (admin access)
	job, err := h.storage.GetJobByIDAdmin(c.Context(), jobID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
	}

	// Can only retry failed jobs
	if job.Status != JobStatusFailed {
		return c.Status(400).JSON(fiber.Map{
			"error":  "Only failed jobs can be retried",
			"status": job.Status,
		})
	}

	if err := h.storage.RequeueJob(c.Context(), jobID); err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("job_id", jobID.String()).
			Str("request_id", reqID).
			Msg("Failed to retry job")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to retry job",
			"request_id": reqID,
		})
	}

	log.Info().Str("job_id", jobID.String()).Msg("Job requeued for retry by admin")

	return c.JSON(fiber.Map{"message": "Job requeued for retry"})
}

// ResubmitJobAdmin creates a new job based on an existing job (admin access)
// Unlike retry, this works for any job status and creates a fresh job
func (h *Handler) ResubmitJobAdmin(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	// Create new job based on the original
	newJob, err := h.storage.ResubmitJob(c.Context(), jobID)
	if err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("job_id", jobID.String()).
			Str("request_id", reqID).
			Msg("Failed to resubmit job")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to resubmit job",
			"request_id": reqID,
		})
	}

	log.Info().
		Str("original_job_id", jobID.String()).
		Str("new_job_id", newJob.ID.String()).
		Msg("Job resubmitted by admin")

	return c.Status(201).JSON(newJob)
}

// SyncJobs syncs job functions to a namespace
// Accepts a batch of job functions with optional delete_missing to remove stale jobs
// Admin-only endpoint - requires authentication and admin role
func (h *Handler) SyncJobs(c *fiber.Ctx) error {
	var req struct {
		Namespace string `json:"namespace"`
		Jobs      []struct {
			Name                   string  `json:"name"`
			Description            *string `json:"description"`
			Code                   string  `json:"code"`
			OriginalCode           *string `json:"original_code"`
			IsPreBundled           bool    `json:"is_pre_bundled"`
			Enabled                *bool   `json:"enabled"`
			Schedule               *string `json:"schedule"`
			TimeoutSeconds         *int    `json:"timeout_seconds"`
			MemoryLimitMB          *int    `json:"memory_limit_mb"`
			MaxRetries             *int    `json:"max_retries"`
			ProgressTimeoutSeconds *int    `json:"progress_timeout_seconds"`
			AllowNet               *bool   `json:"allow_net"`
			AllowEnv               *bool   `json:"allow_env"`
			AllowRead              *bool   `json:"allow_read"`
			AllowWrite             *bool   `json:"allow_write"`
			RequireRole            *string `json:"require_role"`
		} `json:"jobs"`
		Options struct {
			DeleteMissing bool `json:"delete_missing"`
			DryRun        bool `json:"dry_run"`
		} `json:"options"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Default namespace to "default" if not specified
	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	ctx := c.Context()

	// Get user ID from context (if authenticated)
	var createdBy *uuid.UUID
	if userID := c.Locals("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			parsed, err := uuid.Parse(uid)
			if err == nil {
				createdBy = &parsed
			}
		}
	}

	// If no jobs provided, fall back to filesystem sync
	if len(req.Jobs) == 0 {
		if err := h.loader.LoadFromFilesystem(ctx, namespace); err != nil {
			reqID := getRequestID(c)
			log.Error().
				Err(err).
				Str("namespace", namespace).
				Str("request_id", reqID).
				Msg("Failed to sync jobs from filesystem")

			return c.Status(500).JSON(fiber.Map{
				"error":      "Failed to sync jobs from filesystem",
				"details":    err.Error(),
				"request_id": reqID,
			})
		}

		// Reschedule jobs after filesystem sync
		h.rescheduleJobsFromNamespace(ctx, namespace)

		return c.JSON(fiber.Map{
			"message":   "Jobs synced from filesystem",
			"namespace": namespace,
			"summary": fiber.Map{
				"created":   0,
				"updated":   0,
				"deleted":   0,
				"unchanged": 0,
				"errors":    0,
			},
			"details": fiber.Map{
				"created":   []string{},
				"updated":   []string{},
				"deleted":   []string{},
				"unchanged": []string{},
			},
			"errors":  []fiber.Map{},
			"dry_run": false,
		})
	}

	// Get all existing job functions in this namespace
	existingFunctions, err := h.storage.ListJobFunctions(ctx, namespace)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to list existing job functions in namespace",
		})
	}

	// Build set of existing function names
	existingNames := make(map[string]*JobFunctionSummary)
	for i := range existingFunctions {
		existingNames[existingFunctions[i].Name] = existingFunctions[i]
	}

	// Build set of payload function names
	payloadNames := make(map[string]bool)
	for _, spec := range req.Jobs {
		payloadNames[spec.Name] = true
	}

	// Determine operations
	toCreate := []string{}
	toUpdate := []string{}
	toDelete := []string{}

	for _, spec := range req.Jobs {
		if _, exists := existingNames[spec.Name]; exists {
			toUpdate = append(toUpdate, spec.Name)
		} else {
			toCreate = append(toCreate, spec.Name)
		}
	}

	if req.Options.DeleteMissing {
		for name := range existingNames {
			if !payloadNames[name] {
				toDelete = append(toDelete, name)
			}
		}
	}

	// Track results
	created := []string{}
	updated := []string{}
	deleted := []string{}
	unchanged := []string{}
	errorList := []fiber.Map{}

	// If dry run, return what would be done without making changes
	if req.Options.DryRun {
		return c.JSON(fiber.Map{
			"message":   "Dry run - no changes made",
			"namespace": namespace,
			"summary": fiber.Map{
				"created":   len(toCreate),
				"updated":   len(toUpdate),
				"deleted":   len(toDelete),
				"unchanged": 0,
				"errors":    0,
			},
			"details": fiber.Map{
				"created":   toCreate,
				"updated":   toUpdate,
				"deleted":   toDelete,
				"unchanged": []string{},
			},
			"errors":  []fiber.Map{},
			"dry_run": true,
		})
	}

	// Process each job function
	for _, spec := range req.Jobs {
		code := spec.Code
		originalCode := spec.Code
		isBundled := false
		var bundleError *string

		// If original_code provided, use it
		if spec.OriginalCode != nil {
			originalCode = *spec.OriginalCode
		}

		// Bundle if not pre-bundled
		if !spec.IsPreBundled {
			bundledCode, bundleErr := h.loader.BundleCode(ctx, spec.Code)
			if bundleErr != nil {
				errMsg := bundleErr.Error()
				bundleError = &errMsg
				// Continue with unbundled code
			} else {
				code = bundledCode
				isBundled = true
			}
		} else {
			isBundled = true
		}

		// Parse annotations from original code
		annotations := h.loader.ParseAnnotations(originalCode)

		// Create or update job function
		if existing, exists := existingNames[spec.Name]; exists {
			// Update existing function - build JobFunction with updated values
			updatedFn := &JobFunction{
				ID:                     existing.ID,
				Name:                   existing.Name,
				Namespace:              existing.Namespace,
				Code:                   &code,
				OriginalCode:           &originalCode,
				IsBundled:              isBundled,
				BundleError:            bundleError,
				Description:            existing.Description,
				Enabled:                existing.Enabled,
				Schedule:               existing.Schedule,
				TimeoutSeconds:         existing.TimeoutSeconds,
				MemoryLimitMB:          existing.MemoryLimitMB,
				MaxRetries:             existing.MaxRetries,
				ProgressTimeoutSeconds: existing.ProgressTimeoutSeconds,
				AllowNet:               existing.AllowNet,
				AllowEnv:               existing.AllowEnv,
				AllowRead:              existing.AllowRead,
				AllowWrite:             existing.AllowWrite,
				RequireRole:            existing.RequireRole,
				Source:                 existing.Source, // Preserve original source
			}

			// Apply request values (take precedence over annotations)
			if spec.Description != nil {
				updatedFn.Description = spec.Description
			}
			if spec.Enabled != nil {
				updatedFn.Enabled = *spec.Enabled
			}
			if spec.Schedule != nil {
				updatedFn.Schedule = spec.Schedule
			}
			if spec.TimeoutSeconds != nil {
				updatedFn.TimeoutSeconds = *spec.TimeoutSeconds
			} else if annotations.TimeoutSeconds > 0 {
				updatedFn.TimeoutSeconds = annotations.TimeoutSeconds
			}
			if spec.MemoryLimitMB != nil {
				updatedFn.MemoryLimitMB = *spec.MemoryLimitMB
			} else if annotations.MemoryLimitMB > 0 {
				updatedFn.MemoryLimitMB = annotations.MemoryLimitMB
			}
			if spec.MaxRetries != nil {
				updatedFn.MaxRetries = *spec.MaxRetries
			} else if annotations.MaxRetries > 0 {
				updatedFn.MaxRetries = annotations.MaxRetries
			}
			if spec.ProgressTimeoutSeconds != nil {
				updatedFn.ProgressTimeoutSeconds = *spec.ProgressTimeoutSeconds
			} else if annotations.ProgressTimeoutSeconds > 0 {
				updatedFn.ProgressTimeoutSeconds = annotations.ProgressTimeoutSeconds
			}
			if spec.AllowNet != nil {
				updatedFn.AllowNet = *spec.AllowNet
			}
			if spec.AllowEnv != nil {
				updatedFn.AllowEnv = *spec.AllowEnv
			}
			if spec.AllowRead != nil {
				updatedFn.AllowRead = *spec.AllowRead
			}
			if spec.AllowWrite != nil {
				updatedFn.AllowWrite = *spec.AllowWrite
			}
			if spec.RequireRole != nil {
				updatedFn.RequireRole = spec.RequireRole
			}

			if err := h.storage.UpdateJobFunction(ctx, updatedFn); err != nil {
				errorList = append(errorList, fiber.Map{
					"job":    spec.Name,
					"error":  err.Error(),
					"action": "update",
				})
				continue
			}
			updated = append(updated, spec.Name)
		} else {
			// Create new function
			fn := &JobFunction{
				ID:                     uuid.New(),
				Name:                   spec.Name,
				Namespace:              namespace,
				Description:            spec.Description,
				Code:                   &code,
				OriginalCode:           &originalCode,
				IsBundled:              isBundled,
				BundleError:            bundleError,
				Enabled:                valueOr(spec.Enabled, true),
				Schedule:               spec.Schedule,
				TimeoutSeconds:         valueOr(spec.TimeoutSeconds, valueOr(&annotations.TimeoutSeconds, 300)),
				MemoryLimitMB:          valueOr(spec.MemoryLimitMB, valueOr(&annotations.MemoryLimitMB, 256)),
				MaxRetries:             valueOr(spec.MaxRetries, annotations.MaxRetries),
				ProgressTimeoutSeconds: valueOr(spec.ProgressTimeoutSeconds, valueOr(&annotations.ProgressTimeoutSeconds, 60)),
				AllowNet:               valueOr(spec.AllowNet, true),
				AllowEnv:               valueOr(spec.AllowEnv, true),
				AllowRead:              valueOr(spec.AllowRead, false),
				AllowWrite:             valueOr(spec.AllowWrite, false),
				RequireRole:            spec.RequireRole,
				Version:                1,
				CreatedBy:              createdBy,
				Source:                 "api",
			}

			if err := h.storage.CreateJobFunction(ctx, fn); err != nil {
				errorList = append(errorList, fiber.Map{
					"job":    spec.Name,
					"error":  err.Error(),
					"action": "create",
				})
				continue
			}
			created = append(created, spec.Name)
		}
	}

	// Delete removed job functions (after successful creates/updates for safety)
	if req.Options.DeleteMissing {
		for _, name := range toDelete {
			if err := h.storage.DeleteJobFunction(ctx, namespace, name); err != nil {
				errorList = append(errorList, fiber.Map{
					"job":    name,
					"error":  err.Error(),
					"action": "delete",
				})
				continue
			}
			deleted = append(deleted, name)
		}
	}

	log.Info().
		Str("namespace", namespace).
		Int("created", len(created)).
		Int("updated", len(updated)).
		Int("deleted", len(deleted)).
		Int("unchanged", len(unchanged)).
		Int("errors", len(errorList)).
		Msg("Jobs synced successfully")

	// Reschedule jobs after sync
	h.rescheduleJobsFromNamespace(ctx, namespace)

	return c.JSON(fiber.Map{
		"message":   "Jobs synced successfully",
		"namespace": namespace,
		"summary": fiber.Map{
			"created":   len(created),
			"updated":   len(updated),
			"deleted":   len(deleted),
			"unchanged": len(unchanged),
			"errors":    len(errorList),
		},
		"details": fiber.Map{
			"created":   created,
			"updated":   updated,
			"deleted":   deleted,
			"unchanged": unchanged,
		},
		"errors":  errorList,
		"dry_run": false,
	})
}

// ListNamespaces lists all unique namespaces that have job functions (Admin only)
func (h *Handler) ListNamespaces(c *fiber.Ctx) error {
	namespaces, err := h.storage.ListJobNamespaces(c.Context())
	if err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("request_id", reqID).
			Msg("Failed to list job namespaces")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to list job namespaces",
			"request_id": reqID,
		})
	}

	// Ensure we always return at least "default"
	if len(namespaces) == 0 {
		namespaces = []string{"default"}
	}

	return c.JSON(fiber.Map{"namespaces": namespaces})
}

// ListJobFunctions lists all job functions (Admin only)
func (h *Handler) ListJobFunctions(c *fiber.Ctx) error {
	namespace := c.Query("namespace", "default")

	functions, err := h.storage.ListJobFunctions(c.Context(), namespace)
	if err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("request_id", reqID).
			Msg("Failed to list job functions")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to list job functions",
			"request_id": reqID,
		})
	}

	return c.JSON(functions)
}

// GetJobFunction gets a job function by namespace and name (Admin only)
func (h *Handler) GetJobFunction(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	function, err := h.storage.GetJobFunction(c.Context(), namespace, name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job function not found"})
	}

	return c.JSON(function)
}

// UpdateJobFunction updates a job function (Admin only)
func (h *Handler) UpdateJobFunction(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	// Get existing function
	fn, err := h.storage.GetJobFunction(c.Context(), namespace, name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job function not found"})
	}

	// Parse update request
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Apply updates
	if req.Enabled != nil {
		fn.Enabled = *req.Enabled
	}

	// Save changes
	if err := h.storage.UpdateJobFunction(c.Context(), fn); err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("namespace", namespace).
			Str("name", name).
			Str("request_id", reqID).
			Msg("Failed to update job function")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to update job function",
			"request_id": reqID,
		})
	}

	log.Info().
		Str("namespace", namespace).
		Str("name", name).
		Bool("enabled", fn.Enabled).
		Msg("Job function updated")

	return c.JSON(fn)
}

// DeleteJobFunction deletes a job function (Admin only)
func (h *Handler) DeleteJobFunction(c *fiber.Ctx) error {
	namespace := c.Params("namespace")
	name := c.Params("name")

	if err := h.storage.DeleteJobFunction(c.Context(), namespace, name); err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("namespace", namespace).
			Str("name", name).
			Str("request_id", reqID).
			Msg("Failed to delete job function")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to delete job function",
			"request_id": reqID,
		})
	}

	log.Info().
		Str("namespace", namespace).
		Str("name", name).
		Msg("Job function deleted")

	return c.SendStatus(204)
}

// GetJobStats returns job statistics (Admin only)
func (h *Handler) GetJobStats(c *fiber.Ctx) error {
	var namespacePtr *string
	if namespace := c.Query("namespace"); namespace != "" {
		namespacePtr = &namespace
	}

	stats, err := h.storage.GetJobStats(c.Context(), namespacePtr)
	if err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("request_id", reqID).
			Msg("Failed to get job stats")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to get job stats",
			"request_id": reqID,
		})
	}

	return c.JSON(stats)
}

// ListWorkers lists all workers (Admin only)
func (h *Handler) ListWorkers(c *fiber.Ctx) error {
	workers, err := h.storage.ListWorkers(c.Context())
	if err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("request_id", reqID).
			Msg("Failed to list workers")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to list workers",
			"request_id": reqID,
		})
	}

	return c.JSON(workers)
}

// TerminateJob forcefully terminates a running job (Admin only)
func (h *Handler) TerminateJob(c *fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	// Get job to check status (use service role context to bypass RLS)
	job, err := h.storage.GetJobByIDAdmin(c.Context(), jobID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
	}

	// Can only terminate running jobs
	if job.Status != JobStatusRunning {
		return c.Status(400).JSON(fiber.Map{
			"error":  "Only running jobs can be terminated",
			"status": job.Status,
		})
	}

	// Cancel the job in database
	if err := h.storage.CancelJob(c.Context(), jobID); err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("job_id", jobID.String()).
			Str("request_id", reqID).
			Msg("Failed to terminate job")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to terminate job",
			"request_id": reqID,
		})
	}

	// Signal the worker to kill the job process immediately
	if h.manager != nil {
		h.manager.CancelJob(jobID)
	}

	log.Warn().
		Str("job_id", jobID.String()).
		Str("admin_user", toString(c.Locals("user_id"))).
		Msg("Job terminated by admin")

	return c.JSON(fiber.Map{"message": "Job terminated"})
}

// ListAllJobs lists all jobs across all users (Admin only)
func (h *Handler) ListAllJobs(c *fiber.Ctx) error {
	// Parse filters
	filters := &JobFilters{}

	if status := c.Query("status"); status != "" {
		s := JobStatus(status)
		filters.Status = &s
	}

	if jobName := c.Query("job_name"); jobName != "" {
		filters.JobName = &jobName
	}

	if namespace := c.Query("namespace"); namespace != "" {
		filters.Namespace = &namespace
	}

	if workerIDStr := c.Query("worker_id"); workerIDStr != "" {
		workerID, err := uuid.Parse(workerIDStr)
		if err == nil {
			filters.WorkerID = &workerID
		}
	}

	if c.Query("include_result") == "true" {
		includeResult := true
		filters.IncludeResult = &includeResult
	}

	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	filters.Limit = &limit
	filters.Offset = &offset

	// Use admin method to bypass RLS
	jobs, err := h.storage.ListJobsAdmin(c.Context(), filters)
	if err != nil {
		reqID := getRequestID(c)
		log.Error().
			Err(err).
			Str("request_id", reqID).
			Msg("Failed to list all jobs")

		return c.Status(500).JSON(fiber.Map{
			"error":      "Failed to list jobs",
			"request_id": reqID,
		})
	}

	// Calculate ETA for running jobs
	for i := range jobs {
		jobs[i].CalculateETA()
	}

	return c.JSON(fiber.Map{
		"jobs":   jobs,
		"limit":  limit,
		"offset": offset,
	})
}

// LoadFromFilesystem loads jobs from filesystem at boot time
func (h *Handler) LoadFromFilesystem(ctx context.Context, namespace string) error {
	// Load builtin jobs first (these ship with Fluxbase and are disabled by default)
	if err := h.loader.LoadBuiltinJobs(ctx, namespace); err != nil {
		log.Warn().Err(err).Msg("Failed to load builtin jobs")
		// Don't fail boot if builtin jobs fail to load
	}

	// Then load user jobs from filesystem
	if err := h.loader.LoadFromFilesystem(ctx, namespace); err != nil {
		return err
	}

	// Reschedule jobs after loading
	if h.scheduler != nil {
		h.rescheduleJobsFromNamespace(ctx, namespace)
	}

	return nil
}

// rescheduleJobsFromNamespace updates the scheduler with jobs from a namespace
func (h *Handler) rescheduleJobsFromNamespace(ctx context.Context, namespace string) {
	if h.scheduler == nil {
		return
	}

	jobs, err := h.storage.ListJobFunctions(ctx, namespace)
	if err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to list jobs for rescheduling")
		return
	}

	for _, job := range jobs {
		if job.Enabled && job.Schedule != nil && *job.Schedule != "" {
			if err := h.scheduler.ScheduleJob(job); err != nil {
				log.Warn().Err(err).Str("job", job.Name).Msg("Failed to schedule job")
			}
		} else {
			h.scheduler.UnscheduleJob(namespace, job.Name)
		}
	}
}

// Helper functions

func valueOr[T any](ptr *T, defaultVal T) T {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

func getRequestID(c *fiber.Ctx) string {
	requestID := c.Locals("requestid")
	if requestID != nil {
		if reqIDStr, ok := requestID.(string); ok {
			return reqIDStr
		}
	}
	return c.Get("X-Request-ID", "")
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if uid, ok := v.(*uuid.UUID); ok {
		if uid == nil {
			return ""
		}
		return uid.String()
	}
	return fmt.Sprintf("%v", v)
}
