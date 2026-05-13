package jobs

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/middleware"

	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/util"
)

// SubmitJob submits a new job to the queue
func (h *Handler) SubmitJob(c fiber.Ctx) error {
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

	if err := c.Bind().Body(&req); err != nil {
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
		if callerRole == nil || (callerRole.(string) != "service_role" && callerRole.(string) != "tenant_service") {
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
		if err := h.storage.DB.Pool().QueryRow(middleware.CtxWithTenant(c), checkQuery, parsed).Scan(&exists); err != nil || !exists {
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
		impersonationClaims, err := h.authService.JWTManager().ValidateToken(impersonationToken)
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
					// Dashboard admins are in platform.users, not auth.users
					var exists bool
					checkQuery := "SELECT EXISTS(SELECT 1 FROM auth.users WHERE id = $1)"
					if err := h.storage.DB.Pool().QueryRow(middleware.CtxWithTenant(c), checkQuery, parsed).Scan(&exists); err == nil && exists {
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
		jobFunction, err = h.storage.GetJobFunction(middleware.CtxWithTenant(c), req.Namespace, req.JobName)
	} else {
		jobFunction, err = h.storage.GetJobFunctionByName(middleware.CtxWithTenant(c), req.JobName)
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
	if len(jobFunction.RequireRoles) > 0 {
		if userRole == nil {
			return c.Status(403).JSON(fiber.Map{
				"error":          "Authentication required",
				"required_roles": jobFunction.RequireRoles,
			})
		}

		// Check if user's role satisfies ANY of the required roles using hierarchy
		// (admin > authenticated > anon)
		if !roleSatisfiesRequirements(*userRole, jobFunction.RequireRoles) {
			return c.Status(403).JSON(fiber.Map{
				"error":          "Insufficient permissions",
				"required_roles": jobFunction.RequireRoles,
				"user_role":      *userRole,
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
		Priority:               util.ValueOr(req.Priority, 0),
		MaxDurationSeconds:     &jobFunction.TimeoutSeconds,
		ProgressTimeoutSeconds: &jobFunction.ProgressTimeoutSeconds,
		MaxRetries:             jobFunction.MaxRetries,
		RetryCount:             0,
		CreatedBy:              userID,
		UserRole:               userRole,
		UserEmail:              userEmail,
		ScheduledAt:            req.Scheduled,
	}

	if err := h.storage.CreateJob(middleware.CtxWithTenant(c), job); err != nil {
		reqID := apperrors.GetRequestID(c)
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
		Str("user_id", util.ToString(userID)).
		Msg("Job submitted")

	return c.Status(201).JSON(job)
}

// GetJob gets a job by ID (RLS enforced)
func (h *Handler) GetJob(c fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	job, err := h.storage.GetJob(middleware.CtxWithTenant(c), jobID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
	}

	// Calculate ETA and flatten progress for running jobs
	job.CalculateETA()
	job.FlattenProgress()

	return c.JSON(job)
}

// GetJobAdmin gets a job by ID (admin access, bypasses RLS)
func (h *Handler) GetJobAdmin(c fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	job, err := h.storage.GetJobByIDAdmin(middleware.CtxWithTenant(c), jobID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Job not found"})
	}

	// Calculate ETA and flatten progress for running jobs
	job.CalculateETA()
	job.FlattenProgress()

	return c.JSON(job)
}

// GetJobLogs gets execution logs for a job (admin access)
func (h *Handler) GetJobLogs(c fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	// Parse after_line query param for pagination
	afterLine := 0
	if afterLineStr := c.Query("after_line"); afterLineStr != "" {
		if l, err := strconv.Atoi(afterLineStr); err == nil {
			afterLine = l
		}
	}

	// Query logs from central logging using job ID as execution ID
	entries, err := h.loggingService.GetExecutionLogs(middleware.CtxWithTenant(c), jobID.String(), afterLine)
	if err != nil {
		log.Error().Err(err).Str("job_id", jobID.String()).Msg("Failed to get job logs")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get job logs",
		})
	}

	return c.JSON(fiber.Map{
		"entries": entries,
		"count":   len(entries),
	})
}

// GetJobLogsUser returns logs for user's own job
// GET /api/v1/jobs/:id/logs
func (h *Handler) GetJobLogsUser(c fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	// Get user context
	userID := ""
	if uid, ok := c.Locals("user_id").(string); ok {
		userID = uid
	}

	// Get job to verify ownership
	job, err := h.storage.GetJob(middleware.CtxWithTenant(c), jobID)
	if err != nil {
		log.Error().Err(err).Str("job_id", jobID.String()).Msg("Failed to get job")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get job",
		})
	}

	if job == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Job not found"})
	}

	// Check ownership (unless service_role)
	role, _ := c.Locals("user_role").(string)
	if role != "service_role" && role != "tenant_service" {
		// Parse userID for comparison
		userUUID, err := uuid.Parse(userID)
		if err != nil || job.CreatedBy == nil || *job.CreatedBy != userUUID {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Job not found"})
		}
	}

	// Parse after_line query param for pagination
	afterLine := 0
	if afterLineStr := c.Query("after_line"); afterLineStr != "" {
		if l, err := strconv.Atoi(afterLineStr); err == nil {
			afterLine = l
		}
	}

	// Query logs from central logging using job ID as execution ID
	entries, err := h.loggingService.GetExecutionLogs(middleware.CtxWithTenant(c), jobID.String(), afterLine)
	if err != nil {
		log.Error().Err(err).Str("job_id", jobID.String()).Msg("Failed to get job logs")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get job logs",
		})
	}

	return c.JSON(fiber.Map{
		"logs":  entries,
		"count": len(entries),
	})
}

// ListJobs lists jobs for the authenticated user (RLS enforced)
func (h *Handler) ListJobs(c fiber.Ctx) error {
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
		if namespace == "default" {
			namespace = ""
		}
		filters.Namespace = &namespace
	}

	if c.Query("include_result") == "true" {
		includeResult := true
		filters.IncludeResult = &includeResult
	}

	limit := fiber.Query[int](c, "limit", 50)
	offset := fiber.Query[int](c, "offset", 0)

	filters.Limit = &limit
	filters.Offset = &offset

	jobs, err := h.storage.ListJobs(middleware.CtxWithTenant(c), filters)
	if err != nil {
		reqID := apperrors.GetRequestID(c)
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
func (h *Handler) CancelJob(c fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	// Get job to check status
	job, err := h.storage.GetJob(middleware.CtxWithTenant(c), jobID)
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

	if err := h.storage.CancelJob(middleware.CtxWithTenant(c), jobID); err != nil {
		reqID := apperrors.GetRequestID(c)
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
func (h *Handler) RetryJob(c fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	// Get job to check status
	job, err := h.storage.GetJob(middleware.CtxWithTenant(c), jobID)
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

	if err := h.storage.RequeueFailedJob(middleware.CtxWithTenant(c), jobID); err != nil {
		reqID := apperrors.GetRequestID(c)
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
func (h *Handler) CancelJobAdmin(c fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	// Get job to check status (admin access)
	job, err := h.storage.GetJobByIDAdmin(middleware.CtxWithTenant(c), jobID)
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

	if err := h.storage.CancelJob(middleware.CtxWithTenant(c), jobID); err != nil {
		reqID := apperrors.GetRequestID(c)
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
func (h *Handler) RetryJobAdmin(c fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	// Get job to check status (admin access)
	job, err := h.storage.GetJobByIDAdmin(middleware.CtxWithTenant(c), jobID)
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

	if err := h.storage.RequeueFailedJob(middleware.CtxWithTenant(c), jobID); err != nil {
		reqID := apperrors.GetRequestID(c)
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
func (h *Handler) ResubmitJobAdmin(c fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	// Create new job based on the original
	newJob, err := h.storage.ResubmitJob(middleware.CtxWithTenant(c), jobID)
	if err != nil {
		reqID := apperrors.GetRequestID(c)
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

// TerminateJob forcefully terminates a running job (Admin only)
func (h *Handler) TerminateJob(c fiber.Ctx) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid job ID"})
	}

	// Get job to check status (use service role context to bypass RLS)
	job, err := h.storage.GetJobByIDAdmin(middleware.CtxWithTenant(c), jobID)
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
	if err := h.storage.CancelJob(middleware.CtxWithTenant(c), jobID); err != nil {
		reqID := apperrors.GetRequestID(c)
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
		Str("admin_user", util.ToString(c.Locals("user_id"))).
		Msg("Job terminated by admin")

	return c.JSON(fiber.Map{"message": "Job terminated"})
}

// ListAllJobs lists all jobs across all users (Admin only)
func (h *Handler) ListAllJobs(c fiber.Ctx) error {
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
		if namespace == "default" {
			namespace = ""
		}
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

	limit := fiber.Query[int](c, "limit", 50)
	offset := fiber.Query[int](c, "offset", 0)

	filters.Limit = &limit
	filters.Offset = &offset

	// Use admin method to bypass RLS
	jobs, err := h.storage.ListJobsAdmin(middleware.CtxWithTenant(c), filters)
	if err != nil {
		reqID := apperrors.GetRequestID(c)
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
